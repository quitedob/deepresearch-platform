package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-research-platform/internal/cache"
	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/ai-research-platform/internal/pkg/eino/agent"
	"github.com/ai-research-platform/internal/models"
	"github.com/ai-research-platform/internal/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ResearchService handles research session operations
type ResearchService struct {
	repo         repository.ResearchRepository
	agent        *agent.ResearchAgent
	orchestrator *agent.ParallelOrchestrator
	tools        []eino.InvokableTool
	cache        cache.Cache
	eventStream  *EventStream
}

// NewResearchService creates a new research service
func NewResearchService(
	repo repository.ResearchRepository,
	researchAgent *agent.ResearchAgent,
	tools []eino.InvokableTool,
	cacheManager cache.Cache,
	eventStream *EventStream,
) *ResearchService {
	return &ResearchService{
		repo:        repo,
		agent:       researchAgent,
		tools:       tools,
		cache:       cacheManager,
		eventStream: eventStream,
	}
}

// SetAgent 设置研究 Agent
func (s *ResearchService) SetAgent(ag *agent.ResearchAgent) {
	s.agent = ag
}

// SetOrchestrator 设置并行编排器
func (s *ResearchService) SetOrchestrator(orch *agent.ParallelOrchestrator) {
	s.orchestrator = orch
}

// StartResearch initiates a new research session
func (s *ResearchService) StartResearch(ctx context.Context, userID, query string, researchType string) (*models.ResearchSession, error) {
	session := &models.ResearchSession{
		ID:           uuid.New().String(),
		UserID:       userID,
		Query:        query,
		Status:       "planning",
		Progress:     0.0,
		ResearchType: researchType,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create research session: %w", err)
	}

	go s.executeResearch(context.Background(), session)

	return session, nil
}

// ExecuteResearch 执行研究（公开方法，供API调用）
func (s *ResearchService) ExecuteResearch(sessionID, query, researchType string) {
	ctx := context.Background()
	session := &models.ResearchSession{
		ID:           sessionID,
		Query:        query,
		ResearchType: researchType,
	}
	s.executeResearch(ctx, session)
}

// executeResearch 执行研究
func (s *ResearchService) executeResearch(ctx context.Context, session *models.ResearchSession) {
	// 优先使用并行编排器（deep/comprehensive类型）
	useParallel := s.orchestrator != nil && (session.ResearchType == "deep" || session.ResearchType == "comprehensive")

	if !useParallel && s.agent == nil {
		if s.eventStream != nil {
			s.eventStream.Send(session.ID, &ResearchEvent{
				Type:      "error",
				Message:   "Research agent not configured",
				Timestamp: time.Now(),
			})
		}
		return
	}

	// 创建进度回调
	progressCallback := func(event *agent.ProgressEvent) {
		if s.repo != nil {
			_ = s.repo.UpdateSessionStatus(ctx, session.ID, event.Stage, event.Progress)
		}

		if s.eventStream != nil {
			s.eventStream.Send(session.ID, &ResearchEvent{
				Type:        "progress",
				Stage:       event.Stage,
				Progress:    event.Progress,
				Message:     event.Message,
				TaskName:    event.TaskName,
				TaskStatus:  event.TaskStatus,
				PartialData: event.PartialData,
				Timestamp:   event.Timestamp,
			})
		}
	}

	// 执行研究
	var result *agent.Result
	var err error

	if useParallel {
		s.orchestrator.RegisterCallback(progressCallback)
		result, err = s.orchestrator.Run(ctx, session.Query)
	} else {
		if s.agent == nil {
			if s.eventStream != nil {
				s.eventStream.Send(session.ID, &ResearchEvent{
					Type:      "error",
					Message:   "Research agent not configured",
					Timestamp: time.Now(),
				})
			}
			return
		}
		s.agent.RegisterCallback(progressCallback)
		result, err = s.agent.Run(ctx, session.Query)
	}

	if err != nil {
		if s.repo != nil {
			_ = s.repo.UpdateSessionStatus(ctx, session.ID, "failed", 0.0)
		}
		if s.eventStream != nil {
			s.eventStream.Send(session.ID, &ResearchEvent{
				Type:      "error",
				Message:   fmt.Sprintf("Research failed: %v", err),
				Timestamp: time.Now(),
			})
		}
		return
	}

	// 保存研究结果
	s.saveResearchResult(ctx, session, result)

	if s.repo != nil {
		_ = s.repo.UpdateSessionStatus(ctx, session.ID, "completed", 1.0)
	}

	// 发送完成事件
	s.sendCompletedEvent(session, result)
}

// saveResearchResult 保存研究结果到数据库
func (s *ResearchService) saveResearchResult(ctx context.Context, session *models.ResearchSession, result *agent.Result) {
	if !result.Success || s.repo == nil {
		return
	}

	for _, step := range result.Steps {
		if step.Phase != "searching" {
			continue
		}
		completedAt := step.Timestamp
		inputJSON, _ := json.Marshal(map[string]string{"thought": step.Thought})
		outputJSON, _ := json.Marshal(map[string]interface{}{
			"observation": step.Observation,
			"quality":     step.Quality,
		})

		researchTask := &models.ResearchTask{
			ResearchID:    session.ID,
			TaskType:      step.Phase,
			ToolName:      step.Action,
			Status:        "completed",
			Input:         datatypes.JSON(inputJSON),
			Output:        datatypes.JSON(outputJSON),
			ExecutionTime: int(result.ExecutionTime / int64(len(result.Steps)+1)),
			CreatedAt:     step.Timestamp,
			CompletedAt:   &completedAt,
		}
		_ = s.repo.SaveTask(ctx, researchTask)
	}

	metadataMap := map[string]interface{}{
		"tools_used":       result.ToolsUsed,
		"execution_time":   result.ExecutionTime,
		"confidence_score": result.ConfidenceScore,
		"source_count":     result.SourceCount,
	}

	if result.StructuredReport != nil {
		metadataMap["has_structured_report"] = true
		metadataMap["conclusions_count"] = len(result.StructuredReport.Structured.Conclusions)
		metadataMap["unresolved_issues"] = result.StructuredReport.Structured.UnresolvedIssues
		metadataMap["key_insights"] = result.StructuredReport.Structured.KeyInsights
	}

	if len(result.CriticResults) > 0 {
		lastCritic := result.CriticResults[len(result.CriticResults)-1]
		metadataMap["critic_quality_score"] = lastCritic.QualityScore
		metadataMap["contradictions_found"] = len(lastCritic.Contradictions)
		metadataMap["evidence_gaps"] = lastCritic.EvidenceGaps
	}

	metadataJSON, _ := json.Marshal(metadataMap)

	researchResult := &models.ResearchResult{
		ResearchID: session.ID,
		Summary:    result.FinalAnswer,
		Findings:   datatypes.JSON("[]"),
		Citations:  datatypes.JSON("[]"),
		Metadata:   datatypes.JSON(metadataJSON),
		CreatedAt:  time.Now(),
	}
	_ = s.repo.SaveResult(ctx, researchResult)
}

// sendCompletedEvent 发送研究完成事件
func (s *ResearchService) sendCompletedEvent(session *models.ResearchSession, result *agent.Result) {
	if s.eventStream == nil {
		return
	}

	evidence := make([]map[string]interface{}, 0)
	for i, step := range result.Steps {
		if step.Phase == "searching" && step.Observation != "" {
			evidence = append(evidence, map[string]interface{}{
				"id":               fmt.Sprintf("evidence_%d", len(evidence)+1),
				"source_type":      getSourceType(step.Action),
				"source_title":     getSourceTitle(step.Action, step.Thought),
				"content":          step.Observation,
				"relevance_score":  step.Quality,
				"confidence_score": result.ConfidenceScore,
				"step_number":      i + 1,
			})
		}
	}

	completedData := map[string]interface{}{
		"report_text": result.FinalAnswer,
		"tools_used":  result.ToolsUsed,
		"session_id":  session.ID,
		"metadata": map[string]interface{}{
			"type":             "research",
			"evidence":         evidence,
			"tools_used":       result.ToolsUsed,
			"execution_time":   result.ExecutionTime,
			"confidence_score": result.ConfidenceScore,
			"source_count":     result.SourceCount,
		},
	}

	if result.StructuredReport != nil {
		completedData["structured_report"] = map[string]interface{}{
			"conclusions":       result.StructuredReport.Structured.Conclusions,
			"key_insights":      result.StructuredReport.Structured.KeyInsights,
			"unresolved_issues": result.StructuredReport.Structured.UnresolvedIssues,
			"confidence_score":  result.StructuredReport.Structured.ConfidenceScore,
		}
	}

	if len(result.CriticResults) > 0 {
		criticSummary := make([]map[string]interface{}, 0)
		for _, cr := range result.CriticResults {
			criticSummary = append(criticSummary, map[string]interface{}{
				"contradictions":  cr.Contradictions,
				"evidence_gaps":   cr.EvidenceGaps,
				"quality_score":   cr.QualityScore,
				"should_continue": cr.ShouldContinue,
			})
		}
		completedData["critic_results"] = criticSummary
	}

	s.eventStream.Send(session.ID, &ResearchEvent{
		Type:      "completed",
		Message:   "Research completed successfully",
		Timestamp: time.Now(),
		Data:      completedData,
	})
}

// GetSession retrieves a research session by ID
func (s *ResearchService) GetSession(ctx context.Context, sessionID string) (*models.ResearchSession, error) {
	cacheKey := fmt.Sprintf("research_session:%s", sessionID)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		if session, ok := cached.(*models.ResearchSession); ok {
			return session, nil
		}
	}

	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	_ = s.cache.Set(ctx, cacheKey, session, 2*time.Minute)
	return session, nil
}

// StreamProgress creates a stream for research progress updates
func (s *ResearchService) StreamProgress(ctx context.Context, sessionID string) (<-chan *ResearchEvent, error) {
	_, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return s.eventStream.CreateStream(sessionID), nil
}

// GetResults retrieves the research results for a session
func (s *ResearchService) GetResults(ctx context.Context, sessionID string) (*models.ResearchResult, error) {
	cacheKey := fmt.Sprintf("research_result:%s", sessionID)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		if result, ok := cached.(*models.ResearchResult); ok {
			return result, nil
		}
	}

	result, err := s.repo.GetResult(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	_ = s.cache.Set(ctx, cacheKey, result, 10*time.Minute)
	return result, nil
}

// GetTasks retrieves all tasks for a research session
func (s *ResearchService) GetTasks(ctx context.Context, sessionID string, limit, offset int) ([]*models.ResearchTask, error) {
	return s.repo.GetTasksByResearch(ctx, sessionID, limit, offset)
}

// GetUserSessions retrieves all research sessions for a user
func (s *ResearchService) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]*models.ResearchSession, error) {
	return s.repo.GetSessionsByUser(ctx, userID, limit, offset)
}

// CancelResearch cancels an ongoing research session
func (s *ResearchService) CancelResearch(ctx context.Context, sessionID string) error {
	if err := s.repo.UpdateSessionStatus(ctx, sessionID, "failed", 0.0); err != nil {
		return fmt.Errorf("failed to cancel research: %w", err)
	}

	if s.eventStream != nil {
		s.eventStream.Send(sessionID, &ResearchEvent{
			Type:      "cancelled",
			Message:   "Research cancelled by user",
			Timestamp: time.Now(),
		})
		s.eventStream.CloseStream(sessionID)
	}

	cacheKey := fmt.Sprintf("research_session:%s", sessionID)
	_ = s.cache.Delete(ctx, cacheKey)

	return nil
}

// getSourceType 根据工具名称返回来源类型
func getSourceType(toolName string) string {
	switch toolName {
	case "web_search":
		return "web"
	case "wikipedia":
		return "wikipedia"
	case "arxiv_search", "arxiv":
		return "arxiv"
	default:
		return "search"
	}
}

// getSourceTitle 根据工具名称和思考内容生成来源标题
func getSourceTitle(toolName, thought string) string {
	switch toolName {
	case "web_search":
		return "网络搜索"
	case "wikipedia":
		return "维基百科"
	case "arxiv_search", "arxiv":
		return "学术论文"
	default:
		if thought != "" && len(thought) < 50 {
			return thought
		}
		return "信息搜索"
	}
}

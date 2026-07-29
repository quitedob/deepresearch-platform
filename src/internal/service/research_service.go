package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ai-research-platform/internal/cache"
	"github.com/ai-research-platform/internal/logger"
	"github.com/ai-research-platform/internal/models"
	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/ai-research-platform/internal/pkg/eino/agent"
	"github.com/ai-research-platform/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// ResearchService handles research session operations
type researchRunner interface {
	RegisterCallback(agent.ProgressCallback)
	ClearCallbacks()
	Run(context.Context, string) (*agent.Result, error)
}

type ResearchService struct {
	repo            repository.ResearchRepository
	agent           researchRunner
	evaluationAgent *agent.ResearchAgent
	orchestrator    researchRunner
	tools           []eino.InvokableTool
	cache           cache.Cache
	eventStream     *EventStream

	// 会话生命周期管理：可取消的 context
	activeSessions   map[string]context.CancelFunc
	activeSessionsMu sync.Mutex
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
		repo:            repo,
		agent:           researchAgent,
		evaluationAgent: researchAgent,
		tools:           tools,
		cache:           cacheManager,
		eventStream:     eventStream,
		activeSessions:  make(map[string]context.CancelFunc),
	}
}

// SetAgent 设置研究 Agent
func (s *ResearchService) SetAgent(ag *agent.ResearchAgent) {
	s.agent = ag
	s.evaluationAgent = ag
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

	// 创建可取消的 context，与会话生命周期绑定
	ctx, cancel := context.WithCancel(context.Background())
	s.trackSession(session.ID, cancel)

	go func() {
		defer s.untrackSession(session.ID)
		_ = s.executeResearch(ctx, session)
	}()

	return session, nil
}

// CanExecute verifies that the service can dispatch the requested research type.
func (s *ResearchService) CanExecute(researchType string) error {
	if s.repo == nil {
		return fmt.Errorf("research repository not configured")
	}
	if s.eventStream == nil {
		return fmt.Errorf("research event stream not configured")
	}
	if (researchType == "deep" || researchType == "comprehensive") && s.orchestrator != nil {
		return nil
	}
	if s.agent == nil {
		return fmt.Errorf("research agent not configured")
	}
	return nil
}

// ExecuteResearch 执行研究（公开方法，供API调用）
func (s *ResearchService) ExecuteResearch(sessionID, query, researchType string) error {
	return s.ExecuteResearchWithConfig(sessionID, query, researchType, "", "", nil)
}

// ExecuteResearchWithConfig 执行研究（含自定义 LLM/工具配置）
func (s *ResearchService) ExecuteResearchWithConfig(sessionID, query, researchType, llmProvider, llmModel string, enabledTools []string) error {
	if llmProvider != "" || llmModel != "" || len(enabledTools) > 0 {
		return fmt.Errorf("request-scoped model and tool configuration is not supported")
	}
	if err := s.CanExecute(researchType); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.trackSession(sessionID, cancel)
	session := &models.ResearchSession{ID: sessionID, Query: query, ResearchType: researchType}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.failResearch(context.Background(), sessionID, fmt.Errorf("research goroutine panicked: %v", recovered))
			}
			s.untrackSession(sessionID)
		}()
		_ = s.executeResearch(ctx, session)
	}()
	return nil
}

// ExecuteResearchJob runs one durable research job in the caller's context.
// It is synchronous so the database worker owns completion and retry decisions.
func (s *ResearchService) ExecuteResearchJob(
	ctx context.Context,
	sessionID, query, researchType string,
) error {
	if err := s.CanExecute(researchType); err != nil {
		return err
	}
	executionCtx, cancel := context.WithCancel(ctx)
	s.trackSession(sessionID, cancel)
	defer func() {
		cancel()
		s.untrackSession(sessionID)
	}()

	session := &models.ResearchSession{ID: sessionID, Query: query, ResearchType: researchType}
	return s.executeResearch(executionCtx, session)
}

// executeResearch 执行研究
func (s *ResearchService) executeResearch(ctx context.Context, session *models.ResearchSession) error {
	// 优先使用并行编排器（deep/comprehensive类型）
	useParallel := s.orchestrator != nil && (session.ResearchType == "deep" || session.ResearchType == "comprehensive")

	if !useParallel && s.agent == nil {
		err := fmt.Errorf("research agent not configured")
		if s.eventStream != nil {
			s.eventStream.Send(session.ID, &ResearchEvent{
				Type:      "error",
				Message:   "Research agent not configured",
				Timestamp: time.Now(),
			})
			s.eventStream.CloseStream(session.ID)
		}
		return err
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

	// 执行研究。回调属于一次执行，必须在返回或 panic 时清理。
	var runner researchRunner
	if useParallel {
		runner = s.orchestrator
	} else {
		runner = s.agent
	}
	runner.RegisterCallback(progressCallback)
	defer runner.ClearCallbacks()
	result, err := runner.Run(ctx, session.Query)

	if err != nil {
		s.failResearch(ctx, session.ID, fmt.Errorf("research failed: %w", err))
		return err
	}

	if result == nil {
		err = fmt.Errorf("research returned no result")
		s.failResearch(ctx, session.ID, err)
		return err
	}

	// ======= Evaluator 接入：对研究结果做质量评分 =======
	if s.evaluationAgent != nil && result.Success {
		evaluationScore := s.evaluateResult(result, session)
		if evaluationScore >= 0 {
			result.ConfidenceScore = (result.ConfidenceScore + evaluationScore) / 2
		}
	}
	if err := s.saveResearchResult(ctx, session, result); err != nil {
		persistErr := fmt.Errorf("persist research result: %w", err)
		s.failResearch(ctx, session.ID, persistErr)
		return persistErr
	}
	s.sendCompletedEvent(session, result)
	if s.eventStream != nil {
		s.eventStream.CloseStream(session.ID)
	}
	return nil
}

func (s *ResearchService) failResearch(ctx context.Context, sessionID string, cause error) {
	logger.Error("research execution failed", zap.String("session_id", sessionID), zap.Error(cause))
	if s.repo != nil {
		if err := s.repo.UpdateSessionStatus(ctx, sessionID, "failed", 0.0); err != nil {
			logger.Error("mark research failed status failed", zap.String("session_id", sessionID), zap.Error(err))
		}
	}
	if s.eventStream != nil {
		s.eventStream.Send(sessionID, &ResearchEvent{Type: "error", Message: cause.Error(), Timestamp: time.Now()})
		s.eventStream.CloseStream(sessionID)
	}
}

// saveResearchResult 保存研究结果到数据库
func (s *ResearchService) saveResearchResult(ctx context.Context, session *models.ResearchSession, result *agent.Result) error {
	if !result.Success {
		return fmt.Errorf("research result was not successful: %s", result.Error)
	}
	if s.repo == nil {
		return fmt.Errorf("research repository not configured")
	}
	return s.repo.WithTransaction(ctx, func(repo repository.ResearchRepository) error {
		for _, step := range result.Steps {
			if step.Phase != "searching" {
				continue
			}
			completedAt := step.Timestamp
			inputJSON, err := json.Marshal(map[string]string{"thought": step.Thought})
			if err != nil {
				return fmt.Errorf("marshal research task input: %w", err)
			}
			outputJSON, err := json.Marshal(map[string]interface{}{"observation": step.Observation, "quality": step.Quality})
			if err != nil {
				return fmt.Errorf("marshal research task output: %w", err)
			}
			task := &models.ResearchTask{ResearchID: session.ID, TaskType: step.Phase, ToolName: step.Action, Status: "completed", Input: datatypes.JSON(inputJSON), Output: datatypes.JSON(outputJSON), ExecutionTime: int(result.ExecutionTime / int64(len(result.Steps)+1)), CreatedAt: step.Timestamp, CompletedAt: &completedAt}
			if err := repo.SaveTask(ctx, task); err != nil {
				return fmt.Errorf("save research task: %w", err)
			}
		}
		metadata := map[string]interface{}{"tools_used": result.ToolsUsed, "execution_time": result.ExecutionTime, "confidence_score": result.ConfidenceScore, "source_count": result.SourceCount}
		if result.StructuredReport != nil {
			metadata["has_structured_report"] = true
			metadata["conclusions_count"] = len(result.StructuredReport.Structured.Conclusions)
			metadata["unresolved_issues"] = result.StructuredReport.Structured.UnresolvedIssues
			metadata["key_insights"] = result.StructuredReport.Structured.KeyInsights
		}
		if len(result.CriticResults) > 0 {
			critic := result.CriticResults[len(result.CriticResults)-1]
			metadata["critic_quality_score"] = critic.QualityScore
			metadata["contradictions_found"] = len(critic.Contradictions)
			metadata["evidence_gaps"] = critic.EvidenceGaps
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal research result metadata: %w", err)
		}
		resultModel := &models.ResearchResult{ResearchID: session.ID, Summary: result.FinalAnswer, Findings: datatypes.JSON("[]"), Citations: datatypes.JSON("[]"), Metadata: datatypes.JSON(metadataJSON), CreatedAt: time.Now()}
		if err := repo.SaveResult(ctx, resultModel); err != nil {
			return fmt.Errorf("save research result: %w", err)
		}
		if err := repo.UpdateSessionStatus(ctx, session.ID, "completed", 1.0); err != nil {
			return fmt.Errorf("mark research completed: %w", err)
		}
		return nil
	})
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
	// 取消正在运行的 goroutine
	s.activeSessionsMu.Lock()
	if cancel, ok := s.activeSessions[sessionID]; ok {
		cancel()
		delete(s.activeSessions, sessionID)
	}
	s.activeSessionsMu.Unlock()

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

// trackSession 追踪活跃会话的 cancel 函数
func (s *ResearchService) trackSession(sessionID string, cancel context.CancelFunc) {
	s.activeSessionsMu.Lock()
	if s.activeSessions == nil {
		s.activeSessions = make(map[string]context.CancelFunc)
	}
	// 如果有旧的同 ID 会话正在运行，先取消它
	if oldCancel, ok := s.activeSessions[sessionID]; ok {
		oldCancel()
	}
	s.activeSessions[sessionID] = cancel
	s.activeSessionsMu.Unlock()
}

// untrackSession 移除已完成的会话
func (s *ResearchService) untrackSession(sessionID string) {
	s.activeSessionsMu.Lock()
	delete(s.activeSessions, sessionID)
	s.activeSessionsMu.Unlock()
}

// evaluateResult 使用 Evaluator 对研究结果质量评分
func (s *ResearchService) evaluateResult(result *agent.Result, session *models.ResearchSession) float64 {
	if s.evaluationAgent == nil || !result.Success || result.FinalAnswer == "" {
		return -1
	}

	// 构造轻量评测用例（仅做质量把关，不做完整回归评测）
	testCase := &agent.EvaluationCase{
		ID:              "runtime_" + session.ID,
		Query:           result.Query,
		ExpectedPoints:  []string{}, // 运行时评测不设预期要点
		RequiredSources: []string{},
		MinConfidence:   0.3,
		MaxTimeSeconds:  0, // 已经执行完毕，不限时
	}

	evaluator := agent.NewEvaluator(s.evaluationAgent)

	// 直接用已有结果构造评测结果，不重新执行研究
	evalResult := &agent.TestEvaluationResult{
		CaseID:          testCase.ID,
		Query:           result.Query,
		Timestamp:       time.Now(),
		Details:         result,
		ConfidenceScore: result.ConfidenceScore,
	}

	// 计算得分
	evalResult.Score = evaluator.CalculateScore(evalResult, testCase)
	evalResult.Passed = evaluator.CheckPassed(evalResult, testCase)

	// 归一化到 0-1
	return evalResult.Score / 100.0
}

// getSourceType 根据工具名称返回来源类型
func getSourceType(toolName string) string {
	switch toolName {
	case "web_search", "web_search_prime":
		return "web"
	case "web_reader":
		return "web_page"
	case "wikipedia":
		return "wikipedia"
	case "arxiv_search", "arxiv":
		return "arxiv"
	case "zread_repo":
		return "github"
	default:
		return "search"
	}
}

// getSourceTitle 根据工具名称和思考内容生成来源标题
func getSourceTitle(toolName, thought string) string {
	switch toolName {
	case "web_search", "web_search_prime":
		return "网络搜索"
	case "web_reader":
		return "网页全文"
	case "wikipedia":
		return "维基百科"
	case "arxiv_search", "arxiv":
		return "学术论文"
	case "zread_repo":
		return "代码仓库"
	default:
		if thought != "" && len(thought) < 50 {
			return thought
		}
		return "信息搜索"
	}
}

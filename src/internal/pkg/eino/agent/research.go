// Package agent 提供研究 Agent 实现
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ResearchAgent 研究智能体，基于 Agentic RAG + ReAct 模式
// 核心升级：Plan-Execute-Critic-Reflect 循环
type ResearchAgent struct {
	chatModel       model.ChatModel
	tools           []tool.InvokableTool
	config          Config
	callbacks       []ProgressCallback
	planner         *ResearchPlanner   // 增强版规划器
	critic          *Critic            // 反证检查器
	reportGenerator *ReportGenerator   // 结构化报告生成器
}

// Config Agent 配置
type Config struct {
	MaxIterations       int           // 最大迭代次数
	Timeout             time.Duration // 超时时间
	MinSources          int           // 最少信息来源数
	ConfidenceThreshold float64       // 置信度阈值 (0-1)
	MaxSearchDepth      int           // 最大搜索深度（反思循环次数）
	EnableCritic        bool          // 启用反证检查
	EnableStructured    bool          // 启用结构化输出
}

// Result 研究结果
type Result struct {
	Query            string            `json:"query"`
	FinalAnswer      string            `json:"final_answer"`
	Steps            []Step            `json:"steps"`
	Success          bool              `json:"success"`
	Error            string            `json:"error,omitempty"`
	ExecutionTime    int64             `json:"execution_time_ms"`
	ToolsUsed        []string          `json:"tools_used"`
	ConfidenceScore  float64           `json:"confidence_score"`
	SourceCount      int               `json:"source_count"`
	StructuredReport *StructuredReport `json:"structured_report,omitempty"` // 新增：结构化报告
	CriticResults    []*CriticResult   `json:"critic_results,omitempty"`    // 新增：反证检查结果
}

// Step 执行步骤
type Step struct {
	StepNumber  int       `json:"step_number"`
	Phase       string    `json:"phase"` // planning, searching, evaluating, reflecting, critic, synthesizing
	Thought     string    `json:"thought"`
	Action      string    `json:"action"`
	Observation string    `json:"observation"`
	Quality     float64   `json:"quality"` // 信息质量评分 0-1
	Timestamp   time.Time `json:"timestamp"`
}

// ResearchPlan 研究计划（保留兼容性）
type ResearchPlan struct {
	MainQuestion   string   `json:"main_question"`
	SubQuestions   []string `json:"sub_questions"`
	RequiredTools  []string `json:"required_tools"`
	ExpectedDepth  int      `json:"expected_depth"`
}

// EvaluationResult 评估结果
type EvaluationResult struct {
	IsComplete      bool     `json:"is_complete"`
	ConfidenceScore float64  `json:"confidence_score"`
	MissingAspects  []string `json:"missing_aspects"`
	QualityScore    float64  `json:"quality_score"`
	Suggestion      string   `json:"suggestion"`
}

// ProgressCallback 进度回调
type ProgressCallback func(event *ProgressEvent)

// ProgressEvent 进度事件
type ProgressEvent struct {
	Stage       string                 `json:"stage"`
	Progress    float32                `json:"progress"`
	Message     string                 `json:"message"`
	TaskName    string                 `json:"task_name,omitempty"`
	TaskStatus  string                 `json:"task_status,omitempty"`
	PartialData map[string]interface{} `json:"partial_data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewResearchAgent 创建研究 Agent
func NewResearchAgent(chatModel model.ChatModel, tools []tool.InvokableTool, config Config) *ResearchAgent {
	// 设置默认值 - 提高标准，确保深入研究
	if config.MaxIterations == 0 {
		config.MaxIterations = 20
	}
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Minute
	}
	if config.MinSources == 0 {
		config.MinSources = 4 // 提高最少来源数
	}
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.75 // 提高置信度阈值
	}
	if config.MaxSearchDepth == 0 {
		config.MaxSearchDepth = 4 // 增加搜索深度
	}
	// 默认启用反证检查和结构化输出
	if !config.EnableCritic {
		config.EnableCritic = true
	}
	if !config.EnableStructured {
		config.EnableStructured = true
	}
	
	agent := &ResearchAgent{
		chatModel:       chatModel,
		tools:           tools,
		config:          config,
		callbacks:       make([]ProgressCallback, 0),
		planner:         NewResearchPlanner(chatModel),
		critic:          NewCritic(chatModel),
		reportGenerator: NewReportGenerator(chatModel),
	}
	
	return agent
}

// RegisterCallback 注册进度回调
func (a *ResearchAgent) RegisterCallback(callback ProgressCallback) {
	a.callbacks = append(a.callbacks, callback)
}

func (a *ResearchAgent) emitProgress(event *ProgressEvent) {
	for _, cb := range a.callbacks {
		cb(event)
	}
}

// Run 执行研究 - 使用 Agentic RAG + Plan-Execute-Critic-Reflect 模式
func (a *ResearchAgent) Run(ctx context.Context, query string) (*Result, error) {
	startTime := time.Now()

	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	result := &Result{
		Query:         query,
		Steps:         make([]Step, 0),
		ToolsUsed:     make([]string, 0),
		CriticResults: make([]*CriticResult, 0),
	}

	// ========== Phase 1: Enhanced Planning (增强规划阶段) ==========
	a.emitProgress(&ProgressEvent{
		Stage: "planning", Progress: 0.05, 
		Message: "正在分析问题，制定研究计划...", 
		Timestamp: time.Now(),
	})

	enhancedPlan, err := a.planner.CreatePlan(ctx, query)
	if err != nil {
		enhancedPlan = a.planner.defaultPlan(query)
	}

	// 转换为兼容格式
	plan := &ResearchPlan{
		MainQuestion:  enhancedPlan.MainQuestion,
		SubQuestions:  make([]string, len(enhancedPlan.SubQuestions)),
		RequiredTools: enhancedPlan.RequiredTools,
		ExpectedDepth: enhancedPlan.ExpectedDepth,
	}
	for i, sq := range enhancedPlan.SubQuestions {
		plan.SubQuestions[i] = sq.Question
	}

	result.Steps = append(result.Steps, Step{
		StepNumber:  1,
		Phase:       "planning",
		Thought:     fmt.Sprintf("制定研究计划：%d个子问题，证据门槛%.0f%%", len(plan.SubQuestions), enhancedPlan.EvidenceThreshold*100),
		Action:      "create_plan",
		Observation: fmt.Sprintf("子问题: %v", plan.SubQuestions),
		Timestamp:   time.Now(),
	})

	// ========== Phase 2: Execute-Critic-Reflect Loop (Agentic RAG核心) ==========
	toolsUsedSet := make(map[string]bool)
	collectedInfo := make([]string, 0)
	searchDepth := 0
	stepNumber := 2

	for searchDepth < a.config.MaxSearchDepth {
		select {
		case <-ctx.Done():
			result.Error = "执行超时"
			result.ExecutionTime = time.Since(startTime).Milliseconds()
			return result, ctx.Err()
		default:
		}

		// ---- Execute: 执行搜索 ----
		progress := 0.1 + float32(searchDepth)*0.2
		a.emitProgress(&ProgressEvent{
			Stage: "executing", Progress: progress,
			Message: fmt.Sprintf("第%d轮信息收集中...", searchDepth+1),
			Timestamp: time.Now(),
		})

		// 使用增强规划器获取下一个子问题
		nextSQ := a.planner.GetNextSubQuestion(enhancedPlan)
		subQuestions := plan.SubQuestions
		if nextSQ != nil {
			// 优先处理规划器推荐的子问题
			subQuestions = []string{nextSQ.Question}
		}

		// 为每个子问题执行搜索
		for i, subQ := range subQuestions {
			if len(collectedInfo) >= a.config.MaxIterations {
				break
			}

			// 根据搜索策略选择工具
			var toolName string
			if nextSQ != nil && i == 0 {
				toolName = a.planner.GetToolForStrategy(nextSQ.SearchStrategy, searchDepth)
			}

			searchResult, usedTool := a.executeSearchWithTool(ctx, subQ, searchDepth, toolName)
			if searchResult != "" {
				collectedInfo = append(collectedInfo, searchResult)
				toolsUsedSet[usedTool] = true

				// 更新子问题的证据计数
				if nextSQ != nil && i == 0 {
					nextSQ.EvidenceCount++
				}

				result.Steps = append(result.Steps, Step{
					StepNumber:  stepNumber,
					Phase:       "searching",
					Thought:     fmt.Sprintf("搜索: %s", subQ),
					Action:      usedTool,
					Observation: truncateString(searchResult, 500),
					Quality:     0.8,
					Timestamp:   time.Now(),
				})
				stepNumber++
			}
		}

		// ---- Critic: 反证检查（Agentic RAG关键组件）----
		if a.config.EnableCritic && len(collectedInfo) >= 2 {
			a.emitProgress(&ProgressEvent{
				Stage: "critic", Progress: progress + 0.08,
				Message: "执行反证检查，验证证据一致性...",
				Timestamp: time.Now(),
			})

			// 生成当前综合（用于反证检查）
			currentSynthesis := a.quickSynthesis(ctx, query, collectedInfo)
			
			criticResult, _ := a.critic.Evaluate(ctx, query, currentSynthesis, collectedInfo)
			if criticResult != nil {
				result.CriticResults = append(result.CriticResults, criticResult)

				result.Steps = append(result.Steps, Step{
					StepNumber:  stepNumber,
					Phase:       "critic",
					Thought:     fmt.Sprintf("反证检查：发现%d个矛盾，%d个证据缺口", len(criticResult.Contradictions), len(criticResult.EvidenceGaps)),
					Action:      "critic_evaluate",
					Observation: criticResult.Reasoning,
					Quality:     criticResult.QualityScore,
					Timestamp:   time.Now(),
				})
				stepNumber++

				// 根据反证结果更新计划
				if criticResult.ShouldContinue && len(criticResult.NextSearchQueries) > 0 {
					enhancedPlan = a.planner.UpdatePlan(ctx, enhancedPlan, criticResult)
					plan.SubQuestions = criticResult.NextSearchQueries
				}

				// 如果反证检查通过且质量足够，可以提前结束
				if !criticResult.ShouldContinue && criticResult.QualityScore >= a.config.ConfidenceThreshold {
					result.ConfidenceScore = criticResult.QualityScore
					break
				}
			}
		}

		// ---- Evaluate: 评估信息饱和度 ----
		a.emitProgress(&ProgressEvent{
			Stage: "evaluating", Progress: progress + 0.12,
			Message: "评估信息完整性...",
			Timestamp: time.Now(),
		})

		evaluation := a.evaluateInformation(ctx, query, collectedInfo)
		
		result.Steps = append(result.Steps, Step{
			StepNumber:  stepNumber,
			Phase:       "evaluating",
			Thought:     fmt.Sprintf("置信度: %.0f%%, 完整性: %v", evaluation.ConfidenceScore*100, evaluation.IsComplete),
			Action:      "evaluate",
			Observation: evaluation.Suggestion,
			Quality:     evaluation.QualityScore,
			Timestamp:   time.Now(),
		})
		stepNumber++

		// ---- Reflect: 决定是否继续 ----
		// 提高终止条件，确保深入研究
		canStop := false
		
		// 条件1：评估完整且置信度达标且来源足够
		if evaluation.IsComplete && 
		   evaluation.ConfidenceScore >= a.config.ConfidenceThreshold && 
		   len(collectedInfo) >= a.config.MinSources {
			canStop = true
		}
		
		// 条件2：已经进行了足够多轮搜索，且有基本的信息
		if searchDepth >= 2 && 
		   len(collectedInfo) >= a.config.MinSources && 
		   evaluation.ConfidenceScore >= 0.7 {
			canStop = true
		}
		
		// 条件3：反证检查通过（如果启用）
		if a.config.EnableCritic && len(result.CriticResults) > 0 {
			lastCritic := result.CriticResults[len(result.CriticResults)-1]
			// 只有当反证检查也认为可以停止时才停止
			if !lastCritic.ShouldContinue && lastCritic.QualityScore >= a.config.ConfidenceThreshold {
				canStop = true
			} else if lastCritic.ShouldContinue {
				// 反证检查认为需要继续，不能停止
				canStop = false
			}
		}
		
		if canStop {
			result.ConfidenceScore = evaluation.ConfidenceScore
			break
		}

		// ---- 反思并调整策略 ----
		a.emitProgress(&ProgressEvent{
			Stage: "reflecting", Progress: progress + 0.15,
			Message: "反思研究策略，准备深入搜索...",
			Timestamp: time.Now(),
		})

		if len(evaluation.MissingAspects) > 0 {
			plan.SubQuestions = evaluation.MissingAspects
		}

		result.Steps = append(result.Steps, Step{
			StepNumber:  stepNumber,
			Phase:       "reflecting",
			Thought:     fmt.Sprintf("需要补充: %v", evaluation.MissingAspects),
			Action:      "reflect",
			Observation: fmt.Sprintf("继续第%d轮搜索", searchDepth+2),
			Timestamp:   time.Now(),
		})
		stepNumber++

		searchDepth++
	}

	// ========== Phase 3: Structured Synthesis (结构化综合阶段) ==========
	a.emitProgress(&ProgressEvent{
		Stage: "synthesizing", Progress: 0.85,
		Message: "正在综合分析，生成研究报告...",
		Timestamp: time.Now(),
	})

	// 收集工具列表
	for tool := range toolsUsedSet {
		result.ToolsUsed = append(result.ToolsUsed, tool)
	}

	// 生成结构化报告
	if a.config.EnableStructured {
		metadata := ReportMetadata{
			Query:           query,
			GeneratedAt:     time.Now(),
			ExecutionTimeMs: time.Since(startTime).Milliseconds(),
			ToolsUsed:       result.ToolsUsed,
			TotalSteps:      len(result.Steps),
			SourceCount:     len(collectedInfo),
			Version:         "2.0",
		}

		structuredReport, err := a.reportGenerator.Generate(ctx, query, result.Steps, collectedInfo, metadata)
		if err == nil && structuredReport != nil {
			result.StructuredReport = structuredReport
			result.FinalAnswer = structuredReport.Markdown
			result.ConfidenceScore = structuredReport.Structured.ConfidenceScore
		} else {
			result.FinalAnswer = a.generateComprehensiveReport(ctx, query, result.Steps, collectedInfo)
		}
	} else {
		result.FinalAnswer = a.generateComprehensiveReport(ctx, query, result.Steps, collectedInfo)
	}

	result.Success = true
	result.SourceCount = len(collectedInfo)
	result.ExecutionTime = time.Since(startTime).Milliseconds()

	a.emitProgress(&ProgressEvent{
		Stage: "completed", Progress: 1.0, 
		Message: "研究完成",
		Timestamp: time.Now(),
		PartialData: map[string]interface{}{
			"tools_used":       result.ToolsUsed,
			"confidence_score": result.ConfidenceScore,
			"source_count":     result.SourceCount,
			"has_structured":   result.StructuredReport != nil,
		},
	})

	return result, nil
}

// quickSynthesis 快速综合（用于反证检查）
func (a *ResearchAgent) quickSynthesis(ctx context.Context, query string, evidence []string) string {
	if len(evidence) == 0 {
		return ""
	}
	
	// 简单拼接前几条证据的摘要
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("关于「%s」的初步发现：\n", query))
	for i, e := range evidence {
		if i >= 3 {
			break
		}
		summary := truncateString(e, 200)
		sb.WriteString(fmt.Sprintf("- %s\n", summary))
	}
	return sb.String()
}

// executeSearchWithTool 使用指定工具执行搜索
func (a *ResearchAgent) executeSearchWithTool(ctx context.Context, question string, depth int, preferredTool string) (string, string) {
	if preferredTool != "" {
		// 尝试使用指定的工具
		for _, t := range a.tools {
			info, _ := t.Info(ctx)
			if info != nil && info.Name == preferredTool {
				args := fmt.Sprintf(`{"query": "%s"}`, strings.ReplaceAll(question, `"`, `\"`))
				result, err := t.InvokableRun(ctx, args)
				if err == nil && result != "" {
					return result, preferredTool
				}
			}
		}
	}
	
	// 回退到默认搜索逻辑
	return a.executeSearch(ctx, question, depth)
}


// createResearchPlan 创建研究计划
func (a *ResearchAgent) createResearchPlan(ctx context.Context, query string) (*ResearchPlan, error) {
	planPrompt := fmt.Sprintf(`你是一个研究规划专家。请分析以下研究问题，并制定研究计划。

## 研究问题
%s

## 任务
请将这个问题分解为2-4个具体的子问题，以便进行深入研究。

请以JSON格式返回，格式如下：
{
  "main_question": "原始问题",
  "sub_questions": ["子问题1", "子问题2", "子问题3"],
  "required_tools": ["web_search", "wikipedia"],
  "expected_depth": 2
}

只返回JSON，不要其他内容。`, query)

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是一个研究规划专家，擅长将复杂问题分解为可研究的子问题。只返回JSON格式的研究计划。"},
		{Role: schema.User, Content: planPrompt},
	}

	response, err := a.chatModel.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}

	// 解析JSON
	plan := &ResearchPlan{}
	content := strings.TrimSpace(response.Content)
	// 移除可能的markdown代码块标记
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	if err := json.Unmarshal([]byte(content), plan); err != nil {
		// 解析失败，返回默认计划
		return &ResearchPlan{
			MainQuestion:  query,
			SubQuestions:  []string{query},
			RequiredTools: []string{"web_search", "wikipedia"},
			ExpectedDepth: 2,
		}, nil
	}

	// 确保至少有一个子问题
	if len(plan.SubQuestions) == 0 {
		plan.SubQuestions = []string{query}
	}

	return plan, nil
}

// executeSearch 执行单次搜索
// 直接调用工具进行搜索，不依赖 LLM 的工具调用功能
func (a *ResearchAgent) executeSearch(ctx context.Context, question string, depth int) (string, string) {
	// 根据搜索深度和问题类型选择工具
	// 第一轮优先使用 web_search，后续轮次可以使用其他工具
	var selectedTool tool.InvokableTool
	var toolName string
	
	// 根据问题内容和深度选择工具
	questionLower := strings.ToLower(question)
	
	if depth == 0 {
		// 第一轮：优先使用 web_search
		for _, t := range a.tools {
			info, _ := t.Info(ctx)
			if info != nil && info.Name == "web_search" {
				selectedTool = t
				toolName = info.Name
				break
			}
		}
	} else if strings.Contains(questionLower, "论文") || strings.Contains(questionLower, "研究") || 
		strings.Contains(questionLower, "学术") || strings.Contains(questionLower, "paper") ||
		strings.Contains(questionLower, "arxiv") {
		// 学术相关问题：使用 arxiv
		for _, t := range a.tools {
			info, _ := t.Info(ctx)
			if info != nil && info.Name == "arxiv_search" {
				selectedTool = t
				toolName = info.Name
				break
			}
		}
	} else if strings.Contains(questionLower, "百科") || strings.Contains(questionLower, "wiki") ||
		strings.Contains(questionLower, "定义") || strings.Contains(questionLower, "概念") {
		// 百科类问题：使用 wikipedia
		for _, t := range a.tools {
			info, _ := t.Info(ctx)
			if info != nil && info.Name == "wikipedia" {
				selectedTool = t
				toolName = info.Name
				break
			}
		}
	}
	
	// 如果没有选中特定工具，默认使用 web_search
	if selectedTool == nil {
		for _, t := range a.tools {
			info, _ := t.Info(ctx)
			if info != nil && info.Name == "web_search" {
				selectedTool = t
				toolName = info.Name
				break
			}
		}
	}
	
	// 如果还是没有找到工具，使用第一个可用的工具
	if selectedTool == nil && len(a.tools) > 0 {
		selectedTool = a.tools[0]
		info, _ := selectedTool.Info(ctx)
		if info != nil {
			toolName = info.Name
		} else {
			toolName = "unknown"
		}
	}
	
	if selectedTool == nil {
		return "", ""
	}
	
	// 构建工具参数
	args := fmt.Sprintf(`{"query": "%s"}`, strings.ReplaceAll(question, `"`, `\"`))
	
	// 直接调用工具
	result, err := selectedTool.InvokableRun(ctx, args)
	if err != nil {
		return fmt.Sprintf("搜索失败: %v", err), toolName
	}
	
	if result == "" {
		return "", toolName
	}
	
	return result, toolName
}

// evaluateInformation 评估收集到的信息
func (a *ResearchAgent) evaluateInformation(ctx context.Context, query string, collectedInfo []string) *EvaluationResult {
	if len(collectedInfo) == 0 {
		return &EvaluationResult{
			IsComplete:      false,
			ConfidenceScore: 0,
			MissingAspects:  []string{query},
			QualityScore:    0,
			Suggestion:      "尚未收集到任何信息，需要开始搜索",
		}
	}

	// 构建评估提示
	infoSummary := make([]string, 0, len(collectedInfo))
	for i, info := range collectedInfo {
		// 截取每条信息的摘要
		summary := truncateString(info, 300)
		infoSummary = append(infoSummary, fmt.Sprintf("%d. %s", i+1, summary))
	}

	evalPrompt := fmt.Sprintf(`你是一个研究质量评估专家。请评估以下收集到的信息是否足以回答研究问题。

## 研究问题
%s

## 已收集的信息（共%d条）
%s

## 评估任务
请评估：
1. 信息是否完整？能否全面回答问题？
2. 信息的置信度（0-1）
3. 还缺少哪些方面的信息？
4. 信息质量如何？

请以JSON格式返回：
{
  "is_complete": true/false,
  "confidence_score": 0.8,
  "missing_aspects": ["缺失方面1", "缺失方面2"],
  "quality_score": 0.7,
  "suggestion": "建议说明"
}

只返回JSON，不要其他内容。`, query, len(collectedInfo), strings.Join(infoSummary, "\n"))

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是一个研究质量评估专家，擅长评估信息的完整性和可靠性。只返回JSON格式的评估结果。"},
		{Role: schema.User, Content: evalPrompt},
	}

	response, err := a.chatModel.Generate(ctx, messages)
	if err != nil {
		// 评估失败，使用启发式方法
		return a.heuristicEvaluation(collectedInfo)
	}

	// 解析JSON
	result := &EvaluationResult{}
	content := strings.TrimSpace(response.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	if err := json.Unmarshal([]byte(content), result); err != nil {
		return a.heuristicEvaluation(collectedInfo)
	}

	return result
}

// heuristicEvaluation 启发式评估（当LLM评估失败时使用）
// 更严格的评估标准
func (a *ResearchAgent) heuristicEvaluation(collectedInfo []string) *EvaluationResult {
	count := len(collectedInfo)
	
	// 计算总字符数和信息多样性
	totalChars := 0
	uniqueContent := make(map[string]bool)
	for _, info := range collectedInfo {
		totalChars += len(info)
		// 简单的内容去重检查
		key := info
		if len(key) > 100 {
			key = key[:100]
		}
		uniqueContent[key] = true
	}
	uniqueCount := len(uniqueContent)
	
	// 更严格的评估标准
	var confidence float64
	var isComplete bool
	var missingAspects []string
	
	switch {
	case count >= 8 && uniqueCount >= 6:
		confidence = 0.9
		isComplete = true
	case count >= 6 && uniqueCount >= 4:
		confidence = 0.8
		isComplete = true
	case count >= 4 && uniqueCount >= 3:
		confidence = 0.65
		isComplete = false
		missingAspects = []string{"需要更多来源验证"}
	case count >= 3:
		confidence = 0.5
		isComplete = false
		missingAspects = []string{"信息来源不足", "需要深入搜索"}
	case count >= 2:
		confidence = 0.35
		isComplete = false
		missingAspects = []string{"证据严重不足", "需要多角度搜索"}
	default:
		confidence = 0.2
		isComplete = false
		missingAspects = []string{"几乎没有有效信息", "需要重新搜索"}
	}

	// 根据内容丰富度调整
	contentRichness := float64(totalChars) / 8000.0 // 期望至少8000字符
	if contentRichness < 1.0 {
		confidence *= (0.7 + 0.3*contentRichness)
		if len(missingAspects) == 0 {
			missingAspects = []string{"内容深度不足"}
		}
	}
	
	quality := min(1.0, float64(totalChars)/10000.0)

	return &EvaluationResult{
		IsComplete:      isComplete,
		ConfidenceScore: confidence,
		MissingAspects:  missingAspects,
		QualityScore:    quality,
		Suggestion:      fmt.Sprintf("已收集%d条信息（%d条独立），置信度%.0f%%", count, uniqueCount, confidence*100),
	}
}

// generateComprehensiveReport 生成综合研究报告
func (a *ResearchAgent) generateComprehensiveReport(ctx context.Context, query string, steps []Step, collectedInfo []string) string {
	if len(collectedInfo) == 0 {
		return fmt.Sprintf(`## 研究报告

### 概述
针对问题「%s」的研究未能收集到足够的信息。

### 建议
请稍后重试，或尝试更具体的问题描述。`, query)
	}

	// 使用新的独立上下文，不受原上下文超时影响
	// 报告生成最多给 3 分钟
	reportCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// 准备信息摘要（限制长度避免超出 token 限制）
	infoSummary := make([]string, 0, len(collectedInfo))
	totalLen := 0
	maxTotalLen := 6000 // 限制总长度，给报告生成留空间
	for i, info := range collectedInfo {
		// 清理搜索结果的标题
		cleanedInfo := cleanSearchResult(info)
		summary := truncateString(cleanedInfo, 500)
		if totalLen+len(summary) > maxTotalLen {
			break
		}
		totalLen += len(summary)
		infoSummary = append(infoSummary, fmt.Sprintf("【资料%d】%s", i+1, summary))
	}

	// 构建报告生成提示
	reportPrompt := fmt.Sprintf(`基于以下研究资料，为问题「%s」撰写一份专业的研究报告。

## 研究资料
%s

## 报告格式要求
请按以下结构输出报告（使用Markdown格式）：

## 研究报告

### 概述
用3-5句话直接回答用户的问题，给出核心结论。

### 主要发现
列出3-5个关键发现，每个发现用简洁的语言说明。

### 详细分析
对主要发现进行深入分析和解释。

### 结论与建议
总结研究结论，给出实用建议。

## 注意事项
- 直接输出报告，不要有多余的开场白
- 综合分析资料，不要简单复制粘贴
- 原始资料会单独展示，报告中不需要引用来源编号`, query, strings.Join(infoSummary, "\n\n"))

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是专业的研究报告撰写专家。请基于提供的资料撰写结构清晰、分析深入的研究报告。"},
		{Role: schema.User, Content: reportPrompt},
	}

	response, err := a.chatModel.Generate(reportCtx, messages)
	if err != nil {
		// 生成失败，返回本地生成的报告
		return a.generateLocalReport(query, collectedInfo)
	}

	if response.Content == "" {
		return a.generateLocalReport(query, collectedInfo)
	}

	return response.Content
}

// cleanSearchResult 清理搜索结果，移除标题和分隔符
func cleanSearchResult(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过标题行和分隔符
		if line == "" || strings.HasPrefix(line, "网络搜索结果") || strings.HasPrefix(line, "===") || strings.HasPrefix(line, "---") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// generateLocalReport 本地生成报告（不依赖 LLM）
// 注意：这个报告不应该包含原始搜索结果，原始结果会通过 evidence 单独展示
func (a *ResearchAgent) generateLocalReport(query string, collectedInfo []string) string {
	var sb strings.Builder
	
	sb.WriteString("## 研究报告\n\n")
	
	// 概述 - 基于收集的信息数量给出概述
	sb.WriteString("### 概述\n\n")
	sb.WriteString(fmt.Sprintf("针对问题「%s」进行了深度研究，共收集到 %d 条相关信息。", query, len(collectedInfo)))
	
	// 尝试从收集的信息中提取核心结论
	if len(collectedInfo) > 0 {
		conclusion := extractConclusion(collectedInfo[0])
		if conclusion != "" {
			sb.WriteString(conclusion)
		}
	}
	sb.WriteString("\n\n")
	
	// 主要发现 - 提取每条信息的核心观点
	sb.WriteString("### 主要发现\n\n")
	findingCount := 0
	for _, info := range collectedInfo {
		if findingCount >= 5 {
			break
		}
		// 提取信息的核心观点
		keyPoint := extractKeyPoint(info, 200)
		if keyPoint != "" && !strings.HasPrefix(keyPoint, "网络搜索") {
			findingCount++
			sb.WriteString(fmt.Sprintf("- %s\n", keyPoint))
		}
	}
	if findingCount == 0 {
		sb.WriteString("- 已收集相关资料，详见下方研究证据\n")
	}
	sb.WriteString("\n")
	
	// 结论与建议
	sb.WriteString("### 结论与建议\n\n")
	sb.WriteString(fmt.Sprintf("基于收集到的 %d 条研究资料，已获取关于「%s」的相关信息。", len(collectedInfo), query))
	sb.WriteString("如需了解详细内容，请展开下方「研究证据」查看原始资料。\n")
	
	return sb.String()
}

// extractConclusion 从文本中提取结论性内容
func extractConclusion(text string) string {
	// 移除搜索结果的标题和分隔符
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过标题行和分隔符
		if line == "" || strings.HasPrefix(line, "网络搜索结果") || strings.HasPrefix(line, "===") || strings.HasPrefix(line, "---") {
			continue
		}
		// 跳过编号开头的行（如 "1. xxx"）
		if len(line) > 2 && line[0] >= '0' && line[0] <= '9' && line[1] == '.' {
			continue
		}
		// 找到第一行有意义的内容
		if len(line) > 20 {
			// 截取合适长度
			if len(line) > 150 {
				line = line[:150] + "..."
			}
			return " " + line
		}
	}
	return ""
}

// extractKeyPoint 从文本中提取关键点
func extractKeyPoint(text string, maxLen int) string {
	// 移除搜索结果的标题行
	lines := strings.Split(text, "\n")
	var content string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "网络搜索结果") || strings.HasPrefix(line, "===") {
			continue
		}
		// 找到第一行有实质内容的文本
		if len(line) > 10 {
			content = line
			break
		}
	}
	
	if content == "" {
		return ""
	}
	
	if len(content) > maxLen {
		content = content[:maxLen] + "..."
	}
	
	return content
}

// truncateString 截取字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// min 返回两个float64中的较小值
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// buildSystemPrompt 构建系统提示（保留兼容性）
func (a *ResearchAgent) buildSystemPrompt() string {
	var toolDescs []string
	for _, t := range a.tools {
		info, _ := t.Info(context.Background())
		if info != nil {
			toolDescs = append(toolDescs, fmt.Sprintf("- %s: %s", info.Name, info.Desc))
		}
	}

	return fmt.Sprintf(`你是一个专业的研究助手。你可以使用以下工具来收集信息：

%s

请根据用户的问题，使用合适的工具收集信息，然后给出全面、准确的答案。
如果需要多个来源的信息，请依次使用不同的工具。
最后，请综合所有收集到的信息，给出结构化的研究报告。`, strings.Join(toolDescs, "\n"))
}

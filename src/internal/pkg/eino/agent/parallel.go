// Package agent 提供并行多Agent研究编排器
// 借鉴 Claude Tasks 的 Plan → Parallel Execute → Sync Summarize 模式
package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// MaxParallelAgents 最大并行Agent数
const MaxParallelAgents = 3

// ParallelOrchestrator 并行研究编排器
// 流程: Plan → Fork(最多3个Agent并行) → Join(同步汇总)
type ParallelOrchestrator struct {
	chatModel       model.ChatModel
	tools           []tool.InvokableTool
	config          Config
	planner         *ResearchPlanner
	critic          *Critic
	reportGenerator *ReportGenerator
	callbacks       []ProgressCallback
}

// SubAgentTask 子Agent任务定义
type SubAgentTask struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Query          string   `json:"query"`
	SearchStrategy string   `json:"search_strategy"` // web, arxiv, wiki, mixed
	Tools          []string `json:"tools"`
	Priority       int      `json:"priority"`
}

// SubAgentResult 子Agent执行结果
type SubAgentResult struct {
	TaskID      string   `json:"task_id"`
	TaskName    string   `json:"task_name"`
	Success     bool     `json:"success"`
	Evidence    []string `json:"evidence"`
	ToolsUsed   []string `json:"tools_used"`
	Steps       []Step   `json:"steps"`
	Error       string   `json:"error,omitempty"`
	Duration    int64    `json:"duration_ms"`
}

// NewParallelOrchestrator 创建并行编排器
func NewParallelOrchestrator(chatModel model.ChatModel, tools []tool.InvokableTool, config Config) *ParallelOrchestrator {
	if config.MaxIterations == 0 {
		config.MaxIterations = 20
	}
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Minute
	}
	if config.MinSources == 0 {
		config.MinSources = 4
	}
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.75
	}
	if config.MaxSearchDepth == 0 {
		config.MaxSearchDepth = 3
	}
	config.EnableCritic = true
	config.EnableStructured = true

	return &ParallelOrchestrator{
		chatModel:       chatModel,
		tools:           tools,
		config:          config,
		planner:         NewResearchPlanner(chatModel),
		critic:          NewCritic(chatModel),
		reportGenerator: NewReportGenerator(chatModel),
		callbacks:       make([]ProgressCallback, 0),
	}
}

// RegisterCallback 注册进度回调
// P0 修复：每次注册时重置回调列表，防止跨会话回调累积
func (o *ParallelOrchestrator) RegisterCallback(callback ProgressCallback) {
	o.callbacks = []ProgressCallback{callback}
}

// ClearCallbacks 清除所有回调（用于会话结束后释放资源）
func (o *ParallelOrchestrator) ClearCallbacks() {
	o.callbacks = nil
}

func (o *ParallelOrchestrator) emit(event *ProgressEvent) {
	for _, cb := range o.callbacks {
		cb(event)
	}
}

// Run 执行并行研究
// Phase 1: Plan (规划) → Phase 2: Fork & Execute (并行执行) → Phase 3: Join & Synthesize (汇总)
func (o *ParallelOrchestrator) Run(ctx context.Context, query string) (*Result, error) {
	startTime := time.Now()

	if o.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.config.Timeout)
		defer cancel()
	}

	result := &Result{
		Query:         query,
		Steps:         make([]Step, 0),
		ToolsUsed:     make([]string, 0),
		CriticResults: make([]*CriticResult, 0),
	}

	// ========== Phase 1: Plan (智能规划) ==========
	o.emit(&ProgressEvent{
		Stage: "planning", Progress: 0.05,
		Message:   "正在分析问题，制定并行研究计划...",
		Timestamp: time.Now(),
	})

	plan, err := o.planner.CreatePlan(ctx, query)
	if err != nil {
		plan = o.planner.defaultPlan(query)
	}

	// 将子问题分配为并行任务（最多3个）
	tasks := o.createSubAgentTasks(plan)

	result.Steps = append(result.Steps, Step{
		StepNumber: 1, Phase: "planning",
		Thought:     fmt.Sprintf("制定并行研究计划：%d个子任务将由%d个Agent并行执行", len(tasks), len(tasks)),
		Action:      "create_parallel_plan",
		Observation: fmt.Sprintf("任务: %s", formatTaskNames(tasks)),
		Timestamp:   time.Now(),
	})

	o.emit(&ProgressEvent{
		Stage: "planning", Progress: 0.10,
		Message:   fmt.Sprintf("研究计划就绪，启动 %d 个并行Agent...", len(tasks)),
		Timestamp: time.Now(),
		PartialData: map[string]interface{}{
			"tasks": tasks,
			"total": len(tasks),
		},
	})

	// ========== Phase 2: Fork & Execute (并行执行) ==========
	agentResults := o.executeParallel(ctx, tasks)

	// 收集所有证据和工具
	allEvidence := make([]string, 0)
	toolsUsedSet := make(map[string]bool)
	stepNumber := 2

	for _, ar := range agentResults {
		allEvidence = append(allEvidence, ar.Evidence...)
		for _, t := range ar.ToolsUsed {
			toolsUsedSet[t] = true
		}
		result.Steps = append(result.Steps, ar.Steps...)
		stepNumber += len(ar.Steps)

		status := "✓ 完成"
		if !ar.Success {
			status = "✗ 失败: " + ar.Error
		}
		result.Steps = append(result.Steps, Step{
			StepNumber: stepNumber, Phase: "agent_complete",
			Thought:     fmt.Sprintf("Agent [%s] %s，收集 %d 条证据", ar.TaskName, status, len(ar.Evidence)),
			Action:      "agent_result",
			Observation: fmt.Sprintf("工具: %v", ar.ToolsUsed),
			Timestamp:   time.Now(),
		})
		stepNumber++
	}

	// ========== Phase 2.5: Critic (反证检查) ==========
	if len(allEvidence) >= 2 {
		o.emit(&ProgressEvent{
			Stage: "critic", Progress: 0.75,
			Message:   "执行反证检查，验证证据一致性...",
			Timestamp: time.Now(),
		})

		synthesis := o.quickSynthesis(query, allEvidence)
		criticResult, _ := o.critic.Evaluate(ctx, query, synthesis, allEvidence)
		if criticResult != nil {
			result.CriticResults = append(result.CriticResults, criticResult)
			result.ConfidenceScore = criticResult.QualityScore
		}
	}

	// ========== Phase 3: Join & Synthesize (同步汇总) ==========
	o.emit(&ProgressEvent{
		Stage: "synthesizing", Progress: 0.85,
		Message:   "所有Agent已完成，正在综合分析生成报告...",
		Timestamp: time.Now(),
	})

	for t := range toolsUsedSet {
		result.ToolsUsed = append(result.ToolsUsed, t)
	}

	metadata := ReportMetadata{
		Query:           query,
		GeneratedAt:     time.Now(),
		ExecutionTimeMs: time.Since(startTime).Milliseconds(),
		ToolsUsed:       result.ToolsUsed,
		TotalSteps:      len(result.Steps),
		SourceCount:     len(allEvidence),
		Version:         "3.0-parallel",
	}

	structuredReport, err := o.reportGenerator.Generate(ctx, query, result.Steps, allEvidence, metadata)
	if err == nil && structuredReport != nil {
		result.StructuredReport = structuredReport
		result.FinalAnswer = structuredReport.Markdown
		if result.ConfidenceScore == 0 {
			result.ConfidenceScore = structuredReport.Structured.ConfidenceScore
		}
	} else {
		result.FinalAnswer = o.fallbackReport(ctx, query, allEvidence)
	}

	result.Success = true
	result.SourceCount = len(allEvidence)
	result.ExecutionTime = time.Since(startTime).Milliseconds()

	o.emit(&ProgressEvent{
		Stage: "completed", Progress: 1.0,
		Message:   "研究完成",
		Timestamp: time.Now(),
		PartialData: map[string]interface{}{
			"tools_used":       result.ToolsUsed,
			"confidence_score": result.ConfidenceScore,
			"source_count":     result.SourceCount,
			"parallel_agents":  len(tasks),
		},
	})

	return result, nil
}

// createSubAgentTasks 从研究计划创建并行子任务（最多3个）
func (o *ParallelOrchestrator) createSubAgentTasks(plan *EnhancedPlan) []SubAgentTask {
	tasks := make([]SubAgentTask, 0, MaxParallelAgents)

	// 按优先级排序子问题，取前3个
	sorted := make([]SubQuestion, len(plan.SubQuestions))
	copy(sorted, plan.SubQuestions)
	// 简单冒泡排序按优先级
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Priority < sorted[i].Priority {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	count := len(sorted)
	if count > MaxParallelAgents {
		count = MaxParallelAgents
	}

	for i := 0; i < count; i++ {
		sq := sorted[i]
		taskTools := o.getToolsForStrategy(sq.SearchStrategy)
		tasks = append(tasks, SubAgentTask{
			ID:             fmt.Sprintf("agent_%d", i+1),
			Name:           fmt.Sprintf("Agent-%d", i+1),
			Query:          sq.Question,
			SearchStrategy: sq.SearchStrategy,
			Tools:          taskTools,
			Priority:       sq.Priority,
		})
	}

	// 至少保证1个任务
	if len(tasks) == 0 {
		tasks = append(tasks, SubAgentTask{
			ID:             "agent_1",
			Name:           "Agent-1",
			Query:          plan.MainQuestion,
			SearchStrategy: "mixed",
			Tools:          []string{"web_search_prime", "web_search", "wikipedia"},
			Priority:       1,
		})
	}

	return tasks
}

// getToolsForStrategy 根据搜索策略返回工具列表
func (o *ParallelOrchestrator) getToolsForStrategy(strategy string) []string {
	switch strategy {
	case "web":
		return []string{"web_search_prime", "web_search", "web_reader"}
	case "arxiv":
		return []string{"arxiv_search", "web_search_prime"}
	case "wiki":
		return []string{"wikipedia", "web_search"}
	case "mixed":
		return []string{"web_search_prime", "wikipedia", "arxiv_search", "web_reader"}
	default:
		return []string{"web_search_prime", "web_search"}
	}
}

// executeParallel 并行执行所有子Agent任务（限制最多 MaxParallelAgents 个并发）
func (o *ParallelOrchestrator) executeParallel(ctx context.Context, tasks []SubAgentTask) []SubAgentResult {
	results := make([]SubAgentResult, len(tasks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, MaxParallelAgents)

	for i, task := range tasks {
		wg.Add(1)
		sem <- struct{}{} // 获取信号量，超过 MaxParallelAgents 时阻塞
		go func(idx int, t SubAgentTask) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			o.emit(&ProgressEvent{
				Stage: "executing", Progress: 0.15 + float32(idx)*0.05,
				Message:    fmt.Sprintf("[%s] 开始调研: %s", t.Name, truncateString(t.Query, 50)),
				TaskName:   t.Name,
				TaskStatus: "running",
				Timestamp:  time.Now(),
			})

			result := o.runSubAgent(ctx, t)
			results[idx] = result

			status := "completed"
			msg := fmt.Sprintf("[%s] 完成，收集 %d 条证据", t.Name, len(result.Evidence))
			if !result.Success {
				status = "failed"
				msg = fmt.Sprintf("[%s] 失败: %s", t.Name, result.Error)
			}

			o.emit(&ProgressEvent{
				Stage: "executing", Progress: 0.30 + float32(idx)*0.15,
				Message:    msg,
				TaskName:   t.Name,
				TaskStatus: status,
				Timestamp:  time.Now(),
				PartialData: map[string]interface{}{
					"task_id":       t.ID,
					"evidence_count": len(result.Evidence),
					"tools_used":    result.ToolsUsed,
				},
			})
		}(i, task)
	}

	wg.Wait()
	return results
}

// runSubAgent 执行单个子Agent的研究任务
func (o *ParallelOrchestrator) runSubAgent(ctx context.Context, task SubAgentTask) SubAgentResult {
	startTime := time.Now()
	result := SubAgentResult{
		TaskID:    task.ID,
		TaskName:  task.Name,
		Evidence:  make([]string, 0),
		ToolsUsed: make([]string, 0),
		Steps:     make([]Step, 0),
	}

	// 每个子Agent执行多轮搜索（最多2轮）
	maxRounds := 2
	toolsUsedSet := make(map[string]bool)
	stepNum := 1

	for round := 0; round < maxRounds; round++ {
		select {
		case <-ctx.Done():
			result.Error = "context cancelled"
			result.Duration = time.Since(startTime).Milliseconds()
			return result
		default:
		}

		// 选择工具并执行搜索
		searchQuery := task.Query
		if round > 0 {
			searchQuery = task.Query + " 深入分析"
		}

		for _, preferredToolName := range task.Tools {
			if len(result.Evidence) >= 4 {
				break // 单个Agent最多收集4条证据
			}

			t := o.findTool(ctx, preferredToolName)
			if t == nil {
				continue
			}

			info, _ := t.Info(ctx)
			toolName := preferredToolName
			if info != nil {
				toolName = info.Name
			}

			args := buildToolArgs(toolName, searchQuery)
			searchResult, err := t.InvokableRun(ctx, args)
			if err != nil || searchResult == "" {
				continue
			}

			result.Evidence = append(result.Evidence, searchResult)
			toolsUsedSet[toolName] = true

			// 主动调度 web_reader：从搜索结果中提取 URL 并深度读取
			deepContents := o.deepReadURLs(ctx, searchResult, 1)
			for _, dc := range deepContents {
				result.Evidence = append(result.Evidence, dc)
				toolsUsedSet["web_reader"] = true
			}

			result.Steps = append(result.Steps, Step{
				StepNumber: stepNum,
				Phase:      "searching",
				Thought:    fmt.Sprintf("[%s] 搜索: %s", task.Name, truncateString(searchQuery, 60)),
				Action:     toolName,
				Observation: truncateString(searchResult, 500),
				Quality:    0.8,
				Timestamp:  time.Now(),
			})
			stepNum++

			if len(deepContents) > 0 {
				result.Steps = append(result.Steps, Step{
					StepNumber: stepNum,
					Phase:      "deep_reading",
					Thought:    fmt.Sprintf("[%s] 深度读取 %d 个网页", task.Name, len(deepContents)),
					Action:     "web_reader",
					Observation: fmt.Sprintf("成功获取 %d 篇网页全文", len(deepContents)),
					Quality:    0.9,
					Timestamp:  time.Now(),
				})
				stepNum++
			}

			// 第一轮每个工具只用一次，避免重复
			if round == 0 {
				break
			}
		}
	}

	for t := range toolsUsedSet {
		result.ToolsUsed = append(result.ToolsUsed, t)
	}
	result.Success = len(result.Evidence) > 0
	if !result.Success {
		result.Error = "未收集到有效证据"
	}
	result.Duration = time.Since(startTime).Milliseconds()
	return result
}

// findTool 根据名称查找工具
func (o *ParallelOrchestrator) findTool(ctx context.Context, name string) tool.InvokableTool {
	for _, t := range o.tools {
		info, _ := t.Info(ctx)
		if info != nil && info.Name == name {
			return t
		}
	}
	return nil
}

// buildToolArgs 根据工具名称构建参数JSON
// 修复：web_reader 需要 URL，不能直接把 query 当 URL 传入
func buildToolArgs(toolName, query string) string {
	escaped := strings.ReplaceAll(query, `"`, `\"`)
	switch toolName {
	case "web_search_prime":
		return fmt.Sprintf(`{"search_query": "%s", "content_size": "medium"}`, escaped)
	case "web_reader":
		// web_reader 需要 URL 参数，不能用搜索 query 直接当 URL
		// 仅当输入看起来像 URL 时使用 web_reader，否则返回空 JSON 让调用方跳过
		if strings.HasPrefix(query, "http://") || strings.HasPrefix(query, "https://") {
			return fmt.Sprintf(`{"url": "%s"}`, escaped)
		}
		// 非 URL 输入，返回空参数让调用失败，由外层 fallback
		return `{"url": ""}`
	case "zread_repo":
		return fmt.Sprintf(`{"operation": "search_doc", "repo_name": "%s", "query": "%s"}`, escaped, escaped)
	default:
		return fmt.Sprintf(`{"query": "%s"}`, escaped)
	}
}

// deepReadURLs 从搜索结果中提取 URL 并使用 web_reader 深度读取
func (o *ParallelOrchestrator) deepReadURLs(ctx context.Context, searchResult string, maxURLs int) []string {
	wrTool := o.findTool(ctx, "web_reader")
	if wrTool == nil {
		return nil
	}

	urls := extractURLsFromText(searchResult)
	if len(urls) == 0 {
		return nil
	}

	if len(urls) > maxURLs {
		urls = urls[:maxURLs]
	}

	var results []string
	for _, u := range urls {
		select {
		case <-ctx.Done():
			return results
		default:
		}

		escaped := strings.ReplaceAll(u, `"`, `\"`)
		args := fmt.Sprintf(`{"url": "%s"}`, escaped)

		readCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		content, err := wrTool.InvokableRun(readCtx, args)
		cancel()

		if err == nil && content != "" && len(content) > 100 {
			if len(content) > 5000 {
				content = content[:5000] + "\n...[内容已截断]"
			}
			results = append(results, content)
		}
	}
	return results
}

// quickSynthesis 快速综合证据
func (o *ParallelOrchestrator) quickSynthesis(query string, evidence []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("关于「%s」的初步发现：\n", query))
	for i, e := range evidence {
		if i >= 5 {
			break
		}
		sb.WriteString(fmt.Sprintf("- %s\n", truncateString(e, 200)))
	}
	return sb.String()
}

// fallbackReport 降级报告生成
func (o *ParallelOrchestrator) fallbackReport(ctx context.Context, query string, evidence []string) string {
	if len(evidence) == 0 {
		return fmt.Sprintf("## 研究报告\n\n针对问题「%s」的研究未能收集到足够信息。请稍后重试。", query)
	}

	// 尝试用LLM生成
	reportCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	infoSummary := make([]string, 0, len(evidence))
	totalLen := 0
	for i, info := range evidence {
		summary := truncateString(info, 500)
		if totalLen+len(summary) > 6000 {
			break
		}
		totalLen += len(summary)
		infoSummary = append(infoSummary, fmt.Sprintf("【资料%d】%s", i+1, summary))
	}

	prompt := fmt.Sprintf(`基于以下研究资料，为问题「%s」撰写专业研究报告。

## 研究资料
%s

## 格式要求（Markdown）
## 研究报告
### 概述（3-5句核心结论）
### 主要发现（3-5个关键点）
### 详细分析
### 结论与建议

直接输出报告，综合分析不要复制粘贴。`, query, strings.Join(infoSummary, "\n\n"))

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是专业研究报告撰写专家。"},
		{Role: schema.User, Content: prompt},
	}

	response, err := o.chatModel.Generate(reportCtx, messages)
	if err != nil || response.Content == "" {
		return o.localReport(query, evidence)
	}
	return response.Content
}

// localReport 本地生成简单报告
func (o *ParallelOrchestrator) localReport(query string, evidence []string) string {
	var sb strings.Builder
	sb.WriteString("## 研究报告\n\n")
	sb.WriteString("### 概述\n\n")
	sb.WriteString(fmt.Sprintf("针对问题「%s」进行了并行深度研究，共收集到 %d 条相关信息。\n\n", query, len(evidence)))
	sb.WriteString("### 主要发现\n\n")
	for i, e := range evidence {
		if i >= 5 {
			break
		}
		point := extractKeyPoint(e, 200)
		if point != "" {
			sb.WriteString(fmt.Sprintf("- %s\n", point))
		}
	}
	sb.WriteString("\n### 结论\n\n")
	sb.WriteString(fmt.Sprintf("基于 %d 条研究资料的综合分析，请展开下方证据查看详情。\n", len(evidence)))
	return sb.String()
}

// formatTaskNames 格式化任务名称列表
func formatTaskNames(tasks []SubAgentTask) string {
	names := make([]string, len(tasks))
	for i, t := range tasks {
		names[i] = fmt.Sprintf("%s(%s)", t.Name, truncateString(t.Query, 30))
	}
	return strings.Join(names, ", ")
}

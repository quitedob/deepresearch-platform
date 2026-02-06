// Package agent 提供研究评测框架
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// EvaluationCase 评测用例
type EvaluationCase struct {
	ID              string   `json:"id"`
	Query           string   `json:"query"`
	ExpectedPoints  []string `json:"expected_points"`   // 期望要点
	RequiredSources []string `json:"required_sources"`  // 必须引用的来源域名
	MinConfidence   float64  `json:"min_confidence"`    // 最低置信度
	MaxTimeSeconds  int      `json:"max_time_seconds"`  // 最大执行时间
	Tags            []string `json:"tags"`              // 标签（用于分类）
}

// TestEvaluationResult 评测结果（用于评测框架）
type TestEvaluationResult struct {
	CaseID          string        `json:"case_id"`
	Query           string        `json:"query"`
	Passed          bool          `json:"passed"`
	Score           float64       `json:"score"`           // 0-100
	PointsCovered   []string      `json:"points_covered"`  // 覆盖的要点
	PointsMissing   []string      `json:"points_missing"`  // 缺失的要点
	SourcesFound    []string      `json:"sources_found"`   // 找到的来源
	SourcesMissing  []string      `json:"sources_missing"` // 缺失的来源
	ConfidenceScore float64       `json:"confidence_score"`
	ExecutionTimeMs int64         `json:"execution_time_ms"`
	Errors          []string      `json:"errors,omitempty"`
	Timestamp       time.Time     `json:"timestamp"`
	Details         *Result       `json:"details,omitempty"` // 完整研究结果
}

// EvaluationSuite 评测套件
type EvaluationSuite struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Cases       []*EvaluationCase `json:"cases"`
	CreatedAt   time.Time         `json:"created_at"`
}

// EvaluationReport 评测报告
type EvaluationReport struct {
	SuiteName     string                  `json:"suite_name"`
	TotalCases    int                     `json:"total_cases"`
	PassedCases   int                     `json:"passed_cases"`
	FailedCases   int                     `json:"failed_cases"`
	AverageScore  float64                 `json:"average_score"`
	AverageTimeMs int64                   `json:"average_time_ms"`
	Results       []*TestEvaluationResult `json:"results"`
	StartTime     time.Time               `json:"start_time"`
	EndTime       time.Time               `json:"end_time"`
	TotalTimeMs   int64                   `json:"total_time_ms"`
}

// Evaluator 评测器
type Evaluator struct {
	agent *ResearchAgent
}

// NewEvaluator 创建评测器
func NewEvaluator(agent *ResearchAgent) *Evaluator {
	return &Evaluator{agent: agent}
}

// RunCase 运行单个评测用例
func (e *Evaluator) RunCase(ctx context.Context, testCase *EvaluationCase) *TestEvaluationResult {
	result := &TestEvaluationResult{
		CaseID:    testCase.ID,
		Query:     testCase.Query,
		Timestamp: time.Now(),
		Errors:    make([]string, 0),
	}

	// 设置超时
	timeout := time.Duration(testCase.MaxTimeSeconds) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 执行研究
	startTime := time.Now()
	researchResult, err := e.agent.Run(ctx, testCase.Query)
	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("研究执行失败: %v", err))
		result.Passed = false
		return result
	}

	result.Details = researchResult
	result.ConfidenceScore = researchResult.ConfidenceScore

	// 评估要点覆盖
	result.PointsCovered, result.PointsMissing = e.evaluatePoints(
		researchResult.FinalAnswer,
		testCase.ExpectedPoints,
	)

	// 评估来源覆盖
	result.SourcesFound, result.SourcesMissing = e.evaluateSources(
		researchResult,
		testCase.RequiredSources,
	)

	// 计算得分
	result.Score = e.calculateScore(result, testCase)

	// 判断是否通过
	result.Passed = e.checkPassed(result, testCase)

	return result
}

// RunSuite 运行评测套件
func (e *Evaluator) RunSuite(ctx context.Context, suite *EvaluationSuite) *EvaluationReport {
	report := &EvaluationReport{
		SuiteName:  suite.Name,
		TotalCases: len(suite.Cases),
		Results:    make([]*TestEvaluationResult, 0, len(suite.Cases)),
		StartTime:  time.Now(),
	}

	var totalScore float64
	var totalTime int64

	for _, testCase := range suite.Cases {
		select {
		case <-ctx.Done():
			report.EndTime = time.Now()
			report.TotalTimeMs = time.Since(report.StartTime).Milliseconds()
			return report
		default:
		}

		result := e.RunCase(ctx, testCase)
		report.Results = append(report.Results, result)

		if result.Passed {
			report.PassedCases++
		} else {
			report.FailedCases++
		}

		totalScore += result.Score
		totalTime += result.ExecutionTimeMs
	}

	report.EndTime = time.Now()
	report.TotalTimeMs = time.Since(report.StartTime).Milliseconds()

	if len(suite.Cases) > 0 {
		report.AverageScore = totalScore / float64(len(suite.Cases))
		report.AverageTimeMs = totalTime / int64(len(suite.Cases))
	}

	return report
}

// evaluatePoints 评估要点覆盖
func (e *Evaluator) evaluatePoints(answer string, expectedPoints []string) (covered, missing []string) {
	answerLower := strings.ToLower(answer)
	covered = make([]string, 0)
	missing = make([]string, 0)

	for _, point := range expectedPoints {
		pointLower := strings.ToLower(point)
		// 简单的关键词匹配，可以扩展为更复杂的语义匹配
		if strings.Contains(answerLower, pointLower) {
			covered = append(covered, point)
		} else {
			// 尝试分词匹配
			words := strings.Fields(pointLower)
			matchCount := 0
			for _, word := range words {
				if len(word) > 2 && strings.Contains(answerLower, word) {
					matchCount++
				}
			}
			if len(words) > 0 && float64(matchCount)/float64(len(words)) >= 0.5 {
				covered = append(covered, point)
			} else {
				missing = append(missing, point)
			}
		}
	}

	return covered, missing
}

// evaluateSources 评估来源覆盖
func (e *Evaluator) evaluateSources(result *Result, requiredSources []string) (found, missing []string) {
	found = make([]string, 0)
	missing = make([]string, 0)

	// 从步骤中提取来源
	allContent := result.FinalAnswer
	for _, step := range result.Steps {
		allContent += " " + step.Observation
	}
	allContentLower := strings.ToLower(allContent)

	for _, source := range requiredSources {
		sourceLower := strings.ToLower(source)
		if strings.Contains(allContentLower, sourceLower) {
			found = append(found, source)
		} else {
			missing = append(missing, source)
		}
	}

	return found, missing
}

// calculateScore 计算得分
func (e *Evaluator) calculateScore(result *TestEvaluationResult, testCase *EvaluationCase) float64 {
	var score float64

	// 要点覆盖得分 (40%)
	if len(testCase.ExpectedPoints) > 0 {
		pointsScore := float64(len(result.PointsCovered)) / float64(len(testCase.ExpectedPoints)) * 40
		score += pointsScore
	} else {
		score += 40 // 没有要点要求，满分
	}

	// 来源覆盖得分 (30%)
	if len(testCase.RequiredSources) > 0 {
		sourcesScore := float64(len(result.SourcesFound)) / float64(len(testCase.RequiredSources)) * 30
		score += sourcesScore
	} else {
		score += 30 // 没有来源要求，满分
	}

	// 置信度得分 (20%)
	confidenceScore := result.ConfidenceScore * 20
	score += confidenceScore

	// 执行时间得分 (10%)
	if testCase.MaxTimeSeconds > 0 {
		timeRatio := float64(result.ExecutionTimeMs) / float64(testCase.MaxTimeSeconds*1000)
		if timeRatio <= 1 {
			score += 10 * (1 - timeRatio*0.5) // 越快越好，但不超过10分
		}
	} else {
		score += 10
	}

	return score
}

// checkPassed 检查是否通过
func (e *Evaluator) checkPassed(result *TestEvaluationResult, testCase *EvaluationCase) bool {
	// 检查置信度
	if testCase.MinConfidence > 0 && result.ConfidenceScore < testCase.MinConfidence {
		return false
	}

	// 检查执行时间
	if testCase.MaxTimeSeconds > 0 && result.ExecutionTimeMs > int64(testCase.MaxTimeSeconds*1000) {
		return false
	}

	// 检查要点覆盖率（至少50%）
	if len(testCase.ExpectedPoints) > 0 {
		coverageRate := float64(len(result.PointsCovered)) / float64(len(testCase.ExpectedPoints))
		if coverageRate < 0.5 {
			return false
		}
	}

	// 检查错误
	if len(result.Errors) > 0 {
		return false
	}

	return true
}

// CreateDefaultSuite 创建默认评测套件
func CreateDefaultSuite() *EvaluationSuite {
	return &EvaluationSuite{
		Name:        "default",
		Description: "默认研究评测套件",
		CreatedAt:   time.Now(),
		Cases: []*EvaluationCase{
			{
				ID:              "tech_ai_trends",
				Query:           "2024年人工智能领域的主要发展趋势",
				ExpectedPoints:  []string{"大语言模型", "多模态", "AI Agent"},
				RequiredSources: []string{},
				MinConfidence:   0.5,
				MaxTimeSeconds:  180,
				Tags:            []string{"tech", "ai"},
			},
			{
				ID:              "science_climate",
				Query:           "气候变化对全球农业的影响",
				ExpectedPoints:  []string{"温度", "降水", "粮食产量"},
				RequiredSources: []string{},
				MinConfidence:   0.5,
				MaxTimeSeconds:  180,
				Tags:            []string{"science", "climate"},
			},
			{
				ID:              "academic_llm",
				Query:           "大语言模型的最新研究进展",
				ExpectedPoints:  []string{"transformer", "训练", "推理"},
				RequiredSources: []string{"arxiv"},
				MinConfidence:   0.5,
				MaxTimeSeconds:  180,
				Tags:            []string{"academic", "llm"},
			},
		},
	}
}

// ToJSON 将报告转换为JSON
func (r *EvaluationReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Summary 生成报告摘要
func (r *EvaluationReport) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("评测套件: %s\n", r.SuiteName))
	sb.WriteString(fmt.Sprintf("总用例数: %d\n", r.TotalCases))
	sb.WriteString(fmt.Sprintf("通过: %d, 失败: %d\n", r.PassedCases, r.FailedCases))
	sb.WriteString(fmt.Sprintf("通过率: %.1f%%\n", float64(r.PassedCases)/float64(r.TotalCases)*100))
	sb.WriteString(fmt.Sprintf("平均得分: %.1f\n", r.AverageScore))
	sb.WriteString(fmt.Sprintf("平均耗时: %.1f秒\n", float64(r.AverageTimeMs)/1000))
	sb.WriteString(fmt.Sprintf("总耗时: %.1f秒\n", float64(r.TotalTimeMs)/1000))
	return sb.String()
}

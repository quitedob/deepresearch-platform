// Package agent 提供增强版研究规划器
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// EnhancedPlan 增强版研究计划
type EnhancedPlan struct {
	MainQuestion      string        `json:"main_question"`
	SubQuestions      []SubQuestion `json:"sub_questions"`
	RequiredTools     []string      `json:"required_tools"`
	ExpectedDepth     int           `json:"expected_depth"`
	EvidenceThreshold float64       `json:"evidence_threshold"` // 证据门槛
	EstimatedTime     int           `json:"estimated_time_min"` // 预估时间（分钟）
}

// SubQuestion 子问题
type SubQuestion struct {
	Question         string   `json:"question"`
	Priority         int      `json:"priority"`           // 优先级 1-5，1最高
	SearchStrategy   string   `json:"search_strategy"`    // web/arxiv/wiki/mixed
	RequiredEvidence []string `json:"required_evidence"`  // 需要的证据类型
	Completed        bool     `json:"completed"`          // 是否已完成
	EvidenceCount    int      `json:"evidence_count"`     // 已收集的证据数
}

// ResearchPlanner 研究规划器
type ResearchPlanner struct {
	chatModel model.ChatModel
}

// NewResearchPlanner 创建研究规划器
func NewResearchPlanner(chatModel model.ChatModel) *ResearchPlanner {
	return &ResearchPlanner{chatModel: chatModel}
}

// CreatePlan 创建增强版研究计划
func (p *ResearchPlanner) CreatePlan(ctx context.Context, query string) (*EnhancedPlan, error) {
	prompt := fmt.Sprintf(`你是一个研究规划专家。请为以下研究问题制定详细的研究计划。

## 研究问题
%s

## 任务
1. 将问题分解为2-5个具体的子问题
2. 为每个子问题设定优先级（1最高，5最低）
3. 为每个子问题选择最佳搜索策略：
   - web: 适合时事、新闻、最新信息
   - arxiv: 适合学术研究、技术论文
   - wiki: 适合背景知识、概念定义
   - mixed: 需要多种来源
4. 列出每个子问题需要的证据类型

请以JSON格式返回：
{
  "main_question": "原始问题",
  "sub_questions": [
    {
      "question": "子问题1",
      "priority": 1,
      "search_strategy": "web",
      "required_evidence": ["数据统计", "专家观点"]
    }
  ],
  "required_tools": ["web_search_prime", "wikipedia"],
  "expected_depth": 2,
  "evidence_threshold": 0.7,
  "estimated_time_min": 5
}

只返回JSON，不要其他内容。`, query)

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是研究规划专家，擅长将复杂问题分解为可执行的研究步骤。只返回JSON格式的研究计划。"},
		{Role: schema.User, Content: prompt},
	}

	response, err := p.chatModel.Generate(ctx, messages)
	if err != nil {
		return p.defaultPlan(query), nil
	}

	plan := &EnhancedPlan{}
	content := cleanJSONResponse(response.Content)
	if err := json.Unmarshal([]byte(content), plan); err != nil {
		return p.defaultPlan(query), nil
	}

	// 验证和修正计划
	p.validatePlan(plan, query)
	return plan, nil
}

// defaultPlan 默认计划 - 更全面的搜索策略
func (p *ResearchPlanner) defaultPlan(query string) *EnhancedPlan {
	return &EnhancedPlan{
		MainQuestion: query,
		SubQuestions: []SubQuestion{
			{
				Question:         query,
				Priority:         1,
				SearchStrategy:   "web",
				RequiredEvidence: []string{"最新信息"},
			},
			{
				Question:         query + " 背景知识",
				Priority:         2,
				SearchStrategy:   "wiki",
				RequiredEvidence: []string{"基础概念"},
			},
			{
				Question:         query + " 深入分析",
				Priority:         3,
				SearchStrategy:   "web",
				RequiredEvidence: []string{"详细分析"},
			},
		},
		RequiredTools:     []string{"web_search_prime", "wikipedia", "arxiv_search"},
		ExpectedDepth:     3,
		EvidenceThreshold: 0.75,
		EstimatedTime:     5,
	}
}

// validatePlan 验证并修正计划
func (p *ResearchPlanner) validatePlan(plan *EnhancedPlan, query string) {
	if plan.MainQuestion == "" {
		plan.MainQuestion = query
	}
	if len(plan.SubQuestions) == 0 {
		plan.SubQuestions = []SubQuestion{
			{Question: query, Priority: 1, SearchStrategy: "mixed"},
		}
	}
	if len(plan.RequiredTools) == 0 {
		plan.RequiredTools = []string{"web_search_prime", "wikipedia"}
	}
	if plan.ExpectedDepth == 0 {
		plan.ExpectedDepth = 2
	}
	if plan.EvidenceThreshold == 0 {
		plan.EvidenceThreshold = 0.6
	}

	// 验证每个子问题
	for i := range plan.SubQuestions {
		if plan.SubQuestions[i].Priority < 1 || plan.SubQuestions[i].Priority > 5 {
			plan.SubQuestions[i].Priority = 2
		}
		if plan.SubQuestions[i].SearchStrategy == "" {
			plan.SubQuestions[i].SearchStrategy = "mixed"
		}
	}
}

// UpdatePlan 根据反馈更新计划
func (p *ResearchPlanner) UpdatePlan(ctx context.Context, plan *EnhancedPlan, criticResult *CriticResult) *EnhancedPlan {
	// 标记已完成的子问题
	for i := range plan.SubQuestions {
		if plan.SubQuestions[i].EvidenceCount >= 2 {
			plan.SubQuestions[i].Completed = true
		}
	}

	// 根据证据缺口添加新的子问题
	if len(criticResult.EvidenceGaps) > 0 {
		for _, gap := range criticResult.EvidenceGaps {
			// 检查是否已存在类似的子问题
			exists := false
			for _, sq := range plan.SubQuestions {
				if strings.Contains(strings.ToLower(sq.Question), strings.ToLower(gap)) {
					exists = true
					break
				}
			}
			if !exists {
				plan.SubQuestions = append(plan.SubQuestions, SubQuestion{
					Question:         gap,
					Priority:         2,
					SearchStrategy:   "mixed",
					RequiredEvidence: []string{"补充证据"},
				})
			}
		}
	}

	// 根据建议的搜索查询添加子问题
	if len(criticResult.NextSearchQueries) > 0 {
		for _, query := range criticResult.NextSearchQueries {
			plan.SubQuestions = append(plan.SubQuestions, SubQuestion{
				Question:       query,
				Priority:       3,
				SearchStrategy: "web",
			})
		}
	}

	return plan
}

// GetNextSubQuestion 获取下一个要处理的子问题
func (p *ResearchPlanner) GetNextSubQuestion(plan *EnhancedPlan) *SubQuestion {
	var best *SubQuestion
	bestPriority := 999

	for i := range plan.SubQuestions {
		sq := &plan.SubQuestions[i]
		if !sq.Completed && sq.Priority < bestPriority {
			best = sq
			bestPriority = sq.Priority
		}
	}

	return best
}

// GetToolForStrategy 根据搜索策略获取工具名称
// 修复：优先使用 web_search_prime（增强搜索 MCP），提高搜索质量
func (p *ResearchPlanner) GetToolForStrategy(strategy string, depth int) string {
	switch strategy {
	case "web":
		// 优先使用增强搜索
		return "web_search_prime"
	case "arxiv":
		return "arxiv_search"
	case "wiki":
		return "wikipedia"
	case "mixed":
		// 混合策略：根据深度轮换，优先使用增强搜索
		tools := []string{"web_search_prime", "wikipedia", "arxiv_search"}
		return tools[depth%len(tools)]
	default:
		return "web_search_prime"
	}
}

// EstimateCompletion 估算完成度
func (p *ResearchPlanner) EstimateCompletion(plan *EnhancedPlan) float64 {
	if len(plan.SubQuestions) == 0 {
		return 0
	}

	completed := 0
	totalWeight := 0

	for _, sq := range plan.SubQuestions {
		weight := 6 - sq.Priority // 优先级越高权重越大
		totalWeight += weight
		if sq.Completed {
			completed += weight
		} else if sq.EvidenceCount > 0 {
			// 部分完成
			completed += weight * sq.EvidenceCount / 2
		}
	}

	if totalWeight == 0 {
		return 0
	}
	return float64(completed) / float64(totalWeight)
}

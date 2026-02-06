// Package agent 提供 Critic/Reflection 模块
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// CriticResult 反证检查结果
type CriticResult struct {
	Contradictions    []string `json:"contradictions"`      // 发现的矛盾
	EvidenceGaps      []string `json:"evidence_gaps"`       // 证据缺口
	NextSearchQueries []string `json:"next_search_queries"` // 下一轮要搜什么
	ShouldContinue    bool     `json:"should_continue"`     // 是否需要继续
	QualityScore      float64  `json:"quality_score"`       // 当前质量评分 0-1
	Reasoning         string   `json:"reasoning"`           // 推理过程
}

// Critic 反证检查器
type Critic struct {
	chatModel model.ChatModel
}

// NewCritic 创建反证检查器
func NewCritic(chatModel model.ChatModel) *Critic {
	return &Critic{chatModel: chatModel}
}

// Evaluate 评估当前综合结果，检查矛盾和缺口
func (c *Critic) Evaluate(ctx context.Context, query string, currentSynthesis string, evidence []string) (*CriticResult, error) {
	if len(evidence) == 0 {
		return &CriticResult{
			Contradictions:    []string{},
			EvidenceGaps:      []string{"尚未收集任何证据"},
			NextSearchQueries: []string{query},
			ShouldContinue:    true,
			QualityScore:      0,
			Reasoning:         "没有证据，需要开始搜索",
		}, nil
	}

	// 构建证据摘要
	evidenceSummary := make([]string, 0, len(evidence))
	for i, e := range evidence {
		summary := truncateForCritic(e, 400)
		evidenceSummary = append(evidenceSummary, fmt.Sprintf("【证据%d】%s", i+1, summary))
	}

	prompt := fmt.Sprintf(`你是一个严谨的研究质量审查专家。请对以下研究进行反证检查。

## 研究问题
%s

## 当前综合结论
%s

## 已收集的证据
%s

## 审查任务
请严格检查：
1. 证据之间是否存在矛盾？列出所有发现的矛盾点
2. 回答问题还缺少哪些关键证据？
3. 如果需要继续搜索，应该搜索什么？
4. 当前研究质量评分（0-1）

请以JSON格式返回：
{
  "contradictions": ["矛盾点1", "矛盾点2"],
  "evidence_gaps": ["缺失的证据1", "缺失的证据2"],
  "next_search_queries": ["下一轮搜索词1", "下一轮搜索词2"],
  "should_continue": true/false,
  "quality_score": 0.7,
  "reasoning": "推理说明"
}

只返回JSON，不要其他内容。`, query, currentSynthesis, strings.Join(evidenceSummary, "\n\n"))

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是一个严谨的研究质量审查专家，擅长发现证据矛盾和研究缺口。只返回JSON格式的审查结果。"},
		{Role: schema.User, Content: prompt},
	}

	response, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		return c.heuristicCritic(evidence), nil
	}

	result := &CriticResult{}
	content := cleanJSONResponse(response.Content)
	if err := json.Unmarshal([]byte(content), result); err != nil {
		return c.heuristicCritic(evidence), nil
	}

	return result, nil
}

// heuristicCritic 启发式反证检查（当LLM失败时使用）
// 更严格的标准
func (c *Critic) heuristicCritic(evidence []string) *CriticResult {
	count := len(evidence)
	
	// 计算内容丰富度
	totalChars := 0
	for _, e := range evidence {
		totalChars += len(e)
	}
	
	var qualityScore float64
	var shouldContinue bool
	var gaps []string

	switch {
	case count >= 8 && totalChars >= 10000:
		qualityScore = 0.85
		shouldContinue = false
	case count >= 6 && totalChars >= 6000:
		qualityScore = 0.75
		shouldContinue = false
	case count >= 4 && totalChars >= 4000:
		qualityScore = 0.6
		shouldContinue = true
		gaps = []string{"建议补充更多来源"}
	case count >= 3:
		qualityScore = 0.45
		shouldContinue = true
		gaps = []string{"证据不足，需要更多搜索", "建议从不同角度验证"}
	case count >= 2:
		qualityScore = 0.3
		shouldContinue = true
		gaps = []string{"证据严重不足", "需要多轮深入搜索"}
	default:
		qualityScore = 0.15
		shouldContinue = true
		gaps = []string{"几乎没有有效证据", "需要重新规划搜索策略"}
	}

	return &CriticResult{
		Contradictions:    []string{},
		EvidenceGaps:      gaps,
		NextSearchQueries: []string{},
		ShouldContinue:    shouldContinue,
		QualityScore:      qualityScore,
		Reasoning:         fmt.Sprintf("启发式评估：%d条证据，%d字符，质量%.0f%%", count, totalChars, qualityScore*100),
	}
}

// CheckContradictions 专门检查证据之间的矛盾
func (c *Critic) CheckContradictions(ctx context.Context, evidence []string) ([]string, error) {
	if len(evidence) < 2 {
		return []string{}, nil
	}

	evidencePairs := make([]string, 0)
	for i := 0; i < len(evidence)-1 && i < 5; i++ {
		for j := i + 1; j < len(evidence) && j < 6; j++ {
			pair := fmt.Sprintf("证据%d vs 证据%d:\n- %s\n- %s",
				i+1, j+1,
				truncateForCritic(evidence[i], 200),
				truncateForCritic(evidence[j], 200))
			evidencePairs = append(evidencePairs, pair)
		}
	}

	prompt := fmt.Sprintf(`请检查以下证据对之间是否存在矛盾：

%s

如果发现矛盾，请列出。如果没有矛盾，返回空数组。
返回JSON格式：{"contradictions": ["矛盾1", "矛盾2"]}`, strings.Join(evidencePairs, "\n\n"))

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是矛盾检测专家。只返回JSON格式。"},
		{Role: schema.User, Content: prompt},
	}

	response, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		return []string{}, nil
	}

	var result struct {
		Contradictions []string `json:"contradictions"`
	}
	content := cleanJSONResponse(response.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return []string{}, nil
	}

	return result.Contradictions, nil
}

// SuggestNextQueries 建议下一轮搜索查询
func (c *Critic) SuggestNextQueries(ctx context.Context, query string, currentEvidence []string, gaps []string) ([]string, error) {
	if len(gaps) == 0 {
		return []string{}, nil
	}

	prompt := fmt.Sprintf(`基于以下研究缺口，生成2-3个具体的搜索查询：

原始问题：%s
已有证据数量：%d
证据缺口：%s

返回JSON格式：{"queries": ["查询1", "查询2"]}`, query, len(currentEvidence), strings.Join(gaps, "; "))

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是搜索查询优化专家。生成精准的搜索查询。只返回JSON格式。"},
		{Role: schema.User, Content: prompt},
	}

	response, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		// 返回基于缺口的简单查询
		return gaps, nil
	}

	var result struct {
		Queries []string `json:"queries"`
	}
	content := cleanJSONResponse(response.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return gaps, nil
	}

	return result.Queries, nil
}

// truncateForCritic 截取文本用于反证检查
func truncateForCritic(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// cleanJSONResponse 清理JSON响应
func cleanJSONResponse(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}

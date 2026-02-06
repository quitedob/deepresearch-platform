// Package agent 提供结构化研究报告
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// StructuredReport 结构化研究报告
type StructuredReport struct {
	Markdown   string             `json:"markdown"`
	Structured StructuredFindings `json:"structured"`
	Metadata   ReportMetadata     `json:"metadata"`
}

// StructuredFindings 结构化发现
type StructuredFindings struct {
	Conclusions      []Conclusion   `json:"conclusions"`
	Evidence         []EvidenceItem `json:"evidence"`
	Citations        []Citation     `json:"citations"`
	ConfidenceScore  float64        `json:"confidence_score"`
	UnresolvedIssues []string       `json:"unresolved_issues"`
	ReproducibleSteps []string      `json:"reproducible_steps,omitempty"`
	KeyInsights      []string       `json:"key_insights"`
}

// Conclusion 结论
type Conclusion struct {
	ID          string   `json:"id"`
	Statement   string   `json:"statement"`
	Confidence  float64  `json:"confidence"`
	SupportedBy []string `json:"supported_by"` // evidence IDs
	Category    string   `json:"category"`     // main/secondary/tentative
}

// EvidenceItem 证据项
type EvidenceItem struct {
	ID              string  `json:"id"`
	SourceType      string  `json:"source_type"` // web/arxiv/wiki
	SourceTitle     string  `json:"source_title"`
	Content         string  `json:"content"`
	RelevanceScore  float64 `json:"relevance_score"`
	ConfidenceScore float64 `json:"confidence_score"`
	URL             string  `json:"url,omitempty"`
	Timestamp       string  `json:"timestamp"`
}

// Citation 引用
type Citation struct {
	ID         string `json:"id"`
	EvidenceID string `json:"evidence_id"`
	Title      string `json:"title"`
	Source     string `json:"source"`
	URL        string `json:"url,omitempty"`
	AccessDate string `json:"access_date"`
}

// ReportMetadata 报告元数据
type ReportMetadata struct {
	Query           string    `json:"query"`
	GeneratedAt     time.Time `json:"generated_at"`
	ExecutionTimeMs int64     `json:"execution_time_ms"`
	ToolsUsed       []string  `json:"tools_used"`
	TotalSteps      int       `json:"total_steps"`
	SourceCount     int       `json:"source_count"`
	Version         string    `json:"version"`
}

// ReportGenerator 报告生成器
type ReportGenerator struct {
	chatModel model.ChatModel
}

// NewReportGenerator 创建报告生成器
func NewReportGenerator(chatModel model.ChatModel) *ReportGenerator {
	return &ReportGenerator{chatModel: chatModel}
}

// Generate 生成结构化报告
func (g *ReportGenerator) Generate(ctx context.Context, query string, steps []Step, collectedInfo []string, metadata ReportMetadata) (*StructuredReport, error) {
	report := &StructuredReport{
		Metadata: metadata,
		Structured: StructuredFindings{
			Evidence:         make([]EvidenceItem, 0),
			Conclusions:      make([]Conclusion, 0),
			Citations:        make([]Citation, 0),
			UnresolvedIssues: make([]string, 0),
			KeyInsights:      make([]string, 0),
		},
	}

	// 1. 构建证据列表
	for i, info := range collectedInfo {
		evidenceID := fmt.Sprintf("ev_%d", i+1)
		sourceType, sourceTitle := extractSourceInfo(info)
		
		report.Structured.Evidence = append(report.Structured.Evidence, EvidenceItem{
			ID:              evidenceID,
			SourceType:      sourceType,
			SourceTitle:     sourceTitle,
			Content:         truncateContent(info, 1000),
			RelevanceScore:  0.8,
			ConfidenceScore: 0.7,
			Timestamp:       time.Now().Format(time.RFC3339),
		})

		// 添加引用
		report.Structured.Citations = append(report.Structured.Citations, Citation{
			ID:         fmt.Sprintf("cite_%d", i+1),
			EvidenceID: evidenceID,
			Title:      sourceTitle,
			Source:     sourceType,
			AccessDate: time.Now().Format("2006-01-02"),
		})
	}

	// 2. 使用LLM生成结构化结论
	conclusions, insights, unresolved, err := g.extractConclusions(ctx, query, collectedInfo)
	if err == nil {
		report.Structured.Conclusions = conclusions
		report.Structured.KeyInsights = insights
		report.Structured.UnresolvedIssues = unresolved
	}

	// 3. 计算整体置信度
	report.Structured.ConfidenceScore = g.calculateConfidence(report.Structured.Evidence, report.Structured.Conclusions)

	// 4. 生成Markdown报告
	report.Markdown = g.generateMarkdown(ctx, query, report)

	return report, nil
}

// extractConclusions 提取结论
func (g *ReportGenerator) extractConclusions(ctx context.Context, query string, evidence []string) ([]Conclusion, []string, []string, error) {
	if len(evidence) == 0 {
		return nil, nil, []string{"证据不足"}, nil
	}

	// 准备证据摘要
	evidenceSummary := make([]string, 0, len(evidence))
	for i, e := range evidence {
		summary := truncateContent(e, 400)
		evidenceSummary = append(evidenceSummary, fmt.Sprintf("【证据%d】%s", i+1, summary))
	}

	prompt := fmt.Sprintf(`基于以下研究证据，提取结构化的研究结论。

## 研究问题
%s

## 证据
%s

## 任务
请提取：
1. 主要结论（有充分证据支持）
2. 次要结论（有部分证据支持）
3. 关键洞察
4. 未解决的问题

返回JSON格式：
{
  "conclusions": [
    {"statement": "结论1", "confidence": 0.9, "supported_by": ["ev_1", "ev_2"], "category": "main"},
    {"statement": "结论2", "confidence": 0.6, "supported_by": ["ev_3"], "category": "secondary"}
  ],
  "key_insights": ["洞察1", "洞察2"],
  "unresolved_issues": ["未解决问题1"]
}

只返回JSON。`, query, strings.Join(evidenceSummary, "\n\n"))

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是研究分析专家，擅长从证据中提取结构化结论。只返回JSON格式。"},
		{Role: schema.User, Content: prompt},
	}

	response, err := g.chatModel.Generate(ctx, messages)
	if err != nil {
		return g.heuristicConclusions(evidence), []string{}, []string{}, nil
	}

	var result struct {
		Conclusions      []Conclusion `json:"conclusions"`
		KeyInsights      []string     `json:"key_insights"`
		UnresolvedIssues []string     `json:"unresolved_issues"`
	}

	content := cleanJSONResponse(response.Content)
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return g.heuristicConclusions(evidence), []string{}, []string{}, nil
	}

	// 为结论添加ID
	for i := range result.Conclusions {
		result.Conclusions[i].ID = fmt.Sprintf("conc_%d", i+1)
	}

	return result.Conclusions, result.KeyInsights, result.UnresolvedIssues, nil
}

// heuristicConclusions 启发式结论提取
func (g *ReportGenerator) heuristicConclusions(evidence []string) []Conclusion {
	if len(evidence) == 0 {
		return []Conclusion{}
	}

	conclusions := make([]Conclusion, 0)
	supportedBy := make([]string, 0)
	for i := range evidence {
		supportedBy = append(supportedBy, fmt.Sprintf("ev_%d", i+1))
	}

	conclusions = append(conclusions, Conclusion{
		ID:          "conc_1",
		Statement:   fmt.Sprintf("基于%d条证据的综合分析", len(evidence)),
		Confidence:  0.6,
		SupportedBy: supportedBy,
		Category:    "main",
	})

	return conclusions
}

// calculateConfidence 计算整体置信度 - 更严格的标准
func (g *ReportGenerator) calculateConfidence(evidence []EvidenceItem, conclusions []Conclusion) float64 {
	if len(evidence) == 0 {
		return 0
	}

	// 基于证据数量（需要更多证据才能达到高分）
	evidenceScore := minFloat(1.0, float64(len(evidence))/8.0) // 需要8条证据才能满分

	// 基于证据内容丰富度
	totalContent := 0
	for _, e := range evidence {
		totalContent += len(e.Content)
	}
	contentScore := minFloat(1.0, float64(totalContent)/10000.0) // 需要10000字符

	// 基于结论置信度
	var conclusionScore float64
	if len(conclusions) > 0 {
		total := 0.0
		for _, c := range conclusions {
			total += c.Confidence
		}
		conclusionScore = total / float64(len(conclusions))
	}

	// 综合评分 - 更严格的权重
	score := evidenceScore*0.35 + contentScore*0.25 + conclusionScore*0.4
	
	// 如果证据太少，强制降低置信度
	if len(evidence) < 4 {
		score *= 0.7
	}
	
	return score
}

// generateMarkdown 生成Markdown报告
func (g *ReportGenerator) generateMarkdown(ctx context.Context, query string, report *StructuredReport) string {
	var sb strings.Builder

	sb.WriteString("## 研究报告\n\n")

	// 概述
	sb.WriteString("### 概述\n\n")
	sb.WriteString(fmt.Sprintf("针对问题「%s」进行了深度研究，", query))
	sb.WriteString(fmt.Sprintf("共收集 %d 条证据，", len(report.Structured.Evidence)))
	sb.WriteString(fmt.Sprintf("得出 %d 个结论。", len(report.Structured.Conclusions)))
	sb.WriteString(fmt.Sprintf("整体置信度：%.0f%%\n\n", report.Structured.ConfidenceScore*100))

	// 主要结论
	sb.WriteString("### 主要结论\n\n")
	mainCount := 0
	for _, c := range report.Structured.Conclusions {
		if c.Category == "main" || mainCount < 3 {
			sb.WriteString(fmt.Sprintf("- %s（置信度 %.0f%%）\n", c.Statement, c.Confidence*100))
			mainCount++
		}
	}
	if mainCount == 0 {
		sb.WriteString("- 正在分析中...\n")
	}
	sb.WriteString("\n")

	// 关键洞察
	if len(report.Structured.KeyInsights) > 0 {
		sb.WriteString("### 关键洞察\n\n")
		for _, insight := range report.Structured.KeyInsights {
			sb.WriteString(fmt.Sprintf("- %s\n", insight))
		}
		sb.WriteString("\n")
	}

	// 未解决问题
	if len(report.Structured.UnresolvedIssues) > 0 {
		sb.WriteString("### 待进一步研究\n\n")
		for _, issue := range report.Structured.UnresolvedIssues {
			sb.WriteString(fmt.Sprintf("- %s\n", issue))
		}
		sb.WriteString("\n")
	}

	// 研究元数据
	sb.WriteString("### 研究信息\n\n")
	sb.WriteString(fmt.Sprintf("- 使用工具：%s\n", strings.Join(report.Metadata.ToolsUsed, ", ")))
	sb.WriteString(fmt.Sprintf("- 执行时间：%.1f秒\n", float64(report.Metadata.ExecutionTimeMs)/1000))
	sb.WriteString(fmt.Sprintf("- 证据来源：%d个\n", report.Metadata.SourceCount))

	return sb.String()
}

// extractSourceInfo 从内容中提取来源信息
func extractSourceInfo(content string) (sourceType, sourceTitle string) {
	content = strings.TrimSpace(content)
	
	if strings.HasPrefix(content, "网络搜索结果") {
		return "web", "网络搜索"
	}
	if strings.HasPrefix(content, "维基百科") {
		return "wikipedia", "维基百科"
	}
	if strings.HasPrefix(content, "ArXiv") || strings.Contains(content, "arxiv.org") {
		return "arxiv", "学术论文"
	}
	
	return "search", "信息搜索"
}

// truncateContent 截取内容
func truncateContent(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// minFloat 返回较小值
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ToJSON 将报告转换为JSON
func (r *StructuredReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetStructuredJSON 只获取结构化部分的JSON
func (r *StructuredReport) GetStructuredJSON() (string, error) {
	data, err := json.MarshalIndent(r.Structured, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/ai-research-platform/internal/pkg/paper"
)

// PaperAgentConfig 论文Agent配置
type PaperAgentConfig struct {
	MaxReviewRounds     int           `json:"max_review_rounds"`
	Timeout             time.Duration `json:"timeout"`
	WordsTolerance      float64       `json:"words_tolerance"`        // 字数容差，默认0.9
	MinCitations        int           `json:"min_citations"`
	EnableSearchEnhance bool          `json:"enable_search_enhance"`
	CitationStyle       string        `json:"citation_style"`
	DefaultModel        string        `json:"default_model"`
}

// DefaultPaperAgentConfig 默认论文Agent配置
func DefaultPaperAgentConfig() PaperAgentConfig {
	return PaperAgentConfig{
		MaxReviewRounds:     3,
		Timeout:             30 * time.Minute,
		WordsTolerance:      0.9,
		MinCitations:        10,
		EnableSearchEnhance: true,
		CitationStyle:       "chinese-gb",
		DefaultModel:        "glm-4.7",
	}
}

// PaperProgressCallback 论文进度回调
type PaperProgressCallback func(event *PaperProgressEvent)

// PaperProgressEvent 论文进度事件
type PaperProgressEvent struct {
	Stage        string                 `json:"stage"`
	Progress     float32                `json:"progress"`
	Message      string                 `json:"message"`
	ChapterType  string                 `json:"chapter_type,omitempty"`
	ChapterTitle string                 `json:"chapter_title,omitempty"`
	CurrentWords int                    `json:"current_words,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// PaperResult 论文生成结果
type PaperResult struct {
	Success       bool             `json:"success"`
	PaperID       string           `json:"paper_id"`
	Title         string           `json:"title"`
	FullContent   string           `json:"full_content"`
	Chapters      []ChapterResult  `json:"chapters"`
	Citations     []CitationResult `json:"citations"`
	TotalWords    int              `json:"total_words"`
	ReviewRounds  int              `json:"review_rounds"`
	ExecutionTime int64            `json:"execution_time"` // 毫秒
	Error         string           `json:"error,omitempty"`
}

// ChapterResult 章节结果
type ChapterResult struct {
	ChapterType string `json:"chapter_type"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	WordCount   int    `json:"word_count"`
	SortOrder   int    `json:"sort_order"`
}

// CitationResult 引用结果
type CitationResult struct {
	Title        string `json:"title"`
	Authors      string `json:"authors"`
	URL          string `json:"url"`
	DOI          string `json:"doi"`
	Year         int    `json:"year"`
	SourceType   string `json:"source_type"`
	FormattedRef string `json:"formatted_ref"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	URL     string `json:"url"`
	Source  string `json:"source"` // web / arxiv / wikipedia
}

// RevisionAction 修订动作
type RevisionAction struct {
	ChapterType string `json:"chapter_type"`
	ActionType  string `json:"action_type"` // expand / modify / add
	Reason      string `json:"reason"`
	TargetWords int    `json:"target_words"` // 预计目标字数
}

// ReviewResult 审查结果
type ReviewResult struct {
	Passed          bool             `json:"passed"`
	TotalWords      int              `json:"total_words"`
	TargetWords     int              `json:"target_words"`
	WordsShortfall  int              `json:"words_shortfall"`
	QualityScore    float64          `json:"quality_score"`
	Issues          []string         `json:"issues"`
	Suggestions     []string         `json:"suggestions"`
	RevisionActions []RevisionAction `json:"revision_actions"`
}

// PaperAgent 论文生成Agent
type PaperAgent struct {
	chatModel   model.ChatModel
	tools       []tool.InvokableTool
	config      PaperAgentConfig
	templateMgr *paper.TemplateManager
	callbacks   []PaperProgressCallback
}

// NewPaperAgent 创建论文生成Agent
func NewPaperAgent(chatModel model.ChatModel, tools []tool.InvokableTool, config PaperAgentConfig) *PaperAgent {
	if config.MaxReviewRounds == 0 {
		config.MaxReviewRounds = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Minute
	}
	if config.WordsTolerance == 0 {
		config.WordsTolerance = 0.9
	}
	if config.MinCitations == 0 {
		config.MinCitations = 10
	}
	if config.CitationStyle == "" {
		config.CitationStyle = "chinese-gb"
	}

	return &PaperAgent{
		chatModel:   chatModel,
		tools:       tools,
		config:      config,
		templateMgr: paper.NewTemplateManager(),
		callbacks:   make([]PaperProgressCallback, 0),
	}
}

// RegisterCallback 注册进度回调
func (a *PaperAgent) RegisterCallback(callback PaperProgressCallback) {
	a.callbacks = append(a.callbacks, callback)
}

// ClearCallbacks 清除所有回调（防止跨会话累积）
func (a *PaperAgent) ClearCallbacks() {
	a.callbacks = nil
}

// SetCitationStyle 设置引用格式
func (a *PaperAgent) SetCitationStyle(style string) {
	a.config.CitationStyle = style
}

// SetMaxReviewRounds 设置最大审查轮数
func (a *PaperAgent) SetMaxReviewRounds(rounds int) {
	if rounds > 0 {
		a.config.MaxReviewRounds = rounds
	}
}

// emitProgress 发送进度事件
func (a *PaperAgent) emitProgress(event *PaperProgressEvent) {
	event.Timestamp = time.Now()
	for _, cb := range a.callbacks {
		cb(event)
	}
}

// Run 执行论文生成
func (a *PaperAgent) Run(ctx context.Context, title, topic, inputContent, paperType string, targetWords int) (*PaperResult, error) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	result := &PaperResult{
		Title: title,
	}

	// Phase 1: Planning (0% - 5%)
	a.emitProgress(&PaperProgressEvent{
		Stage:    "planning",
		Progress: 0.01,
		Message:  "正在分析主题，规划论文结构...",
	})

	chapters, err := a.templateMgr.GetChaptersForPaper(paperType, targetWords)
	if err != nil {
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}

	outline, err := a.generateOutline(ctx, title, topic, inputContent, paperType, chapters)
	if err != nil {
		return nil, fmt.Errorf("生成大纲失败: %w", err)
	}

	a.emitProgress(&PaperProgressEvent{
		Stage:    "planning",
		Progress: 0.05,
		Message:  "论文大纲规划完成",
		Data:     map[string]interface{}{"outline": outline},
	})

	// Phase 2: Search Enhancement (5% - 15%)
	var searchResults map[string][]SearchResult
	if a.config.EnableSearchEnhance {
		a.emitProgress(&PaperProgressEvent{
			Stage:    "searching",
			Progress: 0.06,
			Message:  "正在搜索相关学术资料...",
		})

		searchResults = a.searchForAllChapters(ctx, title, topic, chapters)

		a.emitProgress(&PaperProgressEvent{
			Stage:    "searching",
			Progress: 0.15,
			Message:  fmt.Sprintf("搜索完成，收集到 %d 类资料", len(searchResults)),
		})
	}

	// Phase 3: Chapter Generation (15% - 65%)
	chapterResults := make([]ChapterResult, 0, len(chapters))
	allCitations := make([]CitationResult, 0)
	totalWords := 0
	chapterProgressBase := float32(0.15)
	chapterProgressStep := float32(0.50) / float32(len(chapters))

	for i, ch := range chapters {
		if ch.Type == "reference" {
			continue // 参考文献最后生成
		}

		progress := chapterProgressBase + chapterProgressStep*float32(i)
		a.emitProgress(&PaperProgressEvent{
			Stage:        "generating",
			Progress:     progress,
			Message:      fmt.Sprintf("正在生成：%s", ch.Title),
			ChapterType:  ch.Type,
			ChapterTitle: ch.Title,
		})

		// 获取该章节对应的搜索结果
		chapterSearchResults := searchResults[ch.Type]

		content, citations, err := a.generateChapter(ctx, title, topic, inputContent, paperType, ch, outline, chapterSearchResults, chapterResults)
		if err != nil {
			// 章节生成失败不中断，记录错误继续
			content = fmt.Sprintf("（%s 生成失败：%v）", ch.Title, err)
		}

		wordCount := paper.CountWords(content)
		totalWords += wordCount

		chapterResult := ChapterResult{
			ChapterType: ch.Type,
			Title:       ch.Title,
			Content:     content,
			WordCount:   wordCount,
			SortOrder:   ch.SortOrder,
		}
		chapterResults = append(chapterResults, chapterResult)
		allCitations = append(allCitations, citations...)

		a.emitProgress(&PaperProgressEvent{
			Stage:        "generating",
			Progress:     progress + chapterProgressStep*0.8,
			Message:      fmt.Sprintf("%s 生成完成（%d字）", ch.Title, wordCount),
			ChapterType:  ch.Type,
			ChapterTitle: ch.Title,
			CurrentWords: totalWords,
		})
	}

	// Phase 4: Review (65% - 85%)
	reviewRounds := 0
	for round := 1; round <= a.config.MaxReviewRounds; round++ {
		a.emitProgress(&PaperProgressEvent{
			Stage:        "reviewing",
			Progress:     0.65 + float32(round-1)*0.06,
			Message:      fmt.Sprintf("第%d轮审查中...", round),
			CurrentWords: totalWords,
		})

		review := a.reviewPaper(ctx, title, topic, paperType, targetWords, chapterResults, allCitations)
		reviewRounds = round

		if review.Passed {
			a.emitProgress(&PaperProgressEvent{
				Stage:    "reviewing",
				Progress: 0.85,
				Message:  fmt.Sprintf("第%d轮审查通过！当前 %d 字", round, totalWords),
			})
			break
		}

		// Phase 5: Revise
		a.emitProgress(&PaperProgressEvent{
			Stage:    "revising",
			Progress: 0.65 + float32(round)*0.06,
			Message:  fmt.Sprintf("第%d轮修订中...（不足 %d 字）", round, review.WordsShortfall),
		})

		chapterResults, allCitations, totalWords = a.reviseChapters(ctx, title, topic, inputContent, paperType, chapters, chapterResults, allCitations, review, searchResults)

		a.emitProgress(&PaperProgressEvent{
			Stage:        "revising",
			Progress:     0.65 + float32(round)*0.06 + 0.04,
			Message:      fmt.Sprintf("第%d轮修订完成，当前 %d 字", round, totalWords),
			CurrentWords: totalWords,
		})
	}

	// Phase 6: Synthesis (85% - 100%)
	a.emitProgress(&PaperProgressEvent{
		Stage:    "synthesis",
		Progress: 0.90,
		Message:  "正在合并生成最终论文...",
	})

	// 生成参考文献章节
	referenceContent := a.generateReferenceSection(allCitations)
	chapterResults = append(chapterResults, ChapterResult{
		ChapterType: "reference",
		Title:       "参考文献",
		Content:     referenceContent,
		WordCount:   paper.CountWords(referenceContent),
		SortOrder:   99,
	})

	// 合并完整论文
	fullContent := a.synthesizeFullPaper(title, chapterResults)

	result.Success = true
	result.FullContent = fullContent
	result.Chapters = chapterResults
	result.Citations = allCitations
	result.TotalWords = totalWords
	result.ReviewRounds = reviewRounds
	result.ExecutionTime = time.Since(startTime).Milliseconds()

	a.emitProgress(&PaperProgressEvent{
		Stage:        "completed",
		Progress:     1.0,
		Message:      fmt.Sprintf("论文生成完成！共 %d 字，%d 条引用，%d 轮审查", totalWords, len(allCitations), reviewRounds),
		CurrentWords: totalWords,
	})

	return result, nil
}

// generateOutline 生成论文大纲
func (a *PaperAgent) generateOutline(ctx context.Context, title, topic, inputContent, paperType string, chapters []paper.ChapterDefinition) (string, error) {
	var chapterList []string
	for _, ch := range chapters {
		if ch.Type == "reference" {
			continue
		}
		chapterList = append(chapterList, fmt.Sprintf("- %s（%s）：目标 %d-%d 字", ch.Title, ch.Type, ch.MinWords, ch.MaxWords))
	}

	prompt := fmt.Sprintf(`你是一位学术论文写作专家。请为以下论文生成详细的写作大纲。

论文标题：%s
论文主题：%s
论文类型：%s
%s

论文章节结构：
%s

请为每个章节生成：
1. 2-3个主要论点或要点
2. 论证思路和逻辑结构
3. 可能涉及的关键概念和理论

输出格式：直接输出大纲文本，使用markdown格式。`,
		title, topic, paperType,
		func() string {
			if inputContent != "" {
				return fmt.Sprintf("用户提供的参考内容：%s", inputContent)
			}
			return ""
		}(),
		strings.Join(chapterList, "\n"))

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是一位资深学术论文写作专家，擅长各类论文的结构规划和内容组织。请用中文回复。"},
		{Role: schema.User, Content: prompt},
	}

	resp, err := a.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// searchForAllChapters 为所有章节搜索相关资料
func (a *PaperAgent) searchForAllChapters(ctx context.Context, title, topic string, chapters []paper.ChapterDefinition) map[string][]SearchResult {
	results := make(map[string][]SearchResult)

	for _, ch := range chapters {
		if ch.Type == "reference" || ch.Type == "keywords" || ch.Type == "abstract" {
			continue
		}

		strategies := paper.GetSearchStrategy(ch.Type)
		var chapterResults []SearchResult

		for _, toolName := range strategies {
			query := fmt.Sprintf("%s %s %s", title, topic, ch.Title)
			searchResult := a.executeSearch(ctx, query, toolName)
			if searchResult != nil {
				chapterResults = append(chapterResults, *searchResult)
			}
		}

		if len(chapterResults) > 0 {
			results[ch.Type] = chapterResults
		}
	}

	return results
}

// executeSearch 使用指定工具执行搜索
func (a *PaperAgent) executeSearch(ctx context.Context, query, preferredTool string) *SearchResult {
	// 精确工具名称映射（不再使用 fuzzy contains 匹配）
	toolAliases := map[string][]string{
		"web_search":   {"web_search", "web_search_prime"},
		"arxiv_search": {"arxiv_search", "arxiv"},
		"wikipedia":    {"wikipedia"},
		"web_reader":   {"web_reader"},
	}

	allowedNames, ok := toolAliases[preferredTool]
	if !ok {
		allowedNames = []string{preferredTool}
	}

	for _, t := range a.tools {
		info, _ := t.Info(ctx)
		if info == nil {
			continue
		}

		toolName := info.Name
		matched := false
		for _, allowed := range allowedNames {
			if toolName == allowed {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		// 根据工具类型构建正确的参数
		escaped := strings.ReplaceAll(query, `"`, `\"`)
		var argsStr string
		switch toolName {
		case "web_search_prime":
			argsStr = fmt.Sprintf(`{"search_query": "%s", "content_size": "medium"}`, escaped)
		case "web_reader":
			if strings.HasPrefix(query, "http://") || strings.HasPrefix(query, "https://") {
				argsStr = fmt.Sprintf(`{"url": "%s"}`, escaped)
			} else {
				continue // web_reader 需要 URL
			}
		case "arxiv_search", "arxiv":
			argsStr = fmt.Sprintf(`{"query": "%s"}`, escaped)
		case "wikipedia":
			argsStr = fmt.Sprintf(`{"query": "%s"}`, escaped)
		default:
			argsStr = fmt.Sprintf(`{"query": "%s"}`, escaped)
		}

		result, err := t.InvokableRun(ctx, argsStr)
		if err != nil || result == "" {
			continue
		}

		// 从工具结果中提取 URL 和标题
		extractedURL := ""
		extractedTitle := query
		urls := extractURLsFromText(result)
		if len(urls) > 0 {
			extractedURL = urls[0]
		}
		// 尝试提取标题（第一行非空文本）
		lines := strings.SplitN(result, "\n", 5)
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && len(line) > 10 && len(line) < 200 {
				extractedTitle = strings.TrimLeft(line, "#- ")
				break
			}
		}

		return &SearchResult{
			Title:   extractedTitle,
			Content: truncatePaperContent(result, 2000),
			URL:     extractedURL,
			Source:  toolName,
		}
	}

	return nil
}

// generateChapter 生成单个章节
func (a *PaperAgent) generateChapter(ctx context.Context, title, topic, inputContent, paperType string, ch paper.ChapterDefinition, outline string, searchResults []SearchResult, previousChapters []ChapterResult) (string, []CitationResult, error) {
	// 构建搜索资料摘要
	var searchSummary string
	if len(searchResults) > 0 {
		var summaries []string
		for _, sr := range searchResults {
			summaries = append(summaries, fmt.Sprintf("【来源：%s】\n%s", sr.Source, truncateContent(sr.Content, 500)))
		}
		searchSummary = "参考资料：\n" + strings.Join(summaries, "\n\n")
	}

	// 构建前文摘要
	var prevSummary string
	if len(previousChapters) > 0 {
		var summaries []string
		for _, prev := range previousChapters {
			content := prev.Content
			if len(content) > 300 {
				content = content[:300] + "..."
			}
			summaries = append(summaries, fmt.Sprintf("【%s】%s", prev.Title, content))
		}
		prevSummary = "已生成内容摘要：\n" + strings.Join(summaries, "\n")
	}

	prompt := fmt.Sprintf(`你是一位学术论文写作专家。请为以下论文撰写【%s】章节。

论文标题：%s
论文主题：%s
论文类型：%s

论文大纲：
%s

当前章节要求：
- 章节名称：%s
- 章节类型：%s
- 目标字数：%d - %d 字（请尽量达到 %d 字以上）
- 说明：%s

%s

%s

%s

撰写要求：
1. 严格按照学术论文规范撰写
2. 语言流畅，逻辑清晰，论证有力
3. 字数必须达到目标要求
4. 适当引用文献（使用[1],[2]等标记）
5. 与前文内容保持连贯，避免重复
6. 使用中文撰写

请直接输出章节内容（不需要输出章节标题），不要有多余的提示语。`,
		ch.Title, title, topic, paperType, outline,
		ch.Title, ch.Type, ch.MinWords, ch.MaxWords, ch.MinWords,
		ch.Description,
		func() string {
			if inputContent != "" {
				return "用户提供的参考内容：\n" + inputContent
			}
			return ""
		}(),
		searchSummary, prevSummary)

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是一位资深学术论文写作专家，擅长撰写高质量的学术论文。请用中文回复，确保内容学术规范、逻辑严谨。每个章节的内容要充实详细，达到目标字数。"},
		{Role: schema.User, Content: prompt},
	}

	resp, err := a.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", nil, err
	}

	content := resp.Content

	// 从搜索结果中提取引用
	citations := extractCitationsFromSearch(searchResults, paper.CitationStyle(a.config.CitationStyle))

	return content, citations, nil
}

// reviewPaper 审查论文
func (a *PaperAgent) reviewPaper(ctx context.Context, title, topic, paperType string, targetWords int, chapters []ChapterResult, citations []CitationResult) *ReviewResult {
	review := &ReviewResult{
		TargetWords: targetWords,
	}

	// 1. 字数检查
	totalWords := 0
	for _, ch := range chapters {
		totalWords += ch.WordCount
	}
	review.TotalWords = totalWords

	threshold := int(float64(targetWords) * a.config.WordsTolerance)
	if totalWords < threshold {
		review.WordsShortfall = threshold - totalWords
		review.Issues = append(review.Issues, fmt.Sprintf("总字数不足：当前%d字，目标%d字（允许最低%d字）", totalWords, targetWords, threshold))

		// 找出字数最少的章节进行扩展
		for _, ch := range chapters {
			if ch.ChapterType == "reference" || ch.ChapterType == "keywords" || ch.ChapterType == "abstract" {
				continue
			}
			if ch.WordCount < 500 { // 对字数过少的章节标记修订
				review.RevisionActions = append(review.RevisionActions, RevisionAction{
					ChapterType: ch.ChapterType,
					ActionType:  "expand",
					Reason:      fmt.Sprintf("%s 字数过少（%d字），需要扩展", ch.Title, ch.WordCount),
					TargetWords: ch.WordCount + (review.WordsShortfall / 3),
				})
			}
		}

		// 如果还没有修订动作，选择可扩展的章节
		if len(review.RevisionActions) == 0 {
			expandableTypes := []string{"analysis", "lit_review", "method", "result", "discussion", "theoretical_framework"}
			for _, chType := range expandableTypes {
				for _, ch := range chapters {
					if ch.ChapterType == chType {
						review.RevisionActions = append(review.RevisionActions, RevisionAction{
							ChapterType: chType,
							ActionType:  "expand",
							Reason:      fmt.Sprintf("为达到目标字数，扩展%s", ch.Title),
							TargetWords: ch.WordCount + (review.WordsShortfall / 2),
						})
						break
					}
				}
				if len(review.RevisionActions) >= 2 {
					break
				}
			}
		}
	}

	// 2. 引用检查
	if len(citations) < a.config.MinCitations {
		review.Issues = append(review.Issues, fmt.Sprintf("引用数量不足：当前%d条，最少需要%d条", len(citations), a.config.MinCitations))
	}

	// 3. 综合判定：字数达标 AND 无严重问题
	hasWordShortfall := totalWords < threshold
	hasCitationShortfall := len(citations) < a.config.MinCitations
	review.Passed = !hasWordShortfall && !hasCitationShortfall

	// 质量分数根据多维度计算，而非固定 0.8
	wordScore := float64(totalWords) / float64(targetWords)
	if wordScore > 1.0 {
		wordScore = 1.0
	}
	citationScore := float64(len(citations)) / float64(a.config.MinCitations)
	if citationScore > 1.0 {
		citationScore = 1.0
	}
	review.QualityScore = wordScore*0.6 + citationScore*0.4

	return review
}

// reviseChapters 修订章节
func (a *PaperAgent) reviseChapters(ctx context.Context, title, topic, inputContent, paperType string, chapters []paper.ChapterDefinition, currentResults []ChapterResult, currentCitations []CitationResult, review *ReviewResult, searchResults map[string][]SearchResult) ([]ChapterResult, []CitationResult, int) {
	updatedResults := make([]ChapterResult, len(currentResults))
	copy(updatedResults, currentResults)
	updatedCitations := make([]CitationResult, len(currentCitations))
	copy(updatedCitations, currentCitations)

	for _, action := range review.RevisionActions {
		for i, ch := range updatedResults {
			if ch.ChapterType != action.ChapterType {
				continue
			}

			// 查找原始章节定义
			var chDef paper.ChapterDefinition
			for _, c := range chapters {
				if c.Type == action.ChapterType {
					chDef = c
					break
				}
			}

			expandedContent, newCitations := a.expandChapter(ctx, title, topic, inputContent, paperType, chDef, ch.Content, action, searchResults[action.ChapterType])
			if expandedContent != "" {
				updatedResults[i].Content = expandedContent
				updatedResults[i].WordCount = paper.CountWords(expandedContent)
				updatedCitations = append(updatedCitations, newCitations...)
			}
			break
		}
	}

	totalWords := 0
	for _, ch := range updatedResults {
		totalWords += ch.WordCount
	}

	return updatedResults, updatedCitations, totalWords
}

// expandChapter 扩展章节内容
func (a *PaperAgent) expandChapter(ctx context.Context, title, topic, inputContent, paperType string, chDef paper.ChapterDefinition, currentContent string, action RevisionAction, searchResults []SearchResult) (string, []CitationResult) {
	var searchSummary string
	if len(searchResults) > 0 {
		var summaries []string
		for _, sr := range searchResults {
			summaries = append(summaries, fmt.Sprintf("【%s】%s", sr.Source, truncateContent(sr.Content, 300)))
		}
		searchSummary = "可参考的资料：\n" + strings.Join(summaries, "\n")
	}

	prompt := fmt.Sprintf(`你是一位学术论文写作专家。当前论文的【%s】章节字数不足，需要扩展补充。

论文标题：%s
论文主题：%s
修订原因：%s
目标字数：至少 %d 字

当前章节内容：
%s

%s

请在保留原有内容的基础上，扩展补充以下方面：
1. 增加更深入的分析和论证
2. 补充更多的例证和数据
3. 加强理论阐述的深度
4. 增加与相关研究的对比讨论
5. 适当添加引用标记[n]

请直接输出扩展后的完整章节内容，不要有标题和多余提示语。`,
		chDef.Title, title, topic, action.Reason, action.TargetWords,
		currentContent, searchSummary)

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是一位资深学术论文写作专家。请在不改变原文核心观点的基础上扩展补充内容，使其更加充实完整。用中文撰写。"},
		{Role: schema.User, Content: prompt},
	}

	resp, err := a.chatModel.Generate(ctx, messages)
	if err != nil {
		return currentContent, nil
	}

	newCitations := extractCitationsFromSearch(searchResults, paper.CitationStyle(a.config.CitationStyle))
	return resp.Content, newCitations
}

// generateReferenceSection 生成参考文献章节
func (a *PaperAgent) generateReferenceSection(citations []CitationResult) string {
	if len(citations) == 0 {
		return "（暂无参考文献）"
	}

	// 去重
	seen := make(map[string]bool)
	var uniqueCitations []CitationResult
	for _, c := range citations {
		key := c.Title + c.URL
		if !seen[key] {
			seen[key] = true
			uniqueCitations = append(uniqueCitations, c)
		}
	}

	var refs []string
	for i, c := range uniqueCitations {
		ref := fmt.Sprintf("[%d] %s", i+1, c.FormattedRef)
		if ref == fmt.Sprintf("[%d] ", i+1) {
			// 如果格式化引用为空，使用标题和URL
			ref = fmt.Sprintf("[%d] %s. %s", i+1, c.Title, c.URL)
		}
		refs = append(refs, ref)
	}

	return strings.Join(refs, "\n")
}

// synthesizeFullPaper 合并完整论文
func (a *PaperAgent) synthesizeFullPaper(title string, chapters []ChapterResult) string {
	var sb strings.Builder

	sb.WriteString("# " + title + "\n\n")

	for _, ch := range chapters {
		if ch.ChapterType == "keywords" {
			sb.WriteString("**关键词：** " + ch.Content + "\n\n")
		} else {
			sb.WriteString("## " + ch.Title + "\n\n")
			sb.WriteString(ch.Content + "\n\n")
		}
	}

	return sb.String()
}

// extractCitationsFromSearch 从搜索结果中提取引用
func extractCitationsFromSearch(searchResults []SearchResult, style paper.CitationStyle) []CitationResult {
	var citations []CitationResult

	for _, sr := range searchResults {
		formatted := paper.FormatCitation(style, sr.Title, "", sr.URL, "", 0)

		citations = append(citations, CitationResult{
			Title:        sr.Title,
			URL:          sr.URL,
			SourceType:   sr.Source,
			FormattedRef: formatted,
		})
	}

	return citations
}

// truncatePaperContent 截断内容
func truncatePaperContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

// RegenerateChapter 重新生成单个章节（用于用户反馈后重生成）
func (a *PaperAgent) RegenerateChapter(ctx context.Context, title, topic, chapterType, chapterTitle, currentContent string, minWords, maxWords int, feedback string) (string, error) {
	var feedbackSection string
	if feedback != "" {
		feedbackSection = "\n用户修改建议：" + feedback
	}

	prompt := fmt.Sprintf(`你是一位学术论文写作专家。请重新撰写论文【%s】的【%s】章节。

论文主题：%s
章节类型：%s
%s

当前内容（需要重写）：
%s

要求：
1. 字数：%d-%d字
2. 保持学术论文规范
3. 逻辑清晰，论证有力
4. 适当引用文献（使用[1],[2]等标记）
5. 使用中文撰写

请直接输出章节内容，不要有标题和多余提示语。`,
		title, chapterTitle, topic, chapterType,
		feedbackSection,
		truncatePaperContent(currentContent, 3000),
		minWords, maxWords)

	messages := []*schema.Message{
		{Role: schema.System, Content: "你是一位资深学术论文写作专家。请根据用户反馈重新撰写章节内容，确保质量提升。用中文回复。"},
		{Role: schema.User, Content: prompt},
	}

	resp, err := a.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("生成失败: %w", err)
	}

	return resp.Content, nil
}

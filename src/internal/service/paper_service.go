package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ai-research-platform/internal/pkg/eino/agent"
	"github.com/ai-research-platform/internal/pkg/paper"
	"github.com/ai-research-platform/internal/repository"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/ai-research-platform/internal/types/response"
	"go.uber.org/zap"
)

// PaperService 论文生成服务
type PaperService struct {
	repo        repository.PaperRepository
	paperAgent  *agent.PaperAgent
	templateMgr *paper.TemplateManager
	eventStream *PaperEventStream
	logger      *zap.Logger
	mu          sync.RWMutex
}

// PaperEventStream 论文事件流管理
type PaperEventStream struct {
	streams    sync.Map // sessionID -> chan *response.PaperProgressEvent
	bufferSize int
}

// NewPaperEventStream 创建论文事件流
func NewPaperEventStream(bufferSize int) *PaperEventStream {
	return &PaperEventStream{
		bufferSize: bufferSize,
	}
}

// Subscribe 订阅事件流
// 如果同一 sessionID 已有旧 channel，先关闭旧的再创建新的，防止泄漏
func (s *PaperEventStream) Subscribe(sessionID string) chan *response.PaperProgressEvent {
	ch := make(chan *response.PaperProgressEvent, s.bufferSize)
	// 关闭并替换旧 channel（防止多次订阅泄漏）
	if old, loaded := s.streams.LoadAndDelete(sessionID); loaded {
		// 用 recover 防止关闭已关闭 channel 的 panic
		func() {
			defer func() { recover() }()
			close(old.(chan *response.PaperProgressEvent))
		}()
	}
	s.streams.Store(sessionID, ch)

	// 35分钟后自动清理
	go func() {
		time.Sleep(35 * time.Minute)
		s.Unsubscribe(sessionID)
	}()

	return ch
}

// Unsubscribe 取消订阅
func (s *PaperEventStream) Unsubscribe(sessionID string) {
	if ch, ok := s.streams.LoadAndDelete(sessionID); ok {
		func() {
			defer func() { recover() }()
			close(ch.(chan *response.PaperProgressEvent))
		}()
	}
}

// Send 发送事件（并发安全，channel 关闭后不 panic）
func (s *PaperEventStream) Send(sessionID string, event *response.PaperProgressEvent) {
	ch, ok := s.streams.Load(sessionID)
	if !ok {
		return
	}
	// 用 recover 防止向已关闭 channel 发送时 panic
	func() {
		defer func() { recover() }()
		select {
		case ch.(chan *response.PaperProgressEvent) <- event:
		default:
			// 缓冲区满，丢弃
		}
	}()
}

// NewPaperService 创建论文服务
func NewPaperService(repo repository.PaperRepository, paperAgent *agent.PaperAgent, logger *zap.Logger) *PaperService {
	return &PaperService{
		repo:        repo,
		paperAgent:  paperAgent,
		templateMgr: paper.NewTemplateManager(),
		eventStream: NewPaperEventStream(200),
		logger:      logger,
	}
}

// StartPaperGeneration 开始论文生成
func (s *PaperService) StartPaperGeneration(ctx context.Context, userID string, title, topic, inputContent string, targetWords int, paperType string, options map[string]interface{}) (*model.PaperSession, error) {
	// 创建会话
	session := &model.PaperSession{
		UserID:       userID,
		Title:        title,
		Topic:        topic,
		InputContent: inputContent,
		TargetWords:  targetWords,
		PaperType:    paperType,
		Status:       "drafting",
		Progress:     0,
	}

	if options != nil {
		metadataJSON, _ := json.Marshal(options)
		session.Metadata = metadataJSON
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("创建论文会话失败: %w", err)
	}

	// 创建章节记录
	chapters, err := s.templateMgr.GetChaptersForPaper(paperType, targetWords)
	if err != nil {
		return nil, fmt.Errorf("获取模板失败: %w", err)
	}

	var chapterModels []*model.PaperChapter
	for _, ch := range chapters {
		chapterModels = append(chapterModels, &model.PaperChapter{
			PaperID:     session.ID,
			ChapterType: ch.Type,
			Title:       ch.Title,
			SortOrder:   ch.SortOrder,
			MinWords:    ch.MinWords,
			MaxWords:    ch.MaxWords,
			Status:      "pending",
		})
	}

	if err := s.repo.CreateChapters(ctx, chapterModels); err != nil {
		return nil, fmt.Errorf("创建章节记录失败: %w", err)
	}

	// 异步执行论文生成，并传递选项
	go s.executePaperGeneration(session.ID, title, topic, inputContent, paperType, targetWords, options)

	return session, nil
}

// executePaperGeneration 执行论文生成（后台运行）
func (s *PaperService) executePaperGeneration(sessionID, title, topic, inputContent, paperType string, targetWords int, options map[string]interface{}) {
	// panic 恢复：防止 goroutine 崩溃导致会话永远停留在 drafting 状态
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("论文生成 goroutine panic",
				zap.String("session_id", sessionID),
				zap.Any("panic", r),
			)
			ctx := context.Background()
			s.repo.UpdateSessionStatus(ctx, sessionID, "failed", 0)
			s.eventStream.Send(sessionID, &response.PaperProgressEvent{
				Type:      "error",
				Message:   fmt.Sprintf("论文生成发生内部错误"),
				Timestamp: time.Now(),
			})
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()

	s.logger.Info("开始论文生成",
		zap.String("session_id", sessionID),
		zap.String("title", title),
		zap.Int("target_words", targetWords),
	)

	// 解析会话级配置（不修改全局 Agent 状态）
	citationStyle := ""
	maxReviewRounds := 0
	if options != nil {
		if style, ok := options["citation_style"].(string); ok {
			citationStyle = style
		}
		if rounds, ok := options["max_review_rounds"].(int); ok {
			maxReviewRounds = rounds
		}
		if rounds, ok := options["max_review_rounds"].(float64); ok {
			maxReviewRounds = int(rounds)
		}
	}
	// 为本次会话设置独立配置（并发安全）
	s.paperAgent.SetSessionConfig(sessionID, citationStyle, maxReviewRounds)

	// 注册会话级别的进度回调（并发安全，不影响其他会话）
	s.paperAgent.RegisterSessionCallback(sessionID, func(event *agent.PaperProgressEvent) {
		s.repo.UpdateSessionStatus(ctx, sessionID, event.Stage, event.Progress)
		if event.CurrentWords > 0 {
			s.repo.UpdateSessionWords(ctx, sessionID, event.CurrentWords)
		}
		s.eventStream.Send(sessionID, &response.PaperProgressEvent{
			Type:         "status_update",
			Stage:        event.Stage,
			Progress:     event.Progress,
			Message:      event.Message,
			ChapterType:  event.ChapterType,
			ChapterTitle: event.ChapterTitle,
			CurrentWords: event.CurrentWords,
			Data:         event.Data,
			Timestamp:    event.Timestamp,
		})
	})
	defer s.paperAgent.ClearSessionCallback(sessionID)

	// 执行论文生成
	result, err := s.paperAgent.RunWithSession(ctx, sessionID, title, topic, inputContent, paperType, targetWords)
	if err != nil {
		s.logger.Error("论文生成失败", zap.String("session_id", sessionID), zap.Error(err))
		s.repo.UpdateSessionStatus(ctx, sessionID, "failed", 0)
		s.eventStream.Send(sessionID, &response.PaperProgressEvent{
			Type:      "error",
			Message:   fmt.Sprintf("论文生成失败: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	// 保存生成结果到数据库
	s.saveResult(ctx, sessionID, result)

	s.logger.Info("论文生成完成",
		zap.String("session_id", sessionID),
		zap.Int("total_words", result.TotalWords),
		zap.Int("citations", len(result.Citations)),
		zap.Int("review_rounds", result.ReviewRounds),
	)
}

// saveResult 保存论文结果
func (s *PaperService) saveResult(ctx context.Context, sessionID string, result *agent.PaperResult) {
	// 更新章节内容
	chapters, err := s.repo.GetChaptersByPaperID(ctx, sessionID)
	if err != nil {
		s.logger.Error("获取章节失败", zap.Error(err))
		return
	}

	chapterMap := make(map[string]*model.PaperChapter)
	for _, ch := range chapters {
		chapterMap[ch.ChapterType] = ch
	}

	for _, chResult := range result.Chapters {
		if dbChapter, ok := chapterMap[chResult.ChapterType]; ok {
			s.repo.UpdateChapterContent(ctx, dbChapter.ID, chResult.Content, chResult.WordCount)
		}
	}

	// 保存引用
	if len(result.Citations) > 0 {
		var citationModels []*model.PaperCitation
		for i, c := range result.Citations {
			citationModels = append(citationModels, &model.PaperCitation{
				PaperID:      sessionID,
				CitationType: "bibliography",
				SourceType:   c.SourceType,
				Title:        c.Title,
				Authors:      c.Authors,
				URL:          c.URL,
				DOI:          c.DOI,
				Year:         c.Year,
				FormattedRef: c.FormattedRef,
				Position:     i + 1,
			})
		}
		s.repo.CreateCitations(ctx, citationModels)
	}

	// 更新会话状态为完成
	session, err := s.repo.GetSession(ctx, sessionID)
	if err == nil {
		session.Status = "completed"
		session.Progress = 1.0
		session.CurrentWords = result.TotalWords
		session.ReviewRound = result.ReviewRounds
		s.repo.UpdateSession(ctx, session)
	}

	// 通知完成
	s.eventStream.Send(sessionID, &response.PaperProgressEvent{
		Type:         "completed",
		Stage:        "completed",
		Progress:     1.0,
		Message:      fmt.Sprintf("论文生成完成！共 %d 字", result.TotalWords),
		CurrentWords: result.TotalWords,
		Timestamp:    time.Now(),
	})
}

// CheckOwnership 检查用户是否拥有该论文会话
func (s *PaperService) CheckOwnership(ctx context.Context, sessionID, userID string) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("会话不存在")
	}
	if session.UserID != userID {
		return fmt.Errorf("无权访问")
	}
	return nil
}

// GetPaperStatus 获取论文生成状态
func (s *PaperService) GetPaperStatus(ctx context.Context, sessionID, userID string) (*response.PaperStatusData, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	// IDOR 校验
	if session.UserID != userID {
		return nil, fmt.Errorf("无权访问该论文")
	}

	chapters, err := s.repo.GetChaptersByPaperID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	chapterResponses := make([]*response.PaperChapterResponse, 0, len(chapters))
	for _, ch := range chapters {
		chapterResponses = append(chapterResponses, response.NewPaperChapterResponse(ch))
	}

	return &response.PaperStatusData{
		SessionID:    session.ID,
		Title:        session.Title,
		Status:       session.Status,
		Progress:     session.Progress,
		CurrentWords: session.CurrentWords,
		TargetWords:  session.TargetWords,
		ReviewRound:  session.ReviewRound,
		Chapters:     chapterResponses,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
	}, nil
}

// StreamProgress 获取进度事件流
func (s *PaperService) StreamProgress(sessionID string) chan *response.PaperProgressEvent {
	return s.eventStream.Subscribe(sessionID)
}

// StopStreamProgress 停止进度事件流
func (s *PaperService) StopStreamProgress(sessionID string) {
	s.eventStream.Unsubscribe(sessionID)
}

// GetPaperResult 获取论文结果
func (s *PaperService) GetPaperResult(ctx context.Context, sessionID, userID string) (*response.PaperResultData, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	// IDOR 校验
	if session.UserID != userID {
		return nil, fmt.Errorf("无权访问该论文")
	}

	if session.Status != "completed" {
		return nil, fmt.Errorf("论文尚未生成完成，当前状态: %s", session.Status)
	}

	chapters, err := s.repo.GetChaptersByPaperID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	citations, err := s.repo.GetCitationsByPaperID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// 构建完整内容
	var fullContent strings.Builder
	fullContent.WriteString("# " + session.Title + "\n\n")

	chapterResponses := make([]*response.PaperChapterFullResponse, 0, len(chapters))
	for _, ch := range chapters {
		chapterResponses = append(chapterResponses, &response.PaperChapterFullResponse{
			ID:          ch.ID,
			ChapterType: ch.ChapterType,
			Title:       ch.Title,
			SortOrder:   ch.SortOrder,
			Content:     ch.Content,
			WordCount:   ch.WordCount,
			Status:      ch.Status,
		})

		if ch.ChapterType == "keywords" {
			fullContent.WriteString("**关键词：** " + ch.Content + "\n\n")
		} else {
			fullContent.WriteString("## " + ch.Title + "\n\n")
			fullContent.WriteString(ch.Content + "\n\n")
		}
	}

	citationResponses := make([]*response.PaperCitationResponse, 0, len(citations))
	for _, c := range citations {
		citationResponses = append(citationResponses, &response.PaperCitationResponse{
			ID:           c.ID,
			CitationType: c.CitationType,
			SourceType:   c.SourceType,
			Title:        c.Title,
			Authors:      c.Authors,
			URL:          c.URL,
			Year:         c.Year,
			FormattedRef: c.FormattedRef,
		})
	}

	return &response.PaperResultData{
		SessionID:     session.ID,
		Title:         session.Title,
		PaperType:     session.PaperType,
		TotalWords:    session.CurrentWords,
		TargetWords:   session.TargetWords,
		CitationCount: len(citations),
		ReviewRounds:  session.ReviewRound,
		FullContent:   fullContent.String(),
		Chapters:      chapterResponses,
		Citations:     citationResponses,
	}, nil
}

// UpdateChapterContent 手动更新章节内容（用户编辑）
func (s *PaperService) UpdateChapterContent(ctx context.Context, sessionID, chapterID, userID, content string) error {
	if err := s.CheckOwnership(ctx, sessionID, userID); err != nil {
		return fmt.Errorf("无权操作该论文")
	}
	chapter, err := s.repo.GetChapterByID(ctx, chapterID)
	if err != nil {
		return fmt.Errorf("章节不存在")
	}
	if chapter.PaperID != sessionID {
		return fmt.Errorf("章节不属于该论文")
	}
	wordCount := paper.CountWords(content)
	return s.repo.UpdateChapterContent(ctx, chapterID, content, wordCount)
}

// ExportPaper 导出论文
func (s *PaperService) ExportPaper(ctx context.Context, sessionID, userID, format string) (string, string, error) {
	// IDOR 校验
	if err := s.CheckOwnership(ctx, sessionID, userID); err != nil {
		return "", "", err
	}

	result, err := s.GetPaperResult(ctx, sessionID, userID)
	if err != nil {
		return "", "", err
	}

	switch format {
	case "markdown", "md":
		return result.FullContent, "text/markdown", nil
	case "latex", "tex":
		tex := s.buildLaTeX(result)
		return tex, "application/x-latex", nil
	case "docx":
		return "", "", fmt.Errorf("DOCX 导出尚未支持，请使用 markdown 或 latex 格式")
	default:
		return result.FullContent, "text/markdown", nil
	}
}

// buildLaTeX 将论文结果转换为 LaTeX 文档
func (s *PaperService) buildLaTeX(result *response.PaperResultData) string {
	var sb strings.Builder

	// 文档头
	sb.WriteString("\\documentclass[12pt,a4paper]{article}\n")
	sb.WriteString("\\usepackage[UTF8]{ctex}\n")
	sb.WriteString("\\usepackage{hyperref}\n")
	sb.WriteString("\\usepackage{geometry}\n")
	sb.WriteString("\\geometry{margin=2.5cm}\n")
	sb.WriteString("\\usepackage{setspace}\n")
	sb.WriteString("\\onehalfspacing\n\n")

	// 标题信息
	sb.WriteString(fmt.Sprintf("\\title{%s}\n", escapeLaTeX(result.Title)))
	sb.WriteString("\\date{\\today}\n\n")
	sb.WriteString("\\begin{document}\n")
	sb.WriteString("\\maketitle\n\n")

	// 章节内容
	for _, ch := range result.Chapters {
		if ch.ChapterType == "keywords" {
			sb.WriteString(fmt.Sprintf("\\noindent\\textbf{关键词：} %s\n\n", escapeLaTeX(ch.Content)))
			continue
		}
		if ch.ChapterType == "abstract" {
			sb.WriteString("\\begin{abstract}\n")
			sb.WriteString(markdownToLaTeX(ch.Content))
			sb.WriteString("\n\\end{abstract}\n\n")
			continue
		}
		if ch.ChapterType == "reference" {
			// 参考文献单独处理
			continue
		}
		sb.WriteString(fmt.Sprintf("\\section{%s}\n", escapeLaTeX(ch.Title)))
		sb.WriteString(markdownToLaTeX(ch.Content))
		sb.WriteString("\n\n")
	}

	// 参考文献：如果有 BibTeX 格式的引用则输出 bibliography，否则用 enumerate
	if len(result.Citations) > 0 {
		// 检查第一条是否是 BibTeX 格式
		if strings.HasPrefix(strings.TrimSpace(result.Citations[0].FormattedRef), "@") {
			// BibTeX 格式：写入 .bib 内容作为注释，并用 thebibliography 环境
			sb.WriteString("\\begin{thebibliography}{99}\n")
			for i, c := range result.Citations {
				sb.WriteString(fmt.Sprintf("\\bibitem{ref%d} %s\n", i+1, escapeLaTeX(c.FormattedRef)))
			}
			sb.WriteString("\\end{thebibliography}\n")
		} else {
			sb.WriteString("\\section*{参考文献}\n")
			sb.WriteString("\\begin{enumerate}\n")
			for _, c := range result.Citations {
				sb.WriteString(fmt.Sprintf("  \\item %s\n", escapeLaTeX(c.FormattedRef)))
			}
			sb.WriteString("\\end{enumerate}\n")
		}
	}

	sb.WriteString("\n\\end{document}\n")
	return sb.String()
}

// escapeLaTeX 转义 LaTeX 特殊字符
func escapeLaTeX(s string) string {
	replacer := strings.NewReplacer(
		"&", "\\&",
		"%", "\\%",
		"$", "\\$",
		"#", "\\#",
		"_", "\\_",
		"{", "\\{",
		"}", "\\}",
		"~", "\\textasciitilde{}",
		"^", "\\textasciicircum{}",
		"\\", "\\textbackslash{}",
	)
	return replacer.Replace(s)
}

// markdownToLaTeX 简单的 Markdown → LaTeX 转换（处理常见格式）
func markdownToLaTeX(md string) string {
	lines := strings.Split(md, "\n")
	var out []string
	for _, line := range lines {
		// 标题
		if strings.HasPrefix(line, "### ") {
			out = append(out, "\\subsubsection{"+escapeLaTeX(line[4:])+"}")
		} else if strings.HasPrefix(line, "## ") {
			out = append(out, "\\subsection{"+escapeLaTeX(line[3:])+"}")
		} else if strings.HasPrefix(line, "# ") {
			out = append(out, "\\section{"+escapeLaTeX(line[2:])+"}")
		} else {
			// 粗体 **text**
			converted := line
			for strings.Contains(converted, "**") {
				start := strings.Index(converted, "**")
				end := strings.Index(converted[start+2:], "**")
				if end < 0 {
					break
				}
				inner := converted[start+2 : start+2+end]
				converted = converted[:start] + "\\textbf{" + escapeLaTeX(inner) + "}" + converted[start+2+end+2:]
			}
			// 斜体 *text*
			for strings.Contains(converted, "*") {
				start := strings.Index(converted, "*")
				end := strings.Index(converted[start+1:], "*")
				if end < 0 {
					break
				}
				inner := converted[start+1 : start+1+end]
				converted = converted[:start] + "\\textit{" + escapeLaTeX(inner) + "}" + converted[start+1+end+1:]
			}
			out = append(out, converted)
		}
	}
	return strings.Join(out, "\n")
}

// ListPapers 获取用户的论文列表
func (s *PaperService) ListPapers(ctx context.Context, userID string, limit, offset int) ([]*response.PaperSessionResponse, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	sessions, err := s.repo.ListSessionsByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountSessionsByUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*response.PaperSessionResponse, 0, len(sessions))
	for _, session := range sessions {
		responses = append(responses, response.NewPaperSessionResponse(session))
	}

	return responses, total, nil
}

// DeletePaper 删除论文
func (s *PaperService) DeletePaper(ctx context.Context, sessionID, userID string) error {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.UserID != userID {
		return fmt.Errorf("无权删除此论文")
	}

	return s.repo.DeleteSession(ctx, sessionID)
}

// RegenerateChapter 重新生成章节
func (s *PaperService) RegenerateChapter(ctx context.Context, sessionID, chapterID, userID, feedback string) error {
	// IDOR 校验
	if err := s.CheckOwnership(ctx, sessionID, userID); err != nil {
		return fmt.Errorf("无权操作该论文")
	}

	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	chapter, err := s.repo.GetChapterByID(ctx, chapterID)
	if err != nil {
		return err
	}

	if chapter.PaperID != sessionID {
		return fmt.Errorf("章节不属于该论文")
	}

	// 标记章节为重新生成中
	s.repo.UpdateChapterStatus(ctx, chapterID, "generating")

	// 异步重新生成：真正调用 LLM
	go func() {
		// panic 恢复
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("章节重新生成 goroutine panic",
					zap.String("chapter_id", chapterID),
					zap.Any("panic", r),
				)
				s.repo.UpdateChapterStatus(context.Background(), chapterID, "failed")
			}
		}()

		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		s.logger.Info("重新生成章节",
			zap.String("session_id", sessionID),
			zap.String("chapter_id", chapterID),
			zap.String("chapter_type", chapter.ChapterType),
		)

		// 调用 PaperAgent 实际重新生成
		newContent, err := s.paperAgent.RegenerateChapter(bgCtx, session.Title, session.Topic, chapter.ChapterType, chapter.Title, chapter.Content, chapter.MinWords, chapter.MaxWords, feedback)
		if err != nil {
			s.logger.Error("重新生成章节失败",
				zap.String("chapter_id", chapterID),
				zap.Error(err),
			)
			s.repo.UpdateChapterStatus(bgCtx, chapterID, "failed")
			s.eventStream.Send(sessionID, &response.PaperProgressEvent{
				Type:         "error",
				Message:      fmt.Sprintf("章节重新生成失败: %v", err),
				ChapterType:  chapter.ChapterType,
				ChapterTitle: chapter.Title,
				Timestamp:    time.Now(),
			})
			return
		}

		// 更新章节内容
		wordCount := len([]rune(newContent))
		s.repo.UpdateChapterContent(bgCtx, chapter.ID, newContent, wordCount)
		s.repo.UpdateChapterStatus(bgCtx, chapterID, "completed")

		s.eventStream.Send(sessionID, &response.PaperProgressEvent{
			Type:         "chapter_regenerated",
			Stage:        "completed",
			Progress:     1.0,
			Message:      fmt.Sprintf("%s 重新生成完成（%d字）", chapter.Title, wordCount),
			ChapterType:  chapter.ChapterType,
			ChapterTitle: chapter.Title,
			CurrentWords: wordCount,
			Timestamp:    time.Now(),
		})
	}()

	return nil
}

// GetTemplates 获取所有论文模板
func (s *PaperService) GetTemplates() []*response.PaperTemplateResponse {
	templates := s.templateMgr.GetAllTemplates()
	responses := make([]*response.PaperTemplateResponse, 0, len(templates))

	for _, tmpl := range templates {
		chapterResponses := make([]*response.PaperTemplateChapterResponse, 0, len(tmpl.Chapters))
		for _, ch := range tmpl.Chapters {
			chapterResponses = append(chapterResponses, &response.PaperTemplateChapterResponse{
				Type:        ch.Type,
				Title:       ch.Title,
				MinWords:    ch.MinWords,
				MaxWords:    ch.MaxWords,
				Description: ch.Description,
			})
		}

		responses = append(responses, &response.PaperTemplateResponse{
			ID:          tmpl.ID,
			Name:        tmpl.Name,
			Type:        string(tmpl.Type),
			Description: tmpl.Description,
			Chapters:    chapterResponses,
		})
	}

	return responses
}

// GetCitationStyles 获取支持的引用格式
func (s *PaperService) GetCitationStyles() []map[string]string {
	styles := []map[string]string{
		{"id": "chinese-gb", "name": "GB/T 7714 国标格式", "description": "中国国家标准引用格式"},
		{"id": "apa", "name": "APA格式", "description": "美国心理学协会引用格式"},
		{"id": "mla", "name": "MLA格式", "description": "现代语言协会引用格式"},
		{"id": "latex", "name": "LaTeX/BibTeX格式", "description": "LaTeX学术论文引用格式"},
	}
	return styles
}

package response

import (
	"time"

	"github.com/ai-research-platform/internal/repository/model"
)

// PaperSessionResponse 论文会话响应
type PaperSessionResponse struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Topic        string    `json:"topic"`
	PaperType    string    `json:"paper_type"`
	Status       string    `json:"status"`
	Progress     float32   `json:"progress"`
	CurrentWords int       `json:"current_words"`
	TargetWords  int       `json:"target_words"`
	ReviewRound  int       `json:"review_round"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PaperStatusResponse 论文生成状态响应
type PaperStatusResponse struct {
	Success bool             `json:"success"`
	Data    *PaperStatusData `json:"data"`
}

// PaperStatusData 论文状态数据
type PaperStatusData struct {
	SessionID    string                  `json:"session_id"`
	Title        string                  `json:"title"`
	Status       string                  `json:"status"`
	Progress     float32                 `json:"progress"`
	CurrentWords int                     `json:"current_words"`
	TargetWords  int                     `json:"target_words"`
	ReviewRound  int                     `json:"review_round"`
	Chapters     []*PaperChapterResponse `json:"chapters,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

// PaperChapterResponse 论文章节响应
type PaperChapterResponse struct {
	ID          string `json:"id"`
	ChapterType string `json:"chapter_type"`
	Title       string `json:"title"`
	SortOrder   int    `json:"sort_order"`
	WordCount   int    `json:"word_count"`
	MinWords    int    `json:"min_words"`
	MaxWords    int    `json:"max_words"`
	Status      string `json:"status"`
}

// PaperResultResponse 论文最终结果响应
type PaperResultResponse struct {
	Success bool            `json:"success"`
	Data    *PaperResultData `json:"data"`
}

// PaperResultData 论文结果数据
type PaperResultData struct {
	SessionID     string                    `json:"session_id"`
	Title         string                    `json:"title"`
	PaperType     string                    `json:"paper_type"`
	TotalWords    int                       `json:"total_words"`
	TargetWords   int                       `json:"target_words"`
	CitationCount int                       `json:"citation_count"`
	ReviewRounds  int                       `json:"review_rounds"`
	FullContent   string                    `json:"full_content"`
	Chapters      []*PaperChapterFullResponse `json:"chapters"`
	Citations     []*PaperCitationResponse  `json:"citations,omitempty"`
}

// PaperChapterFullResponse 完整章节响应（含内容）
type PaperChapterFullResponse struct {
	ID          string `json:"id"`
	ChapterType string `json:"chapter_type"`
	Title       string `json:"title"`
	SortOrder   int    `json:"sort_order"`
	Content     string `json:"content"`
	WordCount   int    `json:"word_count"`
	Status      string `json:"status"`
}

// PaperCitationResponse 引用响应
type PaperCitationResponse struct {
	ID           string `json:"id"`
	CitationType string `json:"citation_type"`
	SourceType   string `json:"source_type"`
	Title        string `json:"title"`
	Authors      string `json:"authors"`
	URL          string `json:"url"`
	Year         int    `json:"year"`
	FormattedRef string `json:"formatted_ref"`
}

// PaperTemplateResponse 论文模板响应
type PaperTemplateResponse struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Type        string                       `json:"type"`
	Description string                       `json:"description"`
	Chapters    []*PaperTemplateChapterResponse `json:"chapters"`
}

// PaperTemplateChapterResponse 模板章节响应
type PaperTemplateChapterResponse struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	MinWords    int    `json:"min_words"`
	MaxWords    int    `json:"max_words"`
	Description string `json:"description"`
}

// PaperProgressEvent SSE进度事件
type PaperProgressEvent struct {
	Type         string                 `json:"type"` // status_update / chapter_started / chapter_completed / review / completed / error
	Stage        string                 `json:"stage,omitempty"`
	Progress     float32                `json:"progress,omitempty"`
	Message      string                 `json:"message"`
	ChapterType  string                 `json:"chapter_type,omitempty"`
	ChapterTitle string                 `json:"chapter_title,omitempty"`
	CurrentWords int                    `json:"current_words,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// NewPaperSessionResponse 从模型转换
func NewPaperSessionResponse(session *model.PaperSession) *PaperSessionResponse {
	return &PaperSessionResponse{
		ID:           session.ID,
		Title:        session.Title,
		Topic:        session.Topic,
		PaperType:    session.PaperType,
		Status:       session.Status,
		Progress:     session.Progress,
		CurrentWords: session.CurrentWords,
		TargetWords:  session.TargetWords,
		ReviewRound:  session.ReviewRound,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
	}
}

// NewPaperChapterResponse 从模型转换
func NewPaperChapterResponse(chapter *model.PaperChapter) *PaperChapterResponse {
	return &PaperChapterResponse{
		ID:          chapter.ID,
		ChapterType: chapter.ChapterType,
		Title:       chapter.Title,
		SortOrder:   chapter.SortOrder,
		WordCount:   chapter.WordCount,
		MinWords:    chapter.MinWords,
		MaxWords:    chapter.MaxWords,
		Status:      chapter.Status,
	}
}

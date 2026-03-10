package request

// StartPaperRequest 开始论文生成请求
type StartPaperRequest struct {
	Title        string        `json:"title" binding:"required,min=2,max=200"`
	Topic        string        `json:"topic" binding:"required,min=10,max=5000"`
	InputContent string        `json:"input_content"`
	TargetWords  int           `json:"target_words" binding:"required,min=1000,max=100000"`
	PaperType    string        `json:"paper_type" binding:"required,oneof=liberal_arts science"`
	Options      *PaperOptions `json:"options"`
}

// PaperOptions 论文生成选项
type PaperOptions struct {
	CitationStyle   string `json:"citation_style"`    // chinese-gb/apa/mla/latex，默认chinese-gb
	EnableSearch    *bool  `json:"enable_search"`     // 默认true
	MaxReviewRounds int    `json:"max_review_rounds"` // 默认3
	Language        string `json:"language"`          // 默认zh-CN
	Model           string `json:"model"`             // 默认glm-4.7
}

// RegeneratePaperRequest 重新生成章节请求
type RegeneratePaperRequest struct {
	PaperID   string `json:"paper_id" binding:"required"`
	ChapterID string `json:"chapter_id" binding:"required"`
	Feedback  string `json:"feedback"` // 用户反馈建议
}

// GetPaperListRequest 获取论文列表请求
type GetPaperListRequest struct {
	Limit  int    `json:"limit" form:"limit"`
	Offset int    `json:"offset" form:"offset"`
	Status string `json:"status" form:"status"`
}

// ExportPaperRequest 导出论文请求
type ExportPaperRequest struct {
	Format string `json:"format" form:"format"` // markdown / docx
}

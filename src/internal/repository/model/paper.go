package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PaperSession 论文生成会话
type PaperSession struct {
	ID           string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID       string         `gorm:"index;not null;type:uuid" json:"user_id"`
	Title        string         `gorm:"type:text;not null" json:"title"`
	Topic        string         `gorm:"type:text;not null" json:"topic"`
	InputContent string         `gorm:"type:text" json:"input_content"`
	TargetWords  int            `gorm:"not null" json:"target_words"`
	PaperType    string         `gorm:"not null" json:"paper_type"` // liberal_arts / science
	Status       string         `gorm:"not null;default:'drafting'" json:"status"` // drafting / reviewing / completed / failed
	Progress     float32        `gorm:"default:0" json:"progress"`
	CurrentWords int            `gorm:"default:0" json:"current_words"`
	ReviewRound  int            `gorm:"default:0" json:"review_round"`
	Metadata     datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定论文会话表名
func (PaperSession) TableName() string {
	return "paper_sessions"
}

// PaperChapter 论文章节
type PaperChapter struct {
	ID          string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	PaperID     string         `gorm:"index;not null;type:uuid" json:"paper_id"`
	ChapterType string         `gorm:"not null" json:"chapter_type"` // abstract/keywords/intro/lit_review/method/result/discussion/conclusion/reference/theoretical_framework/analysis
	Title       string         `gorm:"type:text;not null" json:"title"`
	SortOrder   int            `gorm:"not null;default:0" json:"sort_order"`
	Content     string         `gorm:"type:text" json:"content"`
	WordCount   int            `gorm:"default:0" json:"word_count"`
	Status      string         `gorm:"not null;default:'pending'" json:"status"` // pending / generating / completed / reviewing
	MinWords    int            `gorm:"default:0" json:"min_words"`
	MaxWords    int            `gorm:"default:0" json:"max_words"`
	Metadata    datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// TableName 指定论文章节表名
func (PaperChapter) TableName() string {
	return "paper_chapters"
}

// PaperCitation 论文引用
type PaperCitation struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	PaperID      string    `gorm:"index;not null;type:uuid" json:"paper_id"`
	ChapterID    string    `gorm:"index;type:uuid" json:"chapter_id"`
	CitationType string    `gorm:"not null" json:"citation_type"` // inline / footnote / bibliography
	SourceType   string    `gorm:"not null" json:"source_type"`   // web / arxiv / wikipedia / journal
	Title        string    `gorm:"type:text" json:"title"`
	Authors      string    `gorm:"type:text" json:"authors"`
	URL          string    `gorm:"type:text" json:"url"`
	DOI          string    `gorm:"type:text" json:"doi"`
	Year         int       `gorm:"default:0" json:"year"`
	FormattedRef string    `gorm:"type:text" json:"formatted_ref"` // 格式化引用(APA/MLA/Chinese-GB)
	Position     int       `gorm:"default:0" json:"position"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定论文引用表名
func (PaperCitation) TableName() string {
	return "paper_citations"
}

// PaperReview 论文审查记录
type PaperReview struct {
	ID             string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	PaperID        string         `gorm:"index;not null;type:uuid" json:"paper_id"`
	ReviewRound    int            `gorm:"not null" json:"review_round"`
	ReviewType     string         `gorm:"not null" json:"review_type"` // word_count / quality / structure / citation
	IsPassed       bool           `gorm:"default:false" json:"is_passed"`
	Issues         datatypes.JSON `gorm:"type:jsonb" json:"issues"`
	Suggestions    datatypes.JSON `gorm:"type:jsonb" json:"suggestions"`
	TargetChapters datatypes.JSON `gorm:"type:jsonb" json:"target_chapters"` // 需要修改的章节ID列表
	Metadata       datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
}

// TableName 指定论文审查表名
func (PaperReview) TableName() string {
	return "paper_reviews"
}

// PaperSearchRecord 搜索记录
type PaperSearchRecord struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	PaperID     string    `gorm:"index;not null;type:uuid" json:"paper_id"`
	ChapterID   string    `gorm:"index;type:uuid" json:"chapter_id"`
	Query       string    `gorm:"type:text;not null" json:"query"`
	ToolName    string    `gorm:"not null" json:"tool_name"`
	ResultCount int       `gorm:"default:0" json:"result_count"`
	Results     string    `gorm:"type:text" json:"results"` // 搜索结果JSON
	UsedForGen  bool      `gorm:"default:false" json:"used_for_gen"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName 指定搜索记录表名
func (PaperSearchRecord) TableName() string {
	return "paper_search_records"
}

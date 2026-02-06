package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AIQuestionSession AI出题会话模型
type AIQuestionSession struct {
	ID            string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID        string         `gorm:"index;not null;type:uuid" json:"user_id"`
	Title         string         `json:"title"`
	Provider      string         `gorm:"not null" json:"provider"`
	Model         string         `gorm:"not null" json:"model"`
	MessageCount  int            `gorm:"default:0" json:"message_count"`
	QuestionCount int            `gorm:"default:0" json:"question_count"`
	Metadata      datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定AI出题会话表名
func (AIQuestionSession) TableName() string {
	return "ai_question_sessions"
}

// AIQuestionMessage AI出题消息模型
type AIQuestionMessage struct {
	ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	SessionID string         `gorm:"index;not null;type:uuid" json:"session_id"`
	Role      string         `gorm:"not null" json:"role"` // user, assistant
	Content   string         `gorm:"type:text;not null" json:"content"`
	Metadata  datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定AI出题消息表名
func (AIQuestionMessage) TableName() string {
	return "ai_question_messages"
}

// AIGeneratedQuestion AI生成的题目模型
type AIGeneratedQuestion struct {
	ID              string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	SessionID       string         `gorm:"index;not null;type:uuid" json:"session_id"`
	UserID          string         `gorm:"index;not null;type:uuid" json:"user_id"`
	Type            string         `gorm:"not null" json:"type"` // single, multiple, judge, essay
	QuestionText    string         `gorm:"type:text;not null" json:"question_text"`
	Subject         string         `json:"subject"`
	Difficulty      string         `json:"difficulty"` // easy, medium, hard
	Score           int            `gorm:"default:0" json:"score"`
	Tags            datatypes.JSON `gorm:"type:jsonb" json:"tags"`
	KnowledgePoints datatypes.JSON `gorm:"type:jsonb" json:"knowledge_points"`
	Options         datatypes.JSON `gorm:"type:jsonb" json:"options"`
	CorrectAnswer   datatypes.JSON `gorm:"type:jsonb" json:"correct_answer"`
	Explanation     string         `gorm:"type:text" json:"explanation"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定AI生成题目表名
func (AIGeneratedQuestion) TableName() string {
	return "ai_generated_questions"
}

// AIQuestionConfig AI出题配置模型
type AIQuestionConfig struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	DefaultProvider string    `gorm:"not null" json:"default_provider"`
	DefaultModel    string    `gorm:"not null" json:"default_model"`
	UpdatedBy       string    `gorm:"type:uuid" json:"updated_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName 指定AI出题配置表名
func (AIQuestionConfig) TableName() string {
	return "ai_question_configs"
}

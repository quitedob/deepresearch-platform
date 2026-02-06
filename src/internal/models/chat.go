package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ChatSession 聊天会话，表示用户与AI助手之间的对话上下文
type ChatSession struct {
	ID           string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID       string         `gorm:"index;not null;type:uuid" json:"user_id"`
	Title        string         `json:"title"`
	Provider     string         `gorm:"not null" json:"provider"` // deepseek, zhipu, ollama
	Model        string         `gorm:"not null" json:"model"`
	ModelType    string         `gorm:"default:'default'" json:"model_type"` // default, deep, research
	SystemPrompt string         `gorm:"type:text" json:"system_prompt"`
	MessageCount int            `gorm:"default:0" json:"message_count"`
	Metadata     datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定ChatSession模型的表名
func (ChatSession) TableName() string {
	return "chat_sessions"
}

// Message 聊天会话中的单条消息
type Message struct {
	ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	SessionID  string         `gorm:"index;not null;type:uuid" json:"session_id"`
	Role       string         `gorm:"not null" json:"role"` // user, assistant, system
	Content    string         `gorm:"type:text;not null" json:"content"`
	TokensUsed int            `gorm:"default:0" json:"tokens_used"`
	Metadata   datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
}

// TableName 指定Message模型的表名
func (Message) TableName() string {
	return "messages"
}

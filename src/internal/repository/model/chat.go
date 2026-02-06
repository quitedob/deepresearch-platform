package model

import (
    "time"
    "gorm.io/datatypes"
    "gorm.io/gorm"
)

// ChatSession 聊天会话模型
// 修复：添加版本字段用于乐观锁
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
    Version      int            `gorm:"default:1" json:"version"` // 乐观锁版本号
    CreatedAt    time.Time      `json:"created_at"`
    UpdatedAt    time.Time      `json:"updated_at"`
    DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定聊天会话表名
func (ChatSession) TableName() string {
    return "chat_sessions"
}

// Message 消息模型
// 修复：添加软删除支持和版本字段用于乐观锁
type Message struct {
    ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    SessionID  string         `gorm:"index;not null;type:uuid" json:"session_id"`
    Role       string         `gorm:"not null" json:"role"` // user, assistant, system
    Content    string         `gorm:"type:text;not null" json:"content"`
    TokensUsed int            `gorm:"default:0" json:"tokens_used"`
    Metadata   datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
    Version    int            `gorm:"default:1" json:"version"` // 乐观锁版本号
    CreatedAt  time.Time      `json:"created_at"`
    UpdatedAt  time.Time      `json:"updated_at"`
    DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// TableName 指定消息表名
func (Message) TableName() string {
    return "messages"
}

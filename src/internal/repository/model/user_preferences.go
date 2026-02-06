package model

import (
	"time"

	"gorm.io/gorm"
)

// UserPreferences 用户偏好设置模型
type UserPreferences struct {
	ID                  string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID              string         `gorm:"uniqueIndex;not null;type:uuid" json:"user_id"`
	Theme               string         `gorm:"default:'light'" json:"theme"`
	Language            string         `gorm:"default:'zh'" json:"language"`
	DefaultLLMProvider  string         `gorm:"default:'deepseek'" json:"default_llm_provider"`
	DefaultModel        string         `gorm:"default:'deepseek-chat'" json:"default_model"`
	StreamEnabled       bool           `gorm:"default:true" json:"stream_enabled"`
	NotificationEnabled bool           `gorm:"default:true" json:"notification_enabled"`
	AutoSaveEnabled     bool           `gorm:"default:true" json:"auto_save_enabled"`
	TimeZone            string         `gorm:"default:'Asia/Shanghai'" json:"timezone"`
	// 聊天记忆功能
	MemoryEnabled       bool           `gorm:"default:true" json:"memory_enabled"`
	CustomSystemPrompt  string         `gorm:"type:text" json:"custom_system_prompt"`
	MaxContextTokens    int            `gorm:"default:128000" json:"max_context_tokens"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (UserPreferences) TableName() string {
	return "user_preferences"
}

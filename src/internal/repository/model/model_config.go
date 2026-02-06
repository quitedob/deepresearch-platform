package model

import (
	"time"

	"gorm.io/gorm"
)

// ModelConfig 模型配置（管理员控制哪些模型对用户可见）
type ModelConfig struct {
	ID          string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Provider    string         `gorm:"not null" json:"provider"` // ollama, deepseek, zhipu
	ModelName   string         `gorm:"not null" json:"model_name"`
	DisplayName string         `json:"display_name"`
	IsEnabled   bool           `gorm:"default:true" json:"is_enabled"`
	SortOrder   int            `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ModelConfig) TableName() string {
	return "model_configs"
}

// ProviderConfig 提供商配置
type ProviderConfig struct {
	ID          string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Provider    string         `gorm:"uniqueIndex;not null" json:"provider"` // ollama, deepseek, zhipu
	DisplayName string         `json:"display_name"`
	IsEnabled   bool           `gorm:"default:true" json:"is_enabled"`
	SortOrder   int            `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProviderConfig) TableName() string {
	return "provider_configs"
}

package model

import (
	"time"

	"gorm.io/gorm"
)

// QuotaConfig 全局配额配置（按会员层级）
type QuotaConfig struct {
	ID             string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	MembershipType string         `gorm:"uniqueIndex;not null" json:"membership_type"` // free, premium
	
	// 聊天配额（普通模型+深度思考合计）
	ChatLimit      int            `gorm:"default:10" json:"chat_limit"`
	
	// 深度研究配额
	ResearchLimit  int            `gorm:"default:1" json:"research_limit"`
	
	// 高级会员专用：配额重置周期（小时）
	ResetPeriodHours int          `gorm:"default:5" json:"reset_period_hours"`
	
	Description    string         `json:"description"`
	UpdatedBy      *string        `gorm:"type:uuid" json:"updated_by"` // nullable UUID
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (QuotaConfig) TableName() string {
	return "quota_configs"
}

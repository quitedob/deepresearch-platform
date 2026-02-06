package model

import (
	"time"

	"gorm.io/gorm"
)

// MembershipType 会员类型
type MembershipType string

const (
	MembershipFree    MembershipType = "free"    // 普通用户
	MembershipPremium MembershipType = "premium" // 高级会员
)

// UserMembership 用户会员信息
type UserMembership struct {
	ID                    string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID                string         `gorm:"uniqueIndex;not null;type:uuid" json:"user_id"`
	MembershipType        MembershipType `gorm:"default:'free'" json:"membership_type"`
	
	// 普通聊天次数限制（普通模型+深度思考模型合计）
	NormalChatLimit       int            `gorm:"default:10" json:"normal_chat_limit"`
	NormalChatUsed        int            `gorm:"default:0" json:"normal_chat_used"`
	
	// 深度研究次数限制
	ResearchLimit         int            `gorm:"default:1" json:"research_limit"`
	ResearchUsed          int            `gorm:"default:0" json:"research_used"`
	
	// 高级会员：5小时内的限制
	PremiumChatLimit      int            `gorm:"default:50" json:"premium_chat_limit"`
	PremiumChatUsed       int            `gorm:"default:0" json:"premium_chat_used"`
	PremiumResearchLimit  int            `gorm:"default:10" json:"premium_research_limit"`
	PremiumResearchUsed   int            `gorm:"default:0" json:"premium_research_used"`
	PremiumResetAt        *time.Time     `json:"premium_reset_at"`
	
	// 会员有效期
	ExpiresAt             *time.Time     `json:"expires_at"`
	ActivatedAt           *time.Time     `json:"activated_at"`
	ActivationMethod      string         `gorm:"default:''" json:"activation_method"` // payment, activation_code
	ActivationCodeID      *string        `gorm:"type:uuid" json:"activation_code_id"`
	
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

func (UserMembership) TableName() string {
	return "user_memberships"
}

// ActivationCode 激活码
type ActivationCode struct {
	ID              string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Code            string         `gorm:"uniqueIndex;not null" json:"code"`
	MaxActivations  int            `gorm:"default:1" json:"max_activations"`
	UsedActivations int            `gorm:"default:0" json:"used_activations"`
	ValidDays       int            `gorm:"default:30" json:"valid_days"` // 激活后会员有效天数
	CreatedBy       string         `gorm:"not null;type:uuid" json:"created_by"`
	IsActive        bool           `gorm:"default:true" json:"is_active"`
	ExpiresAt       *time.Time     `json:"expires_at"` // 激活码本身的过期时间
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ActivationCode) TableName() string {
	return "activation_codes"
}

// ActivationRecord 激活记录
type ActivationRecord struct {
	ID               string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ActivationCodeID string    `gorm:"index;not null;type:uuid" json:"activation_code_id"`
	UserID           string    `gorm:"index;not null;type:uuid" json:"user_id"`
	ActivatedAt      time.Time `json:"activated_at"`
}

func (ActivationRecord) TableName() string {
	return "activation_records"
}

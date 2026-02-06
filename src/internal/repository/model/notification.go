package model

import (
	"time"

	"gorm.io/gorm"
)

// NotificationType 通知类型
type NotificationType string

const (
	NotificationSystem  NotificationType = "system"  // 系统通知
	NotificationAnnounce NotificationType = "announce" // 公告
	NotificationAlert   NotificationType = "alert"   // 警告
)

// Notification 通知
type Notification struct {
	ID        string           `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Title     string           `gorm:"not null" json:"title"`
	Content   string           `gorm:"type:text;not null" json:"content"`
	Type      NotificationType `gorm:"default:'system'" json:"type"`
	CreatedBy string           `gorm:"not null;type:uuid" json:"created_by"`
	IsGlobal  bool             `gorm:"default:true" json:"is_global"` // 是否全局通知
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	DeletedAt gorm.DeletedAt   `gorm:"index" json:"-"`
}

func (Notification) TableName() string {
	return "notifications"
}

// UserNotification 用户通知状态
type UserNotification struct {
	ID             string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID         string    `gorm:"index;not null;type:uuid" json:"user_id"`
	NotificationID string    `gorm:"index;not null;type:uuid" json:"notification_id"`
	IsRead         bool      `gorm:"default:false" json:"is_read"`
	ReadAt         *time.Time `json:"read_at"`
	CreatedAt      time.Time `json:"created_at"`
}

func (UserNotification) TableName() string {
	return "user_notifications"
}

package model

import (
    "time"
    "gorm.io/gorm"
)

// User 用户模型
type User struct {
    ID        string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    Email     string         `gorm:"uniqueIndex;not null" json:"email"`
    Username  string         `gorm:"uniqueIndex;not null" json:"username"`
    Password  string         `gorm:"not null" json:"-"` // bcrypt hashed, never expose in JSON
    FullName  *string        `json:"full_name,omitempty"`
    Phone     *string        `json:"phone,omitempty"`
    Role      string         `gorm:"default:'user'" json:"role"`
    Status    string         `gorm:"default:'active'" json:"status"`
    IsAdmin   bool           `gorm:"default:false" json:"is_admin"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定用户表名
func (User) TableName() string {
    return "users"
}

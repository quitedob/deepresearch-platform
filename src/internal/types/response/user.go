package response

import (
	"time"
)

// UserResponse 用户响应
type UserResponse struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	FullName    *string    `json:"full_name,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	Avatar      *string    `json:"avatar,omitempty"`
	Bio         *string    `json:"bio,omitempty"`
	Role        string     `json:"role"`
	Status      string     `json:"status"` // active, inactive, suspended
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	IsAdmin     bool       `json:"is_admin"`
}

// TokenResponse 令牌响应
type TokenResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int64         `json:"expires_in"` // 过期时间（秒）
	User         *UserResponse `json:"user"`
}

// UserPreferencesResponse 用户偏好设置响应
type UserPreferencesResponse struct {
	Theme               string    `json:"theme"`
	Language            string    `json:"language"`
	DefaultLLMProvider  string    `json:"default_llm_provider"`
	DefaultModel        string    `json:"default_model"`
	StreamEnabled       bool      `json:"stream_enabled"`
	NotificationEnabled bool      `json:"notification_enabled"`
	AutoSaveEnabled     bool      `json:"auto_save_enabled"`
	TimeZone            string    `json:"timezone"`
	// 聊天记忆功能
	MemoryEnabled       bool      `json:"memory_enabled"`        // 是否启用聊天记忆
	CustomSystemPrompt  string    `json:"custom_system_prompt"`  // 用户自定义系统提示词
	MaxContextTokens    int       `json:"max_context_tokens"`    // 最大上下文 token 数
	UpdatedAt           time.Time `json:"updated_at"`
}

// UserStatsResponse 用户统计响应
type UserStatsResponse struct {
	Success    bool            `json:"success"`
	Statistics *UserStatistics `json:"statistics"`
	Message    string          `json:"message"`
}

// UserStatistics 用户统计信息
type UserStatistics struct {
	UserID                string    `json:"user_id"`
	TotalChatSessions     int64     `json:"total_chat_sessions"`
	TotalResearchSessions int64     `json:"total_research_sessions"`
	TotalMessages         int64     `json:"total_messages"`
	TotalTokensUsed       int64     `json:"total_tokens_used"`
	LastActivity          time.Time `json:"last_activity"`
	AccountAge            int64     `json:"account_age"` // 天数
	MostUsedProvider      string    `json:"most_used_provider"`
	MostUsedModel         string    `json:"most_used_model"`
	PreferredResearchType string    `json:"preferred_research_type"`
}

// UserActivityResponse 用户活动响应
type UserActivityResponse struct {
	Success  bool           `json:"success"`
	Activity []ActivityItem `json:"activity"`
	Total    int            `json:"total"`
	Message  string         `json:"message"`
}

// ActivityItem 活动项
type ActivityItem struct {
	ID         int64                  `json:"id"`
	UserID     string                 `json:"user_id"`
	Type       string                 `json:"type"`   // chat, research, login, etc.
	Action     string                 `json:"action"` // create, update, delete, etc.
	ResourceID string                 `json:"resource_id,omitempty"`
	Resource   string                 `json:"resource,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// UserProfileResponse 用户档案响应
type UserProfileResponse struct {
	User           *UserResponse            `json:"user"`
	Preferences    *UserPreferencesResponse `json:"preferences,omitempty"`
	Statistics     *UserStatistics          `json:"statistics,omitempty"`
	RecentActivity []ActivityItem           `json:"recent_activity,omitempty"`
}

// PasswordResetResponse 密码重置响应
type PasswordResetResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Email   string `json:"email,omitempty"`
}

// AccountDeletionResponse 账户删除响应
type AccountDeletionResponse struct {
	Success     bool       `json:"success"`
	Message     string     `json:"message"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}

// UsersListResponse 用户列表响应（管理员用）
type UsersListResponse struct {
	Success bool            `json:"success"`
	Users   []*UserResponse `json:"users"`
	Total   int             `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
	Message string          `json:"message"`
}

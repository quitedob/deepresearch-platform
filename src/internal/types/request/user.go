package request


// UserRegister 用户注册请求
type UserRegister struct {
    Username string `json:"username" binding:"required,min=3,max=50,alphanum"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=6,max=100"`
    FullName string `json:"full_name" binding:"omitempty,max=100"`
}

// UserLogin 用户登录请求
type UserLogin struct {
    Username string `json:"username" binding:"required,min=1,max=100"` // 用户名或邮箱
    Password string `json:"password" binding:"required,min=1,max=100"`
}

// UserUpdate 用户更新请求
type UserUpdate struct {
    FullName *string `json:"full_name" binding:"omitempty,max=100"`
    Avatar   *string `json:"avatar" binding:"omitempty,url"`
    Bio      *string `json:"bio" binding:"omitempty,max=500"`
}

// UserPreferences 用户偏好设置
type UserPreferences struct {
    Theme               string `json:"theme" binding:"omitempty,oneof=light dark auto"`
    Language            string `json:"language" binding:"omitempty,len=2"`
    DefaultLLMProvider  string `json:"default_llm_provider" binding:"omitempty,oneof=deepseek zhipu ollama"`
    DefaultModel        string `json:"default_model" binding:"omitempty,max=100"`
    StreamEnabled       bool   `json:"stream_enabled"`
    NotificationEnabled bool   `json:"notification_enabled"`
    AutoSaveEnabled     bool   `json:"auto_save_enabled"`
    TimeZone            string `json:"timezone" binding:"omitempty,max=50"`
    // 聊天记忆功能
    MemoryEnabled      *bool   `json:"memory_enabled,omitempty"`       // 是否启用聊天记忆
    CustomSystemPrompt *string `json:"custom_system_prompt,omitempty"` // 用户自定义系统提示词
    MaxContextTokens   *int    `json:"max_context_tokens,omitempty"`   // 最大上下文 token 数
}

// MemorySettingsRequest 记忆设置请求
type MemorySettingsRequest struct {
    MemoryEnabled      bool   `json:"memory_enabled"`                                  // 是否启用聊天记忆
    CustomSystemPrompt string `json:"custom_system_prompt" binding:"omitempty,max=5000"` // 用户自定义系统提示词
    MaxContextTokens   int    `json:"max_context_tokens" binding:"omitempty,min=1000,max=200000"` // 最大上下文 token 数
}

// UserRefreshTokenRequest 用户刷新令牌请求
type UserRefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
    CurrentPassword string `json:"current_password" binding:"required,min=1"`
    NewPassword     string `json:"new_password" binding:"required,min=6,max=100"`
    ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=NewPassword"`
}

// ForgotPasswordRequest 忘记密码请求
type ForgotPasswordRequest struct {
    Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest 重置密码请求
type ResetPasswordRequest struct {
    Token    string `json:"token" binding:"required"`
    Password string `json:"password" binding:"required,min=6,max=100"`
}

// UserStatsRequest 用户统计请求
type UserStatsRequest struct {
    UserID string `json:"user_id,omitempty" form:"user_id"` // 管理员可查询其他用户
}

// UserActivityRequest 用户活动请求
type UserActivityRequest struct {
    UserID string `json:"user_id,omitempty" form:"user_id"`
    Limit  int    `json:"limit" form:"limit" binding:"min=1,max=100"`
    Offset int    `json:"offset" form:"offset" binding:"min=0"`
}

// DeleteAccountRequest 删除账户请求
type DeleteAccountRequest struct {
    Password string `json:"password" binding:"required"`
    Confirm  bool   `json:"confirm" binding:"required,eq=true"`
}

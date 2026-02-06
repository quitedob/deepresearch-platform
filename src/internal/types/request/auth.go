package request

// RegisterRequest 用户注册请求
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Username string `json:"username" binding:"required,min=3,max=50"`
    Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest 用户登录请求
type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest 令牌刷新请求
type RefreshTokenRequest struct {
    Token string `json:"token" binding:"required"`
}

package auth

import (
    "context"
    "github.com/ai-research-platform/internal/types/request"
    "github.com/ai-research-platform/internal/types/response"
)

// Service 认证服务接口
type Service interface {
    // Register 用户注册
    Register(ctx context.Context, req request.RegisterRequest) (*response.AuthResponse, error)

    // Login 用户登录
    Login(ctx context.Context, req request.LoginRequest) (*response.AuthResponse, error)

    // RefreshToken 刷新令牌
    RefreshToken(ctx context.Context, req request.RefreshTokenRequest) (*response.AuthResponse, error)

    // ValidateToken 验证令牌
    ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
}

// TokenClaims 令牌声明
type TokenClaims struct {
    UserID   string `json:"user_id"`
    Email    string `json:"email"`
    Username string `json:"username"`
}

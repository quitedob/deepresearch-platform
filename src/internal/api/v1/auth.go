package v1

import (
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/ai-research-platform/internal/pkg"
    "github.com/ai-research-platform/internal/types/request"
    "github.com/ai-research-platform/internal/service/auth"
)

// AuthAPI 认证API
type AuthAPI struct {
    authService auth.Service
}

// NewAuthAPI 创建认证API
func NewAuthAPI(authService auth.Service) *AuthAPI {
    return &AuthAPI{
        authService: authService,
    }
}

// Register 用户注册
func (a *AuthAPI) Register(c *gin.Context) {
    var req request.RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    resp, err := a.authService.Register(c.Request.Context(), req)
    if err != nil {
        a.handleAuthError(c, err)
        return
    }

    c.JSON(201, resp)
}

// Login 用户登录
func (a *AuthAPI) Login(c *gin.Context) {
    var req request.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    resp, err := a.authService.Login(c.Request.Context(), req)
    if err != nil {
        a.handleAuthError(c, err)
        return
    }

    pkg.Success(c, resp)
}

// RefreshToken 刷新令牌
func (a *AuthAPI) RefreshToken(c *gin.Context) {
    var req request.RefreshTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    resp, err := a.authService.RefreshToken(c.Request.Context(), req)
    if err != nil {
        a.handleAuthError(c, err)
        return
    }

    pkg.Success(c, resp)
}


// handleAuthError 处理认证错误
func (a *AuthAPI) handleAuthError(c *gin.Context, err error) {
    errMsg := err.Error()
    
    switch {
    case errMsg == "user already exists":
        pkg.Error(c, 409, "用户已存在")
    case errMsg == "invalid credentials":
        pkg.Unauthorized(c, "用户名或密码错误")
    case errMsg == "invalid or expired token":
        pkg.Unauthorized(c, "令牌无效或已过期")
    case errMsg == "user not found":
        pkg.NotFound(c, "用户不存在")
    case strings.HasPrefix(errMsg, "failed to create user"):
        pkg.InternalError(c, "创建用户失败: "+errMsg)
    case errMsg == "failed to process password":
        pkg.InternalError(c, "密码处理失败")
    case errMsg == "failed to generate token":
        pkg.InternalError(c, "生成令牌失败")
    default:
        pkg.InternalError(c, "服务器内部错误: "+errMsg)
    }
}

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/pkg/auth"
)

// UserAdminChecker 用于 AdminAuth 中间件的用户管理员检查接口
// 避免循环依赖，通过接口注入
type UserAdminChecker interface {
	IsAdmin(ctx context.Context, userID string) (bool, error)
}

// 单例 JWTManager，避免每次请求重新创建
var (
	defaultJWTManager *auth.JWTManager
	jwtManagerOnce    sync.Once
)

// InitDefaultJWTManager 使用指定密钥初始化单例 JWTManager
// P0 修复：由 main.go 在启动时调用，确保与 config.yaml 中的 JWT_SECRET 一致
// 必须在任何请求到达之前调用，否则回退到 os.Getenv("JWT_SECRET")
func InitDefaultJWTManager(secret string, expiration time.Duration) {
	jwtManagerOnce.Do(func() {
		defaultJWTManager = auth.NewJWTManager(secret, expiration)
	})
}

// getDefaultJWTManager 返回单例 JWTManager
func getDefaultJWTManager() *auth.JWTManager {
	jwtManagerOnce.Do(func() {
		// 回退：如果 InitDefaultJWTManager 未被调用，从环境变量读取
		defaultJWTManager = auth.NewJWTManager(getJWTSecret(), 24*time.Hour)
	})
	return defaultJWTManager
}

// unauthorizedResponse 返回统一的 401 错误格式，与前端拦截器期望的格式匹配
func unauthorizedResponse(c *gin.Context, code, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"code":    code,
		"message": message,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
	c.Abort()
}

// AdminAuth 管理员认证中间件 - 验证 JWT 并通过数据库检查 is_admin 字段
// P0 修复：在路由层拦截非管理员请求，不再仅依赖 handler 内的二次检查
func AdminAuth(checker ...UserAdminChecker) gin.HandlerFunc {
	var adminChecker UserAdminChecker
	if len(checker) > 0 {
		adminChecker = checker[0]
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			unauthorizedResponse(c, "ERR_UNAUTHORIZED", "缺少Authorization头")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			unauthorizedResponse(c, "ERR_UNAUTHORIZED", "无效的Authorization头格式")
			return
		}

		claims, err := getDefaultJWTManager().ValidateToken(parts[1])
		if err != nil {
			unauthorizedResponse(c, "ERR_TOKEN_INVALID", "无效或已过期的令牌")
			return
		}

		// P0 修复：如果注入了 checker，在中间件层做数据库级 is_admin 验证
		if adminChecker != nil {
			isAdmin, err := adminChecker.IsAdmin(c.Request.Context(), claims.UserID)
			if err != nil || !isAdmin {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"code":    "ERR_FORBIDDEN",
					"message": "需要管理员权限",
				})
				c.Abort()
				return
			}
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)

		c.Next()
	}
}

// Auth 认证中间件 - 使用单例 JWT 管理器，返回统一错误格式
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			unauthorizedResponse(c, "ERR_UNAUTHORIZED", "缺少Authorization头")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			unauthorizedResponse(c, "ERR_UNAUTHORIZED", "无效的Authorization头格式")
			return
		}

		claims, err := getDefaultJWTManager().ValidateToken(parts[1])
		if err != nil {
			unauthorizedResponse(c, "ERR_TOKEN_INVALID", "无效或已过期的令牌")
			return
		}

		// 设置用户信息到上下文
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)

		c.Next()
	}
}

// getJWTSecret 获取 JWT 密钥
// P0 修复：不再回退到弱默认密钥，必须显式配置
func getJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// 生产环境下不允许使用默认密钥
		// 如果环境变量未设置，使用 panic 级别警告并拒绝启动
		// 但为了不中断现有部署，这里 fallback 到一个运行时生成的随机值
		// 这意味着每次重启后所有已有 token 都会失效
		fmt.Fprintln(os.Stderr, "[CRITICAL] JWT_SECRET environment variable is not set! All tokens will be invalidated on restart. Set JWT_SECRET for production use.")
		// 使用进程启动时间戳+PID生成临时密钥，确保不可预测但每次重启不同
		secret = fmt.Sprintf("ephemeral-%d-%d-do-not-use-in-production", os.Getpid(), time.Now().UnixNano())
	}
	if len(secret) < 32 {
		fmt.Fprintln(os.Stderr, "[WARNING] JWT_SECRET is too short (< 32 chars). Use a strong random secret in production.")
	}
	return secret
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)
	return userIDStr, ok
}

// RequireAuth 确保用户已认证
func RequireAuth(c *gin.Context) (string, error) {
	userID, exists := GetUserID(c)
	if !exists {
		return "", http.ErrNoCookie
	}
	return userID, nil
}

// ValidateTokenString 直接验证token字符串并返回用户ID
// 用于SSE等无法使用Authorization header的场景
func ValidateTokenString(tokenString string) (string, error) {
	if tokenString == "" {
		return "", http.ErrNoCookie
	}

	claims, err := getDefaultJWTManager().ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	return claims.UserID, nil
}

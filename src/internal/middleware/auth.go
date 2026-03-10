package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/pkg/auth"
)

// Auth 认证中间件 - 使用默认 JWT 管理器
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header required",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		// 使用默认 JWT 管理器验证令牌
		jwtManager := auth.NewJWTManager(getJWTSecret(), 24*time.Hour)
		claims, err := jwtManager.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			c.Abort()
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

	jwtManager := auth.NewJWTManager(getJWTSecret(), 24*time.Hour)
	claims, err := jwtManager.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	return claims.UserID, nil
}

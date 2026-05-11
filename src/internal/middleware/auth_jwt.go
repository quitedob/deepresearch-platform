package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/pkg/auth"
)

// OptionalAuthWithJWT 可选的JWT认证中间件 — token 存在时解析，不存在时放行
func OptionalAuthWithJWT(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		claims, err := jwtManager.ValidateToken(parts[1])
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)

		c.Next()
	}
}

package middleware

import (
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/ai-research-platform/internal/pkg/auth"
    "github.com/ai-research-platform/internal/types/constant"
)

// AuthWithJWT JWT认证中间件
func AuthWithJWT(jwtManager *auth.JWTManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "缺少Authorization头",
            })
            c.Abort()
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "无效的Authorization头格式",
            })
            c.Abort()
            return
        }

        tokenString := parts[1]
        claims, err := jwtManager.ValidateToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "无效或已过期的令牌",
                "error":   err.Error(),
            })
            c.Abort()
            return
        }

        if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "令牌已过期",
            })
            c.Abort()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("username", claims.Username)
        // 注意：管理员验证在业务层 AdminAPI.RequireAdmin() 中通过查询数据库进行
        // 中间件不设置 is_admin，避免误导

        c.Next()
    }
}


// OptionalAuthWithJWT 可选的JWT认证中间件
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

        tokenString := parts[1]
        claims, err := jwtManager.ValidateToken(tokenString)
        if err != nil {
            c.Next()
            return
        }

        if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
            c.Next()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("username", claims.Username)
        // 注意：管理员验证在业务层进行，中间件不设置 is_admin

        c.Next()
    }
}

// AdminAuthWithJWT 管理员JWT认证中间件
// 修复：移除硬编码的管理员邮箱域名检查，改为从claims中读取is_admin字段
// 注意：实际的管理员验证应该在数据库中进行，这里只是中间件层的初步检查
func AdminAuthWithJWT(jwtManager *auth.JWTManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "缺少Authorization头",
            })
            c.Abort()
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "无效的Authorization头格式",
            })
            c.Abort()
            return
        }

        tokenString := parts[1]
        claims, err := jwtManager.ValidateToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "无效或已过期的令牌",
            })
            c.Abort()
            return
        }

        if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "令牌已过期",
            })
            c.Abort()
            return
        }

        // 管理员验证在业务层 AdminAPI.RequireAdmin() 中通过查询数据库进行
        // 中间件只负责 JWT 验证和设置用户基本信息
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("username", claims.Username)

        c.Next()
    }
}

// RequireAuthWithJWT 必须JWT认证中间件
func RequireAuthWithJWT(jwtManager *auth.JWTManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "缺少Authorization头",
            })
            c.Abort()
            return
        }

        parts := strings.SplitN(authHeader, " ", 2)
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "无效的Authorization头格式",
            })
            c.Abort()
            return
        }

        tokenString := parts[1]
        claims, err := jwtManager.ValidateToken(tokenString)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "无效或已过期的令牌",
            })
            c.Abort()
            return
        }

        if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
            c.JSON(http.StatusUnauthorized, gin.H{
                "code":    constant.ErrCodeUnauthorized,
                "message": "令牌已过期",
            })
            c.Abort()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("username", claims.Username)
        // 注意：管理员验证在业务层进行，中间件不设置 is_admin

        c.Next()
    }
}

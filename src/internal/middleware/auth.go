package middleware

import (
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/ai-research-platform/internal/pkg/auth"
)

// Auth 璁よ瘉涓棿浠?- 浣跨敤榛樿 JWT 绠＄悊鍣?
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

        // 浣跨敤榛樿 JWT 绠＄悊鍣ㄩ獙璇佷护鐗?
        jwtManager := auth.NewJWTManager(getJWTSecret(), 24*time.Hour)
        claims, err := jwtManager.ValidateToken(parts[1])
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "invalid or expired token",
            })
            c.Abort()
            return
        }

        // 璁剧疆鐢ㄦ埛淇℃伅鍒颁笂涓嬫枃
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("username", claims.Username)

        c.Next()
    }
}

// getJWTSecret 获取 JWT 密钥
func getJWTSecret() string {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = "change-this-to-a-strong-random-secret-in-production"
    }
    return secret
}

// GetUserID 浠庝笂涓嬫枃鑾峰彇鐢ㄦ埛ID
func GetUserID(c *gin.Context) (string, bool) {
    userID, exists := c.Get("user_id")
    if !exists {
        return "", false
    }

    userIDStr, ok := userID.(string)
    return userIDStr, ok
}

// RequireAuth 纭繚鐢ㄦ埛宸茶璇?
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

package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/types/constant"
)

// CORS CORS中间件
// P1 修复：不再使用 Allow-Origin: * + Allow-Credentials: true 的非法组合
// 根据 Fetch 规范，当 credentials 模式为 "include" 时，Access-Control-Allow-Origin 不能为 *
func CORS() gin.HandlerFunc {
	// 从环境变量读取允许的源列表，逗号分隔
	// 例如: CORS_ALLOWED_ORIGINS=http://localhost:5173,https://yourdomain.com
	allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins map[string]bool
	if allowedOriginsStr != "" {
		allowedOrigins = make(map[string]bool)
		for _, origin := range strings.Split(allowedOriginsStr, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				allowedOrigins[origin] = true
			}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// 确定是否允许该来源
		var allowOrigin string
		if allowedOrigins != nil && len(allowedOrigins) > 0 {
			// 显式白名单模式
			if allowedOrigins[origin] {
				allowOrigin = origin
			}
		} else {
			// 开发模式：允许 localhost 和常见开发端口
			if origin != "" && isDevOrigin(origin) {
				allowOrigin = origin
			} else if origin != "" {
				// 生产环境下如果没配置白名单，拒绝 CORS
				// 但仍允许同源请求（无 Origin 头时不设置 CORS 头）
				allowOrigin = ""
			}
		}

		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", constant.CORSCacheMaxAge)
			// Vary header 确保代理按 Origin 正确缓存
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isDevOrigin 判断是否是开发环境来源
func isDevOrigin(origin string) bool {
	devPatterns := []string{
		"http://localhost:",
		"http://127.0.0.1:",
		"http://[::1]:",
		"https://localhost:",
	}
	for _, pattern := range devPatterns {
		if strings.HasPrefix(origin, pattern) {
			return true
		}
	}
	return origin == "http://localhost" || origin == "https://localhost"
}

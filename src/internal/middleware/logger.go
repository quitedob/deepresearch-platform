package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// Logger 日志中间件 - 使用 zap 结构化日志
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		// 使用结构化日志记录请求
		log := logger.GetLogger()
		if log != nil {
			log.Info("HTTP Request",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.Duration("latency", latency),
				zap.String("client_ip", clientIP),
				zap.String("user_agent", c.Request.UserAgent()),
				zap.Int("body_size", c.Writer.Size()),
			)
		}
	}
}

// CustomLogger 自定义日志中间件
func CustomLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		// 使用结构化日志记录请求
		log := logger.GetLogger()
		if log != nil {
			log.Info("request processed",
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", statusCode),
				zap.Duration("latency", latency),
				zap.String("ip", clientIP),
			)
		}
	}
}

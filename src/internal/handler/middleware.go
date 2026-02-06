package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ai-research-platform/internal/pkg/errors"
)

// LoggingMiddleware 创建记录所有HTTP请求的中间件
func LoggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 获取用户ID（如果已认证）
		userID := ""
		if uid, exists := c.Get("user_id"); exists {
			if uidStr, ok := uid.(string); ok {
				userID = uidStr
			}
		}

		// 记录请求
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", userAgent),
		}

		if userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}

		// 单独记录错误
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
			logger.Error("Request completed with errors", fields...)
		} else if statusCode >= 500 {
			logger.Error("Request failed", fields...)
		} else if statusCode >= 400 {
			logger.Warn("Request client error", fields...)
		} else {
			logger.Info("Request completed", fields...)
		}
	}
}

// SecurityHeadersMiddleware 为所有响应添加安全头
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

// ErrorHandlingMiddleware 创建处理错误和panic的中间件
func ErrorHandlingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 记录panic和堆栈跟踪
				logger.Error("Panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Stack("stack"),
				)

				// 返回内部服务器错误
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "An unexpected error occurred",
				})
				c.Abort()
			}
		}()

		c.Next()

		// 处理来自处理器的错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			// 检查是否为AppError
			if appErr, ok := err.Err.(*errors.AppError); ok {
				// 记录带有上下文的错误
				fields := []zap.Field{
					zap.String("code", appErr.Code),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				}

				if appErr.Err != nil {
					fields = append(fields, zap.Error(appErr.Err))
				}

				if appErr.Details != "" {
					fields = append(fields, zap.String("details", appErr.Details))
				}

				// 根据严重程度记录日志
				if appErr.StatusCode >= 500 {
					logger.Error(appErr.Message, fields...)
				} else {
					logger.Warn(appErr.Message, fields...)
				}

				// 返回错误响应
				response := gin.H{
					"code":    appErr.Code,
					"message": appErr.Message,
				}
				if appErr.Details != "" {
					response["details"] = appErr.Details
				}

				c.JSON(appErr.StatusCode, response)
			} else {
				// 通用错误处理
				logger.Error("Unhandled error",
					zap.Error(err.Err),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)

				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "An error occurred processing your request",
				})
			}
			c.Abort()
		}
	}
}

package handler

import (
	"net/http"

	"github.com/ai-research-platform/internal/cache"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthCheckHandler 返回简单的健康检查
func HealthCheckHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	}
}

// ReadinessCheckHandler 检查服务是否准备好接受流量
func ReadinessCheckHandler(db *gorm.DB, cache cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查数据库连接
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"reason": "database connection error",
			})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"reason": "database ping failed",
			})
			return
		}

		// 检查缓存（可选 - 缓存失败不影响服务）
		if cache != nil {
			if err := cache.Ping(c.Request.Context()); err != nil {
				c.JSON(http.StatusOK, gin.H{
					"status":  "degraded",
					"reason":  "cache unavailable",
					"details": "service operational but cache is down",
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	}
}

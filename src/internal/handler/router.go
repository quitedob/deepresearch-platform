package handler

import (
	"time"

	"github.com/ai-research-platform/internal/auth"
	"github.com/ai-research-platform/internal/cache"
	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/monitoring"
	"github.com/ai-research-platform/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RouterConfig 包含路由器的配置
type RouterConfig struct {
	AllowOrigins []string
	AllowMethods []string
	JWTSecret    string
	JWTExpiration time.Duration
	RateLimitRPM int
}

// SetupRouter 创建并配置带有所有处理器和中间件的Gin路由器
func SetupRouter(
	db *gorm.DB,
	chatService *service.ChatService,
	researchService *service.ResearchService,
	llmScheduler *eino.LLMScheduler,
	cacheManager cache.Cache,
	config RouterConfig,
	logger *zap.Logger,
	metrics *monitoring.Metrics,
) *gin.Engine {
	// 创建路由器
	router := gin.New()

	// 添加恢复中间件
	router.Use(gin.Recovery())

	// 添加日志中间件
	router.Use(LoggingMiddleware(logger))

	// 添加指标中间件
	if metrics != nil {
		router.Use(monitoring.MetricsMiddleware(metrics))
	}

	// 添加CORS中间件
	corsConfig := cors.Config{
		AllowOrigins:     config.AllowOrigins,
		AllowMethods:     config.AllowMethods,
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(corsConfig))

	// 添加安全头中间件
	router.Use(SecurityHeadersMiddleware())

	// 指标端点（无需认证）
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 健康检查端点（无需认证）
	router.GET("/health", HealthCheckHandler(db))
	router.GET("/ready", ReadinessCheckHandler(db, cacheManager))

	// 创建JWT管理器
	jwtManager := auth.NewJWTManager(config.JWTSecret, config.JWTExpiration)

	// 创建速率限制器
	rateLimiter := middleware.NewRateLimiter(config.RateLimitRPM)

	// 认证处理器（无需认证）
	authHandler := NewAuthHandler(db, jwtManager)
	authGroup := router.Group("/api/v1/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
	}

	// API v1 路由（需要认证）
	v1 := router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(jwtManager))
	v1.Use(rateLimiter.RateLimitMiddleware())

	// 聊天处理器
	chatHandler := NewChatHandler(chatService)
	chatGroup := v1.Group("/chat")
	{
		chatGroup.POST("/sessions", chatHandler.CreateSession)
		chatGroup.GET("/sessions", chatHandler.ListSessions)
		chatGroup.GET("/sessions/:session_id", chatHandler.GetSession)
		chatGroup.DELETE("/sessions/:session_id", chatHandler.DeleteSession)
		chatGroup.POST("/sessions/:session_id/messages", chatHandler.SendMessage)
		chatGroup.GET("/sessions/:session_id/messages", chatHandler.GetMessages)
		chatGroup.GET("/sessions/:session_id/stream", chatHandler.StreamMessage)
		chatGroup.PUT("/sessions/:session_id/provider", chatHandler.UpdateProvider)
	}

	// 研究处理器
	researchHandler := NewResearchHandler(researchService)
	researchGroup := v1.Group("/research")
	{
		researchGroup.POST("/sessions", researchHandler.StartResearch)
		researchGroup.GET("/sessions", researchHandler.ListSessions)
		researchGroup.GET("/sessions/:session_id", researchHandler.GetSession)
		researchGroup.GET("/sessions/:session_id/results", researchHandler.GetResults)
		researchGroup.GET("/sessions/:session_id/tasks", researchHandler.GetTasks)
		researchGroup.GET("/sessions/:session_id/stream", researchHandler.StreamProgress)
		researchGroup.POST("/sessions/:session_id/cancel", researchHandler.CancelResearch)
	}

	// LLM处理器
	llmHandler := NewLLMHandler(llmScheduler)
	llmGroup := v1.Group("/llm")
	{
		llmGroup.GET("/providers", llmHandler.ListProviders)
		llmGroup.GET("/providers/:provider/metrics", llmHandler.GetProviderMetrics)
		llmGroup.GET("/models", llmHandler.ListModels)
		llmGroup.POST("/test", llmHandler.TestProvider)
	}

	return router
}

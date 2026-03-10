package api

import (
    "github.com/gin-gonic/gin"
    "github.com/ai-research-platform/internal/api/v1"
    "github.com/ai-research-platform/internal/middleware"
    "github.com/ai-research-platform/internal/pkg/auth"
)

// RouterEnhanced 增强的路由器
type RouterEnhanced struct {
    authAPI         *v1.UserAPIEnhanced
    chatAPI         *v1.ChatAPI
    researchAPI     *v1.ResearchAPI
    llmAPI          *v1.LLMAPI
    healthAPI       *v1.HealthAPI
    mcpAPI          *v1.MCPAPI
    adminAPI        *v1.AdminAPI
    membershipAPI   *v1.MembershipAPI
    notificationAPI *v1.NotificationAPI
    aiQuestionAPI   *v1.AIQuestionAPI
    paperAPI        *v1.PaperAPI
}

// NewRouterEnhanced 创建增强的路由器
func NewRouterEnhanced(
    authAPI *v1.UserAPIEnhanced,
    chatAPI *v1.ChatAPI,
    researchAPI *v1.ResearchAPI,
    llmAPI *v1.LLMAPI,
    healthAPI *v1.HealthAPI,
    mcpAPI *v1.MCPAPI,
) *RouterEnhanced {
    return &RouterEnhanced{
        authAPI:     authAPI,
        chatAPI:     chatAPI,
        researchAPI: researchAPI,
        llmAPI:      llmAPI,
        healthAPI:   healthAPI,
        mcpAPI:      mcpAPI,
    }
}

// NewRouterEnhancedWithAdmin 创建带管理员API的增强路由器
func NewRouterEnhancedWithAdmin(
    authAPI *v1.UserAPIEnhanced,
    chatAPI *v1.ChatAPI,
    researchAPI *v1.ResearchAPI,
    llmAPI *v1.LLMAPI,
    healthAPI *v1.HealthAPI,
    mcpAPI *v1.MCPAPI,
    adminAPI *v1.AdminAPI,
) *RouterEnhanced {
    return &RouterEnhanced{
        authAPI:     authAPI,
        chatAPI:     chatAPI,
        researchAPI: researchAPI,
        llmAPI:      llmAPI,
        healthAPI:   healthAPI,
        mcpAPI:      mcpAPI,
        adminAPI:    adminAPI,
    }
}

// NewRouterEnhancedFull 创建完整的增强路由器（包含所有API）
func NewRouterEnhancedFull(
    authAPI *v1.UserAPIEnhanced,
    chatAPI *v1.ChatAPI,
    researchAPI *v1.ResearchAPI,
    llmAPI *v1.LLMAPI,
    healthAPI *v1.HealthAPI,
    mcpAPI *v1.MCPAPI,
    adminAPI *v1.AdminAPI,
    membershipAPI *v1.MembershipAPI,
    notificationAPI *v1.NotificationAPI,
    aiQuestionAPI *v1.AIQuestionAPI,
    paperAPI ...*v1.PaperAPI,
) *RouterEnhanced {
    r := &RouterEnhanced{
        authAPI:         authAPI,
        chatAPI:         chatAPI,
        researchAPI:     researchAPI,
        llmAPI:          llmAPI,
        healthAPI:       healthAPI,
        mcpAPI:          mcpAPI,
        adminAPI:        adminAPI,
        membershipAPI:   membershipAPI,
        notificationAPI: notificationAPI,
        aiQuestionAPI:   aiQuestionAPI,
    }
    if len(paperAPI) > 0 && paperAPI[0] != nil {
        r.paperAPI = paperAPI[0]
    }
    return r
}

// SetupEnhanced 设置增强路由
func (r *RouterEnhanced) SetupEnhanced() *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    engine := gin.New()

    engine.Use(gin.Recovery())
    engine.Use(middleware.Logger())
    engine.Use(middleware.CORS())

    // 健康检查
    engine.GET("/health", r.healthAPI.Health)
    engine.GET("/ready", r.healthAPI.Ready)

    // API v1 路由组
    v1Group := engine.Group("/api/v1")
    {
        // 认证路由（公开）
        authGroup := v1Group.Group("/auth")
        {
            authGroup.POST("/register", r.authAPI.Register)
            authGroup.POST("/login", r.authAPI.Login)
            authGroup.POST("/refresh", r.authAPI.RefreshToken)
            authGroup.POST("/logout", r.authAPI.Logout)
        }

        // 用户路由（需要认证）
        userGroup := v1Group.Group("/user")
        userGroup.Use(middleware.Auth())
        {
            userGroup.GET("/profile", r.authAPI.GetCurrentUser)
            userGroup.PUT("/profile", r.authAPI.UpdateProfile)
            userGroup.GET("/preferences", r.authAPI.GetPreferences)
            userGroup.PUT("/preferences", r.authAPI.UpdatePreferences)
            userGroup.POST("/change-password", r.authAPI.ChangePassword)
            userGroup.GET("/stats", r.authAPI.GetUserStats)
            userGroup.DELETE("/delete-account", r.authAPI.DeleteAccount)
            // 记忆设置
            userGroup.GET("/memory-settings", r.authAPI.GetMemorySettings)
            userGroup.PUT("/memory-settings", r.authAPI.UpdateMemorySettings)
        }

        // 会员路由（需要认证）
        if r.membershipAPI != nil {
            membershipGroup := v1Group.Group("/membership")
            membershipGroup.Use(middleware.Auth())
            {
                membershipGroup.GET("", r.membershipAPI.GetMembership)
                membershipGroup.GET("/quota", r.membershipAPI.GetQuota)
                membershipGroup.POST("/activate", r.membershipAPI.ActivateCode)
                membershipGroup.GET("/check-chat-quota", r.membershipAPI.CheckChatQuota)
                membershipGroup.GET("/check-research-quota", r.membershipAPI.CheckResearchQuota)
            }
        }

        // 通知路由（需要认证）
        if r.notificationAPI != nil {
            notificationGroup := v1Group.Group("/notifications")
            notificationGroup.Use(middleware.Auth())
            {
                notificationGroup.GET("", r.notificationAPI.GetNotifications)
                notificationGroup.GET("/unread-count", r.notificationAPI.GetUnreadCount)
                notificationGroup.POST("/:id/read", r.notificationAPI.MarkAsRead)
                notificationGroup.POST("/read-all", r.notificationAPI.MarkAllAsRead)
            }
        }

        // 聊天路由
        chatGroup := v1Group.Group("/chat")
        {
            chatGroup.GET("/models", r.chatAPI.GetModels)
        }

        authorizedChat := v1Group.Group("/chat")
        authorizedChat.Use(middleware.Auth())
        {
            authorizedChat.POST("/sessions", r.chatAPI.CreateSession)
            authorizedChat.GET("/sessions", r.chatAPI.GetSessions)
            authorizedChat.GET("/sessions/:id", r.chatAPI.GetSession)
            authorizedChat.PUT("/sessions/:id", r.chatAPI.UpdateSession)
            authorizedChat.DELETE("/sessions/:id", r.chatAPI.DeleteSession)
            authorizedChat.GET("/sessions/:id/messages", r.chatAPI.GetMessages)
            authorizedChat.DELETE("/sessions/:id/messages", r.chatAPI.ClearMessages)
            authorizedChat.POST("/chat", r.chatAPI.Chat)
            authorizedChat.POST("/chat/stream", r.chatAPI.ChatStream)
            authorizedChat.POST("/chat/web-search", r.chatAPI.ChatWebSearch)
            // 上下文状态和总结
            authorizedChat.GET("/sessions/:id/context-status", r.chatAPI.GetContextStatus)
            authorizedChat.POST("/sessions/:id/summarize-and-new", r.chatAPI.SummarizeAndNewSession)
        }

        // 研究路由 - SSE流端点不使用Auth中间件（在handler中自行验证token）
        researchStreamGroup := v1Group.Group("/research")
        {
            // SSE流端点 - 支持query参数传递token（因为EventSource不支持自定义header）
            researchStreamGroup.GET("/stream/:session_id", r.researchAPI.StreamResearchProgress)
        }

        // 研究路由 - 需要认证
        researchGroup := v1Group.Group("/research")
        researchGroup.Use(middleware.Auth())
        {
            researchGroup.POST("/start", r.researchAPI.StartResearch)
            researchGroup.GET("/status/:session_id", r.researchAPI.GetResearchStatus)
            researchGroup.GET("/sessions", r.researchAPI.GetResearchSessions)
            researchGroup.GET("/export/:session_id", r.researchAPI.ExportResearch)
            researchGroup.GET("/search", r.researchAPI.SearchResearch)
            researchGroup.GET("/statistics", r.researchAPI.GetResearchStatistics)
        }

        // LLM管理路由
        llmGroup := v1Group.Group("/llm")
        {
            llmGroup.GET("/providers", r.llmAPI.ListProviders)
            llmGroup.GET("/models", r.llmAPI.ListModels)
            llmGroup.GET("/metrics", r.llmAPI.GetMetrics)
        }

        authorizedLLM := v1Group.Group("/llm")
        authorizedLLM.Use(middleware.Auth())
        {
            authorizedLLM.POST("/test", r.llmAPI.TestProvider)
        }

        // MCP工具路由
        mcpGroup := v1Group.Group("/mcp")
        {
            mcpGroup.GET("/tools", r.mcpAPI.GetTools)
            mcpGroup.GET("/tools/:tool_name", r.mcpAPI.GetToolInfo)
        }

        authorizedMCP := v1Group.Group("/mcp")
        authorizedMCP.Use(middleware.Auth())
        {
            authorizedMCP.POST("/tools/call", r.mcpAPI.CallTool)
        }

        // 管理员路由
        if r.adminAPI != nil {
            adminGroup := v1Group.Group("/admin")
            adminGroup.Use(middleware.Auth())
            {
                // 统计信息
                adminGroup.GET("/stats", r.adminAPI.GetAdminStats)

                // 用户管理
                adminGroup.GET("/users", r.adminAPI.ListUsers)
                adminGroup.PUT("/users/:id/status", r.adminAPI.UpdateUserStatus)
                adminGroup.PUT("/users/:id/membership", r.adminAPI.UpdateUserMembership)
                adminGroup.POST("/users/:id/reset-quota", r.adminAPI.ResetUserQuota)
                adminGroup.PUT("/users/:id/quota", r.adminAPI.SetUserQuota)

                // 聊天记录管理
                adminGroup.GET("/users/:id/chat-history", r.adminAPI.GetUserChatHistory)
                adminGroup.GET("/users/:id/chat-history/export", r.adminAPI.ExportUserChatHistory)

                // 激活码管理
                adminGroup.GET("/activation-codes", r.adminAPI.ListActivationCodes)
                adminGroup.POST("/activation-codes", r.adminAPI.CreateActivationCode)
                adminGroup.GET("/activation-codes/:id", r.adminAPI.GetActivationCodeDetails)
                adminGroup.PUT("/activation-codes/:id", r.adminAPI.UpdateActivationCode)
                adminGroup.DELETE("/activation-codes/:id", r.adminAPI.DeleteActivationCode)

                // 通知管理
                adminGroup.GET("/notifications", r.adminAPI.ListNotifications)
                adminGroup.POST("/notifications", r.adminAPI.CreateNotification)
                adminGroup.DELETE("/notifications/:id", r.adminAPI.DeleteNotification)

                // 模型配置管理
                adminGroup.GET("/providers", r.adminAPI.GetProviderConfigs)
                adminGroup.PUT("/providers", r.adminAPI.UpdateProviderConfig)
                adminGroup.GET("/models", r.adminAPI.GetModelConfigs)
                adminGroup.PUT("/models", r.adminAPI.UpdateModelConfig)
                adminGroup.PUT("/models/batch", r.adminAPI.BatchUpdateModelConfigs)

                // 模型测试（管理端专用）
                adminGroup.POST("/models/test", r.adminAPI.TestModel)
                adminGroup.GET("/models/registered", r.adminAPI.GetAllRegisteredModels)
                adminGroup.POST("/models/sync", r.adminAPI.SyncModelsToDatabase)

                // 配额配置管理
                adminGroup.GET("/quota-configs", r.adminAPI.GetQuotaConfigs)
                adminGroup.PUT("/quota-configs", r.adminAPI.UpdateQuotaConfig)
                adminGroup.PUT("/users/:id/custom-quota", r.adminAPI.SetUserCustomQuota)
                adminGroup.PUT("/users/batch-quota", r.adminAPI.BatchSetUserQuota)

                // 批量操作
                adminGroup.PUT("/users/batch-status", r.adminAPI.BatchUpdateUserStatus)
                adminGroup.POST("/users/batch-reset-quota", r.adminAPI.BatchResetUserQuotas)
                adminGroup.DELETE("/users/batch", r.adminAPI.BatchDeleteUsers)
            }
        }

        // AI题目生成路由
        if r.aiQuestionAPI != nil {
            aiGroup := v1Group.Group("/ai")
            aiGroup.Use(middleware.Auth())
            {
                // 题目生成
                aiGroup.POST("/generate-questions", r.aiQuestionAPI.GenerateQuestions)
                
                // 会话管理
                aiGroup.POST("/question-sessions", r.aiQuestionAPI.CreateSession)
                aiGroup.GET("/question-sessions", r.aiQuestionAPI.ListSessions)
                aiGroup.GET("/question-sessions/:id", r.aiQuestionAPI.GetSession)
                aiGroup.PUT("/question-sessions/:id", r.aiQuestionAPI.UpdateSessionTitle)
                aiGroup.DELETE("/question-sessions/:id", r.aiQuestionAPI.DeleteSession)
                aiGroup.POST("/question-sessions/:id/questions", r.aiQuestionAPI.SaveQuestionsToSession)
                
                // 配置（公开获取，管理员更新）
                aiGroup.GET("/question-config", r.aiQuestionAPI.GetAIQuestionConfig)
            }
            
            // 管理员AI出题配置
            if r.adminAPI != nil {
                adminAIGroup := v1Group.Group("/admin/ai")
                adminAIGroup.Use(middleware.Auth())
                {
                    adminAIGroup.PUT("/question-config", r.aiQuestionAPI.UpdateAIQuestionConfig)
                }
            }
        }

        // 论文生成路由
        if r.paperAPI != nil {
            // 公开接口（模板/引用格式查询）
            paperPublicGroup := v1Group.Group("/paper")
            {
                paperPublicGroup.GET("/templates", r.paperAPI.GetTemplates)
                paperPublicGroup.GET("/citation-styles", r.paperAPI.GetCitationStyles)
            }

            // 需要认证的论文路由（包括 SSE 流）
            paperGroup := v1Group.Group("/paper")
            paperGroup.Use(middleware.Auth())
            {
                paperGroup.POST("/start", r.paperAPI.StartPaperGeneration)
                paperGroup.GET("/status/:id", r.paperAPI.GetPaperStatus)
                paperGroup.GET("/result/:id", r.paperAPI.GetPaperResult)
                paperGroup.GET("/export/:id", r.paperAPI.ExportPaper)
                paperGroup.GET("/list", r.paperAPI.ListPapers)
                paperGroup.DELETE("/:id", r.paperAPI.DeletePaper)
                paperGroup.POST("/regenerate", r.paperAPI.RegenerateChapter)
                // SSE 流端点也需要认证
                paperGroup.GET("/stream/:id", r.paperAPI.StreamProgress)
            }
        }
    }

    return engine
}

// SetupJWTAuthMiddleware 设置JWT认证中间件
func SetupJWTAuthMiddleware(jwtSecret string, expiration int64) gin.HandlerFunc {
    jwtManager := auth.NewJWTManager(jwtSecret, 0)
    return middleware.AuthWithJWT(jwtManager)
}

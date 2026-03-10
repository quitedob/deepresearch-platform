package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-research-platform/internal/api"
	v1 "github.com/ai-research-platform/internal/api/v1"
	"github.com/ai-research-platform/internal/cache"
	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/ai-research-platform/internal/pkg/eino/agent"
	"github.com/ai-research-platform/internal/infrastructure/config"
	"github.com/ai-research-platform/internal/infrastructure/database"
	"github.com/ai-research-platform/internal/infrastructure/logger"
	pkgauth "github.com/ai-research-platform/internal/pkg/auth"
	"github.com/ai-research-platform/internal/repository"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/service"
	"github.com/ai-research-platform/internal/types/constant"
	"go.uber.org/zap"
)

// maskAPIKey 安全地掩码 API 密钥，仅显示前4字符
// P1 修复：避免对短字符串切片导致 panic，减少敏感信息暴露
func maskAPIKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return key[:4] + "****"
}

func main() {
	configPath := flag.String("config", "src/configs/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 加载模型配置文件
	modelsConfigPath := "src/configs/models.yaml"
	if _, err := config.LoadModelsConfig(modelsConfigPath); err != nil {
		fmt.Fprintf(os.Stderr, "加载模型配置失败: %v\n", err)
		// 不退出，使用默认值
	}

	loggerCfg := logger.Config{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		OutputPath: cfg.Logging.OutputPath,
	}
	if err := logger.Initialize(loggerCfg); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志器失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	log := logger.GetLogger()
	log.Info("启动AI研究平台",
		zap.String("环境", cfg.Server.Env),
		zap.Int("端口", cfg.Server.Port),
	)

	// 初始化数据库连接（自动创建数据库如果不存在）
	log.Info("初始化数据库...")
	dbConfig := database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		DBName:          cfg.Database.DBName,
		SSLMode:         cfg.Database.SSLMode,
		MaxConnections:  cfg.Database.MaxConnections,
		IdleConnections: cfg.Database.IdleConnections,
	}

	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Warn("数据库连接失败，使用模拟模式", zap.Error(err))
		db = nil
	}

	var sqlDB interface{ Close() error }
	if db != nil {
		// 执行完整的数据库迁移（检查表结构、创建缺失表、初始化默认数据）
		log.Info("执行数据库迁移...")

		// 准备管理员配置
		adminCfg := &database.AdminConfig{
			Email:    cfg.Admin.Email,
			Username: cfg.Admin.Username,
			Password: cfg.Admin.Password,
		}

		if err := database.RunFullMigrationWithAdmin(db, log, adminCfg); err != nil {
			log.Error("数据库迁移失败", zap.Error(err))
			// 迁移失败不应该阻止启动，但需要警告
			log.Warn("数据库迁移失败，部分功能可能不可用")
		}

		sqlDBInner, err := db.DB()
		if err == nil {
			sqlDB = sqlDBInner
		}
	}

	// 初始化DAO层
	var userDAO *dao.UserDAO
	var chatDAO *dao.ChatDAO
	var researchDAO *dao.ResearchDAO
	var notificationDAO *dao.NotificationDAO
	var userPreferencesDAO *dao.UserPreferencesDAO
	var membershipDAO *dao.MembershipDAO
	var activationCodeDAO *dao.ActivationCodeDAO
	var modelConfigDAO *dao.ModelConfigDAO
	var quotaConfigDAO *dao.QuotaConfigDAO
	var aiQuestionDAO *dao.AIQuestionDAO
	var paperDAO *dao.PaperDAO
	var toolCallDAO *dao.ToolCallDAO
	if db != nil {
		userDAO = dao.NewUserDAO(db)
		chatDAO = dao.NewChatDAO(db)
		researchDAO = dao.NewResearchDAO(db)
		notificationDAO = dao.NewNotificationDAO(db)
		userPreferencesDAO = dao.NewUserPreferencesDAO(db)
		membershipDAO = dao.NewMembershipDAO(db)
		activationCodeDAO = dao.NewActivationCodeDAO(db)
		modelConfigDAO = dao.NewModelConfigDAO(db)
		quotaConfigDAO = dao.NewQuotaConfigDAO(db)
		aiQuestionDAO = dao.NewAIQuestionDAO(db)
		paperDAO = dao.NewPaperDAO(db)
		toolCallDAO = dao.NewToolCallDAO(db)
	}

	// 初始化认证组件
	jwtManager := pkgauth.NewJWTManager(cfg.Security.JWTSecret, time.Duration(cfg.Security.JWTExpiration)*time.Second)

	// 初始化LLM调度器
	llmScheduler := eino.NewLLMScheduler()

	// 注册LLM提供商
	// DeepSeek - 为每个模型创建单独的实例
	if deepseekCfg, ok := cfg.LLM.Providers["deepseek"]; ok && deepseekCfg.APIKey != "" {
		log.Info("注册 DeepSeek Provider",
			zap.Strings("models", deepseekCfg.Models),
			zap.String("api_key_masked", maskAPIKey(deepseekCfg.APIKey)))
		for _, modelName := range deepseekCfg.Models {
			deepseekModel, err := createDeepSeekModel(deepseekCfg.APIKey, modelName)
			if err != nil {
				log.Error("DeepSeek模型创建失败",
					zap.String("model", modelName),
					zap.Error(err))
				continue
			}
			// 使用 "provider:model" 作为唯一的 provider name
			providerKey := constant.ProviderDeepSeek + ":" + modelName
			llmScheduler.RegisterProvider(providerKey, deepseekModel, []string{modelName})
			log.Info("DeepSeek模型注册成功", zap.String("model", modelName))
		}
	} else {
		log.Warn("DeepSeek Provider未配置或API密钥为空")
	}

	// 智谱AI - 使用JWT认证
	if zhipuCfg, ok := cfg.LLM.Providers["zhipu"]; ok && zhipuCfg.APIKey != "" {
		log.Info("注册智谱AI Provider",
			zap.Strings("models", zhipuCfg.Models),
			zap.String("api_key_masked", maskAPIKey(zhipuCfg.APIKey)),
			zap.String("base_url", zhipuCfg.BaseURL))
		for _, modelName := range zhipuCfg.Models {
			zhipuModel, err := createZhipuModelWithJWT(zhipuCfg.APIKey, zhipuCfg.BaseURL, modelName)
			if err != nil {
				log.Error("智谱AI模型创建失败",
					zap.String("model", modelName),
					zap.Error(err))
				continue
			}
			// 使用 "provider:model" 作为唯一的 provider name
			providerKey := constant.ProviderZhipu + ":" + modelName
			llmScheduler.RegisterProvider(providerKey, zhipuModel, []string{modelName})
			log.Info("智谱AI模型注册成功", zap.String("model", modelName))
		}
	} else {
		log.Warn("智谱AI Provider未配置或API密钥为空")
	}

	// Ollama (本地，不需要API密钥) - 为每个模型创建单独的实例
	if ollamaCfg, ok := cfg.LLM.Providers["ollama"]; ok {
		log.Info("注册 Ollama Provider",
			zap.Strings("models", ollamaCfg.Models),
			zap.String("base_url", ollamaCfg.BaseURL))
		for _, modelName := range ollamaCfg.Models {
			ollamaModel, err := createOllamaModel(ollamaCfg.BaseURL, modelName)
			if err != nil {
				log.Error("Ollama模型创建失败",
					zap.String("model", modelName),
					zap.Error(err))
				continue
			}
			// 使用 "provider:model" 作为唯一的 provider name
			providerKey := constant.ProviderOllama + ":" + modelName
			llmScheduler.RegisterProvider(providerKey, ollamaModel, []string{modelName})
			log.Info("Ollama模型注册成功", zap.String("model", modelName))
		}
	} else {
		log.Warn("Ollama Provider未配置")
	}

	// OpenRouter - 支持 400+ AI 模型
	if openrouterCfg, ok := cfg.LLM.Providers["openrouter"]; ok && openrouterCfg.APIKey != "" {
		log.Info("注册 OpenRouter Provider",
			zap.Strings("models", openrouterCfg.Models),
			zap.String("api_key_masked", maskAPIKey(openrouterCfg.APIKey)),
			zap.String("base_url", openrouterCfg.BaseURL))
		for _, modelName := range openrouterCfg.Models {
			openrouterModel, err := createOpenRouterModel(openrouterCfg.APIKey, openrouterCfg.BaseURL, modelName)
			if err != nil {
				log.Error("OpenRouter模型创建失败",
					zap.String("model", modelName),
					zap.Error(err))
				continue
			}
			// 使用 "provider:model" 作为唯一的 provider name
			providerKey := constant.ProviderOpenRouter + ":" + modelName
			llmScheduler.RegisterProvider(providerKey, openrouterModel, []string{modelName})
			log.Info("OpenRouter模型注册成功", zap.String("model", modelName))
		}
	} else {
		log.Warn("OpenRouter Provider未配置或API密钥为空")
	}

	// OpenAI兼容 - GLM Coding Plan (白嫖coding plan余额)
	if openaiCfg, ok := cfg.LLM.Providers["openai"]; ok && openaiCfg.APIKey != "" {
		log.Info("注册 OpenAI兼容 Provider (GLM Coding Plan)",
			zap.Strings("models", openaiCfg.Models),
			zap.String("api_key_masked", maskAPIKey(openaiCfg.APIKey)),
			zap.String("base_url", openaiCfg.BaseURL))
		for _, modelName := range openaiCfg.Models {
			openaiModel, err := createOpenAICompatibleModel(openaiCfg.APIKey, openaiCfg.BaseURL, modelName)
			if err != nil {
				log.Error("OpenAI兼容模型创建失败",
					zap.String("model", modelName),
					zap.Error(err))
				continue
			}
			// 使用 "provider:model" 作为唯一的 provider name
			providerKey := constant.ProviderOpenAI + ":" + modelName
			llmScheduler.RegisterProvider(providerKey, openaiModel, []string{modelName})
			log.Info("OpenAI兼容模型注册成功", zap.String("model", modelName))
		}
	} else {
		log.Warn("OpenAI兼容 Provider未配置或API密钥为空")
	}

	// ==================== 初始化深度研究服务 ====================
	var researchService *service.ResearchService
	var researchTools []eino.InvokableTool // 提前声明，供 MCP API 复用

	// 优先使用智谱AI模型进行深度研究（glm-4.7 或 glm-4.5-air）
	if zhipuCfg, ok := cfg.LLM.Providers["zhipu"]; ok && zhipuCfg.APIKey != "" {
		log.Info("初始化深度研究服务...")

		// 选择研究用的模型（优先使用 glm-4.7）
		researchModelName := "glm-4.7"
		if len(zhipuCfg.Models) > 0 {
			// 检查是否有 glm-4.7
			hasGLM47 := false
			for _, m := range zhipuCfg.Models {
				if m == "glm-4.7" {
					hasGLM47 = true
					break
				}
			}
			if !hasGLM47 {
				researchModelName = zhipuCfg.Models[0] // 使用第一个可用模型
			}
		}

		// 创建研究用的ChatModel
		researchChatModel, err := createZhipuModelWithJWT(zhipuCfg.APIKey, zhipuCfg.BaseURL, researchModelName)
		if err != nil {
			log.Error("创建研究ChatModel失败", zap.Error(err))
		} else if researchChatModel != nil {
			// 创建研究工具（含 Web Search Prime MCP）
			toolsConfig := eino.ToolsConfig{
				WebSearchAPIKey:    zhipuCfg.APIKey,
				ArxivMaxResults:    10,
				WikipediaLanguage:  "zh",
				Timeout:            60 * time.Second,
				EnableReliability:  true,
				EnableZRead:        true,
				EnableWebReader:    true,
				EnableSearchPrime:  true, // 启用 Web Search Prime MCP（增强搜索）
			}
			researchTools = eino.CreateResearchTools(toolsConfig)
			log.Info("研究工具创建成功", zap.Int("tools_count", len(researchTools)))

			// 创建研究Agent配置
			agentConfig := eino.AgentConfig{
				MaxIterations: cfg.Research.MaxIterations,
				Timeout:       time.Duration(cfg.Research.SessionTimeout) * time.Second,
			}
			if agentConfig.MaxIterations == 0 {
				agentConfig.MaxIterations = 10
			}
			if agentConfig.Timeout == 0 {
				agentConfig.Timeout = 30 * time.Minute
			}

			// 创建单Agent（用于quick模式）
			researchAgent := eino.NewResearchAgent(researchChatModel, researchTools, agentConfig)
			log.Info("研究Agent创建成功",
				zap.String("model", researchModelName),
				zap.Int("max_iterations", agentConfig.MaxIterations))

			// 创建并行编排器（用于deep/comprehensive模式，最多3个Agent并行）
			parallelOrchestrator := eino.NewParallelOrchestrator(researchChatModel, researchTools, agentConfig)
			log.Info("并行研究编排器创建成功（最多3个Agent并行）")

			// 创建EventStream用于SSE推送
			eventStream := service.NewEventStream(100)

			// 创建缓存
			researchCache := cache.NewMemoryCache(1000)

			// 创建ResearchRepository
			var researchRepo repository.ResearchRepository
			if db != nil {
				researchRepo = repository.NewResearchRepository(db)
			}

			// 创建ResearchService
			researchService = service.NewResearchService(
				researchRepo,
				researchAgent,
				researchTools,
				researchCache,
				eventStream,
			)
			researchService.SetOrchestrator(parallelOrchestrator)
			log.Info("深度研究服务初始化成功（支持并行多Agent调研）")
		}
	} else {
		log.Warn("智谱AI未配置，深度研究服务将不可用")
	}

	// ==================== 初始化论文生成服务 ====================
	var paperService *service.PaperService
	var paperAPI *v1.PaperAPI

	if zhipuCfg, ok := cfg.LLM.Providers["zhipu"]; ok && zhipuCfg.APIKey != "" && db != nil {
		log.Info("初始化论文生成服务...")

		// 选择论文生成用的模型（优先使用 glm-4.7）
		paperModelName := "glm-4.7"
		if len(zhipuCfg.Models) > 0 {
			hasGLM47 := false
			for _, m := range zhipuCfg.Models {
				if m == "glm-4.7" {
					hasGLM47 = true
					break
				}
			}
			if !hasGLM47 {
				paperModelName = zhipuCfg.Models[0]
			}
		}

		paperChatModel, err := createZhipuModelWithJWT(zhipuCfg.APIKey, zhipuCfg.BaseURL, paperModelName)
		if err != nil {
			log.Error("创建论文ChatModel失败", zap.Error(err))
		} else if paperChatModel != nil {
			// 创建论文工具
			paperToolsConfig := eino.ToolsConfig{
				WebSearchAPIKey:    zhipuCfg.APIKey,
				ArxivMaxResults:    10,
				WikipediaLanguage:  "zh",
				Timeout:            60 * time.Second,
				EnableReliability:  true,
				EnableZRead:        true,
				EnableWebReader:    true,
				EnableSearchPrime:  true,
			}
			paperTools := eino.CreateResearchTools(paperToolsConfig)

			// 创建PaperAgent
			paperAgentConfig := agent.DefaultPaperAgentConfig()
			paperAgent := agent.NewPaperAgent(paperChatModel, paperTools, paperAgentConfig)
			log.Info("论文Agent创建成功", zap.String("model", paperModelName))

			// 创建PaperRepository
			paperRepo := repository.NewPaperRepository(db)

			// 创建PaperService
			paperService = service.NewPaperService(paperRepo, paperAgent, log)
			paperAPI = v1.NewPaperAPI(paperDAO, paperService)
			log.Info("论文生成服务初始化成功")
		}
	} else {
		log.Warn("论文生成服务未初始化（需要智谱AI配置和数据库连接）")
	}

	// 初始化API层
	userAPI := v1.NewUserAPIEnhancedWithPreferences(jwtManager, nil, userDAO, userPreferencesDAO)
	chatAPI := v1.NewChatAPIFull(chatDAO, userPreferencesDAO, membershipDAO, modelConfigDAO, llmScheduler)
	researchAPI := v1.NewResearchAPI(researchDAO, researchService)
	llmAPI := v1.NewLLMAPIWithDAO(llmScheduler, modelConfigDAO)
	healthAPI := v1.NewHealthAPI(db, nil)
	mcpAPI := v1.NewMCPAPIWithTools(researchTools, toolCallDAO)
	notificationAPI := v1.NewNotificationAPI(notificationDAO)
	adminAPI := v1.NewAdminAPIWithScheduler(userDAO, chatDAO, membershipDAO, activationCodeDAO, notificationDAO, modelConfigDAO, quotaConfigDAO, llmScheduler)
	membershipAPI := v1.NewMembershipAPI(membershipDAO, activationCodeDAO)
	aiQuestionAPI := v1.NewAIQuestionAPIFull(llmScheduler, aiQuestionDAO)

	// 初始化增强路由（包含所有API）
	router := api.NewRouterEnhancedFull(userAPI, chatAPI, researchAPI, llmAPI, healthAPI, mcpAPI, adminAPI, membershipAPI, notificationAPI, aiQuestionAPI, paperAPI)
	engine := router.SetupEnhanced()

	// 启动HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	log.Info("服务器启动成功",
		zap.String("地址", server.Addr),
		zap.String("API版本", "v1"),
		zap.Strings("支持的LLM提供商", []string{constant.ProviderDeepSeek, constant.ProviderZhipu, constant.ProviderOpenAI, constant.ProviderOllama, constant.ProviderOpenRouter}),
	)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("服务器启动失败", zap.Error(err))
		}
	}()

	<-quit
	log.Info("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("服务器关闭失败", zap.Error(err))
	}

	if sqlDB != nil {
		if err := sqlDB.Close(); err != nil {
			log.Error("数据库关闭失败", zap.Error(err))
		}
	}

	log.Info("服务器已停止")
}

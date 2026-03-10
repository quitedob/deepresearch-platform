package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/ai-research-platform/internal/infrastructure/config"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/ai-research-platform/internal/types/constant"
)

// LLMAPI LLM API
type LLMAPI struct {
	llmScheduler   *eino.LLMScheduler
	modelConfigDAO *dao.ModelConfigDAO
}

// NewLLMAPI 创建LLM API
func NewLLMAPI(scheduler *eino.LLMScheduler) *LLMAPI {
	return &LLMAPI{
		llmScheduler: scheduler,
	}
}

// NewLLMAPIWithDAO 创建带DAO的LLM API
func NewLLMAPIWithDAO(scheduler *eino.LLMScheduler, modelConfigDAO *dao.ModelConfigDAO) *LLMAPI {
	return &LLMAPI{
		llmScheduler:   scheduler,
		modelConfigDAO: modelConfigDAO,
	}
}

// ListProviders 获取LLM提供商列表（从数据库动态读取启用的配置）
func (l *LLMAPI) ListProviders(c *gin.Context) {
	providers := make([]gin.H, 0)
	modelsConfig := config.GetModelsConfig()

	// 必须有 modelConfigDAO 才能动态获取配置
	if l.modelConfigDAO == nil {
		c.JSON(http.StatusOK, gin.H{
			"providers": providers,
			"count":     0,
			"message":   "模型配置DAO未初始化",
		})
		return
	}

	// 从数据库获取启用的 providers
	enabledProviders, err := l.modelConfigDAO.GetEnabledProviders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取提供商配置失败: " + err.Error(),
		})
		return
	}

	// 从数据库获取启用的 models
	enabledModels, err := l.modelConfigDAO.GetEnabledModels(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取模型配置失败: " + err.Error(),
		})
		return
	}

	// 按 provider 分组 models
	providerModelsMap := make(map[string][]*model.ModelConfig)
	for _, m := range enabledModels {
		providerModelsMap[m.Provider] = append(providerModelsMap[m.Provider], m)
	}

	// 构建 provider 列表
	for _, providerConfig := range enabledProviders {
		providerName := providerConfig.Provider
		
		// 获取该 provider 下启用的 models
		models := providerModelsMap[providerName]
		if len(models) == 0 {
			continue // 跳过没有启用模型的 provider
		}

		// 构建 models 列表
		modelsList := make([]gin.H, 0, len(models))
		var defaultModel, deepThinkModel string
		
		for _, m := range models {
			modelInfo := gin.H{
				"name":             m.ModelName,
				"display_name":     m.DisplayName,
				"is_deep_thinking": false,
				"sort_order":       m.SortOrder,
			}
			
			// 从 YAML 配置获取额外的模型元数据
			if modelsConfig != nil {
				if modelMeta, ok := modelsConfig.Models[m.ModelName]; ok {
					modelInfo["description"] = modelMeta.Description
					modelInfo["context_length"] = modelMeta.ContextLength
					modelInfo["max_tokens"] = modelMeta.MaxTokens
					modelInfo["is_deep_thinking"] = modelMeta.IsDeepThinking
					modelInfo["capabilities"] = modelMeta.Capabilities
					
					// 确定默认模型和深度思考模型
					if modelMeta.IsDeepThinking {
						deepThinkModel = m.ModelName
					} else if defaultModel == "" {
						defaultModel = m.ModelName
					}
				}
			}
			
			modelsList = append(modelsList, modelInfo)
		}
		
		// 如果没有找到默认模型，使用第一个
		if defaultModel == "" && len(models) > 0 {
			defaultModel = models[0].ModelName
		}
		// 如果没有深度思考模型，使用第一个
		if deepThinkModel == "" && len(models) > 0 {
			deepThinkModel = models[0].ModelName
		}

		providerInfo := gin.H{
			"name":             providerName,
			"display_name":     providerConfig.DisplayName,
			"models":           modelsList,
			"default_model":    defaultModel,
			"deep_think_model": deepThinkModel,
			"status":           "available",
			"enabled":          true,
			"sort_order":       providerConfig.SortOrder,
		}

		// 设置 provider 特定属性
		// 使用集中定义的常量
		if desc, ok := constant.ProviderDescriptions[providerName]; ok {
			providerInfo["description"] = desc
		} else {
			providerInfo["description"] = "LLM Provider"
		}
		providerInfo["base_url"] = constant.ProviderBaseURL(providerName)
		providerInfo["requires_key"] = constant.ProviderRequiresAPIKey(providerName)
		if icon, ok := constant.ProviderIcons[providerName]; ok {
			providerInfo["icon"] = icon
		} else {
			providerInfo["icon"] = "🤖"
		}

		providers = append(providers, providerInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"count":     len(providers),
	})
}

// ListModels 获取所有可用模型
func (l *LLMAPI) ListModels(c *gin.Context) {
	models := make([]gin.H, 0)
	modelsConfig := config.GetModelsConfig()

	if l.llmScheduler != nil {
		registeredModels := l.llmScheduler.GetRegisteredModels()
		for modelName, providerName := range registeredModels {
			description := "LLM模型"
			
			// 从配置文件获取模型描述
			if modelsConfig != nil {
				if modelMeta, ok := modelsConfig.Models[modelName]; ok {
					description = modelMeta.Description
				}
			}
			
			models = append(models, gin.H{
				"name":        modelName,
				"provider":    providerName,
				"description": description,
				"available":   true,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"models": models, "count": len(models)})
}

// TestProvider 测试LLM提供商
func (l *LLMAPI) TestProvider(c *gin.Context) {
	var req struct {
		Provider string   `json:"provider" binding:"required"`
		Model    string   `json:"model" binding:"required"`
		Messages []string `json:"messages"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	if l.llmScheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "LLM调度器未配置"})
		return
	}

	messages := []*eino.Message{
		{Role: "user", Content: "Hello, this is a test message."},
	}

	result, err := l.llmScheduler.ExecuteWithFallback(c.Request.Context(), messages, req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":  false,
			"provider": req.Provider,
			"model":    req.Model,
			"error":    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"provider": req.Provider,
		"model":    req.Model,
		"response": result.Content,
	})
}

// GetMetrics 获取LLM指标
func (l *LLMAPI) GetMetrics(c *gin.Context) {
	if l.llmScheduler == nil {
		c.JSON(http.StatusOK, gin.H{"metrics": gin.H{}, "message": "LLM调度器未配置"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"metrics": gin.H{}})
}

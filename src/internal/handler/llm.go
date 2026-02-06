package handler

import (
	"net/http"

	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/gin-gonic/gin"
)

// LLMHandler 处理LLM相关请求
type LLMHandler struct {
	llmScheduler *eino.LLMScheduler
}

// NewLLMHandler 创建LLM处理器
func NewLLMHandler(llmScheduler *eino.LLMScheduler) *LLMHandler {
	return &LLMHandler{
		llmScheduler: llmScheduler,
	}
}

// TestProviderRequest 表示测试提供商的请求
type TestProviderRequest struct {
	Provider string   `json:"provider" binding:"required"`
	Model    string   `json:"model" binding:"required"`
	Messages []string `json:"messages" binding:"required,min=1"`
}

// ListProviders 列出所有已注册的LLM提供商
func (h *LLMHandler) ListProviders(c *gin.Context) {
	providers := h.llmScheduler.ListProviders()

	// 获取每个提供商的指标
	providersWithMetrics := make([]gin.H, 0, len(providers))
	for _, name := range providers {
		metric, exists := h.llmScheduler.GetMetrics(name)

		providerInfo := gin.H{
			"name": name,
		}

		if exists {
			successRate := h.llmScheduler.GetSuccessRate(name)
			avgLatency := h.llmScheduler.GetAverageLatency(name)

			providerInfo["metrics"] = gin.H{
				"success_count":    metric.SuccessCount,
				"failure_count":    metric.FailureCount,
				"success_rate":     successRate,
				"average_latency":  avgLatency.Milliseconds(),
				"last_success":     metric.LastSuccess,
				"last_failure":     metric.LastFailure,
			}
		}

		providersWithMetrics = append(providersWithMetrics, providerInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providersWithMetrics,
		"count":     len(providers),
	})
}

// GetProviderMetrics 获取特定提供商的指标
func (h *LLMHandler) GetProviderMetrics(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}

	metric, exists := h.llmScheduler.GetMetrics(provider)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	successRate := h.llmScheduler.GetSuccessRate(provider)
	avgLatency := h.llmScheduler.GetAverageLatency(provider)

	c.JSON(http.StatusOK, gin.H{
		"provider":         provider,
		"success_count":    metric.SuccessCount,
		"failure_count":    metric.FailureCount,
		"success_rate":     successRate,
		"average_latency":  avgLatency.Milliseconds(),
		"total_latency":    metric.TotalLatency.Milliseconds(),
		"last_success":     metric.LastSuccess,
		"last_failure":     metric.LastFailure,
	})
}

// ListModels 列出所有提供商的所有可用模型
func (h *LLMHandler) ListModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"models": []gin.H{
			// DeepSeek 模型
			{
				"name":        "deepseek-chat",
				"provider":    "deepseek",
				"description": "DeepSeek-V3.2-Exp 非思考模型",
				"available":   true,
			},
			{
				"name":        "deepseek-reasoner",
				"provider":    "deepseek",
				"description": "DeepSeek-V3.2-Exp 思考模型",
				"available":   true,
			},
			// 智谱AI 模型
			{
				"name":        "glm-4.7",
				"provider":    "zhipu",
				"description": "高智能旗舰模型",
				"available":   true,
			},
			{
				"name":        "glm-4.5-air",
				"provider":    "zhipu",
				"description": "高性价比模型",
				"available":   true,
			},
			// Ollama 本地模型
			{
				"name":        "gemma3:12b",
				"provider":    "ollama",
				"description": "Google Gemma 3 12B本地模型",
				"available":   true,
			},
			{
				"name":        "qwen3:8b",
				"provider":    "ollama",
				"description": "阿里通义千问3本地模型",
				"available":   true,
			},
			{
				"name":        "gemma3:4b",
				"provider":    "ollama",
				"description": "Google Gemma 3 4B本地模型",
				"available":   true,
			},
		},
	})
}

// TestProvider 使用简单请求测试提供商
func (h *LLMHandler) TestProvider(c *gin.Context) {
	var req TestProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 将字符串消息转换为eino消息
	messages := make([]*eino.Message, 0, len(req.Messages))
	for i, content := range req.Messages {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages = append(messages, &eino.Message{
			Role:    role,
			Content: content,
		})
	}

	// 使用备用机制执行
	response, err := h.llmScheduler.ExecuteWithFallback(c.Request.Context(), messages, req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "provider test failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"provider": req.Provider,
		"model":    req.Model,
		"response": response.Content,
	})
}

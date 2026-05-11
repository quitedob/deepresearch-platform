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
	providers := h.llmScheduler.GetRegisteredProviders()

	providersWithInfo := make([]gin.H, 0, len(providers))
	for _, name := range providers {
		providersWithInfo = append(providersWithInfo, gin.H{
			"name": name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providersWithInfo,
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

	// Check if provider exists
	providers := h.llmScheduler.GetRegisteredProviders()
	found := false
	for _, name := range providers {
		if name == provider {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"provider": provider,
	})
}

// ListModels 列出所有提供商的所有可用模型
func (h *LLMHandler) ListModels(c *gin.Context) {
	registered := h.llmScheduler.GetRegisteredModels()

	models := make([]gin.H, 0, len(registered))
	for modelName, provider := range registered {
		models = append(models, gin.H{
			"name":     modelName,
			"provider": provider,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"models": models,
		"count":  len(models),
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

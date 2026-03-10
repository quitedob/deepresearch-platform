package llm

import (
	"fmt"

	"github.com/ai-research-platform/internal/infrastructure/config"
	"github.com/ai-research-platform/internal/pkg/llm/provider"
	"github.com/ai-research-platform/internal/types/constant"
)

// ProviderFactory 提供商工厂
type ProviderFactory struct{}

// NewProviderFactory 创建提供商工厂
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProvider 创建LLM提供商
// 支持的提供商从配置文件读取
func (f *ProviderFactory) CreateProvider(providerName string, cfg ChatModelConfig) (Provider, error) {
	providerConfig := provider.ChatModelConfig{
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		Timeout:     cfg.Timeout,
		MaxRetries:  cfg.MaxRetries,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
		TopP:        cfg.TopP,
	}

	switch providerName {
	case "deepseek":
		return provider.NewDeepSeekProvider(providerConfig)
	case "zhipu":
		return provider.NewZhipuProvider(providerConfig)
	case "openai":
		return provider.NewOpenAIProvider(providerConfig)
	case "ollama":
		return provider.NewOllamaProvider(providerConfig)
	case "openrouter":
		return provider.NewOpenRouterProvider(providerConfig)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// GetSupportedProviders 获取支持的提供商列表（从 models.yaml 读取）
// 不做 fallback，只返回配置文件中定义的 Provider
func (f *ProviderFactory) GetSupportedProviders() []string {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return []string{}
	}
	return modelsConfig.GetProviderNames()
}

// IsProviderSupported 检查提供商是否支持
// 不做 fallback，只检查 models.yaml 中定义的 Provider
func (f *ProviderFactory) IsProviderSupported(providerName string) bool {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return false
	}
	return modelsConfig.IsProviderSupported(providerName)
}

// GetProviderInfo 获取提供商信息（从配置文件读取）
func (f *ProviderFactory) GetProviderInfo(providerName string) (*ProviderInfo, error) {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return nil, fmt.Errorf("models config not loaded")
	}

	providerMeta, ok := modelsConfig.Providers[providerName]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	// 获取该提供商的所有模型
	providerModels := modelsConfig.GetModelsByProvider(providerName)
	modelInfos := make([]ModelInfo, 0, len(providerModels))
	for modelName, modelMeta := range providerModels {
		modelInfos = append(modelInfos, ModelInfo{
			Name:        modelName,
			DisplayName: modelMeta.DisplayName,
			Description: modelMeta.Description,
			MaxTokens:   modelMeta.MaxTokens,
			ContextSize: modelMeta.ContextLength,
		})
	}

	// 根据提供商设置基础URL和其他属性
	baseURL := constant.ProviderBaseURL(providerName)
	requiresAPIKey := constant.ProviderRequiresAPIKey(providerName)

	return &ProviderInfo{
		Name:           providerName,
		DisplayName:    providerMeta.DisplayName,
		Description:    fmt.Sprintf("%s LLM Provider", providerMeta.DisplayName),
		BaseURL:        baseURL,
		Models:         modelInfos,
		RequiresAPIKey: requiresAPIKey,
		SupportsStream: true,
		SupportsTools:  true,
	}, nil
}

// ProviderInfo 提供商信息
type ProviderInfo struct {
	Name           string      `json:"name"`
	DisplayName    string      `json:"display_name"`
	Description    string      `json:"description"`
	BaseURL        string      `json:"base_url"`
	Models         []ModelInfo `json:"models"`
	RequiresAPIKey bool        `json:"requires_api_key"`
	SupportsStream bool        `json:"supports_stream"`
	SupportsTools  bool        `json:"supports_tools"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	MaxTokens   int    `json:"max_tokens"`
	ContextSize int    `json:"context_size"`
}

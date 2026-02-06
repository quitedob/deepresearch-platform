// Package model 提供 ChatModel Provider 封装
package model

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/ai-research-platform/internal/infrastructure/config"
)

// Config ChatModel 配置
type Config struct {
	Provider    string  `json:"provider"`    // deepseek, zhipu, ollama
	Model       string  `json:"model"`       // 模型名称
	APIKey      string  `json:"api_key"`     // API Key
	BaseURL     string  `json:"base_url"`    // API Base URL
	Temperature float32 `json:"temperature"` // 温度
	MaxTokens   int     `json:"max_tokens"`  // 最大token
}

// ChatModel Eino 官方 model.ChatModel 的别名
type ChatModel = model.ChatModel

// Message Eino 官方 schema.Message 的别名
type Message = schema.Message

// NewMessage 创建消息
func NewMessage(role, content string) *Message {
	return &Message{
		Role:    schema.RoleType(role),
		Content: content,
	}
}

// NewUserMessage 创建用户消息
func NewUserMessage(content string) *Message {
	return &Message{
		Role:    schema.User,
		Content: content,
	}
}

// NewAssistantMessage 创建助手消息
func NewAssistantMessage(content string) *Message {
	return &Message{
		Role:    schema.Assistant,
		Content: content,
	}
}

// NewSystemMessage 创建系统消息
func NewSystemMessage(content string) *Message {
	return &Message{
		Role:    schema.System,
		Content: content,
	}
}

// Provider 封装不同LLM Provider
type Provider struct {
	config    Config
	chatModel ChatModel
}

// NewProvider 创建 Provider（需要传入已初始化的 ChatModel）
func NewProvider(config Config, chatModel ChatModel) *Provider {
	return &Provider{
		config:    config,
		chatModel: chatModel,
	}
}

// Generate 生成响应
func (p *Provider) Generate(ctx context.Context, messages []*Message, opts ...model.Option) (*Message, error) {
	if p.chatModel == nil {
		return nil, fmt.Errorf("chatModel is nil")
	}
	return p.chatModel.Generate(ctx, messages, opts...)
}

// Stream 流式生成响应
func (p *Provider) Stream(ctx context.Context, messages []*Message, opts ...model.Option) (*schema.StreamReader[*Message], error) {
	if p.chatModel == nil {
		return nil, fmt.Errorf("chatModel is nil")
	}
	return p.chatModel.Stream(ctx, messages, opts...)
}

// GetConfig 获取配置
func (p *Provider) GetConfig() Config {
	return p.config
}

// SupportedProviders 返回支持的 Provider 列表（从 models.yaml 读取）
// 不做 fallback，只返回配置文件中定义的 Provider
func SupportedProviders() []string {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return []string{}
	}
	return modelsConfig.GetProviderNames()
}

// ProviderModels 返回 Provider 支持的模型列表（从 models.yaml 读取）
// 不做 fallback，没有配置就返回空
func ProviderModels(providerType string) []string {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return []string{}
	}
	return modelsConfig.GetEnabledModelsByProvider(providerType)
}

// DefaultConfig 返回默认配置
// 注意：API Key 需要从外部传入，这里不做 fallback
func DefaultConfig(providerType string) Config {
	modelsConfig := config.GetModelsConfig()
	
	// 获取默认模型（第一个启用的）
	defaultModel := ""
	if modelsConfig != nil {
		models := modelsConfig.GetEnabledModelsByProvider(providerType)
		if len(models) > 0 {
			defaultModel = models[0]
		}
	}

	// 根据 provider 设置默认 BaseURL
	var baseURL string
	switch providerType {
	case "deepseek":
		baseURL = "https://api.deepseek.com/v1"
	case "zhipu":
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	case "openai":
		baseURL = "https://api.z.ai/api/coding/paas/v4"
	case "ollama":
		baseURL = "http://localhost:11434/v1"
	case "openrouter":
		baseURL = "https://openrouter.ai/api/v1"
	}

	return Config{
		Provider:    providerType,
		Model:       defaultModel,
		BaseURL:     baseURL,
		Temperature: 0.7,
		MaxTokens:   4096,
		// APIKey 需要从外部传入
	}
}

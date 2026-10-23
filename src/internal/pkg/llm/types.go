package llm

import (
	"context"
	"github.com/ai-research-platform/internal/pkg/llm/provider"
)

// Message 消息结构
type Message = provider.Message

// StreamChunk 流式响应块
type StreamChunk = provider.StreamChunk

// ChatModelConfig 聊天模型配置
type ChatModelConfig = provider.ChatModelConfig

// Provider preserves the legacy factory contract. New integrations use Eino.
type Provider interface {
	// Generate 生成文本
	Generate(ctx context.Context, messages []Message) (*Message, error)

	// StreamGenerate 流式生成文本
	StreamGenerate(ctx context.Context, messages []Message) (<-chan StreamChunk, error)

	// GetCapabilities 获取模型能力
	GetCapabilities() *Capabilities

	// GetName 获取提供商名称
	GetName() string

	// GetModel 获取模型名称
	GetModel() string
}

// Capabilities 模型能力
type Capabilities = provider.Capabilities

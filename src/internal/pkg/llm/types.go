package llm

import (
    "context"
    "time"
)

// Message 消息结构
type Message struct {
    Role    string `json:"role"`    // user, assistant, system
    Content string `json:"content"`
    // 可选字段
    TokenCount int                    `json:"token_count,omitempty"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
    Content string `json:"content"`
    Done    bool   `json:"done"`
    Error   error  `json:"error,omitempty"`
}

// ChatModelConfig 聊天模型配置
type ChatModelConfig struct {
    APIKey  string `json:"api_key"`
    BaseURL string `json:"base_url"`
    Model   string `json:"model"`
    // 可选配置
    Timeout    int           `json:"timeout"`
    MaxRetries int           `json:"max_retries"`
    Temperature float32      `json:"temperature,omitempty"`
    MaxTokens   int          `json:"max_tokens,omitempty"`
    TopP        float32      `json:"top_p,omitempty"`
}

// Provider LLM提供商接口
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
type Capabilities struct {
    MaxTokens       int      `json:"max_tokens"`
    SupportsStream  bool     `json:"supports_stream"`
    SupportsTools   bool     `json:"supports_tools"`
    SupportedTools  []string `json:"supported_tools,omitempty"`
    MaxContextSize  int      `json:"max_context_size,omitempty"`
    InputPricePer1K float64  `json:"input_price_per_1k,omitempty"`
    OutputPricePer1K float64 `json:"output_price_per_1k,omitempty"`
}

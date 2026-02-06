package provider

import (
	"fmt"
	"strings"
)

// APIErrorCode API错误码
type APIErrorCode string

const (
	ErrCodeAuthFailed    APIErrorCode = "ERR_API_AUTH_FAILED"
	ErrCodeRateLimit     APIErrorCode = "ERR_API_RATE_LIMIT"
	ErrCodeQuotaExceeded APIErrorCode = "ERR_API_QUOTA_EXCEEDED"
	ErrCodeBadRequest    APIErrorCode = "ERR_API_BAD_REQUEST"
	ErrCodeServerError   APIErrorCode = "ERR_API_SERVER_ERROR"
	ErrCodeUnavailable   APIErrorCode = "ERR_API_UNAVAILABLE"
	ErrCodeUnknown       APIErrorCode = "ERR_API_UNKNOWN"
)

// APIError 结构化的API错误，包含用户友好的提示信息
type APIError struct {
	Code        APIErrorCode
	UserMessage string // 面向用户的友好提示（中文）
	RawMessage  string // 原始错误信息（用于日志）
	StatusCode  int
	Provider    string
}

func (e *APIError) Error() string {
	return e.UserMessage
}

// ClassifyAPIError 根据HTTP状态码和响应体分类API错误，返回用户友好的错误信息
func ClassifyAPIError(providerName string, statusCode int, responseBody string) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Provider:   providerName,
		RawMessage: responseBody,
	}

	bodyLower := strings.ToLower(responseBody)

	switch {
	case statusCode == 401 || strings.Contains(bodyLower, "authentication") || strings.Contains(bodyLower, "unauthorized") || strings.Contains(bodyLower, "invalid api key") || strings.Contains(bodyLower, "invalid_request_error"):
		apiErr.Code = ErrCodeAuthFailed
		apiErr.UserMessage = fmt.Sprintf("%s API 密钥无效或已过期，请联系管理员更新配置", providerName)

	case statusCode == 429 || strings.Contains(bodyLower, "rate limit") || strings.Contains(bodyLower, "too many requests"):
		apiErr.Code = ErrCodeRateLimit
		apiErr.UserMessage = fmt.Sprintf("%s API 请求过于频繁，请稍后再试", providerName)

	case statusCode == 402 || strings.Contains(bodyLower, "insufficient") || strings.Contains(bodyLower, "quota") || strings.Contains(bodyLower, "balance"):
		apiErr.Code = ErrCodeQuotaExceeded
		apiErr.UserMessage = fmt.Sprintf("%s API 额度不足，请联系管理员充值", providerName)

	case statusCode == 400:
		apiErr.Code = ErrCodeBadRequest
		apiErr.UserMessage = fmt.Sprintf("%s 请求参数错误，请稍后重试", providerName)

	case statusCode >= 500 && statusCode < 600:
		apiErr.Code = ErrCodeServerError
		apiErr.UserMessage = fmt.Sprintf("%s 服务暂时不可用，请稍后重试", providerName)

	case statusCode == 503 || statusCode == 502 || statusCode == 504:
		apiErr.Code = ErrCodeUnavailable
		apiErr.UserMessage = fmt.Sprintf("%s 服务暂时不可用，请稍后重试", providerName)

	default:
		apiErr.Code = ErrCodeUnknown
		apiErr.UserMessage = fmt.Sprintf("%s 服务调用失败 (HTTP %d)，请稍后重试", providerName, statusCode)
	}

	return apiErr
}

// Message 消息结构
type Message struct {
    Role       string                 `json:"role"`    // user, assistant, system
    Content    string                 `json:"content"`
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
    APIKey      string  `json:"api_key"`
    BaseURL     string  `json:"base_url"`
    Model       string  `json:"model"`
    Timeout     int     `json:"timeout"`
    MaxRetries  int     `json:"max_retries"`
    Temperature float32 `json:"temperature,omitempty"`
    MaxTokens   int     `json:"max_tokens,omitempty"`
    TopP        float32 `json:"top_p,omitempty"`
}

// Capabilities 模型能力
type Capabilities struct {
    MaxTokens        int      `json:"max_tokens"`
    SupportsStream   bool     `json:"supports_stream"`
    SupportsTools    bool     `json:"supports_tools"`
    SupportedTools   []string `json:"supported_tools,omitempty"`
    MaxContextSize   int      `json:"max_context_size,omitempty"`
    InputPricePer1K  float64  `json:"input_price_per_1k,omitempty"`
    OutputPricePer1K float64  `json:"output_price_per_1k,omitempty"`
}

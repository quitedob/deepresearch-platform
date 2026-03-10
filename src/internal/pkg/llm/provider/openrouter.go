package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ai-research-platform/internal/types/constant"
)

// OpenRouterProvider OpenRouter LLM提供商
// 支持 400+ AI 模型，使用 OpenAI 兼容 API
type OpenRouterProvider struct {
	config ChatModelConfig
	client *http.Client
}

// NewOpenRouterProvider 创建 OpenRouter 提供商
func NewOpenRouterProvider(config ChatModelConfig) (*OpenRouterProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = constant.BaseURLOpenRouter
	}
	if config.Model == "" {
		config.Model = "openai/gpt-4o" // 默认模型
	}

	timeout := 60 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	return &OpenRouterProvider{
		config: config,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}, nil
}

// openRouterRequest OpenRouter 请求结构（OpenAI 兼容）
type openRouterRequest struct {
	Model       string                   `json:"model"`
	Messages    []map[string]interface{} `json:"messages"`
	Temperature float32                  `json:"temperature,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	TopP        float32                  `json:"top_p,omitempty"`
	Stream      bool                     `json:"stream"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
}

// openRouterResponse OpenRouter 响应结构
type openRouterResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Generate 生成文本
func (p *OpenRouterProvider) Generate(ctx context.Context, messages []Message) (*Message, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := openRouterRequest{
		Model:       p.config.Model,
		Messages:    reqMessages,
		Temperature: p.config.Temperature,
		MaxTokens:   p.config.MaxTokens,
		TopP:        p.config.TopP,
		Stream:      false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("HTTP-Referer", constant.OpenRouterReferer)
	req.Header.Set("X-Title", constant.OpenRouterTitle)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, ClassifyAPIError("OpenRouter", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openRouterResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s (code: %s)", apiResp.Error.Message, apiResp.Error.Code)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &Message{
		Role:       apiResp.Choices[0].Message.Role,
		Content:    apiResp.Choices[0].Message.Content,
		TokenCount: apiResp.Usage.TotalTokens,
	}, nil
}


// StreamGenerate 流式生成文本
func (p *OpenRouterProvider) StreamGenerate(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := openRouterRequest{
		Model:       p.config.Model,
		Messages:    reqMessages,
		Temperature: p.config.Temperature,
		MaxTokens:   p.config.MaxTokens,
		TopP:        p.config.TopP,
		Stream:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("HTTP-Referer", constant.OpenRouterReferer)
	req.Header.Set("X-Title", constant.OpenRouterTitle)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, ClassifyAPIError("OpenRouter", resp.StatusCode, string(bodyBytes))
	}

	chunks := make(chan StreamChunk, 100)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					chunks <- StreamChunk{Error: err}
				}
				chunks <- StreamChunk{Done: true}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				chunks <- StreamChunk{Done: true}
				return
			}

			var streamResp openRouterResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
				chunks <- StreamChunk{
					Content: streamResp.Choices[0].Delta.Content,
					Done:    false,
				}
			}
		}
	}()

	return chunks, nil
}

// GetCapabilities 获取模型能力
func (p *OpenRouterProvider) GetCapabilities() *Capabilities {
	// OpenRouter 支持多种模型，返回通用能力
	return &Capabilities{
		MaxTokens:      128000,
		MaxContextSize: 200000,
		SupportsStream: true,
		SupportsTools:  true,
		SupportedTools: []string{"function_calling"},
	}
}

// GetName 获取提供商名称
func (p *OpenRouterProvider) GetName() string {
	return "openrouter"
}

// GetModel 获取模型名称
func (p *OpenRouterProvider) GetModel() string {
	return p.config.Model
}

// GetSupportedModels 获取支持的模型列表（常用模型）
func (p *OpenRouterProvider) GetSupportedModels() []string {
	return []string{
		// OpenAI
		"openai/gpt-4o",
		"openai/gpt-4o-mini",
		"openai/gpt-4-turbo",
		"openai/o1",
		"openai/o1-mini",
		// Anthropic
		"anthropic/claude-3.5-sonnet",
		"anthropic/claude-3-opus",
		"anthropic/claude-3-haiku",
		// Google
		"google/gemini-2.0-flash-exp",
		"google/gemini-pro-1.5",
		// Meta
		"meta-llama/llama-3.3-70b-instruct",
		"meta-llama/llama-3.1-405b-instruct",
		// DeepSeek
		"deepseek/deepseek-chat",
		"deepseek/deepseek-reasoner",
		// Mistral
		"mistralai/mistral-large",
		"mistralai/mixtral-8x22b-instruct",
	}
}

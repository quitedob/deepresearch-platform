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
)

// OpenAIProvider 通用OpenAI兼容API提供商
// 支持所有兼容OpenAI Chat Completions API的服务，包括：
// - GLM Coding Plan (https://api.z.ai/api/coding/paas/v4)
// - 智谱AI通用API (https://open.bigmodel.cn/api/paas/v4)
// - 其他OpenAI兼容服务
type OpenAIProvider struct {
	config ChatModelConfig
	client *http.Client
}

// NewOpenAIProvider 创建OpenAI兼容提供商
func NewOpenAIProvider(config ChatModelConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI-compatible API key is required")
	}
	if config.BaseURL == "" {
		return nil, fmt.Errorf("OpenAI-compatible base URL is required")
	}
	if config.Model == "" {
		config.Model = "glm-4.5-air"
	}

	timeout := 60 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	return &OpenAIProvider{
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

// openaiRequest OpenAI兼容请求结构
type openaiRequest struct {
	Model       string                   `json:"model"`
	Messages    []map[string]interface{} `json:"messages"`
	Temperature float32                  `json:"temperature,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	TopP        float32                  `json:"top_p,omitempty"`
	Stream      bool                     `json:"stream"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
}

// openaiResponse OpenAI兼容响应结构
type openaiResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
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
}

// Generate 生成文本
func (p *OpenAIProvider) Generate(ctx context.Context, messages []Message) (*Message, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := openaiRequest{
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

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, ClassifyAPIError("OpenAI-Compatible", resp.StatusCode, string(bodyBytes))
	}

	var apiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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
func (p *OpenAIProvider) StreamGenerate(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := openaiRequest{
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
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, ClassifyAPIError("OpenAI-Compatible", resp.StatusCode, string(bodyBytes))
	}

	chunks := make(chan StreamChunk, 100)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			// 检查上下文是否已取消，及时退出
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

			var streamResp openaiResponse
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
func (p *OpenAIProvider) GetCapabilities() *Capabilities {
	caps := &Capabilities{
		SupportsStream: true,
		SupportsTools:  true,
		SupportedTools: []string{"function_calling"},
	}

	switch p.config.Model {
	case "glm-4.7", "GLM-4.7":
		caps.MaxTokens = 128000
		caps.MaxContextSize = 200000
		caps.SupportedTools = []string{"function_calling", "web_search"}
	case "glm-4.5-air", "GLM-4.5-Air":
		caps.MaxTokens = 96000
		caps.MaxContextSize = 128000
	default:
		caps.MaxTokens = 4096
		caps.MaxContextSize = 128000
	}

	return caps
}

// GetName 获取提供商名称
func (p *OpenAIProvider) GetName() string {
	return "openai"
}

// GetModel 获取模型名称
func (p *OpenAIProvider) GetModel() string {
	return p.config.Model
}

// GetSupportedModels 获取支持的模型列表
func (p *OpenAIProvider) GetSupportedModels() []string {
	return []string{
		"glm-4.7",
		"glm-4.5-air",
	}
}

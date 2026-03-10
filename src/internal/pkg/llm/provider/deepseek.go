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

// DeepSeekProvider DeepSeek LLM提供商
// 支持模型: deepseek-chat (DeepSeek-V3.2-Exp非思考模型), deepseek-reasoner (DeepSeek-V3.2-Exp思考模型)
type DeepSeekProvider struct {
	config ChatModelConfig
	client *http.Client
}

// NewDeepSeekProvider 创建DeepSeek提供商
func NewDeepSeekProvider(config ChatModelConfig) (*DeepSeekProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = constant.BaseURLDeepSeek
	}
	if config.Model == "" {
		config.Model = "deepseek-chat"
	}

	timeout := 60 * time.Second
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	return &DeepSeekProvider{
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

// deepseekRequest DeepSeek请求结构
type deepseekRequest struct {
	Model          string                   `json:"model"`
	Messages       []map[string]interface{} `json:"messages"`
	Temperature    float32                  `json:"temperature,omitempty"`
	MaxTokens      int                      `json:"max_tokens,omitempty"`
	TopP           float32                  `json:"top_p,omitempty"`
	Stream         bool                     `json:"stream"`
	ResponseFormat *responseFormat          `json:"response_format,omitempty"`
	Tools          []map[string]interface{} `json:"tools,omitempty"`
	Stop           []string                 `json:"stop,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"` // "json_object" 或 JSON schema
}

// deepseekResponse DeepSeek响应结构
type deepseekResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"` // deepseek-reasoner 思维链内容
			ToolCalls        []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls,omitempty"`
		} `json:"message"`
		Delta struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content,omitempty"`
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
func (p *DeepSeekProvider) Generate(ctx context.Context, messages []Message) (*Message, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := deepseekRequest{
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

	// 根据模型选择endpoint
	endpoint := "/chat/completions"
	if p.config.Model == "deepseek-reasoner" {
		endpoint = "/chat/completions" // reasoner也使用相同endpoint
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+endpoint, bytes.NewReader(body))
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
		return nil, ClassifyAPIError("DeepSeek", resp.StatusCode, string(bodyBytes))
	}

	var apiResp deepseekResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	metadata := map[string]interface{}{
		"model":        apiResp.Model,
		"total_tokens": apiResp.Usage.TotalTokens,
	}

	// 如果是reasoner模型，包含思维链内容
	if apiResp.Choices[0].Message.ReasoningContent != "" {
		metadata["reasoning_content"] = apiResp.Choices[0].Message.ReasoningContent
	}

	return &Message{
		Role:       apiResp.Choices[0].Message.Role,
		Content:    apiResp.Choices[0].Message.Content,
		TokenCount: apiResp.Usage.TotalTokens,
		Metadata:   metadata,
	}, nil
}

// StreamGenerate 流式生成文本
func (p *DeepSeekProvider) StreamGenerate(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := deepseekRequest{
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
		return nil, ClassifyAPIError("DeepSeek", resp.StatusCode, string(bodyBytes))
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
			// 处理keep-alive注释
			if line == "" || line == ": keep-alive" {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				chunks <- StreamChunk{Done: true}
				return
			}

			var streamResp deepseekResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				delta := streamResp.Choices[0].Delta.Content
				if delta != "" {
					chunks <- StreamChunk{
						Content: delta,
						Done:    false,
					}
				}
				// 处理reasoner模型的思维链流式输出
				if streamResp.Choices[0].Delta.ReasoningContent != "" {
					chunks <- StreamChunk{
						Content: streamResp.Choices[0].Delta.ReasoningContent,
						Done:    false,
					}
				}
			}
		}
	}()

	return chunks, nil
}

// GenerateJSON 生成JSON格式输出
func (p *DeepSeekProvider) GenerateJSON(ctx context.Context, messages []Message) (*Message, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := deepseekRequest{
		Model:       p.config.Model,
		Messages:    reqMessages,
		Temperature: p.config.Temperature,
		MaxTokens:   p.config.MaxTokens,
		TopP:        p.config.TopP,
		Stream:      false,
		ResponseFormat: &responseFormat{
			Type: "json_object",
		},
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
		return nil, ClassifyAPIError("DeepSeek", resp.StatusCode, string(bodyBytes))
	}

	var apiResp deepseekResponse
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

// GenerateWithPrefixContinuation 对话前缀续写 (Beta功能)
func (p *DeepSeekProvider) GenerateWithPrefixContinuation(ctx context.Context, messages []Message, assistantPrefix string) (*Message, error) {
	reqMessages := make([]map[string]interface{}, len(messages)+1)
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	// 添加assistant前缀消息
	reqMessages[len(messages)] = map[string]interface{}{
		"role":    "assistant",
		"content": assistantPrefix,
		"prefix":  true,
	}

	reqBody := deepseekRequest{
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

	// 使用beta endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", constant.BaseURLDeepSeekBeta+"/chat/completions", bytes.NewReader(body))
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
		return nil, ClassifyAPIError("DeepSeek", resp.StatusCode, string(bodyBytes))
	}

	var apiResp deepseekResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &Message{
		Role:       apiResp.Choices[0].Message.Role,
		Content:    assistantPrefix + apiResp.Choices[0].Message.Content,
		TokenCount: apiResp.Usage.TotalTokens,
	}, nil
}

// GetCapabilities 获取模型能力
func (p *DeepSeekProvider) GetCapabilities() *Capabilities {
	caps := &Capabilities{
		SupportsStream: true,
		SupportsTools:  true,
		SupportedTools: []string{"function_calling", "json_output"},
	}

	switch p.config.Model {
	case "deepseek-chat":
		caps.MaxTokens = 8000
		caps.MaxContextSize = 128000
	case "deepseek-reasoner":
		caps.MaxTokens = 64000
		caps.MaxContextSize = 128000
		caps.SupportsTools = false // reasoner不支持function calling
		caps.SupportedTools = []string{"json_output", "reasoning"}
	default:
		caps.MaxTokens = 8000
		caps.MaxContextSize = 128000
	}

	return caps
}

// GetName 获取提供商名称
func (p *DeepSeekProvider) GetName() string {
	return "deepseek"
}

// GetModel 获取模型名称
func (p *DeepSeekProvider) GetModel() string {
	return p.config.Model
}

// GetSupportedModels 获取支持的模型列表
func (p *DeepSeekProvider) GetSupportedModels() []string {
	return []string{
		"deepseek-chat",     // DeepSeek-V3.2-Exp 非思考模型
		"deepseek-reasoner", // DeepSeek-V3.2-Exp 思考模型
	}
}

// GetRecommendedTemperature 获取推荐的temperature设置
func (p *DeepSeekProvider) GetRecommendedTemperature(useCase string) float32 {
	switch useCase {
	case "code", "math":
		return 0.0
	case "data_extraction", "analysis":
		return 1.0
	case "conversation", "translation":
		return 1.3
	case "creative", "poetry":
		return 1.5
	default:
		return 1.0
	}
}

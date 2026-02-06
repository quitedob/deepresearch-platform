package provider

// ============================================================
// 注意：此文件暂时不使用
// GLM模型已迁移至 openai.go，通过 OpenAI 兼容接口调用
// 使用 GLM Coding Plan 端点: https://api.z.ai/api/coding/paas/v4
// 保留此文件作为自研智谱JWT认证方式的参考实现
// ============================================================

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ZhipuProvider 智谱AI LLM提供商（自研实现，暂时不使用）
// 只支持模型: glm-4.7, glm-4.5-air
type ZhipuProvider struct {
	config ChatModelConfig
	client *http.Client
}

// NewZhipuProvider 创建智谱AI提供商
func NewZhipuProvider(config ChatModelConfig) (*ZhipuProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Zhipu AI API key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://open.bigmodel.cn"
	}
	if config.Model == "" {
		config.Model = "glm-4.5-air" // 默认使用高性价比模型
	}

	return &ZhipuProvider{
		config: config,
		client: &http.Client{},
	}, nil
}

// zhipuRequest 智谱AI请求结构
type zhipuRequest struct {
	Model       string                   `json:"model"`
	Messages    []map[string]interface{} `json:"messages"`
	Temperature float32                  `json:"temperature,omitempty"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	TopP        float32                  `json:"top_p,omitempty"`
	Stream      bool                     `json:"stream"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
}

// zhipuResponse 智谱AI响应结构
type zhipuResponse struct {
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
}

// Generate 生成文本
func (p *ZhipuProvider) Generate(ctx context.Context, messages []Message) (*Message, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := zhipuRequest{
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

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/api/paas/v4/chat/completions", bytes.NewReader(body))
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
		return nil, ClassifyAPIError("智谱AI", resp.StatusCode, string(bodyBytes))
	}

	var apiResp zhipuResponse
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
func (p *ZhipuProvider) StreamGenerate(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := zhipuRequest{
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

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/api/paas/v4/chat/completions", bytes.NewReader(body))
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
		return nil, ClassifyAPIError("智谱AI", resp.StatusCode, string(bodyBytes))
	}

	chunks := make(chan StreamChunk, 100)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
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

			var streamResp zhipuResponse
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
func (p *ZhipuProvider) GetCapabilities() *Capabilities {
	// 根据模型返回不同的能力
	caps := &Capabilities{
		SupportsStream: true,
		SupportsTools:  true,
		SupportedTools: []string{"function_calling", "web_search"},
	}

	switch p.config.Model {
	case "glm-4.7":
		caps.MaxTokens = 128000
		caps.MaxContextSize = 200000
	case "glm-4.5-air":
		caps.MaxTokens = 96000
		caps.MaxContextSize = 128000
		caps.SupportedTools = []string{"vision", "reasoning"}
	default:
		caps.MaxTokens = 16000
		caps.MaxContextSize = 128000
	}

	return caps
}

// GetName 获取提供商名称
func (p *ZhipuProvider) GetName() string {
	return "zhipu"
}

// GetModel 获取模型名称
func (p *ZhipuProvider) GetModel() string {
	return p.config.Model
}

// GetSupportedModels 获取支持的模型列表
func (p *ZhipuProvider) GetSupportedModels() []string {
	return []string{
		"glm-4.7",           // 高智能旗舰模型
		"glm-4.5-air",       // 高性价比模型
	}
}

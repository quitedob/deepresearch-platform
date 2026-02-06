package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaProvider Ollama本地LLM提供商
// 支持本地部署的各种开源模型: llama2, gemma3, mistral, qwen等
type OllamaProvider struct {
	config ChatModelConfig
	client *http.Client
}

// NewOllamaProvider 创建Ollama提供商
func NewOllamaProvider(config ChatModelConfig) (*OllamaProvider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}
	if config.Model == "" {
		config.Model = "gemma3:4b"
	}

	return &OllamaProvider{
		config: config,
		client: &http.Client{},
	}, nil
}

// ollamaRequest Ollama请求结构
type ollamaRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages,omitempty"`
	Prompt   string                   `json:"prompt,omitempty"`
	System   string                   `json:"system,omitempty"`
	Stream   bool                     `json:"stream"`
	Options  map[string]interface{}   `json:"options,omitempty"`
	Format   interface{}              `json:"format,omitempty"` // 支持"json"或JSON schema
	Raw      bool                     `json:"raw,omitempty"`
	Images   []string                 `json:"images,omitempty"` // base64编码的图片
}

// ollamaResponse Ollama响应结构
type ollamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string   `json:"role"`
		Content string   `json:"content"`
		Images  []string `json:"images,omitempty"`
	} `json:"message"`
	Response         string `json:"response,omitempty"` // /api/generate 使用
	Done             bool   `json:"done"`
	DoneReason       string `json:"done_reason,omitempty"`
	Context          []int  `json:"context,omitempty"`
	TotalDuration    int64  `json:"total_duration,omitempty"`
	LoadDuration     int64  `json:"load_duration,omitempty"`
	PromptEvalCount  int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount        int    `json:"eval_count,omitempty"`
	EvalDuration     int64  `json:"eval_duration,omitempty"`
}

// Generate 生成文本 (使用 /api/chat 接口)
func (p *OllamaProvider) Generate(ctx context.Context, messages []Message) (*Message, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	options := make(map[string]interface{})
	if p.config.Temperature > 0 {
		options["temperature"] = p.config.Temperature
	}
	if p.config.MaxTokens > 0 {
		options["num_predict"] = p.config.MaxTokens
	}
	if p.config.TopP > 0 {
		options["top_p"] = p.config.TopP
	}

	reqBody := ollamaRequest{
		Model:    p.config.Model,
		Messages: reqMessages,
		Stream:   false,
		Options:  options,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, ClassifyAPIError("Ollama", resp.StatusCode, string(bodyBytes))
	}

	var apiResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tokenCount := apiResp.PromptEvalCount + apiResp.EvalCount

	return &Message{
		Role:       apiResp.Message.Role,
		Content:    apiResp.Message.Content,
		TokenCount: tokenCount,
		Metadata: map[string]interface{}{
			"model":              apiResp.Model,
			"total_duration":     apiResp.TotalDuration,
			"load_duration":      apiResp.LoadDuration,
			"eval_count":         apiResp.EvalCount,
			"prompt_eval_count":  apiResp.PromptEvalCount,
		},
	}, nil
}

// StreamGenerate 流式生成文本
func (p *OllamaProvider) StreamGenerate(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	options := make(map[string]interface{})
	if p.config.Temperature > 0 {
		options["temperature"] = p.config.Temperature
	}
	if p.config.MaxTokens > 0 {
		options["num_predict"] = p.config.MaxTokens
	}
	if p.config.TopP > 0 {
		options["top_p"] = p.config.TopP
	}

	reqBody := ollamaRequest{
		Model:    p.config.Model,
		Messages: reqMessages,
		Stream:   true,
		Options:  options,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, ClassifyAPIError("Ollama", resp.StatusCode, string(bodyBytes))
	}

	chunks := make(chan StreamChunk, 100)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					chunks <- StreamChunk{Error: err}
				}
				chunks <- StreamChunk{Done: true}
				return
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}

			var streamResp ollamaResponse
			if err := json.Unmarshal(line, &streamResp); err != nil {
				continue
			}

			if streamResp.Done {
				chunks <- StreamChunk{Done: true}
				return
			}

			if streamResp.Message.Content != "" {
				chunks <- StreamChunk{
					Content: streamResp.Message.Content,
					Done:    false,
				}
			}
		}
	}()

	return chunks, nil
}

// GenerateWithPrompt 使用 /api/generate 接口生成文本 (适用于简单提示词)
func (p *OllamaProvider) GenerateWithPrompt(ctx context.Context, prompt string, system string) (*Message, error) {
	options := make(map[string]interface{})
	if p.config.Temperature > 0 {
		options["temperature"] = p.config.Temperature
	}
	if p.config.MaxTokens > 0 {
		options["num_predict"] = p.config.MaxTokens
	}

	reqBody := ollamaRequest{
		Model:   p.config.Model,
		Prompt:  prompt,
		System:  system,
		Stream:  false,
		Options: options,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, ClassifyAPIError("Ollama", resp.StatusCode, string(bodyBytes))
	}

	var apiResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Message{
		Role:       "assistant",
		Content:    apiResp.Response,
		TokenCount: apiResp.PromptEvalCount + apiResp.EvalCount,
	}, nil
}

// GenerateJSON 生成JSON格式输出
func (p *OllamaProvider) GenerateJSON(ctx context.Context, messages []Message, schema interface{}) (*Message, error) {
	reqMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		reqMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	options := make(map[string]interface{})
	if p.config.Temperature > 0 {
		options["temperature"] = p.config.Temperature
	}
	if p.config.MaxTokens > 0 {
		options["num_predict"] = p.config.MaxTokens
	}

	reqBody := ollamaRequest{
		Model:    p.config.Model,
		Messages: reqMessages,
		Stream:   false,
		Options:  options,
		Format:   schema, // 可以是"json"或JSON schema
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, ClassifyAPIError("Ollama", resp.StatusCode, string(bodyBytes))
	}

	var apiResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Message{
		Role:       apiResp.Message.Role,
		Content:    apiResp.Message.Content,
		TokenCount: apiResp.PromptEvalCount + apiResp.EvalCount,
	}, nil
}

// ListModels 列出本地可用模型
func (p *OllamaProvider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, ClassifyAPIError("Ollama", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Models []struct {
			Name       string `json:"name"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// GetCapabilities 获取模型能力
func (p *OllamaProvider) GetCapabilities() *Capabilities {
	return &Capabilities{
		MaxTokens:      4096,
		SupportsStream: true,
		SupportsTools:  true, // Ollama支持工具调用
		SupportedTools: []string{"function_calling", "json_output", "structured_output"},
	}
}

// GetName 获取提供商名称
func (p *OllamaProvider) GetName() string {
	return "ollama"
}

// GetModel 获取模型名称
func (p *OllamaProvider) GetModel() string {
	return p.config.Model
}

// GetSupportedModels 获取常用模型列表
func (p *OllamaProvider) GetSupportedModels() []string {
	return []string{
		"gemma3:12b",
		"qwen3:8b",
		"gemma3:4b",
	}
}

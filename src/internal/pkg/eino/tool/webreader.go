// Package tool 提供 Web Reader MCP 工具实现
// Web Reader MCP 是智谱提供的网页内容抓取能力
package tool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	webReaderMCPEndpoint = "https://open.bigmodel.cn/api/mcp/web_reader/mcp"
)

// WebReaderTool Web Reader MCP 工具，提供网页内容抓取能力
type WebReaderTool struct {
	apiKey  string
	timeout time.Duration
	client  *http.Client
}

// WebReaderConfig Web Reader 配置
type WebReaderConfig struct {
	APIKey  string
	Timeout time.Duration
}

// NewWebReaderTool 创建 Web Reader 工具
func NewWebReaderTool(config WebReaderConfig) *WebReaderTool {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	return &WebReaderTool{
		apiKey:  config.APIKey,
		timeout: config.Timeout,
		client:  &http.Client{Timeout: config.Timeout},
	}
}

// Info 返回工具信息
func (t *WebReaderTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_reader",
		Desc: "Fetch and extract content from a web page URL. Returns structured data including title, content, metadata, and links.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Type:     schema.String,
				Desc:     "The URL of the web page to read",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行网页读取
func (t *WebReaderTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.URL == "" {
		return "", fmt.Errorf("url is required")
	}

	if t.apiKey == "" {
		return "", fmt.Errorf("API key not configured for Web Reader MCP")
	}

	return t.readWebPage(ctx, args.URL)
}

// readWebPage 读取网页内容
func (t *WebReaderTool) readWebPage(ctx context.Context, url string) (string, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "webReader",
			"arguments": map[string]string{
				"url": url,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webReaderMCPEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.apiKey))

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Web Reader API failed (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析 JSON-RPC 响应
	var mcpResp struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &mcpResp); err != nil {
		return "", fmt.Errorf("failed to parse MCP response: %w", err)
	}

	if mcpResp.Error != nil {
		return "", fmt.Errorf("MCP error (code %d): %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}

	// 提取文本内容
	var result strings.Builder
	for _, content := range mcpResp.Result.Content {
		if content.Type == "text" {
			result.WriteString(content.Text)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🌐 网页内容 - %s\n", url))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(result.String())
	return sb.String(), nil
}

var _ tool.InvokableTool = (*WebReaderTool)(nil)

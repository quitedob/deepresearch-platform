// Package tool 提供 Web Search Prime MCP 工具实现
// Web Search Prime 是智谱提供的增强版网络搜索能力
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
	webSearchPrimeMCPEndpoint = "https://open.bigmodel.cn/api/mcp/web_search_prime/mcp"
)

// WebSearchPrimeTool Web Search Prime MCP 工具
type WebSearchPrimeTool struct {
	apiKey  string
	timeout time.Duration
	client  *http.Client
}

// WebSearchPrimeConfig 配置
type WebSearchPrimeConfig struct {
	APIKey  string
	Timeout time.Duration
}

// NewWebSearchPrimeTool 创建 Web Search Prime 工具
func NewWebSearchPrimeTool(config WebSearchPrimeConfig) *WebSearchPrimeTool {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	return &WebSearchPrimeTool{
		apiKey:  config.APIKey,
		timeout: config.Timeout,
		client:  &http.Client{Timeout: config.Timeout},
	}
}

// Info 返回工具信息
func (t *WebSearchPrimeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_search_prime",
		Desc: "Enhanced web search with richer results including titles, URLs, summaries, and site metadata. Best for comprehensive web research.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"search_query": {
				Type:     schema.String,
				Desc:     "The search query (recommended under 70 characters)",
				Required: true,
			},
			"search_recency_filter": {
				Type: schema.String,
				Desc: "Time range filter: oneDay, oneWeek, oneMonth, oneYear, noLimit (default: noLimit)",
			},
			"content_size": {
				Type: schema.String,
				Desc: "Summary size: medium (400-600 words) or high (2500 words, default: medium)",
			},
		}),
	}, nil
}

// InvokableRun 执行搜索
func (t *WebSearchPrimeTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		SearchQuery         string `json:"search_query"`
		SearchRecencyFilter string `json:"search_recency_filter"`
		ContentSize         string `json:"content_size"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.SearchQuery == "" {
		return "", fmt.Errorf("search_query is required")
	}
	if t.apiKey == "" {
		return "", fmt.Errorf("API key not configured for Web Search Prime")
	}

	return t.search(ctx, args.SearchQuery, args.SearchRecencyFilter, args.ContentSize)
}

func (t *WebSearchPrimeTool) search(ctx context.Context, query, recencyFilter, contentSize string) (string, error) {
	arguments := map[string]string{
		"search_query": query,
	}
	if recencyFilter != "" {
		arguments["search_recency_filter"] = recencyFilter
	}
	if contentSize != "" {
		arguments["content_size"] = contentSize
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "webSearchPrime",
			"arguments": arguments,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webSearchPrimeMCPEndpoint, bytes.NewBuffer(jsonBody))
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
		return "", fmt.Errorf("Web Search Prime API failed (status %d): %s", resp.StatusCode, string(body))
	}

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

	var result strings.Builder
	for _, content := range mcpResp.Result.Content {
		if content.Type == "text" {
			result.WriteString(content.Text)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 增强搜索结果 - %s\n", query))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(result.String())
	return sb.String(), nil
}

var _ tool.InvokableTool = (*WebSearchPrimeTool)(nil)

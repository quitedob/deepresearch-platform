// Package tool 提供 Web Search Prime MCP 工具实现
// Web Search Prime 是智谱提供的增强版网络搜索能力
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/ai-research-platform/internal/types/constant"
)

const (
	webSearchPrimeMCPEndpoint = constant.ZhipuWebSearchPrimeMCPEndpoint
)

// WebSearchPrimeTool Web Search Prime MCP 工具
type WebSearchPrimeTool struct {
	mcpClient *MCPClient
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
		mcpClient: NewMCPClient(MCPClientConfig{
			APIKey:  config.APIKey,
			BaseURL: webSearchPrimeMCPEndpoint,
			Timeout: config.Timeout,
		}),
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

	// 构建参数
	arguments := map[string]interface{}{
		"search_query": args.SearchQuery,
	}
	if args.SearchRecencyFilter != "" {
		arguments["search_recency_filter"] = args.SearchRecencyFilter
	}
	if args.ContentSize != "" {
		arguments["content_size"] = args.ContentSize
	}

	// 使用 MCP 客户端调用工具
	result, err := t.mcpClient.CallTool(ctx, "web_search_prime", arguments)
	if err != nil {
		return "", err
	}

	// 格式化输出
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 增强搜索结果 - %s\n", args.SearchQuery))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(result)
	return sb.String(), nil
}

var _ tool.InvokableTool = (*WebSearchPrimeTool)(nil)

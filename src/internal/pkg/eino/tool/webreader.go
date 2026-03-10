// Package tool 提供 Web Reader MCP 工具实现
// Web Reader MCP 是智谱提供的网页内容抓取能力
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
	webReaderMCPEndpoint = constant.ZhipuWebReaderMCPEndpoint
)

// WebReaderTool Web Reader MCP 工具，提供网页内容抓取能力
type WebReaderTool struct {
	mcpClient *MCPClient
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
		mcpClient: NewMCPClient(MCPClientConfig{
			APIKey:  config.APIKey,
			BaseURL: webReaderMCPEndpoint,
			Timeout: config.Timeout,
		}),
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

	// 构建参数
	arguments := map[string]interface{}{
		"url": args.URL,
	}

	// 使用 MCP 客户端调用工具
	result, err := t.mcpClient.CallTool(ctx, "webReader", arguments)
	if err != nil {
		return "", err
	}

	// 格式化输出
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🌐 网页内容 - %s\n", args.URL))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(result)
	return sb.String(), nil
}

var _ tool.InvokableTool = (*WebReaderTool)(nil)

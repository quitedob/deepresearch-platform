// Package tool 提供 ZRead MCP 工具实现
// ZRead MCP 是智谱提供的开源仓库读取能力
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
	zreadMCPEndpoint = constant.ZhipuZReadMCPEndpoint
)

// ZReadTool ZRead MCP 工具，提供开源仓库读取能力
type ZReadTool struct {
	mcpClient *MCPClient
}

// ZReadConfig ZRead 配置
type ZReadConfig struct {
	APIKey  string
	Timeout time.Duration
}

// NewZReadTool 创建 ZRead 工具
func NewZReadTool(config ZReadConfig) *ZReadTool {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	return &ZReadTool{
		mcpClient: NewMCPClient(MCPClientConfig{
			APIKey:  config.APIKey,
			BaseURL: zreadMCPEndpoint,
			Timeout: config.Timeout,
		}),
	}
}

// Info 返回工具信息
func (t *ZReadTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "zread_repo",
		Desc: "Access GitHub repository content including documentation search, file reading, and repository structure. Supports search_doc, read_file, and get_repo_structure operations.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"operation": {
				Type:     schema.String,
				Desc:     "Operation type: search_doc, read_file, or get_repo_structure",
				Required: true,
			},
			"repo_name": {
				Type:     schema.String,
				Desc:     "GitHub repository in format 'owner/repo' (e.g., 'vitejs/vite')",
				Required: true,
			},
			"query": {
				Type: schema.String,
				Desc: "Search query (required for search_doc operation)",
			},
			"file_path": {
				Type: schema.String,
				Desc: "File path to read (required for read_file operation)",
			},
			"dir_path": {
				Type: schema.String,
				Desc: "Directory path to inspect (optional for get_repo_structure, default: root '/')",
			},
			"language": {
				Type: schema.String,
				Desc: "Language for search results: 'zh' or 'en' (optional for search_doc)",
			},
		}),
	}, nil
}

// InvokableRun 执行 ZRead 操作
func (t *ZReadTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Operation string `json:"operation"`
		RepoName  string `json:"repo_name"`
		Query     string `json:"query"`
		FilePath  string `json:"file_path"`
		DirPath   string `json:"dir_path"`
		Language  string `json:"language"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.RepoName == "" {
		return "", fmt.Errorf("repo_name is required")
	}

	// 根据操作类型构建参数
	var toolName string
	var arguments map[string]interface{}

	switch args.Operation {
	case "search_doc":
		if args.Query == "" {
			return "", fmt.Errorf("query is required for search_doc operation")
		}
		toolName = "search_doc"
		arguments = map[string]interface{}{
			"repo_name": args.RepoName,
			"query":     args.Query,
			"language":  args.Language,
		}
		if args.Language == "" {
			arguments["language"] = "zh"
		}

	case "read_file":
		if args.FilePath == "" {
			return "", fmt.Errorf("file_path is required for read_file operation")
		}
		toolName = "read_file"
		arguments = map[string]interface{}{
			"repo_name": args.RepoName,
			"file_path": args.FilePath,
		}

	case "get_repo_structure":
		toolName = "get_repo_structure"
		arguments = map[string]interface{}{
			"repo_name": args.RepoName,
		}
		if args.DirPath != "" {
			arguments["dir_path"] = args.DirPath
		}

	default:
		return "", fmt.Errorf("unknown operation: %s (supported: search_doc, read_file, get_repo_structure)", args.Operation)
	}

	// 使用 MCP 客户端调用工具
	result, err := t.mcpClient.CallTool(ctx, toolName, arguments)
	if err != nil {
		return "", err
	}

	// 格式化输出
	var sb strings.Builder
	switch args.Operation {
	case "search_doc":
		sb.WriteString(fmt.Sprintf("📚 仓库文档搜索结果 - %s\n", args.RepoName))
		sb.WriteString(fmt.Sprintf("🔍 查询: %s\n", args.Query))
	case "read_file":
		sb.WriteString(fmt.Sprintf("📄 文件内容 - %s/%s\n", args.RepoName, args.FilePath))
	case "get_repo_structure":
		sb.WriteString(fmt.Sprintf("📁 仓库结构 - %s\n", args.RepoName))
		if args.DirPath != "" {
			sb.WriteString(fmt.Sprintf("📂 目录: %s\n", args.DirPath))
		}
	}
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(result)
	return sb.String(), nil
}

var _ tool.InvokableTool = (*ZReadTool)(nil)

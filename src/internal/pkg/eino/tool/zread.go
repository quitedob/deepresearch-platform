// Package tool 提供 ZRead MCP 工具实现
// ZRead MCP 是智谱提供的开源仓库读取能力
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

	"github.com/ai-research-platform/internal/types/constant"
)

const (
	zreadMCPEndpoint = constant.ZhipuZReadMCPEndpoint
)

// ZReadTool ZRead MCP 工具，提供开源仓库读取能力
type ZReadTool struct {
	apiKey  string
	timeout time.Duration
	client  *http.Client
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
		apiKey:  config.APIKey,
		timeout: config.Timeout,
		client:  &http.Client{Timeout: config.Timeout},
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

	if t.apiKey == "" {
		return "", fmt.Errorf("API key not configured for ZRead MCP")
	}

	switch args.Operation {
	case "search_doc":
		if args.Query == "" {
			return "", fmt.Errorf("query is required for search_doc operation")
		}
		return t.searchDoc(ctx, args.RepoName, args.Query, args.Language)
	case "read_file":
		if args.FilePath == "" {
			return "", fmt.Errorf("file_path is required for read_file operation")
		}
		return t.readFile(ctx, args.RepoName, args.FilePath)
	case "get_repo_structure":
		return t.getRepoStructure(ctx, args.RepoName, args.DirPath)
	default:
		return "", fmt.Errorf("unknown operation: %s (supported: search_doc, read_file, get_repo_structure)", args.Operation)
	}
}


// searchDoc 搜索仓库文档
func (t *ZReadTool) searchDoc(ctx context.Context, repoName, query, language string) (string, error) {
	if language == "" {
		language = "zh"
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "search_doc",
			"arguments": map[string]string{
				"repo_name": repoName,
				"query":     query,
				"language":  language,
			},
		},
	}

	result, err := t.callMCP(ctx, reqBody)
	if err != nil {
		return "", fmt.Errorf("search_doc failed: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 仓库文档搜索结果 - %s\n", repoName))
	sb.WriteString(fmt.Sprintf("🔍 查询: %s\n", query))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(result)
	return sb.String(), nil
}

// readFile 读取仓库文件
func (t *ZReadTool) readFile(ctx context.Context, repoName, filePath string) (string, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "read_file",
			"arguments": map[string]string{
				"repo_name": repoName,
				"file_path": filePath,
			},
		},
	}

	result, err := t.callMCP(ctx, reqBody)
	if err != nil {
		return "", fmt.Errorf("read_file failed: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📄 文件内容 - %s/%s\n", repoName, filePath))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(result)
	return sb.String(), nil
}

// getRepoStructure 获取仓库结构
func (t *ZReadTool) getRepoStructure(ctx context.Context, repoName, dirPath string) (string, error) {
	args := map[string]string{
		"repo_name": repoName,
	}
	if dirPath != "" {
		args["dir_path"] = dirPath
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "get_repo_structure",
			"arguments": args,
		},
	}

	result, err := t.callMCP(ctx, reqBody)
	if err != nil {
		return "", fmt.Errorf("get_repo_structure failed: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📁 仓库结构 - %s\n", repoName))
	if dirPath != "" {
		sb.WriteString(fmt.Sprintf("📂 目录: %s\n", dirPath))
	}
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(result)
	return sb.String(), nil
}

// callMCP 调用 MCP 接口
func (t *ZReadTool) callMCP(ctx context.Context, reqBody map[string]interface{}) (string, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", zreadMCPEndpoint, bytes.NewBuffer(jsonBody))
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
		return "", fmt.Errorf("MCP API failed (status %d): %s", resp.StatusCode, string(body))
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

	return result.String(), nil
}

var _ tool.InvokableTool = (*ZReadTool)(nil)

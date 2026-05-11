// Package tool 提供 MiniMax 网络搜索工具实现
// 通过 MCP stdio 协议调用 MiniMax Token Plan 的 web_search 工具
// 使用 uvx minimax-coding-plan-mcp 子进程，JSON-RPC over stdin/stdout 通信
package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// MiniMaxWebSearchTool 使用 MCP stdio 协议调用 MiniMax Token Plan 网络搜索
type MiniMaxWebSearchTool struct {
	apiKey  string
	uvxPath string
	timeout time.Duration

	mu          sync.Mutex
	cmd         *exec.Cmd
	stdinPipe   io.WriteCloser
	stdinWriter *bufio.Writer
	stdout      *bufio.Reader
	initialized bool
	msgID       int
}

// MiniMaxWebSearchConfig MiniMax 网络搜索配置
type MiniMaxWebSearchConfig struct {
	APIKey  string
	Timeout time.Duration
	UvxPath string // 可选，为空时自动检测 uvx 路径
}

// NewMiniMaxWebSearchTool 创建 MiniMax 网络搜索工具
func NewMiniMaxWebSearchTool(config MiniMaxWebSearchConfig) *MiniMaxWebSearchTool {
	if config.Timeout == 0 {
		config.Timeout = 120 * time.Second
	}
	t := &MiniMaxWebSearchTool{
		apiKey:  config.APIKey,
		uvxPath: config.UvxPath,
		timeout: config.Timeout,
	}
	runtime.SetFinalizer(t, (*MiniMaxWebSearchTool).Close)
	return t
}

// Close 停止 MCP 子进程，释放资源
func (t *MiniMaxWebSearchTool) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopProcess()
}

// Info 返回工具描述
func (t *MiniMaxWebSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "minimax_web_search",
		Desc: "Use MiniMax to search the web for current information, news, and real-time data.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "搜索关键词或问题",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun 执行 MiniMax 网络搜索
func (t *MiniMaxWebSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// 确保 MCP 子进程已启动并初始化
	if err := t.ensureProcess(); err != nil {
		return "", fmt.Errorf("MCP 进程启动失败: %w", err)
	}

	// 调用 web_search 工具
	result, err := t.callTool("web_search", map[string]interface{}{"query": args.Query})
	if err != nil {
		log.Printf("[WARN] MiniMax MCP web_search 失败, 将重启进程: %v", err)
		t.stopProcess()
		return "", err
	}

	return result, nil
}

// ensureProcess 确保子进程已启动并完成 MCP 握手
func (t *MiniMaxWebSearchTool) ensureProcess() error {
	if t.initialized && t.cmd != nil && t.cmd.Process != nil {
		if t.cmd.ProcessState == nil || !t.cmd.ProcessState.Exited() {
			return nil
		}
	}
	return t.startProcess()
}

// startProcess 启动 uvx minimax-coding-plan-mcp 子进程并完成 MCP 握手
func (t *MiniMaxWebSearchTool) startProcess() error {
	t.stopProcess()

	uvxPath := t.uvxPath
	if uvxPath == "" {
		var err error
		uvxPath, err = findUvx()
		if err != nil {
			return err
		}
	}

	t.cmd = exec.Command(uvxPath, "minimax-coding-plan-mcp", "-y")
	t.cmd.Env = append(os.Environ(),
		"MINIMAX_API_KEY="+t.apiKey,
		"MINIMAX_API_HOST=https://api.minimaxi.com",
	)

	var err error
	t.stdinPipe, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	t.stdinWriter = bufio.NewWriter(t.stdinPipe)

	stdoutPipe, err := t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	t.stdout = bufio.NewReaderSize(stdoutPipe, 2*1024*1024)

	t.cmd.Stderr = nil

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("启动 uvx 失败: %w", err)
	}

	t.msgID = 0

	// MCP 握手: initialize → notifications/initialized
	if _, err := t.sendRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "ai-research-platform",
			"version": "1.0.0",
		},
	}); err != nil {
		t.stopProcess()
		return fmt.Errorf("MCP initialize 失败: %w", err)
	}

	if err := t.sendNotification("notifications/initialized"); err != nil {
		t.stopProcess()
		return fmt.Errorf("MCP initialized 通知失败: %w", err)
	}

	t.initialized = true
	log.Printf("[INFO] MiniMax MCP 子进程已启动并完成握手")
	return nil
}

// callTool 调用 MCP 工具并提取文本内容
func (t *MiniMaxWebSearchTool) callTool(name string, arguments map[string]interface{}) (string, error) {
	resp, err := t.sendRequest("tools/call", map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	})
	if err != nil {
		return "", err
	}

	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("MCP 响应无 result 字段")
	}

	content, ok := result["content"].([]interface{})
	if !ok {
		return "", fmt.Errorf("MCP result 无 content 字段")
	}

	var sb strings.Builder
	for _, item := range content {
		if ci, ok := item.(map[string]interface{}); ok {
			if text, ok := ci["text"].(string); ok {
				sb.WriteString(text)
			}
		}
	}

	if sb.Len() == 0 {
		return "", fmt.Errorf("搜索结果为空")
	}
	return sb.String(), nil
}

// sendRequest 发送 JSON-RPC 请求并等待响应
func (t *MiniMaxWebSearchTool) sendRequest(method string, params interface{}) (map[string]interface{}, error) {
	t.msgID++
	if err := t.writeJSON(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      t.msgID,
		"method":  method,
		"params":  params,
	}); err != nil {
		return nil, err
	}
	return t.readResponse(t.timeout)
}

// sendNotification 发送 JSON-RPC 通知（无 id，不等待响应）
func (t *MiniMaxWebSearchTool) sendNotification(method string) error {
	return t.writeJSON(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
	})
}

// writeJSON 写入一行 JSON 到子进程 stdin
func (t *MiniMaxWebSearchTool) writeJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := t.stdinWriter.Write(append(data, '\n')); err != nil {
		return err
	}
	return t.stdinWriter.Flush()
}

// readResponse 带超时读取 JSON-RPC 响应
func (t *MiniMaxWebSearchTool) readResponse(timeout time.Duration) (map[string]interface{}, error) {
	type rr struct {
		resp map[string]interface{}
		err  error
	}
	ch := make(chan rr, 1)
	go func() {
		line, err := t.stdout.ReadString('\n')
		if err != nil {
			ch <- rr{nil, err}
			return
		}
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &resp); err != nil {
			ch <- rr{nil, err}
			return
		}
		ch <- rr{resp, nil}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return nil, r.err
		}
		// 检查 JSON-RPC 错误
		if errMsg, ok := r.resp["error"].(map[string]interface{}); ok {
			msg, _ := errMsg["message"].(string)
			code, _ := errMsg["code"].(float64)
			return nil, fmt.Errorf("MCP error (code=%.0f): %s", code, msg)
		}
		return r.resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("MCP 响应超时 (%v)", timeout)
	}
}

// stopProcess 停止子进程
func (t *MiniMaxWebSearchTool) stopProcess() {
	if t.stdinPipe != nil {
		t.stdinPipe.Close()
		t.stdinPipe = nil
	}
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}
	t.cmd = nil
	t.stdinWriter = nil
	t.stdout = nil
	t.initialized = false
}

// findUvx 查找 uvx 可执行文件
func findUvx() (string, error) {
	// 从 PATH 查找
	for _, name := range []string{"uvx", "uvx.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	// Windows 默认安装位置
	if homeDir, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(homeDir, ".local", "bin", "uvx.exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("uvx 未找到，请先安装 uv: https://astral.sh/uv")
}

// 编译期校验接口实现
var _ tool.InvokableTool = (*MiniMaxWebSearchTool)(nil)

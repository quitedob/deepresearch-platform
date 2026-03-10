// Package tool 提供 MCP (Model Context Protocol) 客户端基础实现
// 支持 MCP Streamable HTTP 传输协议，包括 initialize 握手
package tool

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// sharedHTTPTransport 共享的 HTTP 连接池，避免每个 MCPClient 创建独立连接
var sharedHTTPTransport = &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 10,
	IdleConnTimeout:     90 * time.Second,
}

// MCPClient MCP 客户端基础实现
type MCPClient struct {
	apiKey     string
	baseURL    string
	timeout    time.Duration
	client     *http.Client
	sessionID  string
	sessionMux sync.RWMutex
}

// MCPClientConfig MCP 客户端配置
type MCPClientConfig struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// NewMCPClient 创建 MCP 客户端
func NewMCPClient(config MCPClientConfig) *MCPClient {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	return &MCPClient{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		timeout: config.Timeout,
		client: &http.Client{
			Timeout:   config.Timeout,
			Transport: sharedHTTPTransport,
		},
	}
}

// GetSessionID 获取当前 Session ID
func (c *MCPClient) GetSessionID() string {
	c.sessionMux.RLock()
	defer c.sessionMux.RUnlock()
	return c.sessionID
}

// Initialize 执行 MCP initialize 握手
// MCP Streamable HTTP 要求先发送 initialize 请求，获取 Session ID，
// 然后发送 notifications/initialized 通知完成握手
func (c *MCPClient) Initialize(ctx context.Context) error {
	c.sessionMux.Lock()
	defer c.sessionMux.Unlock()

	// 如果已有 Session ID，跳过
	if c.sessionID != "" {
		return nil
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]string{
				"name":    "ai-research-platform",
				"version": "1.0.0",
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal initialize request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create initialize request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取完整响应体（必须消费完毕）
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("initialize failed (status %d): %s", resp.StatusCode, string(body))
	}

	// 从响应头获取 Session ID
	sessionID := resp.Header.Get("Mcp-Session-Id")
	if sessionID != "" {
		c.sessionID = sessionID
	}

	// 尝试从 SSE 响应体中解析 Session ID（如果响应头没有）
	if c.sessionID == "" {
		// SSE 响应可能包含 "data: {...}" 前缀
		bodyStr := string(body)
		jsonStr := bodyStr
		if strings.HasPrefix(bodyStr, "data:") {
			jsonStr = strings.TrimSpace(strings.TrimPrefix(bodyStr, "data:"))
		}
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
			if res, ok := result["result"].(map[string]interface{}); ok {
				if sid, ok := res["sessionId"].(string); ok {
					c.sessionID = sid
				}
			}
		}
	}

	// 发送 notifications/initialized 通知完成握手
	if err := c.sendInitializedNotification(ctx); err != nil {
		// 不阻断流程，某些服务可能不要求此通知
	}

	return nil
}

// sendInitializedNotification 发送 MCP initialized 通知
func (c *MCPClient) sendInitializedNotification(ctx context.Context) error {
	notifBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}

	jsonBody, err := json.Marshal(notifBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body) // 消费响应体

	return nil
}

// CallTool 调用 MCP 工具
func (c *MCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	// 先执行 initialize（如果需要）
	if err := c.Initialize(ctx); err != nil {
		log.Printf("[WARN] MCP initialize failed (will attempt call anyway): %v", err)
	}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	// MCP 服务使用 Bearer {api_key} 直接认证
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 添加 Session ID（如果有）
	c.sessionMux.RLock()
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	c.sessionMux.RUnlock()

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 更新 Session ID（如果响应中有新的）
	if newSessionID := resp.Header.Get("Mcp-Session-Id"); newSessionID != "" {
		c.sessionMux.Lock()
		c.sessionID = newSessionID
		c.sessionMux.Unlock()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("MCP API failed (status %d): %s", resp.StatusCode, string(body))
	}

	// 读取 SSE 流式响应
	return c.parseSSEResponse(resp.Body)
}

// parseSSEResponse 解析 SSE 流式响应
func (c *MCPClient) parseSSEResponse(body io.Reader) (string, error) {
	var fullResult strings.Builder
	scanner := newSSEScanner(body)
	lineNum := 0
	var rawLines []string // 收集原始行用于诊断

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// 跳过空行
		if line == "" {
			continue
		}

		rawLines = append(rawLines, truncateString(line, 200))

		// SSE 格式: "data: {json}"
		if strings.HasPrefix(line, "data:") {
			jsonStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			// 解析 JSON 数据
			var eventData map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &eventData); err != nil {
				log.Printf("[WARN] MCP SSE JSON parse failed (line %d): %v, raw: %s", lineNum, err, truncateString(jsonStr, 100))
				continue
			}

			// 检查 API 级别的错误
			if errMsg, ok := eventData["error"].(map[string]interface{}); ok {
				msg, _ := errMsg["message"].(string)
				code, _ := errMsg["code"].(float64)
				if msg != "" {
					return "", fmt.Errorf("MCP API error (code=%.0f): %s", code, msg)
				}
				// error 对象存在但没有 message 字段
				errJSON, _ := json.Marshal(errMsg)
				return "", fmt.Errorf("MCP API error: %s", string(errJSON))
			}

			// 提取 result 中的文本内容（兼容多种格式）
			if result, ok := eventData["result"].(map[string]interface{}); ok {
				extracted := false

				// 格式1: content 是数组 [{"type": "text", "text": "..."}]
				if content, ok := result["content"].([]interface{}); ok {
					for _, item := range content {
						if contentItem, ok := item.(map[string]interface{}); ok {
							if text, ok := contentItem["text"].(string); ok {
								fullResult.WriteString(text)
								extracted = true
							}
						}
					}
				}

				// 格式2: content 是字符串
				if !extracted {
					if contentStr, ok := result["content"].(string); ok {
						fullResult.WriteString(contentStr)
						extracted = true
					}
				}

				// 格式3: 直接是 text 字段
				if !extracted {
					if text, ok := result["text"].(string); ok {
						fullResult.WriteString(text)
						extracted = true
					}
				}

				if !extracted {
					// 记录未识别的 result 结构以便诊断
					resultKeys := make([]string, 0, len(result))
					for k := range result {
						resultKeys = append(resultKeys, k)
					}
					log.Printf("[WARN] MCP SSE: unrecognized result structure, keys: %v", resultKeys)
				}
			}

			// 检查是否结束 (done 标志)
			if _, ok := eventData["done"].(bool); ok {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read SSE stream: %w", err)
	}

	result := fullResult.String()
	if result == "" {
		// 提供诊断信息区分"无响应"和"格式不匹配"
		if len(rawLines) == 0 {
			return "", fmt.Errorf("MCP API returned empty response (no SSE data lines)")
		}
		return "", fmt.Errorf("MCP API returned %d SSE lines but no extractable content; first line: %s", len(rawLines), rawLines[0])
	}

	return result, nil
}

// truncateString 截取字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// sseScanner SSE 流扫描器
type sseScanner struct {
	scanner *bufio.Scanner
}

// newSSEScanner 创建 SSE 扫描器
func newSSEScanner(r io.Reader) *sseScanner {
	return &sseScanner{
		scanner: bufio.NewScanner(r),
	}
}

// Scan 扫描下一行
func (s *sseScanner) Scan() bool {
	return s.scanner.Scan()
}

// Text 获取当前行文本
func (s *sseScanner) Text() string {
	return s.scanner.Text()
}

// Err 返回错误
func (s *sseScanner) Err() error {
	return s.scanner.Err()
}

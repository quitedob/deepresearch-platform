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
	"github.com/golang-jwt/jwt/v5"
)

// WebSearchTool 网络搜索工具，使用智谱AI web_search
type WebSearchTool struct {
	apiKey  string
	timeout time.Duration
	client  *http.Client
}

// WebSearchConfig 网络搜索配置
type WebSearchConfig struct {
	APIKey  string
	Timeout time.Duration
}

// NewWebSearchTool 创建网络搜索工具
// API Key 必须通过 config 传入，不做 fallback
func NewWebSearchTool(config WebSearchConfig) *WebSearchTool {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	return &WebSearchTool{
		apiKey:  config.APIKey,
		timeout: config.Timeout,
		client:  &http.Client{Timeout: config.Timeout},
	}
}

// generateJWTToken 生成智谱AI的JWT Token
func (t *WebSearchTool) generateJWTToken() string {
	parts := strings.Split(t.apiKey, ".")
	if len(parts) != 2 {
		return t.apiKey // 如果不是 id.secret 格式，直接返回原始 key
	}
	
	id, secret := parts[0], parts[1]
	
	payload := jwt.MapClaims{
		"api_key":   id,
		"exp":       time.Now().Add(10 * time.Minute).UnixMilli(),
		"timestamp": time.Now().UnixMilli(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token.Header["sign_type"] = "SIGN"
	
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return t.apiKey
	}
	return signedToken
}

// Info 返回工具信息
func (t *WebSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_search",
		Desc: "Search the web for current information. Use this for recent events, news, or real-time data.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type: schema.String,
				Desc: "The search query",
			},
		}),
	}, nil
}

// InvokableRun 执行搜索
func (t *WebSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	if t.apiKey == "" {
		return t.mockSearch(args.Query), nil
	}

	return t.searchZhipu(ctx, args.Query)
}

func (t *WebSearchTool) searchZhipu(ctx context.Context, query string) (string, error) {
	// 使用 web_search 工具进行搜索
	reqBody := map[string]interface{}{
		"model": "glm-4-flash",
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("请搜索并总结关于以下主题的最新信息：%s", query)},
		},
		"tools": []map[string]interface{}{
			{
				"type": "web_search",
				"web_search": map[string]interface{}{
					"enable":       true,
					"search_query": query,
				},
			},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://open.bigmodel.cn/api/paas/v4/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	
	// 使用 JWT Token 进行认证
	jwtToken := t.generateJWTToken()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

	// 使用已有 client（复用连接池）
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API failed (status %d): %s", resp.StatusCode, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Error.Message != "" {
		return "", fmt.Errorf("API error: %s (code: %s)", response.Error.Message, response.Error.Code)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := response.Choices[0].Message.Content
	if content == "" {
		return "", fmt.Errorf("empty response content")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("网络搜索结果 - '%s'\n", query))
	sb.WriteString(strings.Repeat("=", 50) + "\n\n")
	sb.WriteString(content)
	return sb.String(), nil
}

func (t *WebSearchTool) mockSearch(query string) string {
	return fmt.Sprintf("网络搜索结果 - '%s'\n\n这是模拟的搜索结果。请配置 ZHIPU_API_KEY 以使用真实搜索。", query)
}

var _ tool.InvokableTool = (*WebSearchTool)(nil)

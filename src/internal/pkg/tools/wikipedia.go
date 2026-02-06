package tools

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
)

// WikipediaTool Wikipedia工具
type WikipediaTool struct {
    baseURL string
    client  *http.Client
}

// NewWikipediaTool 创建Wikipedia工具
func NewWikipediaTool() *WikipediaTool {
    return &WikipediaTool{
        baseURL: "https://en.wikipedia.org/api/rest_v1",
        client:  &http.Client{},
    }
}

// Name 工具名称
func (w *WikipediaTool) Name() string {
    return "wikipedia"
}

// Description 工具描述
func (w *WikipediaTool) Description() string {
    return "Search Wikipedia articles and get article summaries"
}

// Execute 执行工具
func (w *WikipediaTool) Execute(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
    query, ok := input["query"].(string)
    if !ok {
        return &ToolResult{
            Success: false,
            Error:   "query parameter is required",
        }, nil
    }

    // 搜索Wikipedia文章
    articles, err := w.searchArticles(ctx, query)
    if err != nil {
        return &ToolResult{
            Success: false,
            Error:   fmt.Sprintf("search failed: %v", err),
        }, nil
    }

    return &ToolResult{
        Success: true,
        Data: map[string]interface{}{
            "articles": articles,
            "query":    query,
        },
    }, nil
}

// GetSchema 获取工具参数模式
func (w *WikipediaTool) GetSchema() *ToolSchema {
    return &ToolSchema{
        Type: "object",
        Properties: map[string]*PropertySchema{
            "query": {
                Type:        "string",
                Description: "Search query for Wikipedia articles",
            },
            "limit": {
                Type:        "integer",
                Description: "Maximum number of results to return",
                Default:     5,
            },
        },
        Required: []string{"query"},
    }
}

// searchArticles 搜索Wikipedia文章
func (w *WikipediaTool) searchArticles(ctx context.Context, query string) ([]map[string]interface{}, error) {
    // 构建搜索URL
    searchURL := fmt.Sprintf("%s/page/summary/%s", w.baseURL, url.QueryEscape(query))

    // 发起HTTP请求
    req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
    if err != nil {
        return nil, err
    }

    resp, err := w.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("Wikipedia API returned status %d", resp.StatusCode)
    }

    // 解析响应
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var article map[string]interface{}
    if err := json.Unmarshal(body, &article); err != nil {
        return nil, err
    }

    return []map[string]interface{}{article}, nil
}

package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/ai-research-platform/internal/types/constant"
)

// WikipediaTool Wikipedia 查询工具
type WikipediaTool struct {
	language string
	timeout  time.Duration
	client   *http.Client
}

// WikipediaConfig Wikipedia 配置
type WikipediaConfig struct {
	Language string
	Timeout  time.Duration
}

// NewWikipediaTool 创建 Wikipedia 工具
func NewWikipediaTool(config WikipediaConfig) *WikipediaTool {
	if config.Language == "" {
		config.Language = "zh"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &WikipediaTool{
		language: config.Language,
		timeout:  config.Timeout,
		client:   &http.Client{Timeout: config.Timeout},
	}
}

// Info 返回工具信息
func (t *WikipediaTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "wikipedia",
		Desc: "Look up information from Wikipedia. Use for encyclopedic knowledge and background information.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type: schema.String,
				Desc: "The article title or search query",
			},
			"language": {
				Type: schema.String,
				Desc: "Wikipedia language code (zh, en, etc.)",
			},
		}),
	}, nil
}

// InvokableRun 执行查询
func (t *WikipediaTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Query    string `json:"query"`
		Language string `json:"language"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	language := t.language
	if args.Language != "" {
		language = args.Language
	}

	content, err := t.getPageContent(ctx, args.Query, language)
	if err != nil {
		return fmt.Sprintf("查询失败: %v", err), nil
	}
	if content == "" {
		return fmt.Sprintf("未找到关于'%s'的维基百科页面", args.Query), nil
	}
	return content, nil
}

func (t *WikipediaTool) getPageContent(ctx context.Context, pageTitle string, language string) (string, error) {
	apiURL := fmt.Sprintf(constant.WikipediaAPITemplate, language)
	params := url.Values{}
	params.Set("action", "query")
	params.Set("prop", "extracts|info")
	params.Set("titles", pageTitle)
	params.Set("format", "json")
	params.Set("explaintext", "1")
	params.Set("exsectionformat", "plain")
	params.Set("inprop", "url")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "DeepResearch/1.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	// P1 修复：使用安全类型断言代替强制断言，防止异常响应 panic
	queryData, ok := data["query"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected API response format: missing 'query' field")
	}
	pages, ok := queryData["pages"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected API response format: missing 'pages' field")
	}

	for pageID, pageData := range pages {
		if pageID == "-1" {
			return "", nil
		}
		page, ok := pageData.(map[string]interface{})
		if !ok {
			continue
		}
		title, _ := page["title"].(string)
		extract := ""
		if e, ok := page["extract"].(string); ok {
			extract = e
		}
		pageURL := ""
		if u, ok := page["fullurl"].(string); ok {
			pageURL = u
		}

		if len(extract) > 3000 {
			extract = extract[:3000] + "...[内容已截断]"
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("维基百科: %s\n", title))
		sb.WriteString(strings.Repeat("=", 50) + "\n\n")
		if pageURL != "" {
			sb.WriteString(fmt.Sprintf("链接: %s\n\n", pageURL))
		}
		sb.WriteString(extract)
		return sb.String(), nil
	}
	return "", nil
}

var _ tool.InvokableTool = (*WikipediaTool)(nil)

// Package tool 提供研究工具实现
package tool

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ArxivTool ArXiv 学术搜索工具，实Eino InvokableTool 接口
type ArxivTool struct {
	maxResults int
	timeout    time.Duration
	client     *http.Client
}

// ArxivConfig ArXiv 配置
type ArxivConfig struct {
	MaxResults int
	Timeout    time.Duration
}

// NewArxivTool 创建 ArXiv 工具
func NewArxivTool(config ArxivConfig) *ArxivTool {
	if config.MaxResults == 0 {
		config.MaxResults = 10
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &ArxivTool{
		maxResults: config.MaxResults,
		timeout:    config.Timeout,
		client:     &http.Client{Timeout: config.Timeout},
	}
}

// Info 返回工具信息，实BaseTool 接口
func (t *ArxivTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "arxiv_search",
		Desc: "Search arXiv for academic papers. Input should be a search query string.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type: schema.String,
				Desc: "The search query for academic papers",
			},
			"max_results": {
				Type: schema.Integer,
				Desc: "Maximum number of papers to return (default 10)",
			},
		}),
	}, nil
}

// InvokableRun 执行搜索，实InvokableTool 接口
func (t *ArxivTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	maxResults := t.maxResults
	if args.MaxResults > 0 && args.MaxResults <= 50 {
		maxResults = args.MaxResults
	}

	papers, err := t.search(ctx, args.Query, maxResults)
	if err != nil {
		return fmt.Sprintf("搜索失败: %v", err), nil
	}

	return t.formatPapers(papers, args.Query), nil
}

type arxivPaper struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Authors   []string `json:"authors"`
	Abstract  string   `json:"abstract"`
	URL       string   `json:"url"`
	Published string   `json:"published"`
}

type arxivFeed struct {
	XMLName xml.Name     `xml:"feed"`
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	ID        string `xml:"id"`
	Title     string `xml:"title"`
	Summary   string `xml:"summary"`
	Published string `xml:"published"`
	Authors   []struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Links []struct {
		Href  string `xml:"href,attr"`
		Title string `xml:"title,attr"`
	} `xml:"link"`
}

func (t *ArxivTool) search(ctx context.Context, query string, maxResults int) ([]arxivPaper, error) {
	params := url.Values{}
	params.Set("search_query", fmt.Sprintf("all:%s", query))
	params.Set("start", "0")
	params.Set("max_results", fmt.Sprintf("%d", maxResults))
	params.Set("sortBy", "relevance")

	apiURL := fmt.Sprintf("http://export.arxiv.org/api/query?%s", params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DeepResearch/1.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API failed with status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var feed arxivFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}

	papers := make([]arxivPaper, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		paperID := entry.ID
		if idx := strings.LastIndex(paperID, "/"); idx != -1 {
			paperID = paperID[idx+1:]
		}

		authors := make([]string, len(entry.Authors))
		for i, a := range entry.Authors {
			authors[i] = strings.TrimSpace(a.Name)
		}

		papers = append(papers, arxivPaper{
			ID:        paperID,
			Title:     strings.TrimSpace(strings.ReplaceAll(entry.Title, "\n", " ")),
			Authors:   authors,
			Abstract:  strings.TrimSpace(strings.ReplaceAll(entry.Summary, "\n", " ")),
			URL:       fmt.Sprintf("https://arxiv.org/abs/%s", paperID),
			Published: entry.Published,
		})
	}
	return papers, nil
}

func (t *ArxivTool) formatPapers(papers []arxivPaper, query string) string {
	if len(papers) == 0 {
		return fmt.Sprintf("未找到关于'%s'的论文", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("ArXiv论文搜索结果 - %s (%d篇)\n\n", query, len(papers)))

	for i, paper := range papers {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, paper.Title))
		authorsStr := strings.Join(paper.Authors, ", ")
		if len(paper.Authors) > 3 {
			authorsStr = strings.Join(paper.Authors[:3], ", ") + " 等"
		}
		sb.WriteString(fmt.Sprintf("   作者: %s\n", authorsStr))
		sb.WriteString(fmt.Sprintf("   链接: %s\n", paper.URL))

		abstract := paper.Abstract
		if len(abstract) > 200 {
			abstract = abstract[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("   摘要: %s\n\n", abstract))
	}
	return sb.String()
}

// 确保实现InvokableTool 接口
var _ tool.InvokableTool = (*ArxivTool)(nil)

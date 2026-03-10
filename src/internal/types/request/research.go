package request

// StartResearchRequest 开始研究请求
type StartResearchRequest struct {
	Query        string            `json:"query" binding:"required,min=1,max=10000"`
	ResearchType string            `json:"research_type"` // quick, deep, comprehensive
	LLMConfig    *LLMConfig        `json:"llm_config,omitempty"`
	ToolsConfig  *ToolsConfig      `json:"tools_config,omitempty"`
	Options      *ResearchOptions  `json:"options,omitempty"`
}

// LLMConfig LLM配置
type LLMConfig struct {
	Provider string `json:"provider"` // deepseek, zhipu, ollama
	Model    string `json:"model"`
	APIKey   string `json:"api_key,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
}

// ToolsConfig 工具配置
type ToolsConfig struct {
	EnabledTools  []string `json:"enabled_tools"`  // web_search, arxiv, wikipedia
	MaxSources    int      `json:"max_sources"`
	MaxIterations int      `json:"max_iterations"`
	EnableCache   bool     `json:"enable_cache"`
}

// ResearchOptions 研究选项
type ResearchOptions struct {
	Language         string `json:"language"`          // zh-CN, en-US, auto
	OutputFormat     string `json:"output_format"`     // structured, markdown, plain
	IncludeSources   bool   `json:"include_sources"`
	IncludeReasoning bool   `json:"include_reasoning"`
}

// GetResearchSessionsRequest 获取研究会话列表请求
type GetResearchSessionsRequest struct {
	Limit  int    `json:"limit" form:"limit" binding:"min=1,max=100"`
	Offset int    `json:"offset" form:"offset" binding:"min=0"`
	Status string `json:"status" form:"status"` // planning, executing, completed, failed
}

// ExportResearchRequest 导出研究请求
type ExportResearchRequest struct {
	SessionID string `json:"session_id" binding:"required,uuid"`
	Format    string `json:"format"` // json, markdown, pdf
}

// SearchResearchRequest 搜索研究请求
type SearchResearchRequest struct {
	Query  string `json:"query" form:"query" binding:"required,min=1"`
	Limit  int    `json:"limit" form:"limit" binding:"min=1,max=100"`
	Offset int    `json:"offset" form:"offset" binding:"min=0"`
}

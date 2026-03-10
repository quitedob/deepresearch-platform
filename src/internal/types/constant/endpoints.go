package constant

// LLM Provider Base URLs
const (
	BaseURLDeepSeek   = "https://api.deepseek.com"
	BaseURLZhipu      = "https://open.bigmodel.cn"
	BaseURLZhipuAPI   = "https://open.bigmodel.cn/api/paas/v4"
	BaseURLOllama     = "http://localhost:11434"
	BaseURLOpenAI     = "https://api.z.ai/api/coding/paas/v4"
	BaseURLOpenRouter = "https://openrouter.ai/api/v1"
)

// LLM Provider Base URLs with version path (for direct API calls)
const (
	BaseURLDeepSeekV1   = "https://api.deepseek.com/v1"
	BaseURLDeepSeekBeta = "https://api.deepseek.com/beta"
	BaseURLOllamaV1     = "http://localhost:11434/v1"
)

// Zhipu MCP Tool Endpoints
const (
	ZhipuChatCompletionsURL    = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	ZhipuWebReaderMCPEndpoint  = "https://open.bigmodel.cn/api/mcp/web_reader/mcp"
	ZhipuZReadMCPEndpoint      = "https://open.bigmodel.cn/api/mcp/zread/mcp"
	ZhipuWebSearchPrimeMCPEndpoint = "https://open.bigmodel.cn/api/mcp/web_search_prime/mcp"
)

// External API Endpoints
const (
	WikipediaAPITemplate = "https://%s.wikipedia.org/w/api.php"
	ArXivAPIURL          = "https://export.arxiv.org/api/query"
	ArXivAbsURLTemplate  = "https://arxiv.org/abs/%s"
)

// ProviderBaseURL returns the base URL for a given provider name.
// Returns empty string if provider is unknown.
func ProviderBaseURL(provider string) string {
	switch provider {
	case ProviderDeepSeek:
		return BaseURLDeepSeek
	case ProviderZhipu:
		return BaseURLZhipu
	case ProviderOpenAI:
		return BaseURLOpenAI
	case ProviderOllama:
		return BaseURLOllama
	case ProviderOpenRouter:
		return BaseURLOpenRouter
	default:
		return ""
	}
}

// ProviderBaseURLWithVersion returns the versioned base URL for a given provider.
func ProviderBaseURLWithVersion(provider string) string {
	switch provider {
	case ProviderDeepSeek:
		return BaseURLDeepSeekV1
	case ProviderZhipu:
		return BaseURLZhipuAPI
	case ProviderOpenAI:
		return BaseURLOpenAI
	case ProviderOllama:
		return BaseURLOllamaV1
	case ProviderOpenRouter:
		return BaseURLOpenRouter
	default:
		return ""
	}
}

// OpenRouter HTTP headers
const (
	OpenRouterReferer = "https://go-deep-research.app"
	OpenRouterTitle   = "AI Research Platform"
)

// ProviderRequiresAPIKey returns whether a provider requires an API key.
func ProviderRequiresAPIKey(provider string) bool {
	return provider != ProviderOllama
}

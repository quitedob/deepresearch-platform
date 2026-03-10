package constant

import "time"

// Token / Session expiration
const (
	TokenExpirationSeconds = 86400                              // 24 hours
	TokenExpiration        = 24 * time.Hour
	SessionTimeout         = 3600 * time.Second                 // 1 hour
	CORSCacheMaxAge        = "86400"                            // 24 hours (string for HTTP header)
	DefaultCacheTTLSeconds = 3600                               // 1 hour
)

// Research limits
const (
	MaxResearchQueryLength = 10000
	MaxResearchSessions    = 100
	ResearchSSETimeout     = 30 * time.Minute
)

// Admin / Export limits
const (
	MaxExportSessions       = 100
	MaxMessagesPerSession   = 500
	AdminSSETimeout         = 30 * time.Minute
)

// Default model / provider
const (
	DefaultProvider = ProviderDeepSeek
	DefaultModel    = "deepseek-chat"
)

// Provider descriptions (for API info endpoints)
var ProviderDescriptions = map[string]string{
	ProviderDeepSeek:   "DeepSeek AI - 高性能大语言模型",
	ProviderZhipu:      "智谱AI - GLM系列大语言模型",
	ProviderOllama:     "Ollama - 本地部署的开源大语言模型",
	ProviderOpenAI:     "OpenAI兼容 - GLM Coding Plan",
	ProviderOpenRouter: "OpenRouter - 统一API访问400+模型",
}

// Provider icons
var ProviderIcons = map[string]string{
	ProviderDeepSeek:   "🚀",
	ProviderZhipu:      "🧠",
	ProviderOllama:     "🦙",
	ProviderOpenAI:     "🔮",
	ProviderOpenRouter: "🌐",
}

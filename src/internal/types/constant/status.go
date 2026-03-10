package constant

// 研究会话状态
const (
    ResearchStatusPlanning    = "planning"
    ResearchStatusExecuting   = "executing"
    ResearchStatusSynthesis   = "synthesis"
    ResearchStatusCompleted   = "completed"
    ResearchStatusFailed      = "failed"
)

// 研究任务状态
const (
    TaskStatusPending   = "pending"
    TaskStatusRunning   = "running"
    TaskStatusCompleted = "completed"
    TaskStatusFailed    = "failed"
)

// 任务类型
const (
    TaskTypeSearch    = "search"
    TaskTypeAnalyze   = "analyze"
    TaskTypeSynthesize = "synthesize"
)

// 研究类型
const (
    ResearchTypeDeep          = "deep"
    ResearchTypeQuick         = "quick"
    ResearchTypeAcademic      = "academic"
    ResearchTypeComprehensive = "comprehensive"
)

// 用户状态
const (
    UserStatusActive = "active"
    UserStatusBanned = "banned"
)

// 会员类型
const (
    MembershipFree    = "free"
    MembershipPremium = "premium"
)

// 用户角色
const (
    UserRoleUser = "user"
    UserRoleAssistant = "assistant"
    UserRoleSystem = "system"
)

// LLM提供商
const (
	ProviderDeepSeek   = "deepseek"
	ProviderZhipu      = "zhipu"
	ProviderOpenAI     = "openai"
	ProviderOllama     = "ollama"
	ProviderOpenRouter = "openrouter"
)

// 消息角色
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

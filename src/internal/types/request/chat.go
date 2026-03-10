package request

// CreateChatSessionRequest 创建聊天会话请求
type CreateChatSessionRequest struct {
    Title        string `json:"title,omitempty" binding:"max=200"`
    LLMProvider  string `json:"llm_provider" binding:"required,oneof=deepseek zhipu ollama openai openrouter"`
    ModelName    string `json:"model_name" binding:"required"`
    ModelType    string `json:"model_type,omitempty" binding:"omitempty,oneof=default deep research"` // 模型类型：default, deep, research
    SystemPrompt string `json:"system_prompt,omitempty" binding:"max=5000"`
}

// UpdateChatSessionRequest 更新聊天会话请求
type UpdateChatSessionRequest struct {
    Title        *string `json:"title,omitempty" binding:"omitempty,max=200"`
    SystemPrompt *string `json:"system_prompt,omitempty" binding:"omitempty,max=5000"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
    SessionID     string `json:"session_id" binding:"required,uuid"`
    Message       string `json:"message" binding:"required,min=1,max=10000"`
    Stream        bool   `json:"stream"`        // 是否使用流式输出
    UseWebSearch  bool   `json:"use_web_search"` // 是否使用网络搜索
    UseDeepThink  bool   `json:"use_deep_think"` // 是否使用深度思考
}

// GetChatSessionsRequest 获取聊天会话列表请求
type GetChatSessionsRequest struct {
    Limit  int `json:"limit" form:"limit" binding:"min=1,max=100"`
    Offset int `json:"offset" form:"offset" binding:"min=0"`
}

// GetChatMessagesRequest 获取聊天消息列表请求
type GetChatMessagesRequest struct {
    SessionID string `json:"session_id" binding:"required,uuid"`
    Limit     int    `json:"limit" form:"limit" binding:"min=1,max=100"`
    Offset    int    `json:"offset" form:"offset" binding:"min=0"`
}

// WebSearchChatRequest 联网搜索聊天请求
type WebSearchChatRequest struct {
    SessionID string `json:"session_id" binding:"required,uuid"`
    Message   string `json:"message" binding:"required,min=1,max=10000"`
    APIKey    string `json:"api_key,omitempty"` // 搜索API密钥
}

// ModelInfoRequest 模型信息请求
type ModelInfoRequest struct {
    Provider string `json:"provider,omitempty" form:"provider"`
}

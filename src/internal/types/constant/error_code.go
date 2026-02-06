package constant

// 错误代码
const (
    // 通用错误
    ErrCodeInternalError     = "INTERNAL_ERROR"
    ErrCodeInvalidInput      = "INVALID_INPUT"
    ErrCodeUnauthorized      = "UNAUTHORIZED"
    ErrCodeForbidden         = "FORBIDDEN"
    ErrCodeNotFound          = "NOT_FOUND"
    ErrCodeConflict          = "CONFLICT"
    ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"

    // 认证相关错误
    ErrCodeInvalidCredentials   = "INVALID_CREDENTIALS"
    ErrCodeTokenExpired        = "TOKEN_EXPIRED"
    ErrCodeInvalidToken        = "INVALID_TOKEN"
    ErrCodeUserExists          = "USER_EXISTS"
    ErrCodeUserNotFound        = "USER_NOT_FOUND"

    // 聊天相关错误
    ErrCodeSessionNotFound     = "SESSION_NOT_FOUND"
    ErrCodeMessageNotFound     = "MESSAGE_NOT_FOUND"
    ErrCodeProviderUnavailable = "PROVIDER_UNAVAILABLE"

    // 研究相关错误
    ErrCodeResearchNotFound    = "RESEARCH_NOT_FOUND"
    ErrCodeTaskFailed          = "TASK_FAILED"
    ErrCodeToolUnavailable     = "TOOL_UNAVAILABLE"

    // 数据库相关错误
    ErrCodeDatabaseError       = "DATABASE_ERROR"
    ErrCodeConnectionFailed    = "CONNECTION_FAILED"

    // 缓存相关错误
    ErrCodeCacheError          = "CACHE_ERROR"
    ErrCodeCacheUnavailable    = "CACHE_UNAVAILABLE"
)

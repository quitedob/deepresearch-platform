package errors

import (
	"fmt"
	"net/http"
)

// APIErrorCode API错误码类型（与ErrorCode区分）
type APIErrorCode string

// 定义API专用错误码（避免与errors.go中的常量冲突）
const (
	// 通用错误 (1xxx)
	APIErrCodeUnknown          APIErrorCode = "ERR_UNKNOWN"
	APIErrCodeInvalidRequest   APIErrorCode = "ERR_INVALID_REQUEST"
	APIErrCodeInvalidParameter APIErrorCode = "ERR_INVALID_PARAMETER"
	APIErrCodeMissingParameter APIErrorCode = "ERR_MISSING_PARAMETER"
	APIErrCodeInternalError    APIErrorCode = "ERR_INTERNAL_ERROR"
	APIErrCodeServiceUnavailable APIErrorCode = "ERR_SERVICE_UNAVAILABLE"

	// 认证授权错误 (2xxx)
	APIErrCodeUnauthorized     APIErrorCode = "ERR_UNAUTHORIZED"
	APIErrCodeForbidden        APIErrorCode = "ERR_FORBIDDEN"
	APIErrCodeTokenExpired     APIErrorCode = "ERR_TOKEN_EXPIRED"
	APIErrCodeTokenInvalid     APIErrorCode = "ERR_TOKEN_INVALID"
	APIErrCodeAdminRequired    APIErrorCode = "ERR_ADMIN_REQUIRED"

	// 资源错误 (3xxx)
	APIErrCodeNotFound         APIErrorCode = "ERR_NOT_FOUND"
	APIErrCodeAlreadyExists    APIErrorCode = "ERR_ALREADY_EXISTS"
	APIErrCodeConflict         APIErrorCode = "ERR_CONFLICT"

	// 配额限制错误 (4xxx)
	APIErrCodeQuotaExceeded    APIErrorCode = "ERR_QUOTA_EXCEEDED"
	APIErrCodeChatQuotaExceeded APIErrorCode = "ERR_CHAT_QUOTA_EXCEEDED"
	APIErrCodeResearchQuotaExceeded APIErrorCode = "ERR_RESEARCH_QUOTA_EXCEEDED"
	APIErrCodeRateLimitExceeded APIErrorCode = "ERR_RATE_LIMIT_EXCEEDED"

	// 会话错误 (5xxx)
	APIErrCodeSessionNotFound  APIErrorCode = "ERR_SESSION_NOT_FOUND"
	APIErrCodeSessionExpired   APIErrorCode = "ERR_SESSION_EXPIRED"
	APIErrCodeSessionForbidden APIErrorCode = "ERR_SESSION_FORBIDDEN"
	APIErrCodeContextOverflow  APIErrorCode = "ERR_CONTEXT_OVERFLOW"
	APIErrCodeMessageTooLong   APIErrorCode = "ERR_MESSAGE_TOO_LONG"

	// LLM错误 (6xxx)
	APIErrCodeLLMUnavailable   APIErrorCode = "ERR_LLM_UNAVAILABLE"
	APIErrCodeLLMTimeout       APIErrorCode = "ERR_LLM_TIMEOUT"
	APIErrCodeLLMError         APIErrorCode = "ERR_LLM_ERROR"
	APIErrCodeModelNotSupported APIErrorCode = "ERR_MODEL_NOT_SUPPORTED"
	APIErrCodeModelNotFound    APIErrorCode = "ERR_MODEL_NOT_FOUND"

	// 研究错误 (7xxx)
	APIErrCodeResearchNotFound APIErrorCode = "ERR_RESEARCH_NOT_FOUND"
	APIErrCodeResearchFailed   APIErrorCode = "ERR_RESEARCH_FAILED"
	APIErrCodeResearchTimeout  APIErrorCode = "ERR_RESEARCH_TIMEOUT"
	APIErrCodeQueryTooLong     APIErrorCode = "ERR_QUERY_TOO_LONG"

	// 用户错误 (8xxx)
	APIErrCodeUserNotFound     APIErrorCode = "ERR_USER_NOT_FOUND"
	APIErrCodeUserBanned       APIErrorCode = "ERR_USER_BANNED"
	APIErrCodeInvalidCredentials APIErrorCode = "ERR_INVALID_CREDENTIALS"
	APIErrCodeEmailExists      APIErrorCode = "ERR_EMAIL_EXISTS"
	APIErrCodeUsernameExists   APIErrorCode = "ERR_USERNAME_EXISTS"

	// 激活码错误 (9xxx)
	APIErrCodeActivationCodeInvalid APIErrorCode = "ERR_ACTIVATION_CODE_INVALID"
	APIErrCodeActivationCodeExpired APIErrorCode = "ERR_ACTIVATION_CODE_EXPIRED"
	APIErrCodeActivationCodeUsed    APIErrorCode = "ERR_ACTIVATION_CODE_USED"
)

// APIError 统一的API错误结构
type APIError struct {
	Code       APIErrorCode           `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	Field      string                 `json:"field,omitempty"`
	HTTPStatus int                    `json:"-"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

// Error 实现error接口
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WithDetails 添加详细信息
func (e *APIError) WithDetails(details string) *APIError {
	e.Details = details
	return e
}

// WithField 添加字段信息
func (e *APIError) WithField(field string) *APIError {
	e.Field = field
	return e
}

// WithExtra 添加额外信息
func (e *APIError) WithExtra(key string, value interface{}) *APIError {
	if e.Extra == nil {
		e.Extra = make(map[string]interface{})
	}
	e.Extra[key] = value
	return e
}

// ToResponse 转换为响应格式
func (e *APIError) ToResponse() map[string]interface{} {
	resp := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    e.Code,
			"message": e.Message,
		},
	}
	
	errorMap := resp["error"].(map[string]interface{})
	if e.Details != "" {
		errorMap["details"] = e.Details
	}
	if e.Field != "" {
		errorMap["field"] = e.Field
	}
	if e.Extra != nil {
		for k, v := range e.Extra {
			errorMap[k] = v
		}
	}
	
	return resp
}

// 预定义错误构造函数

// NewAPIUnauthorizedError 创建未授权错误
func NewAPIUnauthorizedError(message string) *APIError {
	return &APIError{
		Code:       APIErrCodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewAPIForbiddenError 创建禁止访问错误
func NewAPIForbiddenError(message string) *APIError {
	return &APIError{
		Code:       APIErrCodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// NewAPINotFoundError 创建资源不存在错误
func NewAPINotFoundError(resource string) *APIError {
	return &APIError{
		Code:       APIErrCodeNotFound,
		Message:    fmt.Sprintf("%s不存在", resource),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewSessionNotFoundError 创建会话不存在错误
func NewSessionNotFoundError() *APIError {
	return &APIError{
		Code:       APIErrCodeSessionNotFound,
		Message:    "会话不存在",
		HTTPStatus: http.StatusNotFound,
	}
}

// NewSessionForbiddenError 创建会话无权访问错误
func NewSessionForbiddenError() *APIError {
	return &APIError{
		Code:       APIErrCodeSessionForbidden,
		Message:    "无权访问此会话",
		HTTPStatus: http.StatusForbidden,
	}
}

// NewInvalidRequestError 创建无效请求错误
func NewInvalidRequestError(message string) *APIError {
	return &APIError{
		Code:       APIErrCodeInvalidRequest,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewInvalidParameterError 创建无效参数错误
func NewInvalidParameterError(field, message string) *APIError {
	return &APIError{
		Code:       APIErrCodeInvalidParameter,
		Message:    message,
		Field:      field,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewMissingParameterError 创建缺少参数错误
func NewMissingParameterError(field string) *APIError {
	return &APIError{
		Code:       APIErrCodeMissingParameter,
		Message:    fmt.Sprintf("缺少必要参数: %s", field),
		Field:      field,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewAPIInternalError 创建内部错误
func NewAPIInternalError(message string) *APIError {
	return &APIError{
		Code:       APIErrCodeInternalError,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

// NewChatQuotaExceededError 创建聊天配额超限错误
func NewChatQuotaExceededError(remaining, limit int) *APIError {
	return &APIError{
		Code:       APIErrCodeChatQuotaExceeded,
		Message:    "聊天配额已用完",
		HTTPStatus: http.StatusForbidden,
		Extra: map[string]interface{}{
			"remaining": remaining,
			"limit":     limit,
		},
	}
}

// NewResearchQuotaExceededError 创建研究配额超限错误
func NewResearchQuotaExceededError(remaining, limit int) *APIError {
	return &APIError{
		Code:       APIErrCodeResearchQuotaExceeded,
		Message:    "深度研究配额已用完",
		HTTPStatus: http.StatusForbidden,
		Extra: map[string]interface{}{
			"remaining": remaining,
			"limit":     limit,
		},
	}
}

// NewContextOverflowError 创建上下文溢出错误
func NewContextOverflowError(currentTokens, maxTokens int) *APIError {
	return &APIError{
		Code:       APIErrCodeContextOverflow,
		Message:    "上下文长度超出限制，建议创建新会话或清理历史消息",
		HTTPStatus: http.StatusBadRequest,
		Extra: map[string]interface{}{
			"current_tokens": currentTokens,
			"max_tokens":     maxTokens,
			"suggestion":     "create_new_session",
		},
	}
}

// NewMessageTooLongError 创建消息过长错误
func NewMessageTooLongError(length, maxLength int) *APIError {
	return &APIError{
		Code:       APIErrCodeMessageTooLong,
		Message:    fmt.Sprintf("消息长度超出限制，最大允许%d字符", maxLength),
		HTTPStatus: http.StatusBadRequest,
		Extra: map[string]interface{}{
			"length":     length,
			"max_length": maxLength,
		},
	}
}

// NewModelNotSupportedError 创建模型不支持错误
func NewModelNotSupportedError(modelName string) *APIError {
	return &APIError{
		Code:       APIErrCodeModelNotSupported,
		Message:    fmt.Sprintf("模型 %s 未注册或不支持", modelName),
		HTTPStatus: http.StatusBadRequest,
		Extra: map[string]interface{}{
			"model": modelName,
		},
	}
}

// NewLLMUnavailableError 创建LLM不可用错误
func NewLLMUnavailableError() *APIError {
	return &APIError{
		Code:       APIErrCodeLLMUnavailable,
		Message:    "LLM服务暂时不可用，请稍后重试",
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// NewLLMError 创建LLM调用错误
func NewLLMError(details string) *APIError {
	return &APIError{
		Code:       APIErrCodeLLMError,
		Message:    "LLM调用失败",
		Details:    details,
		HTTPStatus: http.StatusInternalServerError,
	}
}

// NewQueryTooLongError 创建查询过长错误
func NewQueryTooLongError(length, maxLength int) *APIError {
	return &APIError{
		Code:       APIErrCodeQueryTooLong,
		Message:    fmt.Sprintf("研究查询过长，最大允许%d字符", maxLength),
		HTTPStatus: http.StatusBadRequest,
		Extra: map[string]interface{}{
			"length":     length,
			"max_length": maxLength,
		},
	}
}

// NewAPIRateLimitExceededError 创建速率限制错误
func NewAPIRateLimitExceededError() *APIError {
	return &APIError{
		Code:       APIErrCodeRateLimitExceeded,
		Message:    "请求过于频繁，请稍后重试",
		HTTPStatus: http.StatusTooManyRequests,
	}
}

// NewActivationCodeInvalidError 创建激活码无效错误
func NewActivationCodeInvalidError() *APIError {
	return &APIError{
		Code:       APIErrCodeActivationCodeInvalid,
		Message:    "激活码无效或已过期",
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewUserBannedError 创建用户被禁用错误
func NewUserBannedError() *APIError {
	return &APIError{
		Code:       APIErrCodeUserBanned,
		Message:    "账户已被禁用，请联系管理员",
		HTTPStatus: http.StatusForbidden,
	}
}

// ==================== 增强的权限错误 ====================

// NewForbiddenWithResourceError 创建带资源类型的禁止访问错误
func NewForbiddenWithResourceError(resourceType, requiredPermission string) *APIError {
	return &APIError{
		Code:       APIErrCodeForbidden,
		Message:    fmt.Sprintf("无权访问此%s", resourceType),
		HTTPStatus: http.StatusForbidden,
		Extra: map[string]interface{}{
			"resource_type":       resourceType,
			"required_permission": requiredPermission,
		},
	}
}

// NewSessionForbiddenWithDetailsError 创建带详情的会话禁止访问错误
func NewSessionForbiddenWithDetailsError(sessionID, ownerID string) *APIError {
	return &APIError{
		Code:       APIErrCodeSessionForbidden,
		Message:    "无权访问此会话",
		HTTPStatus: http.StatusForbidden,
		Extra: map[string]interface{}{
			"resource_type":       "session",
			"required_permission": "owner",
			"session_id":          sessionID,
		},
	}
}

// NewAdminRequiredWithResourceError 创建带资源类型的管理员权限错误
func NewAdminRequiredWithResourceError(resourceType, action string) *APIError {
	return &APIError{
		Code:       APIErrCodeAdminRequired,
		Message:    fmt.Sprintf("需要管理员权限才能%s%s", action, resourceType),
		HTTPStatus: http.StatusForbidden,
		Extra: map[string]interface{}{
			"resource_type":       resourceType,
			"required_permission": "admin",
			"action":              action,
		},
	}
}

// ==================== 乐观锁错误 ====================

// APIErrCodeOptimisticLock 乐观锁冲突错误码
const APIErrCodeOptimisticLock APIErrorCode = "ERR_OPTIMISTIC_LOCK"

// APIErrCodeConcurrentModification 并发修改错误码
const APIErrCodeConcurrentModification APIErrorCode = "ERR_CONCURRENT_MODIFICATION"

// NewOptimisticLockError 创建乐观锁冲突错误
func NewOptimisticLockError(resourceType, resourceID string) *APIError {
	return &APIError{
		Code:       APIErrCodeOptimisticLock,
		Message:    "数据已被其他操作修改，请刷新后重试",
		HTTPStatus: http.StatusConflict,
		Extra: map[string]interface{}{
			"resource_type": resourceType,
			"resource_id":   resourceID,
			"suggestion":    "refresh_and_retry",
		},
	}
}

// NewConcurrentModificationError 创建并发修改错误
func NewConcurrentModificationError(resourceType string) *APIError {
	return &APIError{
		Code:       APIErrCodeConcurrentModification,
		Message:    fmt.Sprintf("%s正在被其他操作修改，请稍后重试", resourceType),
		HTTPStatus: http.StatusConflict,
		Extra: map[string]interface{}{
			"resource_type": resourceType,
			"suggestion":    "wait_and_retry",
		},
	}
}

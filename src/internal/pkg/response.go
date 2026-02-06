package pkg

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apierrors "github.com/ai-research-platform/internal/pkg/errors"
)

// Response 统一响应格式
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
    c.JSON(200, Response{
        Code:    0,
        Message: "success",
        Data:    data,
    })
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
    c.JSON(code, Response{
        Code:    code,
        Message: message,
    })
}

// BadRequest 400错误
func BadRequest(c *gin.Context, message string) {
    Error(c, 400, message)
}

// Unauthorized 401错误
func Unauthorized(c *gin.Context, message string) {
    Error(c, 401, message)
}

// Forbidden 403错误
func Forbidden(c *gin.Context, message string) {
    Error(c, 403, message)
}

// NotFound 404错误
func NotFound(c *gin.Context, message string) {
    Error(c, 404, message)
}

// InternalError 500错误
func InternalError(c *gin.Context, message string) {
    Error(c, 500, message)
}

// APIErrorResponse 返回统一的API错误响应
// 使用 apierrors.APIError 结构体返回详细的错误信息
func APIErrorResponse(c *gin.Context, err *apierrors.APIError) {
	c.JSON(err.HTTPStatus, err.ToResponse())
}

// APIErrorFromError 从普通error创建API错误响应
func APIErrorFromError(c *gin.Context, httpStatus int, code apierrors.APIErrorCode, err error) {
	apiErr := &apierrors.APIError{
		Code:       code,
		Message:    err.Error(),
		HTTPStatus: httpStatus,
	}
	c.JSON(httpStatus, apiErr.ToResponse())
}

// GenerateUUID 生成UUID
func GenerateUUID() string {
	return uuid.New().String()
}

// PaginatedResponse 统一分页响应格式
type PaginatedResponse struct {
	Code     int         `json:"code"`
	Message  string      `json:"message"`
	Data     interface{} `json:"data"`
	Total    int64       `json:"total"`
	Limit    int         `json:"limit"`
	Offset   int         `json:"offset"`
	HasMore  bool        `json:"has_more"`
	MaxLimit int         `json:"max_limit,omitempty"`
}

// SuccessPaginated 分页成功响应
func SuccessPaginated(c *gin.Context, data interface{}, total int64, limit, offset, maxLimit int) {
	hasMore := int64(offset+limit) < total
	c.JSON(200, PaginatedResponse{
		Code:     0,
		Message:  "success",
		Data:     data,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		HasMore:  hasMore,
		MaxLimit: maxLimit,
	})
}

// ValidatePagination 验证并规范化分页参数
func ValidatePagination(limit, offset, maxLimit int) (int, int) {
	if limit < 1 {
		limit = 20
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// SuccessResponse 统一成功响应格式
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse 统一错误响应格式
type ErrorResponse struct {
	Success bool       `json:"success"`
	Error   ErrorInfo  `json:"error"`
}

// ErrorInfo 错误详情
type ErrorInfo struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details string                 `json:"details,omitempty"`
	Field   string                 `json:"field,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

// SuccessWithData 返回带数据的成功响应
func SuccessWithData(c *gin.Context, data interface{}) {
	c.JSON(200, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMessage 返回带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string) {
	c.JSON(200, SuccessResponse{
		Success: true,
		Message: message,
	})
}

// ErrorWithCode 返回带错误码的错误响应
func ErrorWithCode(c *gin.Context, httpStatus int, code, message string) {
	c.JSON(httpStatus, ErrorResponse{
		Success: false,
		Error: ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorWithDetails 返回带详情的错误响应
func ErrorWithDetails(c *gin.Context, httpStatus int, code, message, details string) {
	c.JSON(httpStatus, ErrorResponse{
		Success: false,
		Error: ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// ErrorWithField 返回带字段信息的错误响应
func ErrorWithField(c *gin.Context, httpStatus int, code, message, field string) {
	c.JSON(httpStatus, ErrorResponse{
		Success: false,
		Error: ErrorInfo{
			Code:    code,
			Message: message,
			Field:   field,
		},
	})
}

// ErrorWithExtra 返回带额外信息的错误响应
func ErrorWithExtra(c *gin.Context, httpStatus int, code, message string, extra map[string]interface{}) {
	c.JSON(httpStatus, ErrorResponse{
		Success: false,
		Error: ErrorInfo{
			Code:    code,
			Message: message,
			Extra:   extra,
		},
	})
}

// ==================== 统一响应格式 V2 ====================
// 所有API应使用以下格式，确保前后端一致性

// UnifiedResponse 统一响应格式（推荐使用）
type UnifiedResponse struct {
	Success bool        `json:"success"`
	Code    string      `json:"code,omitempty"`    // 错误码，成功时为空
	Message string      `json:"message,omitempty"` // 消息
	Data    interface{} `json:"data,omitempty"`    // 数据
	Error   *ErrorInfo  `json:"error,omitempty"`   // 错误详情，成功时为空
}

// OK 返回成功响应（无数据）
func OK(c *gin.Context) {
	c.JSON(200, UnifiedResponse{
		Success: true,
		Message: "操作成功",
	})
}

// OKWithData 返回成功响应（带数据）
func OKWithData(c *gin.Context, data interface{}) {
	c.JSON(200, UnifiedResponse{
		Success: true,
		Data:    data,
	})
}

// OKWithMessage 返回成功响应（带消息）
func OKWithMessage(c *gin.Context, message string) {
	c.JSON(200, UnifiedResponse{
		Success: true,
		Message: message,
	})
}

// OKWithDataAndMessage 返回成功响应（带数据和消息）
func OKWithDataAndMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(200, UnifiedResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Created 返回创建成功响应
func Created(c *gin.Context, data interface{}) {
	c.JSON(201, UnifiedResponse{
		Success: true,
		Message: "创建成功",
		Data:    data,
	})
}

// Fail 返回失败响应
func Fail(c *gin.Context, httpStatus int, code, message string) {
	c.JSON(httpStatus, UnifiedResponse{
		Success: false,
		Code:    code,
		Message: message,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// FailWithDetails 返回带详情的失败响应
func FailWithDetails(c *gin.Context, httpStatus int, code, message, details string) {
	c.JSON(httpStatus, UnifiedResponse{
		Success: false,
		Code:    code,
		Message: message,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// FailWithExtra 返回带额外信息的失败响应
func FailWithExtra(c *gin.Context, httpStatus int, code, message string, extra map[string]interface{}) {
	c.JSON(httpStatus, UnifiedResponse{
		Success: false,
		Code:    code,
		Message: message,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Extra:   extra,
		},
	})
}

// ==================== 常用错误响应快捷方法 ====================

// BadRequestError 400错误
func BadRequestError(c *gin.Context, message string) {
	Fail(c, 400, "ERR_INVALID_REQUEST", message)
}

// UnauthorizedError 401错误
func UnauthorizedError(c *gin.Context, message string) {
	Fail(c, 401, "ERR_UNAUTHORIZED", message)
}

// ForbiddenError 403错误
func ForbiddenError(c *gin.Context, message string) {
	Fail(c, 403, "ERR_FORBIDDEN", message)
}

// NotFoundError 404错误
func NotFoundError(c *gin.Context, message string) {
	Fail(c, 404, "ERR_NOT_FOUND", message)
}

// InternalServerError 500错误
func InternalServerError(c *gin.Context, message string) {
	Fail(c, 500, "ERR_INTERNAL_ERROR", message)
}

// SessionNotFoundError 会话不存在错误
func SessionNotFoundError(c *gin.Context) {
	Fail(c, 404, "ERR_SESSION_NOT_FOUND", "会话不存在或已被删除")
}

// SessionForbiddenError 无权访问会话错误
func SessionForbiddenError(c *gin.Context) {
	Fail(c, 403, "ERR_SESSION_FORBIDDEN", "无权访问此会话")
}

// QuotaExceededError 配额超限错误
func QuotaExceededError(c *gin.Context, quotaType string, remaining, limit int) {
	FailWithExtra(c, 403, "ERR_"+quotaType+"_QUOTA_EXCEEDED", quotaType+"配额已用完", map[string]interface{}{
		"remaining": remaining,
		"limit":     limit,
	})
}

// ModelNotSupportedError 模型不支持错误
func ModelNotSupportedError(c *gin.Context, modelName string) {
	FailWithExtra(c, 400, "ERR_MODEL_NOT_SUPPORTED", "模型不支持或未注册", map[string]interface{}{
		"model": modelName,
	})
}

// LLMUnavailableError LLM服务不可用错误
func LLMUnavailableError(c *gin.Context) {
	Fail(c, 503, "ERR_LLM_UNAVAILABLE", "LLM服务暂时不可用，请稍后重试")
}

// ==================== 分页响应 V2 ====================

// PaginatedData 分页数据
type PaginatedData struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Limit    int         `json:"limit"`
	Offset   int         `json:"offset"`
	HasMore  bool        `json:"has_more"`
	MaxLimit int         `json:"max_limit,omitempty"`
}

// OKPaginated 返回分页成功响应
func OKPaginated(c *gin.Context, items interface{}, total int64, limit, offset, maxLimit int) {
	hasMore := int64(offset+limit) < total
	c.JSON(200, UnifiedResponse{
		Success: true,
		Data: PaginatedData{
			Items:    items,
			Total:    total,
			Limit:    limit,
			Offset:   offset,
			HasMore:  hasMore,
			MaxLimit: maxLimit,
		},
	})
}

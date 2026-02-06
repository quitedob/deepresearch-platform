package errors

import (
	"fmt"
	"net/http"
)

// Error codes
const (
	ErrCodeInvalidInput      = "INVALID_INPUT"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeProviderFailed    = "PROVIDER_FAILED"
	ErrCodeDatabaseError     = "DATABASE_ERROR"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeTimeout           = "TIMEOUT"
	ErrCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	ErrCodeValidationFailed  = "VALIDATION_FAILED"
	ErrCodeConflict          = "CONFLICT"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// AppError represents an application error with additional context
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	StatusCode int    `json:"-"`
	Err        error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError
func NewAppError(code, message string, statusCode int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// WithDetails adds details to an AppError
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// Common error constructors

// NewInvalidInputError creates an invalid input error
func NewInvalidInputError(message string, err error) *AppError {
	return NewAppError(ErrCodeInvalidInput, message, http.StatusBadRequest, err)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(message string, err error) *AppError {
	return NewAppError(ErrCodeUnauthorized, message, http.StatusUnauthorized, err)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(message string, err error) *AppError {
	return NewAppError(ErrCodeForbidden, message, http.StatusForbidden, err)
}

// NewNotFoundError creates a not found error
func NewNotFoundError(message string, err error) *AppError {
	return NewAppError(ErrCodeNotFound, message, http.StatusNotFound, err)
}

// NewProviderFailedError creates a provider failed error
func NewProviderFailedError(message string, err error) *AppError {
	return NewAppError(ErrCodeProviderFailed, message, http.StatusBadGateway, err)
}

// NewDatabaseError creates a database error
func NewDatabaseError(message string, err error) *AppError {
	return NewAppError(ErrCodeDatabaseError, message, http.StatusInternalServerError, err)
}

// NewInternalError creates an internal error
func NewInternalError(message string, err error) *AppError {
	return NewAppError(ErrCodeInternalError, message, http.StatusInternalServerError, err)
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(message string, err error) *AppError {
	return NewAppError(ErrCodeTimeout, message, http.StatusGatewayTimeout, err)
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(message string, err error) *AppError {
	return NewAppError(ErrCodeRateLimitExceeded, message, http.StatusTooManyRequests, err)
}

// NewValidationError creates a validation error
func NewValidationError(message string, err error) *AppError {
	return NewAppError(ErrCodeValidationFailed, message, http.StatusBadRequest, err)
}

// NewConflictError creates a conflict error
func NewConflictError(message string, err error) *AppError {
	return NewAppError(ErrCodeConflict, message, http.StatusConflict, err)
}

// NewServiceUnavailableError creates a service unavailable error
func NewServiceUnavailableError(message string, err error) *AppError {
	return NewAppError(ErrCodeServiceUnavailable, message, http.StatusServiceUnavailable, err)
}

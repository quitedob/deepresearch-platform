# Error Handling Package

This package provides a structured error handling system for the AI Research Platform.

## Overview

The error handling system provides:
- Structured error types with error codes
- HTTP status code mapping
- Error context and details
- Integration with logging middleware

## Error Types

### AppError

The `AppError` type is the core error structure:

```go
type AppError struct {
    Code       string // Error code (e.g., "INVALID_INPUT")
    Message    string // Human-readable message
    Details    string // Additional details (optional)
    StatusCode int    // HTTP status code
    Err        error  // Underlying error (optional)
}
```

### Error Codes

The following error codes are defined:

- `ErrCodeInvalidInput` - Invalid input validation (400)
- `ErrCodeUnauthorized` - Authentication required (401)
- `ErrCodeForbidden` - Insufficient permissions (403)
- `ErrCodeNotFound` - Resource not found (404)
- `ErrCodeConflict` - Resource conflict (409)
- `ErrCodeValidationFailed` - Validation failed (400)
- `ErrCodeRateLimitExceeded` - Rate limit exceeded (429)
- `ErrCodeInternalError` - Internal server error (500)
- `ErrCodeDatabaseError` - Database operation failed (500)
- `ErrCodeProviderFailed` - External provider failed (502)
- `ErrCodeServiceUnavailable` - Service unavailable (503)
- `ErrCodeTimeout` - Request timeout (504)

## Usage

### Creating Errors

Use the constructor functions to create errors:

```go
import "github.com/ai-research-platform/pkg/errors"

// Invalid input
err := errors.NewInvalidInputError("Email is required", nil)

// Not found
err := errors.NewNotFoundError("User not found", nil)

// With underlying error
err := errors.NewDatabaseError("Failed to save user", dbErr)

// With details
err := errors.NewInvalidInputError("Validation failed", nil).
    WithDetails("Field 'email' must be a valid email address")
```

### In Handlers

Return errors from handlers using `c.Error()`:

```go
func (h *Handler) GetUser(c *gin.Context) {
    user, err := h.service.GetUser(userID)
    if err != nil {
        c.Error(errors.NewNotFoundError("User not found", err))
        return
    }
    c.JSON(http.StatusOK, user)
}
```

### Error Response Format

Errors are automatically formatted by the error handling middleware:

```json
{
    "code": "NOT_FOUND",
    "message": "User not found",
    "details": "User with ID 123 does not exist"
}
```

## Integration with Logging

The error handling middleware automatically logs errors with appropriate severity:

- 4xx errors: Logged as warnings
- 5xx errors: Logged as errors with stack traces
- Panics: Logged as errors with full stack traces

## Best Practices

1. **Use specific error constructors**: Use the appropriate constructor for the error type
2. **Include context**: Add details to help with debugging
3. **Wrap underlying errors**: Pass the underlying error when available
4. **Don't expose internal details**: Keep error messages user-friendly
5. **Log at the right level**: Let the middleware handle logging based on severity

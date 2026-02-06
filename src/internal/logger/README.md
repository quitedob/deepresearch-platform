# Logger Package

This package provides structured logging with sensitive data redaction for the AI Research Platform.

## Features

- Structured logging using Zap
- Automatic sensitive data redaction
- Multiple log levels (debug, info, warn, error, fatal)
- JSON and console output formats
- Stack trace capture for errors
- Critical error alerting support

## Configuration

Initialize the logger with a configuration:

```go
import "github.com/ai-research-platform/internal/logger"

config := logger.Config{
    Level:      "info",        // debug, info, warn, error
    Format:     "json",        // json or console
    OutputPath: "stdout",      // stdout or file path
}

err := logger.Initialize(config)
if err != nil {
    log.Fatal(err)
}
```

## Basic Usage

### Standard Logging

```go
import (
    "github.com/ai-research-platform/internal/logger"
    "go.uber.org/zap"
)

// Info level
logger.Info("User logged in", zap.String("user_id", userID))

// Debug level
logger.Debug("Processing request", zap.String("path", path))

// Warning level
logger.Warn("Rate limit approaching", zap.Int("requests", count))

// Error level
logger.Error("Failed to save data", zap.Error(err))
```

### Error Logging with Stack Traces

```go
// Log error with automatic stack trace
logger.ErrorWithStack("Database operation failed", err,
    zap.String("operation", "insert"),
    zap.String("table", "users"),
)

// Log critical error (triggers alerts in production)
logger.CriticalError("Payment processing failed", err,
    zap.String("transaction_id", txID),
    zap.Float64("amount", amount),
)
```

## Sensitive Data Redaction

The logger automatically redacts sensitive information to prevent leaking credentials or secrets.

### Automatic Field Redaction

Use `SafeString` for fields that might contain sensitive data:

```go
// This will redact if fieldName is sensitive
logger.Info("User action",
    logger.SafeString("password", userInput),  // Will be [REDACTED]
    logger.SafeString("username", username),   // Will be logged normally
)
```

### Sensitive Field Names

The following field names are automatically redacted:
- password, passwd, pwd
- token, jwt, bearer
- api_key, apikey
- secret, private_key
- authorization, auth

### Pattern-Based Redaction

Sensitive patterns in strings are automatically redacted:

```go
// These patterns will be redacted
message := "password=secret123"
redacted := logger.RedactSensitiveData(message)
// Result: "password=[REDACTED]"

message := "Authorization: Bearer token123"
redacted := logger.RedactSensitiveData(message)
// Result: "Authorization: [REDACTED]"
```

### Manual Redaction

Check if a field is sensitive:

```go
if logger.IsSensitiveField("password") {
    // Handle sensitive field
}
```

Create redacted fields:

```go
// Automatically redacts if key is sensitive
field := logger.RedactField("api_key", apiKey)
logger.Info("API call", field)
```

## Log Output Format

### JSON Format (Production)

```json
{
    "level": "error",
    "ts": "2024-01-15T10:30:45.123Z",
    "caller": "handler/user.go:45",
    "msg": "Failed to create user",
    "error": "database connection failed",
    "user_id": "123",
    "stack": "..."
}
```

### Console Format (Development)

```
2024-01-15T10:30:45.123Z  ERROR  handler/user.go:45  Failed to create user
    error: database connection failed
    user_id: 123
```

## Integration with Middleware

The logging middleware automatically logs all HTTP requests:

```go
import (
    "github.com/ai-research-platform/internal/handler"
    "github.com/ai-research-platform/internal/logger"
)

router := gin.New()
router.Use(handler.LoggingMiddleware(logger.Log))
```

Request logs include:
- HTTP method and path
- Status code
- Request duration
- Client IP
- User agent
- User ID (if authenticated)
- Errors (if any)

## Best Practices

1. **Use structured fields**: Always use zap fields instead of string formatting
   ```go
   // Good
   logger.Info("User created", zap.String("user_id", id))
   
   // Bad
   logger.Info(fmt.Sprintf("User created: %s", id))
   ```

2. **Redact sensitive data**: Use `SafeString` for user input that might contain secrets
   ```go
   logger.Info("Request received", logger.SafeString("header", headerValue))
   ```

3. **Include context**: Add relevant fields to help with debugging
   ```go
   logger.Error("Operation failed",
       zap.Error(err),
       zap.String("operation", "create_user"),
       zap.String("user_id", userID),
   )
   ```

4. **Use appropriate levels**:
   - Debug: Detailed information for debugging
   - Info: General informational messages
   - Warn: Warning messages for potentially harmful situations
   - Error: Error messages for failures
   - Fatal: Critical errors that require immediate shutdown

5. **Don't log in hot paths**: Avoid excessive logging in performance-critical code

6. **Flush on shutdown**: Always call `logger.Sync()` before application exit
   ```go
   defer logger.Sync()
   ```

## Testing

The logger package includes property-based tests to verify:
- Error logging completeness (stack traces, context, error details)
- Sensitive data redaction (field names and patterns)

Run tests:
```bash
go test ./internal/logger/...
```

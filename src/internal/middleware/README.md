# Middleware Package

This package provides HTTP middleware for the AI Research Platform using Gin framework.

## Components

### Authentication Middleware (`auth.go`)

- **AuthMiddleware**: Validates JWT tokens from Authorization headers
- **GetUserID**: Extracts user ID from Gin context
- **RequireAuth**: Helper to ensure user is authenticated

### Authorization Middleware (`authorization.go`)

- **AuthorizeResource**: Checks if authenticated user owns a resource
- **ResourceOwnerChecker**: Interface for checking resource ownership
- **ChatSessionOwnerChecker**: Checks chat session ownership
- **ResearchSessionOwnerChecker**: Checks research session ownership

### Rate Limiting Middleware (`rate_limit.go`)

- **RateLimiter**: Token bucket rate limiter
- **RateLimitMiddleware**: Limits requests per user/IP

## Usage

### Authentication

```go
import (
    "github.com/ai-research-platform/internal/auth"
    "github.com/ai-research-platform/internal/middleware"
    "github.com/gin-gonic/gin"
)

router := gin.Default()
jwtManager := auth.NewJWTManager("secret", 24*time.Hour)

// Apply authentication middleware
router.Use(middleware.AuthMiddleware(jwtManager))

// Protected route
router.GET("/profile", func(c *gin.Context) {
    userID, _ := middleware.GetUserID(c)
    // Use userID
})
```

### Authorization

```go
// Create owner checker
getSessionOwner := func(ctx context.Context, sessionID string) (string, error) {
    // Query database for session owner
    return ownerID, nil
}

checker := middleware.NewChatSessionOwnerChecker(getSessionOwner)

// Apply authorization middleware
router.GET("/sessions/:id", 
    middleware.AuthMiddleware(jwtManager),
    middleware.AuthorizeResource(checker, "id"),
    func(c *gin.Context) {
        // User is authenticated and owns the resource
    })
```

### Rate Limiting

```go
limiter := middleware.NewRateLimiter(100) // 100 requests per minute

router.Use(limiter.RateLimitMiddleware())
```

## Testing

The package includes comprehensive property-based tests:

- **Property 37**: Resource authorization enforcement (Requirements 12.4)

Run tests:
```bash
go test ./internal/middleware/...
```

## Features

### Authentication
- Bearer token validation
- User identity extraction
- Automatic 401 responses for invalid tokens

### Authorization
- Resource ownership verification
- Flexible checker interface
- Automatic 403 responses for unauthorized access

### Rate Limiting
- Per-user rate limiting
- IP-based limiting for unauthenticated requests
- Token bucket algorithm with automatic refill
- Automatic cleanup of old buckets

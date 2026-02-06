# Developer Setup Guide

## Overview

This guide will help you set up a local development environment for the AI Research Platform. The platform is built with Go and uses PostgreSQL and Redis for data storage and caching.

## Prerequisites

### Required Software

1. **Go 1.21 or higher**
   - Download from: https://golang.org/dl/
   - Verify installation: `go version`

2. **PostgreSQL 14 or higher**
   - Download from: https://www.postgresql.org/download/
   - Or use Docker: `docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:14`
   - Verify installation: `psql --version`

3. **Redis 6 or higher**
   - Download from: https://redis.io/download
   - Or use Docker: `docker run -d -p 6379:6379 redis:6`
   - Verify installation: `redis-cli --version`

4. **Git**
   - Download from: https://git-scm.com/downloads
   - Verify installation: `git --version`

### Optional Software

1. **Docker and Docker Compose**
   - For containerized development
   - Download from: https://www.docker.com/products/docker-desktop

2. **Make**
   - For using Makefile commands
   - Usually pre-installed on Linux/Mac
   - Windows: Install via Chocolatey (`choco install make`)

3. **VS Code or GoLand**
   - Recommended IDEs for Go development

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/ai-research-platform.git
cd ai-research-platform
```

### 2. Install Dependencies

```bash
# Download Go dependencies
go mod download

# Or use Make
make deps
```

### 3. Set Up Database

#### Option A: Using Docker Compose (Recommended)

```bash
# Start PostgreSQL and Redis
docker-compose up -d postgres redis

# Wait for services to be ready
sleep 5

# Run database migrations
make migrate
```

#### Option B: Manual Setup

```bash
# Create database
createdb ai_research_platform

# Or using psql
psql -U postgres -c "CREATE DATABASE ai_research_platform;"

# Run initialization script
psql -U postgres -d ai_research_platform -f scripts/init-db.sql
```

### 4. Configure Environment

```bash
# Copy example configuration
cp config.yaml config.local.yaml

# Edit config.local.yaml with your settings
# At minimum, update:
# - database credentials
# - LLM provider API keys
# - JWT secret
```

**Important Configuration Values:**

```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: your_password
  db_name: ai_research_platform

redis:
  host: localhost
  port: 6379
  password: ""

llm:
  providers:
    deepseek:
      api_key: "your_deepseek_api_key"
    openai:
      api_key: "your_openai_api_key"

security:
  jwt_secret: "change-this-to-a-secure-random-string"
```

### 5. Run the Application

```bash
# Using Make
make run

# Or directly with Go
go run cmd/server/main.go -config config.local.yaml

# Or build and run
make build
./bin/ai-research-platform -config config.local.yaml
```

The server will start on `http://localhost:8080`

### 6. Verify Installation

```bash
# Check health endpoint
curl http://localhost:8080/health

# Expected response: {"status":"healthy"}

# Check readiness endpoint
curl http://localhost:8080/ready

# Expected response: {"status":"ready"}
```

## Development Workflow

### Project Structure

```
ai-research-platform/
├── cmd/
│   ├── server/              # Main application entry point
│   ├── arxiv-demo/          # ArXiv tool demo
│   └── wiki-demo/           # Wikipedia tool demo
├── internal/
│   ├── auth/                # Authentication (JWT, password hashing)
│   ├── cache/               # Cache implementations (memory, Redis)
│   ├── config/              # Configuration management
│   ├── database/            # Database connection and initialization
│   ├── eino/                # Eino framework components
│   │   ├── *_provider.go    # LLM provider implementations
│   │   ├── *_tool.go        # Research tool implementations
│   │   ├── llm_scheduler.go # Provider scheduling and fallback
│   │   ├── mcp_handler.go   # Model Context Protocol
│   │   └── research_workflow.go # Research orchestration
│   ├── handler/             # HTTP handlers and routing
│   ├── logger/              # Structured logging
│   ├── middleware/          # HTTP middleware (auth, rate limit)
│   ├── models/              # Database models
│   ├── monitoring/          # Metrics and tracing
│   ├── repository/          # Data access layer
│   └── service/             # Business logic layer
├── pkg/
│   ├── errors/              # Custom error types
│   └── utils/               # Utility functions
├── test/
│   └── integration/         # Integration tests
├── k8s/                     # Kubernetes manifests
├── scripts/                 # Utility scripts
├── docs/                    # Documentation
├── config.yaml              # Default configuration
├── docker-compose.yml       # Docker Compose configuration
├── Dockerfile               # Docker image definition
├── Makefile                 # Build and development commands
└── go.mod                   # Go module definition
```

### Available Make Commands

```bash
# Development
make deps              # Download dependencies
make build             # Build the application
make run               # Run the application
make clean             # Clean build artifacts

# Testing
make test              # Run all tests
make test-unit         # Run unit tests only
make test-integration  # Run integration tests
make test-coverage     # Run tests with coverage report
make test-verbose      # Run tests with verbose output

# Code Quality
make fmt               # Format code with gofmt
make lint              # Run golangci-lint
make vet               # Run go vet
make check             # Run fmt, vet, and lint

# Docker
make docker-build      # Build Docker image
make docker-run        # Run Docker container
make docker-compose-up # Start all services with Docker Compose
make docker-compose-down # Stop all services

# Kubernetes
make k8s-deploy        # Deploy to Kubernetes
make k8s-delete        # Remove from Kubernetes
make k8s-status        # Check deployment status
make test-k8s          # Test Kubernetes deployment

# Database
make migrate           # Run database migrations
make migrate-down      # Rollback migrations
make db-reset          # Reset database (drop and recreate)

# Help
make help              # Show all available commands
```

### Running Tests

#### Unit Tests

```bash
# Run all unit tests
make test-unit

# Run tests for a specific package
go test -v ./internal/config/...
go test -v ./internal/auth/...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
make test-coverage
# Opens coverage.html in browser
```

#### Property-Based Tests

The project uses property-based testing with `gopter` for validating correctness properties:

```bash
# Run property tests
go test -v ./internal/config/ -run Property
go test -v ./internal/logger/ -run Property

# Run all property tests
go test -v ./... -run Property
```

#### Integration Tests

```bash
# Run integration tests (requires running services)
make test-integration

# Or manually
go test -v ./test/integration/...
```

### Code Style and Linting

The project follows standard Go conventions and uses `golangci-lint` for linting.

```bash
# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet

# Run all checks
make check
```

**Pre-commit Checklist:**
- [ ] Code is formatted (`make fmt`)
- [ ] No linting errors (`make lint`)
- [ ] All tests pass (`make test`)
- [ ] Coverage is maintained or improved
- [ ] Documentation is updated

### Adding New Features

#### 1. Create Feature Branch

```bash
git checkout -b feature/your-feature-name
```

#### 2. Implement Feature

Follow the layered architecture:

1. **Models** (`internal/models/`) - Define data structures
2. **Repository** (`internal/repository/`) - Data access layer
3. **Service** (`internal/service/`) - Business logic
4. **Handler** (`internal/handler/`) - HTTP endpoints
5. **Tests** - Unit and integration tests

#### 3. Write Tests

```go
// Unit test example
func TestYourFeature(t *testing.T) {
    // Arrange
    service := NewYourService(...)
    
    // Act
    result, err := service.DoSomething(...)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}

// Property test example
func TestProperty_YourFeature(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("your property description",
        prop.ForAll(
            func(input string) bool {
                result := YourFunction(input)
                return ValidateProperty(result)
            },
            gen.AnyString(),
        ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

#### 4. Update Documentation

- Update API documentation if adding new endpoints
- Update README if changing setup process
- Add inline code comments for complex logic

#### 5. Submit Pull Request

```bash
# Commit changes
git add .
git commit -m "feat: add your feature description"

# Push to remote
git push origin feature/your-feature-name

# Create pull request on GitHub
```

### Debugging

#### Using VS Code

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/server",
      "args": ["-config", "config.local.yaml"],
      "env": {
        "GO_ENV": "development"
      }
    }
  ]
}
```

#### Using Delve (CLI)

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug application
dlv debug cmd/server/main.go -- -config config.local.yaml

# Set breakpoint
(dlv) break main.main
(dlv) continue
```

#### Logging

Enable debug logging in `config.local.yaml`:

```yaml
logging:
  level: debug  # Change from info to debug
  format: json
  output_path: stdout
```

### Working with LLM Providers

#### Testing Providers

```bash
# Test DeepSeek provider
curl -X POST http://localhost:8080/api/v1/llm/test \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "deepseek",
    "model": "deepseek-chat",
    "messages": ["Hello!"]
  }'

# Check provider metrics
curl http://localhost:8080/api/v1/llm/providers \
  -H "Authorization: Bearer <token>"
```

#### Adding New Provider

1. Create provider implementation in `internal/eino/`:

```go
// internal/eino/newprovider_provider.go
type NewProviderChatModel struct {
    apiKey  string
    baseURL string
    model   string
}

func NewNewProviderChatModel(apiKey, baseURL, model string) *NewProviderChatModel {
    return &NewProviderChatModel{
        apiKey:  apiKey,
        baseURL: baseURL,
        model:   model,
    }
}

func (m *NewProviderChatModel) Generate(ctx context.Context, messages []*Message) (*Message, error) {
    // Implementation
}

func (m *NewProviderChatModel) Stream(ctx context.Context, messages []*Message) (<-chan *StreamChunk, error) {
    // Implementation
}
```

2. Register in provider factory:

```go
// internal/eino/provider_factory.go
func CreateChatModel(provider, model, apiKey, baseURL string) (ChatModel, error) {
    switch provider {
    case "newprovider":
        return NewNewProviderChatModel(apiKey, baseURL, model), nil
    // ... other providers
    }
}
```

3. Add configuration:

```yaml
# config.yaml
llm:
  providers:
    newprovider:
      api_key: "your_api_key"
      base_url: "https://api.newprovider.com"
      models:
        - model-name-1
        - model-name-2
```

4. Write tests:

```go
// internal/eino/newprovider_provider_test.go
func TestNewProviderChatModel_Generate(t *testing.T) {
    // Test implementation
}
```

### Working with Research Tools

#### Testing Tools

```bash
# Run tool demos
go run cmd/arxiv-demo/main.go
go run cmd/wiki-demo/main.go
```

#### Adding New Tool

1. Create tool implementation in `internal/eino/`:

```go
// internal/eino/newtool_tool.go
type NewTool struct {
    name        string
    description string
    apiKey      string
}

func NewNewTool(apiKey string) *NewTool {
    return &NewTool{
        name:        "new_tool",
        description: "Description of what the tool does",
        apiKey:      apiKey,
    }
}

func (t *NewTool) Name() string {
    return t.name
}

func (t *NewTool) Description() string {
    return t.description
}

func (t *NewTool) Execute(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
    // Implementation
}
```

2. Register in tool registry:

```go
// internal/eino/tool_registry.go
func NewToolRegistry() *ToolRegistry {
    registry := &ToolRegistry{
        tools: make(map[string]Tool),
    }
    
    registry.Register(NewWebSearchTool())
    registry.Register(NewWikipediaTool())
    registry.Register(NewArxivTool())
    registry.Register(NewNewTool("api_key")) // Add new tool
    
    return registry
}
```

3. Write tests:

```go
// internal/eino/newtool_tool_test.go
func TestNewTool_Execute(t *testing.T) {
    // Test implementation
}
```

### Database Migrations

#### Creating Migrations

```bash
# Create new migration
migrate create -ext sql -dir migrations -seq add_new_table

# This creates:
# migrations/000001_add_new_table.up.sql
# migrations/000001_add_new_table.down.sql
```

#### Running Migrations

```bash
# Apply migrations
make migrate

# Or manually
migrate -path migrations -database "postgresql://user:pass@localhost:5432/dbname?sslmode=disable" up

# Rollback last migration
make migrate-down

# Reset database
make db-reset
```

### Performance Profiling

#### CPU Profiling

```bash
# Run with CPU profiling
go test -cpuprofile=cpu.prof -bench=.

# Analyze profile
go tool pprof cpu.prof
```

#### Memory Profiling

```bash
# Run with memory profiling
go test -memprofile=mem.prof -bench=.

# Analyze profile
go tool pprof mem.prof
```

#### HTTP Profiling

Add to `main.go`:

```go
import _ "net/http/pprof"

// In main()
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Access profiles at:
- `http://localhost:6060/debug/pprof/`
- `http://localhost:6060/debug/pprof/heap`
- `http://localhost:6060/debug/pprof/goroutine`

## Troubleshooting

### Common Issues

#### 1. Database Connection Failed

```
Error: failed to connect to database
```

**Solution:**
- Verify PostgreSQL is running: `pg_isready`
- Check credentials in `config.local.yaml`
- Ensure database exists: `psql -l`

#### 2. Redis Connection Failed

```
Error: failed to connect to redis
```

**Solution:**
- Verify Redis is running: `redis-cli ping`
- Check Redis host/port in configuration
- Ensure Redis is not password-protected (or provide password)

#### 3. LLM Provider API Key Invalid

```
Error: provider authentication failed
```

**Solution:**
- Verify API key is correct in configuration
- Check API key has not expired
- Ensure API key has proper permissions

#### 4. Port Already in Use

```
Error: bind: address already in use
```

**Solution:**
- Change port in `config.local.yaml`
- Or kill process using port: `lsof -ti:8080 | xargs kill`

#### 5. Go Module Issues

```
Error: cannot find module
```

**Solution:**
```bash
go mod tidy
go mod download
```

### Getting Help

- **Documentation**: Check `docs/` directory
- **GitHub Issues**: https://github.com/your-org/ai-research-platform/issues
- **Discussions**: https://github.com/your-org/ai-research-platform/discussions
- **Slack**: Join our developer Slack channel

## Best Practices

### Code Organization

- Follow the layered architecture (handler → service → repository)
- Keep handlers thin (validation and HTTP concerns only)
- Put business logic in services
- Use repositories for data access only
- Create interfaces for testability

### Error Handling

```go
// Use custom error types
if err != nil {
    return errors.NewDatabaseError("failed to query users", err)
}

// Log errors with context
logger.Error("operation failed",
    zap.Error(err),
    zap.String("user_id", userID),
    zap.String("operation", "create_session"),
)
```

### Testing

- Write tests for all new code
- Aim for >80% code coverage
- Use table-driven tests for multiple cases
- Use property-based tests for universal properties
- Mock external dependencies

### Security

- Never commit API keys or secrets
- Use environment variables for sensitive data
- Validate all user input
- Use parameterized queries (GORM handles this)
- Implement rate limiting
- Use HTTPS in production

### Performance

- Use connection pooling for database and Redis
- Implement caching for frequently accessed data
- Use goroutines for concurrent operations
- Profile before optimizing
- Monitor metrics in production

## Next Steps

1. **Explore the Codebase**: Read through the code to understand the architecture
2. **Run Tests**: Ensure all tests pass in your environment
3. **Make a Small Change**: Try adding a simple feature or fixing a bug
4. **Read Documentation**: Review all docs in the `docs/` directory
5. **Join the Community**: Connect with other developers

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Gin Framework](https://gin-gonic.com/docs/)
- [GORM Documentation](https://gorm.io/docs/)
- [Eino Framework](https://github.com/cloudwego/eino)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Redis Documentation](https://redis.io/documentation)

Happy coding! 🚀

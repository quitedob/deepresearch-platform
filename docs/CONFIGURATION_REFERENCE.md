# Configuration Reference

## Overview

The AI Research Platform uses YAML-based configuration with support for environment variable overrides. This document provides a comprehensive reference for all configuration options.

## Configuration Files

### File Locations

- `config.yaml` - Default configuration (committed to repository)
- `config.local.yaml` - Local overrides (gitignored, for development)
- `config.development.yaml` - Development environment
- `config.production.yaml` - Production environment

### Loading Order

Configuration is loaded in the following order (later values override earlier ones):

1. Default values in code
2. `config.yaml`
3. Environment-specific config (e.g., `config.production.yaml`)
4. Environment variables

### Environment Variable Mapping

Environment variables override configuration file values using the following pattern:

```
CONFIG_SECTION_KEY=value
```

Examples:
```bash
# Override database host
export DATABASE_HOST=db.example.com

# Override JWT secret
export SECURITY_JWT_SECRET=my-secret-key

# Override LLM provider API key
export LLM_PROVIDERS_DEEPSEEK_API_KEY=sk-xxx
```

## Configuration Sections

### Server Configuration

Controls HTTP server behavior.

```yaml
server:
  port: 8080                    # HTTP server port
  env: development              # Environment: development, staging, production
  read_timeout: 30              # Read timeout in seconds
  write_timeout: 30             # Write timeout in seconds
```

**Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | int | 8080 | Port number for HTTP server |
| `env` | string | development | Environment name (development, staging, production) |
| `read_timeout` | int | 30 | Maximum duration for reading request (seconds) |
| `write_timeout` | int | 30 | Maximum duration for writing response (seconds) |

**Environment Variables:**
```bash
SERVER_PORT=8080
SERVER_ENV=production
SERVER_READ_TIMEOUT=30
SERVER_WRITE_TIMEOUT=30
```

**Validation Rules:**
- `port` must be between 1 and 65535
- `env` must be one of: development, staging, production
- `read_timeout` must be > 0
- `write_timeout` must be > 0

---

### Database Configuration

PostgreSQL database connection settings.

```yaml
database:
  host: localhost               # Database host
  port: 5432                    # Database port
  user: postgres                # Database user
  password: password            # Database password
  db_name: ai_research_platform # Database name
  ssl_mode: disable             # SSL mode: disable, require, verify-ca, verify-full
  max_connections: 100          # Maximum open connections
  idle_connections: 10          # Maximum idle connections
```

**Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `host` | string | localhost | Database server hostname or IP |
| `port` | int | 5432 | Database server port |
| `user` | string | postgres | Database username |
| `password` | string | - | Database password (required) |
| `db_name` | string | ai_research_platform | Database name |
| `ssl_mode` | string | disable | SSL connection mode |
| `max_connections` | int | 100 | Maximum number of open connections |
| `idle_connections` | int | 10 | Maximum number of idle connections |

**Environment Variables:**
```bash
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=secret
DATABASE_DB_NAME=ai_research_platform
DATABASE_SSL_MODE=require
DATABASE_MAX_CONNECTIONS=100
DATABASE_IDLE_CONNECTIONS=10
```

**SSL Modes:**
- `disable` - No SSL (development only)
- `require` - SSL required but no verification
- `verify-ca` - SSL with CA verification
- `verify-full` - SSL with full verification (recommended for production)

**Connection String Format:**
```
postgresql://user:password@host:port/dbname?sslmode=disable
```

**Validation Rules:**
- `host` is required
- `port` must be between 1 and 65535
- `user` is required
- `password` is required
- `db_name` is required
- `max_connections` must be > 0
- `idle_connections` must be > 0 and <= max_connections

---

### Redis Configuration

Redis cache connection settings.

```yaml
redis:
  host: localhost               # Redis host
  port: 6379                    # Redis port
  password: ""                  # Redis password (empty for no auth)
  db: 0                         # Redis database number
  ttl: 3600                     # Default TTL in seconds
```

**Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `host` | string | localhost | Redis server hostname or IP |
| `port` | int | 6379 | Redis server port |
| `password` | string | "" | Redis password (empty for no authentication) |
| `db` | int | 0 | Redis database number (0-15) |
| `ttl` | int | 3600 | Default time-to-live for cache entries (seconds) |

**Environment Variables:**
```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=secret
REDIS_DB=0
REDIS_TTL=3600
```

**Validation Rules:**
- `host` is required
- `port` must be between 1 and 65535
- `db` must be between 0 and 15
- `ttl` must be > 0

---

### LLM Configuration

Large Language Model provider settings.

```yaml
llm:
  default_provider: deepseek    # Default provider to use
  timeout: 120                  # Request timeout in seconds
  retries: 3                    # Number of retry attempts
  providers:
    deepseek:
      api_key: ""               # DeepSeek API key
      base_url: https://api.deepseek.com
      models:
        - deepseek-chat
        - deepseek-reasoner
    zhipu:
      api_key: ""               # Zhipu AI API key
      base_url: https://open.bigmodel.cn
      models:
        - glm-4
        - glm-4-plus
    openai:
      api_key: ""               # OpenAI API key
      base_url: https://api.openai.com
      models:
        - gpt-4
        - gpt-3.5-turbo
    ollama:
      api_key: ""               # Not required for Ollama
      base_url: http://localhost:11434
      models:
        - llama2
        - mistral
```

**Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default_provider` | string | deepseek | Default LLM provider to use |
| `timeout` | int | 120 | Request timeout in seconds |
| `retries` | int | 3 | Number of retry attempts on failure |
| `providers` | map | - | Provider-specific configurations |

**Provider Configuration:**

| Field | Type | Description |
|-------|------|-------------|
| `api_key` | string | API key for the provider |
| `base_url` | string | Base URL for API requests |
| `models` | []string | List of supported models |

**Environment Variables:**
```bash
LLM_DEFAULT_PROVIDER=deepseek
LLM_TIMEOUT=120
LLM_RETRIES=3
LLM_PROVIDERS_DEEPSEEK_API_KEY=sk-xxx
LLM_PROVIDERS_DEEPSEEK_BASE_URL=https://api.deepseek.com
LLM_PROVIDERS_OPENAI_API_KEY=sk-xxx
```

**Supported Providers:**

1. **DeepSeek**
   - Base URL: `https://api.deepseek.com`
   - Models: `deepseek-chat`, `deepseek-reasoner`
   - API Key: Required

2. **Zhipu AI**
   - Base URL: `https://open.bigmodel.cn`
   - Models: `glm-4`, `glm-4-plus`
   - API Key: Required

3. **OpenAI**
   - Base URL: `https://api.openai.com`
   - Models: `gpt-4`, `gpt-3.5-turbo`, `gpt-4-turbo`
   - API Key: Required

4. **Ollama**
   - Base URL: `http://localhost:11434`
   - Models: Any locally installed model
   - API Key: Not required

**Validation Rules:**
- `default_provider` must be a registered provider
- `timeout` must be > 0
- `retries` must be >= 0
- Each provider must have a `base_url`
- Each provider must have at least one model

---

### Research Configuration

Research workflow settings.

```yaml
research:
  max_iterations: 10            # Maximum research iterations
  session_timeout: 3600         # Session timeout in seconds
  max_sources: 20               # Maximum sources per research
  enable_cache: true            # Enable result caching
  worker_pool_size: 10          # Number of concurrent workers
```

**Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_iterations` | int | 10 | Maximum number of research iterations |
| `session_timeout` | int | 3600 | Research session timeout (seconds) |
| `max_sources` | int | 20 | Maximum number of sources to gather |
| `enable_cache` | bool | true | Enable caching of research results |
| `worker_pool_size` | int | 10 | Number of concurrent research workers |

**Environment Variables:**
```bash
RESEARCH_MAX_ITERATIONS=10
RESEARCH_SESSION_TIMEOUT=3600
RESEARCH_MAX_SOURCES=20
RESEARCH_ENABLE_CACHE=true
RESEARCH_WORKER_POOL_SIZE=10
```

**Validation Rules:**
- `max_iterations` must be > 0 and <= 100
- `session_timeout` must be > 0
- `max_sources` must be > 0 and <= 100
- `worker_pool_size` must be > 0 and <= 50

---

### Security Configuration

Authentication, authorization, and security settings.

```yaml
security:
  jwt_secret: change-this-secret-in-production  # JWT signing secret
  jwt_expiration: 86400                         # JWT expiration (seconds)
  bcrypt_cost: 12                               # Bcrypt hashing cost
  cors:
    allow_origins:
      - http://localhost:3000
      - http://localhost:8080
    allow_methods:
      - GET
      - POST
      - PUT
      - DELETE
      - OPTIONS
  rate_limit:
    enabled: true                               # Enable rate limiting
    requests_per_min: 100                       # Requests per minute
```

**Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `jwt_secret` | string | - | Secret key for JWT signing (required) |
| `jwt_expiration` | int | 86400 | JWT token expiration time (seconds) |
| `bcrypt_cost` | int | 12 | Bcrypt hashing cost (4-31) |
| `cors.allow_origins` | []string | - | Allowed CORS origins |
| `cors.allow_methods` | []string | - | Allowed HTTP methods |
| `rate_limit.enabled` | bool | true | Enable rate limiting |
| `rate_limit.requests_per_min` | int | 100 | Maximum requests per minute per user |

**Environment Variables:**
```bash
SECURITY_JWT_SECRET=my-secret-key
SECURITY_JWT_EXPIRATION=86400
SECURITY_BCRYPT_COST=12
SECURITY_RATE_LIMIT_ENABLED=true
SECURITY_RATE_LIMIT_REQUESTS_PER_MIN=100
```

**JWT Secret Requirements:**
- Minimum 32 characters
- Use cryptographically secure random string
- Never commit to version control
- Rotate regularly in production

**Bcrypt Cost:**
- Range: 4-31
- Higher = more secure but slower
- Recommended: 12-14 for production
- 10 for development

**CORS Configuration:**
- `allow_origins`: List of allowed origins (use `*` for all, not recommended)
- `allow_methods`: HTTP methods to allow
- `allow_headers`: Automatically includes Origin, Content-Type, Authorization

**Rate Limiting:**
- Applied per authenticated user
- Sliding window algorithm
- Returns 429 status when exceeded
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`

**Validation Rules:**
- `jwt_secret` is required and must be >= 32 characters
- `jwt_expiration` must be > 0
- `bcrypt_cost` must be between 4 and 31
- `requests_per_min` must be > 0

---

### Logging Configuration

Structured logging settings.

```yaml
logging:
  level: info                   # Log level: debug, info, warn, error
  format: json                  # Log format: json, console
  output_path: stdout           # Output path: stdout, stderr, or file path
```

**Options:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `level` | string | info | Minimum log level to output |
| `format` | string | json | Log output format |
| `output_path` | string | stdout | Where to write logs |

**Environment Variables:**
```bash
LOGGING_LEVEL=info
LOGGING_FORMAT=json
LOGGING_OUTPUT_PATH=stdout
```

**Log Levels:**
- `debug` - Detailed debugging information
- `info` - General informational messages
- `warn` - Warning messages
- `error` - Error messages only

**Log Formats:**
- `json` - Structured JSON format (recommended for production)
- `console` - Human-readable console format (recommended for development)

**Output Paths:**
- `stdout` - Standard output
- `stderr` - Standard error
- `/path/to/file.log` - File path

**Validation Rules:**
- `level` must be one of: debug, info, warn, error
- `format` must be one of: json, console
- `output_path` must be valid path or stdout/stderr

---

## Complete Configuration Example

### Development Configuration

```yaml
# config.development.yaml
server:
  port: 8080
  env: development
  read_timeout: 30
  write_timeout: 30

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  db_name: ai_research_platform_dev
  ssl_mode: disable
  max_connections: 50
  idle_connections: 10

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  ttl: 1800

llm:
  default_provider: deepseek
  timeout: 120
  retries: 3
  providers:
    deepseek:
      api_key: "${DEEPSEEK_API_KEY}"
      base_url: https://api.deepseek.com
      models:
        - deepseek-chat
    ollama:
      api_key: ""
      base_url: http://localhost:11434
      models:
        - llama2

research:
  max_iterations: 5
  session_timeout: 1800
  max_sources: 10
  enable_cache: true
  worker_pool_size: 5

security:
  jwt_secret: "dev-secret-change-in-production"
  jwt_expiration: 86400
  bcrypt_cost: 10
  cors:
    allow_origins:
      - http://localhost:3000
      - http://localhost:8080
    allow_methods:
      - GET
      - POST
      - PUT
      - DELETE
      - OPTIONS
  rate_limit:
    enabled: false
    requests_per_min: 1000

logging:
  level: debug
  format: console
  output_path: stdout
```

### Production Configuration

```yaml
# config.production.yaml
server:
  port: 8080
  env: production
  read_timeout: 30
  write_timeout: 30

database:
  host: "${DATABASE_HOST}"
  port: 5432
  user: "${DATABASE_USER}"
  password: "${DATABASE_PASSWORD}"
  db_name: ai_research_platform
  ssl_mode: verify-full
  max_connections: 100
  idle_connections: 10

redis:
  host: "${REDIS_HOST}"
  port: 6379
  password: "${REDIS_PASSWORD}"
  db: 0
  ttl: 3600

llm:
  default_provider: deepseek
  timeout: 120
  retries: 3
  providers:
    deepseek:
      api_key: "${DEEPSEEK_API_KEY}"
      base_url: https://api.deepseek.com
      models:
        - deepseek-chat
        - deepseek-reasoner
    openai:
      api_key: "${OPENAI_API_KEY}"
      base_url: https://api.openai.com
      models:
        - gpt-4
        - gpt-3.5-turbo

research:
  max_iterations: 10
  session_timeout: 3600
  max_sources: 20
  enable_cache: true
  worker_pool_size: 10

security:
  jwt_secret: "${JWT_SECRET}"
  jwt_expiration: 86400
  bcrypt_cost: 14
  cors:
    allow_origins:
      - https://app.example.com
    allow_methods:
      - GET
      - POST
      - PUT
      - DELETE
      - OPTIONS
  rate_limit:
    enabled: true
    requests_per_min: 100

logging:
  level: info
  format: json
  output_path: stdout
```

## Environment Variables

### Complete List

```bash
# Server
SERVER_PORT=8080
SERVER_ENV=production
SERVER_READ_TIMEOUT=30
SERVER_WRITE_TIMEOUT=30

# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=secret
DATABASE_DB_NAME=ai_research_platform
DATABASE_SSL_MODE=require
DATABASE_MAX_CONNECTIONS=100
DATABASE_IDLE_CONNECTIONS=10

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=secret
REDIS_DB=0
REDIS_TTL=3600

# LLM
LLM_DEFAULT_PROVIDER=deepseek
LLM_TIMEOUT=120
LLM_RETRIES=3
LLM_PROVIDERS_DEEPSEEK_API_KEY=sk-xxx
LLM_PROVIDERS_DEEPSEEK_BASE_URL=https://api.deepseek.com
LLM_PROVIDERS_OPENAI_API_KEY=sk-xxx
LLM_PROVIDERS_OPENAI_BASE_URL=https://api.openai.com

# Research
RESEARCH_MAX_ITERATIONS=10
RESEARCH_SESSION_TIMEOUT=3600
RESEARCH_MAX_SOURCES=20
RESEARCH_ENABLE_CACHE=true
RESEARCH_WORKER_POOL_SIZE=10

# Security
SECURITY_JWT_SECRET=my-secret-key
SECURITY_JWT_EXPIRATION=86400
SECURITY_BCRYPT_COST=12
SECURITY_RATE_LIMIT_ENABLED=true
SECURITY_RATE_LIMIT_REQUESTS_PER_MIN=100

# Logging
LOGGING_LEVEL=info
LOGGING_FORMAT=json
LOGGING_OUTPUT_PATH=stdout
```

### Docker Environment File

```bash
# .env
SERVER_PORT=8080
DATABASE_HOST=postgres
DATABASE_PASSWORD=secure_password
REDIS_HOST=redis
JWT_SECRET=your-secure-jwt-secret-here
DEEPSEEK_API_KEY=your-deepseek-api-key
OPENAI_API_KEY=your-openai-api-key
```

## Configuration Validation

The application validates configuration on startup and will fail with descriptive errors if configuration is invalid.

### Validation Errors

```
Error: invalid configuration
- server.port must be between 1 and 65535
- database.password is required
- security.jwt_secret must be at least 32 characters
- llm.default_provider 'invalid' is not registered
```

### Testing Configuration

```bash
# Validate configuration without starting server
go run cmd/server/main.go -config config.yaml -validate

# Check configuration values
go run cmd/server/main.go -config config.yaml -print-config
```

## Best Practices

### Security

1. **Never commit secrets** to version control
2. **Use environment variables** for sensitive data in production
3. **Rotate JWT secrets** regularly
4. **Use strong bcrypt cost** (12-14) in production
5. **Enable SSL** for database connections in production
6. **Restrict CORS origins** to known domains

### Performance

1. **Tune database connections** based on load
2. **Adjust worker pool size** based on CPU cores
3. **Configure appropriate timeouts** for your use case
4. **Enable caching** for better performance
5. **Use connection pooling** for Redis

### Monitoring

1. **Use JSON logging** in production for parsing
2. **Set appropriate log levels** (info in production, debug in development)
3. **Monitor rate limit metrics** to adjust limits
4. **Track LLM provider metrics** for reliability

### Development

1. **Use local configuration files** (config.local.yaml)
2. **Lower bcrypt cost** (10) for faster tests
3. **Disable rate limiting** for easier testing
4. **Use console logging** for readability
5. **Enable debug logging** when troubleshooting

## Troubleshooting

### Configuration Not Loading

**Problem:** Configuration values not being applied

**Solutions:**
1. Check file path is correct
2. Verify YAML syntax is valid
3. Ensure environment variables are exported
4. Check configuration loading order

### Environment Variables Not Working

**Problem:** Environment variables not overriding config

**Solutions:**
1. Verify variable naming matches pattern
2. Ensure variables are exported before running
3. Check for typos in variable names
4. Restart application after setting variables

### Invalid Configuration

**Problem:** Application fails to start with validation error

**Solutions:**
1. Read error message carefully
2. Check required fields are present
3. Verify value types match expected types
4. Ensure values are within valid ranges

## Additional Resources

- [Viper Documentation](https://github.com/spf13/viper)
- [12-Factor App Config](https://12factor.net/config)
- [YAML Specification](https://yaml.org/spec/)
- [Environment Variables Best Practices](https://12factor.net/config)

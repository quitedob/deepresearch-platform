# API Documentation

## Overview

The AI Research Platform provides a RESTful API for managing chat sessions, conducting AI-powered research, and interacting with multiple LLM providers. All API endpoints (except authentication and health checks) require JWT authentication.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

### Register a New User

Create a new user account.

**Endpoint:** `POST /api/v1/auth/register`

**Request Body:**
```json
{
  "email": "user@example.com",
  "username": "johndoe",
  "password": "securepassword123"
}
```

**Response:** `201 Created`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "username": "johndoe",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "expires_in": 86400
}
```

**Error Responses:**
- `400 Bad Request` - Invalid input
- `409 Conflict` - User already exists

### Login

Authenticate and receive a JWT token.

**Endpoint:** `POST /api/v1/auth/login`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "username": "johndoe"
  },
  "expires_in": 86400
}
```

**Error Responses:**
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Invalid credentials

### Refresh Token

Refresh an existing JWT token.

**Endpoint:** `POST /api/v1/auth/refresh`

**Request Body:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "username": "johndoe"
  },
  "expires_in": 86400
}
```

**Error Responses:**
- `401 Unauthorized` - Invalid or expired token

## Chat API

All chat endpoints require authentication via JWT token in the `Authorization` header:
```
Authorization: Bearer <token>
```

### Create Chat Session

Create a new chat session with a specific LLM provider.

**Endpoint:** `POST /api/v1/chat/sessions`

**Request Body:**
```json
{
  "provider": "deepseek",
  "model": "deepseek-chat",
  "title": "My Chat Session",
  "system_prompt": "You are a helpful assistant."
}
```

**Response:** `201 Created`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "My Chat Session",
  "provider": "deepseek",
  "model": "deepseek-chat",
  "system_prompt": "You are a helpful assistant.",
  "message_count": 0,
  "created_at": "2024-01-15T10:35:00Z",
  "updated_at": "2024-01-15T10:35:00Z"
}
```

**Supported Providers:**
- `deepseek` - DeepSeek AI (models: deepseek-chat, deepseek-reasoner)
- `zhipu` - Zhipu AI (models: glm-4, glm-4-plus)
- `openai` - OpenAI (models: gpt-4, gpt-3.5-turbo)
- `ollama` - Ollama (models: llama2, mistral)

### List Chat Sessions

Retrieve all chat sessions for the authenticated user.

**Endpoint:** `GET /api/v1/chat/sessions`

**Query Parameters:**
- `limit` (optional, default: 20, max: 100) - Number of sessions to return
- `offset` (optional, default: 0) - Pagination offset

**Response:** `200 OK`
```json
{
  "sessions": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "My Chat Session",
      "provider": "deepseek",
      "model": "deepseek-chat",
      "message_count": 5,
      "created_at": "2024-01-15T10:35:00Z",
      "updated_at": "2024-01-15T10:40:00Z"
    }
  ],
  "limit": 20,
  "offset": 0
}
```

### Get Chat Session

Retrieve a specific chat session.

**Endpoint:** `GET /api/v1/chat/sessions/:session_id`

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "My Chat Session",
  "provider": "deepseek",
  "model": "deepseek-chat",
  "system_prompt": "You are a helpful assistant.",
  "message_count": 5,
  "created_at": "2024-01-15T10:35:00Z",
  "updated_at": "2024-01-15T10:40:00Z"
}
```

**Error Responses:**
- `404 Not Found` - Session not found
- `403 Forbidden` - Access denied (not session owner)

### Delete Chat Session

Delete a chat session and all its messages.

**Endpoint:** `DELETE /api/v1/chat/sessions/:session_id`

**Response:** `200 OK`
```json
{
  "message": "session deleted successfully"
}
```

### Send Message

Send a message to a chat session and receive a response.

**Endpoint:** `POST /api/v1/chat/sessions/:session_id/messages`

**Request Body:**
```json
{
  "content": "What is the capital of France?",
  "stream": false
}
```

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440002",
  "session_id": "550e8400-e29b-41d4-a716-446655440001",
  "role": "assistant",
  "content": "The capital of France is Paris.",
  "tokens_used": 15,
  "created_at": "2024-01-15T10:36:00Z"
}
```

### Get Message History

Retrieve message history for a chat session.

**Endpoint:** `GET /api/v1/chat/sessions/:session_id/messages`

**Query Parameters:**
- `limit` (optional, default: 50, max: 200) - Number of messages to return
- `offset` (optional, default: 0) - Pagination offset

**Response:** `200 OK`
```json
{
  "messages": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "session_id": "550e8400-e29b-41d4-a716-446655440001",
      "role": "user",
      "content": "What is the capital of France?",
      "tokens_used": 8,
      "created_at": "2024-01-15T10:36:00Z"
    },
    {
      "id": "550e8400-e29b-41d4-a716-446655440003",
      "session_id": "550e8400-e29b-41d4-a716-446655440001",
      "role": "assistant",
      "content": "The capital of France is Paris.",
      "tokens_used": 15,
      "created_at": "2024-01-15T10:36:01Z"
    }
  ],
  "limit": 50,
  "offset": 0
}
```

### Stream Message (SSE)

Stream a message response in real-time using Server-Sent Events.

**Endpoint:** `GET /api/v1/chat/sessions/:session_id/stream?content=<message>`

**Query Parameters:**
- `content` (required) - The message content to send

**Response:** `200 OK` (text/event-stream)

**Event Types:**
- `chunk` - Partial response content
- `done` - Streaming complete
- `error` - Error occurred

**Example Events:**
```
event: chunk
data: {"content":"The","metadata":{}}

event: chunk
data: {"content":" capital","metadata":{}}

event: chunk
data: {"content":" of","metadata":{}}

event: done
data: {"message_id":"550e8400-e29b-41d4-a716-446655440003"}
```

### Update Provider

Switch the LLM provider for a chat session.

**Endpoint:** `PUT /api/v1/chat/sessions/:session_id/provider`

**Request Body:**
```json
{
  "provider": "openai",
  "model": "gpt-4"
}
```

**Response:** `200 OK`
```json
{
  "message": "provider updated successfully"
}
```

## Research API

### Start Research Session

Initiate a new AI-powered research session.

**Endpoint:** `POST /api/v1/research/sessions`

**Request Body:**
```json
{
  "query": "What are the latest developments in quantum computing?",
  "research_type": "deep"
}
```

**Research Types:**
- `deep` - Comprehensive research with multiple sources
- `quick` - Fast research with limited sources
- `academic` - Focus on scholarly sources (arXiv)

**Response:** `201 Created`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440004",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "query": "What are the latest developments in quantum computing?",
  "status": "planning",
  "progress": 0.0,
  "research_type": "deep",
  "created_at": "2024-01-15T10:40:00Z",
  "updated_at": "2024-01-15T10:40:00Z"
}
```

**Status Values:**
- `planning` - Creating research plan
- `executing` - Executing research tasks
- `synthesis` - Synthesizing findings
- `completed` - Research complete
- `failed` - Research failed
- `cancelled` - Research cancelled

### List Research Sessions

Retrieve all research sessions for the authenticated user.

**Endpoint:** `GET /api/v1/research/sessions`

**Query Parameters:**
- `limit` (optional, default: 20, max: 100)
- `offset` (optional, default: 0)

**Response:** `200 OK`
```json
{
  "sessions": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440004",
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "query": "What are the latest developments in quantum computing?",
      "status": "completed",
      "progress": 100.0,
      "research_type": "deep",
      "created_at": "2024-01-15T10:40:00Z",
      "updated_at": "2024-01-15T10:45:00Z"
    }
  ],
  "limit": 20,
  "offset": 0
}
```

### Get Research Session

Retrieve a specific research session.

**Endpoint:** `GET /api/v1/research/sessions/:session_id`

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440004",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "query": "What are the latest developments in quantum computing?",
  "status": "completed",
  "progress": 100.0,
  "research_type": "deep",
  "created_at": "2024-01-15T10:40:00Z",
  "updated_at": "2024-01-15T10:45:00Z"
}
```

### Get Research Results

Retrieve the final results of a completed research session.

**Endpoint:** `GET /api/v1/research/sessions/:session_id/results`

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440005",
  "research_id": "550e8400-e29b-41d4-a716-446655440004",
  "summary": "Recent developments in quantum computing include...",
  "findings": [
    {
      "topic": "Quantum Error Correction",
      "description": "New techniques for error correction...",
      "sources": ["arxiv:2401.12345", "wikipedia:Quantum_error_correction"]
    }
  ],
  "citations": [
    {
      "id": "arxiv:2401.12345",
      "title": "Advances in Quantum Error Correction",
      "url": "https://arxiv.org/abs/2401.12345",
      "type": "arxiv"
    }
  ],
  "created_at": "2024-01-15T10:45:00Z"
}
```

### Get Research Tasks

Retrieve the tasks executed during a research session.

**Endpoint:** `GET /api/v1/research/sessions/:session_id/tasks`

**Query Parameters:**
- `limit` (optional, default: 50, max: 200)
- `offset` (optional, default: 0)

**Response:** `200 OK`
```json
{
  "tasks": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440006",
      "research_id": "550e8400-e29b-41d4-a716-446655440004",
      "task_type": "search",
      "tool_name": "web_search",
      "status": "completed",
      "input": {"query": "quantum computing developments 2024"},
      "output": {"results": [...]},
      "execution_time": 1250,
      "created_at": "2024-01-15T10:41:00Z",
      "completed_at": "2024-01-15T10:41:01Z"
    }
  ],
  "limit": 50,
  "offset": 0
}
```

### Stream Research Progress (SSE)

Stream real-time progress updates for a research session.

**Endpoint:** `GET /api/v1/research/sessions/:session_id/stream`

**Response:** `200 OK` (text/event-stream)

**Event Types:**
- `progress` - Progress update
- `completed` - Research complete
- `error` - Error occurred
- `cancelled` - Research cancelled

**Example Events:**
```
event: progress
data: {"stage":"planning","progress":10,"message":"Creating research plan","task_name":"","task_status":"","timestamp":"2024-01-15T10:40:05Z"}

event: progress
data: {"stage":"executing","progress":30,"message":"Searching web sources","task_name":"web_search","task_status":"running","partial_data":{"sources_found":5},"timestamp":"2024-01-15T10:40:15Z"}

event: progress
data: {"stage":"synthesis","progress":90,"message":"Synthesizing findings","task_name":"","task_status":"","timestamp":"2024-01-15T10:44:50Z"}

event: completed
data: {"message":"Research completed successfully","data":{"result_id":"550e8400-e29b-41d4-a716-446655440005"}}
```

### Cancel Research

Cancel an ongoing research session.

**Endpoint:** `POST /api/v1/research/sessions/:session_id/cancel`

**Response:** `200 OK`
```json
{
  "message": "research cancelled successfully"
}
```

## LLM Provider API

### List Providers

List all registered LLM providers with their metrics.

**Endpoint:** `GET /api/v1/llm/providers`

**Response:** `200 OK`
```json
{
  "providers": [
    {
      "name": "deepseek",
      "metrics": {
        "success_count": 150,
        "failure_count": 5,
        "success_rate": 0.9677,
        "average_latency": 850,
        "last_success": "2024-01-15T10:45:00Z",
        "last_failure": "2024-01-15T09:30:00Z"
      }
    },
    {
      "name": "openai",
      "metrics": {
        "success_count": 200,
        "failure_count": 2,
        "success_rate": 0.9901,
        "average_latency": 1200,
        "last_success": "2024-01-15T10:44:00Z",
        "last_failure": "2024-01-15T08:15:00Z"
      }
    }
  ],
  "count": 2
}
```

### Get Provider Metrics

Get detailed metrics for a specific provider.

**Endpoint:** `GET /api/v1/llm/providers/:provider/metrics`

**Response:** `200 OK`
```json
{
  "provider": "deepseek",
  "success_count": 150,
  "failure_count": 5,
  "success_rate": 0.9677,
  "average_latency": 850,
  "total_latency": 127500,
  "last_success": "2024-01-15T10:45:00Z",
  "last_failure": "2024-01-15T09:30:00Z"
}
```

### List Models

List all available models across all providers.

**Endpoint:** `GET /api/v1/llm/models`

**Response:** `200 OK`
```json
{
  "models": [
    {
      "name": "deepseek-chat",
      "provider": "deepseek",
      "available": true
    },
    {
      "name": "gpt-4",
      "provider": "openai",
      "available": true
    }
  ]
}
```

### Test Provider

Test a provider with a simple request.

**Endpoint:** `POST /api/v1/llm/test`

**Request Body:**
```json
{
  "provider": "deepseek",
  "model": "deepseek-chat",
  "messages": ["Hello, how are you?"]
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "provider": "deepseek",
  "model": "deepseek-chat",
  "response": "Hello! I'm doing well, thank you for asking. How can I assist you today?"
}
```

## Health Check API

### Health Check

Basic health check endpoint (no authentication required).

**Endpoint:** `GET /health`

**Response:** `200 OK`
```json
{
  "status": "healthy"
}
```

### Readiness Check

Check if the service is ready to accept traffic (no authentication required).

**Endpoint:** `GET /ready`

**Response:** `200 OK`
```json
{
  "status": "ready"
}
```

**Degraded Response:** `200 OK`
```json
{
  "status": "degraded",
  "reason": "cache unavailable",
  "details": "service operational but cache is down"
}
```

**Not Ready Response:** `503 Service Unavailable`
```json
{
  "status": "not ready",
  "reason": "database connection error"
}
```

## Metrics API

### Prometheus Metrics

Expose Prometheus metrics for monitoring (no authentication required).

**Endpoint:** `GET /metrics`

**Response:** `200 OK` (text/plain)

**Metrics Exposed:**
- `http_request_duration_seconds` - HTTP request duration histogram
- `http_requests_total` - Total HTTP requests counter
- `http_requests_in_flight` - Current in-flight requests gauge
- `llm_requests_total` - Total LLM requests by provider/model/status
- `llm_request_duration_seconds` - LLM request duration histogram
- `llm_request_errors_total` - LLM request errors by provider/model/type
- `llm_provider_status` - LLM provider availability status
- `research_sessions_active` - Active research sessions gauge
- `research_sessions_total` - Total research sessions by status
- `research_session_duration_seconds` - Research session duration histogram
- `research_tasks_total` - Total research tasks by type/status
- `db_connections_active` - Active database connections gauge
- `db_query_duration_seconds` - Database query duration histogram
- `cache_hits_total` - Cache hits by tier
- `cache_misses_total` - Cache misses by tier

## Error Responses

All API endpoints follow a consistent error response format:

```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable error message",
  "details": "Additional error details (optional)"
}
```

### Common Error Codes

- `INVALID_INPUT` (400) - Invalid request parameters
- `UNAUTHORIZED` (401) - Authentication required or failed
- `FORBIDDEN` (403) - Access denied
- `NOT_FOUND` (404) - Resource not found
- `CONFLICT` (409) - Resource conflict (e.g., duplicate email)
- `VALIDATION_FAILED` (400) - Input validation failed
- `RATE_LIMIT_EXCEEDED` (429) - Rate limit exceeded
- `INTERNAL_ERROR` (500) - Internal server error
- `DATABASE_ERROR` (500) - Database operation failed
- `PROVIDER_FAILED` (502) - LLM provider failed
- `SERVICE_UNAVAILABLE` (503) - Service temporarily unavailable
- `TIMEOUT` (504) - Request timeout

## Rate Limiting

API requests are rate-limited to prevent abuse. The default limit is 100 requests per minute per user.

**Rate Limit Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705318800
```

When rate limit is exceeded:
```json
{
  "code": "RATE_LIMIT_EXCEEDED",
  "message": "Rate limit exceeded. Please try again later.",
  "details": "Limit: 100 requests per minute"
}
```

## CORS

The API supports Cross-Origin Resource Sharing (CORS) for the following origins (configurable):
- `http://localhost:3000`
- `http://localhost:8080`

**Allowed Methods:** GET, POST, PUT, DELETE, OPTIONS

**Allowed Headers:** Origin, Content-Type, Authorization

## Examples

### Complete Chat Flow

```bash
# 1. Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","username":"john","password":"password123"}'

# 2. Create chat session
curl -X POST http://localhost:8080/api/v1/chat/sessions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"provider":"deepseek","model":"deepseek-chat","title":"My Chat"}'

# 3. Send message
curl -X POST http://localhost:8080/api/v1/chat/sessions/<session_id>/messages \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"content":"Hello!"}'

# 4. Get message history
curl -X GET http://localhost:8080/api/v1/chat/sessions/<session_id>/messages \
  -H "Authorization: Bearer <token>"
```

### Complete Research Flow

```bash
# 1. Start research
curl -X POST http://localhost:8080/api/v1/research/sessions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"query":"quantum computing","research_type":"deep"}'

# 2. Stream progress (SSE)
curl -N -X GET http://localhost:8080/api/v1/research/sessions/<session_id>/stream \
  -H "Authorization: Bearer <token>"

# 3. Get results
curl -X GET http://localhost:8080/api/v1/research/sessions/<session_id>/results \
  -H "Authorization: Bearer <token>"

# 4. Get tasks
curl -X GET http://localhost:8080/api/v1/research/sessions/<session_id>/tasks \
  -H "Authorization: Bearer <token>"
```

## SDK Examples

### JavaScript/TypeScript

```typescript
// Initialize client
const client = new AIResearchClient({
  baseURL: 'http://localhost:8080',
  token: '<your-jwt-token>'
});

// Create chat session
const session = await client.chat.createSession({
  provider: 'deepseek',
  model: 'deepseek-chat',
  title: 'My Chat'
});

// Send message
const response = await client.chat.sendMessage(session.id, {
  content: 'What is AI?'
});

// Stream message
const stream = await client.chat.streamMessage(session.id, {
  content: 'Explain quantum computing'
});

stream.on('chunk', (chunk) => {
  console.log(chunk.content);
});

stream.on('done', () => {
  console.log('Streaming complete');
});

// Start research
const research = await client.research.start({
  query: 'Latest AI developments',
  researchType: 'deep'
});

// Stream research progress
const progressStream = await client.research.streamProgress(research.id);

progressStream.on('progress', (event) => {
  console.log(`Progress: ${event.progress}% - ${event.message}`);
});

progressStream.on('completed', (event) => {
  console.log('Research complete!');
});
```

### Python

```python
from ai_research_client import AIResearchClient

# Initialize client
client = AIResearchClient(
    base_url='http://localhost:8080',
    token='<your-jwt-token>'
)

# Create chat session
session = client.chat.create_session(
    provider='deepseek',
    model='deepseek-chat',
    title='My Chat'
)

# Send message
response = client.chat.send_message(
    session_id=session['id'],
    content='What is AI?'
)

# Stream message
for chunk in client.chat.stream_message(session['id'], 'Explain quantum computing'):
    print(chunk['content'], end='', flush=True)

# Start research
research = client.research.start(
    query='Latest AI developments',
    research_type='deep'
)

# Stream research progress
for event in client.research.stream_progress(research['id']):
    if event['type'] == 'progress':
        print(f"Progress: {event['progress']}% - {event['message']}")
    elif event['type'] == 'completed':
        print('Research complete!')
        break
```

## Webhooks (Future)

Webhook support for asynchronous notifications is planned for future releases:
- Research completion notifications
- Error alerts
- Provider status changes

## API Versioning

The API uses URL-based versioning (`/api/v1/`). Breaking changes will be introduced in new versions while maintaining backward compatibility for existing versions.

## Support

For API support and questions:
- GitHub Issues: https://github.com/your-org/ai-research-platform/issues
- Documentation: https://docs.ai-research-platform.com
- Email: support@ai-research-platform.com

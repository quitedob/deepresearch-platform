# API 使用示例

本文档提供了AI研究平台API的详细使用示例。

## 认证说明

所有需要认证的API都使用JWT令牌。在请求头中添加：
```
Authorization: Bearer <your_jwt_token>
```

## 用户管理API

### 用户注册

```bash
curl -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "user@example.com",
    "password": "securepassword123",
    "full_name": "测试用户"
  }'
```

**响应示例：**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 86400,
  "user": {
    "id": "user_001",
    "username": "testuser",
    "email": "user@example.com",
    "full_name": "测试用户",
    "role": "user",
    "status": "active",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z",
    "is_admin": false
  }
}
```

### 用户登录

```bash
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "securepassword123"
  }'
```

### 获取当前用户信息

```bash
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <your_jwt_token>"
```

### 更新用户资料

```bash
curl -X PUT http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <your_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "更新后的姓名",
    "avatar": "https://example.com/avatar.jpg",
    "bio": "这是我的个人简介"
  }'
```

## 聊天API

### 创建聊天会话

```bash
curl -X POST http://localhost:8080/api/v1/chat/sessions \
  -H "Authorization: Bearer <your_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "关于AI的对话",
    "llm_provider": "deepseek",
    "model_name": "deepseek-chat",
    "system_prompt": "你是一个专业的AI助手，请用简洁明了的语言回答问题。"
  }'
```

**响应示例：**
```json
{
  "id": "session_1642230400",
  "user_id": "user_001",
  "title": "关于AI的对话",
  "provider": "deepseek",
  "model": "deepseek-chat",
  "system_prompt": "你是一个专业的AI助手，请用简洁明了的语言回答问题。",
  "message_count": 0,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### 发送聊天消息

```bash
curl -X POST http://localhost:8080/api/v1/chat/chat \
  -H "Authorization: Bearer <your_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "session_1642230400",
    "message": "请解释一下机器学习的基本概念",
    "stream": false
  }'
```

### 流式聊天

```bash
curl -X POST http://localhost:8080/api/v1/chat/stream \
  -H "Authorization: Bearer <your_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "session_1642230400",
    "message": "请解释一下机器学习的基本概念",
    "stream": true
  }'
```

**SSE响应格式：**
```
data: {"type":"start","content":""}

data: {"type":"content","content":"机"}

data: {"type":"content","content":"器"}

data: {"type":"content","content":"学"}

data: {"type":"content","content":"习"}

...

data: {"type":"end","content":""}
```

### 获取可用模型

```bash
curl -X GET http://localhost:8080/api/v1/chat/models \
  -H "Authorization: Bearer <your_jwt_token>"
```

**响应示例：**
```json
{
  "models": [
    {
      "id": "deepseek-chat",
      "name": "deepseek-chat",
      "display_name": "DeepSeek Chat",
      "provider": "deepseek",
      "description": "DeepSeek的高性能聊天模型",
      "context_length": 32000,
      "max_tokens": 4000,
      "capabilities": ["streaming", "tools"],
      "pricing": {
        "input_price_per_1k": 0.0001,
        "output_price_per_1k": 0.0002,
        "currency": "USD"
      }
    }
  ],
  "total": 1
}
```

## 研究API

### 启动深度研究

```bash
curl -X POST http://localhost:8080/api/v1/research/start \
  -H "Authorization: Bearer <your_jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "人工智能在医疗领域的最新发展",
    "research_type": "deep",
    "sources": ["web", "wikipedia", "arxiv"],
    "include_images": false,
    "llm_config": {
      "provider": "deepseek",
      "model": "deepseek-chat"
    }
  }'
```

**响应示例：**
```json
{
  "success": true,
  "session_id": "research_1642230400",
  "message": "研究任务已启动",
  "data": {
    "query": "人工智能在医疗领域的最新发展",
    "research_type": "deep",
    "sources": ["web", "wikipedia", "arxiv"],
    "include_images": false,
    "status": "planning",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### 获取研究状态

```bash
curl -X GET http://localhost:8080/api/v1/research/status/research_1642230400 \
  -H "Authorization: Bearer <your_jwt_token>"
```

### 流式获取研究进度

```bash
curl -X GET http://localhost:8080/api/v1/research/stream/research_1642230400 \
  -H "Authorization: Bearer <your_jwt_token>" \
  -H "Accept: text/event-stream"
```

**SSE响应示例：**
```json
data: {"type":"connected","session_id":"research_1642230400"}

data: {"type":"status_update","session_id":"research_1642230400","status":"planning"}

data: {"type":"status_update","session_id":"research_1642230400","status":"executing","data":{"current_step":"执行网络搜索","progress":0.25}}

data: {"type":"completed","session_id":"research_1642230400","status":"completed","data":{"report_text":"完整的研究报告...","evidence":[{"id":1,"source_type":"web","source_title":"医疗AI发展报告","source_url":"https://example.com","confidence_score":0.90}]}}
```

### 导出研究数据

```bash
curl -X GET http://localhost:8080/api/v1/research/export/research_1642230400 \
  -H "Authorization: Bearer <your_jwt_token>"
```

## 错误处理

所有API错误都遵循统一的响应格式：

```json
{
  "code": "ERROR_CODE",
  "message": "错误描述",
  "details": "详细错误信息（可选）"
}
```

### 常见错误代码

- `UNAUTHORIZED` (401): 认证失败
- `FORBIDDEN` (403): 权限不足
- `NOT_FOUND` (404): 资源不存在
- `INVALID_INPUT` (400): 请求参数无效
- `RATE_LIMIT_EXCEEDED` (429): 请求频率超限
- `INTERNAL_ERROR` (500): 服务器内部错误

## 分页参数

所有列表API都支持分页：

- `limit`: 每页记录数（默认：20，最大：100）
- `offset`: 偏移量（默认：0）

示例：
```bash
curl -X GET "http://localhost:8080/api/v1/chat/sessions?limit=10&offset=20" \
  -H "Authorization: Bearer <your_jwt_token>"
```

## 完整的API使用流程

1. **注册或登录**获取JWT令牌
2. **在请求头中包含JWT令牌**进行认证
3. **创建聊天会话**开始对话
4. **发送消息**进行交互（支持流式和非流式）
5. **启动深度研究**进行复杂查询
6. **使用SSE流**实时监控研究进度
7. **导出和查看研究结果**

## 开发建议

1. **使用适当的HTTP状态码**
2. **实现重试机制**处理网络错误
3. **监听SSE事件**处理实时更新
4. **缓存API响应**提高性能
5. **实现请求限流**防止滥用
6. **添加适当的错误处理**
7. **使用结构化日志**便于调试

## 测试用户

系统提供以下测试账户：

**管理员账户：**
- 用户名：admin
- 密码：password

**普通用户账户：**
- 用户名：testuser
- 密码：password

这些账户仅用于开发和测试环境，生产环境中需要创建真实的用户账户。
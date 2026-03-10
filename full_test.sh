#!/bin/bash
# 清理之前的日志
rm -f server.log
# 启动服务器（后台）
./server.exe > server.log 2>&1 &
SERVER_PID=$!
echo "服务器 PID: $SERVER_PID"

# 等待服务器启动
sleep 3

# 注册/登录用户（使用管理员账户）
echo "登录中..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }')
echo "登录响应: $LOGIN_RESPONSE"

# 提取 token（处理不同的响应格式）
if echo "$LOGIN_RESPONSE" | grep -q "token"; then
  TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
elif echo "$LOGIN_RESPONSE" | grep -q "access_token"; then
  TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
fi

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
  echo "登录失败或无法提取 token"
  echo "尝试注册新用户..."
  REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
    -H "Content-Type: application/json" \
    -d '{
      "email": "test@example.com",
      "username": "testuser",
      "password": "testpass123"
    }')
  echo "注册响应: $REGISTER_RESPONSE"
  
  # 再次尝试登录
  LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{
      "email": "test@example.com",
      "password": "testpass123"
    }')
  echo "第二次登录响应: $LOGIN_RESPONSE"
  TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
fi

if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
  echo "Token 获取成功: ${TOKEN:0:20}..."
  
  # 测试深度研究 API
  echo ""
  echo "发送深度研究请求..."
  RESEARCH_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/research/start \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{
      "query": "今日金价是多少",
      "research_type": "quick",
      "sources": ["web"]
    }')
  echo "研究响应: $RESEARCH_RESPONSE"
  
  # 等待研究进行
  sleep 10
  
  # 检查服务器日志
  echo ""
  echo "=== 服务器日志（工具调用相关） ==="
  grep -i -E "(DEBUG|工具|tool|call|search|prime|web_reader|zread|error)" server.log | tail -50
else
  echo "无法获取 token，跳过测试"
fi

# 停止服务器
kill $SERVER_PID 2>/dev/null

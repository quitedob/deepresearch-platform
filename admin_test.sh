#!/bin/bash
# 清理之前的日志
rm -f server.log
# 启动服务器（后台）
./server.exe > server.log 2>&1 &
SERVER_PID=$!
echo "服务器 PID: $SERVER_PID"

# 等待服务器启动
sleep 3

# 使用 admin 账号登录
echo "使用 admin 账号登录..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "admin123"
  }')
echo "登录响应: $LOGIN_RESPONSE"

# 提取 token
TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
  # 尝试从 access_token 字段提取
  TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
fi

if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
  echo "Token 获取成功: ${TOKEN:0:30}..."
  
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
  sleep 15
  
  # 检查服务器日志中的工具调用
  echo ""
  echo "=== 服务器日志（工具调用相关） ==="
  tail -100 server.log | grep -i -E "(DEBUG|工具|tool|call|search|prime|web_reader|zread|error|研究|research)"
else
  echo "无法获取 token"
fi

# 停止服务器
kill $SERVER_PID 2>/dev/null

#!/bin/bash
# 清理之前的日志
rm -f server.log
# 启动服务器（后台）
./server.exe > server.log 2>&1 &
SERVER_PID=$!
echo "服务器 PID: $SERVER_PID"

# 等待服务器启动
sleep 3

# 先登录获取 token
echo "登录中..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123"
  }')
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
echo "Token: ${TOKEN:0:20}..."

# 测试深度研究 API（带认证）
echo ""
echo "发送深度研究请求..."
curl -s -X POST http://localhost:8080/api/v1/research/start \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "query": "今日金价是多少",
    "research_type": "quick",
    "sources": ["web"]
  }' 2>&1

# 等待研究进行
sleep 5

# 检查服务器日志
echo ""
echo "=== 最近的服务器日志 ==="
tail -30 server.log | grep -i -E "(研究|research|工具|tool|search|call|zread|web_reader)"

# 停止服务器
kill $SERVER_PID 2>/dev/null

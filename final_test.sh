#!/bin/bash
rm -f server.log
./server.exe > server.log 2>&1 &
SERVER_PID=$!
echo "服务器 PID: $SERVER_PID"
sleep 3

# 登录
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "email": "admin@example.com", "password": "admin123"}')
TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
  echo "Token OK, 启动研究..."
  curl -s -X POST http://localhost:8080/api/v1/research/start \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"query": "今日金价", "research_type": "quick", "sources": ["web"]}' > /dev/null
  
  echo "等待研究进行（20秒）..."
  sleep 20
  
  echo ""
  echo "=== DEBUG 日志 ==="
  cat server.log | grep -i "DEBUG"
fi

kill $SERVER_PID 2>/dev/null

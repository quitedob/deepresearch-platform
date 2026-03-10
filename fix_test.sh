#!/bin/bash
rm -f server.log
./server.exe > server.log 2>&1 &
SERVER_PID=$!
sleep 3

TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "email": "admin@example.com", "password": "admin123"}' | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$TOKEN" ]; then
  echo "启动研究..."
  curl -s -X POST http://localhost:8080/api/v1/research/start \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"query": "今日金价", "research_type": "quick", "sources": ["web"]}' > /dev/null
  
  echo "等待30秒..."
  sleep 30
  echo ""
  echo "=== 完整 DEBUG 日志 ==="
  cat server.log | grep "DEBUG" | tail -40
fi

kill $SERVER_PID 2>/dev/null

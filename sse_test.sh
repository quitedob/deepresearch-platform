#!/bin/bash
# 清理之前的日志
rm -f server.log
# 启动服务器（后台）
./server.exe > server.log 2>&1 &
SERVER_PID=$!
echo "服务器 PID: $SERVER_PID"
sleep 3

# 登录获取 token
echo "登录中..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "email": "admin@example.com", "password": "admin123"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
  echo "Token 获取成功"
  
  # 启动研究
  echo ""
  echo "启动研究任务..."
  RESEARCH_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/research/start \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"query": "今日金价是多少", "research_type": "quick", "sources": ["web"]}')
  
  SESSION_ID=$(echo "$RESEARCH_RESPONSE" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)
  echo "研究 Session ID: $SESSION_ID"
  
  # 通过 SSE 流监听进度
  echo ""
  echo "监听研究进度（30秒）..."
  timeout 30 curl -N -s "http://localhost:8080/api/v1/research/stream/$SESSION_ID?token=$TOKEN" 2>&1 &
  SSE_PID=$!
  
  # 同时等待并查看服务器日志
  sleep 30
  kill $SSE_PID 2>/dev/null
  
  echo ""
  echo "=== 服务器日志中的工具调用 ==="
  tail -50 server.log | grep -i -E "(DEBUG|工具|tool.*成功|tool.*失败|search|prime)"
else
  echo "无法获取 token"
fi

# 停止服务器
kill $SERVER_PID 2>/dev/null

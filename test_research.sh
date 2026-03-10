#!/bin/bash
# 启动服务器（后台）
./server.exe > server.log 2>&1 &
SERVER_PID=$!
echo "服务器 PID: $SERVER_PID"

# 等待服务器启动
sleep 3

# 测试深度研究 API
echo "发送深度研究请求..."
curl -X POST http://localhost:8080/research/start \
  -H "Content-Type: application/json" \
  -d '{
    "query": "今日金价",
    "research_type": "quick",
    "sources": ["web"]
  }' 2>&1

# 等待一下让请求完成
sleep 2

# 检查服务器日志中的工具调用
echo ""
echo "=== 服务器日志中的工具调用 ==="
grep -i -E "(工具|tool|call|search|mcp)" server.log | tail -20

# 停止服务器
kill $SERVER_PID 2>/dev/null

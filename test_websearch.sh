#!/bin/bash
# 清理之前的日志
rm -f server.log
# 启动服务器（后台）
./server.exe > server.log 2>&1 &
SERVER_PID=$!
echo "服务器 PID: $SERVER_PID"

# 等待服务器启动
sleep 3

# 直接测试工具调用（通过 MCP API，不需要认证）
echo ""
echo "测试 Web Search Prime 工具调用..."
curl -s -X POST "http://localhost:8080/api/v1/mcp/tools/call" \
  -H "Content-Type: application/json" \
  -d '{
    "tool_name": "web_search_prime",
    "arguments": {
      "search_query": "今日金价"
    }
  }' 2>&1 | head -50

# 等待请求完成
sleep 2

# 检查服务器日志中的工具调用
echo ""
echo "=== 服务器日志 ==="
tail -50 server.log | grep -i -E "(tool|call|search|mcp|zread|web_reader|prime)"

# 停止服务器
kill $SERVER_PID 2>/dev/null

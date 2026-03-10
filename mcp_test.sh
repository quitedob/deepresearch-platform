#!/bin/bash
# MCP + 深度研究完整流程验证测试
# 测试目标：验证 MCP 工具（web_search_prime, web_reader, zread）与深度研究流程的完整调用链

rm -f server.log mcp_test.log

echo "=========================================="
echo "MCP + 深度研究完整流程验证测试"
echo "=========================================="

# 启动服务器
echo "[1/5] 启动服务器..."
./server.exe > server.log 2>&1 &
SERVER_PID=$!
sleep 3

# 检查服务器是否启动
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ 服务器启动失败"
    cat server.log
    exit 1
fi
echo "✓ 服务器已启动 (PID: $SERVER_PID)"

# 登录获取 Token
echo "[2/5] 登录获取 Token..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "email": "admin@example.com", "password": "admin123"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败"
    echo "$LOGIN_RESPONSE"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi
echo "✓ 登录成功，Token: ${TOKEN:0:20}..."

# 获取 MCP 工具列表
echo "[3/5] 获取 MCP 工具列表..."
TOOLS_RESPONSE=$(curl -s http://localhost:8080/api/v1/mcp/tools \
  -H "Authorization: Bearer $TOKEN")

echo "可用工具："
echo "$TOOLS_RESPONSE" | grep -o '"name":"[^"]*"' | cut -d'"' -f4
TOOL_COUNT=$(echo "$TOOLS_RESPONSE" | grep -o '"count":[0-9]*' | cut -d':' -f2)
echo "工具总数: $TOOL_COUNT"

# 直接测试 MCP 工具调用（使用正确的路由 /mcp/tools/call）
echo ""
echo "[4/5] 测试 MCP 工具直接调用..."

# 测试 web_search_prime
echo ""
echo "--- 测试 web_search_prime ---"
SEARCH_RESULT=$(curl -s -X POST http://localhost:8080/api/v1/mcp/tools/call \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tool_name": "web_search_prime", "parameters": {"search_query": "Go语言并发编程最佳实践", "content_size": "medium"}}')

if echo "$SEARCH_RESULT" | grep -q '"success":true'; then
    echo "✓ web_search_prime 调用成功"
    # 提取结果长度
    RESULT_LEN=$(echo "$SEARCH_RESULT" | grep -o '"data":"[^"]*"' | wc -c)
    echo "结果长度约: $RESULT_LEN 字节"
else
    echo "❌ web_search_prime 调用失败"
    echo "$SEARCH_RESULT" | head -c 500
fi

# 测试 web_reader
echo ""
echo "--- 测试 web_reader ---"
READER_RESULT=$(curl -s -X POST http://localhost:8080/api/v1/mcp/tools/call \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tool_name": "web_reader", "parameters": {"url": "https://go.dev/doc/tutorial/getting-started"}}')

if echo "$READER_RESULT" | grep -q '"success":true'; then
    echo "✓ web_reader 调用成功"
    RESULT_LEN=$(echo "$READER_RESULT" | grep -o '"data":"[^"]*"' | wc -c)
    echo "结果长度约: $RESULT_LEN 字节"
else
    echo "❌ web_reader 调用失败"
    echo "$READER_RESULT" | head -c 500
fi

# 测试 zread_repo
echo ""
echo "--- 测试 zread_repo ---"
ZREAD_RESULT=$(curl -s -X POST http://localhost:8080/api/v1/mcp/tools/call \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"tool_name": "zread_repo", "parameters": {"operation": "get_repo_structure", "repo_name": "golang/go"}}')

if echo "$ZREAD_RESULT" | grep -q '"success":true'; then
    echo "✓ zread_repo 调用成功"
    RESULT_LEN=$(echo "$ZREAD_RESULT" | grep -o '"data":"[^"]*"' | wc -c)
    echo "结果长度约: $RESULT_LEN 字节"
else
    echo "❌ zread_repo 调用失败"
    echo "$ZREAD_RESULT" | head -c 500
fi

# 启动深度研究
echo ""
echo "[5/5] 启动深度研究（验证 MCP 工具集成）..."
RESEARCH_START=$(curl -s -X POST http://localhost:8080/api/v1/research/start \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "Go语言并发编程最佳实践", "research_type": "quick", "sources": ["web"]}')

# 检查研究启动结果
if echo "$RESEARCH_START" | grep -q '"success":true'; then
    SESSION_ID=$(echo "$RESEARCH_START" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)
    echo "✓ 研究任务已启动，Session ID: $SESSION_ID"

    # 等待研究完成
    echo "等待研究完成（最多90秒）..."
    for i in {1..90}; do
        sleep 1
        STATUS=$(curl -s "http://localhost:8080/api/v1/research/status/$SESSION_ID" \
          -H "Authorization: Bearer $TOKEN")

        if echo "$STATUS" | grep -q '"status":"completed"'; then
            echo "✓ 研究已完成！（第${i}秒）"

            # 提取关键信息
            TOOLS_USED=$(echo "$STATUS" | grep -o '"tools_used":\[[^]]*\]')
            SOURCE_COUNT=$(echo "$STATUS" | grep -o '"source_count":[0-9]*' | cut -d':' -f2)
            CONFIDENCE=$(echo "$STATUS" | grep -o '"confidence_score":[0-9.]*' | cut -d':' -f2)

            echo ""
            echo "=========================================="
            echo "研究结果摘要"
            echo "=========================================="
            echo "使用的工具: $TOOLS_USED"
            echo "信息来源数: $SOURCE_COUNT"
            echo "置信度: $CONFIDENCE"
            break
        elif echo "$STATUS" | grep -q '"status":"failed"'; then
            echo "❌ 研究失败"
            echo "$STATUS"
            break
        fi

        # 每10秒输出一次进度
        if [ $((i % 10)) -eq 0 ]; then
            PROGRESS=$(echo "$STATUS" | grep -o '"progress":[0-9.]*' | cut -d':' -f2)
            CURRENT_STAGE=$(echo "$STATUS" | grep -o '"current_stage":"[^"]*"' | cut -d'"' -f4)
            echo "进度: ${PROGRESS}%, 阶段: ${CURRENT_STAGE}"
        fi

        if [ $i -eq 90 ]; then
            echo "⚠ 研究超时，查看当前状态..."
            echo "$STATUS"
        fi
    done
else
    echo "❌ 研究任务启动失败"
    echo "$RESEARCH_START"
fi

# 输出调试日志
echo ""
echo "=========================================="
echo "DEBUG 日志（MCP 调用相关）"
echo "=========================================="
cat server.log | grep -E "DEBUG|MCP|web_search|web_reader|zread|工具|SSE|搜索" | tail -100

# 清理
kill $SERVER_PID 2>/dev/null
echo ""
echo "测试完成"

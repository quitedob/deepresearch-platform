package v1

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/datatypes"
)

// MCPAPI MCP工具API — 真正调用已注册的 eino tool
type MCPAPI struct {
	tools       []tool.InvokableTool
	toolIndex   map[string]tool.InvokableTool // name -> tool (精确匹配)
	toolCallDAO *dao.ToolCallDAO              // 工具调用记录 DAO（可选）
}

// NewMCPAPI 创建MCP API（无工具注册，纯占位）
func NewMCPAPI() *MCPAPI {
	return &MCPAPI{
		toolIndex: make(map[string]tool.InvokableTool),
	}
}

// NewMCPAPIWithTools 创建带真实工具注册的 MCP API
// P0 修复：移除模糊匹配，仅使用精确匹配 + 显式别名白名单
func NewMCPAPIWithTools(tools []tool.InvokableTool, toolCallDAO ...*dao.ToolCallDAO) *MCPAPI {
	api := &MCPAPI{
		tools:     tools,
		toolIndex: make(map[string]tool.InvokableTool),
	}
	if len(toolCallDAO) > 0 && toolCallDAO[0] != nil {
		api.toolCallDAO = toolCallDAO[0]
	}

	// 构建工具索引（精确匹配）
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil || info == nil {
			continue
		}
		api.toolIndex[info.Name] = t

		// 注册显式别名映射（白名单），保证前后端名称兼容
		switch info.Name {
		case "arxiv_search":
			api.toolIndex["arxiv"] = t
		case "web_search":
			api.toolIndex["websearch"] = t
		case "web_reader":
			api.toolIndex["webreader"] = t
		case "web_search_prime":
			api.toolIndex["search_prime"] = t
		case "zread_repo":
			api.toolIndex["zread"] = t
		}
	}
	return api
}

// GetTools 获取可用工具列表（从真实工具注册表生成）
// 不暴露工具内部参数 schema，仅暴露工具名、描述、是否启用
func (api *MCPAPI) GetTools(c *gin.Context) {
	// 需要认证才能查看工具列表
	_, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "认证失败"})
		return
	}

	toolList := make([]gin.H, 0, len(api.tools))

	for _, t := range api.tools {
		info, err := t.Info(context.Background())
		if err != nil || info == nil {
			continue
		}

		// 构建参数描述
		var params interface{}
		if info.ParamsOneOf != nil {
			if jsonSchema, err := info.ParamsOneOf.ToJSONSchema(); err == nil && jsonSchema != nil {
				params = jsonSchema
			}
		}

		toolList = append(toolList, gin.H{
			"name":         info.Name,
			"display_name": toolDisplayName(info.Name),
			"description":  info.Desc,
			"enabled":      true,
			"parameters":   params,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tools":   toolList,
		"count":   len(toolList),
	})
}

// CallTool 调用工具 — 真正执行
// P0 修复：移除模糊匹配，仅允许精确匹配的注册工具名
func (api *MCPAPI) CallTool(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "认证失败"})
		return
	}

	var req CallToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数: " + err.Error()})
		return
	}

	// P0 修复：仅从注册表精确查找工具，不做模糊匹配
	targetTool, ok := api.toolIndex[req.ToolName]
	if !ok || targetTool == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":         false,
			"error":           fmt.Sprintf("工具 '%s' 不存在或未注册", req.ToolName),
			"available_tools": api.registeredNames(),
		})
		return
	}

	// 序列化参数
	paramsJSON, err := json.Marshal(req.Parameters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "参数序列化失败"})
		return
	}

	// 执行工具（带超时保护）
	execCtx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	startTime := time.Now()
	result, execErr := targetTool.InvokableRun(execCtx, string(paramsJSON))
	elapsed := time.Since(startTime)

	// 记录工具调用（异步，不阻塞响应）
	if api.toolCallDAO != nil {
		go api.recordToolCall(userID, req.ToolName, string(paramsJSON), result, execErr, elapsed)
	}

	if execErr != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":        false,
			"error":          fmt.Sprintf("工具执行失败: %v", execErr),
			"tool_name":      req.ToolName,
			"execution_time": elapsed.Milliseconds(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"data":           result,
		"tool_name":      req.ToolName,
		"execution_time": elapsed.Milliseconds(),
	})
}

// GetToolInfo 获取工具详情
// P0 修复：移除模糊匹配
func (api *MCPAPI) GetToolInfo(c *gin.Context) {
	// 需要认证
	_, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "认证失败"})
		return
	}

	toolName := c.Param("tool_name")

	// 精确查找
	t, ok := api.toolIndex[toolName]
	if !ok || t == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success":         false,
			"error":           "工具不存在: " + toolName,
			"available_tools": api.registeredNames(),
		})
		return
	}

	info, err := t.Info(context.Background())
	if err != nil || info == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "获取工具信息失败"})
		return
	}

	var params interface{}
	if info.ParamsOneOf != nil {
		if jsonSchema, err := info.ParamsOneOf.ToJSONSchema(); err == nil && jsonSchema != nil {
			params = jsonSchema
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tool": gin.H{
			"name":         info.Name,
			"display_name": toolDisplayName(info.Name),
			"description":  info.Desc,
			"version":      "1.0.0",
			"enabled":      true,
			"parameters":   params,
		},
	})
}

// ===================== 内部辅助函数 =====================

// registeredNames 返回已注册的工具名列表（仅规范名，不含别名）
func (api *MCPAPI) registeredNames() []string {
	seen := make(map[string]bool)
	var names []string
	for _, t := range api.tools {
		info, err := t.Info(context.Background())
		if err != nil || info == nil {
			continue
		}
		if !seen[info.Name] {
			names = append(names, info.Name)
			seen[info.Name] = true
		}
	}
	return names
}

// toolDisplayName 返回工具的中文显示名
func toolDisplayName(name string) string {
	displayNames := map[string]string{
		"web_search":       "网络搜索",
		"arxiv_search":     "arXiv学术搜索",
		"wikipedia":        "维基百科",
		"web_reader":       "网页读取",
		"web_search_prime": "增强网络搜索",
		"zread_repo":       "开源仓库读取",
	}
	if dn, ok := displayNames[name]; ok {
		return dn
	}
	return name
}

// recordToolCall 记录工具调用到数据库（异步）
func (api *MCPAPI) recordToolCall(userID, toolName, input, output string, err error, elapsed time.Duration) {
	if api.toolCallDAO == nil {
		return
	}

	success := err == nil
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	inputHash := fmt.Sprintf("%x", md5.Sum([]byte(input)))
	outputHash := fmt.Sprintf("%x", md5.Sum([]byte(output)))

	inputJSON, _ := json.Marshal(map[string]interface{}{
		"tool_name":  toolName,
		"user_id":    userID,
		"parameters": input,
	})

	record := &model.ToolCallRecord{
		ID:         uuid.New().String(),
		ToolName:   toolName,
		Input:      datatypes.JSON(inputJSON),
		InputHash:  inputHash,
		OutputHash: outputHash,
		OutputLen:  len(output),
		DurationMs: elapsed.Milliseconds(),
		Success:    success,
		Error:      errorMsg,
	}

	api.toolCallDAO.Create(context.Background(), record)
}

// CallToolRequest 工具调用请求
type CallToolRequest struct {
	ToolName   string                 `json:"tool_name" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
}

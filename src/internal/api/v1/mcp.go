package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/middleware"
)

// MCPAPI MCP工具API
type MCPAPI struct {
}

// NewMCPAPI 创建MCP API
func NewMCPAPI() *MCPAPI {
	return &MCPAPI{}
}

// GetTools 获取可用工具列表
func (api *MCPAPI) GetTools(c *gin.Context) {
	tools := []gin.H{
		{
			"name":         "web_search",
			"display_name": "网络搜索",
			"description":  "搜索最新的网络信息",
			"enabled":      true,
			"parameters": gin.H{
				"query":       gin.H{"type": "string", "required": true, "description": "搜索查询"},
				"max_results": gin.H{"type": "integer", "required": false, "description": "最大结果数"},
			},
		},
		{
			"name":         "arxiv",
			"display_name": "arXiv学术搜索",
			"description":  "搜索学术论文和研究",
			"enabled":      true,
			"parameters": gin.H{
				"query":       gin.H{"type": "string", "required": true, "description": "搜索查询"},
				"max_results": gin.H{"type": "integer", "required": false, "description": "最大结果数"},
			},
		},
		{
			"name":         "wikipedia",
			"display_name": "Wikipedia",
			"description":  "搜索维基百科知识",
			"enabled":      true,
			"parameters": gin.H{
				"query":    gin.H{"type": "string", "required": true, "description": "搜索查询"},
				"language": gin.H{"type": "string", "required": false, "description": "语言代码"},
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tools":   tools,
		"count":   len(tools),
	})
}

// CallTool 调用工具
func (api *MCPAPI) CallTool(c *gin.Context) {
	_, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "认证失败"})
		return
	}

	var req CallToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数: " + err.Error()})
		return
	}

	// 模拟工具执行
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    "工具执行结果",
		"error":   "",
	})
}

// GetToolInfo 获取工具详情
func (api *MCPAPI) GetToolInfo(c *gin.Context) {
	toolName := c.Param("tool_name")

	toolInfo := map[string]gin.H{
		"web_search": {
			"name":         "web_search",
			"display_name": "网络搜索",
			"description":  "使用智谱AI进行实时网络搜索",
			"version":      "1.0.0",
			"enabled":      true,
		},
		"arxiv": {
			"name":         "arxiv",
			"display_name": "arXiv学术搜索",
			"description":  "搜索arXiv学术论文库",
			"version":      "1.0.0",
			"enabled":      true,
		},
		"wikipedia": {
			"name":         "wikipedia",
			"display_name": "Wikipedia",
			"description":  "搜索维基百科",
			"version":      "1.0.0",
			"enabled":      true,
		},
	}

	info, exists := toolInfo[toolName]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "工具不存在: " + toolName})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"tool":    info,
	})
}

// CallToolRequest 工具调用请求
type CallToolRequest struct {
	ToolName   string                 `json:"tool_name" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
}

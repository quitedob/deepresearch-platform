package tools

import "context"

// Tool 工具接口
type Tool interface {
    // Name 工具名称
    Name() string

    // Description 工具描述
    Description() string

    // Execute 执行工具
    Execute(ctx context.Context, input map[string]interface{}) (*ToolResult, error)

    // GetSchema 获取工具参数模式
    GetSchema() *ToolSchema
}

// ToolResult 工具执行结果
type ToolResult struct {
    Success bool                   `json:"success"`
    Data    map[string]interface{} `json:"data"`
    Error   string                 `json:"error,omitempty"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolSchema 工具参数模式
type ToolSchema struct {
    Type       string                    `json:"type"`
    Properties map[string]*PropertySchema `json:"properties"`
    Required   []string                  `json:"required"`
}

// PropertySchema 属性模type PropertySchema struct 
{
    Type        string      `json:"type"`
    Description string      `json:"description"`
    Default     interface{} `json:"default,omitempty"`
    Enum        []string    `json:"enum,omitempty"`
}

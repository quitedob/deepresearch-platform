package model

import (
	"time"

	"gorm.io/datatypes"
)

// ToolCallRecord 工具调用记录（用于复盘与评测）
type ToolCallRecord struct {
	ID           string         `gorm:"primaryKey;type:varchar(64)" json:"id"`
	ResearchID   string         `gorm:"type:varchar(36);index" json:"research_id"`
	ToolName     string         `gorm:"type:varchar(64);index" json:"tool_name"`
	Input        datatypes.JSON `gorm:"type:json" json:"input"`
	InputHash    string         `gorm:"type:varchar(32);index" json:"input_hash"`
	OutputHash   string         `gorm:"type:varchar(32)" json:"output_hash"`
	OutputLen    int            `gorm:"type:int" json:"output_len"`
	DurationMs   int64          `gorm:"type:bigint" json:"duration_ms"`
	Success      bool           `gorm:"type:boolean;index" json:"success"`
	Error        string         `gorm:"type:text" json:"error,omitempty"`
	RetryCount   int            `gorm:"type:int" json:"retry_count"`
	ResponseCode int            `gorm:"type:int" json:"response_code,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 表名
func (ToolCallRecord) TableName() string {
	return "tool_call_records"
}

// ToolCallStats 工具调用统计
type ToolCallStats struct {
	ToolName      string  `json:"tool_name"`
	TotalCalls    int64   `json:"total_calls"`
	SuccessCalls  int64   `json:"success_calls"`
	FailedCalls   int64   `json:"failed_calls"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
	TotalRetries  int64   `json:"total_retries"`
	SuccessRate   float64 `json:"success_rate"`
}

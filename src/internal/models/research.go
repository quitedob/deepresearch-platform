package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ResearchSession 研究会话，表示使用AI代理进行的结构化调查过程
type ResearchSession struct {
	ID           string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID       string         `gorm:"index;not null;type:uuid" json:"user_id"`
	Query        string         `gorm:"type:text;not null" json:"query"`
	Status       string         `gorm:"not null" json:"status"` // planning, executing, synthesis, completed, failed
	Progress     float32        `gorm:"default:0" json:"progress"`
	ResearchType string         `json:"research_type"` // deep, quick, academic
	Metadata     datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定ResearchSession模型的表名
func (ResearchSession) TableName() string {
	return "research_sessions"
}

// ResearchTask 研究会话中的单个任务
type ResearchTask struct {
	ID            string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ResearchID    string         `gorm:"index;not null;type:uuid" json:"research_id"`
	TaskType      string         `gorm:"not null" json:"task_type"` // search, analyze, synthesize
	ToolName      string         `json:"tool_name"`                 // web_search, wikipedia, arxiv
	Status        string         `gorm:"not null" json:"status"`    // pending, running, completed, failed
	Input         datatypes.JSON `gorm:"type:jsonb" json:"input"`
	Output        datatypes.JSON `gorm:"type:jsonb" json:"output"`
	Error         string         `gorm:"type:text" json:"error"`
	ExecutionTime int            `json:"execution_time"` // 毫秒
	CreatedAt     time.Time      `json:"created_at"`
	CompletedAt   *time.Time     `json:"completed_at"`
}

// TableName 指定ResearchTask模型的表名
func (ResearchTask) TableName() string {
	return "research_tasks"
}

// ResearchResult 研究会话的最终输出
type ResearchResult struct {
	ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ResearchID string         `gorm:"uniqueIndex;not null;type:uuid" json:"research_id"`
	Summary    string         `gorm:"type:text;not null" json:"summary"`
	Findings   datatypes.JSON `gorm:"type:jsonb" json:"findings"` // 结构化发现
	Citations  datatypes.JSON `gorm:"type:jsonb" json:"citations"` // 来源引用
	Metadata   datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
}

// TableName 指定ResearchResult模型的表名
func (ResearchResult) TableName() string {
	return "research_results"
}

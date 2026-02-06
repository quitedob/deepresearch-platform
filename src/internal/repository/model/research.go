package model

import (
    "time"
    "gorm.io/datatypes"
    "gorm.io/gorm"
)

// ResearchSession 研究会话模型
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

// TableName 指定研究会话表名
func (ResearchSession) TableName() string {
    return "research_sessions"
}

// ResearchTask 研究任务模型
type ResearchTask struct {
    ID            string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    ResearchID    string         `gorm:"index;not null;type:uuid" json:"research_id"`
    TaskType      string         `gorm:"not null" json:"task_type"` // search, analyze, synthesize
    ToolName      string         `json:"tool_name"`                 // web_search, wikipedia, arxiv
    Status        string         `gorm:"not null" json:"status"`    // pending, running, completed, failed
    Input         datatypes.JSON `gorm:"type:jsonb" json:"input"`
    Output        datatypes.JSON `gorm:"type:jsonb" json:"output"`
    Error         string         `gorm:"type:text" json:"error"`
    ExecutionTime int            `json:"execution_time"` // milliseconds
    CreatedAt     time.Time      `json:"created_at"`
    CompletedAt   *time.Time     `json:"completed_at"`
}

// TableName 指定研究任务表名
func (ResearchTask) TableName() string {
    return "research_tasks"
}

// ResearchResult 研究结果模型
type ResearchResult struct {
    ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    ResearchID string         `gorm:"uniqueIndex;not null;type:uuid" json:"research_id"`
    Summary    string         `gorm:"type:text;not null" json:"summary"`
    Findings   datatypes.JSON `gorm:"type:jsonb" json:"findings"` // structured findings
    Citations  datatypes.JSON `gorm:"type:jsonb" json:"citations"` // source citations
    Metadata   datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
    CreatedAt  time.Time      `json:"created_at"`
}

// TableName 指定研究结果表名
func (ResearchResult) TableName() string {
    return "research_results"
}

// ResearchEvidence 研究证据模型
// 用于存储研究过程中收集的证据和来源
type ResearchEvidence struct {
    ID           string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    ResearchID   string         `gorm:"index;not null;type:uuid" json:"research_id"`
    TaskID       string         `gorm:"index;type:uuid" json:"task_id"` // 关联的任务ID
    SourceType   string         `gorm:"not null" json:"source_type"`    // web, academic, wikipedia, arxiv, etc.
    SourceURL    string         `gorm:"type:text" json:"source_url"`
    SourceTitle  string         `gorm:"type:text" json:"source_title"`
    Content      string         `gorm:"type:text" json:"content"`       // 证据内容/摘要
    Relevance    float32        `gorm:"default:0" json:"relevance"`     // 相关性评分 0-1
    Credibility  float32        `gorm:"default:0" json:"credibility"`   // 可信度评分 0-1
    Metadata     datatypes.JSON `gorm:"type:jsonb" json:"metadata"`     // 额外元数据
    IsVerified   bool           `gorm:"default:false" json:"is_verified"` // 是否已验证
    CreatedAt    time.Time      `json:"created_at"`
}

// TableName 指定研究证据表名
func (ResearchEvidence) TableName() string {
    return "research_evidences"
}

// ResearchCitation 研究引用模型
// 用于存储研究报告中的引用信息
type ResearchCitation struct {
    ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    ResearchID   string    `gorm:"index;not null;type:uuid" json:"research_id"`
    EvidenceID   string    `gorm:"index;type:uuid" json:"evidence_id"` // 关联的证据ID
    CitationType string    `gorm:"not null" json:"citation_type"`      // inline, footnote, bibliography
    Position     int       `gorm:"default:0" json:"position"`          // 在报告中的位置
    Text         string    `gorm:"type:text" json:"text"`              // 引用文本
    FormattedRef string    `gorm:"type:text" json:"formatted_ref"`     // 格式化的引用（如APA格式）
    CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定研究引用表名
func (ResearchCitation) TableName() string {
    return "research_citations"
}

// ResearchFinding 研究发现模型
// 用于存储研究过程中的关键发现
type ResearchFinding struct {
    ID          string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
    ResearchID  string         `gorm:"index;not null;type:uuid" json:"research_id"`
    Category    string         `gorm:"not null" json:"category"`     // fact, insight, conclusion, recommendation
    Title       string         `gorm:"type:text" json:"title"`
    Content     string         `gorm:"type:text;not null" json:"content"`
    Confidence  float32        `gorm:"default:0" json:"confidence"`  // 置信度 0-1
    Importance  int            `gorm:"default:0" json:"importance"`  // 重要性 1-5
    EvidenceIDs datatypes.JSON `gorm:"type:jsonb" json:"evidence_ids"` // 支持此发现的证据ID列表
    Metadata    datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
    CreatedAt   time.Time      `json:"created_at"`
}

// TableName 指定研究发现表名
func (ResearchFinding) TableName() string {
    return "research_findings"
}

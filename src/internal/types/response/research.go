package response

import (
	"time"

	"github.com/ai-research-platform/internal/repository/model"
)

// ResearchSessionResponse 研究会话响应
type ResearchSessionResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Query        string    `json:"query"`
	Status       string    `json:"status"`
	Progress     float32   `json:"progress"`
	ResearchType string    `json:"research_type"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ResearchStatusResponse 研究状态响应
type ResearchStatusResponse struct {
	Success    bool                    `json:"success"`
	StatusData *ResearchStatusData     `json:"status_data"`
}

// ResearchStatusData 研究状态数据
type ResearchStatusData struct {
	SessionID    string                   `json:"session_id"`
	Status       string                   `json:"status"`
	Progress     float32                  `json:"progress"`
	ResearchType string                   `json:"research_type"`
	Query        string                   `json:"query"`
	CurrentStep  string                   `json:"current_step,omitempty"`
	Tasks        []*ResearchTaskResponse  `json:"tasks,omitempty"`
	Metadata     map[string]interface{}   `json:"metadata,omitempty"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

// ResearchTaskResponse 研究任务响应
type ResearchTaskResponse struct {
	ID            string                 `json:"id"`
	ResearchID    string                 `json:"research_id"`
	TaskType      string                 `json:"task_type"`
	ToolName      string                 `json:"tool_name"`
	Status        string                 `json:"status"`
	Input         map[string]interface{} `json:"input,omitempty"`
	Output        map[string]interface{} `json:"output,omitempty"`
	Error         string                 `json:"error,omitempty"`
	ExecutionTime int                    `json:"execution_time"`
	CreatedAt     time.Time              `json:"created_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
}

// ResearchResultResponse 研究结果响应
type ResearchResultResponse struct {
	ID         string                 `json:"id"`
	ResearchID string                 `json:"research_id"`
	Summary    string                 `json:"summary"`
	Findings   []ResearchFinding      `json:"findings"`
	Citations  []ResearchCitation     `json:"citations"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ResearchFinding 研究发现
type ResearchFinding struct {
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Confidence  float32  `json:"confidence"`
	Sources     []string `json:"sources"`
	Category    string   `json:"category,omitempty"`
}

// ResearchCitation 研究引用
type ResearchCitation struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Source    string `json:"source"` // arxiv, wikipedia, web
	Snippet   string `json:"snippet,omitempty"`
	Authors   string `json:"authors,omitempty"`
	Published string `json:"published,omitempty"`
}

// ResearchSessionsListResponse 研究会话列表响应
type ResearchSessionsListResponse struct {
	Success  bool                       `json:"success"`
	Sessions []*ResearchSessionResponse `json:"sessions"`
	Total    int64                      `json:"total"`
	Limit    int                        `json:"limit"`
	Offset   int                        `json:"offset"`
}

// ResearchStatisticsResponse 研究统计响应
type ResearchStatisticsResponse struct {
	Success    bool                `json:"success"`
	Statistics *ResearchStatistics `json:"statistics"`
}

// ResearchStatistics 研究统计数据
type ResearchStatistics struct {
	Total       int64   `json:"total"`
	Completed   int64   `json:"completed"`
	Failed      int64   `json:"failed"`
	InProgress  int64   `json:"in_progress"`
	SuccessRate float32 `json:"success_rate"`
}

// NewResearchSessionResponse 从模型转换
func NewResearchSessionResponse(session *model.ResearchSession) *ResearchSessionResponse {
	return &ResearchSessionResponse{
		ID:           session.ID,
		UserID:       session.UserID,
		Query:        session.Query,
		Status:       session.Status,
		Progress:     session.Progress,
		ResearchType: session.ResearchType,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
	}
}

// NewResearchTaskResponse 从模型转换
func NewResearchTaskResponse(task *model.ResearchTask) *ResearchTaskResponse {
	resp := &ResearchTaskResponse{
		ID:            task.ID,
		ResearchID:    task.ResearchID,
		TaskType:      task.TaskType,
		ToolName:      task.ToolName,
		Status:        task.Status,
		Error:         task.Error,
		ExecutionTime: task.ExecutionTime,
		CreatedAt:     task.CreatedAt,
		CompletedAt:   task.CompletedAt,
	}

	// 解析JSON字段
	if task.Input != nil {
		var input map[string]interface{}
		if err := task.Input.UnmarshalJSON(task.Input); err == nil {
			resp.Input = input
		}
	}

	if task.Output != nil {
		var output map[string]interface{}
		if err := task.Output.UnmarshalJSON(task.Output); err == nil {
			resp.Output = output
		}
	}

	return resp
}

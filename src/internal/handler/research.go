package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/service"
	"github.com/gin-gonic/gin"
)

// ResearchHandler 处理研究相关请求
type ResearchHandler struct {
	researchService *service.ResearchService
}

// NewResearchHandler 创建研究处理器
func NewResearchHandler(researchService *service.ResearchService) *ResearchHandler {
	return &ResearchHandler{
		researchService: researchService,
	}
}

// StartResearchRequest 表示开始研究会话的请求
type StartResearchRequest struct {
	Query        string `json:"query" binding:"required"`
	ResearchType string `json:"research_type"` // deep, quick, academic
}

// StartResearch 发起新的研究会话
func (h *ResearchHandler) StartResearch(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req StartResearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 默认研究类型
	if req.ResearchType == "" {
		req.ResearchType = "deep"
	}

	session, err := h.researchService.StartResearch(c.Request.Context(), userID, req.Query, req.ResearchType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to start research",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetSession 获取研究会话
func (h *ResearchHandler) GetSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	session, err := h.researchService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "session not found",
			"details": err.Error(),
		})
		return
	}

	// 验证所有权
	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// ListSessions 列出已认证用户的所有研究会话
func (h *ResearchHandler) ListSessions(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// 解析分页参数
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	sessions, err := h.researchService.GetUserSessions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to retrieve sessions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetResults 获取研究会话的研究结果
func (h *ResearchHandler) GetResults(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	// 验证所有权
	session, err := h.researchService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 获取结果
	results, err := h.researchService.GetResults(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "results not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, results)
}

// GetTasks 获取研究会话的研究任务
func (h *ResearchHandler) GetTasks(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	// 验证所有权
	session, err := h.researchService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 解析分页参数
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 200 {
		limit = 200
	}

	tasks, err := h.researchService.GetTasks(c.Request.Context(), sessionID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to retrieve tasks",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":  tasks,
		"limit":  limit,
		"offset": offset,
	})
}

// StreamProgress 使用SSE流式传输研究进度更新
func (h *ResearchHandler) StreamProgress(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	// 验证所有权
	session, err := h.researchService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 开始流式传输
	streamChan, err := h.researchService.StreamProgress(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to start streaming",
			"details": err.Error(),
		})
		return
	}

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 流式传输事件
	c.Stream(func(w io.Writer) bool {
		if event, ok := <-streamChan; ok {
			eventType := event.Type

			switch eventType {
			case "error":
				c.SSEvent("error", gin.H{
					"message": event.Message,
				})
				return false
			case "completed":
				c.SSEvent("completed", gin.H{
					"message": event.Message,
					"data":    event.Data,
				})
				return false
			case "cancelled":
				c.SSEvent("cancelled", gin.H{
					"message": event.Message,
				})
				return false
			case "progress":
				c.SSEvent("progress", gin.H{
					"stage":        event.Stage,
					"progress":     event.Progress,
					"message":      event.Message,
					"task_name":    event.TaskName,
					"task_status":  event.TaskStatus,
					"partial_data": event.PartialData,
					"timestamp":    event.Timestamp,
				})
				return true
			default:
				c.SSEvent("event", event)
				return true
			}
		}
		return false
	})
}

// CancelResearch 取消正在进行的研究会话
func (h *ResearchHandler) CancelResearch(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	// 验证所有权
	session, err := h.researchService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 取消研究
	if err := h.researchService.CancelResearch(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to cancel research",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "research cancelled successfully"})
}

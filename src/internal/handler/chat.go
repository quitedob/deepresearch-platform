package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/service"
	"github.com/gin-gonic/gin"
)

// ChatHandler 处理聊天相关请求
type ChatHandler struct {
	chatService *service.ChatService
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// CreateSessionRequest 表示创建聊天会话的请求
type CreateSessionRequest struct {
	Provider     string `json:"provider" binding:"required"`
	Model        string `json:"model" binding:"required"`
	Title        string `json:"title"`
	SystemPrompt string `json:"system_prompt"`
}

// SendMessageRequest 表示发送消息的请求
type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
	Stream  bool   `json:"stream"`
}

// UpdateProviderRequest 表示更新会话提供商的请求
type UpdateProviderRequest struct {
	Provider string `json:"provider" binding:"required"`
	Model    string `json:"model" binding:"required"`
}

// CreateSession 创建新的聊天会话
func (h *ChatHandler) CreateSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	session, err := h.chatService.CreateSession(c.Request.Context(), userID, req.Provider, req.Model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to create session",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetSession 获取聊天会话
func (h *ChatHandler) GetSession(c *gin.Context) {
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

	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
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

// ListSessions 列出已认证用户的所有聊天会话
func (h *ChatHandler) ListSessions(c *gin.Context) {
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

	sessions, err := h.chatService.GetUserSessions(c.Request.Context(), userID, limit, offset)
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

// DeleteSession 删除聊天会话
func (h *ChatHandler) DeleteSession(c *gin.Context) {
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
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.chatService.DeleteSession(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to delete session",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session deleted successfully"})
}

// SendMessage 向聊天会话发送消息
func (h *ChatHandler) SendMessage(c *gin.Context) {
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

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 验证所有权
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 发送消息
	message, err := h.chatService.SendMessage(c.Request.Context(), sessionID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to send message",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, message)
}

// GetMessages 获取会话的消息历史
func (h *ChatHandler) GetMessages(c *gin.Context) {
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
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
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

	messages, err := h.chatService.GetSessionHistory(c.Request.Context(), sessionID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to retrieve messages",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"limit":    limit,
		"offset":   offset,
	})
}

// StreamMessage 使用SSE流式传输消息响应
func (h *ChatHandler) StreamMessage(c *gin.Context) {
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

	content := c.Query("content")
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content query parameter is required"})
		return
	}

	// 验证所有权
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 开始流式传输
	streamChan, err := h.chatService.StreamMessage(c.Request.Context(), sessionID, content)
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

	// 流式传输数据块
	c.Stream(func(w io.Writer) bool {
		if chunk, ok := <-streamChan; ok {
			if chunk.Type == "error" {
				c.SSEvent("error", gin.H{"error": chunk.Error})
				return false
			}

			if chunk.Type == "done" {
				c.SSEvent("done", chunk.Metadata)
				return false
			}

			c.SSEvent("chunk", gin.H{
				"content":  chunk.Content,
				"metadata": chunk.Metadata,
			})
			return true
		}
		return false
	})
}

// UpdateProvider 更新会话的提供商和模型
func (h *ChatHandler) UpdateProvider(c *gin.Context) {
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

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 验证所有权
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if session.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 更新提供商
	if err := h.chatService.UpdateSessionProvider(c.Request.Context(), sessionID, req.Provider, req.Model); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update provider",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "provider updated successfully"})
}

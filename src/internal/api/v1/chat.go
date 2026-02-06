package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/pkg/llm/provider"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/ai-research-platform/internal/types/constant"
	"github.com/ai-research-platform/internal/types/request"
	"github.com/ai-research-platform/internal/types/response"
	"gorm.io/gorm"
)

// ChatAPI 聊天API
type ChatAPI struct {
	chatDAO            *dao.ChatDAO
	userPreferencesDAO *dao.UserPreferencesDAO
	membershipDAO      *dao.MembershipDAO
	modelConfigDAO     *dao.ModelConfigDAO
	llmScheduler       *eino.LLMScheduler
}

// NewChatAPI 创建聊天API
func NewChatAPI(chatDAO *dao.ChatDAO, scheduler *eino.LLMScheduler) *ChatAPI {
	return &ChatAPI{
		chatDAO:      chatDAO,
		llmScheduler: scheduler,
	}
}

// NewChatAPIWithPreferences 创建带偏好设置的聊天API
func NewChatAPIWithPreferences(chatDAO *dao.ChatDAO, prefsDAO *dao.UserPreferencesDAO, scheduler *eino.LLMScheduler) *ChatAPI {
	return &ChatAPI{
		chatDAO:            chatDAO,
		userPreferencesDAO: prefsDAO,
		llmScheduler:       scheduler,
	}
}

// NewChatAPIFull 创建完整的聊天API（包含会员和模型配置）
func NewChatAPIFull(chatDAO *dao.ChatDAO, prefsDAO *dao.UserPreferencesDAO, membershipDAO *dao.MembershipDAO, modelConfigDAO *dao.ModelConfigDAO, scheduler *eino.LLMScheduler) *ChatAPI {
	return &ChatAPI{
		chatDAO:            chatDAO,
		userPreferencesDAO: prefsDAO,
		membershipDAO:      membershipDAO,
		modelConfigDAO:     modelConfigDAO,
		llmScheduler:       scheduler,
	}
}

// friendlyLLMError 从LLM错误中提取用户友好的错误信息和错误码
func friendlyLLMError(err error) (string, string) {
	var apiErr *provider.APIError
	if errors.As(err, &apiErr) {
		return string(apiErr.Code), apiErr.UserMessage
	}
	// 非API错误，返回通用提示
	if strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "timeout") {
		return "ERR_LLM_TIMEOUT", "AI 服务响应超时，请稍后重试"
	}
	if strings.Contains(err.Error(), "connection refused") {
		return "ERR_LLM_UNAVAILABLE", "AI 服务暂时不可用，请稍后重试"
	}
	return "ERR_LLM_ERROR", "AI 服务调用失败，请稍后重试"
}

// CreateSession 创建聊天会话
func (api *ChatAPI) CreateSession(c *gin.Context) {
	var req request.CreateChatSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_INVALID_REQUEST",
				"message": "无效的请求参数",
				"details": err.Error(),
			},
		})
		return
	}

	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_UNAUTHORIZED",
				"message": "认证失败，请重新登录",
			},
		})
		return
	}

	// 验证必要参数
	if req.Title == "" {
		req.Title = "新对话"
	}
	if req.LLMProvider == "" {
		req.LLMProvider = "deepseek"
	}
	if req.ModelName == "" {
		req.ModelName = "deepseek-chat"
	}

	session := &model.ChatSession{
		UserID:       userID,
		Title:        req.Title,
		Provider:     req.LLMProvider,
		Model:        req.ModelName,
		SystemPrompt: req.SystemPrompt,
		MessageCount: 0,
	}

	if api.chatDAO != nil {
		if err := api.chatDAO.CreateSession(c.Request.Context(), session); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_INTERNAL_ERROR",
					"message": "创建会话失败",
					"details": err.Error(),
				},
			})
			return
		}
	} else {
		session.ID = "session_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": response.ChatSessionResponse{
			ID:           session.ID,
			UserID:       session.UserID,
			Title:        session.Title,
			Provider:     session.Provider,
			Model:        session.Model,
			SystemPrompt: session.SystemPrompt,
			MessageCount: session.MessageCount,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		},
	})
}


// 会话查询的最大限制
const maxSessionLimit = 100

// GetSessions 获取聊天会话列表
// 修复：添加hasMore字段，统一分页响应格式
func (api *ChatAPI) GetSessions(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit < 1 {
		limit = 20
	}
	if limit > maxSessionLimit {
		limit = maxSessionLimit
	}
	if offset < 0 {
		offset = 0
	}

	var sessions []*model.ChatSession
	var total int64

	if api.chatDAO != nil {
		sessions, err = api.chatDAO.ListSessionsByUserID(c.Request.Context(), userID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会话列表失败"})
			return
		}
		total, _ = api.chatDAO.CountSessionsByUserID(c.Request.Context(), userID)
	}

	sessionResponses := make([]*response.ChatSessionResponse, len(sessions))
	// 批量获取所有会话的最后一条消息（避免N+1查询）
	sessionIDs := make([]string, len(sessions))
	for i, s := range sessions {
		sessionIDs[i] = s.ID
	}
	lastMessages := make(map[string]*model.Message)
	if api.chatDAO != nil && len(sessionIDs) > 0 {
		var err2 error
		lastMessages, err2 = api.chatDAO.GetLastMessagesBySessionIDs(c.Request.Context(), sessionIDs)
		if err2 != nil {
			lastMessages = make(map[string]*model.Message)
		}
	}

	for i, s := range sessions {
		var lastMessage string
		if msg, ok := lastMessages[s.ID]; ok && msg != nil {
			// 截取前100个字符作为预览
			if len(msg.Content) > 100 {
				lastMessage = msg.Content[:100] + "..."
			} else {
				lastMessage = msg.Content
			}
		}

		sessionResponses[i] = response.NewChatSessionResponseWithLastMessage(s, lastMessage)
	}

	// 修复：添加hasMore和maxLimit字段
	hasMore := int64(offset+len(sessions)) < total

	c.JSON(http.StatusOK, gin.H{
		"sessions":  sessionResponses,
		"total":     int(total),
		"limit":     limit,
		"offset":    offset,
		"has_more":  hasMore,
		"max_limit": maxSessionLimit,
	})
}

// GetSession 获取聊天会话详情
func (api *ChatAPI) GetSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_UNAUTHORIZED",
				"message": "认证失败，请重新登录",
			},
		})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_MISSING_PARAMETER",
				"message": "会话ID不能为空",
				"field":   "id",
			},
		})
		return
	}

	var session *model.ChatSession
	if api.chatDAO != nil {
		session, err = api.chatDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_NOT_FOUND",
					"message": "会话不存在或已被删除",
				},
			})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_FORBIDDEN",
					"message": "无权访问此会话",
				},
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": response.ChatSessionResponse{
			ID:           session.ID,
			UserID:       session.UserID,
			Title:        session.Title,
			Provider:     session.Provider,
			Model:        session.Model,
			SystemPrompt: session.SystemPrompt,
			MessageCount: session.MessageCount,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		},
	})
}

// UpdateSession 更新聊天会话
func (api *ChatAPI) UpdateSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID不能为空"})
		return
	}

	var req request.UpdateChatSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	if api.chatDAO != nil {
		session, err := api.chatDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此会话"})
			return
		}

		if req.Title != nil {
			session.Title = *req.Title
		}
		if req.SystemPrompt != nil {
			session.SystemPrompt = *req.SystemPrompt
		}

		if err := api.chatDAO.UpdateSession(c.Request.Context(), session); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新会话失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "会话更新成功"})
}

// DeleteSession 删除聊天会话
// 修复：使用事务确保级联删除会话和消息
func (api *ChatAPI) DeleteSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_UNAUTHORIZED",
				"message": "认证失败，请重新登录",
			},
		})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_MISSING_PARAMETER",
				"message": "会话ID不能为空",
				"field":   "id",
			},
		})
		return
	}

	if api.chatDAO != nil {
		session, err := api.chatDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_NOT_FOUND",
					"message": "会话不存在或已被删除",
				},
			})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_FORBIDDEN",
					"message": "无权删除此会话",
				},
			})
			return
		}

		// 使用事务删除会话及其消息
		if err := api.chatDAO.DeleteSessionWithMessages(c.Request.Context(), sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_INTERNAL_ERROR",
					"message": "删除会话失败",
					"details": err.Error(),
				},
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "会话删除成功",
		"session_id": sessionID,
	})
}


// 消息查询的最大限制
const maxMessageLimit = 200

// GetMessages 获取会话消息列表
// 修复：添加hasMore字段，统一分页响应格式，明确limit限制
func (api *ChatAPI) GetMessages(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_UNAUTHORIZED",
				"message": "认证失败，请重新登录",
			},
		})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_MISSING_PARAMETER",
				"message": "会话ID不能为空",
				"field":   "id",
			},
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 修复：明确limit限制，并在响应中告知前端
	if limit < 1 {
		limit = 50
	}
	if limit > maxMessageLimit {
		limit = maxMessageLimit
	}
	if offset < 0 {
		offset = 0
	}

	var messages []*model.Message
	var total int64

	if api.chatDAO != nil {
		session, err := api.chatDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_NOT_FOUND",
					"message": "会话不存在或已被删除",
				},
			})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_FORBIDDEN",
					"message": "无权访问此会话",
				},
			})
			return
		}

		messages, err = api.chatDAO.GetMessagesBySessionID(c.Request.Context(), sessionID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_INTERNAL_ERROR",
					"message": "获取消息失败",
					"details": err.Error(),
				},
			})
			return
		}
		total, _ = api.chatDAO.CountMessagesBySessionID(c.Request.Context(), sessionID)
	}

	messageResponses := make([]*response.ChatMessageResponse, len(messages))
	for i, m := range messages {
		messageResponses[i] = &response.ChatMessageResponse{
			ID:         m.ID,
			SessionID:  m.SessionID,
			Role:       m.Role,
			Content:    m.Content,
			TokensUsed: m.TokensUsed,
			CreatedAt:  m.CreatedAt,
		}
	}

	// 修复：添加hasMore和maxLimit字段
	hasMore := int64(offset+len(messages)) < total

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"messages":  messageResponses,
		"total":     int(total),
		"limit":     limit,
		"offset":    offset,
		"has_more":  hasMore,
		"max_limit": maxMessageLimit,
	})
}

// ClearMessages 清空会话消息
// 修复：使用事务确保消息删除和计数更新的原子性
func (api *ChatAPI) ClearMessages(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID不能为空"})
		return
	}

	if api.chatDAO != nil {
		session, err := api.chatDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此会话"})
			return
		}

		// 使用事务删除消息并更新计数
		db := api.chatDAO.GetDB()
		err = db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			// 删除会话的所有消息
			if err := tx.Where("session_id = ?", sessionID).Delete(&model.Message{}).Error; err != nil {
				return err
			}
			// 重置消息计数
			return tx.Model(&model.ChatSession{}).Where("id = ?", sessionID).Update("message_count", 0).Error
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "清空消息失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "消息已清空"})
}

// Chat 发送聊天消息（非流式）
func (api *ChatAPI) Chat(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_UNAUTHORIZED",
				"message": "认证失败，请重新登录",
			},
		})
		return
	}

	var req request.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_INVALID_REQUEST",
				"message": "无效的请求参数",
				"details": err.Error(),
			},
		})
		return
	}

	if req.Stream {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_INVALID_REQUEST",
				"message": "此接口不支持流式输出，请使用 /chat/chat/stream 接口",
			},
		})
		return
	}

	// 检查并扣减聊天配额（原子操作，先扣减后执行）
	var quotaDeducted bool
	if api.membershipDAO != nil {
		hasQuota, remaining, limit, err := api.membershipDAO.CheckAndDeductChatQuota(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_INTERNAL_ERROR",
					"message": "检查配额失败",
				},
			})
			return
		}
		if !hasQuota {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_CHAT_QUOTA_EXCEEDED",
					"message": "聊天配额已用完，请升级会员或等待配额重置",
					"extra": gin.H{
						"remaining": remaining,
						"limit":     limit,
					},
				},
			})
			return
		}
		quotaDeducted = true
	}

	// 如果后续处理失败，需要退还配额
	defer func() {
		if quotaDeducted && recover() != nil {
			api.membershipDAO.RefundChatQuota(c.Request.Context(), userID)
		}
	}()

	startTime := time.Now()

	var session *model.ChatSession
	var providerName, modelName string

	if api.chatDAO != nil {
		session, err = api.chatDAO.GetSessionByID(c.Request.Context(), req.SessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_NOT_FOUND",
					"message": "会话不存在或已被删除",
				},
			})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_FORBIDDEN",
					"message": "无权访问此会话",
				},
			})
			return
		}
		providerName = session.Provider
		modelName = session.Model

		// 根据请求切换模型
		if req.UseDeepThink {
			// 切换到深度思考模型
			modelName = api.getDeepThinkingModel(providerName, modelName)
		}
	} else {
		providerName = constant.ProviderDeepSeek
		modelName = "deepseek-chat"
	}

	// 构建消息
	var messageContent string
	if req.UseWebSearch {
		// 添加网络搜索提示
		messageContent = fmt.Sprintf("[使用网络搜索]\n%s", req.Message)
	} else {
		messageContent = req.Message
	}

	// 构建完整的上下文消息
	messages := api.buildContextMessages(c.Request.Context(), userID, req.SessionID, messageContent)

	var responseContent string
	var tokensUsed int

	if api.llmScheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_LLM_UNAVAILABLE",
				"message": "LLM服务暂时不可用，请稍后重试",
			},
		})
		return
	}

	if !api.llmScheduler.SupportsModel(modelName) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_MODEL_NOT_SUPPORTED",
				"message": fmt.Sprintf("模型 %s 未注册或不支持", modelName),
				"extra": gin.H{
					"model": modelName,
				},
			},
		})
		return
	}

	result, err := api.llmScheduler.ExecuteWithFallback(c.Request.Context(), messages, modelName)
	if err != nil {
		errCode, errMsg := friendlyLLMError(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    errCode,
				"message": errMsg,
			},
		})
		return
	}
	
	responseContent = result.Content
	// 估算token使用量（实际应该从LLM响应中获取）
	tokensUsed = len(req.Message)/4 + len(responseContent)/4

	if api.chatDAO != nil {
		userMsg := &model.Message{
			SessionID:  req.SessionID,
			Role:       constant.RoleUser,
			Content:    req.Message,
			TokensUsed: 0,
		}
		api.chatDAO.CreateMessage(c.Request.Context(), userMsg)

		assistantMsg := &model.Message{
			SessionID:  req.SessionID,
			Role:       constant.RoleAssistant,
			Content:    responseContent,
			TokensUsed: tokensUsed,
		}
		api.chatDAO.CreateMessage(c.Request.Context(), assistantMsg)
	}

	// 配额已在请求开始时扣减，无需再次增加
	// 如果需要在失败时退还配额，可以在错误处理中调用 RefundChatQuota

	c.JSON(http.StatusOK, response.ChatResponse{
		SessionID:    req.SessionID,
		MessageID:    "msg_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Content:      responseContent,
		Role:         constant.RoleAssistant,
		TokensUsed:   tokensUsed,
		Model:        modelName,
		Provider:     providerName,
		Stream:       false,
		ResponseTime: time.Since(startTime).Milliseconds(),
	})
}

// getDeepThinkingModel 根据提供商获取深度思考模型
func (api *ChatAPI) getDeepThinkingModel(provider string, currentModel string) string {
	// 如果已经是深度思考模型，直接返回
	if constant.IsDeepThinkingModel(currentModel) {
		return currentModel
	}

	// 根据提供商返回对应的深度思考模型
	switch provider {
	case constant.ProviderDeepSeek:
		return "deepseek-reasoner"
	case constant.ProviderZhipu:
		return "glm-4.7"
	case constant.ProviderOllama:
		return "gemma3:12b"
	default:
		return currentModel
	}
}


// ChatWebSearch 联网搜索聊天（非流式）
// 使用 WebSearchTool 进行网络搜索，然后将搜索结果作为上下文传给 LLM
func (api *ChatAPI) ChatWebSearch(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_UNAUTHORIZED",
				"message": "认证失败，请重新登录",
			},
		})
		return
	}

	var req request.WebSearchChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_INVALID_REQUEST",
				"message": "无效的请求参数",
				"details": err.Error(),
			},
		})
		return
	}

	// 检查并扣减聊天配额（原子操作）
	var quotaDeducted bool
	if api.membershipDAO != nil {
		hasQuota, remaining, limit, err := api.membershipDAO.CheckAndDeductChatQuota(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_INTERNAL_ERROR",
					"message": "检查配额失败",
				},
			})
			return
		}
		if !hasQuota {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_CHAT_QUOTA_EXCEEDED",
					"message": "聊天配额已用完，请升级会员或等待配额重置",
					"extra": gin.H{
						"remaining": remaining,
						"limit":     limit,
					},
				},
			})
			return
		}
		quotaDeducted = true
	}

	// 如果后续处理失败，需要退还配额
	defer func() {
		if quotaDeducted && recover() != nil {
			api.membershipDAO.RefundChatQuota(c.Request.Context(), userID)
		}
	}()

	startTime := time.Now()

	var session *model.ChatSession
	var providerName, modelName string

	if api.chatDAO != nil {
		session, err = api.chatDAO.GetSessionByID(c.Request.Context(), req.SessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_NOT_FOUND",
					"message": "会话不存在或已被删除",
				},
			})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_FORBIDDEN",
					"message": "无权访问此会话",
				},
			})
			return
		}
		providerName = session.Provider
		modelName = session.Model
	} else {
		providerName = constant.ProviderDeepSeek
		modelName = "deepseek-chat"
	}

	// 1. 使用 WebSearchTool 进行网络搜索
	webSearchTool := eino.CreateWebSearchTool()
	searchArgs := fmt.Sprintf(`{"query": "%s"}`, req.Message)
	searchResult, searchErr := webSearchTool.InvokableRun(c.Request.Context(), searchArgs)
	
	// 2. 构建包含搜索结果的提示词
	var searchPrompt string
	if searchErr != nil || searchResult == "" {
		// 搜索失败时，仍然尝试让 LLM 回答
		searchPrompt = fmt.Sprintf(`用户问题: %s

注意：网络搜索暂时不可用，请根据您的知识尽可能回答用户的问题。`, req.Message)
	} else {
		// 搜索成功，将搜索结果作为上下文
		searchPrompt = fmt.Sprintf(`用户问题: %s

以下是网络搜索获取的最新信息：
%s

请根据以上搜索结果，为用户提供详细、准确的回答。请在回答中适当引用搜索结果中的信息来源。`, req.Message, searchResult)
	}

	messages := api.buildContextMessages(c.Request.Context(), userID, req.SessionID, searchPrompt)

	if api.llmScheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_LLM_UNAVAILABLE",
				"message": "LLM服务暂时不可用，请稍后重试",
			},
		})
		return
	}

	result, err := api.llmScheduler.ExecuteWithFallback(c.Request.Context(), messages, modelName)
	if err != nil {
		errCode, errMsg := friendlyLLMError(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    errCode,
				"message": errMsg,
			},
		})
		return
	}

	responseContent := result.Content
	tokensUsed := len(req.Message)/4 + len(responseContent)/4

	// 保存消息（保存原始用户消息，不包含搜索结果）
	if api.chatDAO != nil {
		userMsg := &model.Message{
			SessionID:  req.SessionID,
			Role:       constant.RoleUser,
			Content:    "[联网搜索] " + req.Message,
			TokensUsed: 0,
		}
		api.chatDAO.CreateMessage(c.Request.Context(), userMsg)

		assistantMsg := &model.Message{
			SessionID:  req.SessionID,
			Role:       constant.RoleAssistant,
			Content:    responseContent,
			TokensUsed: tokensUsed,
		}
		api.chatDAO.CreateMessage(c.Request.Context(), assistantMsg)
	}

	// 配额已在请求开始时扣减，无需再次增加

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": gin.H{
			"role":    constant.RoleAssistant,
			"content": responseContent,
		},
		"session_id":    req.SessionID,
		"tokens_used":   tokensUsed,
		"model":         modelName,
		"provider":      providerName,
		"response_time": time.Since(startTime).Milliseconds(),
	})
}

// ChatStream 发送聊天消息（流式）
// 修复：添加并发安全的内容累积、SSE连接清理、超时控制
func (api *ChatAPI) ChatStream(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_UNAUTHORIZED",
				"message": "认证失败，请重新登录",
			},
		})
		return
	}

	var req request.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_INVALID_REQUEST",
				"message": "无效的请求参数",
				"details": err.Error(),
			},
		})
		return
	}

	// 检查并扣减聊天配额（原子操作）
	var quotaDeducted bool
	if api.membershipDAO != nil {
		hasQuota, remaining, limit, err := api.membershipDAO.CheckAndDeductChatQuota(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_INTERNAL_ERROR",
					"message": "检查配额失败",
				},
			})
			return
		}
		if !hasQuota {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_CHAT_QUOTA_EXCEEDED",
					"message": "聊天配额已用完，请升级会员或等待配额重置",
					"extra": gin.H{
						"remaining": remaining,
						"limit":     limit,
					},
				},
			})
			return
		}
		quotaDeducted = true
		_ = quotaDeducted // 标记已扣减，用于后续可能的退还逻辑
	}

	var session *model.ChatSession
	var modelName string

	if api.chatDAO != nil {
		session, err = api.chatDAO.GetSessionByID(c.Request.Context(), req.SessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_NOT_FOUND",
					"message": "会话不存在或已被删除",
				},
			})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_SESSION_FORBIDDEN",
					"message": "无权访问此会话",
				},
			})
			return
		}
		modelName = session.Model
	} else {
		modelName = "deepseek-chat"
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 构建完整的上下文消息（包含记忆功能）
	messages := api.buildContextMessages(c.Request.Context(), userID, req.SessionID, req.Message)

	c.SSEvent("message", gin.H{"type": "start", "content": ""})
	c.Writer.Flush()

	// 创建带超时的上下文，防止长时间运行
	streamCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	if api.llmScheduler != nil {
		reader, _, err := api.llmScheduler.StreamWithFallback(streamCtx, messages, modelName)
		if err != nil {
			_, errMsg := friendlyLLMError(err)
			c.SSEvent("message", gin.H{"type": "error", "error": errMsg})
			c.Writer.Flush()
			return
		}

		// 使用 strings.Builder 进行高效的字符串拼接（线程安全由单goroutine保证）
		var contentBuilder strings.Builder
		
		// 监听客户端断开连接
		clientGone := c.Request.Context().Done()
		
	streamLoop:
		for {
			select {
			case <-clientGone:
				// 客户端断开连接，清理资源
				break streamLoop
			case <-streamCtx.Done():
				// 超时，发送错误并退出
				c.SSEvent("message", gin.H{"type": "error", "error": "流式响应超时"})
				c.Writer.Flush()
				break streamLoop
			default:
				chunk, err := reader.Recv()
				if err != nil {
					// EOF 是正常结束，其他错误需要通知前端
					if err.Error() != "EOF" {
						c.SSEvent("message", gin.H{"type": "error", "error": err.Error()})
						c.Writer.Flush()
					}
					break streamLoop
				}
				contentBuilder.WriteString(chunk.Content)
				c.SSEvent("message", gin.H{"type": "content", "content": chunk.Content})
				c.Writer.Flush()
			}
		}

		fullContent := contentBuilder.String()

		// 只有在有内容时才保存消息
		if fullContent != "" && api.chatDAO != nil {
			userMsg := &model.Message{
				SessionID: req.SessionID,
				Role:      constant.RoleUser,
				Content:   req.Message,
			}
			if err := api.chatDAO.CreateMessage(c.Request.Context(), userMsg); err != nil {
				// 记录错误但不中断流程
				c.SSEvent("warning", gin.H{"warning": "保存用户消息失败"})
			}

			assistantMsg := &model.Message{
				SessionID: req.SessionID,
				Role:      constant.RoleAssistant,
				Content:   fullContent,
			}
			if err := api.chatDAO.CreateMessage(c.Request.Context(), assistantMsg); err != nil {
				c.SSEvent("warning", gin.H{"warning": "保存助手消息失败"})
			}
		}

		// 配额已在请求开始时扣减，无需再次增加
	} else {
		content := "这是一个流式回复的示例。(LLM调度器未配置)"
		for _, char := range content {
			c.SSEvent("message", gin.H{"type": "content", "content": string(char)})
			c.Writer.Flush()
			time.Sleep(time.Millisecond * 30)
		}
	}

	c.SSEvent("message", gin.H{"type": "end", "content": ""})
	c.Writer.Flush()
}

// GetModels 获取可用模型列表
func (api *ChatAPI) GetModels(c *gin.Context) {
	provider := c.Query("provider")

	// 模型元数据映射
	modelMetadata := map[string]response.ModelInfo{
		// DeepSeek
		"deepseek-chat":     {ID: "deepseek-chat", Name: "deepseek-chat", DisplayName: "DeepSeek Chat", Provider: constant.ProviderDeepSeek, Description: "DeepSeek-V3 非思考模式", ContextLen: 128000, MaxTokens: 8000, Capabilities: []string{"streaming", "tools", "json_output"}},
		"deepseek-reasoner": {ID: "deepseek-reasoner", Name: "deepseek-reasoner", DisplayName: "DeepSeek Reasoner", Provider: constant.ProviderDeepSeek, Description: "DeepSeek-V3 思考模式", ContextLen: 128000, MaxTokens: 64000, Capabilities: []string{"streaming", "reasoning", "json_output"}},
		// 智谱AI
		"glm-4.7":      {ID: "glm-4.7", Name: "glm-4.7", DisplayName: "GLM-4.7", Provider: constant.ProviderZhipu, Description: "智谱AI高智能旗舰模型", ContextLen: 200000, MaxTokens: 128000, Capabilities: []string{"streaming", "tools", "web_search"}},
		"glm-4.5-air":  {ID: "glm-4.5-air", Name: "glm-4.5-air", DisplayName: "GLM-4.5-Air", Provider: constant.ProviderZhipu, Description: "智谱AI高性价比模型", ContextLen: 128000, MaxTokens: 96000, Capabilities: []string{"streaming", "tools"}},
		// Ollama
		"gemma3:4b":  {ID: "gemma3:4b", Name: "gemma3:4b", DisplayName: "Gemma 3 4B", Provider: constant.ProviderOllama, Description: "Google Gemma 3 4B本地模型", ContextLen: 8192, MaxTokens: 4096, Capabilities: []string{"streaming", "json_output", "tools"}},
		"gemma3:12b": {ID: "gemma3:12b", Name: "gemma3:12b", DisplayName: "Gemma 3 12B", Provider: constant.ProviderOllama, Description: "Google Gemma 3 12B本地模型", ContextLen: 8192, MaxTokens: 4096, Capabilities: []string{"streaming", "json_output", "tools"}},
		"qwen3:8b":   {ID: "qwen3:8b", Name: "qwen3:8b", DisplayName: "Qwen 3 8B", Provider: constant.ProviderOllama, Description: "阿里通义千问3本地模型", ContextLen: 32768, MaxTokens: 4096, Capabilities: []string{"streaming", "json_output", "tools"}},
		// OpenRouter
		"moonshotai/kimi-k2:free": {ID: "moonshotai/kimi-k2:free", Name: "moonshotai/kimi-k2:free", DisplayName: "Kimi K2 (Free)", Provider: constant.ProviderOpenRouter, Description: "Moonshot AI Kimi K2 免费模型", ContextLen: 128000, MaxTokens: 8192, Capabilities: []string{"streaming", "tools", "json_output"}},
	}

	// 获取管理员配置的启用模型
	var enabledModels map[string]bool
	var enabledProviders map[string]bool
	if api.modelConfigDAO != nil {
		enabledModels = make(map[string]bool)
		enabledProviders = make(map[string]bool)
		
		// 获取启用的提供商
		providerConfigs, _ := api.modelConfigDAO.GetEnabledProviders(c.Request.Context())
		for _, p := range providerConfigs {
			enabledProviders[p.Provider] = true
		}
		
		// 获取启用的模型
		modelConfigs, _ := api.modelConfigDAO.GetEnabledModels(c.Request.Context())
		for _, m := range modelConfigs {
			enabledModels[m.ModelName] = true
		}
	}

	models := make([]response.ModelInfo, 0)

	// 从LLMScheduler获取已注册的模型
	if api.llmScheduler != nil {
		registeredModels := api.llmScheduler.GetRegisteredModels()
		for modelName, providerKey := range registeredModels {
			// providerKey 格式为 "provider:model"，需要提取真正的 provider 名称
			realProvider := providerKey
			if idx := strings.Index(providerKey, ":"); idx != -1 {
				realProvider = providerKey[:idx]
			}
			
			// 检查提供商是否启用（如果有配置的话）
			if enabledProviders != nil && len(enabledProviders) > 0 && !enabledProviders[realProvider] {
				continue
			}
			// 检查模型是否启用（如果有配置的话）
			if enabledModels != nil && len(enabledModels) > 0 && !enabledModels[modelName] {
				continue
			}
			
			// 使用元数据，如果没有则创建基本信息
			if meta, ok := modelMetadata[modelName]; ok {
				if provider == "" || meta.Provider == provider {
					models = append(models, meta)
				}
			} else {
				// 如果没有元数据，创建基本模型信息
				if provider == "" || realProvider == provider {
					models = append(models, response.ModelInfo{
						ID:           modelName,
						Name:         modelName,
						DisplayName:  modelName,
						Provider:     realProvider,
						Description:  "LLM模型",
						Capabilities: []string{"streaming"},
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, response.ModelListResponse{Models: models, Total: len(models)})
}


// 历史消息查询的最大限制，防止OOM
const maxHistoryMessages = 100

// buildContextMessages 构建完整的上下文消息
// 顺序: 系统安全提示词 → 用户自定义提示词 → 会话系统提示词 → 历史消息 → 当前消息
// 修复：限制历史消息数量防止OOM，添加错误处理
func (api *ChatAPI) buildContextMessages(ctx context.Context, userID string, sessionID string, currentMessage string) []*eino.Message {
	// 预分配合理的容量
	messages := make([]*eino.Message, 0, maxHistoryMessages+10)

	// 1. 添加系统安全提示词（最高优先级）
	safetyPrompt := os.Getenv("SYSTEM_SAFETY_PROMPT")
	if safetyPrompt != "" {
		messages = append(messages, &eino.Message{
			Role:    "system",
			Content: safetyPrompt,
		})
	}

	// 2. 获取用户偏好设置
	var memoryEnabled bool = true
	var customPrompt string
	var maxContextTokens int = 128000

	if api.userPreferencesDAO != nil {
		prefs, err := api.userPreferencesDAO.GetOrCreate(ctx, userID)
		if err == nil && prefs != nil {
			memoryEnabled = prefs.MemoryEnabled
			customPrompt = prefs.CustomSystemPrompt
			if prefs.MaxContextTokens > 0 {
				maxContextTokens = prefs.MaxContextTokens
			}
		}
		// 注意：如果获取偏好失败，使用默认值继续执行
	}

	// 3. 添加用户自定义提示词（限制长度防止内存溢出）
	if customPrompt != "" {
		// 限制自定义提示词最大长度为10000字符
		if len(customPrompt) > 10000 {
			customPrompt = customPrompt[:10000]
		}
		messages = append(messages, &eino.Message{
			Role:    "system",
			Content: customPrompt,
		})
	}

	// 4. 获取会话信息和系统提示词
	if api.chatDAO != nil && sessionID != "" {
		session, err := api.chatDAO.GetSessionByID(ctx, sessionID)
		if err == nil && session != nil && session.SystemPrompt != "" {
			// 限制系统提示词长度
			systemPrompt := session.SystemPrompt
			if len(systemPrompt) > 10000 {
				systemPrompt = systemPrompt[:10000]
			}
			messages = append(messages, &eino.Message{
				Role:    "system",
				Content: systemPrompt,
			})
		}
	}

	// 5. 如果启用记忆功能，添加历史消息
	if memoryEnabled && api.chatDAO != nil && sessionID != "" {
		// 修复：限制历史消息数量，防止加载过多数据导致OOM
		historyMessages, err := api.chatDAO.GetMessagesBySessionID(ctx, sessionID, maxHistoryMessages, 0)
		if err == nil && len(historyMessages) > 0 {
			// 计算当前token使用量，确保不超过限制
			currentTokens := 0
			for _, msg := range historyMessages {
				msgTokens := estimateTokens(msg.Content)
				// 如果添加这条消息会超过限制，停止添加
				if currentTokens+msgTokens > maxContextTokens*8/10 { // 保留20%给当前消息
					break
				}
				currentTokens += msgTokens
				messages = append(messages, &eino.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
		// 注意：查询失败时返回空历史，调用者可以正常继续
	}

	// 6. 添加当前用户消息
	messages = append(messages, &eino.Message{
		Role:    "user",
		Content: currentMessage,
	})

	return messages
}

// GetContextStatus 获取上下文状态
// 修复：返回整数百分比，限制查询数量
func (api *ChatAPI) GetContextStatus(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID不能为空"})
		return
	}

	// 验证会话权限
	if api.chatDAO != nil {
		session, err := api.chatDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此会话"})
			return
		}
	}

	// 计算上下文状态
	maxTokens := 128000
	warningThreshold := 100000
	currentTokens := 0
	messageCount := 0
	memoryEnabled := true

	// 获取用户偏好
	if api.userPreferencesDAO != nil {
		prefs, err := api.userPreferencesDAO.GetOrCreate(c.Request.Context(), userID)
		if err == nil && prefs != nil {
			memoryEnabled = prefs.MemoryEnabled
			if prefs.MaxContextTokens > 0 {
				maxTokens = prefs.MaxContextTokens
			}
		}
		// 修复：如果获取偏好失败，使用默认值继续
	}

	// 计算当前 token 使用量（限制查询数量防止OOM）
	if api.chatDAO != nil {
		messages, _ := api.chatDAO.GetMessagesBySessionID(c.Request.Context(), sessionID, maxHistoryMessages, 0)
		for _, msg := range messages {
			currentTokens += estimateTokens(msg.Content)
		}
		messageCount = len(messages)
	}

	// 修复：返回整数百分比，便于前端处理
	usagePercent := int(float64(currentTokens) / float64(maxTokens) * 100)
	if usagePercent > 100 {
		usagePercent = 100
	}
	isNearLimit := currentTokens >= warningThreshold
	isOverLimit := currentTokens >= maxTokens

	c.JSON(http.StatusOK, gin.H{
		"current_tokens":     currentTokens,
		"max_tokens":         maxTokens,
		"warning_threshold":  warningThreshold,
		"usage_percent":      usagePercent,
		"message_count":      messageCount,
		"memory_enabled":     memoryEnabled,
		"is_near_limit":      isNearLimit,
		"is_over_limit":      isOverLimit,
	})
}

// SummarizeAndNewSession 总结当前会话并创建新会话
// 修复：限制消息查询数量，添加错误处理
func (api *ChatAPI) SummarizeAndNewSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID不能为空"})
		return
	}

	// 验证会话权限
	var oldSession *model.ChatSession
	if api.chatDAO != nil {
		oldSession, err = api.chatDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		if oldSession.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此会话"})
			return
		}
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "服务不可用"})
		return
	}

	// 获取消息（限制数量防止OOM）
	messages, err := api.chatDAO.GetMessagesBySessionID(c.Request.Context(), sessionID, maxHistoryMessages, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败"})
		return
	}

	if len(messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话没有消息，无需总结"})
		return
	}

	// 构建总结请求（限制内容长度）
	var contentBuilder strings.Builder
	contentBuilder.WriteString("请总结以下对话的主要内容和关键信息，用简洁的语言概括：\n\n")
	totalLen := 0
	const maxContentLen = 50000 // 限制总结内容最大长度
	for _, msg := range messages {
		line := fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		if totalLen+len(line) > maxContentLen {
			contentBuilder.WriteString("...(内容过长，已截断)")
			break
		}
		contentBuilder.WriteString(line)
		totalLen += len(line)
	}

	// 调用 LLM 生成总结
	summaryMessages := []*eino.Message{
		{Role: "system", Content: "你是一个专业的对话总结助手，请用简洁的语言总结对话内容，保留关键信息。"},
		{Role: "user", Content: contentBuilder.String()},
	}

	var summary string
	if api.llmScheduler != nil {
		summaryResult, err := api.llmScheduler.ExecuteWithFallback(c.Request.Context(), summaryMessages, oldSession.Model)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成总结失败: " + err.Error()})
			return
		}
		summary = summaryResult.Content
	} else {
		summary = "（无法生成总结，LLM调度器未初始化）"
	}

	// 创建新会话
	newSession := &model.ChatSession{
		UserID:       userID,
		Title:        "续: " + oldSession.Title,
		Provider:     oldSession.Provider,
		Model:        oldSession.Model,
		ModelType:    oldSession.ModelType,
		SystemPrompt: fmt.Sprintf("这是一个延续的对话。上一个对话的总结：\n%s", summary),
		MessageCount: 0,
	}

	if err := api.chatDAO.CreateSession(c.Request.Context(), newSession); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建新会话失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":        true,
		"new_session_id": newSession.ID,
		"summary":        summary,
		"message":        "已创建新会话并保留上下文总结",
	})
}

// estimateTokens 估算文本的 token 数量
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	chineseCount := 0
	englishCount := 0

	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		} else {
			englishCount++
		}
	}

	return int(float64(chineseCount)/1.5) + int(float64(englishCount)/4)
}

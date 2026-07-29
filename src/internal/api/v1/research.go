package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ai-research-platform/internal/logger"
	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/ai-research-platform/internal/service"
	"github.com/ai-research-platform/internal/types/constant"
	"github.com/ai-research-platform/internal/types/request"
	"github.com/ai-research-platform/internal/types/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ResearchAPI 研究API
type ResearchAPI struct {
	researchDAO     *dao.ResearchDAO
	researchService *service.ResearchService
	membershipDAO   *dao.MembershipDAO
	modelConfigDAO  *dao.ModelConfigDAO
	submissionStore ResearchSubmissionStore
}

type ResearchSubmissionStore interface {
	Submit(context.Context, dao.ResearchSubmissionCommand) (*dao.ResearchSubmissionResult, error)
}

// NewResearchAPI 创建研究API
func NewResearchAPI(researchDAO *dao.ResearchDAO, researchService *service.ResearchService) *ResearchAPI {
	return &ResearchAPI{
		researchDAO:     researchDAO,
		researchService: researchService,
	}
}

// NewResearchAPIWithMembership 创建带会员检查的研究API
func NewResearchAPIWithMembership(researchDAO *dao.ResearchDAO, researchService *service.ResearchService, membershipDAO *dao.MembershipDAO) *ResearchAPI {
	return &ResearchAPI{
		researchDAO:     researchDAO,
		researchService: researchService,
		membershipDAO:   membershipDAO,
	}
}

// NewResearchAPIFull 创建完整的研究API（包含模型配置验证）
func NewResearchAPIFull(
	researchDAO *dao.ResearchDAO,
	researchService *service.ResearchService,
	membershipDAO *dao.MembershipDAO,
	modelConfigDAO *dao.ModelConfigDAO,
	submissionStores ...ResearchSubmissionStore,
) *ResearchAPI {
	api := &ResearchAPI{
		researchDAO:     researchDAO,
		researchService: researchService,
		membershipDAO:   membershipDAO,
		modelConfigDAO:  modelConfigDAO,
	}
	if len(submissionStores) > 0 {
		api.submissionStore = submissionStores[0]
	}
	return api
}

// 研究查询的最大长度限制
const maxResearchQueryLength = constant.MaxResearchQueryLength
const researchStartEndpoint = "/api/v1/research/start"

// StartResearch 开始研究
// 修复：添加输入验证、状态同步
func (api *ResearchAPI) StartResearch(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "认证失败"})
		return
	}
	var req request.StartResearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的请求参数: " + err.Error()})
		return
	}
	if len(req.Query) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "研究查询不能为空"})
		return
	}
	if len(req.Query) > maxResearchQueryLength {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "研究查询过长，最大允许10000字符", "code": "QUERY_TOO_LONG"})
		return
	}
	validResearchTypes := map[string]bool{
		constant.ResearchTypeQuick:         true,
		constant.ResearchTypeDeep:          true,
		constant.ResearchTypeComprehensive: true,
	}
	if req.ResearchType == "" {
		req.ResearchType = constant.ResearchTypeDeep
	} else if !validResearchTypes[req.ResearchType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的研究类型，支持: quick, deep, comprehensive",
			"code":    "INVALID_RESEARCH_TYPE",
		})
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if !validResearchIdempotencyKey(idempotencyKey) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Idempotency-Key 必须为8到255个可见ASCII字符",
			"code":    "INVALID_IDEMPOTENCY_KEY",
		})
		return
	}
	var llmProvider, llmModel string
	var enabledTools []string
	if req.LLMConfig != nil {
		llmProvider, llmModel = req.LLMConfig.Provider, req.LLMConfig.Model
	}
	if req.ToolsConfig != nil {
		enabledTools = req.ToolsConfig.EnabledTools
	}
	if llmProvider != "" || llmModel != "" || len(enabledTools) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "当前不支持按请求指定研究模型或工具配置", "code": "UNSUPPORTED_RESEARCH_CONFIGURATION"})
		return
	}
	if api.submissionStore == nil {
		logger.Error("research submission store is not configured")
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "深度研究提交服务未配置", "code": "RESEARCH_SERVICE_UNAVAILABLE"})
		return
	}
	var metadataJSON []byte
	if req.Options != nil {
		metadataJSON, err = json.Marshal(map[string]interface{}{"options": req.Options})
		if err != nil {
			logger.Error("marshal research options failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "处理研究配置失败"})
			return
		}
	}
	result, err := api.submissionStore.Submit(c.Request.Context(), dao.ResearchSubmissionCommand{
		UserID:         userID,
		Endpoint:       researchStartEndpoint,
		IdempotencyKey: idempotencyKey,
		Query:          req.Query,
		ResearchType:   req.ResearchType,
		Metadata:       metadataJSON,
	})
	if err != nil {
		switch {
		case errors.Is(err, dao.ErrIdempotencyConflict):
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": "Idempotency-Key 已用于不同的研究请求", "code": "IDEMPOTENCY_CONFLICT"})
		case errors.Is(err, dao.ErrResearchQuotaExceeded):
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "深度研究配额已用完", "code": "QUOTA_EXCEEDED"})
		default:
			logger.Error("durable research submission failed", zap.String("user_id", userID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "创建研究任务失败"})
		}
		return
	}
	c.Data(result.HTTPStatus, "application/json; charset=utf-8", result.Body)
}

func validResearchIdempotencyKey(key string) bool {
	if len(key) < 8 || len(key) > 255 {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] < 0x21 || key[i] > 0x7e {
			return false
		}
	}
	return true
}

// GetResearchStatus 获取研究状态
func (api *ResearchAPI) GetResearchStatus(c *gin.Context) {
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

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "ERR_MISSING_PARAMETER",
				"message": "会话ID不能为空",
				"field":   "session_id",
			},
		})
		return
	}

	var session *model.ResearchSession
	if api.researchDAO != nil {
		session, err = api.researchDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_RESEARCH_NOT_FOUND",
					"message": "研究会话不存在或已被删除",
				},
			})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "ERR_FORBIDDEN",
					"message": "无权访问此研究会话",
				},
			})
			return
		}
	}

	var tasks []*model.ResearchTask
	if api.researchDAO != nil {
		tasks, _ = api.researchDAO.GetTasksByResearchID(c.Request.Context(), sessionID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status_data": gin.H{
			"session_id":    session.ID,
			"status":        session.Status,
			"progress":      session.Progress,
			"research_type": session.ResearchType,
			"query":         session.Query,
			"tasks":         tasks,
			"created_at":    session.CreatedAt,
			"updated_at":    session.UpdatedAt,
		},
	})
}


// 研究会话查询的最大限制
const maxResearchSessionLimit = 100

// GetResearchSessions 获取研究会话列表
// 修复：添加hasMore字段，统一分页响应格式
func (api *ResearchAPI) GetResearchSessions(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "认证失败"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit < 1 {
		limit = 20
	}
	if limit > maxResearchSessionLimit {
		limit = maxResearchSessionLimit
	}
	if offset < 0 {
		offset = 0
	}

	var sessions []*model.ResearchSession
	var total int64

	if api.researchDAO != nil {
		sessions, err = api.researchDAO.ListSessionsByUserID(c.Request.Context(), userID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "获取会话列表失败"})
			return
		}
		total, _ = api.researchDAO.CountSessionsByUserID(c.Request.Context(), userID)
	}

	sessionResponses := make([]*response.ResearchSessionResponse, len(sessions))
	for i, s := range sessions {
		sessionResponses[i] = &response.ResearchSessionResponse{
			ID:           s.ID,
			UserID:       s.UserID,
			Query:        s.Query,
			Status:       s.Status,
			Progress:     s.Progress,
			ResearchType: s.ResearchType,
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
		}
	}

	// 修复：添加hasMore和maxLimit字段
	hasMore := int64(offset+len(sessions)) < total

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"sessions":  sessionResponses,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"has_more":  hasMore,
		"max_limit": maxResearchSessionLimit,
	})
}

// StreamResearchProgress 流式获取研究进度
// 修复：添加超时控制、客户端断开检测、资源清理
// 支持从查询参数获取token（用于SSE连接，因为EventSource不支持自定义header）
func (api *ResearchAPI) StreamResearchProgress(c *gin.Context) {
	var userID string
	var err error

	// 优先从查询参数获取token（SSE场景）
	token := c.Query("token")
	if token != "" {
		userID, err = middleware.ValidateTokenString(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效或已过期"})
			return
		}
	} else {
		// 尝试从header获取认证（普通请求场景）
		userID, err = middleware.RequireAuth(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败，请提供token参数"})
			return
		}
	}

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话ID不能为空"})
		return
	}

	if api.researchDAO != nil {
		session, err := api.researchDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此会话"})
			return
		}
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate") // P0: 防止代理/日志缓存含token的URL
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Header("Referrer-Policy", "no-referrer")                       // P0: 防止token通过Referer头泄露
	c.Header("Content-Security-Policy", "default-src 'none'")        // SSE不需要加载任何资源

	c.SSEvent("message", gin.H{"type": "connected", "session_id": sessionID})
	c.Writer.Flush()

	// 创建带超时的上下文（研究任务最长30分钟）
	streamCtx, cancel := context.WithTimeout(c.Request.Context(), constant.ResearchSSETimeout)
	defer cancel()

	if api.researchService != nil {
		streamChan, err := api.researchService.StreamProgress(streamCtx, sessionID)
		if err != nil {
			c.SSEvent("error", gin.H{"error": err.Error()})
			c.Writer.Flush()
			return
		}

		// 监听客户端断开
		clientGone := c.Request.Context().Done()
		
		// 创建心跳定时器，每30秒发送一次心跳保持连接
		heartbeatTicker := time.NewTicker(30 * time.Second)
		defer heartbeatTicker.Stop()

		c.Stream(func(w io.Writer) bool {
			select {
			case <-clientGone:
				// 客户端断开，清理资源
				return false
			case <-streamCtx.Done():
				// 超时
				c.SSEvent("message", gin.H{"type": "timeout", "error": "研究任务超时"})
				return false
			case <-heartbeatTicker.C:
				// 发送心跳保持连接
				c.SSEvent("message", gin.H{
					"type":      "heartbeat",
					"timestamp": time.Now().Unix(),
				})
				c.Writer.Flush()
				return true
			case event, ok := <-streamChan:
				if !ok {
					// 通道已关闭
					return false
				}
				switch event.Type {
				case "error":
					c.SSEvent("message", gin.H{"type": "failed", "error": event.Message})
					return false
				case "completed":
					// 从 event.Data 中获取报告内容和 metadata
					reportText := ""
					var metadata map[string]interface{}
					
					if event.Data != nil {
						if rt, ok := event.Data["report_text"].(string); ok {
							reportText = rt
						}
						// 获取嵌套的 metadata
						if md, ok := event.Data["metadata"].(map[string]interface{}); ok {
							metadata = md
						}
					}
					
					if metadata == nil {
						metadata = make(map[string]interface{})
					}
					metadata["session_id"] = sessionID
					
					c.SSEvent("message", gin.H{
						"type": "completed",
						"data": gin.H{
							"session_id":  sessionID,
							"report_text": reportText,
							"metadata":    metadata,
						},
					})
					c.Writer.Flush() // Ensure completed event is flushed before closing
					return false
				default:
					eventData := gin.H{
						"progress":     event.Progress,
						"current_step": event.Stage,
						"message":      event.Message,
					}
					// 传递并行Agent任务信息
					if event.TaskName != "" {
						eventData["task_name"] = event.TaskName
						eventData["task_status"] = event.TaskStatus
					}
					if event.PartialData != nil {
						eventData["partial_data"] = event.PartialData
					}
					c.SSEvent("message", gin.H{
						"type":   "status_update",
						"status": "in_progress",
						"data":   eventData,
					})
					return true
				}
			}
		})
	} else {
		// researchService 未初始化，返回错误而非模拟数据
		c.SSEvent("message", gin.H{
			"type":  "failed",
			"error": "研究服务未初始化，请联系管理员",
		})
		c.Writer.Flush()
	}
}

// ExportResearch 导出研究结果
func (api *ResearchAPI) ExportResearch(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "认证失败"})
		return
	}

	sessionID := c.Param("session_id")
	format := c.DefaultQuery("format", "json")

	var session *model.ResearchSession
	if api.researchDAO != nil {
		session, err = api.researchDAO.GetSessionByID(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "会话不存在"})
			return
		}
		if session.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "无权访问此会话"})
			return
		}
	}

	var result *model.ResearchResult
	if api.researchDAO != nil {
		result, _ = api.researchDAO.GetResultByResearchID(c.Request.Context(), sessionID)
	}

	switch format {
	case "markdown", "md":
		c.Header("Content-Type", "text/markdown")
		c.Header("Content-Disposition", "attachment; filename=research_"+sessionID+".md")
		if result != nil {
			c.String(http.StatusOK, result.Summary)
		} else {
			c.String(http.StatusOK, "# 研究报告\n\n暂无结果")
		}
	default:
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=research_"+sessionID+".json")
		c.JSON(http.StatusOK, gin.H{
			"session": session,
			"result":  result,
		})
	}
}

// SearchResearch 搜索研究结果
func (api *ResearchAPI) SearchResearch(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "认证失败"})
		return
	}

	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "搜索关键词不能为空"})
		return
	}

	var sessions []*model.ResearchSession
	if api.researchDAO != nil {
		sessions, _ = api.researchDAO.ListSessionsByUserID(c.Request.Context(), userID, 100, 0)
	}

	var results []*model.ResearchSession
	for _, s := range sessions {
		if strings.Contains(strings.ToLower(s.Query), strings.ToLower(query)) {
			results = append(results, s)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"results": results,
		"count":   len(results),
	})
}

// GetResearchStatistics 获取研究统计
func (api *ResearchAPI) GetResearchStatistics(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "认证失败"})
		return
	}

	var total int64
	if api.researchDAO != nil {
		total, _ = api.researchDAO.CountSessionsByUserID(c.Request.Context(), userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"statistics": gin.H{
			"total":        total,
			"completed":    0,
			"failed":       0,
			"success_rate": 0,
		},
	})
}


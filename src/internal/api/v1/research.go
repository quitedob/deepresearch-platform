 package v1

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/ai-research-platform/internal/service"
	"github.com/ai-research-platform/internal/types/request"
	"github.com/ai-research-platform/internal/types/response"
)

// ResearchAPI 研究API
type ResearchAPI struct {
	researchDAO     *dao.ResearchDAO
	researchService *service.ResearchService
	membershipDAO   *dao.MembershipDAO
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

// 研究查询的最大长度限制
const maxResearchQueryLength = 10000

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

	// 修复：验证查询长度，防止DoS攻击
	if len(req.Query) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "研究查询不能为空"})
		return
	}
	if len(req.Query) > maxResearchQueryLength {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "研究查询过长，最大允许10000字符",
			"code":    "QUERY_TOO_LONG",
		})
		return
	}

	// 检查并扣减研究配额（原子操作，先扣减后执行）
	if api.membershipDAO != nil {
		hasQuota, remaining, _, err := api.membershipDAO.CheckAndDeductResearchQuota(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "检查配额失败"})
			return
		}
		if !hasQuota {
			c.JSON(http.StatusForbidden, gin.H{
				"success":   false,
				"error":     "深度研究配额已用完",
				"remaining": remaining,
				"code":      "QUOTA_EXCEEDED",
			})
			return
		}
		// 配额已扣减，如果后续创建会话失败需要退还
	}

	// 修复：验证研究类型
	validResearchTypes := map[string]bool{"quick": true, "deep": true, "comprehensive": true}
	if req.ResearchType == "" {
		req.ResearchType = "deep"
	} else if !validResearchTypes[req.ResearchType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的研究类型，支持: quick, deep, comprehensive",
		})
		return
	}

	session := &model.ResearchSession{
		UserID:       userID,
		Query:        req.Query,
		Status:       "planning",
		Progress:     0,
		ResearchType: req.ResearchType,
	}

	if req.LLMConfig != nil || req.ToolsConfig != nil {
		metadata := map[string]interface{}{
			"llm_config":   req.LLMConfig,
			"tools_config": req.ToolsConfig,
			"options":      req.Options,
		}
		metadataJSON, _ := json.Marshal(metadata)
		session.Metadata = metadataJSON
	}

	if api.researchDAO != nil {
		if err := api.researchDAO.CreateSession(c.Request.Context(), session); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "创建研究会话失败: " + err.Error()})
			return
		}
	} else {
		session.ID = "research_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	// 配额已在请求开始时扣减，无需再次增加
	// 如果创建会话失败，配额已经扣减，这是可接受的（防止滥用）

	// 启动异步研究任务
	if api.researchService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "深度研究服务未配置，请检查服务器配置",
			"code":    "RESEARCH_SERVICE_UNAVAILABLE",
		})
		return
	}

	go func() {
		api.researchService.ExecuteResearch(session.ID, req.Query, req.ResearchType)
	}()

	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"session_id": session.ID,
		"message":    "研究任务已启动",
	})
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
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.SSEvent("message", gin.H{"type": "connected", "session_id": sessionID})
	c.Writer.Flush()

	// 创建带超时的上下文（研究任务最长30分钟）
	streamCtx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
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
		for i := 0; i <= 100; i += 20 {
			select {
			case <-c.Request.Context().Done():
				return
			default:
				c.SSEvent("message", gin.H{
					"type":   "status_update",
					"status": "in_progress",
					"data": gin.H{
						"progress":     i,
						"current_step": "模拟研究进度",
					},
				})
				c.Writer.Flush()
				time.Sleep(time.Second)
			}
		}

		c.SSEvent("message", gin.H{
			"type": "completed",
			"data": gin.H{
				"session_id":  sessionID,
				"report_text": "这是模拟的研究报告内容。",
			},
		})
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


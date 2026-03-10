package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/service"
	"github.com/ai-research-platform/internal/types/request"
	"github.com/ai-research-platform/internal/types/response"
	"github.com/gin-gonic/gin"
)

// PaperAPI 论文生成API
type PaperAPI struct {
	paperDAO     *dao.PaperDAO
	paperService *service.PaperService
}

// NewPaperAPI 创建论文API
func NewPaperAPI(paperDAO *dao.PaperDAO, paperService *service.PaperService) *PaperAPI {
	return &PaperAPI{
		paperDAO:     paperDAO,
		paperService: paperService,
	}
}

// StartPaperGeneration 开始论文生成
// POST /api/v1/paper/start
func (api *PaperAPI) StartPaperGeneration(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	var req request.StartPaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": fmt.Sprintf("请求参数无效: %v", err)})
		return
	}

	// Parse options
	options := map[string]interface{}{
		"citation_style":    "chinese-gb",
		"enable_search":     true,
		"max_review_rounds": 3,
		"language":          "zh-CN",
		"model":             "glm-4.7",
	}
	if req.Options != nil {
		if req.Options.CitationStyle != "" {
			options["citation_style"] = req.Options.CitationStyle
		}
		if req.Options.EnableSearch != nil {
			options["enable_search"] = *req.Options.EnableSearch
		}
		if req.Options.MaxReviewRounds > 0 {
			options["max_review_rounds"] = req.Options.MaxReviewRounds
		}
		if req.Options.Language != "" {
			options["language"] = req.Options.Language
		}
		if req.Options.Model != "" {
			options["model"] = req.Options.Model
		}
	}

	session, err := api.paperService.StartPaperGeneration(c.Request.Context(), userID,
		req.Title, req.Topic, req.InputContent, req.TargetWords, req.PaperType, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": fmt.Sprintf("启动论文生成失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response.NewPaperSessionResponse(session),
	})
}

// GetPaperStatus 获取论文生成状态
// GET /api/v1/paper/status/:id
func (api *PaperAPI) GetPaperStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "论文ID不能为空"})
		return
	}

	statusData, err := api.paperService.GetPaperStatus(c.Request.Context(), sessionID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": fmt.Sprintf("获取论文状态失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    statusData,
	})
}

// StreamProgress 流式获取进度（SSE）
// GET /api/v1/paper/stream/:id
func (api *PaperAPI) StreamProgress(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "论文ID不能为空"})
		return
	}

	// IDOR 校验：确认该会话属于当前用户
	if err := api.paperService.CheckOwnership(c.Request.Context(), sessionID, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "无权访问该论文"})
		return
	}

	// SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Content-Type-Options", "nosniff")

	// 订阅事件流
	eventChan := api.paperService.StreamProgress(sessionID)
	defer api.paperService.StopStreamProgress(sessionID)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "不支持SSE"})
		return
	}

	// 发送初始连接确认
	fmt.Fprintf(c.Writer, "data: {\"type\":\"connected\",\"message\":\"已连接\"}\n\n")
	flusher.Flush()

	// 超时设定
	timeout := time.After(35 * time.Minute)

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				fmt.Fprintf(c.Writer, "data: {\"type\":\"stream_end\",\"message\":\"流结束\"}\n\n")
				flusher.Flush()
				return
			}

			eventJSON, err := json.Marshal(event)
			if err != nil {
				continue
			}

			fmt.Fprintf(c.Writer, "data: %s\n\n", string(eventJSON))
			flusher.Flush()

			// 如果是完成或错误事件，结束流
			if event.Type == "completed" || event.Type == "error" {
				return
			}

		case <-timeout:
			fmt.Fprintf(c.Writer, "data: {\"type\":\"timeout\",\"message\":\"连接超时\"}\n\n")
			flusher.Flush()
			return

		case <-c.Request.Context().Done():
			return
		}
	}
}

// GetPaperResult 获取论文结果
// GET /api/v1/paper/result/:id
func (api *PaperAPI) GetPaperResult(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "论文ID不能为空"})
		return
	}

	result, err := api.paperService.GetPaperResult(c.Request.Context(), sessionID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": fmt.Sprintf("获取论文结果失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// ExportPaper 导出论文
// GET /api/v1/paper/export/:id
func (api *PaperAPI) ExportPaper(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "论文ID不能为空"})
		return
	}

	format := c.DefaultQuery("format", "markdown")

	content, contentType, err := api.paperService.ExportPaper(c.Request.Context(), sessionID, userID, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": fmt.Sprintf("导出失败: %v", err)})
		return
	}

	// 获取论文标题用作文件名
	statusData, _ := api.paperService.GetPaperStatus(c.Request.Context(), sessionID, userID)
	fileName := "paper"
	if statusData != nil {
		fileName = statusData.Title
	}

	ext := ".md"
	if format == "docx" {
		ext = ".docx"
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s%s\"", fileName, ext))
	c.Data(http.StatusOK, contentType, []byte(content))
}

// ListPapers 获取论文列表
// GET /api/v1/paper/list
func (api *PaperAPI) ListPapers(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	var req request.GetPaperListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req.Limit = 20
		req.Offset = 0
	}

	sessions, total, err := api.paperService.ListPapers(c.Request.Context(), userID, req.Limit, req.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": fmt.Sprintf("获取论文列表失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items": sessions,
			"total": total,
			"limit": req.Limit,
			"offset": req.Offset,
		},
	})
}

// DeletePaper 删除论文
// DELETE /api/v1/paper/:id
func (api *PaperAPI) DeletePaper(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "论文ID不能为空"})
		return
	}

	if err := api.paperService.DeletePaper(c.Request.Context(), sessionID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": fmt.Sprintf("删除论文失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

// RegenerateChapter 重新生成章节
// POST /api/v1/paper/regenerate
func (api *PaperAPI) RegenerateChapter(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "未登录"})
		return
	}

	var req request.RegeneratePaperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": fmt.Sprintf("请求参数无效: %v", err)})
		return
	}

	if err := api.paperService.RegenerateChapter(c.Request.Context(), req.PaperID, req.ChapterID, userID, req.Feedback); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": fmt.Sprintf("重新生成失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "章节重新生成已开始"})
}

// GetTemplates 获取论文模板列表
// GET /api/v1/paper/templates
func (api *PaperAPI) GetTemplates(c *gin.Context) {
	templates := api.paperService.GetTemplates()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    templates,
	})
}

// GetCitationStyles 获取支持的引用格式
// GET /api/v1/paper/citation-styles
func (api *PaperAPI) GetCitationStyles(c *gin.Context) {
	styles := api.paperService.GetCitationStyles()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    styles,
	})
}

// Ensure io import is used (for streaming)
var _ = io.EOF

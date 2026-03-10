package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/pkg"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/ai-research-platform/internal/types/constant"
)

// AdminAPI 管理员API
type AdminAPI struct {
	userDAO           *dao.UserDAO
	chatDAO           *dao.ChatDAO
	membershipDAO     *dao.MembershipDAO
	activationCodeDAO *dao.ActivationCodeDAO
	notificationDAO   *dao.NotificationDAO
	modelConfigDAO    *dao.ModelConfigDAO
	quotaConfigDAO    *dao.QuotaConfigDAO
	llmScheduler      interface {
		GetRegisteredModels() map[string]string
		ExecuteWithFallback(ctx context.Context, messages []*eino.Message, model string) (*eino.Message, error)
	}
}

// NewAdminAPI 创建管理员API
func NewAdminAPI(
	userDAO *dao.UserDAO,
	chatDAO *dao.ChatDAO,
	membershipDAO *dao.MembershipDAO,
	activationCodeDAO *dao.ActivationCodeDAO,
	notificationDAO *dao.NotificationDAO,
	modelConfigDAO *dao.ModelConfigDAO,
) *AdminAPI {
	return &AdminAPI{
		userDAO:           userDAO,
		chatDAO:           chatDAO,
		membershipDAO:     membershipDAO,
		activationCodeDAO: activationCodeDAO,
		notificationDAO:   notificationDAO,
		modelConfigDAO:    modelConfigDAO,
	}
}

// NewAdminAPIFull 创建完整的管理员API（包含配额配置）
func NewAdminAPIFull(
	userDAO *dao.UserDAO,
	chatDAO *dao.ChatDAO,
	membershipDAO *dao.MembershipDAO,
	activationCodeDAO *dao.ActivationCodeDAO,
	notificationDAO *dao.NotificationDAO,
	modelConfigDAO *dao.ModelConfigDAO,
	quotaConfigDAO *dao.QuotaConfigDAO,
) *AdminAPI {
	return &AdminAPI{
		userDAO:           userDAO,
		chatDAO:           chatDAO,
		membershipDAO:     membershipDAO,
		activationCodeDAO: activationCodeDAO,
		notificationDAO:   notificationDAO,
		modelConfigDAO:    modelConfigDAO,
		quotaConfigDAO:    quotaConfigDAO,
	}
}

// NewAdminAPIWithScheduler 创建带 LLM 调度器的管理员API
func NewAdminAPIWithScheduler(
	userDAO *dao.UserDAO,
	chatDAO *dao.ChatDAO,
	membershipDAO *dao.MembershipDAO,
	activationCodeDAO *dao.ActivationCodeDAO,
	notificationDAO *dao.NotificationDAO,
	modelConfigDAO *dao.ModelConfigDAO,
	quotaConfigDAO *dao.QuotaConfigDAO,
	llmScheduler *eino.LLMScheduler,
) *AdminAPI {
	return &AdminAPI{
		userDAO:           userDAO,
		chatDAO:           chatDAO,
		membershipDAO:     membershipDAO,
		activationCodeDAO: activationCodeDAO,
		notificationDAO:   notificationDAO,
		modelConfigDAO:    modelConfigDAO,
		quotaConfigDAO:    quotaConfigDAO,
		llmScheduler:      llmScheduler,
	}
}

// RequireAdmin 验证管理员权限
func (api *AdminAPI) RequireAdmin(c *gin.Context) bool {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return false
	}

	user, err := api.userDAO.FindByID(c.Request.Context(), userID)
	if err != nil || !user.IsAdmin {
		pkg.Forbidden(c, "需要管理员权限")
		return false
	}

	return true
}

// ==================== 用户管理 ====================

// ListUsers 获取用户列表
func (api *AdminAPI) ListUsers(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, total, err := api.userDAO.ListAll(c.Request.Context(), limit, offset)
	if err != nil {
		pkg.InternalError(c, "获取用户列表失败")
		return
	}

	// 获取每个用户的会员信息
	var userList []gin.H
	for _, user := range users {
		membership, _ := api.membershipDAO.GetOrCreateMembership(c.Request.Context(), user.ID)
		userList = append(userList, gin.H{
			"id":              user.ID,
			"username":        user.Username,
			"email":           user.Email,
			"full_name":       user.FullName,
			"role":            user.Role,
			"status":          user.Status,
			"is_admin":        user.IsAdmin,
			"created_at":      user.CreatedAt,
			"membership_type": membership.MembershipType,
			"membership":      membership,
		})
	}

	pkg.Success(c, gin.H{
		"users":  userList,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateUserStatus 更新用户状态
func (api *AdminAPI) UpdateUserStatus(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"` // active, banned
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if req.Status != constant.UserStatusActive && req.Status != constant.UserStatusBanned {
		pkg.BadRequest(c, "无效的状态值")
		return
	}

	user, err := api.userDAO.FindByID(c.Request.Context(), userID)
	if err != nil {
		pkg.NotFound(c, "用户不存在")
		return
	}

	user.Status = req.Status
	if err := api.userDAO.Update(c.Request.Context(), user); err != nil {
		pkg.InternalError(c, "更新用户状态失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "用户状态已更新"})
}

// UpdateUserMembership 更新用户会员状态
func (api *AdminAPI) UpdateUserMembership(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID := c.Param("id")
	var req struct {
		MembershipType string `json:"membership_type" binding:"required"` // free, premium
		ValidDays      int    `json:"valid_days"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if req.MembershipType == constant.MembershipPremium {
		if req.ValidDays <= 0 {
			req.ValidDays = 30
		}
		if err := api.membershipDAO.UpgradeToPremium(c.Request.Context(), userID, req.ValidDays, "admin", nil); err != nil {
			pkg.InternalError(c, "升级会员失败")
			return
		}
	} else {
		if err := api.membershipDAO.DowngradeToFree(c.Request.Context(), userID); err != nil {
			pkg.InternalError(c, "降级会员失败")
			return
		}
	}

	pkg.Success(c, gin.H{"success": true, "message": "会员状态已更新"})
}

// ResetUserQuota 重置用户配额
func (api *AdminAPI) ResetUserQuota(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID := c.Param("id")
	if err := api.membershipDAO.ResetUserQuota(c.Request.Context(), userID); err != nil {
		pkg.InternalError(c, "重置配额失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "用户配额已重置"})
}

// SetUserQuota 设置用户配额
func (api *AdminAPI) SetUserQuota(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID := c.Param("id")
	var req struct {
		ChatLimit     int `json:"chat_limit"`
		ResearchLimit int `json:"research_limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if err := api.membershipDAO.SetUserQuota(c.Request.Context(), userID, req.ChatLimit, req.ResearchLimit); err != nil {
		pkg.InternalError(c, "设置配额失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "用户配额已设置"})
}

// ==================== 聊天记录管理 ====================

// GetUserChatHistory 获取用户聊天记录
func (api *AdminAPI) GetUserChatHistory(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	sessions, err := api.chatDAO.ListSessionsByUserID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		pkg.InternalError(c, "获取聊天记录失败")
		return
	}

	total, _ := api.chatDAO.CountSessionsByUserID(c.Request.Context(), userID)

	var sessionList []gin.H
	for _, session := range sessions {
		messages, _ := api.chatDAO.GetMessagesBySessionID(c.Request.Context(), session.ID, 100, 0)
		sessionList = append(sessionList, gin.H{
			"id":            session.ID,
			"title":         session.Title,
			"provider":      session.Provider,
			"model":         session.Model,
			"message_count": session.MessageCount,
			"created_at":    session.CreatedAt,
			"updated_at":    session.UpdatedAt,
			"messages":      messages,
		})
	}

	pkg.Success(c, gin.H{
		"sessions": sessionList,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// ExportUserChatHistory 导出用户聊天记录
// 修复：添加错误日志记录，限制导出数据量
func (api *AdminAPI) ExportUserChatHistory(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID := c.Param("id")
	adminID, _ := middleware.RequireAuth(c)

	// 获取用户信息
	user, err := api.userDAO.FindByID(c.Request.Context(), userID)
	if err != nil {
		pkg.NotFound(c, "用户不存在")
		return
	}

	// 限制导出的会话数量，防止内存溢出
	maxExportSessions := constant.MaxExportSessions
	maxMessagesPerSession := constant.MaxMessagesPerSession

	// 获取聊天会话（限制数量）
	sessions, err := api.chatDAO.ListSessionsByUserID(c.Request.Context(), userID, maxExportSessions, 0)
	if err != nil {
		// 记录错误日志
		c.Error(err)
		pkg.InternalError(c, "获取聊天记录失败")
		return
	}

	var exportData struct {
		User struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
		} `json:"user"`
		ExportedAt   string        `json:"exported_at"`
		ExportedBy   string        `json:"exported_by"`
		Sessions     []interface{} `json:"sessions"`
		TotalCount   int           `json:"total_count"`
		ExportedCount int          `json:"exported_count"`
	}

	exportData.User.ID = user.ID
	exportData.User.Username = user.Username
	exportData.User.Email = user.Email
	exportData.ExportedAt = time.Now().Format(time.RFC3339)
	exportData.ExportedBy = adminID

	// 获取总会话数
	totalCount, _ := api.chatDAO.CountSessionsByUserID(c.Request.Context(), userID)
	exportData.TotalCount = int(totalCount)
	exportData.ExportedCount = len(sessions)

	for _, session := range sessions {
		messages, err := api.chatDAO.GetMessagesBySessionID(c.Request.Context(), session.ID, maxMessagesPerSession, 0)
		if err != nil {
			// 记录错误但继续处理其他会话
			c.Error(err)
			continue
		}
		exportData.Sessions = append(exportData.Sessions, gin.H{
			"id":            session.ID,
			"title":         session.Title,
			"provider":      session.Provider,
			"model":         session.Model,
			"system_prompt": session.SystemPrompt,
			"message_count": session.MessageCount,
			"created_at":    session.CreatedAt,
			"updated_at":    session.UpdatedAt,
			"messages":      messages,
		})
	}

	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		// 记录JSON编码错误
		c.Error(err)
		pkg.InternalError(c, "导出失败: JSON编码错误")
		return
	}

	c.Header("Content-Disposition", "attachment; filename=chat_history_"+user.Username+".json")
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", jsonData)
}

// ==================== 激活码管理 ====================

// ListActivationCodes 获取激活码列表
func (api *AdminAPI) ListActivationCodes(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	codes, total, err := api.activationCodeDAO.List(c.Request.Context(), limit, offset)
	if err != nil {
		pkg.InternalError(c, "获取激活码列表失败")
		return
	}

	pkg.Success(c, gin.H{
		"codes":  codes,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateActivationCode 创建激活码
func (api *AdminAPI) CreateActivationCode(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID, _ := middleware.RequireAuth(c)

	var req struct {
		MaxActivations int    `json:"max_activations" binding:"required,min=1"`
		ValidDays      int    `json:"valid_days"`
		ExpiresInDays  int    `json:"expires_in_days"`
		Code           string `json:"code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if req.ValidDays <= 0 {
		req.ValidDays = 30
	}

	code := &model.ActivationCode{
		Code:           req.Code,
		MaxActivations: req.MaxActivations,
		ValidDays:      req.ValidDays,
		CreatedBy:      userID,
		IsActive:       true,
	}

	if req.ExpiresInDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, req.ExpiresInDays)
		code.ExpiresAt = &expiresAt
	}

	if err := api.activationCodeDAO.Create(c.Request.Context(), code); err != nil {
		pkg.InternalError(c, "创建激活码失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "code": code})
}

// GetActivationCodeDetails 获取激活码详情（包含激活用户）
func (api *AdminAPI) GetActivationCodeDetails(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	codeID := c.Param("id")

	code, err := api.activationCodeDAO.GetByID(c.Request.Context(), codeID)
	if err != nil {
		pkg.NotFound(c, "激活码不存在")
		return
	}

	records, _ := api.activationCodeDAO.GetActivationRecordsWithUsers(c.Request.Context(), codeID)

	pkg.Success(c, gin.H{
		"code":    code,
		"records": records,
	})
}

// UpdateActivationCode 更新激活码
func (api *AdminAPI) UpdateActivationCode(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	codeID := c.Param("id")
	var req struct {
		IsActive       *bool `json:"is_active"`
		MaxActivations *int  `json:"max_activations"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	code, err := api.activationCodeDAO.GetByID(c.Request.Context(), codeID)
	if err != nil {
		pkg.NotFound(c, "激活码不存在")
		return
	}

	if req.IsActive != nil {
		code.IsActive = *req.IsActive
	}
	if req.MaxActivations != nil {
		code.MaxActivations = *req.MaxActivations
	}

	if err := api.activationCodeDAO.Update(c.Request.Context(), code); err != nil {
		pkg.InternalError(c, "更新激活码失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "激活码已更新"})
}

// DeleteActivationCode 删除激活码
func (api *AdminAPI) DeleteActivationCode(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	codeID := c.Param("id")
	if err := api.activationCodeDAO.Delete(c.Request.Context(), codeID); err != nil {
		pkg.InternalError(c, "删除激活码失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "激活码已删除"})
}

// ==================== 通知管理 ====================

// ListNotifications 获取通知列表
func (api *AdminAPI) ListNotifications(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, total, err := api.notificationDAO.List(c.Request.Context(), limit, offset)
	if err != nil {
		pkg.InternalError(c, "获取通知列表失败")
		return
	}

	pkg.Success(c, gin.H{
		"notifications": notifications,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
	})
}

// CreateNotification 创建通知
func (api *AdminAPI) CreateNotification(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID, _ := middleware.RequireAuth(c)

	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Type     string `json:"type"`
		IsGlobal bool   `json:"is_global"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	notificationType := model.NotificationSystem
	if req.Type == "announce" {
		notificationType = model.NotificationAnnounce
	} else if req.Type == "alert" {
		notificationType = model.NotificationAlert
	}

	notification := &model.Notification{
		Title:     req.Title,
		Content:   req.Content,
		Type:      notificationType,
		CreatedBy: userID,
		IsGlobal:  req.IsGlobal,
	}

	if err := api.notificationDAO.Create(c.Request.Context(), notification); err != nil {
		pkg.InternalError(c, "创建通知失败")
		return
	}

	// 如果是全局通知，发送给所有用户
	if req.IsGlobal {
		api.notificationDAO.SendToAllUsers(c.Request.Context(), notification.ID)
	}

	pkg.Success(c, gin.H{"success": true, "notification": notification})
}

// DeleteNotification 删除通知
func (api *AdminAPI) DeleteNotification(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	notificationID := c.Param("id")
	if err := api.notificationDAO.Delete(c.Request.Context(), notificationID); err != nil {
		pkg.InternalError(c, "删除通知失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "通知已删除"})
}

// ==================== 模型配置管理 ====================

// GetProviderConfigs 获取提供商配置
func (api *AdminAPI) GetProviderConfigs(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	configs, err := api.modelConfigDAO.GetProviderConfigs(c.Request.Context())
	if err != nil {
		pkg.InternalError(c, "获取提供商配置失败")
		return
	}

	pkg.Success(c, gin.H{"providers": configs})
}

// UpdateProviderConfig 更新提供商配置
func (api *AdminAPI) UpdateProviderConfig(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	var req struct {
		Provider  string `json:"provider" binding:"required"`
		IsEnabled bool   `json:"is_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if err := api.modelConfigDAO.UpdateProviderConfig(c.Request.Context(), req.Provider, req.IsEnabled); err != nil {
		pkg.InternalError(c, "更新提供商配置失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "提供商配置已更新"})
}

// GetModelConfigs 获取模型配置
func (api *AdminAPI) GetModelConfigs(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	configs, err := api.modelConfigDAO.GetModelConfigs(c.Request.Context())
	if err != nil {
		pkg.InternalError(c, "获取模型配置失败")
		return
	}

	pkg.Success(c, gin.H{"models": configs})
}

// UpdateModelConfig 更新模型配置
func (api *AdminAPI) UpdateModelConfig(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	var req struct {
		Provider  string `json:"provider" binding:"required"`
		ModelName string `json:"model_name" binding:"required"`
		IsEnabled bool   `json:"is_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if err := api.modelConfigDAO.UpdateModelConfig(c.Request.Context(), req.Provider, req.ModelName, req.IsEnabled); err != nil {
		pkg.InternalError(c, "更新模型配置失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "模型配置已更新"})
}

// BatchUpdateModelConfigs 批量更新模型配置
func (api *AdminAPI) BatchUpdateModelConfigs(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	var req struct {
		Configs []struct {
			Provider  string `json:"provider"`
			ModelName string `json:"model_name"`
			IsEnabled bool   `json:"is_enabled"`
		} `json:"configs"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	configs := make([]struct {
		Provider  string
		ModelName string
		IsEnabled bool
	}, len(req.Configs))

	for i, c := range req.Configs {
		configs[i] = struct {
			Provider  string
			ModelName string
			IsEnabled bool
		}{c.Provider, c.ModelName, c.IsEnabled}
	}

	if err := api.modelConfigDAO.BatchUpdateModelConfigs(c.Request.Context(), configs); err != nil {
		pkg.InternalError(c, "批量更新模型配置失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "模型配置已批量更新"})
}

// ==================== 统计信息 ====================

// GetAdminStats 获取管理员统计信息
// 修复：使用数据库聚合查询代替遍历，提高性能
func (api *AdminAPI) GetAdminStats(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	ctx := c.Request.Context()

	// 用户统计
	_, totalUsers, _ := api.userDAO.ListAll(ctx, 1, 0)

	// 会员统计 - 使用COUNT查询代替遍历
	var premiumCount int64
	if api.membershipDAO != nil {
		premiumCount, _ = api.membershipDAO.CountByMembershipType(ctx, model.MembershipPremium)
	}

	// 激活码统计 - 使用COUNT查询代替遍历
	var totalCodes, activeCodes int64
	if api.activationCodeDAO != nil {
		_, totalCodes, _ = api.activationCodeDAO.List(ctx, 1, 0)
		activeCodes, _ = api.activationCodeDAO.CountActive(ctx)
	}

	pkg.Success(c, gin.H{
		"total_users":   totalUsers,
		"premium_users": premiumCount,
		"free_users":    totalUsers - premiumCount,
		"total_codes":   totalCodes,
		"active_codes":  activeCodes,
	})
}

// ==================== 配额配置管理 ====================

// GetQuotaConfigs 获取所有配额配置
func (api *AdminAPI) GetQuotaConfigs(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	if api.quotaConfigDAO == nil {
		pkg.InternalError(c, "配额配置服务未初始化")
		return
	}

	configs, err := api.quotaConfigDAO.GetAll(c.Request.Context())
	if err != nil {
		pkg.InternalError(c, "获取配额配置失败")
		return
	}

	pkg.Success(c, gin.H{"configs": configs})
}

// UpdateQuotaConfig 更新配额配置（按会员层级）
func (api *AdminAPI) UpdateQuotaConfig(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID, _ := middleware.RequireAuth(c)

	var req struct {
		MembershipType   string `json:"membership_type" binding:"required"` // free, premium
		ChatLimit        int    `json:"chat_limit" binding:"min=0"`
		ResearchLimit    int    `json:"research_limit" binding:"min=0"`
		ResetPeriodHours int    `json:"reset_period_hours" binding:"min=0"`
		ApplyToAll       bool   `json:"apply_to_all"` // 是否应用到所有该类型用户
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if api.quotaConfigDAO == nil {
		pkg.InternalError(c, "配额配置服务未初始化")
		return
	}

	// 更新全局配置
	if err := api.quotaConfigDAO.UpdateByMembershipType(
		c.Request.Context(),
		req.MembershipType,
		req.ChatLimit,
		req.ResearchLimit,
		req.ResetPeriodHours,
		userID,
	); err != nil {
		pkg.InternalError(c, "更新配额配置失败")
		return
	}

	// 如果需要应用到所有用户
	if req.ApplyToAll {
		if err := api.quotaConfigDAO.ApplyToAllUsers(
			c.Request.Context(),
			req.MembershipType,
			req.ChatLimit,
			req.ResearchLimit,
		); err != nil {
			pkg.InternalError(c, "应用配额到用户失败")
			return
		}
	}

	pkg.Success(c, gin.H{
		"success":      true,
		"message":      "配额配置已更新",
		"applied_to_all": req.ApplyToAll,
	})
}

// SetUserCustomQuota 设置用户自定义配额（覆盖全局配置）
func (api *AdminAPI) SetUserCustomQuota(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	userID := c.Param("id")
	var req struct {
		ChatLimit     int  `json:"chat_limit" binding:"min=0"`
		ResearchLimit int  `json:"research_limit" binding:"min=0"`
		ResetUsage    bool `json:"reset_usage"` // 是否同时重置使用量
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	// 设置用户配额
	if err := api.membershipDAO.SetUserQuota(c.Request.Context(), userID, req.ChatLimit, req.ResearchLimit); err != nil {
		pkg.InternalError(c, "设置用户配额失败")
		return
	}

	// 如果需要重置使用量
	if req.ResetUsage {
		if err := api.membershipDAO.ResetUserQuota(c.Request.Context(), userID); err != nil {
			pkg.InternalError(c, "重置用户使用量失败")
			return
		}
	}

	pkg.Success(c, gin.H{
		"success": true,
		"message": "用户配额已设置",
	})
}

// BatchSetUserQuota 批量设置用户配额
func (api *AdminAPI) BatchSetUserQuota(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	var req struct {
		UserIDs       []string `json:"user_ids" binding:"required"`
		ChatLimit     int      `json:"chat_limit" binding:"min=0"`
		ResearchLimit int      `json:"research_limit" binding:"min=0"`
		ResetUsage    bool     `json:"reset_usage"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	successCount := 0
	for _, userID := range req.UserIDs {
		if err := api.membershipDAO.SetUserQuota(c.Request.Context(), userID, req.ChatLimit, req.ResearchLimit); err == nil {
			successCount++
			if req.ResetUsage {
				api.membershipDAO.ResetUserQuota(c.Request.Context(), userID)
			}
		}
	}

	pkg.Success(c, gin.H{
		"success":       true,
		"message":       "批量设置完成",
		"success_count": successCount,
		"total_count":   len(req.UserIDs),
	})
}

// BatchUpdateUserStatus 批量更新用户状态
func (api *AdminAPI) BatchUpdateUserStatus(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	var req struct {
		UserIDs []string `json:"user_ids" binding:"required"`
		Status  string   `json:"status" binding:"required"` // active, banned
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if req.Status != constant.UserStatusActive && req.Status != constant.UserStatusBanned {
		pkg.BadRequest(c, "无效的状态值，只能是 active 或 banned")
		return
	}

	if len(req.UserIDs) == 0 {
		pkg.BadRequest(c, "用户ID列表不能为空")
		return
	}

	// 使用批量更新
	if err := api.userDAO.BatchUpdateStatus(c.Request.Context(), req.UserIDs, req.Status); err != nil {
		pkg.InternalError(c, "批量更新用户状态失败")
		return
	}

	pkg.Success(c, gin.H{
		"success":     true,
		"message":     "批量更新用户状态完成",
		"total_count": len(req.UserIDs),
	})
}

// BatchResetUserQuotas 批量重置用户配额
func (api *AdminAPI) BatchResetUserQuotas(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	var req struct {
		UserIDs []string `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if len(req.UserIDs) == 0 {
		pkg.BadRequest(c, "用户ID列表不能为空")
		return
	}

	// 使用批量重置
	if err := api.membershipDAO.BatchResetUserQuotas(c.Request.Context(), req.UserIDs); err != nil {
		pkg.InternalError(c, "批量重置用户配额失败")
		return
	}

	pkg.Success(c, gin.H{
		"success":     true,
		"message":     "批量重置用户配额完成",
		"total_count": len(req.UserIDs),
	})
}

// BatchDeleteUsers 批量删除用户
func (api *AdminAPI) BatchDeleteUsers(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	var req struct {
		UserIDs []string `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if len(req.UserIDs) == 0 {
		pkg.BadRequest(c, "用户ID列表不能为空")
		return
	}

	// 使用批量删除
	if err := api.userDAO.BatchDelete(c.Request.Context(), req.UserIDs); err != nil {
		pkg.InternalError(c, "批量删除用户失败")
		return
	}

	pkg.Success(c, gin.H{
		"success":     true,
		"message":     "批量删除用户完成",
		"total_count": len(req.UserIDs),
	})
}


// ==================== 模型测试 ====================

// TestModel 测试模型连接
func (api *AdminAPI) TestModel(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	var req struct {
		Provider string `json:"provider" binding:"required"`
		Model    string `json:"model" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if api.llmScheduler == nil {
		pkg.InternalError(c, "LLM调度器未初始化")
		return
	}

	// 检查模型是否已注册
	registeredModels := api.llmScheduler.GetRegisteredModels()
	if _, ok := registeredModels[req.Model]; !ok {
		pkg.BadRequest(c, "模型未注册: "+req.Model)
		return
	}

	// 发送测试消息
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	testMessages := []*eino.Message{
		{Role: "user", Content: "Hello, please respond with 'OK' to confirm you are working."},
	}

	startTime := time.Now()
	response, err := api.llmScheduler.ExecuteWithFallback(ctx, testMessages, req.Model)
	duration := time.Since(startTime)

	if err != nil {
		pkg.Success(c, gin.H{
			"success":  false,
			"provider": req.Provider,
			"model":    req.Model,
			"error":    err.Error(),
			"duration": duration.Milliseconds(),
		})
		return
	}

	pkg.Success(c, gin.H{
		"success":  true,
		"provider": req.Provider,
		"model":    req.Model,
		"response": response.Content,
		"duration": duration.Milliseconds(),
	})
}

// GetAllRegisteredModels 获取所有已注册的模型（管理端专用，显示所有模型）
func (api *AdminAPI) GetAllRegisteredModels(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	if api.llmScheduler == nil {
		pkg.InternalError(c, "LLM调度器未初始化")
		return
	}

	registeredModels := api.llmScheduler.GetRegisteredModels()

	// 获取数据库中的启用状态
	var enabledModels map[string]bool
	if api.modelConfigDAO != nil {
		enabledModels = make(map[string]bool)
		modelConfigs, _ := api.modelConfigDAO.GetModelConfigs(c.Request.Context())
		for _, m := range modelConfigs {
			enabledModels[m.ModelName] = m.IsEnabled
		}
	}

	// 构建模型列表
	models := make([]gin.H, 0)
	for modelName, providerKey := range registeredModels {
		// 提取 provider 名称
		provider := providerKey
		if idx := len(providerKey) - len(modelName) - 1; idx > 0 {
			provider = providerKey[:idx]
		}

		isEnabled := true
		if enabledModels != nil {
			if enabled, ok := enabledModels[modelName]; ok {
				isEnabled = enabled
			}
		}

		models = append(models, gin.H{
			"model":      modelName,
			"provider":   provider,
			"is_enabled": isEnabled,
			"registered": true,
		})
	}

	pkg.Success(c, gin.H{
		"models": models,
		"total":  len(models),
	})
}

// SyncModelsToDatabase 同步已注册的模型到数据库
func (api *AdminAPI) SyncModelsToDatabase(c *gin.Context) {
	if !api.RequireAdmin(c) {
		return
	}

	if api.llmScheduler == nil {
		pkg.InternalError(c, "LLM调度器未初始化")
		return
	}

	if api.modelConfigDAO == nil {
		pkg.InternalError(c, "模型配置DAO未初始化")
		return
	}

	registeredModels := api.llmScheduler.GetRegisteredModels()

	// 获取现有的模型配置
	existingConfigs, _ := api.modelConfigDAO.GetModelConfigs(c.Request.Context())
	existingMap := make(map[string]bool)
	for _, m := range existingConfigs {
		existingMap[m.ModelName] = true
	}

	// 同步新模型到数据库
	addedCount := 0
	for modelName, providerKey := range registeredModels {
		if existingMap[modelName] {
			continue
		}

		// 提取 provider 名称
		provider := providerKey
		if idx := len(providerKey) - len(modelName) - 1; idx > 0 {
			provider = providerKey[:idx]
		}

		// 创建新的模型配置
		newConfig := &model.ModelConfig{
			Provider:    provider,
			ModelName:   modelName,
			DisplayName: modelName,
			IsEnabled:   true,
			SortOrder:   100, // 新模型排在后面
		}

		if err := api.modelConfigDAO.CreateModelConfig(c.Request.Context(), newConfig); err == nil {
			addedCount++
		}
	}

	pkg.Success(c, gin.H{
		"success":     true,
		"message":     "模型同步完成",
		"added_count": addedCount,
		"total_registered": len(registeredModels),
	})
}

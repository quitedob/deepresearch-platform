package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/pkg"
	"github.com/ai-research-platform/internal/repository/dao"
)

// NotificationAPI 通知API（用户端）
type NotificationAPI struct {
	notificationDAO *dao.NotificationDAO
}

// NewNotificationAPI 创建通知API
func NewNotificationAPI(notificationDAO *dao.NotificationDAO) *NotificationAPI {
	return &NotificationAPI{
		notificationDAO: notificationDAO,
	}
}

// GetNotifications 获取用户通知
// 修复：添加hasMore字段，统一分页响应格式
func (api *NotificationAPI) GetNotifications(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 验证分页参数
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	notifications, total, err := api.notificationDAO.GetUserNotifications(c.Request.Context(), userID, limit, offset)
	if err != nil {
		pkg.InternalError(c, "获取通知失败")
		return
	}

	// 计算是否有更多数据
	hasMore := int64(offset+len(notifications)) < total

	pkg.Success(c, gin.H{
		"notifications": notifications,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
		"has_more":      hasMore,
	})
}

// GetUnreadCount 获取未读通知数量
func (api *NotificationAPI) GetUnreadCount(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return
	}

	count, err := api.notificationDAO.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		pkg.InternalError(c, "获取未读数量失败")
		return
	}

	pkg.Success(c, gin.H{"unread_count": count})
}

// MarkAsRead 标记通知为已读
func (api *NotificationAPI) MarkAsRead(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return
	}

	notificationID := c.Param("id")
	if err := api.notificationDAO.MarkAsRead(c.Request.Context(), userID, notificationID); err != nil {
		pkg.InternalError(c, "标记已读失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "已标记为已读"})
}

// MarkAllAsRead 标记所有通知为已读
func (api *NotificationAPI) MarkAllAsRead(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return
	}

	if err := api.notificationDAO.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		pkg.InternalError(c, "标记已读失败")
		return
	}

	pkg.Success(c, gin.H{"success": true, "message": "已全部标记为已读"})
}

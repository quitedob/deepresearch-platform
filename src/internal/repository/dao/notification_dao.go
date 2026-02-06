package dao

import (
	"context"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// NotificationDAO 通知数据访问对象
type NotificationDAO struct {
	db *gorm.DB
}

// NewNotificationDAO 创建通知DAO
func NewNotificationDAO(db *gorm.DB) *NotificationDAO {
	return &NotificationDAO{db: db}
}

// Create 创建通知
func (d *NotificationDAO) Create(ctx context.Context, notification *model.Notification) error {
	return d.db.WithContext(ctx).Create(notification).Error
}

// GetByID 根据ID获取通知
func (d *NotificationDAO) GetByID(ctx context.Context, id string) (*model.Notification, error) {
	var notification model.Notification
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&notification).Error
	return &notification, err
}

// Update 更新通知
func (d *NotificationDAO) Update(ctx context.Context, notification *model.Notification) error {
	return d.db.WithContext(ctx).Save(notification).Error
}

// Delete 删除通知
func (d *NotificationDAO) Delete(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&model.Notification{}, "id = ?", id).Error
}

// List 获取通知列表
func (d *NotificationDAO) List(ctx context.Context, limit, offset int) ([]*model.Notification, int64, error) {
	var notifications []*model.Notification
	var total int64

	d.db.WithContext(ctx).Model(&model.Notification{}).Count(&total)
	err := d.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifications).Error

	return notifications, total, err
}

// SendToAllUsers 发送通知给所有用户
func (d *NotificationDAO) SendToAllUsers(ctx context.Context, notificationID string) error {
	// 获取所有活跃用户
	var users []model.User
	if err := d.db.WithContext(ctx).Where("status = ?", "active").Find(&users).Error; err != nil {
		return err
	}

	// 为每个用户创建通知记录
	for _, user := range users {
		userNotification := &model.UserNotification{
			UserID:         user.ID,
			NotificationID: notificationID,
			IsRead:         false,
		}
		d.db.WithContext(ctx).Create(userNotification)
	}

	return nil
}

// GetUserNotifications 获取用户的通知
func (d *NotificationDAO) GetUserNotifications(ctx context.Context, userID string, limit, offset int) ([]*model.Notification, int64, error) {
	var notifications []*model.Notification
	var total int64

	// 获取全局通知和用户特定通知
	subQuery := d.db.WithContext(ctx).Table("user_notifications").
		Select("notification_id").
		Where("user_id = ?", userID)

	d.db.WithContext(ctx).Model(&model.Notification{}).
		Where("is_global = ? OR id IN (?)", true, subQuery).
		Count(&total)

	err := d.db.WithContext(ctx).
		Where("is_global = ? OR id IN (?)", true, subQuery).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error

	return notifications, total, err
}

// GetUnreadCount 获取用户未读通知数量
func (d *NotificationDAO) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	
	// 获取用户已读的通知ID
	var readNotificationIDs []string
	d.db.WithContext(ctx).Table("user_notifications").
		Select("notification_id").
		Where("user_id = ? AND is_read = ?", userID, true).
		Pluck("notification_id", &readNotificationIDs)

	// 统计未读通知
	query := d.db.WithContext(ctx).Model(&model.Notification{}).Where("is_global = ?", true)
	if len(readNotificationIDs) > 0 {
		query = query.Where("id NOT IN ?", readNotificationIDs)
	}
	err := query.Count(&count).Error

	return count, err
}

// MarkAsRead 标记通知为已读
func (d *NotificationDAO) MarkAsRead(ctx context.Context, userID, notificationID string) error {
	var userNotification model.UserNotification
	err := d.db.WithContext(ctx).Where("user_id = ? AND notification_id = ?", userID, notificationID).First(&userNotification).Error
	
	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		now := time.Now()
		userNotification = model.UserNotification{
			UserID:         userID,
			NotificationID: notificationID,
			IsRead:         true,
			ReadAt:         &now,
		}
		return d.db.WithContext(ctx).Create(&userNotification).Error
	}

	if err != nil {
		return err
	}

	// 更新已有记录
	now := time.Now()
	userNotification.IsRead = true
	userNotification.ReadAt = &now
	return d.db.WithContext(ctx).Save(&userNotification).Error
}

// MarkAllAsRead 标记所有通知为已读
func (d *NotificationDAO) MarkAllAsRead(ctx context.Context, userID string) error {
	// 获取所有全局通知
	var notifications []*model.Notification
	if err := d.db.WithContext(ctx).Where("is_global = ?", true).Find(&notifications).Error; err != nil {
		return err
	}

	now := time.Now()
	for _, notification := range notifications {
		var userNotification model.UserNotification
		err := d.db.WithContext(ctx).Where("user_id = ? AND notification_id = ?", userID, notification.ID).First(&userNotification).Error
		
		if err == gorm.ErrRecordNotFound {
			userNotification = model.UserNotification{
				UserID:         userID,
				NotificationID: notification.ID,
				IsRead:         true,
				ReadAt:         &now,
			}
			d.db.WithContext(ctx).Create(&userNotification)
		} else if err == nil {
			userNotification.IsRead = true
			userNotification.ReadAt = &now
			d.db.WithContext(ctx).Save(&userNotification)
		}
	}

	return nil
}

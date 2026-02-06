package dao

import (
	"context"

	"github.com/ai-research-platform/internal/pkg/utils"
	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// UserDAO 用户数据访问对象
type UserDAO struct {
	db *gorm.DB
}

// NewUserDAO 创建用户DAO
func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		db: db,
	}
}

// Create 创建用户
func (u *UserDAO) Create(ctx context.Context, user *model.User) error {
	return u.db.WithContext(ctx).Create(user).Error
}

// FindByEmail 根据邮箱查找用户
// 修复：添加邮箱格式验证防止SQL注入
func (u *UserDAO) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	// 验证邮箱格式
	if !utils.IsValidEmail(email) {
		return nil, ErrInvalidID
	}
	// 检查SQL注入
	if utils.ContainsSQLInjection(email) {
		return nil, ErrInvalidID
	}

	var user model.User
	err := u.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 根据ID查找用户
// 修复：添加ID格式验证防止SQL注入
func (u *UserDAO) FindByID(ctx context.Context, id string) (*model.User, error) {
	// 验证ID格式
	if sanitizedID, valid := utils.ValidateAndSanitizeID(id); !valid {
		return nil, ErrInvalidID
	} else {
		id = sanitizedID
	}

	var user model.User
	err := u.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (u *UserDAO) Update(ctx context.Context, user *model.User) error {
	return u.db.WithContext(ctx).Save(user).Error
}

// Delete 删除用户
// 修复：添加ID格式验证
func (u *UserDAO) Delete(ctx context.Context, id string) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(id); !valid {
		return ErrInvalidID
	} else {
		id = sanitizedID
	}
	return u.db.WithContext(ctx).Delete(&model.User{}, "id = ?", id).Error
}

// ExistsByEmail 检查邮箱是否存在
// 修复：添加邮箱格式验证
func (u *UserDAO) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if !utils.IsValidEmail(email) {
		return false, ErrInvalidID
	}
	if utils.ContainsSQLInjection(email) {
		return false, ErrInvalidID
	}

	var count int64
	err := u.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// FindByUsername 根据用户名查找用户
// 修复：添加用户名格式验证
func (u *UserDAO) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	// 验证用户名格式
	if !utils.IsValidUsername(username) {
		return nil, ErrInvalidID
	}
	if utils.ContainsSQLInjection(username) {
		return nil, ErrInvalidID
	}

	var user model.User
	err := u.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ExistsByUsername 检查用户名是否存在
// 修复：添加用户名格式验证
func (u *UserDAO) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if !utils.IsValidUsername(username) {
		return false, ErrInvalidID
	}
	if utils.ContainsSQLInjection(username) {
		return false, ErrInvalidID
	}

	var count int64
	err := u.db.WithContext(ctx).Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// ListAll 获取所有用户列表
// 修复：添加分页参数验证
func (u *UserDAO) ListAll(ctx context.Context, limit, offset int) ([]*model.User, int64, error) {
	// 验证分页参数
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var users []*model.User
	var total int64

	u.db.WithContext(ctx).Model(&model.User{}).Count(&total)
	err := u.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error

	return users, total, err
}

// ListByStatus 根据状态获取用户列表
// 修复：添加状态值验证
func (u *UserDAO) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*model.User, int64, error) {
	// 验证状态值（只允许预定义的状态）
	validStatuses := map[string]bool{"active": true, "banned": true, "pending": true}
	if !validStatuses[status] {
		return nil, 0, ErrInvalidID
	}

	// 验证分页参数
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var users []*model.User
	var total int64

	u.db.WithContext(ctx).Model(&model.User{}).Where("status = ?", status).Count(&total)
	err := u.db.WithContext(ctx).Where("status = ?", status).Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error

	return users, total, err
}

// CountByStatus 统计各状态用户数量
func (u *UserDAO) CountByStatus(ctx context.Context) (map[string]int64, error) {
	result := make(map[string]int64)

	var activeCount, bannedCount int64
	u.db.WithContext(ctx).Model(&model.User{}).Where("status = ?", "active").Count(&activeCount)
	u.db.WithContext(ctx).Model(&model.User{}).Where("status = ?", "banned").Count(&bannedCount)

	result["active"] = activeCount
	result["banned"] = bannedCount

	return result, nil
}

// UserWithAutoClean 用户自动清理设置
type UserWithAutoClean struct {
	ID            string
	AutoCleanDays int
}

// GetUsersWithAutoClean 获取设置了自动清理的用户列表
func (u *UserDAO) GetUsersWithAutoClean(ctx context.Context) ([]*UserWithAutoClean, error) {
	var results []*UserWithAutoClean

	// 从用户偏好设置表中获取设置了自动清理的用户
	err := u.db.WithContext(ctx).
		Table("user_preferences").
		Select("user_id as id, auto_clean_days").
		Where("auto_clean_days > 0").
		Scan(&results).Error

	return results, err
}

// BatchUpdateStatus 批量更新用户状态
func (u *UserDAO) BatchUpdateStatus(ctx context.Context, userIDs []string, status string) error {
	// 验证状态值
	validStatuses := map[string]bool{"active": true, "banned": true, "pending": true}
	if !validStatuses[status] {
		return ErrInvalidID
	}

	if len(userIDs) == 0 {
		return nil
	}

	return u.db.WithContext(ctx).Model(&model.User{}).
		Where("id IN ?", userIDs).
		Update("status", status).Error
}

// BatchDelete 批量删除用户
func (u *UserDAO) BatchDelete(ctx context.Context, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	return u.db.WithContext(ctx).Where("id IN ?", userIDs).Delete(&model.User{}).Error
}

package dao

import (
	"context"

	"github.com/ai-research-platform/internal/repository/model"
	"github.com/ai-research-platform/internal/types/constant"
	"gorm.io/gorm"
)

// UserPreferencesDAO 用户偏好设置数据访问对象
type UserPreferencesDAO struct {
	db *gorm.DB
}

// NewUserPreferencesDAO 创建用户偏好设置DAO
func NewUserPreferencesDAO(db *gorm.DB) *UserPreferencesDAO {
	return &UserPreferencesDAO{db: db}
}

// GetByUserID 根据用户ID获取偏好设置
func (d *UserPreferencesDAO) GetByUserID(ctx context.Context, userID string) (*model.UserPreferences, error) {
	var prefs model.UserPreferences
	err := d.db.WithContext(ctx).Where("user_id = ?", userID).First(&prefs).Error
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}

// Create 创建用户偏好设置
func (d *UserPreferencesDAO) Create(ctx context.Context, prefs *model.UserPreferences) error {
	return d.db.WithContext(ctx).Create(prefs).Error
}

// Update 更新用户偏好设置
func (d *UserPreferencesDAO) Update(ctx context.Context, prefs *model.UserPreferences) error {
	return d.db.WithContext(ctx).Save(prefs).Error
}

// GetOrCreate 获取或创建用户偏好设置
func (d *UserPreferencesDAO) GetOrCreate(ctx context.Context, userID string) (*model.UserPreferences, error) {
	prefs, err := d.GetByUserID(ctx, userID)
	if err == nil {
		return prefs, nil
	}

	// 如果不存在，创建默认设置
	if err == gorm.ErrRecordNotFound {
		prefs = &model.UserPreferences{
			UserID:              userID,
			Theme:               "light",
			Language:            "zh",
			DefaultLLMProvider:  constant.DefaultProvider,
			DefaultModel:        constant.DefaultModel,
			StreamEnabled:       true,
			NotificationEnabled: true,
			AutoSaveEnabled:     true,
			TimeZone:            "Asia/Shanghai",
			MemoryEnabled:       true,
			MaxContextTokens:    128000,
		}
		if err := d.Create(ctx, prefs); err != nil {
			return nil, err
		}
		return prefs, nil
	}

	return nil, err
}

// UpdateMemorySettings 更新记忆设置
func (d *UserPreferencesDAO) UpdateMemorySettings(ctx context.Context, userID string, memoryEnabled bool, customPrompt string, maxTokens int) error {
	return d.db.WithContext(ctx).
		Model(&model.UserPreferences{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"memory_enabled":       memoryEnabled,
			"custom_system_prompt": customPrompt,
			"max_context_tokens":   maxTokens,
		}).Error
}

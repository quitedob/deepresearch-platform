package dao

import (
	"context"

	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// QuotaConfigDAO 配额配置数据访问对象
type QuotaConfigDAO struct {
	db *gorm.DB
}

// NewQuotaConfigDAO 创建配额配置DAO
func NewQuotaConfigDAO(db *gorm.DB) *QuotaConfigDAO {
	return &QuotaConfigDAO{db: db}
}

// InitDefaultConfigs 初始化默认配额配置
func (d *QuotaConfigDAO) InitDefaultConfigs(ctx context.Context) error {
	configs := []model.QuotaConfig{
		{
			MembershipType:   "free",
			ChatLimit:        10,
			ResearchLimit:    1,
			ResetPeriodHours: 0, // 普通用户不自动重置
			Description:      "普通用户默认配额",
		},
		{
			MembershipType:   "premium",
			ChatLimit:        50,
			ResearchLimit:    10,
			ResetPeriodHours: 5, // 5小时重置
			Description:      "高级会员默认配额",
		},
	}

	for _, cfg := range configs {
		var existing model.QuotaConfig
		if err := d.db.WithContext(ctx).Where("membership_type = ?", cfg.MembershipType).First(&existing).Error; err == gorm.ErrRecordNotFound {
			d.db.WithContext(ctx).Create(&cfg)
		}
	}

	return nil
}

// GetByMembershipType 根据会员类型获取配额配置
func (d *QuotaConfigDAO) GetByMembershipType(ctx context.Context, membershipType string) (*model.QuotaConfig, error) {
	var config model.QuotaConfig
	err := d.db.WithContext(ctx).Where("membership_type = ?", membershipType).First(&config).Error
	return &config, err
}

// GetAll 获取所有配额配置
func (d *QuotaConfigDAO) GetAll(ctx context.Context) ([]*model.QuotaConfig, error) {
	var configs []*model.QuotaConfig
	err := d.db.WithContext(ctx).Find(&configs).Error
	return configs, err
}

// Update 更新配额配置
func (d *QuotaConfigDAO) Update(ctx context.Context, config *model.QuotaConfig) error {
	return d.db.WithContext(ctx).Save(config).Error
}

// UpdateByMembershipType 根据会员类型更新配额配置
func (d *QuotaConfigDAO) UpdateByMembershipType(ctx context.Context, membershipType string, chatLimit, researchLimit, resetPeriodHours int, updatedBy string) error {
	return d.db.WithContext(ctx).Model(&model.QuotaConfig{}).
		Where("membership_type = ?", membershipType).
		Updates(map[string]interface{}{
			"chat_limit":        chatLimit,
			"research_limit":    researchLimit,
			"reset_period_hours": resetPeriodHours,
			"updated_by":        updatedBy,
		}).Error
}

// ApplyToAllUsers 将配额配置应用到所有该类型的用户
func (d *QuotaConfigDAO) ApplyToAllUsers(ctx context.Context, membershipType string, chatLimit, researchLimit int) error {
	if membershipType == "premium" {
		return d.db.WithContext(ctx).Model(&model.UserMembership{}).
			Where("membership_type = ?", membershipType).
			Updates(map[string]interface{}{
				"premium_chat_limit":     chatLimit,
				"premium_research_limit": researchLimit,
			}).Error
	}
	return d.db.WithContext(ctx).Model(&model.UserMembership{}).
		Where("membership_type = ?", membershipType).
		Updates(map[string]interface{}{
			"normal_chat_limit": chatLimit,
			"research_limit":    researchLimit,
		}).Error
}

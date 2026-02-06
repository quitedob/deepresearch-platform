package dao

import (
	"context"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// MembershipDAO 会员数据访问对象
type MembershipDAO struct {
	db *gorm.DB
}

// NewMembershipDAO 创建会员DAO
func NewMembershipDAO(db *gorm.DB) *MembershipDAO {
	return &MembershipDAO{db: db}
}

// GetOrCreateMembership 获取或创建用户会员信息
func (d *MembershipDAO) GetOrCreateMembership(ctx context.Context, userID string) (*model.UserMembership, error) {
	var membership model.UserMembership
	err := d.db.WithContext(ctx).Where("user_id = ?", userID).First(&membership).Error
	if err == gorm.ErrRecordNotFound {
		// 从配额配置表读取默认值
		var quotaConfig model.QuotaConfig
		chatLimit := 10      // 默认值
		researchLimit := 1   // 默认值
		if err := d.db.WithContext(ctx).Where("membership_type = ?", "free").First(&quotaConfig).Error; err == nil {
			chatLimit = quotaConfig.ChatLimit
			researchLimit = quotaConfig.ResearchLimit
		}
		
		membership = model.UserMembership{
			UserID:           userID,
			MembershipType:   model.MembershipFree,
			NormalChatLimit:  chatLimit,
			NormalChatUsed:   0,
			ResearchLimit:    researchLimit,
			ResearchUsed:     0,
		}
		if err := d.db.WithContext(ctx).Create(&membership).Error; err != nil {
			return nil, err
		}
		return &membership, nil
	}
	return &membership, err
}

// GetMembershipByUserID 根据用户ID获取会员信息
func (d *MembershipDAO) GetMembershipByUserID(ctx context.Context, userID string) (*model.UserMembership, error) {
	var membership model.UserMembership
	err := d.db.WithContext(ctx).Where("user_id = ?", userID).First(&membership).Error
	return &membership, err
}

// UpdateMembership 更新会员信息
func (d *MembershipDAO) UpdateMembership(ctx context.Context, membership *model.UserMembership) error {
	return d.db.WithContext(ctx).Save(membership).Error
}

// IncrementChatUsage 增加聊天使用次数
// 修复：使用原子更新避免并发问题
func (d *MembershipDAO) IncrementChatUsage(ctx context.Context, userID string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var membership model.UserMembership
		// 使用 FOR UPDATE 锁定行，防止并发更新
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("user_id = ?", userID).First(&membership).Error
		if err == gorm.ErrRecordNotFound {
			// 从配额配置表读取默认值
			var quotaConfig model.QuotaConfig
			chatLimit := 10      // 默认值
			researchLimit := 1   // 默认值
			if err := tx.Where("membership_type = ?", "free").First(&quotaConfig).Error; err == nil {
				chatLimit = quotaConfig.ChatLimit
				researchLimit = quotaConfig.ResearchLimit
			}
			
			// 创建新记录
			membership = model.UserMembership{
				UserID:          userID,
				MembershipType:  model.MembershipFree,
				NormalChatLimit: chatLimit,
				NormalChatUsed:  1, // 直接设置为1
				ResearchLimit:   researchLimit,
				ResearchUsed:    0,
			}
			return tx.Create(&membership).Error
		} else if err != nil {
			return err
		}

		if membership.MembershipType == model.MembershipPremium {
			// 检查是否需要重置
			if membership.PremiumResetAt != nil && time.Now().After(*membership.PremiumResetAt) {
				membership.PremiumChatUsed = 0
				membership.PremiumResearchUsed = 0
				resetTime := time.Now().Add(5 * time.Hour)
				membership.PremiumResetAt = &resetTime
			}
			membership.PremiumChatUsed++
		} else {
			membership.NormalChatUsed++
		}

		return tx.Save(&membership).Error
	})
}

// IncrementResearchUsage 增加研究使用次数
// 修复：使用原子更新避免并发问题
func (d *MembershipDAO) IncrementResearchUsage(ctx context.Context, userID string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var membership model.UserMembership
		// 使用 FOR UPDATE 锁定行，防止并发更新
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("user_id = ?", userID).First(&membership).Error
		if err == gorm.ErrRecordNotFound {
			membership = model.UserMembership{
				UserID:          userID,
				MembershipType:  model.MembershipFree,
				NormalChatLimit: 10,
				NormalChatUsed:  0,
				ResearchLimit:   1,
				ResearchUsed:    1, // 直接设置为1
			}
			return tx.Create(&membership).Error
		} else if err != nil {
			return err
		}

		if membership.MembershipType == model.MembershipPremium {
			if membership.PremiumResetAt != nil && time.Now().After(*membership.PremiumResetAt) {
				membership.PremiumChatUsed = 0
				membership.PremiumResearchUsed = 0
				resetTime := time.Now().Add(5 * time.Hour)
				membership.PremiumResetAt = &resetTime
			}
			membership.PremiumResearchUsed++
		} else {
			membership.ResearchUsed++
		}

		return tx.Save(&membership).Error
	})
}

// CheckChatQuota 检查聊天配额
func (d *MembershipDAO) CheckChatQuota(ctx context.Context, userID string) (bool, int, int, error) {
	membership, err := d.GetOrCreateMembership(ctx, userID)
	if err != nil {
		return false, 0, 0, err
	}

	if membership.MembershipType == model.MembershipPremium {
		// 检查是否需要重置
		if membership.PremiumResetAt != nil && time.Now().After(*membership.PremiumResetAt) {
			membership.PremiumChatUsed = 0
			membership.PremiumResearchUsed = 0
			resetTime := time.Now().Add(5 * time.Hour)
			membership.PremiumResetAt = &resetTime
			d.db.WithContext(ctx).Save(membership)
		}
		remaining := membership.PremiumChatLimit - membership.PremiumChatUsed
		return remaining > 0, remaining, membership.PremiumChatLimit, nil
	}

	remaining := membership.NormalChatLimit - membership.NormalChatUsed
	return remaining > 0, remaining, membership.NormalChatLimit, nil
}

// CheckResearchQuota 检查研究配额
func (d *MembershipDAO) CheckResearchQuota(ctx context.Context, userID string) (bool, int, int, error) {
	membership, err := d.GetOrCreateMembership(ctx, userID)
	if err != nil {
		return false, 0, 0, err
	}

	if membership.MembershipType == model.MembershipPremium {
		if membership.PremiumResetAt != nil && time.Now().After(*membership.PremiumResetAt) {
			membership.PremiumChatUsed = 0
			membership.PremiumResearchUsed = 0
			resetTime := time.Now().Add(5 * time.Hour)
			membership.PremiumResetAt = &resetTime
			d.db.WithContext(ctx).Save(membership)
		}
		remaining := membership.PremiumResearchLimit - membership.PremiumResearchUsed
		return remaining > 0, remaining, membership.PremiumResearchLimit, nil
	}

	remaining := membership.ResearchLimit - membership.ResearchUsed
	return remaining > 0, remaining, membership.ResearchLimit, nil
}

// ResetUserQuota 重置用户配额
func (d *MembershipDAO) ResetUserQuota(ctx context.Context, userID string) error {
	membership, err := d.GetOrCreateMembership(ctx, userID)
	if err != nil {
		return err
	}

	membership.NormalChatUsed = 0
	membership.ResearchUsed = 0
	membership.PremiumChatUsed = 0
	membership.PremiumResearchUsed = 0

	return d.db.WithContext(ctx).Save(membership).Error
}

// SetUserQuota 设置用户配额
func (d *MembershipDAO) SetUserQuota(ctx context.Context, userID string, chatLimit, researchLimit int) error {
	membership, err := d.GetOrCreateMembership(ctx, userID)
	if err != nil {
		return err
	}

	if membership.MembershipType == model.MembershipPremium {
		membership.PremiumChatLimit = chatLimit
		membership.PremiumResearchLimit = researchLimit
	} else {
		membership.NormalChatLimit = chatLimit
		membership.ResearchLimit = researchLimit
	}

	return d.db.WithContext(ctx).Save(membership).Error
}

// UpgradeToPremium 升级为高级会员
// 修复：使用事务确保会员升级和激活记录的原子性
func (d *MembershipDAO) UpgradeToPremium(ctx context.Context, userID string, validDays int, method string, codeID *string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 在事务中获取或创建会员信息
		var membership model.UserMembership
		err := tx.Where("user_id = ?", userID).First(&membership).Error
		if err == gorm.ErrRecordNotFound {
			membership = model.UserMembership{
				UserID:          userID,
				MembershipType:  model.MembershipFree,
				NormalChatLimit: 10,
				NormalChatUsed:  0,
				ResearchLimit:   1,
				ResearchUsed:    0,
			}
			if err := tx.Create(&membership).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		now := time.Now()
		expiresAt := now.AddDate(0, 0, validDays)
		resetAt := now.Add(5 * time.Hour)

		membership.MembershipType = model.MembershipPremium
		membership.ActivatedAt = &now
		membership.ExpiresAt = &expiresAt
		membership.ActivationMethod = method
		membership.ActivationCodeID = codeID
		membership.PremiumResetAt = &resetAt
		membership.PremiumChatUsed = 0
		membership.PremiumResearchUsed = 0

		// 设置默认的高级会员配额
		if membership.PremiumChatLimit == 0 {
			membership.PremiumChatLimit = 1000
		}
		if membership.PremiumResearchLimit == 0 {
			membership.PremiumResearchLimit = 50
		}

		return tx.Save(&membership).Error
	})
}

// DowngradeToFree 降级为普通用户
func (d *MembershipDAO) DowngradeToFree(ctx context.Context, userID string) error {
	membership, err := d.GetOrCreateMembership(ctx, userID)
	if err != nil {
		return err
	}

	membership.MembershipType = model.MembershipFree
	membership.ExpiresAt = nil
	membership.NormalChatUsed = 0
	membership.ResearchUsed = 0

	return d.db.WithContext(ctx).Save(membership).Error
}

// ListAllMemberships 获取所有会员信息
func (d *MembershipDAO) ListAllMemberships(ctx context.Context, limit, offset int) ([]*model.UserMembership, int64, error) {
	var memberships []*model.UserMembership
	var total int64

	d.db.WithContext(ctx).Model(&model.UserMembership{}).Count(&total)
	err := d.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&memberships).Error

	return memberships, total, err
}

// CountByMembershipType 按会员类型统计数量
// 修复：使用数据库聚合查询代替遍历
func (d *MembershipDAO) CountByMembershipType(ctx context.Context, membershipType model.MembershipType) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.UserMembership{}).
		Where("membership_type = ?", membershipType).
		Count(&count).Error
	return count, err
}

// CheckAndDeductChatQuota 原子性检查并扣减聊天配额
// 返回: 是否成功扣减, 剩余次数, 总限制, 错误
// 修复：使用事务确保检查和扣减的原子性，防止并发超额使用
func (d *MembershipDAO) CheckAndDeductChatQuota(ctx context.Context, userID string) (bool, int, int, error) {
	var hasQuota bool
	var remaining, limit int

	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var membership model.UserMembership
		// 使用 FOR UPDATE 锁定行
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("user_id = ?", userID).First(&membership).Error
		if err == gorm.ErrRecordNotFound {
			// 创建新记录并扣减1次
			membership = model.UserMembership{
				UserID:          userID,
				MembershipType:  model.MembershipFree,
				NormalChatLimit: 10,
				NormalChatUsed:  1, // 直接扣减
				ResearchLimit:   1,
				ResearchUsed:    0,
			}
			if err := tx.Create(&membership).Error; err != nil {
				return err
			}
			hasQuota = true
			remaining = 9
			limit = 10
			return nil
		} else if err != nil {
			return err
		}

		if membership.MembershipType == model.MembershipPremium {
			// 检查是否需要重置
			if membership.PremiumResetAt != nil && time.Now().After(*membership.PremiumResetAt) {
				membership.PremiumChatUsed = 0
				membership.PremiumResearchUsed = 0
				resetTime := time.Now().Add(5 * time.Hour)
				membership.PremiumResetAt = &resetTime
			}
			
			limit = membership.PremiumChatLimit
			if membership.PremiumChatUsed >= membership.PremiumChatLimit {
				hasQuota = false
				remaining = 0
				return nil // 不扣减，返回配额不足
			}
			
			membership.PremiumChatUsed++
			remaining = membership.PremiumChatLimit - membership.PremiumChatUsed
			hasQuota = true
		} else {
			limit = membership.NormalChatLimit
			if membership.NormalChatUsed >= membership.NormalChatLimit {
				hasQuota = false
				remaining = 0
				return nil // 不扣减，返回配额不足
			}
			
			membership.NormalChatUsed++
			remaining = membership.NormalChatLimit - membership.NormalChatUsed
			hasQuota = true
		}

		return tx.Save(&membership).Error
	})

	return hasQuota, remaining, limit, err
}

// CheckAndDeductResearchQuota 原子性检查并扣减研究配额
// 返回: 是否成功扣减, 剩余次数, 总限制, 错误
// 修复：使用事务确保检查和扣减的原子性，防止并发超额使用
func (d *MembershipDAO) CheckAndDeductResearchQuota(ctx context.Context, userID string) (bool, int, int, error) {
	var hasQuota bool
	var remaining, limit int

	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var membership model.UserMembership
		// 使用 FOR UPDATE 锁定行
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("user_id = ?", userID).First(&membership).Error
		if err == gorm.ErrRecordNotFound {
			// 创建新记录并扣减1次
			membership = model.UserMembership{
				UserID:          userID,
				MembershipType:  model.MembershipFree,
				NormalChatLimit: 10,
				NormalChatUsed:  0,
				ResearchLimit:   1,
				ResearchUsed:    1, // 直接扣减
			}
			if err := tx.Create(&membership).Error; err != nil {
				return err
			}
			hasQuota = true
			remaining = 0
			limit = 1
			return nil
		} else if err != nil {
			return err
		}

		if membership.MembershipType == model.MembershipPremium {
			// 检查是否需要重置
			if membership.PremiumResetAt != nil && time.Now().After(*membership.PremiumResetAt) {
				membership.PremiumChatUsed = 0
				membership.PremiumResearchUsed = 0
				resetTime := time.Now().Add(5 * time.Hour)
				membership.PremiumResetAt = &resetTime
			}
			
			limit = membership.PremiumResearchLimit
			if membership.PremiumResearchUsed >= membership.PremiumResearchLimit {
				hasQuota = false
				remaining = 0
				return nil // 不扣减，返回配额不足
			}
			
			membership.PremiumResearchUsed++
			remaining = membership.PremiumResearchLimit - membership.PremiumResearchUsed
			hasQuota = true
		} else {
			limit = membership.ResearchLimit
			if membership.ResearchUsed >= membership.ResearchLimit {
				hasQuota = false
				remaining = 0
				return nil // 不扣减，返回配额不足
			}
			
			membership.ResearchUsed++
			remaining = membership.ResearchLimit - membership.ResearchUsed
			hasQuota = true
		}

		return tx.Save(&membership).Error
	})

	return hasQuota, remaining, limit, err
}

// RefundChatQuota 退还聊天配额（用于请求失败时）
func (d *MembershipDAO) RefundChatQuota(ctx context.Context, userID string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var membership model.UserMembership
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("user_id = ?", userID).First(&membership).Error
		if err != nil {
			return err
		}

		if membership.MembershipType == model.MembershipPremium {
			if membership.PremiumChatUsed > 0 {
				membership.PremiumChatUsed--
			}
		} else {
			if membership.NormalChatUsed > 0 {
				membership.NormalChatUsed--
			}
		}

		return tx.Save(&membership).Error
	})
}

// RefundResearchQuota 退还研究配额（用于请求失败时）
func (d *MembershipDAO) RefundResearchQuota(ctx context.Context, userID string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var membership model.UserMembership
		err := tx.Set("gorm:query_option", "FOR UPDATE").
			Where("user_id = ?", userID).First(&membership).Error
		if err != nil {
			return err
		}

		if membership.MembershipType == model.MembershipPremium {
			if membership.PremiumResearchUsed > 0 {
				membership.PremiumResearchUsed--
			}
		} else {
			if membership.ResearchUsed > 0 {
				membership.ResearchUsed--
			}
		}

		return tx.Save(&membership).Error
	})
}

// ResetExpiredQuotas 重置所有过期的配额
// 返回重置的用户数量
func (d *MembershipDAO) ResetExpiredQuotas(ctx context.Context) (int, error) {
	now := time.Now()
	
	// 重置高级会员的过期配额
	result := d.db.WithContext(ctx).Model(&model.UserMembership{}).
		Where("membership_type = ? AND premium_reset_at IS NOT NULL AND premium_reset_at < ?", 
			model.MembershipPremium, now).
		Updates(map[string]interface{}{
			"premium_chat_used":     0,
			"premium_research_used": 0,
			"premium_reset_at":      now.Add(5 * time.Hour),
		})
	
	if result.Error != nil {
		return 0, result.Error
	}
	
	return int(result.RowsAffected), nil
}

// BatchResetUserQuotas 批量重置用户配额
func (d *MembershipDAO) BatchResetUserQuotas(ctx context.Context, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}
	
	return d.db.WithContext(ctx).Model(&model.UserMembership{}).
		Where("user_id IN ?", userIDs).
		Updates(map[string]interface{}{
			"normal_chat_used":      0,
			"research_used":         0,
			"premium_chat_used":     0,
			"premium_research_used": 0,
		}).Error
}

// BatchSetUserQuotas 批量设置用户配额
func (d *MembershipDAO) BatchSetUserQuotas(ctx context.Context, userIDs []string, chatLimit, researchLimit int) error {
	if len(userIDs) == 0 {
		return nil
	}
	
	return d.db.WithContext(ctx).Model(&model.UserMembership{}).
		Where("user_id IN ?", userIDs).
		Updates(map[string]interface{}{
			"normal_chat_limit": chatLimit,
			"research_limit":    researchLimit,
		}).Error
}

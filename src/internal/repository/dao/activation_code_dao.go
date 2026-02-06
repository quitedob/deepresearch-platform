package dao

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// 激活码相关错误
var (
	ErrCodeNotFound       = errors.New("activation code not found")
	ErrCodeInactive       = errors.New("activation code is inactive")
	ErrCodeExpired        = errors.New("activation code has expired")
	ErrCodeExhausted      = errors.New("activation code has reached maximum activations")
	ErrCodeAlreadyUsed    = errors.New("user has already used this activation code")
	ErrDeadlockRetryLimit = errors.New("deadlock retry limit exceeded")
)

// ActivationCodeDAO 激活码数据访问对象
type ActivationCodeDAO struct {
	db *gorm.DB
}

// NewActivationCodeDAO 创建激活码DAO
func NewActivationCodeDAO(db *gorm.DB) *ActivationCodeDAO {
	return &ActivationCodeDAO{db: db}
}

// GenerateCode 生成随机激活码
func GenerateCode() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Create 创建激活码
func (d *ActivationCodeDAO) Create(ctx context.Context, code *model.ActivationCode) error {
	if code.Code == "" {
		code.Code = GenerateCode()
	}
	return d.db.WithContext(ctx).Create(code).Error
}

// GetByID 根据ID获取激活码
func (d *ActivationCodeDAO) GetByID(ctx context.Context, id string) (*model.ActivationCode, error) {
	var code model.ActivationCode
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&code).Error
	return &code, err
}

// GetByCode 根据激活码获取
func (d *ActivationCodeDAO) GetByCode(ctx context.Context, code string) (*model.ActivationCode, error) {
	var activationCode model.ActivationCode
	err := d.db.WithContext(ctx).Where("code = ?", code).First(&activationCode).Error
	return &activationCode, err
}

// Update 更新激活码
func (d *ActivationCodeDAO) Update(ctx context.Context, code *model.ActivationCode) error {
	return d.db.WithContext(ctx).Save(code).Error
}

// Delete 删除激活码
func (d *ActivationCodeDAO) Delete(ctx context.Context, id string) error {
	return d.db.WithContext(ctx).Delete(&model.ActivationCode{}, "id = ?", id).Error
}

// List 获取激活码列表
func (d *ActivationCodeDAO) List(ctx context.Context, limit, offset int) ([]*model.ActivationCode, int64, error) {
	var codes []*model.ActivationCode
	var total int64

	d.db.WithContext(ctx).Model(&model.ActivationCode{}).Count(&total)
	err := d.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&codes).Error

	return codes, total, err
}

// UseCodeResult 激活码使用结果
type UseCodeResult struct {
	ActivationCode *model.ActivationCode
	RetryCount     int
	Error          error
	ErrorCode      string // 错误码，用于前端展示
}

// UseCode 使用激活码
// 修复：添加重试机制处理deadlock，返回详细错误信息
func (d *ActivationCodeDAO) UseCode(ctx context.Context, code string, userID string) (*model.ActivationCode, error) {
	result := d.UseCodeWithRetry(ctx, code, userID, 3)
	return result.ActivationCode, result.Error
}

// UseCodeWithRetry 使用激活码（带重试机制）
// maxRetries: 最大重试次数，建议3-5次
func (d *ActivationCodeDAO) UseCodeWithRetry(ctx context.Context, code string, userID string, maxRetries int) *UseCodeResult {
	result := &UseCodeResult{}
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		result.RetryCount = attempt
		
		activationCode, err := d.useCodeInternal(ctx, code, userID)
		if err == nil {
			result.ActivationCode = activationCode
			result.Error = nil
			return result
		}
		
		// 检查是否是deadlock错误，如果是则重试
		if isDeadlockError(err) && attempt < maxRetries {
			// 指数退避：50ms, 100ms, 200ms...
			backoff := time.Duration(50*(1<<attempt)) * time.Millisecond
			time.Sleep(backoff)
			continue
		}
		
		// 设置详细错误信息
		result.Error = err
		result.ErrorCode = getActivationErrorCode(err)
		return result
	}
	
	result.Error = ErrDeadlockRetryLimit
	result.ErrorCode = "ERR_DEADLOCK_RETRY_LIMIT"
	return result
}

// useCodeInternal 内部激活码使用逻辑
func (d *ActivationCodeDAO) useCodeInternal(ctx context.Context, code string, userID string) (*model.ActivationCode, error) {
	var activationCode *model.ActivationCode
	
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用 FOR UPDATE NOWAIT 避免长时间等待锁
		var ac model.ActivationCode
		if err := tx.Set("gorm:query_option", "FOR UPDATE NOWAIT").
			Where("code = ?", code).First(&ac).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCodeNotFound
			}
			return err
		}

		// 检查激活码是否有效
		if !ac.IsActive {
			return ErrCodeInactive
		}

		// 检查是否过期
		if ac.ExpiresAt != nil && time.Now().After(*ac.ExpiresAt) {
			return ErrCodeExpired
		}

		// 检查使用次数
		if ac.UsedActivations >= ac.MaxActivations {
			return ErrCodeExhausted
		}

		// 检查用户是否已使用过此激活码
		var existingRecord model.ActivationRecord
		err := tx.Where("activation_code_id = ? AND user_id = ?", ac.ID, userID).First(&existingRecord).Error
		if err == nil {
			return ErrCodeAlreadyUsed
		}

		// 创建激活记录
		record := &model.ActivationRecord{
			ActivationCodeID: ac.ID,
			UserID:           userID,
			ActivatedAt:      time.Now(),
		}
		if err := tx.Create(record).Error; err != nil {
			return fmt.Errorf("failed to create activation record: %w", err)
		}

		// 更新使用次数
		ac.UsedActivations++
		if err := tx.Save(&ac).Error; err != nil {
			return fmt.Errorf("failed to update activation count: %w", err)
		}

		activationCode = &ac
		return nil
	})

	return activationCode, err
}

// isDeadlockError 检查是否是deadlock错误
func isDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "deadlock") || 
		contains(errStr, "lock wait timeout") ||
		contains(errStr, "could not obtain lock")
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || len(substr) == 0 || 
		 findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFold(s, t string) bool {
	for i := 0; i < len(s); i++ {
		sr, tr := s[i], t[i]
		if sr >= 'A' && sr <= 'Z' {
			sr += 'a' - 'A'
		}
		if tr >= 'A' && tr <= 'Z' {
			tr += 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}
	return true
}

// getActivationErrorCode 获取激活错误码
func getActivationErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrCodeNotFound):
		return "ERR_CODE_NOT_FOUND"
	case errors.Is(err, ErrCodeInactive):
		return "ERR_CODE_INACTIVE"
	case errors.Is(err, ErrCodeExpired):
		return "ERR_CODE_EXPIRED"
	case errors.Is(err, ErrCodeExhausted):
		return "ERR_CODE_EXHAUSTED"
	case errors.Is(err, ErrCodeAlreadyUsed):
		return "ERR_CODE_ALREADY_USED"
	default:
		return "ERR_ACTIVATION_FAILED"
	}
}

// GetActivationRecords 获取激活码的激活记录
func (d *ActivationCodeDAO) GetActivationRecords(ctx context.Context, codeID string) ([]*model.ActivationRecord, error) {
	var records []*model.ActivationRecord
	err := d.db.WithContext(ctx).Where("activation_code_id = ?", codeID).Find(&records).Error
	return records, err
}

// GetActivationRecordsWithUsers 获取激活码的激活记录（包含用户信息）
func (d *ActivationCodeDAO) GetActivationRecordsWithUsers(ctx context.Context, codeID string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	rows, err := d.db.WithContext(ctx).Table("activation_records").
		Select("activation_records.*, users.username, users.email").
		Joins("LEFT JOIN users ON activation_records.user_id = users.id").
		Where("activation_records.activation_code_id = ?", codeID).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var record model.ActivationRecord
		var username, email string
		if err := rows.Scan(&record.ID, &record.ActivationCodeID, &record.UserID, &record.ActivatedAt, &username, &email); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id":                 record.ID,
			"activation_code_id": record.ActivationCodeID,
			"user_id":            record.UserID,
			"activated_at":       record.ActivatedAt,
			"username":           username,
			"email":              email,
		})
	}

	return results, nil
}

// CountActive 统计有效激活码数量
// 修复：使用数据库聚合查询代替遍历
func (d *ActivationCodeDAO) CountActive(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.ActivationCode{}).
		Where("is_active = ?", true).
		Count(&count).Error
	return count, err
}

package dao

import (
	"context"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// ToolCallDAO 工具调用记录数据访问对象
type ToolCallDAO struct {
	db *gorm.DB
}

// NewToolCallDAO 创建 ToolCallDAO
func NewToolCallDAO(db *gorm.DB) *ToolCallDAO {
	return &ToolCallDAO{db: db}
}

// Create 创建工具调用记录
func (d *ToolCallDAO) Create(ctx context.Context, record *model.ToolCallRecord) error {
	return d.db.WithContext(ctx).Create(record).Error
}

// GetByID 根据ID获取记录
func (d *ToolCallDAO) GetByID(ctx context.Context, id string) (*model.ToolCallRecord, error) {
	var record model.ToolCallRecord
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByResearchID 根据研究ID获取所有工具调用记录
func (d *ToolCallDAO) GetByResearchID(ctx context.Context, researchID string, limit, offset int) ([]*model.ToolCallRecord, error) {
	var records []*model.ToolCallRecord
	query := d.db.WithContext(ctx).Where("research_id = ?", researchID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	err := query.Find(&records).Error
	return records, err
}

// GetByToolName 根据工具名称获取记录
func (d *ToolCallDAO) GetByToolName(ctx context.Context, toolName string, limit, offset int) ([]*model.ToolCallRecord, error) {
	var records []*model.ToolCallRecord
	query := d.db.WithContext(ctx).Where("tool_name = ?", toolName).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	err := query.Find(&records).Error
	return records, err
}

// GetFailedCalls 获取失败的调用记录
func (d *ToolCallDAO) GetFailedCalls(ctx context.Context, limit, offset int) ([]*model.ToolCallRecord, error) {
	var records []*model.ToolCallRecord
	query := d.db.WithContext(ctx).Where("success = ?", false).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}
	err := query.Find(&records).Error
	return records, err
}

// GetStats 获取工具调用统计
func (d *ToolCallDAO) GetStats(ctx context.Context) ([]*model.ToolCallStats, error) {
	var stats []*model.ToolCallStats
	err := d.db.WithContext(ctx).Model(&model.ToolCallRecord{}).
		Select(`
			tool_name,
			COUNT(*) as total_calls,
			SUM(CASE WHEN success = true THEN 1 ELSE 0 END) as success_calls,
			SUM(CASE WHEN success = false THEN 1 ELSE 0 END) as failed_calls,
			AVG(duration_ms) as avg_duration_ms,
			SUM(retry_count) as total_retries,
			CAST(SUM(CASE WHEN success = true THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*) as success_rate
		`).
		Group("tool_name").
		Scan(&stats).Error
	return stats, err
}

// GetStatsByTimeRange 获取指定时间范围内的统计
func (d *ToolCallDAO) GetStatsByTimeRange(ctx context.Context, start, end time.Time) ([]*model.ToolCallStats, error) {
	var stats []*model.ToolCallStats
	err := d.db.WithContext(ctx).Model(&model.ToolCallRecord{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Select(`
			tool_name,
			COUNT(*) as total_calls,
			SUM(CASE WHEN success = true THEN 1 ELSE 0 END) as success_calls,
			SUM(CASE WHEN success = false THEN 1 ELSE 0 END) as failed_calls,
			AVG(duration_ms) as avg_duration_ms,
			SUM(retry_count) as total_retries,
			CAST(SUM(CASE WHEN success = true THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*) as success_rate
		`).
		Group("tool_name").
		Scan(&stats).Error
	return stats, err
}

// DeleteOldRecords 删除旧记录（保留最近N天）
func (d *ToolCallDAO) DeleteOldRecords(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := d.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&model.ToolCallRecord{})
	return result.RowsAffected, result.Error
}

// CountByInputHash 统计相同输入的调用次数（用于分析去重效果）
func (d *ToolCallDAO) CountByInputHash(ctx context.Context, inputHash string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.ToolCallRecord{}).
		Where("input_hash = ?", inputHash).
		Count(&count).Error
	return count, err
}

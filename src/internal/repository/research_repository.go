package repository

import (
	"context"
	"fmt"

	"github.com/ai-research-platform/internal/models"
	"gorm.io/gorm"
)

// ResearchRepository 定义研究相关数据库操作的接口
type ResearchRepository interface {
	// 会话操作
	CreateSession(ctx context.Context, session *models.ResearchSession) error
	GetSession(ctx context.Context, sessionID string) (*models.ResearchSession, error)
	GetSessionsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.ResearchSession, error)
	UpdateSessionStatus(ctx context.Context, sessionID string, status string, progress float32) error
	DeleteSession(ctx context.Context, sessionID string) error

	// 任务操作
	SaveTask(ctx context.Context, task *models.ResearchTask) error
	GetTask(ctx context.Context, taskID string) (*models.ResearchTask, error)
	GetTasksByResearch(ctx context.Context, researchID string, limit, offset int) ([]*models.ResearchTask, error)
	UpdateTaskStatus(ctx context.Context, taskID string, status string) error

	// 结果操作
	SaveResult(ctx context.Context, result *models.ResearchResult) error
	GetResult(ctx context.Context, researchID string) (*models.ResearchResult, error)

	// 事务支持
	WithTransaction(ctx context.Context, fn func(repo ResearchRepository) error) error
}

// researchRepository 是 ResearchRepository 的具体实现
type researchRepository struct {
	db *gorm.DB
}

// NewResearchRepository 创建新的 ResearchRepository 实例
func NewResearchRepository(db *gorm.DB) ResearchRepository {
	return &researchRepository{db: db}
}

// CreateSession 在数据库中创建新的研究会话
func (r *researchRepository) CreateSession(ctx context.Context, session *models.ResearchSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("failed to create research session: %w", err)
	}
	return nil
}

// GetSession 根据 ID 获取研究会话
func (r *researchRepository) GetSession(ctx context.Context, sessionID string) (*models.ResearchSession, error) {
	var session models.ResearchSession
	if err := r.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("research session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get research session: %w", err)
	}
	return &session, nil
}

// GetSessionsByUser 获取特定用户的研究会话（支持分页）
func (r *researchRepository) GetSessionsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.ResearchSession, error) {
	var sessions []*models.ResearchSession
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to get user research sessions: %w", err)
	}
	return sessions, nil
}

// UpdateSessionStatus 更新研究会话的状态和进度
func (r *researchRepository) UpdateSessionStatus(ctx context.Context, sessionID string, status string, progress float32) error {
	result := r.db.WithContext(ctx).
		Model(&models.ResearchSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"status":   status,
			"progress": progress,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update research session status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("research session not found: %s", sessionID)
	}
	return nil
}

// DeleteSession 软删除研究会话
func (r *researchRepository) DeleteSession(ctx context.Context, sessionID string) error {
	result := r.db.WithContext(ctx).Delete(&models.ResearchSession{}, "id = ?", sessionID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete research session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("research session not found: %s", sessionID)
	}
	return nil
}

// SaveTask 保存研究任务到数据库
func (r *researchRepository) SaveTask(ctx context.Context, task *models.ResearchTask) error {
	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return fmt.Errorf("failed to save research task: %w", err)
	}
	return nil
}

// GetTask 根据 ID 获取研究任务
func (r *researchRepository) GetTask(ctx context.Context, taskID string) (*models.ResearchTask, error) {
	var task models.ResearchTask
	if err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("research task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to get research task: %w", err)
	}
	return &task, nil
}

// GetTasksByResearch 获取研究会话的所有任务（支持分页）
func (r *researchRepository) GetTasksByResearch(ctx context.Context, researchID string, limit, offset int) ([]*models.ResearchTask, error) {
	var tasks []*models.ResearchTask
	query := r.db.WithContext(ctx).
		Where("research_id = ?", researchID).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get research tasks: %w", err)
	}
	return tasks, nil
}

// UpdateTaskStatus 更新研究任务的状态
func (r *researchRepository) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
	result := r.db.WithContext(ctx).
		Model(&models.ResearchTask{}).
		Where("id = ?", taskID).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update task status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("research task not found: %s", taskID)
	}
	return nil
}

// SaveResult 保存研究结果到数据库
func (r *researchRepository) SaveResult(ctx context.Context, result *models.ResearchResult) error {
	if err := r.db.WithContext(ctx).Create(result).Error; err != nil {
		return fmt.Errorf("failed to save research result: %w", err)
	}
	return nil
}

// GetResult 根据研究会话 ID 获取研究结果
func (r *researchRepository) GetResult(ctx context.Context, researchID string) (*models.ResearchResult, error) {
	var result models.ResearchResult
	if err := r.db.WithContext(ctx).Where("research_id = ?", researchID).First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("research result not found for session: %s", researchID)
		}
		return nil, fmt.Errorf("failed to get research result: %w", err)
	}
	return &result, nil
}

// WithTransaction 在数据库事务中执行函数
func (r *researchRepository) WithTransaction(ctx context.Context, fn func(repo ResearchRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &researchRepository{db: tx}
		return fn(txRepo)
	})
}

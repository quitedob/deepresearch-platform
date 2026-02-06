package dao

import (
	"context"

	"github.com/ai-research-platform/internal/pkg/utils"
	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// ResearchDAO 研究数据访问对象
type ResearchDAO struct {
	db *gorm.DB
}

// NewResearchDAO 创建研究DAO
func NewResearchDAO(db *gorm.DB) *ResearchDAO {
    return &ResearchDAO{
        db: db,
    }
}

// CreateSession 创建研究会话
func (r *ResearchDAO) CreateSession(ctx context.Context, session *model.ResearchSession) error {
    return r.db.WithContext(ctx).Create(session).Error
}

// GetSessionByID 根据ID获取研究会话
// 修复：添加ID格式验证防止SQL注入
func (r *ResearchDAO) GetSessionByID(ctx context.Context, sessionID string) (*model.ResearchSession, error) {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return nil, ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	var session model.ResearchSession
	err := r.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListSessionsByUserID 根据用户ID获取研究会话列表
// 修复：添加ID格式验证和分页参数验证
func (r *ResearchDAO) ListSessionsByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.ResearchSession, error) {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(userID); !valid {
		return nil, ErrInvalidID
	} else {
		userID = sanitizedID
	}

	// 验证分页参数
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var sessions []*model.ResearchSession
	query := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset)

	err := query.Find(&sessions).Error
	return sessions, err
}

// UpdateSession 更新研究会话
func (r *ResearchDAO) UpdateSession(ctx context.Context, session *model.ResearchSession) error {
    return r.db.WithContext(ctx).Save(session).Error
}

// DeleteSession 删除研究会话
// 修复：添加ID格式验证
func (r *ResearchDAO) DeleteSession(ctx context.Context, sessionID string) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除相关任务
		if err := tx.Where("research_id = ?", sessionID).Delete(&model.ResearchTask{}).Error; err != nil {
			return err
		}
		// 删除相关结果
		if err := tx.Where("research_id = ?", sessionID).Delete(&model.ResearchResult{}).Error; err != nil {
			return err
		}
		// 删除会话
		return tx.Delete(&model.ResearchSession{}, "id = ?", sessionID).Error
	})
}

// CreateTask 创建研究任务
func (r *ResearchDAO) CreateTask(ctx context.Context, task *model.ResearchTask) error {
    return r.db.WithContext(ctx).Create(task).Error
}

// GetTasksByResearchID 根据研究ID获取任务列表
// 修复：添加ID格式验证
func (r *ResearchDAO) GetTasksByResearchID(ctx context.Context, researchID string) ([]*model.ResearchTask, error) {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(researchID); !valid {
		return nil, ErrInvalidID
	} else {
		researchID = sanitizedID
	}

	var tasks []*model.ResearchTask
	err := r.db.WithContext(ctx).Where("research_id = ?", researchID).
		Order("created_at ASC").Find(&tasks).Error
	return tasks, err
}

// UpdateTask 更新研究任务
func (r *ResearchDAO) UpdateTask(ctx context.Context, task *model.ResearchTask) error {
    return r.db.WithContext(ctx).Save(task).Error
}

// CreateResult 创建研究结果
func (r *ResearchDAO) CreateResult(ctx context.Context, result *model.ResearchResult) error {
    return r.db.WithContext(ctx).Create(result).Error
}

// GetResultByResearchID 根据研究ID获取结果
// 修复：添加ID格式验证
func (r *ResearchDAO) GetResultByResearchID(ctx context.Context, researchID string) (*model.ResearchResult, error) {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(researchID); !valid {
		return nil, ErrInvalidID
	} else {
		researchID = sanitizedID
	}

	var result model.ResearchResult
	err := r.db.WithContext(ctx).Where("research_id = ?", researchID).First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateResult 更新研究结果
func (r *ResearchDAO) UpdateResult(ctx context.Context, result *model.ResearchResult) error {
    return r.db.WithContext(ctx).Save(result).Error
}

// CountSessionsByUserID 统计用户研究会话数量
func (r *ResearchDAO) CountSessionsByUserID(ctx context.Context, userID string) (int64, error) {
    var count int64
    err := r.db.WithContext(ctx).Model(&model.ResearchSession{}).
        Where("user_id = ?", userID).Count(&count).Error
    return count, err
}

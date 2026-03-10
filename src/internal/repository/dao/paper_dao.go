package dao

import (
	"context"

	"github.com/ai-research-platform/internal/pkg/utils"
	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// PaperDAO 论文数据访问对象
type PaperDAO struct {
	db *gorm.DB
}

// NewPaperDAO 创建论文DAO
func NewPaperDAO(db *gorm.DB) *PaperDAO {
	return &PaperDAO{db: db}
}

// ==================== PaperSession ====================

// CreateSession 创建论文会话
func (d *PaperDAO) CreateSession(ctx context.Context, session *model.PaperSession) error {
	return d.db.WithContext(ctx).Create(session).Error
}

// GetSessionByID 根据ID获取论文会话
func (d *PaperDAO) GetSessionByID(ctx context.Context, sessionID string) (*model.PaperSession, error) {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return nil, ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	var session model.PaperSession
	err := d.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListSessionsByUserID 根据用户ID获取论文会话列表
func (d *PaperDAO) ListSessionsByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.PaperSession, error) {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(userID); !valid {
		return nil, ErrInvalidID
	} else {
		userID = sanitizedID
	}

	if limit < 1 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var sessions []*model.PaperSession
	err := d.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&sessions).Error
	return sessions, err
}

// CountSessionsByUserID 统计用户论文会话数量
func (d *PaperDAO) CountSessionsByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.PaperSession{}).
		Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// UpdateSession 更新论文会话
func (d *PaperDAO) UpdateSession(ctx context.Context, session *model.PaperSession) error {
	return d.db.WithContext(ctx).Save(session).Error
}

// UpdateSessionStatus 更新论文会话状态和进度
func (d *PaperDAO) UpdateSessionStatus(ctx context.Context, sessionID string, status string, progress float32) error {
	return d.db.WithContext(ctx).
		Model(&model.PaperSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"status":   status,
			"progress": progress,
		}).Error
}

// UpdateSessionWords 更新论文会话的当前字数
func (d *PaperDAO) UpdateSessionWords(ctx context.Context, sessionID string, currentWords int) error {
	return d.db.WithContext(ctx).
		Model(&model.PaperSession{}).
		Where("id = ?", sessionID).
		Update("current_words", currentWords).Error
}

// DeleteSession 删除论文会话及关联数据
func (d *PaperDAO) DeleteSession(ctx context.Context, sessionID string) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除搜索记录
		if err := tx.Where("paper_id = ?", sessionID).Delete(&model.PaperSearchRecord{}).Error; err != nil {
			return err
		}
		// 删除审查记录
		if err := tx.Where("paper_id = ?", sessionID).Delete(&model.PaperReview{}).Error; err != nil {
			return err
		}
		// 删除引用
		if err := tx.Where("paper_id = ?", sessionID).Delete(&model.PaperCitation{}).Error; err != nil {
			return err
		}
		// 删除章节
		if err := tx.Where("paper_id = ?", sessionID).Delete(&model.PaperChapter{}).Error; err != nil {
			return err
		}
		// 删除会话
		return tx.Delete(&model.PaperSession{}, "id = ?", sessionID).Error
	})
}

// ==================== PaperChapter ====================

// CreateChapter 创建论文章节
func (d *PaperDAO) CreateChapter(ctx context.Context, chapter *model.PaperChapter) error {
	return d.db.WithContext(ctx).Create(chapter).Error
}

// CreateChapters 批量创建论文章节
func (d *PaperDAO) CreateChapters(ctx context.Context, chapters []*model.PaperChapter) error {
	return d.db.WithContext(ctx).Create(&chapters).Error
}

// GetChapterByID 根据ID获取章节
func (d *PaperDAO) GetChapterByID(ctx context.Context, chapterID string) (*model.PaperChapter, error) {
	var chapter model.PaperChapter
	err := d.db.WithContext(ctx).Where("id = ?", chapterID).First(&chapter).Error
	if err != nil {
		return nil, err
	}
	return &chapter, nil
}

// GetChaptersByPaperID 获取论文的所有章节（按排序）
func (d *PaperDAO) GetChaptersByPaperID(ctx context.Context, paperID string) ([]*model.PaperChapter, error) {
	var chapters []*model.PaperChapter
	err := d.db.WithContext(ctx).Where("paper_id = ?", paperID).
		Order("sort_order ASC").
		Find(&chapters).Error
	return chapters, err
}

// UpdateChapter 更新章节
func (d *PaperDAO) UpdateChapter(ctx context.Context, chapter *model.PaperChapter) error {
	return d.db.WithContext(ctx).Save(chapter).Error
}

// UpdateChapterContent 更新章节内容和字数
func (d *PaperDAO) UpdateChapterContent(ctx context.Context, chapterID, content string, wordCount int) error {
	return d.db.WithContext(ctx).
		Model(&model.PaperChapter{}).
		Where("id = ?", chapterID).
		Updates(map[string]interface{}{
			"content":    content,
			"word_count": wordCount,
			"status":     "completed",
		}).Error
}

// UpdateChapterStatus 更新章节状态
func (d *PaperDAO) UpdateChapterStatus(ctx context.Context, chapterID, status string) error {
	return d.db.WithContext(ctx).
		Model(&model.PaperChapter{}).
		Where("id = ?", chapterID).
		Update("status", status).Error
}

// ==================== PaperCitation ====================

// CreateCitation 创建引用
func (d *PaperDAO) CreateCitation(ctx context.Context, citation *model.PaperCitation) error {
	return d.db.WithContext(ctx).Create(citation).Error
}

// CreateCitations 批量创建引用
func (d *PaperDAO) CreateCitations(ctx context.Context, citations []*model.PaperCitation) error {
	if len(citations) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Create(&citations).Error
}

// GetCitationsByPaperID 获取论文的所有引用
func (d *PaperDAO) GetCitationsByPaperID(ctx context.Context, paperID string) ([]*model.PaperCitation, error) {
	var citations []*model.PaperCitation
	err := d.db.WithContext(ctx).Where("paper_id = ?", paperID).
		Order("position ASC").
		Find(&citations).Error
	return citations, err
}

// GetCitationsByChapterID 获取章节的引用
func (d *PaperDAO) GetCitationsByChapterID(ctx context.Context, chapterID string) ([]*model.PaperCitation, error) {
	var citations []*model.PaperCitation
	err := d.db.WithContext(ctx).Where("chapter_id = ?", chapterID).
		Order("position ASC").
		Find(&citations).Error
	return citations, err
}

// CountCitationsByPaperID 统计论文引用数量
func (d *PaperDAO) CountCitationsByPaperID(ctx context.Context, paperID string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.PaperCitation{}).
		Where("paper_id = ?", paperID).Count(&count).Error
	return count, err
}

// ==================== PaperReview ====================

// CreateReview 创建审查记录
func (d *PaperDAO) CreateReview(ctx context.Context, review *model.PaperReview) error {
	return d.db.WithContext(ctx).Create(review).Error
}

// GetReviewsByPaperID 获取论文的审查记录
func (d *PaperDAO) GetReviewsByPaperID(ctx context.Context, paperID string) ([]*model.PaperReview, error) {
	var reviews []*model.PaperReview
	err := d.db.WithContext(ctx).Where("paper_id = ?", paperID).
		Order("review_round ASC, created_at ASC").
		Find(&reviews).Error
	return reviews, err
}

// GetLatestReview 获取论文最新一轮审查
func (d *PaperDAO) GetLatestReview(ctx context.Context, paperID string) (*model.PaperReview, error) {
	var review model.PaperReview
	err := d.db.WithContext(ctx).Where("paper_id = ?", paperID).
		Order("review_round DESC, created_at DESC").
		First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// ==================== PaperSearchRecord ====================

// CreateSearchRecord 创建搜索记录
func (d *PaperDAO) CreateSearchRecord(ctx context.Context, record *model.PaperSearchRecord) error {
	return d.db.WithContext(ctx).Create(record).Error
}

// GetSearchRecordsByPaperID 获取论文的搜索记录
func (d *PaperDAO) GetSearchRecordsByPaperID(ctx context.Context, paperID string) ([]*model.PaperSearchRecord, error) {
	var records []*model.PaperSearchRecord
	err := d.db.WithContext(ctx).Where("paper_id = ?", paperID).
		Order("created_at ASC").
		Find(&records).Error
	return records, err
}

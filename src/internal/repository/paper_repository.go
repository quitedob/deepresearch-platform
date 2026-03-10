package repository

import (
	"context"

	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// PaperRepository 论文数据仓储接口
type PaperRepository interface {
	// Session
	CreateSession(ctx context.Context, session *model.PaperSession) error
	GetSession(ctx context.Context, sessionID string) (*model.PaperSession, error)
	ListSessionsByUser(ctx context.Context, userID string, limit, offset int) ([]*model.PaperSession, error)
	CountSessionsByUser(ctx context.Context, userID string) (int64, error)
	UpdateSession(ctx context.Context, session *model.PaperSession) error
	UpdateSessionStatus(ctx context.Context, sessionID, status string, progress float32) error
	UpdateSessionWords(ctx context.Context, sessionID string, currentWords int) error
	DeleteSession(ctx context.Context, sessionID string) error

	// Chapter
	CreateChapters(ctx context.Context, chapters []*model.PaperChapter) error
	GetChaptersByPaperID(ctx context.Context, paperID string) ([]*model.PaperChapter, error)
	GetChapterByID(ctx context.Context, chapterID string) (*model.PaperChapter, error)
	UpdateChapter(ctx context.Context, chapter *model.PaperChapter) error
	UpdateChapterContent(ctx context.Context, chapterID, content string, wordCount int) error
	UpdateChapterStatus(ctx context.Context, chapterID, status string) error

	// Citation
	CreateCitations(ctx context.Context, citations []*model.PaperCitation) error
	GetCitationsByPaperID(ctx context.Context, paperID string) ([]*model.PaperCitation, error)
	CountCitationsByPaperID(ctx context.Context, paperID string) (int64, error)

	// Review
	CreateReview(ctx context.Context, review *model.PaperReview) error
	GetReviewsByPaperID(ctx context.Context, paperID string) ([]*model.PaperReview, error)

	// Search Record
	CreateSearchRecord(ctx context.Context, record *model.PaperSearchRecord) error
	GetSearchRecordsByPaperID(ctx context.Context, paperID string) ([]*model.PaperSearchRecord, error)
}

// paperRepository 论文数据仓储实现
type paperRepository struct {
	dao *dao.PaperDAO
}

// NewPaperRepository 创建论文Repository
func NewPaperRepository(db *gorm.DB) PaperRepository {
	return &paperRepository{
		dao: dao.NewPaperDAO(db),
	}
}

func (r *paperRepository) CreateSession(ctx context.Context, session *model.PaperSession) error {
	return r.dao.CreateSession(ctx, session)
}

func (r *paperRepository) GetSession(ctx context.Context, sessionID string) (*model.PaperSession, error) {
	return r.dao.GetSessionByID(ctx, sessionID)
}

func (r *paperRepository) ListSessionsByUser(ctx context.Context, userID string, limit, offset int) ([]*model.PaperSession, error) {
	return r.dao.ListSessionsByUserID(ctx, userID, limit, offset)
}

func (r *paperRepository) CountSessionsByUser(ctx context.Context, userID string) (int64, error) {
	return r.dao.CountSessionsByUserID(ctx, userID)
}

func (r *paperRepository) UpdateSession(ctx context.Context, session *model.PaperSession) error {
	return r.dao.UpdateSession(ctx, session)
}

func (r *paperRepository) UpdateSessionStatus(ctx context.Context, sessionID, status string, progress float32) error {
	return r.dao.UpdateSessionStatus(ctx, sessionID, status, progress)
}

func (r *paperRepository) UpdateSessionWords(ctx context.Context, sessionID string, currentWords int) error {
	return r.dao.UpdateSessionWords(ctx, sessionID, currentWords)
}

func (r *paperRepository) DeleteSession(ctx context.Context, sessionID string) error {
	return r.dao.DeleteSession(ctx, sessionID)
}

func (r *paperRepository) CreateChapters(ctx context.Context, chapters []*model.PaperChapter) error {
	return r.dao.CreateChapters(ctx, chapters)
}

func (r *paperRepository) GetChaptersByPaperID(ctx context.Context, paperID string) ([]*model.PaperChapter, error) {
	return r.dao.GetChaptersByPaperID(ctx, paperID)
}

func (r *paperRepository) GetChapterByID(ctx context.Context, chapterID string) (*model.PaperChapter, error) {
	return r.dao.GetChapterByID(ctx, chapterID)
}

func (r *paperRepository) UpdateChapter(ctx context.Context, chapter *model.PaperChapter) error {
	return r.dao.UpdateChapter(ctx, chapter)
}

func (r *paperRepository) UpdateChapterContent(ctx context.Context, chapterID, content string, wordCount int) error {
	return r.dao.UpdateChapterContent(ctx, chapterID, content, wordCount)
}

func (r *paperRepository) UpdateChapterStatus(ctx context.Context, chapterID, status string) error {
	return r.dao.UpdateChapterStatus(ctx, chapterID, status)
}

func (r *paperRepository) CreateCitations(ctx context.Context, citations []*model.PaperCitation) error {
	return r.dao.CreateCitations(ctx, citations)
}

func (r *paperRepository) GetCitationsByPaperID(ctx context.Context, paperID string) ([]*model.PaperCitation, error) {
	return r.dao.GetCitationsByPaperID(ctx, paperID)
}

func (r *paperRepository) CountCitationsByPaperID(ctx context.Context, paperID string) (int64, error) {
	return r.dao.CountCitationsByPaperID(ctx, paperID)
}

func (r *paperRepository) CreateReview(ctx context.Context, review *model.PaperReview) error {
	return r.dao.CreateReview(ctx, review)
}

func (r *paperRepository) GetReviewsByPaperID(ctx context.Context, paperID string) ([]*model.PaperReview, error) {
	return r.dao.GetReviewsByPaperID(ctx, paperID)
}

func (r *paperRepository) CreateSearchRecord(ctx context.Context, record *model.PaperSearchRecord) error {
	return r.dao.CreateSearchRecord(ctx, record)
}

func (r *paperRepository) GetSearchRecordsByPaperID(ctx context.Context, paperID string) ([]*model.PaperSearchRecord, error) {
	return r.dao.GetSearchRecordsByPaperID(ctx, paperID)
}

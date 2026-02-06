package repository

import (
	"context"
	"fmt"

	"github.com/ai-research-platform/internal/models"
	"gorm.io/gorm"
)

// ChatRepository 定义聊天相关数据库操作的接口
type ChatRepository interface {
	// 会话操作
	CreateSession(ctx context.Context, session *models.ChatSession) error
	GetSession(ctx context.Context, sessionID string) (*models.ChatSession, error)
	GetSessionsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.ChatSession, error)
	UpdateSessionMetadata(ctx context.Context, sessionID string, metadata map[string]interface{}) error
	DeleteSession(ctx context.Context, sessionID string) error

	// 消息操作
	SaveMessage(ctx context.Context, message *models.Message) error
	GetMessages(ctx context.Context, sessionID string, limit, offset int) ([]*models.Message, error)
	GetMessageCount(ctx context.Context, sessionID string) (int64, error)
	DeleteMessages(ctx context.Context, sessionID string) error

	// 事务支持
	WithTransaction(ctx context.Context, fn func(repo ChatRepository) error) error
}

// chatRepository 是 ChatRepository 的具体实现
type chatRepository struct {
	db *gorm.DB
}

// NewChatRepository 创建新的 ChatRepository 实例
func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

// CreateSession 在数据库中创建新的聊天会话
func (r *chatRepository) CreateSession(ctx context.Context, session *models.ChatSession) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("failed to create chat session: %w", err)
	}
	return nil
}

// GetSession 根据 ID 获取聊天会话
func (r *chatRepository) GetSession(ctx context.Context, sessionID string) (*models.ChatSession, error) {
	var session models.ChatSession
	if err := r.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("chat session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get chat session: %w", err)
	}
	return &session, nil
}

// GetSessionsByUser 获取特定用户的聊天会话（支持分页）
func (r *chatRepository) GetSessionsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.ChatSession, error) {
	var sessions []*models.ChatSession
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
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	return sessions, nil
}

// UpdateSessionMetadata 更新聊天会话的元数据字段
func (r *chatRepository) UpdateSessionMetadata(ctx context.Context, sessionID string, metadata map[string]interface{}) error {
	result := r.db.WithContext(ctx).
		Model(&models.ChatSession{}).
		Where("id = ?", sessionID).
		Update("metadata", metadata)

	if result.Error != nil {
		return fmt.Errorf("failed to update session metadata: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("chat session not found: %s", sessionID)
	}
	return nil
}

// DeleteSession 软删除聊天会话
func (r *chatRepository) DeleteSession(ctx context.Context, sessionID string) error {
	result := r.db.WithContext(ctx).Delete(&models.ChatSession{}, "id = ?", sessionID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete chat session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("chat session not found: %s", sessionID)
	}
	return nil
}

// SaveMessage 保存消息到数据库并增加会话消息计数
func (r *chatRepository) SaveMessage(ctx context.Context, message *models.Message) error {
	// 使用事务确保消息保存和计数增加是原子操作
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 保存消息
		if err := tx.Create(message).Error; err != nil {
			return fmt.Errorf("failed to save message: %w", err)
		}

		// 增加会话中的消息计数
		if err := tx.Model(&models.ChatSession{}).
			Where("id = ?", message.SessionID).
			UpdateColumn("message_count", gorm.Expr("message_count + ?", 1)).Error; err != nil {
			return fmt.Errorf("failed to update message count: %w", err)
		}

		return nil
	})
}

// GetMessages 获取会话的消息（支持分页，按时间顺序排列）
func (r *chatRepository) GetMessages(ctx context.Context, sessionID string, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	query := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	return messages, nil
}

// GetMessageCount 返回会话中的消息总数
func (r *chatRepository) GetMessageCount(ctx context.Context, sessionID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Message{}).
		Where("session_id = ?", sessionID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}
	return count, nil
}

// DeleteMessages 删除会话的所有消息
func (r *chatRepository) DeleteMessages(ctx context.Context, sessionID string) error {
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&models.Message{}).Error; err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}
	return nil
}

// WithTransaction 在数据库事务中执行函数
func (r *chatRepository) WithTransaction(ctx context.Context, fn func(repo ChatRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &chatRepository{db: tx}
		return fn(txRepo)
	})
}

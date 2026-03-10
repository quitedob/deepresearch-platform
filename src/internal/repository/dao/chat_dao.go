package dao

import (
	"context"
	"errors"
	"time"

	"github.com/ai-research-platform/internal/pkg/utils"
	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// ChatDAO 相关错误
var (
	ErrInvalidID        = errors.New("invalid id format")
	ErrInvalidSortField = errors.New("invalid sort field")
	ErrInvalidSortOrder = errors.New("invalid sort order")
)

// 允许的排序字段白名单
var allowedSortFields = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"title":      "title",
}

// 允许的排序方向白名单
var allowedSortOrders = map[string]string{
	"asc":  "ASC",
	"desc": "DESC",
	"ASC":  "ASC",
	"DESC": "DESC",
}

// ValidateSortParams 验证排序参数
func ValidateSortParams(field, order string) (string, string, error) {
	// 默认值
	if field == "" {
		field = "updated_at"
	}
	if order == "" {
		order = "desc"
	}

	// 验证排序字段
	validField, ok := allowedSortFields[field]
	if !ok {
		return "", "", ErrInvalidSortField
	}

	// 验证排序方向
	validOrder, ok := allowedSortOrders[order]
	if !ok {
		return "", "", ErrInvalidSortOrder
	}

	return validField, validOrder, nil
}

// ChatDAO 聊天数据访问对象
type ChatDAO struct {
	db *gorm.DB
}

// NewChatDAO 创建聊天DAO
func NewChatDAO(db *gorm.DB) *ChatDAO {
	return &ChatDAO{
		db: db,
	}
}

// CreateSession 创建聊天会话
func (c *ChatDAO) CreateSession(ctx context.Context, session *model.ChatSession) error {
	return c.db.WithContext(ctx).Create(session).Error
}

// GetSessionByID 根据ID获取聊天会话
// 修复：添加ID格式验证防止SQL注入
func (c *ChatDAO) GetSessionByID(ctx context.Context, sessionID string) (*model.ChatSession, error) {
	// 验证ID格式
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return nil, ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	var session model.ChatSession
	err := c.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListSessionsByUserID 根据用户ID获取聊天会话列表
// 修复：按updated_at排序，确保最近活跃的会话排在前面；添加ID验证
func (c *ChatDAO) ListSessionsByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.ChatSession, error) {
	return c.ListSessionsByUserIDWithSort(ctx, userID, limit, offset, "updated_at", "desc")
}

// ListSessionsByUserIDWithSort 根据用户ID获取聊天会话列表（支持自定义排序）
// 修复：使用白名单模式验证排序参数，防止SQL注入
func (c *ChatDAO) ListSessionsByUserIDWithSort(ctx context.Context, userID string, limit, offset int, sortField, sortOrder string) ([]*model.ChatSession, error) {
	// 验证用户ID格式
	if sanitizedID, valid := utils.ValidateAndSanitizeID(userID); !valid {
		return nil, ErrInvalidID
	} else {
		userID = sanitizedID
	}

	// 验证分页参数
	if limit < 1 {
		limit = 20
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	// 使用白名单验证排序参数
	validSortField, validSortOrder, err := ValidateSortParams(sortField, sortOrder)
	if err != nil {
		// 使用默认排序
		validSortField = "updated_at"
		validSortOrder = "DESC"
	}

	var sessions []*model.ChatSession
	// 使用验证后的排序参数构建安全的ORDER BY子句
	orderClause := validSortField + " " + validSortOrder + ", created_at DESC"

	query := c.db.WithContext(ctx).Where("user_id = ?", userID).
		Order(orderClause).
		Limit(limit).Offset(offset)

	err = query.Find(&sessions).Error
	return sessions, err
}

// UpdateSession 更新聊天会话
func (c *ChatDAO) UpdateSession(ctx context.Context, session *model.ChatSession) error {
	return c.db.WithContext(ctx).Save(session).Error
}

// DeleteSession 删除聊天会话
// 修复：添加ID验证
func (c *ChatDAO) DeleteSession(ctx context.Context, sessionID string) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return ErrInvalidID
	} else {
		sessionID = sanitizedID
	}
	return c.db.WithContext(ctx).Delete(&model.ChatSession{}, "id = ?", sessionID).Error
}

// CreateMessage 创建消息
func (c *ChatDAO) CreateMessage(ctx context.Context, message *model.Message) error {
	return c.db.WithContext(ctx).Create(message).Error
}

// GetMessagesBySessionID 根据会话ID获取消息列表
// 修复：添加ID验证和分页参数验证
func (c *ChatDAO) GetMessagesBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*model.Message, error) {
	// 验证会话ID格式
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return nil, ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	// 验证分页参数
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var messages []*model.Message
	query := c.db.WithContext(ctx).Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Limit(limit).Offset(offset)

	err := query.Find(&messages).Error
	return messages, err
}

// CountSessionsByUserID 统计用户会话数量
func (c *ChatDAO) CountSessionsByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := c.db.WithContext(ctx).Model(&model.ChatSession{}).
		Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// CountMessagesBySessionID 统计会话消息数量
func (c *ChatDAO) CountMessagesBySessionID(ctx context.Context, sessionID string) (int64, error) {
	var count int64
	err := c.db.WithContext(ctx).Model(&model.Message{}).
		Where("session_id = ?", sessionID).Count(&count).Error
	return count, err
}

// DeleteMessagesBySessionID 删除会话的所有消息
// 修复：添加ID验证
func (c *ChatDAO) DeleteMessagesBySessionID(ctx context.Context, sessionID string) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return ErrInvalidID
	} else {
		sessionID = sanitizedID
	}
	return c.db.WithContext(ctx).Where("session_id = ?", sessionID).Delete(&model.Message{}).Error
}

// GetLastMessageBySessionID 获取会话的最后一条消息
// 修复：添加ID验证；对于空会话返回nil而不是错误
func (c *ChatDAO) GetLastMessageBySessionID(ctx context.Context, sessionID string) (*model.Message, error) {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return nil, ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	var message model.Message
	err := c.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(1).
		First(&message).Error
	if err != nil {
		// 会话没有消息是正常情况，不应该作为错误返回
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

// GetLastMessagesBySessionIDs 批量获取多个会话的最后一条消息（避免N+1查询）
func (c *ChatDAO) GetLastMessagesBySessionIDs(ctx context.Context, sessionIDs []string) (map[string]*model.Message, error) {
	result := make(map[string]*model.Message)
	if len(sessionIDs) == 0 {
		return result, nil
	}

	// 使用 DISTINCT ON (PostgreSQL) 获取每个 session 的最新消息
	var messages []*model.Message
	err := c.db.WithContext(ctx).
		Raw(`SELECT DISTINCT ON (session_id) * FROM messages
			WHERE session_id IN ? AND deleted_at IS NULL
			ORDER BY session_id, created_at DESC`, sessionIDs).
		Scan(&messages).Error
	if err != nil {
		return nil, err
	}

	for _, msg := range messages {
		result[msg.SessionID] = msg
	}
	return result, nil
}

// GetDB 获取数据库连接（用于事务操作）
func (c *ChatDAO) GetDB() *gorm.DB {
	return c.db
}

// CreateSessionWithMessages 创建会话并添加初始消息（事务）
// 修复：使用事务确保会话创建和消息添加的原子性
func (c *ChatDAO) CreateSessionWithMessages(ctx context.Context, session *model.ChatSession, messages []*model.Message) error {
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建会话
		if err := tx.Create(session).Error; err != nil {
			return err
		}

		// 创建消息
		for _, msg := range messages {
			msg.SessionID = session.ID
			if err := tx.Create(msg).Error; err != nil {
				return err
			}
		}

		// 更新消息计数
		session.MessageCount = len(messages)
		return tx.Save(session).Error
	})
}

// DeleteSessionWithMessages 删除会话及其所有消息（事务）
// 修复：使用事务确保级联删除的原子性；添加ID验证
func (c *ChatDAO) DeleteSessionWithMessages(ctx context.Context, sessionID string) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除消息
		if err := tx.Where("session_id = ?", sessionID).Delete(&model.Message{}).Error; err != nil {
			return err
		}
		// 再删除会话
		return tx.Delete(&model.ChatSession{}, "id = ?", sessionID).Error
	})
}

// DeleteOldSessionsByUserID 删除用户的旧会话
// 返回删除的会话数量
func (c *ChatDAO) DeleteOldSessionsByUserID(ctx context.Context, userID string, cutoffTime time.Time) (int, error) {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(userID); !valid {
		return 0, ErrInvalidID
	} else {
		userID = sanitizedID
	}

	// 获取需要删除的会话ID列表
	var sessionIDs []string
	err := c.db.WithContext(ctx).Model(&model.ChatSession{}).
		Where("user_id = ? AND updated_at < ?", userID, cutoffTime).
		Pluck("id", &sessionIDs).Error
	if err != nil {
		return 0, err
	}

	if len(sessionIDs) == 0 {
		return 0, nil
	}

	// 使用事务删除会话和消息
	err = c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除消息
		if err := tx.Where("session_id IN ?", sessionIDs).Delete(&model.Message{}).Error; err != nil {
			return err
		}
		// 删除会话
		return tx.Where("id IN ?", sessionIDs).Delete(&model.ChatSession{}).Error
	})

	if err != nil {
		return 0, err
	}

	return len(sessionIDs), nil
}

// DeleteEmptySessions 删除空会话（没有消息且创建时间超过指定时长）
// 返回删除的会话数量
func (c *ChatDAO) DeleteEmptySessions(ctx context.Context, minAge time.Duration) (int, error) {
	cutoffTime := time.Now().Add(-minAge)

	// 查找空会话
	var sessionIDs []string
	err := c.db.WithContext(ctx).Model(&model.ChatSession{}).
		Where("message_count = 0 AND created_at < ?", cutoffTime).
		Pluck("id", &sessionIDs).Error
	if err != nil {
		return 0, err
	}

	if len(sessionIDs) == 0 {
		return 0, nil
	}

	// 删除空会话
	result := c.db.WithContext(ctx).Where("id IN ?", sessionIDs).Delete(&model.ChatSession{})
	return int(result.RowsAffected), result.Error
}

// DeleteOrphanedMessages 删除孤立消息（会话已删除但消息还在）
// 返回删除的消息数量
func (c *ChatDAO) DeleteOrphanedMessages(ctx context.Context) (int, error) {
	// 使用子查询找出孤立消息
	result := c.db.WithContext(ctx).Exec(`
		DELETE FROM messages
		WHERE session_id NOT IN (SELECT id FROM chat_sessions)
	`)
	return int(result.RowsAffected), result.Error
}

// BatchDeleteSessions 批量删除会话
func (c *ChatDAO) BatchDeleteSessions(ctx context.Context, sessionIDs []string) error {
	if len(sessionIDs) == 0 {
		return nil
	}

	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除消息
		if err := tx.Where("session_id IN ?", sessionIDs).Delete(&model.Message{}).Error; err != nil {
			return err
		}
		// 删除会话
		return tx.Where("id IN ?", sessionIDs).Delete(&model.ChatSession{}).Error
	})
}

// ==================== 乐观锁更新方法 ====================

// ErrOptimisticLock 乐观锁冲突错误
var ErrOptimisticLock = errors.New("optimistic lock conflict: record has been modified")

// UpdateSessionWithVersion 使用乐观锁更新会话
// 如果版本号不匹配，返回 ErrOptimisticLock
func (c *ChatDAO) UpdateSessionWithVersion(ctx context.Context, session *model.ChatSession) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(session.ID); !valid {
		return ErrInvalidID
	} else {
		session.ID = sanitizedID
	}

	currentVersion := session.Version
	session.Version = currentVersion + 1

	result := c.db.WithContext(ctx).Model(&model.ChatSession{}).
		Where("id = ? AND version = ?", session.ID, currentVersion).
		Updates(map[string]interface{}{
			"title":         session.Title,
			"system_prompt": session.SystemPrompt,
			"message_count": session.MessageCount,
			"metadata":      session.Metadata,
			"version":       session.Version,
			"updated_at":    time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrOptimisticLock
	}

	return nil
}

// UpdateMessageWithVersion 使用乐观锁更新消息
// 如果版本号不匹配，返回 ErrOptimisticLock
func (c *ChatDAO) UpdateMessageWithVersion(ctx context.Context, message *model.Message) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(message.ID); !valid {
		return ErrInvalidID
	} else {
		message.ID = sanitizedID
	}

	currentVersion := message.Version
	message.Version = currentVersion + 1

	result := c.db.WithContext(ctx).Model(&model.Message{}).
		Where("id = ? AND version = ?", message.ID, currentVersion).
		Updates(map[string]interface{}{
			"content":    message.Content,
			"metadata":   message.Metadata,
			"version":    message.Version,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrOptimisticLock
	}

	return nil
}

// SoftDeleteMessage 软删除消息
func (c *ChatDAO) SoftDeleteMessage(ctx context.Context, messageID string) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(messageID); !valid {
		return ErrInvalidID
	} else {
		messageID = sanitizedID
	}

	return c.db.WithContext(ctx).Delete(&model.Message{}, "id = ?", messageID).Error
}

// SoftDeleteMessagesBySessionID 软删除会话的所有消息
func (c *ChatDAO) SoftDeleteMessagesBySessionID(ctx context.Context, sessionID string) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	return c.db.WithContext(ctx).Where("session_id = ?", sessionID).Delete(&model.Message{}).Error
}

// GetDeletedMessages 获取已软删除的消息（用于恢复或永久删除）
func (c *ChatDAO) GetDeletedMessages(ctx context.Context, sessionID string, limit int) ([]*model.Message, error) {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(sessionID); !valid {
		return nil, ErrInvalidID
	} else {
		sessionID = sanitizedID
	}

	var messages []*model.Message
	err := c.db.WithContext(ctx).Unscoped().
		Where("session_id = ? AND deleted_at IS NOT NULL", sessionID).
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

// RestoreMessage 恢复软删除的消息
func (c *ChatDAO) RestoreMessage(ctx context.Context, messageID string) error {
	if sanitizedID, valid := utils.ValidateAndSanitizeID(messageID); !valid {
		return ErrInvalidID
	} else {
		messageID = sanitizedID
	}

	return c.db.WithContext(ctx).Unscoped().
		Model(&model.Message{}).
		Where("id = ?", messageID).
		Update("deleted_at", nil).Error
}

// PermanentlyDeleteMessages 永久删除已软删除的消息
func (c *ChatDAO) PermanentlyDeleteMessages(ctx context.Context, olderThan time.Duration) (int, error) {
	cutoffTime := time.Now().Add(-olderThan)

	result := c.db.WithContext(ctx).Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoffTime).
		Delete(&model.Message{})

	return int(result.RowsAffected), result.Error
}

// CountMessagesByUserID 统计用户所有消息数量
func (c *ChatDAO) CountMessagesByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := c.db.WithContext(ctx).Model(&model.Message{}).
		Joins("JOIN chat_sessions ON chat_sessions.id = messages.session_id").
		Where("chat_sessions.user_id = ?", userID).
		Count(&count).Error
	return count, err
}

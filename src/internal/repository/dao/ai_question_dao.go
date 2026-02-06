package dao

import (
	"context"
	"encoding/json"

	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// AIQuestionDAO AI出题数据访问对象
type AIQuestionDAO struct {
	db *gorm.DB
}

// NewAIQuestionDAO 创建AI出题DAO
func NewAIQuestionDAO(db *gorm.DB) *AIQuestionDAO {
	return &AIQuestionDAO{db: db}
}

// ==================== Session Operations ====================

// CreateSession 创建AI出题会话
func (d *AIQuestionDAO) CreateSession(ctx context.Context, session *model.AIQuestionSession) error {
	return d.db.WithContext(ctx).Create(session).Error
}

// GetSessionByID 根据ID获取会话
func (d *AIQuestionDAO) GetSessionByID(ctx context.Context, sessionID string) (*model.AIQuestionSession, error) {
	var session model.AIQuestionSession
	err := d.db.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListSessionsByUserID 获取用户的AI出题会话列表
func (d *AIQuestionDAO) ListSessionsByUserID(ctx context.Context, userID string, limit, offset int) ([]model.AIQuestionSession, error) {
	var sessions []model.AIQuestionSession
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&sessions).Error
	return sessions, err
}

// CountSessionsByUserID 统计用户的会话数量
func (d *AIQuestionDAO) CountSessionsByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.AIQuestionSession{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// UpdateSession 更新会话
func (d *AIQuestionDAO) UpdateSession(ctx context.Context, session *model.AIQuestionSession) error {
	return d.db.WithContext(ctx).Save(session).Error
}

// DeleteSession 删除会话（软删除）
func (d *AIQuestionDAO) DeleteSession(ctx context.Context, sessionID string) error {
	return d.db.WithContext(ctx).Delete(&model.AIQuestionSession{}, "id = ?", sessionID).Error
}

// ==================== Message Operations ====================

// CreateMessage 创建消息
func (d *AIQuestionDAO) CreateMessage(ctx context.Context, message *model.AIQuestionMessage) error {
	return d.db.WithContext(ctx).Create(message).Error
}

// GetMessagesBySessionID 获取会话的消息列表
func (d *AIQuestionDAO) GetMessagesBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]model.AIQuestionMessage, error) {
	var messages []model.AIQuestionMessage
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

// DeleteMessagesBySessionID 删除会话的所有消息
func (d *AIQuestionDAO) DeleteMessagesBySessionID(ctx context.Context, sessionID string) error {
	return d.db.WithContext(ctx).Delete(&model.AIQuestionMessage{}, "session_id = ?", sessionID).Error
}

// ==================== Question Operations ====================

// CreateQuestion 创建生成的题目
func (d *AIQuestionDAO) CreateQuestion(ctx context.Context, question *model.AIGeneratedQuestion) error {
	return d.db.WithContext(ctx).Create(question).Error
}

// CreateQuestions 批量创建题目
func (d *AIQuestionDAO) CreateQuestions(ctx context.Context, questions []model.AIGeneratedQuestion) error {
	if len(questions) == 0 {
		return nil
	}
	return d.db.WithContext(ctx).Create(&questions).Error
}

// GetQuestionsBySessionID 获取会话的题目列表
func (d *AIQuestionDAO) GetQuestionsBySessionID(ctx context.Context, sessionID string) ([]model.AIGeneratedQuestion, error) {
	var questions []model.AIGeneratedQuestion
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&questions).Error
	return questions, err
}

// GetQuestionsByUserID 获取用户的所有题目
func (d *AIQuestionDAO) GetQuestionsByUserID(ctx context.Context, userID string, limit, offset int) ([]model.AIGeneratedQuestion, error) {
	var questions []model.AIGeneratedQuestion
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&questions).Error
	return questions, err
}

// DeleteQuestionsBySessionID 删除会话的所有题目
func (d *AIQuestionDAO) DeleteQuestionsBySessionID(ctx context.Context, sessionID string) error {
	return d.db.WithContext(ctx).Delete(&model.AIGeneratedQuestion{}, "session_id = ?", sessionID).Error
}

// ==================== Config Operations ====================

// GetConfig 获取AI出题配置
func (d *AIQuestionDAO) GetConfig(ctx context.Context) (*model.AIQuestionConfig, error) {
	var config model.AIQuestionConfig
	err := d.db.WithContext(ctx).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		// 返回默认配置
		return &model.AIQuestionConfig{
			DefaultProvider: "deepseek",
			DefaultModel:    "deepseek-chat",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateConfig 更新AI出题配置
func (d *AIQuestionDAO) UpdateConfig(ctx context.Context, provider, modelName, updatedBy string) error {
	var config model.AIQuestionConfig
	err := d.db.WithContext(ctx).First(&config).Error
	
	if err == gorm.ErrRecordNotFound {
		// 创建新配置
		config = model.AIQuestionConfig{
			DefaultProvider: provider,
			DefaultModel:    modelName,
			UpdatedBy:       updatedBy,
		}
		return d.db.WithContext(ctx).Create(&config).Error
	}
	
	if err != nil {
		return err
	}
	
	// 更新现有配置
	config.DefaultProvider = provider
	config.DefaultModel = modelName
	config.UpdatedBy = updatedBy
	return d.db.WithContext(ctx).Save(&config).Error
}

// ==================== Helper Functions ====================

// QuestionToJSON 将题目转换为JSON格式（用于API响应）
func QuestionToJSON(q *model.AIGeneratedQuestion) map[string]interface{} {
	result := map[string]interface{}{
		"id":           q.ID,
		"session_id":   q.SessionID,
		"type":         q.Type,
		"questionText": q.QuestionText,
		"subject":      q.Subject,
		"difficulty":   q.Difficulty,
		"score":        q.Score,
		"explanation":  q.Explanation,
		"created_at":   q.CreatedAt,
	}
	
	// Parse JSON fields
	if q.Tags != nil {
		var tags []string
		json.Unmarshal(q.Tags, &tags)
		result["tags"] = tags
	}
	
	if q.KnowledgePoints != nil {
		var kp []string
		json.Unmarshal(q.KnowledgePoints, &kp)
		result["knowledgePoints"] = kp
	}
	
	if q.Options != nil {
		var options []map[string]string
		json.Unmarshal(q.Options, &options)
		result["options"] = options
	}
	
	if q.CorrectAnswer != nil {
		var answer interface{}
		json.Unmarshal(q.CorrectAnswer, &answer)
		result["correctAnswer"] = answer
	}
	
	return result
}

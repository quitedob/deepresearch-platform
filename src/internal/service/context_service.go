package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
)

// ContextService 上下文管理服务
type ContextService struct {
	chatDAO            *dao.ChatDAO
	userPreferencesDAO *dao.UserPreferencesDAO
	llmScheduler       *eino.LLMScheduler
}

// NewContextService 创建上下文服务
func NewContextService(chatDAO *dao.ChatDAO, prefsDAO *dao.UserPreferencesDAO, scheduler *eino.LLMScheduler) *ContextService {
	return &ContextService{
		chatDAO:            chatDAO,
		userPreferencesDAO: prefsDAO,
		llmScheduler:       scheduler,
	}
}

// ContextResult 上下文构建结果
type ContextResult struct {
	Messages         []*eino.Message // 完整的消息列表
	TotalTokens      int             // 估算的总 token 数
	IsNearLimit      bool            // 是否接近限制
	IsOverLimit      bool            // 是否超过限制
	MaxTokens        int             // 最大 token 限制
	WarningThreshold int             // 警告阈值
}

// BuildContext 构建聊天上下文
// 顺序: 系统安全提示词 → 用户自定义提示词 → 会话系统提示词 → 历史消息 → 当前消息
func (s *ContextService) BuildContext(ctx context.Context, userID string, sessionID string, currentMessage string) (*ContextResult, error) {
	result := &ContextResult{
		Messages:         make([]*eino.Message, 0),
		MaxTokens:        getMaxContextTokens(),
		WarningThreshold: getWarningThreshold(),
	}

	// 1. 获取用户偏好设置
	var prefs *model.UserPreferences
	var memoryEnabled bool = true
	var customPrompt string

	if s.userPreferencesDAO != nil {
		prefs, _ = s.userPreferencesDAO.GetOrCreate(ctx, userID)
		if prefs != nil {
			memoryEnabled = prefs.MemoryEnabled
			customPrompt = prefs.CustomSystemPrompt
			if prefs.MaxContextTokens > 0 {
				result.MaxTokens = prefs.MaxContextTokens
			}
		}
	}

	// 2. 添加系统安全提示词（最高优先级）
	safetyPrompt := getSystemSafetyPrompt()
	if safetyPrompt != "" {
		result.Messages = append(result.Messages, &eino.Message{
			Role:    "system",
			Content: safetyPrompt,
		})
		result.TotalTokens += estimateTokens(safetyPrompt)
	}

	// 3. 添加用户自定义提示词
	if customPrompt != "" {
		result.Messages = append(result.Messages, &eino.Message{
			Role:    "system",
			Content: customPrompt,
		})
		result.TotalTokens += estimateTokens(customPrompt)
	}

	// 4. 获取会话信息和系统提示词
	var session *model.ChatSession
	if s.chatDAO != nil && sessionID != "" {
		session, _ = s.chatDAO.GetSessionByID(ctx, sessionID)
		if session != nil && session.SystemPrompt != "" {
			result.Messages = append(result.Messages, &eino.Message{
				Role:    "system",
				Content: session.SystemPrompt,
			})
			result.TotalTokens += estimateTokens(session.SystemPrompt)
		}
	}

	// 5. 如果启用记忆功能，添加历史消息
	if memoryEnabled && s.chatDAO != nil && sessionID != "" {
		historyMessages, err := s.chatDAO.GetMessagesBySessionID(ctx, sessionID, 1000, 0)
		if err == nil {
			for _, msg := range historyMessages {
				msgTokens := estimateTokens(msg.Content)

				// 检查是否会超过限制
				if result.TotalTokens+msgTokens+estimateTokens(currentMessage) > result.MaxTokens {
					result.IsOverLimit = true
					break
				}

				result.Messages = append(result.Messages, &eino.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
				result.TotalTokens += msgTokens
			}
		}
	}

	// 6. 添加当前用户消息
	currentTokens := estimateTokens(currentMessage)
	result.TotalTokens += currentTokens
	result.Messages = append(result.Messages, &eino.Message{
		Role:    "user",
		Content: currentMessage,
	})

	// 7. 检查是否接近限制
	if result.TotalTokens >= result.WarningThreshold {
		result.IsNearLimit = true
	}
	if result.TotalTokens >= result.MaxTokens {
		result.IsOverLimit = true
	}

	return result, nil
}

// SummarizeAndCreateNewSession 总结当前会话并创建新会话
func (s *ContextService) SummarizeAndCreateNewSession(ctx context.Context, userID string, oldSessionID string) (*model.ChatSession, string, error) {
	if s.chatDAO == nil || s.llmScheduler == nil {
		return nil, "", fmt.Errorf("服务未初始化")
	}

	// 获取旧会话
	oldSession, err := s.chatDAO.GetSessionByID(ctx, oldSessionID)
	if err != nil {
		return nil, "", fmt.Errorf("获取会话失败: %w", err)
	}

	// 获取所有消息
	messages, err := s.chatDAO.GetMessagesBySessionID(ctx, oldSessionID, 1000, 0)
	if err != nil {
		return nil, "", fmt.Errorf("获取消息失败: %w", err)
	}

	// 构建总结请求
	var contentBuilder strings.Builder
	contentBuilder.WriteString("请总结以下对话的主要内容和关键信息，用简洁的语言概括：\n\n")
	for _, msg := range messages {
		contentBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	// 调用 LLM 生成总结
	summaryMessages := []*eino.Message{
		{Role: "system", Content: "你是一个专业的对话总结助手，请用简洁的语言总结对话内容。"},
		{Role: "user", Content: contentBuilder.String()},
	}

	summaryResult, err := s.llmScheduler.ExecuteWithFallback(ctx, summaryMessages, oldSession.Model)
	if err != nil {
		return nil, "", fmt.Errorf("生成总结失败: %w", err)
	}

	summary := summaryResult.Content

	// 创建新会话
	newSession := &model.ChatSession{
		UserID:       userID,
		Title:        "续: " + oldSession.Title,
		Provider:     oldSession.Provider,
		Model:        oldSession.Model,
		ModelType:    oldSession.ModelType,
		SystemPrompt: fmt.Sprintf("这是一个延续的对话。上一个对话的总结：\n%s", summary),
		MessageCount: 0,
	}

	if err := s.chatDAO.CreateSession(ctx, newSession); err != nil {
		return nil, "", fmt.Errorf("创建新会话失败: %w", err)
	}

	return newSession, summary, nil
}

// GetContextStatus 获取当前上下文状态
func (s *ContextService) GetContextStatus(ctx context.Context, userID string, sessionID string) (*ContextStatusResponse, error) {
	result := &ContextStatusResponse{
		MaxTokens:        getMaxContextTokens(),
		WarningThreshold: getWarningThreshold(),
	}

	// 获取用户偏好
	if s.userPreferencesDAO != nil {
		prefs, _ := s.userPreferencesDAO.GetOrCreate(ctx, userID)
		if prefs != nil {
			result.MemoryEnabled = prefs.MemoryEnabled
			if prefs.MaxContextTokens > 0 {
				result.MaxTokens = prefs.MaxContextTokens
			}
		}
	}

	// 计算当前 token 使用量
	if s.chatDAO != nil && sessionID != "" {
		messages, _ := s.chatDAO.GetMessagesBySessionID(ctx, sessionID, 1000, 0)
		for _, msg := range messages {
			result.CurrentTokens += estimateTokens(msg.Content)
		}
		result.MessageCount = len(messages)
	}

	result.UsagePercent = float64(result.CurrentTokens) / float64(result.MaxTokens) * 100
	result.IsNearLimit = result.CurrentTokens >= result.WarningThreshold
	result.IsOverLimit = result.CurrentTokens >= result.MaxTokens

	return result, nil
}

// ContextStatusResponse 上下文状态响应
type ContextStatusResponse struct {
	CurrentTokens    int     `json:"current_tokens"`
	MaxTokens        int     `json:"max_tokens"`
	WarningThreshold int     `json:"warning_threshold"`
	UsagePercent     float64 `json:"usage_percent"`
	MessageCount     int     `json:"message_count"`
	MemoryEnabled    bool    `json:"memory_enabled"`
	IsNearLimit      bool    `json:"is_near_limit"`
	IsOverLimit      bool    `json:"is_over_limit"`
}

// 辅助函数

// getSystemSafetyPrompt 获取系统安全提示词
func getSystemSafetyPrompt() string {
	prompt := os.Getenv("SYSTEM_SAFETY_PROMPT")
	if prompt == "" {
		prompt = "你是一个有帮助的AI助手。请提供准确、有用的信息，并遵守道德和法律规范。"
	}
	return prompt
}

// getMaxContextTokens 获取最大上下文 token 数
func getMaxContextTokens() int {
	maxStr := os.Getenv("MAX_CONTEXT_TOKENS")
	if maxStr == "" {
		return 128000
	}
	max, err := strconv.Atoi(maxStr)
	if err != nil {
		return 128000
	}
	return max
}

// getWarningThreshold 获取警告阈值
func getWarningThreshold() int {
	thresholdStr := os.Getenv("CONTEXT_WARNING_THRESHOLD")
	if thresholdStr == "" {
		return 100000
	}
	threshold, err := strconv.Atoi(thresholdStr)
	if err != nil {
		return 100000
	}
	return threshold
}

// estimateTokens 估算文本的 token 数量
// 简单估算：中文约 1.5 字符/token，英文约 4 字符/token
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	chineseCount := 0
	englishCount := 0

	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			chineseCount++
		} else {
			englishCount++
		}
	}

	// 中文约 1.5 字符/token，英文约 4 字符/token
	return int(float64(chineseCount)/1.5) + int(float64(englishCount)/4)
}

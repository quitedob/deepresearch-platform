package service

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ai-research-platform/internal/cache"
	"github.com/ai-research-platform/internal/pkg/eino"
	"github.com/ai-research-platform/internal/models"
	"github.com/ai-research-platform/internal/repository"
	"github.com/google/uuid"
)

// LRUMessageCache LRU消息缓存
type LRUMessageCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
	mu       sync.RWMutex
	maxSize  int64 // 最大缓存大小（字节）
	curSize  int64 // 当前缓存大小
}

type messageCacheEntry struct {
	key      string
	messages []*models.Message
	size     int64
	expireAt time.Time
}

// NewLRUMessageCache 创建LRU消息缓存
func NewLRUMessageCache(capacity int, maxSizeMB int64) *LRUMessageCache {
	return &LRUMessageCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
		maxSize:  maxSizeMB * 1024 * 1024, // 转换为字节
	}
}

// Get 获取缓存
func (c *LRUMessageCache) Get(key string) ([]*models.Message, bool) {
	c.mu.RLock()
	elem, ok := c.cache[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	entry := elem.Value.(*messageCacheEntry)

	// 检查是否过期
	if time.Now().After(entry.expireAt) {
		c.mu.Lock()
		c.removeElement(elem)
		c.mu.Unlock()
		return nil, false
	}

	// 移动到队首（最近使用）
	c.mu.Lock()
	c.list.MoveToFront(elem)
	c.mu.Unlock()

	return entry.messages, true
}

// Set 设置缓存
func (c *LRUMessageCache) Set(key string, messages []*models.Message, ttl time.Duration) {
	// 计算消息大小
	size := c.calculateSize(messages)

	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果已存在，先删除旧的
	if elem, ok := c.cache[key]; ok {
		c.removeElement(elem)
	}

	// 检查是否需要淘汰
	for c.list.Len() >= c.capacity || (c.maxSize > 0 && c.curSize+size > c.maxSize) {
		if c.list.Len() == 0 {
			break
		}
		// 淘汰最久未使用的
		oldest := c.list.Back()
		if oldest != nil {
			c.removeElement(oldest)
		}
	}

	// 添加新条目
	entry := &messageCacheEntry{
		key:      key,
		messages: messages,
		size:     size,
		expireAt: time.Now().Add(ttl),
	}
	elem := c.list.PushFront(entry)
	c.cache[key] = elem
	c.curSize += size
}

// Delete 删除缓存
func (c *LRUMessageCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.removeElement(elem)
	}
}

// removeElement 移除元素（内部方法，需要持有锁）
func (c *LRUMessageCache) removeElement(elem *list.Element) {
	entry := elem.Value.(*messageCacheEntry)
	delete(c.cache, entry.key)
	c.list.Remove(elem)
	c.curSize -= entry.size
}

// calculateSize 计算消息列表大小
func (c *LRUMessageCache) calculateSize(messages []*models.Message) int64 {
	var size int64
	for _, msg := range messages {
		size += int64(len(msg.Content))
		size += 100 // 估算其他字段大小
	}
	return size
}

// Stats 获取缓存统计
func (c *LRUMessageCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"count":       c.list.Len(),
		"capacity":    c.capacity,
		"current_size": c.curSize,
		"max_size":    c.maxSize,
	}
}

// ChatService handles chat session and message operations
type ChatService struct {
	repo          repository.ChatRepository
	llmScheduler  *eino.LLMScheduler
	cache         cache.Cache
	streamManager *StreamManager
	messageCache  *LRUMessageCache // LRU消息缓存
}

// NewChatService creates a new chat service
func NewChatService(
	repo repository.ChatRepository,
	llmScheduler *eino.LLMScheduler,
	cacheManager cache.Cache,
	streamManager *StreamManager,
) *ChatService {
	return &ChatService{
		repo:          repo,
		llmScheduler:  llmScheduler,
		cache:         cacheManager,
		streamManager: streamManager,
		messageCache:  NewLRUMessageCache(100, 50), // 最多100个会话，最大50MB
	}
}

// GetMessageCacheStats 获取消息缓存统计
func (s *ChatService) GetMessageCacheStats() map[string]interface{} {
	return s.messageCache.Stats()
}

// CreateSession creates a new chat session
func (s *ChatService) CreateSession(ctx context.Context, userID, provider, model string) (*models.ChatSession, error) {
	// Validate that the model is supported
	if !s.llmScheduler.SupportsModel(model) {
		return nil, fmt.Errorf("model not supported: %s", model)
	}

	session := &models.ChatSession{
		ID:           uuid.New().String(),
		UserID:       userID,
		Provider:     provider,
		Model:        model,
		MessageCount: 0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a chat session by ID
func (s *ChatService) GetSession(ctx context.Context, sessionID string) (*models.ChatSession, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("session:%s", sessionID)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		if session, ok := cached.(*models.ChatSession); ok {
			return session, nil
		}
	}

	// Get from database
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Cache the session
	_ = s.cache.Set(ctx, cacheKey, session, 5*time.Minute)

	return session, nil
}

// SendMessage sends a message and gets a response from the LLM
func (s *ChatService) SendMessage(ctx context.Context, sessionID, content string) (*models.Message, error) {
	// Get session
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Save user message
	userMessage := &models.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "user",
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveMessage(ctx, userMessage); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Get message history for context
	messages, err := s.repo.GetMessages(ctx, sessionID, 50, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	// Convert to eino messages
	einoMessages := make([]*eino.Message, 0, len(messages))
	if session.SystemPrompt != "" {
		einoMessages = append(einoMessages, &eino.Message{
			Role:    "system",
			Content: session.SystemPrompt,
		})
	}
	for _, msg := range messages {
		einoMessages = append(einoMessages, &eino.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Get response from LLM with fallback
	response, err := s.llmScheduler.ExecuteWithFallback(ctx, einoMessages, session.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response: %w", err)
	}

	// Save assistant message
	assistantMessage := &models.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "assistant",
		Content:   response.Content,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveMessage(ctx, assistantMessage); err != nil {
		return nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	// Invalidate session cache
	cacheKey := fmt.Sprintf("session:%s", sessionID)
	_ = s.cache.Delete(ctx, cacheKey)

	return assistantMessage, nil
}

// StreamMessage sends a message and streams the response
func (s *ChatService) StreamMessage(ctx context.Context, sessionID, content string) (<-chan StreamChunk, error) {
	// Get session
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Save user message
	userMessage := &models.Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "user",
		Content:   content,
		CreatedAt: time.Now(),
	}

	if err := s.repo.SaveMessage(ctx, userMessage); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Get message history for context
	messages, err := s.repo.GetMessages(ctx, sessionID, 50, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	// Convert to eino messages
	einoMessages := make([]*eino.Message, 0, len(messages))
	if session.SystemPrompt != "" {
		einoMessages = append(einoMessages, &eino.Message{
			Role:    "system",
			Content: session.SystemPrompt,
		})
	}
	for _, msg := range messages {
		einoMessages = append(einoMessages, &eino.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Get streaming response from LLM with fallback
	reader, providerName, err := s.llmScheduler.StreamWithFallback(ctx, einoMessages, session.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get streaming LLM response: %w", err)
	}

	// Create output channel
	outputChan := make(chan StreamChunk, 100)

	// Start goroutine to read stream and accumulate content
	go func() {
		defer close(outputChan)

		fullContent := ""
		for {
			chunk, err := reader.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					outputChan <- StreamChunk{
						Type:  "error",
						Error: err.Error(),
					}
				}
				break
			}

			fullContent += chunk.Content

			outputChan <- StreamChunk{
				Type:    "chunk",
				Content: chunk.Content,
				Metadata: map[string]interface{}{
					"provider": providerName,
				},
			}
		}

		// Save complete assistant message
		assistantMessage := &models.Message{
			ID:        uuid.New().String(),
			SessionID: sessionID,
			Role:      "assistant",
			Content:   fullContent,
			CreatedAt: time.Now(),
		}

		if err := s.repo.SaveMessage(ctx, assistantMessage); err != nil {
			outputChan <- StreamChunk{
				Type:  "error",
				Error: fmt.Sprintf("failed to save message: %v", err),
			}
			return
		}

		// Send completion
		outputChan <- StreamChunk{
			Type: "done",
			Metadata: map[string]interface{}{
				"message_id": assistantMessage.ID,
			},
		}

		// Invalidate session cache
		cacheKey := fmt.Sprintf("session:%s", sessionID)
		_ = s.cache.Delete(ctx, cacheKey)
	}()

	return outputChan, nil
}

// GetSessionHistory retrieves message history for a session
// 修复：使用LRU缓存替代无限缓存，添加缓存大小限制
func (s *ChatService) GetSessionHistory(ctx context.Context, sessionID string, limit, offset int) ([]*models.Message, error) {
	// 只对首页数据使用LRU缓存
	if offset == 0 && limit <= 50 {
		cacheKey := fmt.Sprintf("history:%s:%d", sessionID, limit)

		// 先尝试LRU缓存
		if messages, ok := s.messageCache.Get(cacheKey); ok {
			return messages, nil
		}
	}

	// Get from database
	messages, err := s.repo.GetMessages(ctx, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}

	// 使用LRU缓存存储，自动淘汰旧数据
	if offset == 0 && limit <= 50 {
		cacheKey := fmt.Sprintf("history:%s:%d", sessionID, limit)
		s.messageCache.Set(cacheKey, messages, 2*time.Minute)
	}

	return messages, nil
}

// InvalidateSessionCache 使会话缓存失效
func (s *ChatService) InvalidateSessionCache(sessionID string) {
	// 清除LRU缓存中该会话的所有条目
	for _, limit := range []int{10, 20, 50} {
		cacheKey := fmt.Sprintf("history:%s:%d", sessionID, limit)
		s.messageCache.Delete(cacheKey)
	}

	// 清除通用缓存
	cacheKey := fmt.Sprintf("session:%s", sessionID)
	_ = s.cache.Delete(context.Background(), cacheKey)
}

// UpdateSessionProvider updates the provider and model for a session
func (s *ChatService) UpdateSessionProvider(ctx context.Context, sessionID, provider, model string) error {
	// Validate that the model is supported
	if !s.llmScheduler.SupportsModel(model) {
		return fmt.Errorf("model not supported: %s", model)
	}

	// Get session to update
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Update provider and model
	session.Provider = provider
	session.Model = model
	session.UpdatedAt = time.Now()

	// Update metadata to track provider changes
	metadata := make(map[string]interface{})
	if session.Metadata != nil {
		// Parse existing metadata
		// In production, you'd unmarshal the JSON properly
	}
	metadata["provider_updated_at"] = time.Now().Format(time.RFC3339)
	metadata["previous_provider"] = session.Provider

	if err := s.repo.UpdateSessionMetadata(ctx, sessionID, metadata); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("session:%s", sessionID)
	_ = s.cache.Delete(ctx, cacheKey)

	return nil
}

// GetUserSessions retrieves all sessions for a user
func (s *ChatService) GetUserSessions(ctx context.Context, userID string, limit, offset int) ([]*models.ChatSession, error) {
	return s.repo.GetSessionsByUser(ctx, userID, limit, offset)
}

// DeleteSession deletes a chat session and its messages
func (s *ChatService) DeleteSession(ctx context.Context, sessionID string) error {
	// Delete messages first
	if err := s.repo.DeleteMessages(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Delete session
	if err := s.repo.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("session:%s", sessionID)
	_ = s.cache.Delete(ctx, cacheKey)

	return nil
}

// Package tool 提供工具可靠性包装器
package tool

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ToolCallRecord 工具调用记录（用于复盘与评测）
type ToolCallRecord struct {
	ID           string                 `json:"id"`
	ToolName     string                 `json:"tool_name"`
	Input        map[string]interface{} `json:"input"`
	InputHash    string                 `json:"input_hash"`
	OutputHash   string                 `json:"output_hash"`
	OutputLen    int                    `json:"output_len"`
	Duration     time.Duration          `json:"duration"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	RetryCount   int                    `json:"retry_count"`
	Timestamp    time.Time              `json:"timestamp"`
	ResponseCode int                    `json:"response_code,omitempty"`
}

// ReliabilityConfig 可靠性配置
type ReliabilityConfig struct {
	MaxRetries       int           // 最大重试次数
	RetryDelay       time.Duration // 初始重试延迟
	MaxRetryDelay    time.Duration // 最大重试延迟
	Timeout          time.Duration // 单次调用超时
	EnableDedup      bool          // 启用去重
	DedupTTL         time.Duration // 去重缓存TTL
	MaxDedupEntries  int           // 去重缓存最大条目数（防止内存无限增长）
	RateLimitPerMin  int           // 每分钟限流
	RecordCallback   func(*ToolCallRecord) // 记录回调（用于入库）
}

// DefaultReliabilityConfig 默认可靠性配置
func DefaultReliabilityConfig() ReliabilityConfig {
	return ReliabilityConfig{
		MaxRetries:      3,
		RetryDelay:      500 * time.Millisecond,
		MaxRetryDelay:   5 * time.Second,
		Timeout:         30 * time.Second,
		EnableDedup:     true,
		DedupTTL:        5 * time.Minute,
		MaxDedupEntries: 10000, // 修复：默认最多10000条缓存
		RateLimitPerMin: 30,
	}
}

// ReliableTool 可靠性包装的工具
type ReliableTool struct {
	inner        tool.InvokableTool
	config       ReliabilityConfig
	dedupCache   sync.Map // map[string]*dedupEntry
	dedupCount   int64    // 当前缓存条目数（近似值）
	rateLimiter  *rateLimiter
	mu           sync.Mutex
	cleanerOnce  sync.Once // 确保清理协程只启动一次
	stopCleaner  context.CancelFunc // 用于停止后台清理协程
}

type dedupEntry struct {
	result    string
	timestamp time.Time
}

type rateLimiter struct {
	tokens    int
	maxTokens int
	lastReset time.Time
	mu        sync.Mutex
}

func newRateLimiter(perMin int) *rateLimiter {
	return &rateLimiter{
		tokens:    perMin,
		maxTokens: perMin,
		lastReset: time.Now(),
	}
}

func (r *rateLimiter) allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.Sub(r.lastReset) >= time.Minute {
		r.tokens = r.maxTokens
		r.lastReset = now
	}

	if r.tokens > 0 {
		r.tokens--
		return true
	}
	return false
}

// NewReliableTool 创建可靠性包装的工具
func NewReliableTool(inner tool.InvokableTool, config ReliabilityConfig) *ReliableTool {
	if config.MaxDedupEntries <= 0 {
		config.MaxDedupEntries = 10000
	}
	t := &ReliableTool{
		inner:       inner,
		config:      config,
		rateLimiter: newRateLimiter(config.RateLimitPerMin),
	}
	// 启动后台清理协程（延迟到第一次调用时）
	return t
}

// Info 返回工具信息
func (t *ReliableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return t.inner.Info(ctx)
}

// InvokableRun 执行工具调用（带可靠性保障）
func (t *ReliableTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	startTime := time.Now()
	info, _ := t.inner.Info(ctx)
	toolName := "unknown"
	if info != nil {
		toolName = info.Name
	}

	record := &ToolCallRecord{
		ID:        generateRecordID(),
		ToolName:  toolName,
		Timestamp: startTime,
	}

	// 解析输入参数
	var inputMap map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &inputMap); err == nil {
		record.Input = inputMap
		record.InputHash = hashString(argumentsInJSON)
	}

	// 1. 去重检查
	if t.config.EnableDedup {
		if cached := t.checkDedup(record.InputHash); cached != "" {
			record.Success = true
			record.Duration = time.Since(startTime)
			record.OutputHash = hashString(cached)
			record.OutputLen = len(cached)
			t.recordCall(record)
			return cached, nil
		}
	}

	// 2. 限流检查
	if !t.rateLimiter.allow() {
		record.Success = false
		record.Error = "rate limit exceeded"
		record.Duration = time.Since(startTime)
		t.recordCall(record)
		return "", fmt.Errorf("rate limit exceeded for tool %s", toolName)
	}

	// 3. 带重试的执行
	var result string
	var lastErr error

	for attempt := 0; attempt <= t.config.MaxRetries; attempt++ {
		record.RetryCount = attempt

		// 创建带超时的上下文
		callCtx, cancel := context.WithTimeout(ctx, t.config.Timeout)
		
		result, lastErr = t.inner.InvokableRun(callCtx, argumentsInJSON, opts...)
		cancel()

		if lastErr == nil && result != "" {
			// 成功
			record.Success = true
			record.Duration = time.Since(startTime)
			record.OutputHash = hashString(result)
			record.OutputLen = len(result)

			// 缓存结果用于去重
			if t.config.EnableDedup {
				t.cacheResult(record.InputHash, result)
			}

			t.recordCall(record)
			return result, nil
		}

		// 失败，准备重试
		if attempt < t.config.MaxRetries {
			delay := t.calculateRetryDelay(attempt)
			select {
			case <-ctx.Done():
				record.Success = false
				record.Error = "context cancelled"
				record.Duration = time.Since(startTime)
				t.recordCall(record)
				return "", ctx.Err()
			case <-time.After(delay):
				// 继续重试
			}
		}
	}

	// 所有重试都失败
	record.Success = false
	if lastErr != nil {
		record.Error = lastErr.Error()
	} else {
		record.Error = "empty result after retries"
	}
	record.Duration = time.Since(startTime)
	t.recordCall(record)

	if lastErr != nil {
		return "", lastErr
	}
	return result, nil
}

// calculateRetryDelay 计算重试延迟（指数退避）
func (t *ReliableTool) calculateRetryDelay(attempt int) time.Duration {
	delay := t.config.RetryDelay
	for i := 0; i < attempt; i++ {
		delay *= 2
	}
	if delay > t.config.MaxRetryDelay {
		delay = t.config.MaxRetryDelay
	}
	return delay
}

// checkDedup 检查去重缓存
func (t *ReliableTool) checkDedup(inputHash string) string {
	if entry, ok := t.dedupCache.Load(inputHash); ok {
		e := entry.(*dedupEntry)
		if time.Since(e.timestamp) < t.config.DedupTTL {
			return e.result
		}
		t.dedupCache.Delete(inputHash)
	}
	return ""
}

// cacheResult 缓存结果
// 修复：添加容量检查，防止无限增长
func (t *ReliableTool) cacheResult(inputHash, result string) {
	// 启动后台清理协程（只执行一次）
	t.cleanerOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		t.stopCleaner = cancel
		go t.startDedupCleaner(ctx)
	})

	// 检查容量，超出时先清理过期条目
	if t.dedupCount >= int64(t.config.MaxDedupEntries) {
		t.evictExpired()
	}

	// 如果清理后仍然超过容量，跳过缓存（降级，不阻塞）
	if t.dedupCount >= int64(t.config.MaxDedupEntries) {
		return
	}

	t.dedupCache.Store(inputHash, &dedupEntry{
		result:    result,
		timestamp: time.Now(),
	})
	t.mu.Lock()
	t.dedupCount++
	t.mu.Unlock()
}

// startDedupCleaner 后台定期清理过期缓存（可通过 Close 停止）
func (t *ReliableTool) startDedupCleaner(ctx context.Context) {
	ticker := time.NewTicker(t.config.DedupTTL)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.evictExpired()
		case <-ctx.Done():
			return
		}
	}
}

// Close 停止后台清理协程，释放资源
func (t *ReliableTool) Close() error {
	if t.stopCleaner != nil {
		t.stopCleaner()
	}
	return nil
}

// evictExpired 清理过期缓存条目
func (t *ReliableTool) evictExpired() {
	var evicted int64
	now := time.Now()
	t.dedupCache.Range(func(key, value interface{}) bool {
		entry := value.(*dedupEntry)
		if now.Sub(entry.timestamp) >= t.config.DedupTTL {
			t.dedupCache.Delete(key)
			evicted++
		}
		return true
	})
	if evicted > 0 {
		t.mu.Lock()
		t.dedupCount -= evicted
		if t.dedupCount < 0 {
			t.dedupCount = 0
		}
		t.mu.Unlock()
	}
}

// recordCall 记录调用
func (t *ReliableTool) recordCall(record *ToolCallRecord) {
	if t.config.RecordCallback != nil {
		t.config.RecordCallback(record)
	}
}

// generateRecordID 生成记录ID
func generateRecordID() string {
	return fmt.Sprintf("tc_%d", time.Now().UnixNano())
}

// hashString 计算字符串哈希
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8]) // 只取前8字节
}

// 确保实现 InvokableTool 接口
var _ tool.InvokableTool = (*ReliableTool)(nil)

// ToolCallRecorder 工具调用记录器
type ToolCallRecorder struct {
	records []ToolCallRecord
	mu      sync.RWMutex
	maxSize int
}

// NewToolCallRecorder 创建记录器
func NewToolCallRecorder(maxSize int) *ToolCallRecorder {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &ToolCallRecorder{
		records: make([]ToolCallRecord, 0, maxSize),
		maxSize: maxSize,
	}
}

// Record 记录调用
func (r *ToolCallRecorder) Record(record *ToolCallRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.records) >= r.maxSize {
		// 移除最旧的记录
		r.records = r.records[1:]
	}
	r.records = append(r.records, *record)
}

// GetRecords 获取所有记录
func (r *ToolCallRecorder) GetRecords() []ToolCallRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ToolCallRecord, len(r.records))
	copy(result, r.records)
	return result
}

// GetStats 获取统计信息
func (r *ToolCallRecorder) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]interface{})
	toolStats := make(map[string]map[string]interface{})

	for _, rec := range r.records {
		if _, ok := toolStats[rec.ToolName]; !ok {
			toolStats[rec.ToolName] = map[string]interface{}{
				"total":      0,
				"success":    0,
				"failed":     0,
				"avg_duration_ms": float64(0),
				"total_retries": 0,
			}
		}
		ts := toolStats[rec.ToolName]
		ts["total"] = ts["total"].(int) + 1
		if rec.Success {
			ts["success"] = ts["success"].(int) + 1
		} else {
			ts["failed"] = ts["failed"].(int) + 1
		}
		ts["total_retries"] = ts["total_retries"].(int) + rec.RetryCount
		
		// 更新平均耗时
		total := ts["total"].(int)
		oldAvg := ts["avg_duration_ms"].(float64)
		newAvg := oldAvg + (float64(rec.Duration.Milliseconds())-oldAvg)/float64(total)
		ts["avg_duration_ms"] = newAvg
	}

	stats["by_tool"] = toolStats
	stats["total_calls"] = len(r.records)
	return stats
}

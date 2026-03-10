// Package eino 提供基于 CloudWeGo Eino AI 研究平台核心功能
package eino

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"

	"github.com/ai-research-platform/internal/pkg/eino/agent"
	einomodel "github.com/ai-research-platform/internal/pkg/eino/model"
	einotool "github.com/ai-research-platform/internal/pkg/eino/tool"
)

// 重新导出类型

// ChatModel Eino 官方 ChatModel 接口
type ChatModel = model.ChatModel

// InvokableTool Eino 官方 InvokableTool 接口
type InvokableTool = tool.InvokableTool

// ResearchAgent 研究智能
type ResearchAgent = agent.ResearchAgent

// AgentConfig Agent 配置
type AgentConfig = agent.Config

// AgentResult Agent 结果
type AgentResult = agent.Result

// ProgressEvent 进度事件
type ProgressEvent = agent.ProgressEvent

// ProgressCallback 进度回调
type ProgressCallback = agent.ProgressCallback

// StructuredReport 结构化报告
type StructuredReport = agent.StructuredReport

// CriticResult 反证检查结果
type CriticResult = agent.CriticResult

// EnhancedPlan 增强版研究计划
type EnhancedPlan = agent.EnhancedPlan

// ToolCallRecord 工具调用记录
type ToolCallRecord = einotool.ToolCallRecord

// ModelConfig 模型配置
type ModelConfig = einomodel.Config

// ParallelOrchestrator 并行研究编排器
type ParallelOrchestrator = agent.ParallelOrchestrator

// SubAgentTask 子Agent任务
type SubAgentTask = agent.SubAgentTask

// SubAgentResult 子Agent结果
type SubAgentResult = agent.SubAgentResult

// ToolsConfig 工具配置
type ToolsConfig struct {
	WebSearchAPIKey    string
	ArxivMaxResults    int
	WikipediaLanguage  string
	Timeout            time.Duration
	EnableReliability  bool                              // 启用可靠性包装
	RecordCallback     func(*einotool.ToolCallRecord)    // 工具调用记录回调
	EnableZRead        bool                              // 启用 ZRead MCP（开源仓库读取）
	EnableWebReader    bool                              // 启用 Web Reader MCP（网页读取）
	EnableSearchPrime  bool                              // 启用 Web Search Prime MCP（增强搜索）
}

// DefaultToolsConfig 返回默认工具配置
// 注意：WebSearchAPIKey 需要从外部传入，这里返回空
func DefaultToolsConfig() ToolsConfig {
	return ToolsConfig{
		WebSearchAPIKey:    "", // 需要从外部传入
		ArxivMaxResults:    10,
		WikipediaLanguage:  "zh",
		Timeout:            90 * time.Second, // 从30秒提升到90秒，避免联网搜索+LLM生成链路超时
		EnableReliability:  true,  // 默认启用可靠性
		EnableZRead:        true,  // 默认启用 ZRead MCP
		EnableWebReader:    true,  // 默认启用 Web Reader MCP
		EnableSearchPrime:  true,  // 默认启用 Web Search Prime MCP
	}
}

// NewResearchAgent 创建研究 Agent
func NewResearchAgent(chatModel ChatModel, tools []InvokableTool, config AgentConfig) *ResearchAgent {
	return agent.NewResearchAgent(chatModel, tools, config)
}

// NewResearchAgentWithFeatures 创建带完整功能的研究 Agent
func NewResearchAgentWithFeatures(chatModel ChatModel, tools []InvokableTool, config AgentConfig) *ResearchAgent {
	// 启用所有新功能
	config.EnableCritic = true
	config.EnableStructured = true
	return agent.NewResearchAgent(chatModel, tools, config)
}

// NewParallelOrchestrator 创建并行研究编排器
func NewParallelOrchestrator(chatModel ChatModel, tools []InvokableTool, config AgentConfig) *ParallelOrchestrator {
	return agent.NewParallelOrchestrator(chatModel, tools, config)
}

// CreateResearchTools 创建研究工具
func CreateResearchTools(config ToolsConfig) []InvokableTool {
	tools := make([]InvokableTool, 0, 6)

	// Web Search
	webSearch := einotool.NewWebSearchTool(einotool.WebSearchConfig{
		APIKey:  config.WebSearchAPIKey,
		Timeout: config.Timeout,
	})

	// ArXiv
	arxiv := einotool.NewArxivTool(einotool.ArxivConfig{
		MaxResults: config.ArxivMaxResults,
		Timeout:    config.Timeout,
	})

	// Wikipedia
	wikipedia := einotool.NewWikipediaTool(einotool.WikipediaConfig{
		Language: config.WikipediaLanguage,
		Timeout:  config.Timeout,
	})

	// 可靠性配置
	var reliabilityConfig einotool.ReliabilityConfig
	if config.EnableReliability {
		reliabilityConfig = einotool.DefaultReliabilityConfig()
		if config.RecordCallback != nil {
			reliabilityConfig.RecordCallback = config.RecordCallback
		}
	}

	// 添加基础工具
	if config.EnableReliability {
		tools = append(tools, einotool.NewReliableTool(webSearch, reliabilityConfig))
		tools = append(tools, einotool.NewReliableTool(arxiv, reliabilityConfig))
		tools = append(tools, einotool.NewReliableTool(wikipedia, reliabilityConfig))
	} else {
		tools = append(tools, webSearch)
		tools = append(tools, arxiv)
		tools = append(tools, wikipedia)
	}

	// ZRead MCP - 开源仓库读取
	if config.EnableZRead && config.WebSearchAPIKey != "" {
		zread := einotool.NewZReadTool(einotool.ZReadConfig{
			APIKey:  config.WebSearchAPIKey,
			Timeout: config.Timeout,
		})
		if config.EnableReliability {
			tools = append(tools, einotool.NewReliableTool(zread, reliabilityConfig))
		} else {
			tools = append(tools, zread)
		}
	}

	// Web Reader MCP - 网页读取
	if config.EnableWebReader && config.WebSearchAPIKey != "" {
		webReader := einotool.NewWebReaderTool(einotool.WebReaderConfig{
			APIKey:  config.WebSearchAPIKey,
			Timeout: config.Timeout,
		})
		if config.EnableReliability {
			tools = append(tools, einotool.NewReliableTool(webReader, reliabilityConfig))
		} else {
			tools = append(tools, webReader)
		}
	}

	// Web Search Prime MCP - 增强搜索
	if config.EnableSearchPrime && config.WebSearchAPIKey != "" {
		searchPrime := einotool.NewWebSearchPrimeTool(einotool.WebSearchPrimeConfig{
			APIKey:  config.WebSearchAPIKey,
			Timeout: config.Timeout,
		})
		if config.EnableReliability {
			tools = append(tools, einotool.NewReliableTool(searchPrime, reliabilityConfig))
		} else {
			tools = append(tools, searchPrime)
		}
	}

	return tools
}

// CreateWebSearchTool 创建单独的网络搜索工具（供联网搜索功能使用）
// apiKey 必须传入，不能为空
func CreateWebSearchTool(apiKey string) InvokableTool {
	return einotool.NewWebSearchTool(einotool.WebSearchConfig{
		APIKey:  apiKey,
		Timeout: 90 * time.Second,
	})
}

// CreateZReadTool 创建 ZRead MCP 工具（开源仓库读取）
// apiKey 必须传入，不能为空
func CreateZReadTool(apiKey string) InvokableTool {
	return einotool.NewZReadTool(einotool.ZReadConfig{
		APIKey:  apiKey,
		Timeout: 30 * time.Second,
	})
}

// CreateWebReaderTool 创建 Web Reader MCP 工具（网页读取）
// apiKey 必须传入，不能为空
func CreateWebReaderTool(apiKey string) InvokableTool {
	return einotool.NewWebReaderTool(einotool.WebReaderConfig{
		APIKey:  apiKey,
		Timeout: 30 * time.Second,
	})
}

// GetToolInfos 获取工具信息列表
func GetToolInfos(tools []InvokableTool) []map[string]string {
	infos := make([]map[string]string, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			continue
		}
		infos = append(infos, map[string]string{
			"name": info.Name,
			"desc": info.Desc,
		})
	}
	return infos
}

// Model 相关便捷函数

// SupportedProviders 返回支持Provider 列表
func SupportedProviders() []string {
	return einomodel.SupportedProviders()
}

// ProviderModels 返回 Provider 支持的模型列
func ProviderModels(providerType string) []string {
	return einomodel.ProviderModels(providerType)
}

// DefaultModelConfig 返回默认模型配置
func DefaultModelConfig(providerType string) ModelConfig {
	return einomodel.DefaultConfig(providerType)
}

// NewUserMessage 创建用户消息
func NewUserMessage(content string) *einomodel.Message {
	return einomodel.NewUserMessage(content)
}

// NewSystemMessage 创建系统消息
func NewSystemMessage(content string) *einomodel.Message {
	return einomodel.NewSystemMessage(content)
}

// NewAssistantMessage 创建助手消息
func NewAssistantMessage(content string) *einomodel.Message {
	return einomodel.NewAssistantMessage(content)
}


// Message 消息类型
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMScheduler LLM调度器，支持多Provider和故障转移
type LLMScheduler struct {
	providers map[string]ChatModel
	models    map[string]string // model -> provider mapping
}

// NewLLMScheduler 创建LLM调度器
func NewLLMScheduler() *LLMScheduler {
	return &LLMScheduler{
		providers: make(map[string]ChatModel),
		models:    make(map[string]string),
	}
}

// RegisterProvider 注册Provider
func (s *LLMScheduler) RegisterProvider(name string, chatModel ChatModel, models []string) {
	s.providers[name] = chatModel
	for _, m := range models {
		s.models[m] = name
	}
}

// SupportsModel 检查是否支持指定模型
func (s *LLMScheduler) SupportsModel(model string) bool {
	_, ok := s.models[model]
	return ok
}

// GetRegisteredModels 获取所有已注册的模型列表
func (s *LLMScheduler) GetRegisteredModels() map[string]string {
	result := make(map[string]string)
	for model, provider := range s.models {
		result[model] = provider
	}
	return result
}

// GetRegisteredProviders 获取所有已注册的Provider列表
func (s *LLMScheduler) GetRegisteredProviders() []string {
	providers := make([]string, 0, len(s.providers))
	seen := make(map[string]bool)
	for _, provider := range s.models {
		if !seen[provider] {
			providers = append(providers, provider)
			seen[provider] = true
		}
	}
	return providers
}

// ExecuteWithFallback 执行LLM调用，支持故障转移
func (s *LLMScheduler) ExecuteWithFallback(ctx context.Context, messages []*Message, model string) (*Message, error) {
	providerName, ok := s.models[model]
	if !ok {
		return nil, fmt.Errorf("model not supported: %s", model)
	}

	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	// Convert messages to schema.Message
	schemaMessages := make([]*einomodel.Message, len(messages))
	for i, msg := range messages {
		schemaMessages[i] = einomodel.NewMessage(msg.Role, msg.Content)
	}

	response, err := provider.Generate(ctx, schemaMessages)
	if err != nil {
		return nil, err
	}

	return &Message{
		Role:    string(response.Role),
		Content: response.Content,
	}, nil
}

// ExecuteWithJSONMode 执行LLM调用，通过prompt强调JSON输出
// 注意：eino库不直接支持response_format，通过prompt工程实现
func (s *LLMScheduler) ExecuteWithJSONMode(ctx context.Context, messages []*Message, modelName string) (*Message, error) {
	// 直接调用普通模式，JSON格式通过system prompt控制
	return s.ExecuteWithFallback(ctx, messages, modelName)
}

// StreamResponse 流式响应
type StreamResponse struct {
	Content string
}

// StreamReader 流式读取器
type StreamReader struct {
	ch     <-chan *StreamResponse
	closed bool
}

// Recv 接收下一个chunk
func (r *StreamReader) Recv() (*StreamResponse, error) {
	chunk, ok := <-r.ch
	if !ok {
		return nil, fmt.Errorf("EOF")
	}
	return chunk, nil
}

// StreamWithFallback 流式执行LLM调用
func (s *LLMScheduler) StreamWithFallback(ctx context.Context, messages []*Message, model string) (*StreamReader, string, error) {
	providerName, ok := s.models[model]
	if !ok {
		return nil, "", fmt.Errorf("model not supported: %s", model)
	}

	provider, ok := s.providers[providerName]
	if !ok {
		return nil, "", fmt.Errorf("provider not found: %s", providerName)
	}

	// Convert messages to schema.Message
	schemaMessages := make([]*einomodel.Message, len(messages))
	for i, msg := range messages {
		schemaMessages[i] = einomodel.NewMessage(msg.Role, msg.Content)
	}

	streamReader, err := provider.Stream(ctx, schemaMessages)
	if err != nil {
		return nil, "", err
	}

	// Create output channel with buffer
	ch := make(chan *StreamResponse, 100)

	go func() {
		defer close(ch)
		for {
			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				return
			default:
			}
			
			msg, err := streamReader.Recv()
			if err != nil {
				// EOF 或其他错误都会导致退出
				return
			}
			
			// 尝试发送，如果通道已满或上下文取消则退出
			select {
			case ch <- &StreamResponse{Content: msg.Content}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return &StreamReader{ch: ch}, providerName, nil
}

package llm

import (
    "context"
    "github.com/ai-research-platform/internal/types/request"
    "github.com/ai-research-platform/internal/types/response"
)

// Service LLM服务接口
type Service interface {
    // ListProviders 获取LLM提供商列表
    ListProviders(ctx context.Context) (*response.ListProvidersResponse, error)

    // TestProvider 测试LLM提供商
    TestProvider(ctx context.Context, req request.TestProviderRequest) (*response.TestProviderResponse, error)

    // GetMetrics 获取LLM指标
    GetMetrics(ctx context.Context) (*response.LLMMetricsResponse, error)

    // GenerateText 生成文本
    GenerateText(ctx context.Context, req request.GenerateTextRequest) (*response.GenerateTextResponse, error)

    // StreamGenerateText 流式生成文本
    StreamGenerateText(ctx context.Context, req request.GenerateTextRequest) (<-chan *response.StreamChunk, error)
}

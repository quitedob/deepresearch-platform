package research

import (
    "context"
    "github.com/ai-research-platform/internal/types/request"
    "github.com/ai-research-platform/internal/types/response"
)

// Service 研究服务接口
type Service interface {
    // CreateSession 创建研究会话
    CreateSession(ctx context.Context, userID string, req request.CreateResearchSessionRequest) (*response.ResearchSessionResponse, error)

    // ListSessions 获取研究会话列表
    ListSessions(ctx context.Context, userID string, req request.ListResearchSessionsRequest) (*response.ListResearchSessionsResponse, error)

    // GetSession 获取研究会话详情
    GetSession(ctx context.Context, userID, sessionID string) (*response.ResearchSessionResponse, error)

    // StartResearch 开始研究
    StartResearch(ctx context.Context, userID, sessionID string, req request.StartResearchRequest) (*response.ResearchResponse, error)

    // StreamProgress 流式获取研究进度
    StreamProgress(ctx context.Context, userID, sessionID string) (<-chan *response.ResearchProgressEvent, error)
}

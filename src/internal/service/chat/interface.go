package chat

import (
    "context"
    "github.com/ai-research-platform/internal/types/request"
    "github.com/ai-research-platform/internal/types/response"
    "github.com/ai-research-platform/internal/repository/model"
)

// Service 聊天服务接口
type Service interface {
    // CreateSession 创建聊天会话
    CreateSession(ctx context.Context, userID string, req request.CreateChatSessionRequest) (*response.ChatSessionResponse, error)

    // ListSessions 获取聊天会话列表
    ListSessions(ctx context.Context, userID string, req request.ListChatSessionsRequest) (*response.ListChatSessionsResponse, error)

    // GetSession 获取聊天会话详情
    GetSession(ctx context.Context, userID, sessionID string) (*response.ChatSessionResponse, error)

    // DeleteSession 删除聊天会话
    DeleteSession(ctx context.Context, userID, sessionID string) error

    // SendMessage 发送消息
    SendMessage(ctx context.Context, userID, sessionID string, req request.SendMessageRequest) (*response.MessageResponse, error)

    // GetMessages 获取消息列表
    GetMessages(ctx context.Context, userID, sessionID string, req request.GetMessagesRequest) (*response.GetMessagesResponse, error)
}

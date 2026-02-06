package response

import (
	"encoding/json"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
)

// ChatSessionResponse 聊天会话响应
type ChatSessionResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Title        string    `json:"title"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	ModelType    string    `json:"model_type,omitempty"`    // default, deep, research
	SystemPrompt string    `json:"system_prompt,omitempty"`
	MessageCount int       `json:"message_count"`
	LastMessage  string    `json:"last_message,omitempty"`  // 最后一条消息预览
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ChatMessageResponse 聊天消息响应
type ChatMessageResponse struct {
	ID         string                 `json:"id"`
	SessionID  string                 `json:"session_id"`
	Role       string                 `json:"role"` // user, assistant, system
	Content    string                 `json:"content"`
	TokensUsed int                    `json:"tokens_used"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	SessionID    string                 `json:"session_id"`
	MessageID    string                 `json:"message_id"`
	Content      string                 `json:"content"`
	Role         string                 `json:"role"`
	TokensUsed   int                    `json:"tokens_used"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Stream       bool                   `json:"stream"`
	ResponseTime int64                  `json:"response_time,omitempty"` // 毫秒
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Provider     string   `json:"provider"`
	Description  string   `json:"description"`
	ContextLen   int      `json:"context_length"`
	MaxTokens    int      `json:"max_tokens"`
	Capabilities []string `json:"capabilities"` // streaming, tools, images, etc.
	Pricing      *Pricing `json:"pricing,omitempty"`
}

// Pricing 定价信息
type Pricing struct {
	InputPricePer1K  float64 `json:"input_price_per_1k"`
	OutputPricePer1K float64 `json:"output_price_per_1k"`
	Currency         string  `json:"currency"`
}

// ModelListResponse 模型列表响应
type ModelListResponse struct {
	Models []ModelInfo `json:"models"`
	Total  int         `json:"total"`
}

// ChatSessionsListResponse 聊天会话列表响应
// 修复：添加hasMore和maxLimit字段，统一分页响应格式
type ChatSessionsListResponse struct {
	Sessions []*ChatSessionResponse `json:"sessions"`
	Total    int                    `json:"total"`
	Limit    int                    `json:"limit"`
	Offset   int                    `json:"offset"`
	HasMore  bool                   `json:"has_more"`
	MaxLimit int                    `json:"max_limit,omitempty"`
}

// ChatMessagesListResponse 聊天消息列表响应
// 修复：添加hasMore和maxLimit字段，统一分页响应格式
type ChatMessagesListResponse struct {
	Messages []*ChatMessageResponse `json:"messages"`
	Total    int                    `json:"total"`
	Limit    int                    `json:"limit"`
	Offset   int                    `json:"offset"`
	HasMore  bool                   `json:"has_more"`
	MaxLimit int                    `json:"max_limit,omitempty"`
}

// NewChatSessionResponse 从模型转换
func NewChatSessionResponse(session *model.ChatSession) *ChatSessionResponse {
	return &ChatSessionResponse{
		ID:           session.ID,
		UserID:       session.UserID,
		Title:        session.Title,
		Provider:     session.Provider,
		Model:        session.Model,
		ModelType:    session.ModelType,
		SystemPrompt: session.SystemPrompt,
		MessageCount: session.MessageCount,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
	}
}

// NewChatSessionResponseWithLastMessage 从模型转换（包含最后一条消息）
func NewChatSessionResponseWithLastMessage(session *model.ChatSession, lastMessage string) *ChatSessionResponse {
	resp := NewChatSessionResponse(session)
	resp.LastMessage = lastMessage
	return resp
}

// NewChatMessageResponse 从模型转换
func NewChatMessageResponse(message *model.Message) *ChatMessageResponse {
	var metadata map[string]interface{}
	if message.Metadata != nil {
		json.Unmarshal(message.Metadata, &metadata)
	}
	return &ChatMessageResponse{
		ID:         message.ID,
		SessionID:  message.SessionID,
		Role:       message.Role,
		Content:    message.Content,
		TokensUsed: message.TokensUsed,
		Metadata:   metadata,
		CreatedAt:  message.CreatedAt,
	}
}

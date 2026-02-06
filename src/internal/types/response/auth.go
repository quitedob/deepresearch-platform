package response

import "github.com/ai-research-platform/internal/repository/model"

// AuthResponse 认证响应
type AuthResponse struct {
    Token     string         `json:"token"`
    User      *model.User    `json:"user"`
    ExpiresIn int64          `json:"expires_in"`
}

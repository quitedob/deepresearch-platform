package auth

import (
    "context"
    "errors"
    "fmt"

    "github.com/google/uuid"
    "github.com/ai-research-platform/internal/types/request"
    "github.com/ai-research-platform/internal/types/response"
    "github.com/ai-research-platform/internal/repository/model"
    "github.com/ai-research-platform/internal/pkg/auth"
)

// service 认证服务实现
type service struct {
    userDAO     UserDAO
    jwtManager  *auth.JWTManager
}

// UserDAO 用户数据访问接口
type UserDAO interface {
    Create(ctx context.Context, user *model.User) error
    FindByEmail(ctx context.Context, email string) (*model.User, error)
    FindByID(ctx context.Context, id string) (*model.User, error)
}

// NewService 创建认证服务
func NewService(userDAO UserDAO, jwtManager *auth.JWTManager) Service {
    return &service{
        userDAO:    userDAO,
        jwtManager: jwtManager,
    }
}

// Register 用户注册
func (s *service) Register(ctx context.Context, req request.RegisterRequest) (*response.AuthResponse, error) {
    // 检查用户是否已存在
    existingUser, err := s.userDAO.FindByEmail(ctx, req.Email)
    if err == nil && existingUser != nil {
        return nil, errors.New("user already exists")
    }
    // 如果是其他错误（非"记录不存在"），继续注册流程

    // 哈希密码
    hashedPassword, err := auth.HashPassword(req.Password)
    if err != nil {
        return nil, errors.New("failed to process password")
    }

    // 创建用户 - 使用 UUID 生成真实的 ID
    // 安全：显式设置 Role/Status/IsAdmin，禁止通过注册接口创建管理员
    user := &model.User{
        ID:       uuid.New().String(),
        Email:    req.Email,
        Username: req.Username,
        Password: hashedPassword,
        Role:     "user",   // 普通用户注册只能是 user 角色
        Status:   "active", // 默认激活状态
        IsAdmin:  false,    // 禁止通过注册接口创建管理员
    }

    if err := s.userDAO.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("failed to create user: %v", err)
    }

    // 生成JWT令牌
    token, err := s.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
    if err != nil {
        return nil, errors.New("failed to generate token")
    }

    return &response.AuthResponse{
        Token:     token,
        User:      user,
        ExpiresIn: 86400, // 24小时
    }, nil
}

// Login 用户登录
func (s *service) Login(ctx context.Context, req request.LoginRequest) (*response.AuthResponse, error) {
    // 查找用户
    user, err := s.userDAO.FindByEmail(ctx, req.Email)
    if err != nil {
        return nil, errors.New("invalid credentials")
    }

    // 验证密码
    if !auth.CheckPassword(req.Password, user.Password) {
        return nil, errors.New("invalid credentials")
    }

    // 生成JWT令牌
    token, err := s.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
    if err != nil {
        return nil, errors.New("failed to generate token")
    }

    return &response.AuthResponse{
        Token:     token,
        User:      user,
        ExpiresIn: 86400, // 24小时
    }, nil
}

// RefreshToken 刷新令牌
func (s *service) RefreshToken(ctx context.Context, req request.RefreshTokenRequest) (*response.AuthResponse, error) {
    // 刷新令牌
    newToken, err := s.jwtManager.RefreshToken(req.Token)
    if err != nil {
        return nil, errors.New("invalid or expired token")
    }

    // 获取用户信息
    claims, err := s.jwtManager.ValidateToken(newToken)
    if err != nil {
        return nil, errors.New("failed to validate new token")
    }

    // 获取用户
    user, err := s.userDAO.FindByID(ctx, claims.UserID)
    if err != nil {
        return nil, errors.New("user not found")
    }

    return &response.AuthResponse{
        Token:     newToken,
        User:      user,
        ExpiresIn: 86400, // 24小时
    }, nil
}

// ValidateToken 验证令牌
func (s *service) ValidateToken(ctx context.Context, token string) (*TokenClaims, error) {
    claims, err := s.jwtManager.ValidateToken(token)
    if err != nil {
        return nil, err
    }

    return &TokenClaims{
        UserID:   claims.UserID,
        Email:    claims.Email,
        Username: claims.Username,
    }, nil
}

package handler

import (
	"net/http"

	"github.com/ai-research-platform/internal/auth"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuthHandler 处理认证相关请求
type AuthHandler struct {
	db         *gorm.DB
	jwtManager *auth.JWTManager
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(db *gorm.DB, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		db:         db,
		jwtManager: jwtManager,
	}
}

// RegisterRequest 表示用户注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest 表示用户登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest 表示令牌刷新请求
type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// AuthResponse 表示认证响应
type AuthResponse struct {
	Token     string       `json:"token"`
	User      *models.User `json:"user"`
	ExpiresIn int64        `json:"expires_in"`
}

// Register 处理用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 检查用户是否已存在
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "user with this email already exists",
		})
		return
	}

	// 哈希密码
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to process password",
		})
		return
	}

	// 创建用户
	user := &models.User{
		ID:       uuid.New().String(),
		Email:    req.Email,
		Username: req.Username,
		Password: hashedPassword,
	}

	if err := h.db.Create(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	// 生成JWT令牌
	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate token",
		})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token:     token,
		User:      user,
		ExpiresIn: 86400, // 24小时
	})
}

// Login 处理用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 根据邮箱查找用户
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid credentials",
		})
		return
	}

	// 验证密码
	if !auth.CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid credentials",
		})
		return
	}

	// 生成JWT令牌
	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:     token,
		User:      &user,
		ExpiresIn: 86400, // 24小时
	})
}

// RefreshToken 处理令牌刷新
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
			"details": err.Error(),
		})
		return
	}

	// 刷新令牌
	newToken, err := h.jwtManager.RefreshToken(req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid or expired token",
		})
		return
	}

	// 从令牌获取用户信息
	claims, err := h.jwtManager.ValidateToken(newToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to validate new token",
		})
		return
	}

	// 从数据库获取用户
	var user models.User
	if err := h.db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Token:     newToken,
		User:      &user,
		ExpiresIn: 86400, // 24小时
	})
}

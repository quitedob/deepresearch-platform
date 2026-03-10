package v1

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/ai-research-platform/internal/pkg"
    "github.com/ai-research-platform/internal/middleware"
    "github.com/ai-research-platform/internal/types/request"
    "github.com/ai-research-platform/internal/types/response"
    "github.com/ai-research-platform/internal/types/constant"
    "github.com/ai-research-platform/internal/pkg/auth"
    "github.com/ai-research-platform/internal/repository/dao"
    "github.com/ai-research-platform/internal/repository/model"
    authService "github.com/ai-research-platform/internal/service/auth"
)

// UserAPIEnhanced 增强的用户API
type UserAPIEnhanced struct {
    jwtManager         *auth.JWTManager
    authService        authService.Service
    userDAO            *dao.UserDAO
    userPreferencesDAO *dao.UserPreferencesDAO
    chatDAO            *dao.ChatDAO
    researchDAO        *dao.ResearchDAO
}

// NewUserAPIEnhanced 创建增强的用户API
func NewUserAPIEnhanced(jwtManager *auth.JWTManager, authSvc authService.Service, userDAO *dao.UserDAO) *UserAPIEnhanced {
    return &UserAPIEnhanced{
        jwtManager:  jwtManager,
        authService: authSvc,
        userDAO:     userDAO,
    }
}

// NewUserAPIEnhancedWithPreferences 创建带偏好设置的用户API
func NewUserAPIEnhancedWithPreferences(jwtManager *auth.JWTManager, authSvc authService.Service, userDAO *dao.UserDAO, prefsDAO *dao.UserPreferencesDAO) *UserAPIEnhanced {
    return &UserAPIEnhanced{
        jwtManager:         jwtManager,
        authService:        authSvc,
        userDAO:            userDAO,
        userPreferencesDAO: prefsDAO,
    }
}

// NewUserAPIEnhancedFull 创建完整的用户API（包含统计数据所需的DAO）
func NewUserAPIEnhancedFull(jwtManager *auth.JWTManager, authSvc authService.Service, userDAO *dao.UserDAO, prefsDAO *dao.UserPreferencesDAO, chatDAO *dao.ChatDAO, researchDAO *dao.ResearchDAO) *UserAPIEnhanced {
    return &UserAPIEnhanced{
        jwtManager:         jwtManager,
        authService:        authSvc,
        userDAO:            userDAO,
        userPreferencesDAO: prefsDAO,
        chatDAO:            chatDAO,
        researchDAO:        researchDAO,
    }
}

// Register 用户注册
func (api *UserAPIEnhanced) Register(c *gin.Context) {
    var req request.UserRegister
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    ctx := c.Request.Context()

    // 检查邮箱是否已存在
    exists, err := api.userDAO.ExistsByEmail(ctx, req.Email)
    if err != nil {
        pkg.InternalError(c, "检查邮箱失败")
        return
    }
    if exists {
        pkg.BadRequest(c, "邮箱已被注册")
        return
    }

    // 检查用户名是否已存在
    exists, err = api.userDAO.ExistsByUsername(ctx, req.Username)
    if err != nil {
        pkg.InternalError(c, "检查用户名失败")
        return
    }
    if exists {
        pkg.BadRequest(c, "用户名已被使用")
        return
    }

    // 哈希密码
    hashedPassword, err := auth.HashPassword(req.Password)
    if err != nil {
        pkg.InternalError(c, "密码处理失败")
        return
    }

    // 创建用户
    user := &model.User{
        ID:       pkg.GenerateUUID(),
        Email:    req.Email,
        Username: req.Username,
        Password: hashedPassword,
        Role:     "user",
        Status:   "active",
        IsAdmin:  false,
    }

    // 设置可选字段
    if req.FullName != "" {
        user.FullName = &req.FullName
    }

    if err := api.userDAO.Create(ctx, user); err != nil {
        pkg.InternalError(c, "创建用户失败: "+err.Error())
        return
    }

    // 生成访问令牌
    accessToken, err := api.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
    if err != nil {
        pkg.InternalError(c, "生成访问令牌失败")
        return
    }

    // 生成刷新令牌
    refreshToken, err := api.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
    if err != nil {
        pkg.InternalError(c, "生成刷新令牌失败")
        return
    }

    tokenResponse := &response.TokenResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    constant.TokenExpirationSeconds,
        User: &response.UserResponse{
            ID:        user.ID,
            Username:  user.Username,
            Email:     user.Email,
            FullName:  user.FullName,
            Role:      user.Role,
            Status:    user.Status,
            CreatedAt: user.CreatedAt,
            UpdatedAt: user.UpdatedAt,
            IsAdmin:   user.IsAdmin,
        },
    }

    c.JSON(http.StatusCreated, tokenResponse)
}

// Login 用户登录
func (api *UserAPIEnhanced) Login(c *gin.Context) {
    var req request.UserLogin
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    ctx := c.Request.Context()

    // 查找用户（支持邮箱或用户名登录）
    var user *model.User
    var err error

    // 先尝试邮箱查找
    user, err = api.userDAO.FindByEmail(ctx, req.Username)
    if err != nil {
        // 如果邮箱查找失败，尝试用户名查找
        user, err = api.userDAO.FindByUsername(ctx, req.Username)
        if err != nil {
            pkg.Unauthorized(c, "用户名或密码错误")
            return
        }
    }

    // 验证密码
    if !auth.CheckPassword(req.Password, user.Password) {
        pkg.Unauthorized(c, "用户名或密码错误")
        return
    }

    // 检查用户状态
    if user.Status != constant.UserStatusActive {
        pkg.Unauthorized(c, "账户已被禁用")
        return
    }

    // 生成访问令牌
    accessToken, err := api.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
    if err != nil {
        pkg.InternalError(c, "生成访问令牌失败")
        return
    }

    // 生成刷新令牌
    refreshToken, err := api.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
    if err != nil {
        pkg.InternalError(c, "生成刷新令牌失败")
        return
    }

    tokenResponse := &response.TokenResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    constant.TokenExpirationSeconds,
        User: &response.UserResponse{
            ID:        user.ID,
            Username:  user.Username,
            Email:     user.Email,
            FullName:  user.FullName,
            Role:      user.Role,
            Status:    user.Status,
            CreatedAt: user.CreatedAt,
            UpdatedAt: user.UpdatedAt,
            IsAdmin:   user.IsAdmin,
        },
    }

    pkg.Success(c, tokenResponse)
}


// GetCurrentUser 获取当前用户信息
func (api *UserAPIEnhanced) GetCurrentUser(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    ctx := c.Request.Context()

    // 从数据库查询用户信息
    user, err := api.userDAO.FindByID(ctx, userID)
    if err != nil {
        pkg.NotFound(c, "用户不存在")
        return
    }

    userResponse := &response.UserResponse{
        ID:        user.ID,
        Username:  user.Username,
        Email:     user.Email,
        FullName:  user.FullName,
        Phone:     user.Phone,
        Avatar:    user.Avatar,
        Bio:       user.Bio,
        Role:      user.Role,
        Status:    user.Status,
        CreatedAt: user.CreatedAt,
        UpdatedAt: user.UpdatedAt,
        IsAdmin:   user.IsAdmin,
    }

    pkg.Success(c, userResponse)
}

// UpdateProfile 更新用户资料
func (api *UserAPIEnhanced) UpdateProfile(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    var req request.UserUpdate
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    ctx := c.Request.Context()

    // 从数据库获取用户
    user, err := api.userDAO.FindByID(ctx, userID)
    if err != nil {
        pkg.NotFound(c, "用户不存在")
        return
    }

    // 更新字段（仅更新非nil字段）
    if req.FullName != nil {
        user.FullName = req.FullName
    }
    if req.Avatar != nil {
        user.Avatar = req.Avatar
    }
    if req.Bio != nil {
        user.Bio = req.Bio
    }

    // 保存到数据库
    if err := api.userDAO.Update(ctx, user); err != nil {
        pkg.InternalError(c, "更新资料失败: "+err.Error())
        return
    }

    pkg.Success(c, gin.H{
        "success": true,
        "message": "资料更新成功",
        "user": gin.H{
            "id":        user.ID,
            "username":  user.Username,
            "email":     user.Email,
            "full_name": user.FullName,
            "avatar":    user.Avatar,
            "bio":       user.Bio,
        },
    })
}

// GetPreferences 获取用户偏好设置
func (api *UserAPIEnhanced) GetPreferences(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    ctx := c.Request.Context()

    // 从数据库获取用户偏好设置
    if api.userPreferencesDAO != nil {
        prefs, err := api.userPreferencesDAO.GetOrCreate(ctx, userID)
        if err == nil && prefs != nil {
            preferences := &response.UserPreferencesResponse{
                Theme:               prefs.Theme,
                Language:            prefs.Language,
                DefaultLLMProvider:  prefs.DefaultLLMProvider,
                DefaultModel:        prefs.DefaultModel,
                StreamEnabled:       prefs.StreamEnabled,
                NotificationEnabled: prefs.NotificationEnabled,
                AutoSaveEnabled:     prefs.AutoSaveEnabled,
                TimeZone:            prefs.TimeZone,
                MemoryEnabled:       prefs.MemoryEnabled,
                CustomSystemPrompt:  prefs.CustomSystemPrompt,
                MaxContextTokens:    prefs.MaxContextTokens,
                UpdatedAt:           prefs.UpdatedAt,
            }
            pkg.Success(c, preferences)
            return
        }
    }

    // 返回默认设置
    preferences := &response.UserPreferencesResponse{
        Theme:               "light",
        Language:            "zh",
        DefaultLLMProvider:  constant.DefaultProvider,
        DefaultModel:        constant.DefaultModel,
        StreamEnabled:       true,
        NotificationEnabled: true,
        AutoSaveEnabled:     true,
        TimeZone:            "Asia/Shanghai",
        MemoryEnabled:       true,
        CustomSystemPrompt:  "",
        MaxContextTokens:    128000,
        UpdatedAt:           time.Now(),
    }

    pkg.Success(c, preferences)
}

// UpdatePreferences 更新用户偏好设置
func (api *UserAPIEnhanced) UpdatePreferences(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    var req request.UserPreferences
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    ctx := c.Request.Context()

    // 更新数据库中的偏好设置
    if api.userPreferencesDAO != nil {
        prefs, err := api.userPreferencesDAO.GetOrCreate(ctx, userID)
        if err != nil {
            pkg.InternalError(c, "获取偏好设置失败")
            return
        }

        // 更新字段
        if req.Theme != "" {
            prefs.Theme = req.Theme
        }
        if req.Language != "" {
            prefs.Language = req.Language
        }
        if req.DefaultLLMProvider != "" {
            prefs.DefaultLLMProvider = req.DefaultLLMProvider
        }
        if req.DefaultModel != "" {
            prefs.DefaultModel = req.DefaultModel
        }
        prefs.StreamEnabled = req.StreamEnabled
        prefs.NotificationEnabled = req.NotificationEnabled
        prefs.AutoSaveEnabled = req.AutoSaveEnabled
        if req.TimeZone != "" {
            prefs.TimeZone = req.TimeZone
        }
        // 更新记忆设置
        if req.MemoryEnabled != nil {
            prefs.MemoryEnabled = *req.MemoryEnabled
        }
        if req.CustomSystemPrompt != nil {
            prefs.CustomSystemPrompt = *req.CustomSystemPrompt
        }
        if req.MaxContextTokens != nil && *req.MaxContextTokens > 0 {
            prefs.MaxContextTokens = *req.MaxContextTokens
        }

        if err := api.userPreferencesDAO.Update(ctx, prefs); err != nil {
            pkg.InternalError(c, "更新偏好设置失败")
            return
        }
    }

    pkg.Success(c, gin.H{
        "success": true,
        "message": "偏好设置更新成功",
    })
}

// GetMemorySettings 获取记忆设置
func (api *UserAPIEnhanced) GetMemorySettings(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    ctx := c.Request.Context()

    if api.userPreferencesDAO != nil {
        prefs, err := api.userPreferencesDAO.GetOrCreate(ctx, userID)
        if err == nil && prefs != nil {
            pkg.Success(c, gin.H{
                "memory_enabled":       prefs.MemoryEnabled,
                "custom_system_prompt": prefs.CustomSystemPrompt,
                "max_context_tokens":   prefs.MaxContextTokens,
            })
            return
        }
    }

    // 返回默认设置
    pkg.Success(c, gin.H{
        "memory_enabled":       true,
        "custom_system_prompt": "",
        "max_context_tokens":   128000,
    })
}

// UpdateMemorySettings 更新记忆设置
func (api *UserAPIEnhanced) UpdateMemorySettings(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    var req request.MemorySettingsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    ctx := c.Request.Context()

    if api.userPreferencesDAO != nil {
        // 设置默认值
        maxTokens := req.MaxContextTokens
        if maxTokens <= 0 {
            maxTokens = 128000
        }

        if err := api.userPreferencesDAO.UpdateMemorySettings(ctx, userID, req.MemoryEnabled, req.CustomSystemPrompt, maxTokens); err != nil {
            pkg.InternalError(c, "更新记忆设置失败")
            return
        }
    }

    pkg.Success(c, gin.H{
        "success": true,
        "message": "记忆设置更新成功",
    })
}

// RefreshToken 刷新访问令牌
func (api *UserAPIEnhanced) RefreshToken(c *gin.Context) {
    var req request.UserRefreshTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    claims, err := api.jwtManager.ValidateToken(req.RefreshToken)
    if err != nil {
        pkg.Unauthorized(c, "刷新令牌无效或已过期")
        return
    }

    newAccessToken, err := api.jwtManager.GenerateToken(claims.UserID, claims.Email, claims.Username)
    if err != nil {
        pkg.InternalError(c, "生成新的访问令牌失败")
        return
    }

    newRefreshToken, err := api.jwtManager.GenerateToken(claims.UserID, claims.Email, claims.Username)
    if err != nil {
        pkg.InternalError(c, "生成新的刷新令牌失败")
        return
    }

    tokenResponse := &response.TokenResponse{
        AccessToken:  newAccessToken,
        RefreshToken: newRefreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    constant.TokenExpirationSeconds,
        User: &response.UserResponse{
            ID:       claims.UserID,
            Username: claims.Username,
            Email:    claims.Email,
            Role:     "user",
            Status:   constant.UserStatusActive,
        },
    }

    pkg.Success(c, tokenResponse)
}

// Logout 用户登出
func (api *UserAPIEnhanced) Logout(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    pkg.Success(c, gin.H{
        "success": true,
        "message": "登出成功",
        "user_id": userID,
    })
}

// ChangePassword 修改密码
func (api *UserAPIEnhanced) ChangePassword(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    var req request.ChangePasswordRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    ctx := c.Request.Context()

    // 获取用户
    user, err := api.userDAO.FindByID(ctx, userID)
    if err != nil {
        pkg.NotFound(c, "用户不存在")
        return
    }

    // 验证旧密码
    if !auth.CheckPassword(req.CurrentPassword, user.Password) {
        pkg.BadRequest(c, "当前密码不正确")
        return
    }

    // 哈希新密码
    hashedPassword, err := auth.HashPassword(req.NewPassword)
    if err != nil {
        pkg.InternalError(c, "密码加密失败")
        return
    }

    // 更新密码
    user.Password = hashedPassword
    if err := api.userDAO.Update(ctx, user); err != nil {
        pkg.InternalError(c, "密码更新失败: "+err.Error())
        return
    }

    pkg.Success(c, gin.H{
        "success": true,
        "message": "密码修改成功",
    })
}

// GetUserStats 获取用户统计
func (api *UserAPIEnhanced) GetUserStats(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    ctx := c.Request.Context()

    // 获取用户信息（用于计算账户年龄）
    user, err := api.userDAO.FindByID(ctx, userID)
    if err != nil {
        pkg.NotFound(c, "用户不存在")
        return
    }

    accountAge := int64(time.Since(user.CreatedAt).Hours() / 24)

    // 查询真实统计数据
    var totalChatSessions int64
    var totalResearchSessions int64
    var totalMessages int64

    if api.chatDAO != nil {
        totalChatSessions, _ = api.chatDAO.CountSessionsByUserID(ctx, userID)
        totalMessages, _ = api.chatDAO.CountMessagesByUserID(ctx, userID)
    }
    if api.researchDAO != nil {
        totalResearchSessions, _ = api.researchDAO.CountSessionsByUserID(ctx, userID)
    }

    // 获取用户偏好中的默认模型信息
    mostUsedProvider := constant.DefaultProvider
    mostUsedModel := constant.DefaultModel
    if api.userPreferencesDAO != nil {
        prefs, err := api.userPreferencesDAO.GetOrCreate(ctx, userID)
        if err == nil && prefs != nil {
            if prefs.DefaultLLMProvider != "" {
                mostUsedProvider = prefs.DefaultLLMProvider
            }
            if prefs.DefaultModel != "" {
                mostUsedModel = prefs.DefaultModel
            }
        }
    }

    statistics := &response.UserStatistics{
        UserID:                userID,
        TotalChatSessions:     totalChatSessions,
        TotalResearchSessions: totalResearchSessions,
        TotalMessages:         totalMessages,
        TotalTokensUsed:       0, // 暂无token统计表
        LastActivity:          user.UpdatedAt,
        AccountAge:            accountAge,
        MostUsedProvider:      mostUsedProvider,
        MostUsedModel:         mostUsedModel,
        PreferredResearchType: constant.ResearchTypeQuick,
    }

    resp := &response.UserStatsResponse{
        Success:    true,
        Statistics: statistics,
        Message:    "获取统计信息成功",
    }

    pkg.Success(c, resp)
}

// DeleteAccount 删除账户
func (api *UserAPIEnhanced) DeleteAccount(c *gin.Context) {
    userID, err := middleware.RequireAuth(c)
    if err != nil {
        pkg.Unauthorized(c, "认证失败")
        return
    }

    var req request.DeleteAccountRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        pkg.BadRequest(c, "无效的请求参数: "+err.Error())
        return
    }

    if !req.Confirm {
        pkg.BadRequest(c, "必须确认删除操作")
        return
    }

    ctx := c.Request.Context()

    // 验证密码
    user, err := api.userDAO.FindByID(ctx, userID)
    if err != nil {
        pkg.NotFound(c, "用户不存在")
        return
    }

    if !auth.CheckPassword(req.Password, user.Password) {
        pkg.BadRequest(c, "密码不正确")
        return
    }

    // 软删除：将状态设为 banned 并触发 GORM 软删除
    user.Status = constant.UserStatusBanned
    if err := api.userDAO.Update(ctx, user); err != nil {
        pkg.InternalError(c, "删除账户失败")
        return
    }

    // 执行 GORM 软删除（设置 deleted_at）
    if err := api.userDAO.Delete(ctx, userID); err != nil {
        pkg.InternalError(c, "删除账户失败")
        return
    }

    scheduledAt := time.Now().Add(time.Hour * 24 * 30)

    resp := &response.AccountDeletionResponse{
        Success:     true,
        Message:     "账户已标记删除，数据将在30天后永久清除",
        ScheduledAt: &scheduledAt,
    }

    pkg.Success(c, resp)
}

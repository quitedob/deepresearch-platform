package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/pkg"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
)

// MembershipAPI 会员API（用户端）
type MembershipAPI struct {
	membershipDAO     *dao.MembershipDAO
	activationCodeDAO *dao.ActivationCodeDAO
}

// NewMembershipAPI 创建会员API
func NewMembershipAPI(membershipDAO *dao.MembershipDAO, activationCodeDAO *dao.ActivationCodeDAO) *MembershipAPI {
	return &MembershipAPI{
		membershipDAO:     membershipDAO,
		activationCodeDAO: activationCodeDAO,
	}
}

// GetMembership 获取会员信息
func (api *MembershipAPI) GetMembership(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return
	}

	membership, err := api.membershipDAO.GetOrCreateMembership(c.Request.Context(), userID)
	if err != nil {
		pkg.InternalError(c, "获取会员信息失败")
		return
	}

	pkg.Success(c, gin.H{
		"membership": membership,
	})
}

// GetQuota 获取配额信息
func (api *MembershipAPI) GetQuota(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return
	}

	membership, err := api.membershipDAO.GetOrCreateMembership(c.Request.Context(), userID)
	if err != nil {
		pkg.InternalError(c, "获取配额信息失败")
		return
	}

	var chatRemaining, chatLimit, researchRemaining, researchLimit int
	var resetAt interface{}

	if membership.MembershipType == model.MembershipPremium {
		chatRemaining = membership.PremiumChatLimit - membership.PremiumChatUsed
		chatLimit = membership.PremiumChatLimit
		researchRemaining = membership.PremiumResearchLimit - membership.PremiumResearchUsed
		researchLimit = membership.PremiumResearchLimit
		resetAt = membership.PremiumResetAt
	} else {
		chatRemaining = membership.NormalChatLimit - membership.NormalChatUsed
		chatLimit = membership.NormalChatLimit
		researchRemaining = membership.ResearchLimit - membership.ResearchUsed
		researchLimit = membership.ResearchLimit
		resetAt = nil
	}

	if chatRemaining < 0 {
		chatRemaining = 0
	}
	if researchRemaining < 0 {
		researchRemaining = 0
	}

	pkg.Success(c, gin.H{
		"membership_type":     membership.MembershipType,
		"chat_remaining":      chatRemaining,
		"chat_limit":          chatLimit,
		"research_remaining":  researchRemaining,
		"research_limit":      researchLimit,
		"reset_at":            resetAt,
		"expires_at":          membership.ExpiresAt,
	})
}

// ActivateCode 使用激活码
func (api *MembershipAPI) ActivateCode(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return
	}

	var req struct {
		Code string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "请输入激活码")
		return
	}

	// 使用激活码
	activationCode, err := api.activationCodeDAO.UseCode(c.Request.Context(), req.Code, userID)
	if err != nil {
		pkg.BadRequest(c, "激活码无效或已过期")
		return
	}

	// 升级为高级会员
	if err := api.membershipDAO.UpgradeToPremium(c.Request.Context(), userID, activationCode.ValidDays, "activation_code", &activationCode.ID); err != nil {
		pkg.InternalError(c, "激活失败")
		return
	}

	pkg.Success(c, gin.H{
		"success":    true,
		"message":    "激活成功",
		"valid_days": activationCode.ValidDays,
	})
}

// CheckChatQuota 检查聊天配额（中间件使用）
func (api *MembershipAPI) CheckChatQuota(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return
	}

	hasQuota, remaining, limit, err := api.membershipDAO.CheckChatQuota(c.Request.Context(), userID)
	if err != nil {
		pkg.InternalError(c, "检查配额失败")
		return
	}

	pkg.Success(c, gin.H{
		"has_quota": hasQuota,
		"remaining": remaining,
		"limit":     limit,
	})
}

// CheckResearchQuota 检查研究配额
func (api *MembershipAPI) CheckResearchQuota(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "认证失败")
		return
	}

	hasQuota, remaining, limit, err := api.membershipDAO.CheckResearchQuota(c.Request.Context(), userID)
	if err != nil {
		pkg.InternalError(c, "检查配额失败")
		return
	}

	pkg.Success(c, gin.H{
		"has_quota": hasQuota,
		"remaining": remaining,
		"limit":     limit,
	})
}

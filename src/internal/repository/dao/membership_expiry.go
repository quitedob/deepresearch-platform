package dao

import (
	"time"

	"github.com/ai-research-platform/internal/repository/model"
)

// expirePremiumMembership must be persisted while holding the membership row lock.
func expirePremiumMembership(membership *model.UserMembership, now time.Time) bool {
	if membership.MembershipType != model.MembershipPremium || membership.ExpiresAt == nil || now.Before(*membership.ExpiresAt) {
		return false
	}
	membership.MembershipType = model.MembershipFree
	membership.ExpiresAt = nil
	membership.NormalChatUsed = 0
	membership.ResearchUsed = 0
	return true
}

package dao

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestResearchQuotaDowngradesExpiredPremiumUnderLock(t *testing.T) {
	database := openResearchWorkflowTestDB(t)
	now := time.Now().UTC()
	for _, test := range []struct {
		name        string
		expiry      time.Time
		wantType    model.MembershipType
		wantCounter string
	}{
		{name: "expired", expiry: now.Add(-time.Minute), wantType: model.MembershipFree, wantCounter: "research_used"},
		{name: "expires now", expiry: now, wantType: model.MembershipFree, wantCounter: "research_used"},
		{name: "active", expiry: now.Add(time.Hour), wantType: model.MembershipPremium, wantCounter: "premium_research_used"},
	} {
		t.Run(test.name, func(t *testing.T) {
			userID := uuid.NewString()
			membership := &model.UserMembership{UserID: userID, MembershipType: model.MembershipPremium, ExpiresAt: &test.expiry, ResearchLimit: 1, ResearchUsed: 1, PremiumResearchLimit: 10, PremiumResearchUsed: 2}
			if err := database.Create(membership).Error; err != nil {
				t.Fatal(err)
			}
			err := database.Transaction(func(transaction *gorm.DB) error {
				updated, counter, _, _, err := deductResearchQuotaCounterTx(transaction, userID, now)
				if err != nil {
					return err
				}
				if updated.MembershipType != test.wantType || counter != test.wantCounter {
					t.Fatalf("type/counter = %s/%s, want %s/%s", updated.MembershipType, counter, test.wantType, test.wantCounter)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if test.wantType == model.MembershipFree {
				if err := database.Where("user_id = ?", userID).First(membership).Error; err != nil {
					t.Fatal(err)
				}
				if membership.ExpiresAt != nil || membership.ResearchUsed != 1 || membership.PremiumResearchUsed != 2 {
					t.Fatalf("expiry downgrade was not persisted correctly: %+v", membership)
				}
				_, err := NewResearchSubmissionDAO(database).Submit(context.Background(), ResearchSubmissionCommand{UserID: userID, Endpoint: "/api/v1/research/start", IdempotencyKey: "expired-quota-retry", Query: "A second research request", ResearchType: "deep"})
				if !errors.Is(err, ErrResearchQuotaExceeded) {
					t.Fatalf("second request error = %v, want exhausted free quota", err)
				}
			}
		})
	}
}

func TestMembershipRefreshDoesNotResetConcurrentResearchCharge(t *testing.T) {
	database := openResearchWorkflowTestDB(t)
	userID := uuid.NewString()
	expiredAt := time.Now().Add(-time.Hour)
	if err := database.Create(&model.UserMembership{UserID: userID, MembershipType: model.MembershipPremium, ExpiresAt: &expiredAt, ResearchLimit: 1}).Error; err != nil {
		t.Fatal(err)
	}
	paused := make(chan struct{})
	resume := make(chan struct{})
	var pauseOnce, resumeOnce sync.Once
	release := func() { resumeOnce.Do(func() { close(resume) }) }
	defer release()
	type membershipReadMarker struct{}
	if err := database.Callback().Query().After("gorm:query").Register("test:pause-membership-read", func(query *gorm.DB) {
		if query.Statement.Context.Value(membershipReadMarker{}) == true {
			pauseOnce.Do(func() { close(paused); <-resume })
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Callback().Query().Remove("test:pause-membership-read") })
	finished := make(chan error, 1)
	go func() {
		_, err := NewMembershipDAO(database).GetOrCreateMembership(context.WithValue(context.Background(), membershipReadMarker{}, true), userID)
		finished <- err
	}()
	select {
	case <-paused:
	case <-time.After(time.Second):
		t.Fatal("membership read did not reach the race boundary")
	}
	_, err := NewResearchSubmissionDAO(database).Submit(context.Background(), ResearchSubmissionCommand{UserID: userID, Endpoint: "/api/v1/research/start", IdempotencyKey: "concurrent-membership-refresh", Query: "Concurrent expiry test", ResearchType: "deep"})
	if err != nil {
		t.Fatal(err)
	}
	release()
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	var stored model.UserMembership
	if err := database.Where("user_id = ?", userID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.MembershipType != model.MembershipFree || stored.ResearchUsed != 1 {
		t.Fatalf("membership refresh erased the concurrent charge: %+v", stored)
	}
}

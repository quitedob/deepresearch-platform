package dao

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestResearchJobClaimUsesExclusiveLeaseAndFence(t *testing.T) {
	db := openResearchWorkflowTestDB(t)
	result := seedQueuedResearch(t, db, 3)
	jobs := NewResearchJobDAO(db)
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)

	first, err := jobs.Claim(context.Background(), "worker-a", now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if first.ResearchID != result.SessionID || first.LeaseOwner != "worker-a" ||
		first.FenceToken != 1 || first.Attempts != 1 || first.Status != model.ResearchJobRunning {
		t.Fatalf("unexpected first lease: %+v", first)
	}
	if second, err := jobs.Claim(context.Background(), "worker-b", now, time.Minute); !errors.Is(err, gorm.ErrRecordNotFound) || second != nil {
		t.Fatalf("second claim = %+v, %v; want no job", second, err)
	}

	if err := db.Model(&model.ResearchJob{}).
		Where("id = ?", first.ID).
		Update("lease_until", now.Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	second, err := jobs.Claim(context.Background(), "worker-b", now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if second.FenceToken != 2 || second.Attempts != 2 || second.LeaseOwner != "worker-b" {
		t.Fatalf("unexpected reclaimed lease: %+v", second)
	}

	stale := LeaseFromJob(first)
	if err := jobs.Heartbeat(context.Background(), stale, now.Add(2*time.Minute)); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("stale heartbeat error = %v, want ErrLeaseLost", err)
	}
	if err := jobs.Complete(context.Background(), stale, now); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("stale completion error = %v, want ErrLeaseLost", err)
	}

	current := LeaseFromJob(second)
	if err := jobs.Heartbeat(context.Background(), current, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := jobs.ConsumeReservation(context.Background(), current, now); err != nil {
		t.Fatal(err)
	}
	if err := jobs.Complete(context.Background(), current, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	var reservation model.ResearchQuotaReservation
	if err := db.Where("research_id = ?", result.SessionID).First(&reservation).Error; err != nil {
		t.Fatal(err)
	}
	if reservation.Status != model.ReservationConsumed || reservation.ConsumedAt == nil {
		t.Fatalf("reservation = %+v, want consumed", reservation)
	}
	var job model.ResearchJob
	if err := db.Where("id = ?", second.ID).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != model.ResearchJobSucceeded || job.CompletedAt == nil {
		t.Fatalf("job = %+v, want succeeded", job)
	}
}

func TestResearchJobRetryAndPermanentFailure(t *testing.T) {
	t.Run("retry returns job to queue", func(t *testing.T) {
		db := openResearchWorkflowTestDB(t)
		seedQueuedResearch(t, db, 3)
		jobs := NewResearchJobDAO(db)
		now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
		job, err := jobs.Claim(context.Background(), "worker-a", now, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		availableAt := now.Add(10 * time.Second)
		if err := jobs.Retry(context.Background(), LeaseFromJob(job), availableAt, errors.New("temporary")); err != nil {
			t.Fatal(err)
		}
		var stored model.ResearchJob
		if err := db.First(&stored, "id = ?", job.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.Status != model.ResearchJobQueued || !stored.AvailableAt.Equal(availableAt) ||
			stored.LeaseOwner != "" || stored.LeaseUntil != nil {
			t.Fatalf("retried job = %+v", stored)
		}
	})

	t.Run("pre-execution failure releases quota", func(t *testing.T) {
		db := openResearchWorkflowTestDB(t)
		result := seedQueuedResearch(t, db, 3)
		jobs := NewResearchJobDAO(db)
		now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
		job, err := jobs.Claim(context.Background(), "worker-a", now, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if err := jobs.Fail(context.Background(), LeaseFromJob(job), now, errors.New("permanent"), true); err != nil {
			t.Fatal(err)
		}

		var reservation model.ResearchQuotaReservation
		if err := db.Where("research_id = ?", result.SessionID).First(&reservation).Error; err != nil {
			t.Fatal(err)
		}
		if reservation.Status != model.ReservationReleased || reservation.ReleasedAt == nil {
			t.Fatalf("reservation = %+v, want released", reservation)
		}
		var session model.ResearchSession
		if err := db.First(&session, "id = ?", result.SessionID).Error; err != nil {
			t.Fatal(err)
		}
		if session.Status != "failed" {
			t.Fatalf("session status = %q, want failed", session.Status)
		}
		var membership model.UserMembership
		if err := db.Where("user_id = ?", session.UserID).First(&membership).Error; err != nil {
			t.Fatal(err)
		}
		if membership.ResearchUsed != 0 {
			t.Fatalf("research used = %d, want 0", membership.ResearchUsed)
		}
	})
}

func TestResearchJobClaimReclaimsExpiredFinalAttemptForTerminalization(t *testing.T) {
	db := openResearchWorkflowTestDB(t)
	seedQueuedResearch(t, db, 1)
	jobs := NewResearchJobDAO(db)
	now := time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)

	first, err := jobs.Claim(context.Background(), "worker-a", now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ResearchJob{}).
		Where("id = ?", first.ID).
		Update("lease_until", now.Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	reclaimed, err := jobs.Claim(context.Background(), "worker-b", now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed.Attempts != 2 || reclaimed.MaxAttempts != 1 {
		t.Fatalf("reclaimed attempts = %d/%d, want 2/1", reclaimed.Attempts, reclaimed.MaxAttempts)
	}
}

func seedQueuedResearch(t *testing.T, db *gorm.DB, maxAttempts int) *ResearchSubmissionResult {
	t.Helper()
	userID := uuid.NewString()
	if err := db.Create(&model.UserMembership{
		UserID:         userID,
		MembershipType: model.MembershipFree,
		ResearchLimit:  5,
	}).Error; err != nil {
		t.Fatal(err)
	}
	result, err := NewResearchSubmissionDAO(db).Submit(context.Background(), ResearchSubmissionCommand{
		UserID:         userID,
		Endpoint:       "/api/v1/research/start",
		IdempotencyKey: "job-test-" + uuid.NewString(),
		Query:          "query",
		ResearchType:   "deep",
		Metadata:       datatypes.JSON(`{"options":{"language":"en"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.ResearchJob{}).
		Where("research_id = ?", result.SessionID).
		Update("max_attempts", maxAttempts).Error; err != nil {
		t.Fatal(err)
	}
	return result
}

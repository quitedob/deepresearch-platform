package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrLeaseLost            = errors.New("research job lease lost")
	ErrJobAttemptsExhausted = errors.New("research job attempts exhausted")
)

type JobLease struct {
	JobID      string
	ResearchID string
	Owner      string
	FenceToken int64
	LeaseUntil time.Time
}

func LeaseFromJob(job *model.ResearchJob) JobLease {
	lease := JobLease{
		JobID:      job.ID,
		ResearchID: job.ResearchID,
		Owner:      job.LeaseOwner,
		FenceToken: job.FenceToken,
	}
	if job.LeaseUntil != nil {
		lease.LeaseUntil = *job.LeaseUntil
	}
	return lease
}

type ResearchJobDAO struct {
	db *gorm.DB
}

func NewResearchJobDAO(db *gorm.DB) *ResearchJobDAO {
	return &ResearchJobDAO{db: db}
}

func (d *ResearchJobDAO) Claim(
	ctx context.Context,
	owner string,
	now time.Time,
	leaseDuration time.Duration,
) (*model.ResearchJob, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("research job database is not configured")
	}
	if owner == "" {
		return nil, errors.New("research job lease owner is required")
	}
	var job model.ResearchJob
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"(status = ? AND available_at <= ? AND attempts < max_attempts) OR (status = ? AND lease_until < ?)",
				model.ResearchJobQueued,
				now,
				model.ResearchJobRunning,
				now,
			).
			Order("available_at ASC, created_at ASC").
			First(&job).Error; err != nil {
			return err
		}
		leaseUntil := now.Add(leaseDuration)
		job.Status = model.ResearchJobRunning
		job.Attempts++
		job.FenceToken++
		job.LeaseOwner = owner
		job.LeaseUntil = &leaseUntil
		if job.StartedAt == nil {
			startedAt := now
			job.StartedAt = &startedAt
		}
		job.UpdatedAt = now
		return tx.Save(&job).Error
	})
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (d *ResearchJobDAO) GetSession(ctx context.Context, researchID string) (*model.ResearchSession, error) {
	var session model.ResearchSession
	if err := d.db.WithContext(ctx).First(&session, "id = ?", researchID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (d *ResearchJobDAO) Heartbeat(ctx context.Context, lease JobLease, until time.Time) error {
	result := d.db.WithContext(ctx).Model(&model.ResearchJob{}).
		Where(
			"id = ? AND status = ? AND lease_owner = ? AND fence_token = ?",
			lease.JobID,
			model.ResearchJobRunning,
			lease.Owner,
			lease.FenceToken,
		).
		Updates(map[string]interface{}{"lease_until": until, "updated_at": time.Now().UTC()})
	return requireLeasedUpdate(result)
}

func (d *ResearchJobDAO) ConsumeReservation(ctx context.Context, lease JobLease, now time.Time) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job, err := loadLeasedJob(tx, lease)
		if err != nil {
			return err
		}
		var reservation model.ResearchQuotaReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("research_id = ?", job.ResearchID).
			First(&reservation).Error; err != nil {
			return fmt.Errorf("load research quota reservation: %w", err)
		}
		switch reservation.Status {
		case model.ReservationConsumed:
			return nil
		case model.ReservationReserved:
			reservation.Status = model.ReservationConsumed
			consumedAt := now
			reservation.ConsumedAt = &consumedAt
			if err := tx.Save(&reservation).Error; err != nil {
				return fmt.Errorf("consume research quota reservation: %w", err)
			}
			return tx.Model(&model.ResearchSession{}).
				Where("id = ?", job.ResearchID).
				Updates(map[string]interface{}{"status": "executing", "updated_at": now}).Error
		default:
			return fmt.Errorf("cannot consume %q research quota reservation", reservation.Status)
		}
	})
}

func (d *ResearchJobDAO) Complete(ctx context.Context, lease JobLease, now time.Time) error {
	result := d.db.WithContext(ctx).Model(&model.ResearchJob{}).
		Where(
			"id = ? AND status = ? AND lease_owner = ? AND fence_token = ?",
			lease.JobID,
			model.ResearchJobRunning,
			lease.Owner,
			lease.FenceToken,
		).
		Updates(map[string]interface{}{
			"status":       model.ResearchJobSucceeded,
			"completed_at": now,
			"lease_owner":  "",
			"lease_until":  nil,
			"last_error":   "",
			"updated_at":   now,
		})
	return requireLeasedUpdate(result)
}

func (d *ResearchJobDAO) Retry(
	ctx context.Context,
	lease JobLease,
	availableAt time.Time,
	cause error,
) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job, err := loadLeasedJob(tx, lease)
		if err != nil {
			return err
		}
		if job.Attempts >= job.MaxAttempts {
			return ErrJobAttemptsExhausted
		}
		job.Status = model.ResearchJobQueued
		job.AvailableAt = availableAt
		job.LeaseOwner = ""
		job.LeaseUntil = nil
		job.LastError = errorString(cause)
		job.UpdatedAt = time.Now().UTC()
		if err := tx.Save(job).Error; err != nil {
			return err
		}
		return tx.Model(&model.ResearchSession{}).
			Where("id = ?", job.ResearchID).
			Updates(map[string]interface{}{"status": "planning", "progress": 0, "updated_at": job.UpdatedAt}).Error
	})
}

func (d *ResearchJobDAO) Fail(
	ctx context.Context,
	lease JobLease,
	now time.Time,
	cause error,
	releaseReservation bool,
) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job, err := loadLeasedJob(tx, lease)
		if err != nil {
			return err
		}
		if releaseReservation {
			if err := releaseResearchReservation(tx, job.ResearchID, now); err != nil {
				return err
			}
		}
		job.Status = model.ResearchJobFailed
		job.CompletedAt = &now
		job.LeaseOwner = ""
		job.LeaseUntil = nil
		job.LastError = errorString(cause)
		job.UpdatedAt = now
		if err := tx.Save(job).Error; err != nil {
			return fmt.Errorf("mark research job failed: %w", err)
		}
		return tx.Model(&model.ResearchSession{}).
			Where("id = ?", job.ResearchID).
			Updates(map[string]interface{}{"status": "failed", "progress": 0, "updated_at": now}).Error
	})
}

func loadLeasedJob(tx *gorm.DB, lease JobLease) (*model.ResearchJob, error) {
	var job model.ResearchJob
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"id = ? AND status = ? AND lease_owner = ? AND fence_token = ?",
			lease.JobID,
			model.ResearchJobRunning,
			lease.Owner,
			lease.FenceToken,
		).
		First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrLeaseLost
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func releaseResearchReservation(tx *gorm.DB, researchID string, now time.Time) error {
	var reservation model.ResearchQuotaReservation
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("research_id = ?", researchID).
		First(&reservation).Error; err != nil {
		return fmt.Errorf("load research quota reservation: %w", err)
	}
	if reservation.Status == model.ReservationReleased {
		return nil
	}
	if reservation.Status != model.ReservationReserved {
		return nil
	}

	var membership model.UserMembership
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", reservation.UserID).
		First(&membership).Error; err != nil {
		return fmt.Errorf("lock membership for quota release: %w", err)
	}
	switch reservation.CounterName {
	case "research_used":
		if membership.ResearchUsed > 0 {
			membership.ResearchUsed -= reservation.Units
			if membership.ResearchUsed < 0 {
				membership.ResearchUsed = 0
			}
		}
	case "premium_research_used":
		if membership.PremiumResearchUsed > 0 {
			membership.PremiumResearchUsed -= reservation.Units
			if membership.PremiumResearchUsed < 0 {
				membership.PremiumResearchUsed = 0
			}
		}
	default:
		return fmt.Errorf("unknown research quota counter %q", reservation.CounterName)
	}
	if err := tx.Save(&membership).Error; err != nil {
		return fmt.Errorf("release membership quota counter: %w", err)
	}
	releasedAt := now
	reservation.Status = model.ReservationReleased
	reservation.ReleasedAt = &releasedAt
	return tx.Save(&reservation).Error
}

func requireLeasedUpdate(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrLeaseLost
	}
	return nil
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

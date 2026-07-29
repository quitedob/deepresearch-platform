package dao

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	researchQueuedTopic = "research.queued"
	idempotencyTTL      = 24 * time.Hour
)

var (
	ErrIdempotencyConflict   = errors.New("idempotency key was already used for a different request")
	ErrResearchQuotaExceeded = errors.New("research quota exceeded")
	errIdempotencyRace       = errors.New("concurrent idempotency claim")
)

type ResearchSubmissionCommand struct {
	UserID         string
	Endpoint       string
	IdempotencyKey string
	Query          string
	ResearchType   string
	Metadata       datatypes.JSON
}

type ResearchSubmissionResult struct {
	HTTPStatus int
	Body       []byte
	SessionID  string
	Replayed   bool
}

type ResearchSubmissionDAO struct {
	db  *gorm.DB
	now func() time.Time
}

func NewResearchSubmissionDAO(db *gorm.DB) *ResearchSubmissionDAO {
	return &ResearchSubmissionDAO{db: db, now: time.Now}
}

func CanonicalResearchRequestHash(cmd ResearchSubmissionCommand) (string, error) {
	var metadata interface{} = map[string]interface{}{}
	if len(cmd.Metadata) > 0 && string(cmd.Metadata) != "null" {
		if err := json.Unmarshal(cmd.Metadata, &metadata); err != nil {
			return "", fmt.Errorf("canonicalize research metadata: %w", err)
		}
	}
	canonical, err := json.Marshal(struct {
		Query        string      `json:"query"`
		ResearchType string      `json:"research_type"`
		Metadata     interface{} `json:"metadata"`
	}{
		Query:        cmd.Query,
		ResearchType: cmd.ResearchType,
		Metadata:     metadata,
	})
	if err != nil {
		return "", fmt.Errorf("marshal canonical research request: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func classifyIdempotencyRecord(record *model.IdempotencyRecord, requestHash string) (*ResearchSubmissionResult, error) {
	if record.RequestHash != requestHash {
		return nil, ErrIdempotencyConflict
	}
	return &ResearchSubmissionResult{
		HTTPStatus: record.HTTPStatus,
		Body:       append([]byte(nil), record.ResponseBody...),
		SessionID:  record.ResourceID,
		Replayed:   true,
	}, nil
}

func (d *ResearchSubmissionDAO) Submit(ctx context.Context, cmd ResearchSubmissionCommand) (*ResearchSubmissionResult, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("research submission database is not configured")
	}
	cmd.UserID = strings.TrimSpace(cmd.UserID)
	cmd.Endpoint = strings.TrimSpace(cmd.Endpoint)
	cmd.IdempotencyKey = strings.TrimSpace(cmd.IdempotencyKey)
	if cmd.UserID == "" || cmd.Endpoint == "" || cmd.IdempotencyKey == "" {
		return nil, errors.New("research submission identity is incomplete")
	}
	requestHash, err := CanonicalResearchRequestHash(cmd)
	if err != nil {
		return nil, err
	}

	result, err := d.submitOnce(ctx, cmd, requestHash)
	if !errors.Is(err, errIdempotencyRace) {
		return result, err
	}

	var record model.IdempotencyRecord
	findErr := d.db.WithContext(ctx).
		Where("user_id = ? AND endpoint = ? AND idempotency_key = ?", cmd.UserID, cmd.Endpoint, cmd.IdempotencyKey).
		First(&record).Error
	if findErr != nil {
		return nil, fmt.Errorf("load concurrent idempotency claim: %w", findErr)
	}
	return classifyIdempotencyRecord(&record, requestHash)
}

func (d *ResearchSubmissionDAO) submitOnce(
	ctx context.Context,
	cmd ResearchSubmissionCommand,
	requestHash string,
) (*ResearchSubmissionResult, error) {
	now := d.now().UTC()
	sessionID := uuid.NewString()
	body, err := json.Marshal(map[string]interface{}{
		"success":    true,
		"session_id": sessionID,
		"message":    "研究任务已排队",
	})
	if err != nil {
		return nil, fmt.Errorf("marshal research submission response: %w", err)
	}
	result := &ResearchSubmissionResult{
		HTTPStatus: http.StatusCreated,
		Body:       body,
		SessionID:  sessionID,
	}

	err = d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.IdempotencyRecord
		findErr := tx.
			Where("user_id = ? AND endpoint = ? AND idempotency_key = ?", cmd.UserID, cmd.Endpoint, cmd.IdempotencyKey).
			First(&existing).Error
		switch {
		case findErr == nil:
			replayed, classifyErr := classifyIdempotencyRecord(&existing, requestHash)
			if classifyErr != nil {
				return classifyErr
			}
			*result = *replayed
			return nil
		case !errors.Is(findErr, gorm.ErrRecordNotFound):
			return fmt.Errorf("lookup idempotency record: %w", findErr)
		}

		record := &model.IdempotencyRecord{
			UserID:         cmd.UserID,
			Endpoint:       cmd.Endpoint,
			IdempotencyKey: cmd.IdempotencyKey,
			RequestHash:    requestHash,
			State:          model.IdempotencyCompleted,
			ResourceID:     sessionID,
			HTTPStatus:     result.HTTPStatus,
			ResponseBody:   datatypes.JSON(body),
			CreatedAt:      now,
			ExpiresAt:      now.Add(idempotencyTTL),
		}
		if createErr := tx.Create(record).Error; createErr != nil {
			if isConstraintViolation(createErr, "idx_idempotency_scope") {
				return errIdempotencyRace
			}
			return fmt.Errorf("claim idempotency key: %w", createErr)
		}

		reservation, _, _, reserveErr := reserveResearchQuotaTx(tx, cmd.UserID, sessionID, now)
		if reserveErr != nil {
			return reserveErr
		}
		session := &model.ResearchSession{
			ID:           sessionID,
			UserID:       cmd.UserID,
			Query:        cmd.Query,
			Status:       "planning",
			Progress:     0,
			ResearchType: cmd.ResearchType,
			Metadata:     cmd.Metadata,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if createErr := tx.Create(session).Error; createErr != nil {
			return fmt.Errorf("create research session: %w", createErr)
		}
		job := &model.ResearchJob{
			ResearchID:  sessionID,
			Status:      model.ResearchJobQueued,
			MaxAttempts: 3,
			AvailableAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if createErr := tx.Create(job).Error; createErr != nil {
			return fmt.Errorf("create research job: %w", createErr)
		}
		payload, marshalErr := json.Marshal(map[string]interface{}{
			"job_id":         job.ID,
			"research_id":    sessionID,
			"reservation_id": reservation.ID,
		})
		if marshalErr != nil {
			return fmt.Errorf("marshal research queued event: %w", marshalErr)
		}
		event := &model.OutboxEvent{
			AggregateType: "research",
			AggregateID:   sessionID,
			Topic:         researchQueuedTopic,
			Payload:       datatypes.JSON(payload),
			CreatedAt:     now,
		}
		if createErr := tx.Create(event).Error; createErr != nil {
			return fmt.Errorf("create research queued outbox event: %w", createErr)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func reserveResearchQuotaTx(
	tx *gorm.DB,
	userID, researchID string,
	now time.Time,
) (*model.ResearchQuotaReservation, int, int, error) {
	membership, counterName, remaining, limit, err := deductResearchQuotaCounterTx(tx, userID, now)
	if err != nil {
		return nil, remaining, limit, err
	}

	reservation := &model.ResearchQuotaReservation{
		UserID:         userID,
		ResearchID:     researchID,
		MembershipType: string(membership.MembershipType),
		CounterName:    counterName,
		Units:          1,
		Status:         model.ReservationReserved,
		ReservedAt:     now,
	}
	if err := tx.Create(reservation).Error; err != nil {
		return nil, 0, 0, fmt.Errorf("create research quota reservation: %w", err)
	}
	return reservation, remaining, limit, nil
}

func deductResearchQuotaCounterTx(
	tx *gorm.DB,
	userID string,
	now time.Time,
) (*model.UserMembership, string, int, int, error) {
	defaultMembership := model.UserMembership{
		UserID:          userID,
		MembershipType:  model.MembershipFree,
		NormalChatLimit: 10,
		ResearchLimit:   1,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&defaultMembership).Error; err != nil {
		return nil, "", 0, 0, fmt.Errorf("ensure membership: %w", err)
	}

	var membership model.UserMembership
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&membership).Error; err != nil {
		return nil, "", 0, 0, fmt.Errorf("lock membership: %w", err)
	}

	counterName := "research_used"
	limit := membership.ResearchLimit
	used := membership.ResearchUsed
	if membership.MembershipType == model.MembershipPremium {
		if membership.PremiumResetAt != nil && now.After(*membership.PremiumResetAt) {
			membership.PremiumChatUsed = 0
			membership.PremiumResearchUsed = 0
			resetAt := now.Add(5 * time.Hour)
			membership.PremiumResetAt = &resetAt
		}
		counterName = "premium_research_used"
		limit = membership.PremiumResearchLimit
		used = membership.PremiumResearchUsed
	}
	if used >= limit {
		return &membership, counterName, 0, limit, ErrResearchQuotaExceeded
	}
	used++
	if counterName == "premium_research_used" {
		membership.PremiumResearchUsed = used
	} else {
		membership.ResearchUsed = used
	}
	if err := tx.Save(&membership).Error; err != nil {
		return nil, "", 0, 0, fmt.Errorf("reserve research quota counter: %w", err)
	}
	return &membership, counterName, limit - used, limit, nil
}

func isConstraintViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && (constraint == "" || pgErr.ConstraintName == constraint)
	}
	return false
}

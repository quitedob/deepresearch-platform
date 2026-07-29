package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	IdempotencyCompleted = "completed"

	ReservationReserved = "reserved"
	ReservationConsumed = "consumed"
	ReservationReleased = "released"

	ResearchJobQueued    = "queued"
	ResearchJobRunning   = "running"
	ResearchJobSucceeded = "succeeded"
	ResearchJobFailed    = "failed"
	ResearchJobCancelled = "cancelled"
)

// IdempotencyRecord stores the committed response for one client operation.
type IdempotencyRecord struct {
	ID             string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID         string         `gorm:"not null;type:uuid;uniqueIndex:idx_idempotency_scope" json:"user_id"`
	Endpoint       string         `gorm:"not null;uniqueIndex:idx_idempotency_scope" json:"endpoint"`
	IdempotencyKey string         `gorm:"not null;uniqueIndex:idx_idempotency_scope" json:"idempotency_key"`
	RequestHash    string         `gorm:"not null" json:"request_hash"`
	State          string         `gorm:"not null" json:"state"`
	ResourceID     string         `gorm:"not null;type:uuid" json:"resource_id"`
	HTTPStatus     int            `gorm:"not null" json:"http_status"`
	ResponseBody   datatypes.JSON `gorm:"type:jsonb;not null" json:"response_body"`
	CreatedAt      time.Time      `json:"created_at"`
	ExpiresAt      time.Time      `gorm:"index" json:"expires_at"`
}

func (IdempotencyRecord) TableName() string {
	return "idempotency_records"
}

// ResearchQuotaReservation ties one quota unit to one accepted research job.
type ResearchQuotaReservation struct {
	ID             string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID         string     `gorm:"index;not null;type:uuid" json:"user_id"`
	ResearchID     string     `gorm:"not null;type:uuid;uniqueIndex" json:"research_id"`
	MembershipType string     `gorm:"not null" json:"membership_type"`
	CounterName    string     `gorm:"not null" json:"counter_name"`
	Units          int        `gorm:"not null;default:1" json:"units"`
	Status         string     `gorm:"not null;index" json:"status"`
	ReservedAt     time.Time  `json:"reserved_at"`
	ConsumedAt     *time.Time `json:"consumed_at,omitempty"`
	ReleasedAt     *time.Time `json:"released_at,omitempty"`
}

func (ResearchQuotaReservation) TableName() string {
	return "research_quota_reservations"
}

// ResearchJob is the durable execution record claimed by database workers.
type ResearchJob struct {
	ID          string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ResearchID  string     `gorm:"not null;type:uuid;uniqueIndex" json:"research_id"`
	Status      string     `gorm:"not null;index:idx_research_job_claim,priority:1" json:"status"`
	Attempts    int        `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts int        `gorm:"not null;default:3" json:"max_attempts"`
	AvailableAt time.Time  `gorm:"not null;index:idx_research_job_claim,priority:2" json:"available_at"`
	LeaseOwner  string     `json:"lease_owner,omitempty"`
	LeaseUntil  *time.Time `gorm:"index:idx_research_job_claim,priority:3" json:"lease_until,omitempty"`
	FenceToken  int64      `gorm:"not null;default:0" json:"fence_token"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	LastError   string     `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (ResearchJob) TableName() string {
	return "research_jobs"
}

// OutboxEvent records a durable integration event in the business transaction.
type OutboxEvent struct {
	ID            string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	AggregateType string         `gorm:"not null" json:"aggregate_type"`
	AggregateID   string         `gorm:"not null;type:uuid;uniqueIndex:idx_outbox_topic_aggregate" json:"aggregate_id"`
	Topic         string         `gorm:"not null;uniqueIndex:idx_outbox_topic_aggregate" json:"topic"`
	Payload       datatypes.JSON `gorm:"type:jsonb;not null" json:"payload"`
	CreatedAt     time.Time      `json:"created_at"`
	PublishedAt   *time.Time     `json:"published_at,omitempty"`
	Attempts      int            `gorm:"not null;default:0" json:"attempts"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}

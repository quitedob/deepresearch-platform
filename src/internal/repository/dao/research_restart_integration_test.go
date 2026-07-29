package dao_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/ai-research-platform/internal/service"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type restartResearchExecutor struct {
	db     *gorm.DB
	called chan struct{}
}

func (*restartResearchExecutor) CanExecute(string) error {
	return nil
}

func (executor *restartResearchExecutor) ExecuteResearchJob(
	ctx context.Context,
	sessionID, _, _ string,
) error {
	select {
	case <-executor.called:
	default:
		close(executor.called)
	}
	return executor.db.WithContext(ctx).
		Model(&model.ResearchSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{"status": "completed", "progress": 1, "updated_at": time.Now().UTC()}).
		Error
}

func TestCommittedResearchJobRunsAfterSubmissionProcessRestart(t *testing.T) {
	db := openRestartWorkflowTestDB(t)
	userID := uuid.NewString()
	if err := db.Create(&model.UserMembership{
		UserID:         userID,
		MembershipType: model.MembershipFree,
		ResearchLimit:  2,
	}).Error; err != nil {
		t.Fatal(err)
	}
	submission, err := dao.NewResearchSubmissionDAO(db).Submit(context.Background(), dao.ResearchSubmissionCommand{
		UserID:         userID,
		Endpoint:       "/api/v1/research/start",
		IdempotencyKey: "restart-" + uuid.NewString(),
		Query:          "Explain durable jobs",
		ResearchType:   "deep",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Recreate every runtime component after the submission has committed.
	executor := &restartResearchExecutor{db: db, called: make(chan struct{})}
	worker := service.NewResearchWorker(
		dao.NewResearchJobDAO(db),
		executor,
		service.WorkerConfig{
			Owner:         "worker-after-restart",
			PollInterval:  5 * time.Millisecond,
			LeaseDuration: time.Second,
		},
	)
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	workerDone := make(chan error, 1)
	go func() {
		workerDone <- worker.Start(workerCtx)
	}()
	t.Cleanup(func() {
		cancelWorker()
		<-workerDone
	})

	select {
	case <-executor.called:
	case <-time.After(3 * time.Second):
		t.Fatal("committed research job was not executed after restart")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		var job model.ResearchJob
		if err := db.Where("research_id = ?", submission.SessionID).First(&job).Error; err != nil {
			t.Fatal(err)
		}
		if job.Status == model.ResearchJobSucceeded {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job status = %q, want succeeded", job.Status)
		}
		time.Sleep(5 * time.Millisecond)
	}

	var reservation model.ResearchQuotaReservation
	if err := db.Where("research_id = ?", submission.SessionID).First(&reservation).Error; err != nil {
		t.Fatal(err)
	}
	if reservation.Status != model.ReservationConsumed {
		t.Fatalf("reservation status = %q, want consumed", reservation.Status)
	}
}

func openRestartWorkflowTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL restart integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var databaseName string
	if err := db.Raw("SELECT current_database()").Scan(&databaseName).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(databaseName, "_test") {
		t.Fatalf("refusing to reset non-test database %q", databaseName)
	}
	models := []interface{}{
		&model.OutboxEvent{},
		&model.ResearchJob{},
		&model.ResearchQuotaReservation{},
		&model.IdempotencyRecord{},
		&model.ResearchSession{},
		&model.UserMembership{},
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(
		"TRUNCATE TABLE outbox_events, research_jobs, research_quota_reservations, idempotency_records, research_sessions, user_memberships RESTART IDENTITY CASCADE",
	).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}

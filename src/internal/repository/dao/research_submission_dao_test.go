package dao

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCanonicalResearchRequestHashIgnoresMetadataKeyOrder(t *testing.T) {
	first := ResearchSubmissionCommand{
		Query:        "How do durable workers recover?",
		ResearchType: "deep",
		Metadata:     datatypes.JSON(`{"language":"en","options":{"sources":5,"citations":true}}`),
	}
	second := ResearchSubmissionCommand{
		Query:        "How do durable workers recover?",
		ResearchType: "deep",
		Metadata:     datatypes.JSON(`{"options":{"citations":true,"sources":5},"language":"en"}`),
	}

	firstHash, err := CanonicalResearchRequestHash(first)
	if err != nil {
		t.Fatal(err)
	}
	secondHash, err := CanonicalResearchRequestHash(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash != secondHash {
		t.Fatalf("equivalent requests produced different hashes: %q != %q", firstHash, secondHash)
	}
}

func TestCanonicalResearchRequestHashRejectsInvalidMetadata(t *testing.T) {
	_, err := CanonicalResearchRequestHash(ResearchSubmissionCommand{
		Query:        "query",
		ResearchType: "deep",
		Metadata:     datatypes.JSON(`{"broken"`),
	})
	if err == nil {
		t.Fatal("expected invalid metadata to fail canonicalization")
	}
}

func TestClassifyIdempotencyRecord(t *testing.T) {
	record := &model.IdempotencyRecord{
		RequestHash:  "same",
		ResourceID:   "session-1",
		HTTPStatus:   201,
		ResponseBody: datatypes.JSON(`{"success":true,"session_id":"session-1"}`),
	}

	result, err := classifyIdempotencyRecord(record, "same")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Replayed || result.SessionID != "session-1" || result.HTTPStatus != 201 {
		t.Fatalf("unexpected replay result: %+v", result)
	}

	if _, err := classifyIdempotencyRecord(record, "different"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestSubmitResearchIsIdempotentUnderConcurrency(t *testing.T) {
	db := openResearchWorkflowTestDB(t)
	userID := uuid.NewString()
	if err := db.Create(&model.UserMembership{
		UserID:          userID,
		MembershipType:  model.MembershipFree,
		NormalChatLimit: 10,
		ResearchLimit:   5,
	}).Error; err != nil {
		t.Fatal(err)
	}
	submissions := NewResearchSubmissionDAO(db)
	command := ResearchSubmissionCommand{
		UserID:         userID,
		Endpoint:       "/api/v1/research/start",
		IdempotencyKey: "concurrent-submission-key",
		Query:          "Explain durable worker leases",
		ResearchType:   "deep",
		Metadata:       datatypes.JSON(`{"options":{"language":"en"}}`),
	}

	const callers = 20
	results := make(chan *ResearchSubmissionResult, callers)
	failures := make(chan error, callers)
	var wait sync.WaitGroup
	for i := 0; i < callers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := submissions.Submit(context.Background(), command)
			if err != nil {
				failures <- err
				return
			}
			results <- result
		}()
	}
	wait.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Errorf("concurrent submit failed: %v", err)
	}

	var sessionID string
	created := 0
	for result := range results {
		if sessionID == "" {
			sessionID = result.SessionID
		}
		if result.SessionID != sessionID {
			t.Errorf("session ID = %q, want %q", result.SessionID, sessionID)
		}
		if !result.Replayed {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("new submissions = %d, want 1", created)
	}

	assertWorkflowCounts(t, db, userID, 1)
	var membership model.UserMembership
	if err := db.Where("user_id = ?", userID).First(&membership).Error; err != nil {
		t.Fatal(err)
	}
	if membership.ResearchUsed != 1 {
		t.Fatalf("research used = %d, want 1", membership.ResearchUsed)
	}
}

func TestSubmitResearchRejectsKeyReuseWithDifferentRequest(t *testing.T) {
	db := openResearchWorkflowTestDB(t)
	userID := uuid.NewString()
	submissions := NewResearchSubmissionDAO(db)
	command := ResearchSubmissionCommand{
		UserID:         userID,
		Endpoint:       "/api/v1/research/start",
		IdempotencyKey: "conflicting-submission-key",
		Query:          "first",
		ResearchType:   "deep",
	}
	if _, err := submissions.Submit(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	command.Query = "different"
	if _, err := submissions.Submit(context.Background(), command); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
	assertWorkflowCounts(t, db, userID, 1)
}

func TestSubmitResearchRollsBackWhenQuotaIsExhausted(t *testing.T) {
	db := openResearchWorkflowTestDB(t)
	userID := uuid.NewString()
	if err := db.Create(&model.UserMembership{
		UserID:         userID,
		MembershipType: model.MembershipFree,
		ResearchLimit:  1,
		ResearchUsed:   1,
	}).Error; err != nil {
		t.Fatal(err)
	}
	_, err := NewResearchSubmissionDAO(db).Submit(context.Background(), ResearchSubmissionCommand{
		UserID:         userID,
		Endpoint:       "/api/v1/research/start",
		IdempotencyKey: "quota-exhausted-key",
		Query:          "query",
		ResearchType:   "deep",
	})
	if !errors.Is(err, ErrResearchQuotaExceeded) {
		t.Fatalf("error = %v, want ErrResearchQuotaExceeded", err)
	}
	assertWorkflowCounts(t, db, userID, 0)
}

func TestSubmitResearchRollsBackInjectedCreateFailures(t *testing.T) {
	for _, table := range []string{"research_sessions", "research_jobs", "outbox_events"} {
		t.Run(table, func(t *testing.T) {
			db := openResearchWorkflowTestDB(t)
			userID := uuid.NewString()
			if err := db.Create(&model.UserMembership{
				UserID:         userID,
				MembershipType: model.MembershipFree,
				ResearchLimit:  2,
			}).Error; err != nil {
				t.Fatal(err)
			}
			callbackName := "test:fail-create-" + table
			if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
				if tx.Statement.Table == table {
					tx.AddError(fmt.Errorf("injected %s failure", table))
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = db.Callback().Create().Remove(callbackName)
			})

			_, err := NewResearchSubmissionDAO(db).Submit(context.Background(), ResearchSubmissionCommand{
				UserID:         userID,
				Endpoint:       "/api/v1/research/start",
				IdempotencyKey: "rollback-" + table,
				Query:          "query",
				ResearchType:   "deep",
			})
			if err == nil || !strings.Contains(err.Error(), "injected") {
				t.Fatalf("error = %v, want injected failure", err)
			}
			assertWorkflowCounts(t, db, userID, 0)

			var membership model.UserMembership
			if findErr := db.Where("user_id = ?", userID).First(&membership).Error; findErr != nil {
				t.Fatal(findErr)
			}
			if membership.ResearchUsed != 0 {
				t.Fatalf("research used = %d, want 0 after rollback", membership.ResearchUsed)
			}
		})
	}
}

func openResearchWorkflowTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL workflow integration tests")
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
	sqlDB.SetMaxOpenConns(30)
	sqlDB.SetConnMaxLifetime(time.Minute)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}

func assertWorkflowCounts(t *testing.T, db *gorm.DB, userID string, want int64) {
	t.Helper()
	for name, query := range map[string]*gorm.DB{
		"idempotency records": db.Model(&model.IdempotencyRecord{}).Where("user_id = ?", userID),
		"quota reservations":  db.Model(&model.ResearchQuotaReservation{}).Where("user_id = ?", userID),
		"research sessions":   db.Model(&model.ResearchSession{}).Where("user_id = ?", userID),
		"research jobs":       db.Model(&model.ResearchJob{}).Joins("JOIN research_sessions ON research_sessions.id = research_jobs.research_id").Where("research_sessions.user_id = ?", userID),
		"outbox events":       db.Model(&model.OutboxEvent{}).Joins("JOIN research_sessions ON research_sessions.id = outbox_events.aggregate_id").Where("research_sessions.user_id = ?", userID),
	} {
		var count int64
		if err := query.Count(&count).Error; err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", name, count, want)
		}
	}
}

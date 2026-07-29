package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

type fakeResearchJobStore struct {
	mu sync.Mutex

	job     *model.ResearchJob
	session *model.ResearchSession

	claimErr     error
	sessionErr   error
	heartbeatErr error
	heartbeatFn  func(context.Context) error
	consumeErr   error
	completeErr  error
	retryErr     error
	failErr      error

	claimed    int
	heartbeats int
	consumed   int
	completed  int
	retried    int
	failed     int
	released   bool
}

func (s *fakeResearchJobStore) Claim(context.Context, string, time.Time, time.Duration) (*model.ResearchJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.claimed++
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	if s.job == nil {
		return nil, gorm.ErrRecordNotFound
	}
	job := *s.job
	s.job = nil
	return &job, nil
}

func (s *fakeResearchJobStore) GetSession(context.Context, string) (*model.ResearchSession, error) {
	if s.sessionErr != nil {
		return nil, s.sessionErr
	}
	if s.session == nil {
		return nil, gorm.ErrRecordNotFound
	}
	session := *s.session
	return &session, nil
}

func (s *fakeResearchJobStore) Heartbeat(ctx context.Context, _ dao.JobLease, _ time.Time) error {
	s.mu.Lock()
	s.heartbeats++
	fn := s.heartbeatFn
	s.mu.Unlock()
	if fn != nil {
		return fn(ctx)
	}
	return s.heartbeatErr
}

func (s *fakeResearchJobStore) ConsumeReservation(context.Context, dao.JobLease, time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consumed++
	return s.consumeErr
}

func (s *fakeResearchJobStore) Complete(context.Context, dao.JobLease, time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completed++
	return s.completeErr
}

func (s *fakeResearchJobStore) Retry(context.Context, dao.JobLease, time.Time, error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retried++
	return s.retryErr
}

func (s *fakeResearchJobStore) Fail(_ context.Context, _ dao.JobLease, _ time.Time, _ error, releaseReservation bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed++
	s.released = releaseReservation
	return s.failErr
}

type fakeResearchExecutor struct {
	canExecuteErr error
	executeErr    error
	panicValue    interface{}
	executeFn     func(context.Context) error
	calls         int
}

func (e *fakeResearchExecutor) CanExecute(string) error {
	return e.canExecuteErr
}

func (e *fakeResearchExecutor) ExecuteResearchJob(ctx context.Context, _, _, _ string) error {
	e.calls++
	if e.panicValue != nil {
		panic(e.panicValue)
	}
	if e.executeFn != nil {
		return e.executeFn(ctx)
	}
	return e.executeErr
}

func TestResearchWorkerConsumesExecutesAndCompletes(t *testing.T) {
	store := newFakeWorkerStore(1, 3)
	executor := &fakeResearchExecutor{}
	worker := NewResearchWorker(store, executor, WorkerConfig{
		Owner:         "worker-a",
		PollInterval:  time.Millisecond,
		LeaseDuration: time.Minute,
		Now:           fixedWorkerClock(),
	})

	worked, err := worker.runOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !worked || store.consumed != 1 || executor.calls != 1 || store.completed != 1 {
		t.Fatalf(
			"worked=%v consumed=%d executed=%d completed=%d",
			worked,
			store.consumed,
			executor.calls,
			store.completed,
		)
	}
	if store.retried != 0 || store.failed != 0 {
		t.Fatalf("retried=%d failed=%d, want zero", store.retried, store.failed)
	}
}

func TestResearchWorkerRetriesPreflightFailureWithoutConsumingQuota(t *testing.T) {
	store := newFakeWorkerStore(1, 3)
	executor := &fakeResearchExecutor{canExecuteErr: errors.New("engine unavailable")}
	worker := NewResearchWorker(store, executor, WorkerConfig{Owner: "worker-a", Now: fixedWorkerClock()})

	worked, err := worker.runOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !worked || store.retried != 1 || store.consumed != 0 || executor.calls != 0 {
		t.Fatalf(
			"worked=%v retried=%d consumed=%d executed=%d",
			worked,
			store.retried,
			store.consumed,
			executor.calls,
		)
	}
}

func TestResearchWorkerReleasesReservedQuotaAfterFinalPreflightFailure(t *testing.T) {
	store := newFakeWorkerStore(3, 3)
	executor := &fakeResearchExecutor{canExecuteErr: errors.New("engine unavailable")}
	worker := NewResearchWorker(store, executor, WorkerConfig{Owner: "worker-a", Now: fixedWorkerClock()})

	if _, err := worker.runOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.failed != 1 || !store.released || store.retried != 0 || store.consumed != 0 {
		t.Fatalf(
			"failed=%d released=%v retried=%d consumed=%d",
			store.failed,
			store.released,
			store.retried,
			store.consumed,
		)
	}
}

func TestResearchWorkerFinalizesAnExpiredLeaseBeyondMaxAttempts(t *testing.T) {
	store := newFakeWorkerStore(4, 3)
	executor := &fakeResearchExecutor{}
	worker := NewResearchWorker(store, executor, WorkerConfig{Owner: "worker-a", Now: fixedWorkerClock()})

	if _, err := worker.runOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.failed != 1 || !store.released || store.consumed != 0 || executor.calls != 0 {
		t.Fatalf(
			"failed=%d released=%v consumed=%d executed=%d, want 1/true/0/0",
			store.failed,
			store.released,
			store.consumed,
			executor.calls,
		)
	}
}

func TestResearchWorkerKeepsConsumedQuotaAfterExecutionFailure(t *testing.T) {
	store := newFakeWorkerStore(3, 3)
	executor := &fakeResearchExecutor{executeErr: errors.New("model failed")}
	worker := NewResearchWorker(store, executor, WorkerConfig{Owner: "worker-a", Now: fixedWorkerClock()})

	if _, err := worker.runOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.consumed != 1 || store.failed != 1 || store.released {
		t.Fatalf("consumed=%d failed=%d released=%v", store.consumed, store.failed, store.released)
	}
}

func TestResearchWorkerRecoversExecutorPanic(t *testing.T) {
	store := newFakeWorkerStore(3, 3)
	executor := &fakeResearchExecutor{panicValue: "boom"}
	worker := NewResearchWorker(store, executor, WorkerConfig{Owner: "worker-a", Now: fixedWorkerClock()})

	if _, err := worker.runOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.failed != 1 || store.released {
		t.Fatalf("failed=%d released=%v", store.failed, store.released)
	}
}

func TestResearchWorkerReturnsIdleForEmptyQueue(t *testing.T) {
	store := &fakeResearchJobStore{claimErr: gorm.ErrRecordNotFound}
	worker := NewResearchWorker(store, &fakeResearchExecutor{}, WorkerConfig{Owner: "worker-a", Now: fixedWorkerClock()})

	worked, err := worker.runOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if worked {
		t.Fatal("empty queue reported work")
	}
}

func TestResearchWorkerIgnoresHeartbeatCancellationDuringSuccessfulShutdown(t *testing.T) {
	store := newFakeWorkerStore(1, 3)
	store.heartbeatFn = func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	executor := &fakeResearchExecutor{executeFn: func(context.Context) error {
		time.Sleep(110 * time.Millisecond)
		return nil
	}}
	worker := NewResearchWorker(store, executor, WorkerConfig{
		Owner:         "worker-a",
		LeaseDuration: 300 * time.Millisecond,
		Now:           fixedWorkerClock(),
	})

	if _, err := worker.runOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.completed != 1 || store.retried != 0 || store.failed != 0 {
		t.Fatalf(
			"completed=%d retried=%d failed=%d, want 1/0/0",
			store.completed,
			store.retried,
			store.failed,
		)
	}
}

func newFakeWorkerStore(attempts, maxAttempts int) *fakeResearchJobStore {
	leaseUntil := time.Date(2026, time.July, 29, 12, 1, 0, 0, time.UTC)
	return &fakeResearchJobStore{
		job: &model.ResearchJob{
			ID:          "job-1",
			ResearchID:  "research-1",
			Status:      model.ResearchJobRunning,
			Attempts:    attempts,
			MaxAttempts: maxAttempts,
			LeaseOwner:  "worker-a",
			LeaseUntil:  &leaseUntil,
			FenceToken:  1,
		},
		session: &model.ResearchSession{
			ID:           "research-1",
			Query:        "query",
			ResearchType: "deep",
		},
	}
}

func fixedWorkerClock() func() time.Time {
	return func() time.Time {
		return time.Date(2026, time.July, 29, 12, 0, 0, 0, time.UTC)
	}
}

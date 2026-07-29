package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ai-research-platform/internal/logger"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ResearchExecutor interface {
	CanExecute(researchType string) error
	ExecuteResearchJob(ctx context.Context, sessionID, query, researchType string) error
}

type ResearchJobStore interface {
	Claim(context.Context, string, time.Time, time.Duration) (*model.ResearchJob, error)
	GetSession(context.Context, string) (*model.ResearchSession, error)
	Heartbeat(context.Context, dao.JobLease, time.Time) error
	ConsumeReservation(context.Context, dao.JobLease, time.Time) error
	Complete(context.Context, dao.JobLease, time.Time) error
	Retry(context.Context, dao.JobLease, time.Time, error) error
	Fail(context.Context, dao.JobLease, time.Time, error, bool) error
}

type WorkerConfig struct {
	Owner         string
	PollInterval  time.Duration
	LeaseDuration time.Duration
	Now           func() time.Time
}

type ResearchWorker struct {
	store    ResearchJobStore
	executor ResearchExecutor
	config   WorkerConfig
}

func NewResearchWorker(store ResearchJobStore, executor ResearchExecutor, config WorkerConfig) *ResearchWorker {
	if config.Owner == "" {
		config.Owner = "research-worker-" + uuid.NewString()
	}
	if config.PollInterval <= 0 {
		config.PollInterval = time.Second
	}
	if config.LeaseDuration <= 0 {
		config.LeaseDuration = 30 * time.Second
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &ResearchWorker{store: store, executor: executor, config: config}
}

func (w *ResearchWorker) Start(ctx context.Context) error {
	if w == nil || w.store == nil || w.executor == nil {
		return errors.New("research worker dependencies are not configured")
	}
	for {
		worked, err := w.runOnce(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("research worker iteration failed", zap.Error(err))
		}
		if ctx.Err() != nil {
			return nil
		}
		if worked && err == nil {
			continue
		}
		timer := time.NewTimer(w.config.PollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}

func (w *ResearchWorker) runOnce(ctx context.Context) (bool, error) {
	if w == nil || w.store == nil || w.executor == nil {
		return false, errors.New("research worker dependencies are not configured")
	}
	now := w.config.Now().UTC()
	job, err := w.store.Claim(ctx, w.config.Owner, now, w.config.LeaseDuration)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim research job: %w", err)
	}
	lease := dao.LeaseFromJob(job)
	if job.Attempts > job.MaxAttempts {
		if err := w.store.Fail(ctx, lease, now, dao.ErrJobAttemptsExhausted, true); err != nil {
			if errors.Is(err, dao.ErrLeaseLost) {
				return true, nil
			}
			return true, fmt.Errorf("finalize exhausted research job: %w", err)
		}
		return true, nil
	}

	session, err := w.store.GetSession(ctx, job.ResearchID)
	if err != nil {
		return true, w.handleFailure(ctx, job, lease, fmt.Errorf("load research session: %w", err), true)
	}
	if err := w.executor.CanExecute(session.ResearchType); err != nil {
		return true, w.handleFailure(ctx, job, lease, fmt.Errorf("research preflight: %w", err), true)
	}
	if err := w.store.ConsumeReservation(ctx, lease, now); err != nil {
		return true, w.handleFailure(ctx, job, lease, fmt.Errorf("consume research quota reservation: %w", err), true)
	}

	executionCtx, cancelExecution := context.WithCancel(ctx)
	heartbeatDone := make(chan error, 1)
	go w.heartbeat(executionCtx, cancelExecution, lease, heartbeatDone)
	executionErr := executeResearchSafely(
		executionCtx,
		w.executor,
		session.ID,
		session.Query,
		session.ResearchType,
	)
	cancelExecution()
	heartbeatErr := <-heartbeatDone
	if errors.Is(heartbeatErr, dao.ErrLeaseLost) {
		return true, nil
	}
	if heartbeatErr != nil && executionErr == nil {
		executionErr = heartbeatErr
	}
	if executionErr != nil {
		return true, w.handleFailure(ctx, job, lease, executionErr, false)
	}
	if err := w.store.Complete(ctx, lease, w.config.Now().UTC()); err != nil {
		if errors.Is(err, dao.ErrLeaseLost) {
			return true, nil
		}
		return true, fmt.Errorf("complete research job: %w", err)
	}
	return true, nil
}

func (w *ResearchWorker) heartbeat(
	ctx context.Context,
	cancelExecution context.CancelFunc,
	lease dao.JobLease,
	done chan<- error,
) {
	interval := w.config.LeaseDuration / 3
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			done <- nil
			return
		case <-ticker.C:
			until := w.config.Now().UTC().Add(w.config.LeaseDuration)
			if err := w.store.Heartbeat(ctx, lease, until); err != nil {
				if ctx.Err() != nil {
					done <- nil
					return
				}
				cancelExecution()
				done <- err
				return
			}
		}
	}
}

func (w *ResearchWorker) handleFailure(
	ctx context.Context,
	job *model.ResearchJob,
	lease dao.JobLease,
	cause error,
	releaseReservation bool,
) error {
	if errors.Is(cause, dao.ErrLeaseLost) {
		return nil
	}
	now := w.config.Now().UTC()
	if job.Attempts < job.MaxAttempts {
		if err := w.store.Retry(ctx, lease, now.Add(researchRetryDelay(job.Attempts)), cause); err != nil {
			if errors.Is(err, dao.ErrLeaseLost) {
				return nil
			}
			return fmt.Errorf("retry research job: %w", err)
		}
		return nil
	}
	if err := w.store.Fail(ctx, lease, now, cause, releaseReservation); err != nil {
		if errors.Is(err, dao.ErrLeaseLost) {
			return nil
		}
		return fmt.Errorf("fail research job: %w", err)
	}
	return nil
}

func executeResearchSafely(
	ctx context.Context,
	executor ResearchExecutor,
	sessionID, query, researchType string,
) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("research executor panicked: %v", recovered)
		}
	}()
	return executor.ExecuteResearchJob(ctx, sessionID, query, researchType)
}

func researchRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	exponent := min(attempt-1, 6)
	delay := time.Second * time.Duration(1<<exponent)
	if delay > time.Minute {
		return time.Minute
	}
	return delay
}

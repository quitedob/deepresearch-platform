# Durable Research Submission Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make an accepted research submission idempotent, quota-safe, and recoverable after API-process failure by committing its session, job, quota reservation, and outbox event in one PostgreSQL transaction and executing it through a leased worker.

**Architecture:** A new submission DAO owns the single PostgreSQL transaction and returns either a newly created response, an idempotent replay, quota exhaustion, or a key conflict. A job DAO implements `FOR UPDATE SKIP LOCKED` claims, leases, heartbeats, retries, and fence-token-guarded terminal transitions. A single-concurrency worker calls a synchronous `ResearchService` entry point; the HTTP handler never starts a goroutine.

**Tech Stack:** Go 1.21, Gin, GORM 1.24, PostgreSQL, Vue 3/Axios, existing Eino research agents.

## Global Constraints

- Preserve every pre-existing dirty-tree change; do not reset, overwrite, or stage unrelated hunks.
- PostgreSQL is the correctness target; lease claims must use `FOR UPDATE SKIP LOCKED`.
- `(user_id, endpoint, idempotency_key)` is unique, and a key reused with a different canonical request hash returns `409`.
- Quota reservation, session, job, outbox, and stored idempotent response commit or roll back together.
- A successful HTTP response never directly launches research execution.
- Worker updates require the current `(job_id, lease_owner, fence_token)`.
- Run one research execution at a time until Eino callback state becomes per-execution.
- Keep the current API response envelope; RFC 9457 is outside this slice.
- Implementation commits are optional because several target files contain pre-existing user changes; never commit those changes accidentally.

## File Structure

- `src/internal/repository/model/research_workflow.go`: durable workflow models and state constants.
- `src/internal/repository/dao/research_submission_dao.go`: canonical request hashing, idempotency lookup, quota reservation, and the submission transaction.
- `src/internal/repository/dao/research_job_dao.go`: job claim, heartbeat, retry, completion, failure, and reservation consume/release operations.
- `src/internal/service/research_worker.go`: polling, heartbeat, execution, backoff, and shutdown lifecycle.
- `src/internal/api/v1/research.go`: required idempotency header and submission result mapping.
- `src/internal/service/research_service.go`: synchronous execution entry point used by the worker.
- `src/cmd/server/main.go`: construct the DAOs/worker, start after wiring, stop before closing the database.
- `src/internal/database/migration.go`: register workflow tables and required columns.
- `vue/src/api/index.js`: attach one stable idempotency key to each research-start request.
- `src/internal/repository/dao/*_test.go`, `src/internal/service/research_worker_test.go`, `src/internal/api/v1/research_submission_test.go`: unit and PostgreSQL integration coverage.

---

### Task 1: Durable Workflow Models and Migration Registration

**Files:**
- Create: `src/internal/repository/model/research_workflow.go`
- Modify: `src/internal/database/migration.go`
- Test: `src/internal/repository/model/research_workflow_test.go`

**Interfaces:**
- Produces: `IdempotencyRecord`, `ResearchQuotaReservation`, `ResearchJob`, `OutboxEvent`.
- Produces constants for reservation states and job states used by all later tasks.

- [ ] **Step 1: Write the model contract test**

```go
func TestResearchWorkflowTableNames(t *testing.T) {
	tests := map[string]string{
		(model.IdempotencyRecord{}).TableName():         "idempotency_records",
		(model.ResearchQuotaReservation{}).TableName():  "research_quota_reservations",
		(model.ResearchJob{}).TableName():               "research_jobs",
		(model.OutboxEvent{}).TableName():               "outbox_events",
	}
	for got, want := range tests {
		if got != want {
			t.Fatalf("table name = %q, want %q", got, want)
		}
	}
}
```

- [ ] **Step 2: Run the test and verify RED**

Run: `go -C src/internal test ./repository/model -run TestResearchWorkflowTableNames -count=1`

Expected: compilation fails because the four model types do not exist.

- [ ] **Step 3: Add the exact workflow models**

```go
type IdempotencyRecord struct {
	ID             string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID         string         `gorm:"not null;type:uuid;uniqueIndex:idx_idempotency_scope"`
	Endpoint       string         `gorm:"not null;uniqueIndex:idx_idempotency_scope"`
	IdempotencyKey string         `gorm:"not null;uniqueIndex:idx_idempotency_scope"`
	RequestHash    string         `gorm:"not null"`
	State          string         `gorm:"not null"`
	ResourceID     string         `gorm:"not null;type:uuid"`
	HTTPStatus     int            `gorm:"not null"`
	ResponseBody   datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt      time.Time
	ExpiresAt      time.Time      `gorm:"index"`
}

type ResearchQuotaReservation struct {
	ID             string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID         string     `gorm:"index;not null;type:uuid"`
	ResearchID     string     `gorm:"not null;type:uuid;uniqueIndex"`
	MembershipType string     `gorm:"not null"`
	CounterName    string     `gorm:"not null"`
	Units          int        `gorm:"not null;default:1"`
	Status         string     `gorm:"not null;index"`
	ReservedAt     time.Time
	ConsumedAt     *time.Time
	ReleasedAt     *time.Time
}

type ResearchJob struct {
	ID          string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ResearchID  string     `gorm:"not null;type:uuid;uniqueIndex"`
	Status      string     `gorm:"not null;index:idx_research_job_claim"`
	Attempts    int        `gorm:"not null;default:0"`
	MaxAttempts int        `gorm:"not null;default:3"`
	AvailableAt time.Time  `gorm:"not null;index:idx_research_job_claim"`
	LeaseOwner  string
	LeaseUntil  *time.Time `gorm:"index:idx_research_job_claim"`
	FenceToken  int64      `gorm:"not null;default:0"`
	StartedAt   *time.Time
	CompletedAt *time.Time
	LastError   string     `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OutboxEvent struct {
	ID            string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AggregateType string         `gorm:"not null"`
	AggregateID   string         `gorm:"not null;type:uuid;uniqueIndex:idx_outbox_topic_aggregate"`
	Topic         string         `gorm:"not null;uniqueIndex:idx_outbox_topic_aggregate"`
	Payload       datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt     time.Time
	PublishedAt   *time.Time
	Attempts      int            `gorm:"not null;default:0"`
}
```

Add table-name methods and these constants: `ReservationReserved`, `ReservationConsumed`, `ReservationReleased`, `ResearchJobQueued`, `ResearchJobRunning`, `ResearchJobSucceeded`, `ResearchJobFailed`, and `ResearchJobCancelled`.

Register all four models in `AllModels`, `RequiredTables`, and `TableColumnRequirements`.

- [ ] **Step 4: Run model and migration package tests**

Run: `go -C src/internal test ./repository/model ./database -count=1`

Expected: the model test passes; the database package may still expose its pre-existing unused-import build failure, which Task 7 removes.

---

### Task 2: Atomic Idempotent Submission and Quota Reservation

**Files:**
- Create: `src/internal/repository/dao/research_submission_dao.go`
- Create: `src/internal/repository/dao/research_submission_dao_test.go`
- Modify: `src/internal/repository/dao/membership_dao.go`

**Interfaces:**
- Produces:

```go
type ResearchSubmissionCommand struct {
	UserID, Endpoint, IdempotencyKey string
	Query, ResearchType              string
	Metadata                         datatypes.JSON
}

type ResearchSubmissionResult struct {
	HTTPStatus int
	Body       []byte
	SessionID  string
	Replayed   bool
}

var ErrIdempotencyConflict error
var ErrResearchQuotaExceeded error

func CanonicalResearchRequestHash(cmd ResearchSubmissionCommand) string
func (d *ResearchSubmissionDAO) Submit(ctx context.Context, cmd ResearchSubmissionCommand) (*ResearchSubmissionResult, error)
```

- Consumes: workflow models and the existing `UserMembership` counters.

- [ ] **Step 1: Write canonical-hash and replay-decision tests**

```go
func TestCanonicalResearchRequestHashIgnoresMapKeyOrder(t *testing.T) {
	a := ResearchSubmissionCommand{Query: "q", ResearchType: "deep", Metadata: datatypes.JSON(`{"a":1,"b":2}`)}
	b := ResearchSubmissionCommand{Query: "q", ResearchType: "deep", Metadata: datatypes.JSON(`{"b":2,"a":1}`)}
	if CanonicalResearchRequestHash(a) != CanonicalResearchRequestHash(b) {
		t.Fatal("equivalent requests produced different hashes")
	}
}

func TestClassifyIdempotencyRecord(t *testing.T) {
	record := &model.IdempotencyRecord{RequestHash: "same", HTTPStatus: 201, ResponseBody: datatypes.JSON(`{"session_id":"s"}`)}
	if _, err := classifyIdempotencyRecord(record, "different"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `go -C src/internal test ./repository/dao -run 'TestCanonicalResearchRequestHash|TestClassifyIdempotencyRecord' -count=1`

Expected: compilation fails because the submission types/functions do not exist.

- [ ] **Step 3: Refactor quota mutation for transaction reuse**

Extract the body of `CheckAndDeductResearchQuota` into:

```go
func reserveResearchQuota(
	tx *gorm.DB,
	userID, researchID string,
	now time.Time,
) (*model.ResearchQuotaReservation, int, int, error)
```

The helper locks the membership row, applies the current free/premium reset
rules, increments the correct counter, and creates exactly one `reserved`
reservation. `CheckAndDeductResearchQuota` continues to work by calling the
shared counter logic without a research reservation so existing chat/research
callers are not silently changed.

- [ ] **Step 4: Implement canonical hashing and the single transaction**

`Submit` must precompute a UUID session ID and the exact `201` response body,
then transactionally:

```go
return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	if existing, err := findIdempotencyRecord(tx, cmd); err == nil {
		result, err = classifyIdempotencyRecord(existing, requestHash)
		return err
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := tx.Create(idempotencyRecord).Error; err != nil {
		return classifyUniqueViolation(err)
	}
	if _, _, _, err := reserveResearchQuota(tx, cmd.UserID, sessionID, now); err != nil {
		return err
	}
	if err := tx.Create(session).Error; err != nil { return err }
	if err := tx.Create(job).Error; err != nil { return err }
	return tx.Create(outboxEvent).Error
})
```

When the unique idempotency insert loses a concurrent race, let the transaction
roll back and read the committed winner in a fresh query before returning replay
or conflict.

- [ ] **Step 5: Add PostgreSQL integration tests**

Use `TEST_DATABASE_URL`; skip with an explicit message only when it is absent.
Migrate just the workflow, membership, and research-session models into an
isolated schema. Add subtests for:

- 20 concurrent identical submissions produce one session/job/reservation/outbox
  and increment the applicable counter once;
- same key/different request returns `ErrIdempotencyConflict`;
- quota exhaustion creates no workflow rows;
- a GORM create callback injected on session, job, and outbox creation rolls
  back the idempotency row, reservation, counter, and earlier inserts.

- [ ] **Step 6: Run DAO tests**

Run: `go -C src/internal test ./repository/dao -run 'TestCanonical|TestClassify|TestSubmitResearch' -count=1`

Expected: unit tests pass; PostgreSQL cases pass when `TEST_DATABASE_URL` is set or report a clear skip otherwise.

---

### Task 3: HTTP Idempotency Contract and Frontend Header

**Files:**
- Modify: `src/internal/api/v1/research.go`
- Create: `src/internal/api/v1/research_submission_test.go`
- Modify: `vue/src/api/index.js`

**Interfaces:**
- Consumes: `ResearchSubmissionDAO.Submit`.
- Produces: required `Idempotency-Key` request header and unchanged `201` body shape.

- [ ] **Step 1: Write handler tests with a fake submission store**

```go
func TestStartResearchRequiresIdempotencyKey(t *testing.T) {
	api := newResearchAPIForSubmissionTest(&fakeSubmissionStore{})
	response := performStartResearch(api, "", `{"query":"q","research_type":"deep"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestStartResearchDoesNotDispatchDirectly(t *testing.T) {
	store := &fakeSubmissionStore{result: &dao.ResearchSubmissionResult{
		HTTPStatus: http.StatusCreated,
		Body: []byte(`{"success":true,"session_id":"session-1","message":"研究任务已排队"}`),
	}}
	response := performStartResearch(newResearchAPIForSubmissionTest(store), "key-1", validBody)
	if response.Code != http.StatusCreated || store.calls != 1 {
		t.Fatalf("response=%d calls=%d", response.Code, store.calls)
	}
}
```

- [ ] **Step 2: Run handler tests and verify RED**

Run: `go -C src/internal test ./api/v1 -run 'TestStartResearchRequiresIdempotencyKey|TestStartResearchDoesNotDispatchDirectly' -count=1`

Expected: tests fail because the current handler ignores the header and calls direct execution.

- [ ] **Step 3: Replace the handler's debit/create/dispatch sequence**

Add a minimal `ResearchSubmissionStore` interface to the API package and inject
it through `NewResearchAPIFull`. After existing pure validation, require a
trimmed key of 8–255 visible ASCII characters, build the command, call `Submit`,
and write the stored JSON bytes with:

```go
c.Data(result.HTTPStatus, "application/json; charset=utf-8", result.Body)
```

Map quota exhaustion to `403`, idempotency conflict to `409`, and database
failures to `500`. Remove the handler calls to
`CheckAndDeductResearchQuota`, `CreateSession`, `ExecuteResearchWithConfig`,
refund, and cleanup.

- [ ] **Step 4: Add a stable frontend key per start call**

```js
function createIdempotencyKey() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID()
  return `research-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

startResearch(data, idempotencyKey = createIdempotencyKey()) {
  return apiClient.post('/research/start', data, {
    headers: { 'Idempotency-Key': idempotencyKey }
  })
}
```

The key is created once before Axios receives the request, so any retry of that
request config reuses it.

- [ ] **Step 5: Run API tests and scoped frontend lint**

Run: `go -C src/internal test ./api/v1 -run 'TestStartResearch' -count=1`

Run: `npm.cmd exec eslint src/api/index.js -- --no-ignore` from `vue/`

Expected: both pass.

---

### Task 4: Leased Job Repository with Fencing

**Files:**
- Create: `src/internal/repository/dao/research_job_dao.go`
- Create: `src/internal/repository/dao/research_job_dao_test.go`

**Interfaces:**
- Produces:

```go
type JobLease struct {
	JobID, ResearchID, Owner string
	FenceToken               int64
	LeaseUntil               time.Time
}

func (d *ResearchJobDAO) Claim(ctx context.Context, owner string, now time.Time, lease time.Duration) (*model.ResearchJob, error)
func (d *ResearchJobDAO) Heartbeat(ctx context.Context, lease JobLease, until time.Time) error
func (d *ResearchJobDAO) ConsumeReservation(ctx context.Context, lease JobLease, now time.Time) error
func (d *ResearchJobDAO) Complete(ctx context.Context, lease JobLease, now time.Time) error
func (d *ResearchJobDAO) Retry(ctx context.Context, lease JobLease, availableAt time.Time, cause error) error
func (d *ResearchJobDAO) Fail(ctx context.Context, lease JobLease, now time.Time, cause error, releaseReservation bool) error
```

- [ ] **Step 1: Write claim and fencing integration tests**

Create queued jobs in PostgreSQL and assert:

```go
first, err := dao.Claim(ctx, "worker-a", now, time.Minute)
if err != nil { t.Fatal(err) }
second, err := dao.Claim(ctx, "worker-b", now, time.Minute)
if !errors.Is(err, gorm.ErrRecordNotFound) || second != nil {
	t.Fatalf("second claim = %+v, %v", second, err)
}
if err := dao.Complete(ctx, staleLease(first), now); !errors.Is(err, dao.ErrLeaseLost) {
	t.Fatalf("stale completion error = %v", err)
}
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `go -C src/internal test ./repository/dao -run 'TestResearchJobClaim|TestResearchJobFence' -count=1`

Expected: compilation fails because `ResearchJobDAO` does not exist.

- [ ] **Step 3: Implement PostgreSQL claim semantics**

In a short GORM transaction select one eligible row ordered by
`available_at, created_at` with:

```go
Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
Where("(status = ? AND available_at <= ?) OR (status = ? AND lease_until < ?)",
	model.ResearchJobQueued, now, model.ResearchJobRunning, now).
First(&job)
```

Then update status/owner/lease, increment `attempts` and `fence_token`, and
return the row. Every later write includes `status = running`, `lease_owner`,
and `fence_token` in its `WHERE`; zero affected rows returns `ErrLeaseLost`.

- [ ] **Step 4: Implement reservation terminal transitions**

`ConsumeReservation` changes only `reserved -> consumed`.
`Fail(..., releaseReservation=true)` locks the reservation and membership row,
changes only `reserved -> released`, decrements the recorded counter without
going below zero, and marks job/session failed in the same transaction.

- [ ] **Step 5: Run job DAO tests**

Run: `go -C src/internal test ./repository/dao -run 'TestResearchJob' -count=1`

Expected: all job claim, lease expiry, fence, retry, consume, and release tests pass with PostgreSQL.

---

### Task 5: Synchronous Research Execution and Worker Loop

**Files:**
- Modify: `src/internal/service/research_service.go`
- Create: `src/internal/service/research_worker.go`
- Create: `src/internal/service/research_worker_test.go`

**Interfaces:**
- Produces:

```go
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
```

- [ ] **Step 1: Write worker behavior tests with fakes**

Cover queued success, preflight failure/retry, execution failure after quota
consumption, lost lease, panic recovery, and shutdown:

```go
func TestResearchWorkerConsumesThenExecutesAndCompletes(t *testing.T) {
	store := newFakeJobStoreWithOneJob()
	executor := &fakeResearchExecutor{}
	worker := NewResearchWorker(store, executor, WorkerConfig{PollInterval: time.Millisecond})
	worker.runOnce(context.Background())
	if store.consumed != 1 || executor.calls != 1 || store.completed != 1 {
		t.Fatalf("consumed=%d executed=%d completed=%d", store.consumed, executor.calls, store.completed)
	}
}
```

- [ ] **Step 2: Run worker tests and verify RED**

Run: `go -C src/internal test ./service -run TestResearchWorker -count=1`

Expected: compilation fails because the worker does not exist.

- [ ] **Step 3: Add synchronous service execution**

Add:

```go
func (s *ResearchService) ExecuteResearchJob(
	ctx context.Context,
	sessionID, query, researchType string,
) (err error)
```

It validates `CanExecute`, tracks the caller-owned cancellable context, invokes
the existing execution body synchronously, returns execution/persistence
errors, and never calls `context.Background` or starts a goroutine. Refactor
`executeResearch` to return its failure after emitting the existing SSE error.
Ensure callback cleanup uses `defer`.

- [ ] **Step 4: Implement the single-concurrency worker**

`Start(ctx)` polls until cancellation. `runOnce` claims a job, loads its
session, preflights the executor, consumes the reservation, starts a heartbeat
ticker, calls `ExecuteResearchJob`, and completes/retries/fails with the lease
token. Use bounded exponential retry:

```go
func retryDelay(attempt int) time.Duration {
	delay := time.Second * time.Duration(1<<min(attempt-1, 6))
	if delay > time.Minute { return time.Minute }
	return delay
}
```

Recover panics into the same retry/fail path. Do not start another execution
while one is active.

- [ ] **Step 5: Run service tests including the race detector**

Run: `go -C src/internal test -race ./service -run 'TestResearchWorker|TestResearchPersistence' -count=1`

Expected: tests pass without a race report.

---

### Task 6: Application Wiring, Shutdown, and Removal of Direct Dispatch

**Files:**
- Modify: `src/cmd/server/main.go`
- Modify: `src/internal/api/v1/research.go`
- Modify: `src/configs/config.yaml`
- Test: `src/internal/api/v1/research_submission_test.go`
- Test: `src/internal/service/research_worker_test.go`

**Interfaces:**
- Consumes: submission DAO, job DAO, worker, and synchronous executor.
- Produces: one durable path from HTTP commit to leased execution.

- [ ] **Step 1: Add a wiring regression test/search assertion**

Add an API test whose fake `ResearchService` would fail the test if direct
execution is invoked. Add a source assertion in the test that
`StartResearch` contains no `ExecuteResearchWithConfig` call.

- [ ] **Step 2: Run the regression test and verify RED**

Run: `go -C src/internal test ./api/v1 -run TestStartResearchDoesNotDispatchDirectly -count=1`

Expected: fails against the old request path.

- [ ] **Step 3: Wire the durable components**

When `db` and `researchService` are non-nil:

```go
submissionDAO := dao.NewResearchSubmissionDAO(db)
jobDAO := dao.NewResearchJobDAO(db)
researchWorker := service.NewResearchWorker(jobDAO, researchService, service.WorkerConfig{
	Concurrency:  1,
	PollInterval: time.Second,
	LeaseDuration: 30 * time.Second,
})
workerCtx, stopWorker := context.WithCancel(context.Background())
go researchWorker.Start(workerCtx)
```

Inject `submissionDAO` into `NewResearchAPIFull`. During shutdown, call
`stopWorker()` and wait for the worker before `sqlDB.Close()`.
Set `research.worker_pool_size: 1` and document that larger values remain
disabled until Eino callbacks are per execution.

- [ ] **Step 4: Remove obsolete request compensation helpers**

Delete `refundResearchQuota` and any handler-only direct-dispatch cleanup that
has no remaining caller. Keep legacy service methods only if another call site
still requires them; otherwise remove them with their tests.

- [ ] **Step 5: Run server and critical package tests**

Run: `go -C src/internal test ./api/v1 ./repository/dao ./service -count=1`

Run: `go test ./src/cmd/server -count=1`

Expected: all pass.

---

### Task 7: Repository-Wide Verification and Documentation

**Files:**
- Modify: `docs/code-reviews/devlog.md`

**Interfaces:**
- Produces: green changed-package tests, server build, scoped frontend lint, and an evidence-based completion log.

- [ ] **Step 1: Run changed-package verification**

Run: `go -C src/internal test -race ./repository/model ./repository/dao ./service ./api/v1 -count=1`

Run: `go test ./src/cmd/server -count=1`

Run: `npm.cmd exec eslint src/api/index.js -- --no-ignore` from `vue/`

Expected: all changed backend packages, the server package, and the changed
frontend API file pass.

- [ ] **Step 2: Run PostgreSQL integration verification**

Run with the isolated PostgreSQL DSN used during Task 2:

`$env:TEST_DATABASE_URL=$env:RESEARCH_TEST_DATABASE_URL; go -C src/internal test -race ./repository/dao ./service ./api/v1 -count=1`

Expected: concurrent idempotency, rollback injection, lease/fence, worker, and
handler tests all pass without race reports.

- [ ] **Step 3: Measure repository-wide status without expanding scope**

Run: `go -C src/internal test ./... -count=1`

Run: `go test ./src/cmd/server -count=1`

Run: `npm.cmd run build` from `vue/`

Run: `npm.cmd run lint:check` from `vue/`

Run: `git diff --check`

Expected: durable-workflow packages and the server pass. Record any unchanged
legacy compiler, frontend parser, or lint failures exactly; do not fold those
independent backlogs into this first slice. `git diff --check` may show only
repository-wide line-ending warnings, not whitespace errors.

- [ ] **Step 4: Update the development log**

Append the durable-submission transaction boundary, idempotency semantics,
quota-reservation lifecycle, worker lease/fence behavior, PostgreSQL test
results, and exact verification commands to `docs/code-reviews/devlog.md`.
State any skipped external integration test explicitly rather than reporting it
as passed.

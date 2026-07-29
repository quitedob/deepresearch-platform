# Durable Research Submission Design

## Scope

This first slice makes research submission durable and idempotent. A successful
HTTP response must mean that PostgreSQL contains everything required to execute
the research after an API-process crash or restart.

The slice includes:

- submission idempotency keyed by `(user_id, endpoint, Idempotency-Key)`;
- a durable quota reservation tied to one research session;
- atomic creation of the research session, job, and queued outbox event;
- a database-backed worker with leases and fencing;
- removal of request-triggered research goroutine dispatch after the worker path
  is verified.

The slice does not implement durable SSE replay, active-stream revocation,
refresh-token families, RFC 9457 migration, or the historical frontend
build/lint backlog. Those remain separate follow-up slices.

## Current Behavior and Failure Boundary

`StartResearch` currently validates the request, decrements the user's quota,
creates a session, and asks `ResearchService` to start an in-process goroutine.
Compensating refunds and session cleanup cover ordinary request-time errors, but
they cannot cover a process exit after the database writes or recover a lost
goroutine after restart.

The current Eino agents are also shared startup objects with mutable callback
slices. The durable worker must not add parallel execution through those shared
objects until callback ownership is made per execution or execution is
serialized.

## Data Model

### `idempotency_records`

One row claims one client operation.

- `id uuid primary key`
- `user_id uuid not null`
- `endpoint text not null`
- `idempotency_key text not null`
- `request_hash text not null`
- `state text not null` with values `completed`
- `resource_id uuid not null`
- `http_status integer not null`
- `response_body jsonb not null`
- `created_at`, `expires_at`
- unique index on `(user_id, endpoint, idempotency_key)`

The request hash is SHA-256 over a canonical representation of the validated
submission command. Reusing a key with a different hash returns a conflict.
Reusing it with the same hash returns the stored status and response without
changing quota or creating another session.

There is no durable `processing` state in this slice because the record and
business objects are committed by the same PostgreSQL transaction.

### `research_quota_reservations`

One row records the quota decision for one accepted research session.

- `id uuid primary key`
- `user_id uuid not null`
- `research_id uuid not null`
- `membership_type text not null`
- `counter_name text not null`
- `units integer not null default 1`
- `status text not null` with values `reserved`, `consumed`, `released`
- `reserved_at`, `consumed_at`, `released_at`
- unique index on `research_id`

Creating a reservation locks the user's membership row and increments the
existing applicable research-used counter in the same transaction. This
preserves current quota reads while adding a durable object that can be consumed
or released exactly once.

The worker changes `reserved` to `consumed` immediately before external research
execution begins. A job canceled or permanently failed before execution begins
changes it to `released` and decrements the same membership counter under a row
lock. Failures after external execution begins remain charged.

### `research_jobs`

One row represents one durable execution.

- `id uuid primary key`
- `research_id uuid not null`
- `status text not null` with values `queued`, `running`, `succeeded`,
  `failed`, `cancelled`
- `attempts integer not null default 0`
- `max_attempts integer not null default 3`
- `available_at timestamptz not null`
- `lease_owner text`
- `lease_until timestamptz`
- `fence_token bigint not null default 0`
- `started_at`, `completed_at`
- `last_error text`
- `created_at`, `updated_at`
- unique index on `research_id`
- claim index on `(status, available_at, lease_until)`

### `outbox_events`

The submission transaction writes a `research.queued` event.

- `id uuid primary key`
- `aggregate_type text not null`
- `aggregate_id uuid not null`
- `topic text not null`
- `payload jsonb not null`
- `created_at`
- `published_at`
- `attempts integer not null default 0`
- unique index on `(topic, aggregate_id)`

The worker polls `research_jobs`; it does not depend on an in-memory notification
from the outbox. The outbox is the durable integration/audit event and can be
published by a later dispatcher without changing submission correctness.

## Submission Transaction

The API requires an `Idempotency-Key` header for new research submissions.
After authentication and pure request validation, it calls one repository
method with a validated command:

1. Begin a PostgreSQL transaction.
2. Look up the idempotency tuple.
3. If it exists with the same request hash, return its stored response.
4. If it exists with a different hash, return an idempotency conflict.
5. Claim the tuple. A unique-index conflict caused by a concurrent request is
   resolved by reading and applying steps 3 or 4.
6. Lock or create the user's membership row.
7. Check the applicable research limit and increment its used counter.
8. Insert a `reserved` quota reservation.
9. Insert the `planning` research session.
10. Insert its `queued` research job.
11. Insert the `research.queued` outbox event.
12. Store the exact `201` response in the idempotency row.
13. Commit.

Any error rolls back every row and counter change. The handler returns only
after commit and never directly starts research execution.

## Worker and Lease Protocol

The worker is a background component with a stable instance ID and a bounded
poll interval. Its repository claims one eligible job using a short transaction
and PostgreSQL row locking:

1. Select one `queued` job whose `available_at <= now`, or one `running` job
   whose lease expired, using `FOR UPDATE SKIP LOCKED`.
2. Set it to `running`, increment `attempts` and `fence_token`, and set
   `lease_owner` and `lease_until`.
3. Commit the claim transaction.
4. Validate that the research engine can execute the stored research type.
5. Atomically consume the quota reservation if it is still `reserved`.
6. Run the existing research execution synchronously for that session.
7. Heartbeat the lease while execution is active.
8. Mark the job succeeded or schedule a retry only when
   `(job_id, lease_owner, fence_token)` still matches.

The same ownership predicate guards heartbeats, progress writes, completion,
failure, and retry scheduling. A stale worker cannot overwrite a newer worker's
state. Retry delay uses bounded exponential backoff. Exhausting attempts marks
the job and session failed. If execution never began, the reservation is
released; otherwise it remains consumed.

Shutdown cancels polling, stops new claims, and waits for active executions up
to the existing server shutdown deadline. Unfinished jobs become reclaimable
when their leases expire.

For this slice, research execution is serialized to one active job unless the
shared Eino callback state is first changed to per-execution ownership. The
worker API and lease protocol remain safe for a larger pool later.

## Service Boundaries

- The API owns HTTP parsing, authentication, validation, idempotency header
  rules, and response mapping.
- A submission repository owns the single PostgreSQL transaction and returns
  either a newly created result, a replayed result, quota exhaustion, or an
  idempotency conflict.
- A job repository owns claim, heartbeat, retry, terminal transitions, and
  fencing predicates.
- The worker owns polling and lifecycle.
- `ResearchService` exposes a synchronous execution entry point. It no longer
  creates a background context or goroutine for durable submissions.

## Error Semantics

- Missing or invalid `Idempotency-Key`: `400`.
- Same key and same canonical request: replay the stored status/body.
- Same key with a different request: `409`.
- Quota exhausted: `403`, with no idempotency/session/job/outbox rows committed.
- Transaction failure: `500`, with no partial state.
- Worker preflight failure: retry; on final failure mark the session failed and
  release an unconsumed reservation.
- Lost lease: stop publishing state and let the current lease holder finish.

The existing response envelope is retained in this slice so the durable
workflow is not coupled to the later RFC 9457 migration.

## Testing

Unit tests cover canonical hashing, duplicate-key decisions, quota reservation
state transitions, worker backoff, and fencing predicates.

PostgreSQL integration tests cover:

- two concurrent identical submissions produce one session, one job, one quota
  reservation, one outbox row, and one quota increment;
- the same key with a different body returns conflict;
- injected failure after each transaction step leaves no partial state;
- quota exhaustion creates no workflow rows;
- two workers cannot own the same lease;
- an expired lease is reclaimed;
- a stale fence token cannot heartbeat or complete a reclaimed job;
- a committed queued job runs after a simulated API restart;
- retry exhaustion produces a failed session and correct quota reservation
  semantics.

API tests assert that submission does not call direct execution and that replay
returns the original session ID and response.

## Rollout

1. Add models, migrations, repositories, and tests without changing the HTTP
   path.
2. Add the leased worker and synchronous service execution path.
3. Switch submission to the atomic repository path.
4. Verify queued jobs execute only through the worker.
5. Remove direct goroutine dispatch from the submission path.

The final state has one durable execution path; no permanent dual-dispatch mode
is retained.

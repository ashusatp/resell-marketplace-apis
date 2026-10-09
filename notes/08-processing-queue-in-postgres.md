# Processing the Queue in PostgreSQL (Outbox Pattern & Worker Queues)

When building background workers or implementing the **Transactional Outbox Pattern** in PostgreSQL, using the database as a job queue is attractive because it eliminates the operational overhead of external brokers like RabbitMQ or Redis. 

However, implementing queue polling naively leads to race conditions, worker blocking, duplicate execution, retry storms, and lost jobs.

This note traces the evolution of job claiming queries from a naive **v0** implementation to a production-grade, highly concurrent, resilient **v8** query.

---

## The Journey: From v0 to v8

```
[ v0: Naive SELECT + UPDATE ] ──► Double-dispatch race condition
           │
[ v1: Optimistic Conditional Update ] ──► Wasted CPU / duplicate side-effects
           │
[ v2: Atomic Subquery UPDATE ] ──► READ COMMITTED snapshot & race pitfalls
           │
[ v3: Subquery with FOR UPDATE ] ──► Lock contention / head-of-line blocking
           │
[ v4: FOR UPDATE SKIP LOCKED ] ──► High throughput, but crash = lost job
           │
[ v5: Lease Pattern (run_after) ] ──► Crash-safe, but causes retry storms
           │
[ v6: Exponential Backoff ] ──► Self-healing, but unbounded retry intervals
           │
[ v7: Capped Backoff (LEAST) ] ──► Controlled backoff, but infinite retries
           │
[ v8: Bound Retries (Final Query) ] ──► Production-ready with DLQ threshold
```

---

## Evolution Breakdown

### v0: The Naive Two-Step Approach

```sql
-- Step 1: Worker claims a job
SELECT id, kind, payload FROM jobs WHERE done = false ORDER BY id LIMIT 1;

-- Step 2: In Go worker
-- Process job...

-- Step 3: Mark as done
UPDATE jobs SET done = true WHERE id = $1;
```

#### What Breaks: Double-Dispatch Race Condition
When you spin up Worker 2, both workers execute the `SELECT` query within milliseconds of each other. Both read the **exact same row** (`id = 1`). Both workers execute the job in Go (e.g., resizing the same image, sending duplicate emails, charging credit cards twice).

---

### v1: Optimistic Claim via Conditional Update

To fix the double-update, we check `done = false` during the update:

```sql
-- Step 1: Select job
SELECT id, kind, payload FROM jobs WHERE done = false ORDER BY id LIMIT 1;

-- Step 2: Process job in Go...

-- Step 3: Conditional update
UPDATE jobs SET done = true WHERE id = $1 AND done = false;
-- If RowsAffected() == 0, someone beat you to it -> rollback / try again
```

#### What Breaks: Massive Computation & Side-Effect Wastage
While this prevents the database row from being marked done twice, **the heavy computation has already happened** in Go!
* If Worker 1 and Worker 2 both pick Job 1, both download images from R2 and compress them.
* Worker 1 commits first (`RowsAffected = 1`).
* Worker 2 finishes compression, tries to update, gets `RowsAffected = 0`, and discards all that spent CPU, memory, and network time.
* If the job sent a non-idempotent webhook or third-party email, the external action already happened twice.

---

### v2: Single Atomic Statement (Subquery Picks, Outer Updates)

Why use two round trips? Let's claim and mark the job in a single SQL statement:

```sql
UPDATE jobs
SET done = true
WHERE id = (
    SELECT id FROM jobs WHERE done = false ORDER BY id LIMIT 1
)
RETURNING id, kind, payload;
```

#### What Breaks: `READ COMMITTED` Snapshot Pitfall
PostgreSQL defaults to the **`READ COMMITTED`** isolation level:
1. Worker A and Worker B run the query concurrently.
2. Both subqueries take a snapshot of the table where `Job 1` has `done = false`.
3. Worker A's outer `UPDATE` locks `Job 1` and marks it `done = true`.
4. Worker B's outer `UPDATE` blocks, waiting for Worker A's transaction to commit.
5. Once Worker A commits, Worker B re-evaluates the `WHERE` clause for `Job 1`. Because `Job 1` is now `done = true`, the condition fails!
6. Worker B updates **0 rows**, wastes a database roundtrip, and remains idle even though other jobs (`Job 2`, `Job 3`) were waiting.

---

### v3: Explicit Row Lock (`FOR UPDATE`)

To stop workers from picking the same row in the subquery, lock the row immediately:

```sql
UPDATE jobs
SET done = true
WHERE id = (
    SELECT id FROM jobs WHERE done = false ORDER BY id FOR UPDATE LIMIT 1
)
RETURNING id, kind, payload;
```

#### What Breaks: Serialization & Head-of-Line Blocking
`FOR UPDATE` acquires an exclusive row-level lock on the chosen row.
* When Worker 2 executes `SELECT ... FOR UPDATE`, it encounters the locked row and **blocks/waits**.
* All workers line up behind each other, completely eliminating parallel processing benefits. Throughput degrades as you add more workers.

---

### v4: Non-Blocking Locks (`FOR UPDATE SKIP LOCKED`)

PostgreSQL 9.5 introduced `SKIP LOCKED`:

```sql
UPDATE jobs
SET done = true
WHERE id = (
    SELECT id FROM jobs WHERE done = false ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1
)
RETURNING id, kind, payload;
```

#### Why This Works: Linear Worker Scaling
* If `Job 1` is currently locked by Worker 1, Worker 2 **skips it immediately** and grabs `Job 2`.
* Zero blocking, zero lock contention. Throughput scales linearly with worker count.

#### The New Flaw: Worker Crashes & Job Loss
If the worker process crashes (OOM killed, panic, server restart, network partition) *while* processing the job in Go:
* The database has already marked `done = true`.
* The job was never completed, yet it is marked done. **The job vanishes into the void forever.**

---

### v5: Failure Recovery via the Lease Pattern (`run_after`)

Instead of marking `done = true` immediately, we acquire a temporary **lease**:

```sql
UPDATE jobs
SET run_after = NOW() + INTERVAL '30 seconds'
WHERE id = (
    SELECT id FROM jobs
    WHERE run_after <= NOW()
    ORDER BY id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, kind, payload;
```
*When Go finishes the job successfully, it deletes the job or marks it completed (`DELETE FROM jobs WHERE id = $1` or `status = 'completed'`).*

#### Why This Works: Crash-Resilience
If Worker 1 crashes mid-processing, the lease expires after 30 seconds (`run_after <= NOW()`), and another worker automatically claims and retries it.

#### What Breaks: The Retry Storm (Thundering Herd)
If an external dependency goes down (e.g., Cloudflare R2 is unreachable, S3 rate limits, downstream payment gateway returns 503):
* Every job fails immediately.
* 30 seconds later, *all* workers retry *all* failed jobs at once.
* This hammers the recovering external service and floods your system with retry storms.

---

### v6: Exponential Backoff

Give external systems breathing room to recover by scaling the delay with every retry attempt:

```sql
UPDATE jobs
SET attempts = attempts + 1,
    run_after = NOW() + INTERVAL '30 seconds' * POWER(2, attempts)
WHERE id = (
    SELECT id FROM jobs
    WHERE run_after <= NOW()
    ORDER BY id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, kind, payload;
```

#### Delay Progression:
* Attempt 0: $30s \times 2^0 = 30\text{s}$
* Attempt 1: $30s \times 2^1 = 60\text{s}$
* Attempt 2: $30s \times 2^2 = 120\text{s}$ (2 min)
* Attempt 3: $30s \times 2^3 = 240\text{s}$ (4 min)

*(Note: In ultra high-scale systems, adding random "jitter" prevents synchronized lockstep retries).*

#### What Breaks: Unbounded Exponential Explosion
Without an upper limit:
* Attempt 10: $30s \times 1024 \approx 8.5\text{ hours}$
* Attempt 16: $\approx 22\text{ days}$
* Attempt 30: **1,000+ years** or integer/interval arithmetic overflow!

---

### v7: Capped Backoff with `LEAST()`

Clamp the retry delay to a sensible maximum (e.g., 1 hour):

```sql
UPDATE jobs
SET attempts = attempts + 1,
    run_after = NOW() + LEAST(
        INTERVAL '30 seconds' * POWER(2, attempts),
        INTERVAL '1 hour'
    )
WHERE id = (
    SELECT id FROM jobs
    WHERE run_after <= NOW()
    ORDER BY id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, kind, payload;
```

#### What Breaks: Poison Pills (Infinite Retries)
A corrupted payload (e.g., unparseable JSON, invalid image file that causes a permanent panic/error) will retry every hour forever. It pollutes the table and wastes resources indefinitely.

---

### v8: The Production Query (Bounded Retries & Dead Letter Queue)

We limit the maximum number of attempts via `$1` (e.g., `attempts < 5`):

```sql
UPDATE jobs
SET attempts = attempts + 1,
    run_after = NOW() + LEAST(
        INTERVAL '30 seconds' * POWER(2, attempts),
        INTERVAL '1 hour'
    )
WHERE id = (
    SELECT id FROM jobs
    WHERE run_after <= NOW()
      AND attempts < $1
    ORDER BY id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, kind, payload;
```

---

## Summary of the Final Production Pattern

| Feature | How It Is Achieved |
| :--- | :--- |
| **Concurrency & Non-blocking** | `FOR UPDATE SKIP LOCKED` lets workers grab distinct rows without blocking each other. |
| **Atomic Claim** | Single `UPDATE ... WHERE id = (SELECT ...)` ensures the claim and lease update happen atomically. |
| **Crash Safety** | Lease pattern via `run_after`: if the worker dies, another worker automatically resumes after expiration. |
| **Backoff Protection** | Exponential backoff (`POWER(2, attempts)`) prevents retry storms against downstream services. |
| **Interval Safety** | `LEAST(..., INTERVAL '1 hour')` prevents unbounded exponential delay growth. |
| **Dead-Letter Handling** | `attempts < $1` quarantines poison jobs after max attempts for alerts and manual triage. |

---

## Schema Reference (from our migration)

Matches [migrations/000006_create_jobs_table.up.sql](file:///Users/ashutosh/Documents/All%20in%20One/coders%20Gyan/resell-products/resell-apis/migrations/000006_create_jobs_table.up.sql):

```sql
CREATE TABLE jobs (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL,
    payload JSONB NOT NULL,
    attempts SMALLINT NOT NULL DEFAULT 0,
    last_error TEXT,
    run_after TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Crucial index for index-scans on pending jobs:
CREATE INDEX idx_jobs_run_after ON jobs(run_after);
```

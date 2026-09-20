# ADR-011: Deployment concurrency, locking and recovery

## Status

Accepted

## Context

`architecture.md` §5.4 lists what the design "must consider": duplicate deployment requests, process interruption, Docker daemon failure, proxy failure, health-check timeout, persistence failure, partial deployment state. It then concludes that "deployment operations should be designed to be as idempotent as practical," which is a sentiment rather than a specification. Nothing in the document or the code says what actually happens in any of those cases.

Two concrete scenarios make the gap sharp.

**Concurrency.** Two `shiphold deploy` invocations start within seconds of each other for the same application and environment — a developer runs one locally while CI runs another. Both pass readiness, both start candidate containers, both write router configuration. The final state depends on interleaving. In the worst case the router points at a container the other invocation has already removed, and the public endpoint is down.

**Interruption.** A deployment is `SIGKILL`ed between "write router config pointing at the candidate" and "verify traffic." There is now a record in `DEPLOYING`, a running candidate, a router pointing somewhere, and no process. The next invocation has no idea what it is walking into.

Both are ordinary. Neither is currently handled, and a deployment *safety* tool that corrupts state when run twice has failed at the one thing it is for.

## Options Considered

**Hope, and document a warning.** Rejected on the grounds that the entire product is an argument against this approach.

**Advisory file lock only.** `flock` on a lock file. Simple, no dependencies, works for the single-host case. Does not survive the process that held it being killed in a way that leaves the *deployment* half-done — the lock is released, the mess is not.

**Database lock only.** A PostgreSQL advisory lock. Correct, and it makes PostgreSQL mandatory, which conflicts with the file-backed default from ADR-010.

**Lock plus a recovery pass, with the lock mechanism chosen by backend.** Locking prevents the race; recovery handles the aftermath of a lock-holder that died. Both are needed, because neither solves the other's problem.

## Decision

### 1. A deployment lock, scoped to (application, environment)

```go
type DeploymentLock interface {
    // Acquire returns ErrDeploymentLocked immediately if held.
    Acquire(ctx context.Context, key LockKey) (Lease, error)
}

type Lease interface {
    Release(ctx context.Context) error
    Token() string       // fencing token; see below
    Holder() LockHolder  // pid, host, deployment ID, acquired-at
}
```

Key is `(application, environment)` — deploying to staging and production simultaneously is fine and normal; two deployments to production is not.

Two implementations, matching the repository backends:

- **File backend:** `flock` on `.shiphold/<app>.<env>.lock`, with holder metadata written into the file so a conflict can name who holds it.
- **PostgreSQL backend:** `pg_try_advisory_lock` on a hash of the key, held on a dedicated connection.

**Acquisition does not block by default.** A second deployment fails fast with `ErrDeploymentLocked` and exit code 6, naming the holder:

```text
Error: another deployment is in progress
       application: payments-api   environment: production
       deployment:  dpl_01HQ8X…    started: 2026-09-18T11:04:22Z
       host:        ci-runner-7    pid: 4412
Hint:  wait for it to finish, or --wait to queue behind it
```

Queuing behind a deployment is opt-in. The default for an unexpected concurrent deploy is to stop and say so, because in CI a silent wait looks identical to a hang.

### 2. Fencing tokens

A lock alone is not sufficient. The classic failure: process A acquires the lock, stalls (GC pause, suspended VM, network partition), its lease expires, process B acquires the lock and deploys — and then A wakes up and writes router configuration it computed before B existed.

Every lease carries a monotonically increasing token. It is written into the router configuration and into the deployment record, and any write carrying a token lower than the current one is rejected. This turns "A wakes up and corrupts the switch" into "A wakes up and is refused."

This is not an exotic concern for a tool that runs on CI runners, which are routinely throttled and suspended.

### 3. Startup reconciliation

Before acquiring the lock, every `deploy` and `rollback` runs a recovery pass.

```text
1. Read the latest record for (application, environment)
2. If it is terminal → nothing to recover, continue
3. If non-terminal, check for a live holder:
     lock held by a live process?  → ErrDeploymentLocked, stop
     no live holder                → the previous run died. Recover:
4. Observe reality:
     - which containers exist, and their state
     - what the router configuration currently points at
     - what the record says was intended
5. Reconcile, preferring the SAFE state over the INTENDED one:
     router → old version, old healthy   → remove candidate,
                                            record INTERRUPTED → FAILED
     router → candidate, candidate healthy → complete forward:
                                            verify, stop old, record SUCCEEDED
     router → candidate, candidate unhealthy → revert router to old,
                                            verify old, remove candidate,
                                            record INTERRUPTED → FAILED
     router config unparseable / no healthy target
                                          → do NOT guess.
                                            record INTERRUPTED, exit 5,
                                            print exactly what was observed
                                            and the manual recovery steps
```

The last branch is the important one. **When the observed state is ambiguous, ShipHold stops and reports rather than guessing.** Automated recovery from a state the tool does not understand is how a partial outage becomes a full one. The report must be specific enough to act on: what containers exist, what the router says, what the record expected.

Recovery is itself recorded — `INTERRUPTED` and the transition out of it are events in the deployment's log (ADR-007), so the audit trail shows that a deployment was interrupted and what was done about it.

### 4. Idempotency of the deploy request

A deploy request is keyed by `(application, environment, image digest, config digest)`. If the most recent terminal record has the same key and outcome `SUCCEEDED`, and observed state matches it, the deployment is a no-op:

```text
payments-api/production is already running sha256:9f2c1a… (dpl_01HQ8W…, 14m ago).
Nothing to do. Use --force to redeploy.
```

This makes re-running a CI job safe, which is the single most common way this command will be invoked twice.

### 5. Operation-level idempotency

Each step in the deployment sequence is individually safe to repeat: container creation is keyed by a deterministic name derived from the deployment ID, so a retry adopts the existing container rather than creating a second one; router config writes are atomic (temp file plus `rename`) and idempotent for a given target and token; the provenance append is keyed on deployment ID with a uniqueness constraint, so a retried write is rejected rather than duplicated.

## Consequences

**Makes easy.** Concurrent invocations fail safely and legibly. An interrupted deployment is detected and either completed or reverted. Re-running CI does not redeploy. The audit trail records interruptions rather than hiding them.

**Makes harder.** Every deployment now begins with a reconciliation pass, which costs a Docker query and a router-config read. Fencing tokens must be threaded through the router adapter and the repository. Reconciliation logic needs testing against deliberately corrupted states, which means the test harness must be able to *create* a half-deployed state — worth building, since it is also how the Week-9 failure-path tests are written.

**Rules out.** Deployments that bypass the lock. Recovery that guesses. Any router write without a fencing token.

**Testing.** The recovery matrix is the highest-value test in the project and also the most tedious: for each combination of (record state, container state, router state), assert the reconciliation outcome. Table-driven, against fakes for Docker and the router. Roughly a dozen cases, and they are the cases that decide whether the word "safety" in the product description is earned.

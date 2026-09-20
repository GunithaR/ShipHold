# ShipHold — Architecture

> **How to read this document.** It describes the system's structure and the boundaries that hold it together. Individual decisions — what was chosen, what was rejected, and why — live in `docs/adr/`, and are cross-referenced throughout. The schedule lives in `docs/roadmap.md`. This document should describe what is true now; where it describes something not yet built, that is marked.

---

## 1. Purpose

ShipHold is a Go-based deployment safety and provenance system for small and medium teams using Git, CI, container registries, and Docker-based deployment environments.

The name carries two meanings: **hold** as a controlled pause before a deployment proceeds (the safety gate), and **hold** as the cargo compartment of a ship where goods are secured and inspected before transit (the provenance trail). See [ADR-000](adr/ADR-000-naming.md).

Its purpose is to make deployment state transitions verifiable before deployment, controlled during deployment, observable after deployment, auditable over time, and reversible when necessary.

The system is designed as a foundation that can later extend to Kubernetes, additional CI providers, and additional registries without rewriting the core.

---

## 2. Core engineering principle

> A deployment is a state transition. Make that transition verifiable, auditable, and reversible.

The system separates seven concerns, and the separations are load-bearing rather than descriptive:

```text
1. Evidence collection   what is true about the proposed deployment?
2. Evaluation            what do the policies say about that evidence?
3. Decision              PASS, WARN, or BLOCK
4. Execution             deploy only when permitted
5. Verification          confirm the resulting state is healthy
6. Provenance            record exactly what happened, tamper-evidently
7. Recovery              restore a previously recorded known-good state
```

Steps 1–3 are pure and reproducible. Given the same evidence and the same policy, evaluation returns the same decision forever — which is what makes a provenance record independently checkable months later, by anyone, with no network access. Everything in this document that looks like extra ceremony exists to protect that property.

---

## 3. Product boundary

ShipHold does not replace GitHub Actions, Docker, Kubernetes, Terraform, Prometheus, or incident-management platforms. It sits around existing delivery infrastructure as a safety layer.

```text
Git
 │
 ▼
CI/CD
 │
 ▼
┌──────────────────────────┐
│         ShipHold         │
├──────────────────────────┤
│ Readiness                │
│ Policy                   │
│ Deployment               │
│ Verification             │
│ Provenance               │
│ Rollback                 │
└────────────┬─────────────┘
             │
             ▼
        Reverse Proxy
             │
             ▼
           Docker
             │
             ▼
      Running Service
```

The initial target is a single-host Docker environment with Traefik. Kubernetes is a future deployment adapter, not a v1 requirement.

GitHub Actions is the initial CI integration, via its REST API. A full GitHub App is not required for v1 and is deferred ([`extended-scope.md`](extended-scope.md) §3.1).

### What ShipHold is not

Not a GitOps controller, a Kubernetes platform, a CI system, a container orchestrator, a reverse proxy, a monitoring platform, an incident-response tool, or an AI system making deployment decisions. The last of these is a firm architectural boundary, not a current limitation — see [ADR-015](adr/ADR-015-ai-as-explanation-layer.md).

---

## 4. Architectural goals

### 4.1 Primary goals

- Keep deployment decisions deterministic and reproducible from recorded evidence.
- Keep external systems behind narrow, consumer-declared interfaces.
- Keep domain logic independent of Docker, GitHub, PostgreSQL, and vendor SDKs.
- Make deployment state durable, reconstructable, and tamper-evident.
- Preserve known-good state during failed deployments.
- Make failure handling explicit — every failure mode has a state, an exit code, and a record.
- Make the system testable without infrastructure for the great majority of its behaviour.
- Use real infrastructure integration tests only where mocks give insufficient confidence.
- Fail closed: when a required condition cannot be verified, refuse.

### 4.2 Non-goals

Building a reverse proxy, a Kubernetes platform, a monitoring platform, an incident-response tool, a CI system, an infrastructure-provisioning engine, or a custom policy language. Making AI responsible for any part of a deployment decision.

---

## 5. Layered architecture

Four layers, dependencies flowing one way only. See [ADR-001](adr/ADR-001-layered-architecture.md).

```text
cmd/shiphold/              binary; exit codes; one os.Exit
  │
  ▼
internal/cli/              parse flags, render results, map errors to exit codes
  │
  ▼
internal/application/      orchestration; DECLARES the ports it needs
  │
  ▼
internal/domain/           types and rules; imports stdlib only
  ▲
  │  implements
internal/infrastructure/   concrete adapters; returns domain types
```

**On ports.** An earlier draft of this document described `Ports / Interfaces` as a layer sitting between domain and infrastructure. That is a faithful translation of hexagonal architecture from languages with explicit interface implementation, and it translates badly to Go. Go interfaces are satisfied implicitly, and the idiom is that **the consumer declares the interface it needs, in its own package, as narrowly as possible**. A central ports package produces wide interfaces shaped by what implementations offer rather than by what callers need, and creates a package every layer must import.

So: `application/check` declares the collector interfaces it consumes; `application/deploy` declares the deployment-target and router interfaces it consumes. There is no ports package. This is what the code already does; the document now matches it.

**The dependency rule is checked in CI**, not merely stated — see ADR-001 for the two `go list` invocations that enforce it. A rule that lives only in a document is broken by month two.

### Layer responsibilities

**`cmd/`** — constructs the composition root, runs the root command, maps the returned error to an exit code, calls `os.Exit` exactly once. Never panics.

**`internal/cli/`** — flag parsing, result rendering (text and JSON), and error-to-exit-code translation, done once in a shared wrapper rather than repeated per command. Contains no business logic: no policy rules, no retry loops, no state transitions. All commands use `RunE`.

**`internal/application/`** — orchestration. `CheckService`, `DeployService`, `RollbackService`, `HistoryService`, `VerifyService`. Coordinates domain operations and the ports it declares. Holds no rules of its own.

**`internal/domain/`** — `deployment`, `readiness`, `policy`, `provenance`. Standard library only: no YAML tags, no DB tags, no SDK types. Serialisation happens at the boundaries.

**`internal/infrastructure/`** — `git`, `github`, `registry`, `docker`, `proxy`, `postgres`, `filestore`. Each adapter returns domain types and classifies its own errors ([ADR-004](adr/ADR-004-errors-and-exit-codes.md)).

### Composition root

Dependencies are wired in exactly one place, `internal/cli/deps.go`, written **before the second command exists**. Hand-wiring inside command closures — which the current `check` command does — is how a "someday" cleanup accumulates for eight commands.

---

## 6. Repository structure

Module path: `github.com/GunithaR/ShipHold`. (Mixed case is a known wart, accepted rather than changed — see [ADR-002](adr/ADR-002-go-and-cobra.md). The binary, the command, and all prose are lowercase `shiphold`.)

```text
shiphold/
├── cmd/shiphold/main.go
│
├── internal/
│   ├── domain/
│   │   ├── deployment/     Deployment, Status, Event, Environment, transitions
│   │   ├── readiness/      Evidence, Decision, per-source evidence types
│   │   ├── policy/         Policy, Rule, RuleResult, Evaluate
│   │   └── provenance/     Record, canonical serialisation, hash chain
│   │
│   ├── application/
│   │   ├── check/          readiness orchestration; declares collector ports
│   │   ├── deploy/         deployment orchestration; declares target + router ports
│   │   ├── rollback/
│   │   ├── history/
│   │   └── verify/         ledger chain verification
│   │
│   ├── infrastructure/
│   │   ├── git/            local Git evidence
│   │   ├── github/         GitHub Actions CI evidence
│   │   ├── registry/       digest resolution
│   │   ├── docker/         container lifecycle
│   │   ├── proxy/          Traefik file-provider router
│   │   ├── postgres/       repository + lock
│   │   └── filestore/      JSON-Lines repository + flock
│   │
│   ├── cli/                commands, renderers, composition root
│   ├── config/             load, validate, hash, discover
│   ├── secret/             Secret type; reference resolution
│   └── shiphold/           error kinds, sentinels, exit codes
│
├── tests/
│   ├── integration/        build-tagged; needs Docker
│   └── e2e/                full topology via Docker Compose fixture
│
├── docs/
│   ├── architecture.md
│   ├── vision.md
│   ├── roadmap.md
│   ├── extended-scope.md
│   ├── adr/
│   ├── development/
│   └── operations/
│
├── examples/
├── .github/workflows/
├── CONTRIBUTING.md
├── LICENSE
├── Makefile
└── go.mod
```

**Create packages when the capability exists, not because they appear here.** This is a design guide, not a checklist.

---

## 7. Core domain model

```text
Environment          a validated, declared deployment environment  (ADR-013)

Evidence             what is true, per source, with availability   (ADR-005)
  ├── GitEvidence
  ├── CIEvidence
  ├── ImageEvidence
  └── ConfigEvidence

Policy               rules and severities, scoped per environment  (ADR-006)
Rule / RuleID        a named, pure predicate over evidence
RuleResult           SATISFIED | VIOLATED | UNKNOWN | SKIPPED      (ADR-003)
EvaluationResult     decision + rule results + policy digest
Decision             PASS | WARN | BLOCK

Deployment           identity, environment, digests, event log     (ADR-007)
Status               lifecycle state
Event                an appended, immutable transition record

HealthCheckConfig    endpoint, interval, timeout, threshold
HealthProbeResult    the outcome of probing a running candidate

Record               the provenance record, hash-chained           (ADR-009)
```

### Two concepts that were previously one

The earlier model had a single `HealthCheck` notion, and the policy evaluator emitted `"health check is invalid"`. These are two different things at two different times:

- **`HealthCheckConfigured`** — pre-deployment evidence. Is a health check defined and well-formed? Feeds policy evaluation.
- **`HealthProbeResult`** — deploy-time fact, produced after a candidate container starts. Feeds the state machine.

Keeping one name for both would have caused confusion precisely at the point where the deploy path is written.

### Deployment identity is the digest

`ImageRef` records what the human asked for. `ImageDigest` records what was actually deployed, and it is the identity. A mutable tag may point somewhere else entirely by the time a record is read, which is the failure mode immutable digests exist to prevent.

---

## 8. Evidence → Evaluation → Decision

The central boundary of the architecture.

```text
External systems
      │        (narrow collectors, concurrent, context-bounded)
      ▼
Evidence                          ← may be marked UNAVAILABLE per source
      │        (pure function, no I/O, injected clock)
      ▼
Policy evaluation
      │
      ▼
PASS / WARN / BLOCK   +  per-rule results  +  policy digest
```

### Collection: narrow, concurrent, and honest about failure

Each source is its own small interface declared by `application/check` ([ADR-005](adr/ADR-005-evidence-collection.md)). Collectors run concurrently under a total time budget, each taking a `context.Context`.

**An unreachable source produces evidence marked unavailable, not an error that aborts the check.** This is the design's most consequential small decision. An unreachable GitHub yields:

```text
BLOCK

  ✗ ci_passed         UNKNOWN    CI status could not be determined:
                                 GitHub API returned 403
  ✓ image_present     SATISFIED  ghcr.io/acme/payments-api:v2.1.0
  ✓ immutable_digest  SATISFIED  sha256:9f2c1a…
  ✓ healthcheck       SATISFIED  GET /health, timeout 60s
```

rather than a stack trace. The operator learns that three of four conditions hold and exactly which one could not be established. Aborting on the first error tells them nothing.

### Evaluation: pure, tri-state, fail-closed

```go
func Evaluate(ev readiness.Evidence, p Policy, at time.Time) EvaluationResult
```

No I/O, no clock read, no logging, no error return. Rules are configured `block`, `warn`, or `off`; each produces a `RuleResult`; the decision is the maximum severity observed ([ADR-003](adr/ADR-003-readiness-decision-model.md)).

**`UNKNOWN` on a blocking rule produces `BLOCK`.** A safety tool that cannot verify a required condition refuses. Failing closed on missing evidence is the product's whole posture, and it belongs in the decision table rather than in an adapter's error handling.

The deployment engine never invents its own readiness logic. It receives a decision.

---

## 9. Configuration and policy

Typed YAML, strictly parsed, environment-scoped. See [ADR-006](adr/ADR-006-typed-yaml-policy.md).

```yaml
version: 1

application:
  name: payments-api

environments:
  staging:
    target: docker
    health: { endpoint: /health, timeout: 60s, interval: 2s, failure_threshold: 3 }
    rules:
      ci_passed:              block
      image_present:          block
      immutable_digest:       warn
      healthcheck_configured: block

  production:
    target: docker
    health: { endpoint: /health, timeout: 120s, interval: 2s, failure_threshold: 3 }
    rules:
      ci_passed:              block
      image_present:          block
      immutable_digest:       block
      healthcheck_configured: block

registry:
  url: ghcr.io
  credentials: env:GHCR_TOKEN        # a reference, never a value

provenance:
  backend: file                      # file | postgres
  path: ./.shiphold/ledger.jsonl
```

Four properties this shape commits to:

**Unknown keys are rejected.** `yaml.Decoder` with `KnownFields(true)`. Previously, a typo such as `require_ci_pas` parsed silently and disabled the rule — **a policy typo failed open**, which in a safety tool is a correctness defect rather than a papercut.

**`version` is mandatory.** The format will change; migration requires knowing what you are migrating from.

**Rules are scoped per environment.** Production requiring more evidence than staging is the core use case ([ADR-013](adr/ADR-013-environments.md)).

**Credentials are references.** Policy files live in Git ([ADR-012](adr/ADR-012-secrets.md)).

**The document is hashed on load**, and the digest is recorded in provenance — so the audit trail can answer *under which version of the rules was this permitted?*

Discovery order: `--config`, `SHIPHOLD_CONFIG`, `./shiphold.yaml`, `./.shiphold/config.yaml`.

Rules are Go predicates registered by `RuleID`. No DSL, no embedded scripting, no expression evaluation.

---

## 10. Errors and exit codes

In CI, **the exit code is the API**. See [ADR-004](adr/ADR-004-errors-and-exit-codes.md).

| Code | Meaning |
| --- | --- |
| `0` | PASS |
| `1` | WARN (non-zero by default; `--warn-exit-code` to change) |
| `2` | BLOCK |
| `3` | Configuration error |
| `4` | Evidence error (a required source unreachable) |
| `5` | Deployment failure (previous version preserved) |
| `6` | Conflict (lock held, duplicate deployment) |
| `7` | Provenance failure (write failed, or chain broken) |
| `70` | Internal error |

**A policy BLOCK is not an error.** It is a successful evaluation with a negative result, returned as a valid `EvaluationResult` with a `nil` error. If `BLOCK` and "GitHub returned 500" shared a path, neither CI nor an operator could tell *"we checked and the answer is no"* from *"we could not check"* — and that confusion is how a gate ends up disabled.

Errors carry a `Kind`, an operation, a wrapped cause, and an optional actionable hint. Sentinels support `errors.Is`. No command calls `os.Exit` or prints an error; each returns one, and a single wrapper translates it.

---

## 11. Docker and reverse-proxy deployment model

**Status: provisional.** The traffic-switching mechanism is [ADR-008](adr/ADR-008-traffic-switching.md), which is `Proposed` pending the Week-1 spike. Build nothing load-bearing on this section until that ADR is promoted.

```text
                    Public Endpoint
                          │
                          ▼
                   ┌────────────┐
                   │   Traefik  │   ← owns the host port
                   └─────┬──────┘
                         │
               ┌─────────┴─────────┐
               ▼                   ▼
          v1 (active)         v2 (candidate)
```

Only the proxy owns the public port. Application containers communicate over the Docker network and do not bind application ports to the host.

### The routing abstraction

The deployment domain never mentions Traefik. It depends on a `TrafficRouter` port with `Activate`, `Active`, and `Verify`. Nothing in `domain/` or `application/` references a label, a weight, or a file path. Substituting Caddy or nginx is one new adapter.

### Deployment sequence

```text
1. Reconcile any interrupted prior deployment        (ADR-011)
2. Acquire the (application, environment) lock       (ADR-011)
3. Evaluate readiness                                (ADR-003)
4. Create the deployment record
5. Start the candidate container — not routed
6. Health-probe the candidate DIRECTLY
7. If unhealthy → remove candidate, previous untouched, record FAILED, exit 5
8. Activate the route, carrying the fencing token
9. Wait for the router to report configuration loaded
10. VERIFY THROUGH THE PROXY
11. If verification fails → revert, re-verify previous, remove candidate, record FAILED
12. Stop the previous container
13. Append the provenance record                     (ADR-009)
```

Steps 6 and 10 are deliberately different checks. Step 6 asks *is this build healthy*; step 10 asks *did the routing change take effect*. Conflating them hides the class of bug where the application is fine and the proxy points at the wrong place.

### What "verify routed traffic" means

Previously undefined, which made it unimplementable and untestable. The predicate:

```text
Send N requests (default 10) to the public endpoint through the proxy,
over a window of at most T (default 10s).

PASS  if all N return < 500 AND a response marker identifies the candidate.
FAIL  otherwise.
```

The identifying marker is the essential half. Without it, verification passes when the proxy is still serving the *old* version perfectly well — the requests succeed and nothing has switched. ShipHold injects `X-ShipHold-Deployment: <id>` via router middleware, or reads an application-provided build identifier.

### Failure path

```text
Candidate starts → health check fails → route never activated
→ previous version continues serving 100% of traffic
→ candidate removed → failure recorded with reasons
```

The previous version is never stopped before the candidate is verified through the proxy.

---

## 12. Deployment lifecycle

See [ADR-007](adr/ADR-007-deployment-lifecycle.md).

```text
              REQUESTED
                  │
                  ▼
              VALIDATING
                  │
           ┌──────┴──────┐
           ▼             ▼
        BLOCKED        READY
                         │
                         ▼
                    DEPLOYING
                    │       │
           ┌────────┘       └────────┐
           ▼                         ▼
       VERIFYING                  FAILED
        │      │                     │
   ┌────┘      └────┐                │
   ▼                ▼                │
SUCCEEDED        FAILED ◄─────────────┘
                    │
                    ▼
              ROLLING_BACK
                 │      │
         ┌───────┘      └───────┐
         ▼                      ▼
   ROLLED_BACK           ROLLBACK_FAILED

  INTERRUPTED ← reachable from any non-terminal state; recovery resolves it
```

Terminal: `SUCCEEDED`, `BLOCKED`, `ROLLED_BACK`, `ROLLBACK_FAILED`.

**`ROLLED_BACK` is distinct from `SUCCEEDED`.** A deployment that had to be rolled back did not succeed, and an audit trail recording it as success is lying by omission.

**Transitions append events; they do not overwrite a field.** `TransitionTo(to, reason, at)` validates against the table, appends an immutable `Event`, and updates the cached status. The deployment can therefore narrate its own history, which is what an auditable system requires and what a mutable status field cannot provide.

Time is injected rather than read, so records are reproducible in tests — a requirement of the hash chain.

The transition table is pure domain logic with no dependencies and should be exhaustively tested: every ordered pair of states asserted valid or invalid. If any part of this codebase is proved rather than merely tested, this is the part.

---

## 13. Concurrency, locking and recovery

See [ADR-011](adr/ADR-011-concurrency-and-recovery.md). Previously this document listed these as things the design "must consider"; it now specifies them.

**Locking.** A lease on `(application, environment)` — `flock` with the file backend, `pg_try_advisory_lock` with PostgreSQL. Non-blocking by default: a concurrent deployment fails fast with exit 6 and names the holder. `--wait` opts into queuing. In CI, a silent wait is indistinguishable from a hang.

**Fencing tokens.** Every lease carries a monotonic token, written into router configuration and the deployment record. A stalled process that wakes after its lease expired is refused rather than allowed to write configuration computed before the current deployment existed. Not exotic for a tool that runs on CI runners, which are routinely throttled and suspended.

**Startup reconciliation.** Before acquiring the lock, `deploy` and `rollback` check for a non-terminal record with no live holder, observe actual container and router state, and reconcile — **preferring the safe state over the intended one**. When observed state is ambiguous, ShipHold records `INTERRUPTED`, reports exactly what it saw, and stops. Automated recovery from a state the tool does not understand is how a partial outage becomes a full one.

**Request idempotency.** A deploy keyed on `(application, environment, image digest, config digest)` that matches the current successful state is a no-op with a message, not a redeploy. This makes re-running a CI job safe, which is the most common way this command gets invoked twice.

---

## 14. Provenance

See [ADR-009](adr/ADR-009-provenance-integrity.md).

Every deployment produces one record, written once, never updated: identity, environment, commit, CI run, image reference **and digest**, config digest, **policy digest**, full evidence, rule-level decision, the complete event log, outcome, timings, actor, and the ShipHold version that wrote it.

### The records form a hash chain

```text
Record 1:  PrevHash = ""          RecordHash = H(canonical(r1) || "")
Record 2:  PrevHash = r1.Hash     RecordHash = H(canonical(r2) || r1.Hash)
Record 3:  PrevHash = r2.Hash     RecordHash = H(canonical(r3) || r2.Hash)
```

Altering record 2 breaks record 3's `PrevHash` and every link after it. `shiphold verify` detects it in one pass and names the first broken link.

This is what makes "auditable provenance trail" a supportable claim. Without it, the trail is rows in a database that anyone with access can edit — and an audit log that can be silently rewritten is a log, not an audit trail. The cost is roughly two days: canonical serialisation with golden tests, chain construction, the `verify` command, and database triggers.

Append-only is enforced rather than intended: the repository port has no `Update` and no `Delete`; PostgreSQL has a trigger that raises on either; the file backend opens `O_APPEND`. `Seq` is dense and gapless, so a deleted record is as detectable as an altered one.

### Honest limits

Worth being able to state precisely. v1 **detects** record-level tampering; it does not prevent it. A full-database rewrite — recomputing every hash from record 1 — is not detected; external anchoring of the chain head closes that and is deferred ([`extended-scope.md`](extended-scope.md) §1.2). There is no authenticity property; that requires signing (§1.1).

---

## 15. Rollback

Rollback is a **new forward deployment** that restores a previously recorded state. It never rewrites history.

```text
Deployment 184  (succeeded)
Deployment 185  (succeeded, but the release is bad)
Deployment 186  (rollback) ── Restores: 184
```

186 gets a new ID, passes the same readiness gate, starts a candidate, health-checks it, and switches traffic through the same code path as any deployment. 184's record is untouched.

It restores the recorded **image digest and configuration digest**, not the image tag — the tag may now point somewhere else entirely. `LastSuccessful(application, environment)` is the query, and it is environment-scoped so that rolling production back can never find a staging deployment.

The system does not assume an image alone defines deployment state.

---

## 16. Secrets

See [ADR-012](adr/ADR-012-secrets.md). Configuration holds references (`env:NAME`, `file:/path`), never values; a literal is rejected at load with a rotation hint, because by the time that error appears the value has usually already been committed.

A `Secret` type implements `String`, `GoString`, `MarshalJSON`, `MarshalYAML`, and `slog.LogValue` to return `***`. Reading the value requires an explicit `.Expose()`, which is greppable and reviewable.

**No secret enters evidence or a provenance record**, enforced structurally: no type reachable from `Record` has a `Secret` field. This matters more here than elsewhere because records are append-only — a credential in a record cannot be redacted afterwards.

---

## 17. Observability

See [ADR-014](adr/ADR-014-observability.md).

**stdout carries the result; stderr carries the log.** Nothing else is ever written to stdout, so `shiphold check --output json | jq .decision` works. `log/slog` from the standard library, constructed in the composition root and passed explicitly — no package-level global.

Every command producing a result supports `--output text|json`, and the JSON carries a `schema` field from the first release. Output other people's scripts consume is an interface, and an interface without a version is a promise you cannot keep.

Text output is designed rather than incidental, because for this project the terminal output *is* the user interface — it is what appears in the README and the demo.

Deployments emit a progress line per state transition. Silence during a minute-long deployment is unnerving and, in CI, indistinguishable from a hang.

Metrics, tracing, and log shipping are out of scope; ShipHold is not a monitoring platform and is not a long-running process.

---

## 18. Testing architecture

Testing is continuous, not a phase.

**Unit tests** — domain rules, policy evaluation, evidence aggregation, state transitions, configuration validation, rollback rules, canonical serialisation. No Docker, no network, milliseconds. The bulk of the suite.

**Application tests** — orchestration against fakes for every declared port.

**Adapter tests** — each infrastructure package against a fake server or a real dependency, separately.

**Contract tests** — one suite run against **both** `DeploymentRepository` implementations ([ADR-010](adr/ADR-010-deployment-repository.md)). This is where the repository's real specification lives: a behaviour asserted only against PostgreSQL is a PostgreSQL detail that leaked, not part of the interface.

**Integration tests** — build-tagged, excluded from the default `go test ./...`, using `testcontainers-go` for PostgreSQL. The default suite must run with no Docker daemon and no credentials. The file-backed repository is what makes this achievable, and it is why it exists.

**End-to-end** — at least one full scenario against a Docker Compose fixture: commit → readiness → deployment → routing → verification → provenance → rollback.

**Failure-path tests** — deliberately: failed CI, missing image, mutable-tag-only image, malformed policy, unknown policy key, Docker down, proxy down mid-switch, health timeout, `SIGKILL` during deploy, repository write failure, failed rollback, corrupted ledger. Each must produce a specific error, a specific exit code, and a specific provenance outcome.

The **recovery matrix** — for each combination of (record state, container state, router state), the asserted reconciliation outcome — is the highest-value test in the project. It is roughly a dozen table-driven cases, and they are the cases that decide whether the word "safety" in the product description is earned.

**Golden tests on canonical serialisation** from the first commit. It is a stored format; changing it invalidates every existing hash.

---

## 19. CI

```text
go build  →  go vet  →  golangci-lint  →  go test -race -cover  →  coverage floor
```

On every push. Green from Week 1.

**The coverage floor is set the day CI is set up**, even at a modest value, and raised over time. Adding a gate later, once coverage has drifted down, is a hard conversation — this is a direct lesson from DiffSage, where coverage runs but is not gated for exactly that reason.

The dependency-rule checks from §5 run here too. The default suite requires no Docker and no credentials; integration tests run in a separate build-tagged job.

The toolchain is pinned to the declared language floor, not `stable`, so accidental use of a newer language feature fails in CI rather than on a contributor's machine.

---

## 20. Documentation as a continuous activity

Documentation evolves with implementation. The rule:

> If a change alters the answer to "how does this system work?", update the relevant documentation in the same cycle.

- **`README.md`** — whenever user-visible behaviour changes. It must never describe a capability the binary does not have.
- **`docs/architecture.md`** — when a boundary changes.
- **`docs/vision.md`** — when product direction changes.
- **`docs/adr/`** — when a decision is made. Not reconstructed later.
- **`docs/roadmap.md`** — weekly.
- **`docs/development/`, `docs/operations/`** — setup, testing, contribution; proxy setup, troubleshooting, rollback, recovery.

A weekly check that every link in `README.md` and `docs/` resolves is worth the two minutes. The branch this architecture was reviewed against failed that check in six places, and every one of them was cheap to prevent.

---

## 21. Development workflow

```text
1. Define the problem
2. Update scope or design if needed
3. Identify the domain behaviour
4. Design the interfaces
5. Write the tests
6. Implement
7. Unit tests
8. Integration tests where warranted
9. Update documentation
10. Write or update the ADR if a decision was made
11. Full CI
12. Commit
```

Step 10 is the one that is easiest to skip and most expensive to skip. An ADR written when the decision is made captures the alternatives that were live at the time; one reconstructed later captures only the outcome, and the alternatives are the interesting part.

---

## 22. Architectural decision records

The full index is in [`docs/adr/README.md`](adr/README.md). Sixteen records, one `Proposed` (ADR-008, pending the traffic-switching spike) and the rest `Accepted`.

Records are never edited to reflect a change of mind. A reversed decision becomes `Superseded by ADR-NNN`, and the new record explains what changed. A superseded ADR is evidence of learning.

---

## 23. Extension strategy

Expansion should be additive.

```text
DeploymentTarget          TrafficRouter           CIProvider
├── DockerTarget          ├── TraefikFileRouter   ├── GitHubActions
└── KubernetesTarget ✦    └── CaddyRouter ✦       └── GitLabCI ✦

RegistryProvider          DeploymentRepository
├── GHCR                  ├── FileRepository      ← both in v1
└── ECR ✦                 └── PostgresRepository

✦ = post-v1 (extended-scope.md)
```

`DeploymentRepository` ships with two implementations in v1, deliberately. The DiffSage playbook's rule is that "an interface shaped by one implementation is a guess; shaped by two, it's tested," and the repository is the seam where the second implementation is cheapest — half a day — and pays for itself immediately by making the whole test suite runnable without Docker.

The other three seams ship with one implementation each, and their ADRs say so explicitly, naming what the second would be and which parts of the signature exist because of it. Designing against a hypothetical second implementation and *saying that is what you did* is defensible; shaping an interface around one implementation and calling it an abstraction is not.

Future GitOps functionality would consume the existing deployment and provenance model rather than introducing a second one.

---

## 24. Architectural quality bar

The foundation is sound when:

- domain logic is testable without Docker, a network, or a database;
- the default test suite runs in seconds with no infrastructure;
- application services are testable without GitHub;
- the dependency rule is enforced in CI, not merely documented;
- provider adapters are replaceable, and at least one seam proves it with two implementations;
- traffic routing is isolated from deployment orchestration;
- every failure mode has an explicit state, an exit code, and a provenance outcome;
- an interrupted deployment is detected and either completed or reverted, and never guessed at;
- rollback is a real operation that preserves history;
- the provenance chain is verifiable by a command;
- a recorded decision is reproducible from its recorded evidence, offline;
- CLI commands remain thin and no command exits 0 on a negative verdict;
- configuration is validated strictly and a typo fails closed;
- no document describes a capability the binary does not have.

The objective is not a perfect architecture upfront. It is an architecture that can evolve safely — and, for this project specifically, one whose claims a reader can check.

---

## 25. Milestones

See [`docs/roadmap.md`](roadmap.md). The schedule lives there so that this document can describe the system rather than the calendar, and so that a slipped week does not make the architecture document stale.

---

## 26. Definition of foundation complete

The foundation is complete when future work arrives through existing boundaries.

```text
                        Core Engine
                             │
        ┌────────────────────┼────────────────────┐
        ▼                    ▼                    ▼
   Readiness            Deployment           Provenance
        │                    │                    │
   ┌────┴────┐          ┌────┴────┐          ┌────┴────┐
   ▼         ▼          ▼         ▼          ▼         ▼
 GitHub   Future     Docker   Kubernetes   File    Postgres
 Actions    CI       Target     future
```

Future capability should be additions to adapters, policies, services, or presentation — not rewrites of the domain.

---

## 27. Definition of done

A feature is not complete because its code works. It is complete when the relevant parts of:

```text
Domain behaviour + Implementation + Tests + Documentation + CI verification
```

are present and aligned. The test level depends on the feature; the documentation depends on whether it changes architecture, configuration, operations, or user behaviour.

This applies from Week 1 to v1.0.

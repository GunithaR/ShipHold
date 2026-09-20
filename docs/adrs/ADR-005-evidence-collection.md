# ADR-005: Evidence collection and provider ports

## Status

Accepted

## Context

The evidence → evaluation → decision boundary is the central idea of the architecture. Evidence is gathered from external systems; evaluation is a pure function over it; the decision follows. Keeping those three apart is what makes decisions reproducible and testable.

The current implementation gets the boundary right and the shape of evidence collection wrong. `application/check` declares:

```go
type EvidenceProvider interface {
    Collect() (readiness.Evidence, error)
}
```

and `infrastructure/git.Provider` is the only implementation. It fills `GitCommitSHA` and `Branch` and leaves `CIPassed`, `ImageReference`, `ImageDigest`, and `HealthCheckValid` at their zero values — so every check currently returns `BLOCK` with four reasons.

That is expected at this stage, but the interface has a real problem: it implies **one component that knows about Git and CI and the registry and configuration**. Those are four systems with four sets of credentials, four failure modes, and four rates of change. A single `Collect()` forces them into one call with one error return, which means one unreachable source aborts the collection of the other three.

There is also a `context` problem. `Collect()` takes no `context.Context`, so nothing that reaches the network can be cancelled or given a deadline. Three of the four collectors will reach the network.

## Options Considered

**Keep the single provider, grow it.** Rejected — it becomes the god-object the layering is meant to prevent, and it cannot express partial availability.

**One fat interface with a method per source** (`CollectGit`, `CollectCI`, …). Rejected: a single implementation must then satisfy all of them, or the interface must be split anyway.

**Narrow collectors plus an aggregator.** Each source is its own small interface, declared by the application layer, with an aggregator composing them.

## Decision

### Each source is a narrow, consumer-declared interface

```go
// internal/application/check — declared where it is used (ADR-001)

type GitCollector interface {
    Collect(ctx context.Context) (readiness.GitEvidence, error)
}

type CICollector interface {
    Collect(ctx context.Context, commitSHA string) (readiness.CIEvidence, error)
}

type RegistryCollector interface {
    Collect(ctx context.Context, imageRef string) (readiness.ImageEvidence, error)
}

type ConfigCollector interface {
    Collect(ctx context.Context, env Environment) (readiness.ConfigEvidence, error)
}
```

Every method takes a `context.Context`. Deadlines and cancellation are not optional in a tool that sits in the critical path of a deployment.

### Unavailability is evidence, not an error

This is the substantive decision.

A collector that cannot reach its source does not abort the check. It returns evidence marked unavailable, with the reason attached:

```go
type Availability string

const (
    Available   Availability = "AVAILABLE"
    Unavailable Availability = "UNAVAILABLE"
    NotApplicable Availability = "NOT_APPLICABLE"
)

type CIEvidence struct {
    Availability Availability
    Reason       string  // "GitHub API returned 403" — set when unavailable
    Status       CIStatus
    RunID        string
    RunURL       string
    CompletedAt  time.Time
}
```

The aggregator collects what it can and assembles a complete `readiness.Evidence` in which each section is marked available or not. Evaluation then maps unavailable evidence to `RuleUnknown`, and ADR-003's decision table turns `UNKNOWN` on a blocking rule into `BLOCK`.

The result is that an unreachable GitHub produces:

```text
BLOCK

  ci_passed          UNKNOWN   CI status could not be determined:
                               GitHub API returned 403
  image_present      SATISFIED ghcr.io/acme/api:v2.1.0
  immutable_digest   SATISFIED sha256:9f2c…
  healthcheck        SATISFIED GET /health, timeout 60s
```

rather than a stack trace. The operator learns that three of four conditions are fine and exactly which one could not be established. Aborting on the first error would have told them nothing.

The error return on a collector is reserved for genuine programming faults — a malformed request, a nil client. Anything attributable to the external world becomes `Unavailable`.

### Collectors run concurrently, with a budget

They are independent and mostly network-bound. `errgroup` with a `context` deadline, a per-collector timeout, and a total budget (default 30s, configurable). A slow registry should not hang a check indefinitely; it should time out and be reported as unavailable, which the decision table already handles correctly.

### Evidence is a domain type and is serialisable as recorded

`readiness.Evidence` is stored verbatim in the provenance record (ADR-009). Two consequences follow. It must contain no secrets — a collector authenticating with a token records *that* it authenticated, never the token (ADR-012). And its serialised form is effectively a stored format, so field changes need the same care as a schema change.

### Time is injected

Collectors that record timestamps take a `Clock` rather than calling `time.Now()`. Provenance records must be reproducible in tests, and a hash chain over records containing wall-clock time is untestable without this.

## Consequences

**Makes easy.** One flaky source degrades the check gracefully instead of breaking it. Each collector is independently testable against a fake HTTP server. Adding a source — a SAST report, an approval record — is a new narrow interface and a new evidence section, with no change to existing collectors. Concurrency is contained in the aggregator.

**Makes harder.** More types than one `Evidence` struct and one interface. Every rule must now handle three states rather than two. Both costs are real and both are the correct trade for a tool whose job is to be honest about what it knows.

**Rules out.** Collectors that make policy decisions — a collector reports what is, never what it means. Evaluation that performs I/O. Any collector whose failure aborts the whole check.

**Migration from current code.** `git.Provider` becomes `git.Collector` returning `GitEvidence` and taking a `context`. `check.EvidenceProvider` is replaced by the four narrow interfaces plus an aggregator. The existing test's `fakeEvidenceProvider` splits into per-source fakes, which will make the test suite clearer rather than larger.

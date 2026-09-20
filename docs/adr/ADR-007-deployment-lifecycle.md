# ADR-007: Deployment lifecycle and event log

## Status

Accepted

## Context

A deployment is a state transition, and the product's core claim is that the transition is verifiable, auditable, and reversible. The state machine is therefore not an implementation detail — it is the thing being audited.

The existing `internal/domain/deployment` package is the best-designed code on the branch: an explicit transition table, a pointer receiver that refuses invalid moves, and an error rather than a silent no-op. The design is right. The table is incomplete in ways that will block real work:

| Gap | Effect |
| --- | --- |
| No `DEPLOYING → FAILED` | A container that fails to start cannot be modelled. The only exit from `DEPLOYING` is `VERIFYING`. |
| No `VERIFYING → ROLLING_BACK` | Rollback is reachable only through `FAILED`, so one event needs two transitions. |
| `INTERRUPTED` is unreachable | Declared, never a transition target, though `architecture.md` §5.4 requires interruption handling. |
| `ROLLING_BACK → SUCCESS` | Conflates "the rollback worked" with "the deployment worked." Provenance cannot distinguish a clean deploy from a recovered failure. |
| `RuntimeState` is never written or read | Initialised to `UNKNOWN` and left there. A field that is always one value is worse than no field. |
| `Deployment` carries no time, no events, no decision link, no environment | Six fields, against the fifteen the provenance record needs. |

There is also a deeper question the current design answers implicitly and should answer deliberately: **is a deployment's status a field that gets overwritten, or a value derived from a log of what happened?**

## Options Considered

**Status as a mutable field.** What exists now. Simple, and every transition destroys the previous state. For an ordinary tool this is fine; for one whose product claim is an auditable trail, the object cannot answer *how did it get here* — only *where is it*.

**Full event sourcing.** Events are the only truth, all state derived by replay, no stored status. Rejected as more machinery than a ten-week budget warrants, and it makes simple queries needlessly indirect.

**Append-only event log with a derived, cached status.** Every transition appends an immutable event; the current status is maintained alongside as a derived value.

## Decision

### States

```text
              REQUESTED
                  │
                  ▼
              VALIDATING
                  │
           ┌──────┴──────┐
           ▼             ▼
        BLOCKED        READY            BLOCKED is terminal
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

  INTERRUPTED  ← reachable from any non-terminal state (see below)
```

Terminal: `SUCCEEDED`, `BLOCKED`, `ROLLED_BACK`, `ROLLBACK_FAILED`.

Two changes worth calling out. **`ROLLED_BACK` replaces `ROLLING_BACK → SUCCESS`** — a deployment that had to be rolled back did not succeed, and an audit trail that records it as success is lying by omission. **`INTERRUPTED` is reachable from any non-terminal state**, entered by the recovery pass on startup when a deployment record is found in a non-terminal state with no live process (ADR-011). It is not itself terminal: recovery either reconciles it forward or moves it to `FAILED`.

### Transitions append events

```go
type Event struct {
    Seq       int
    At        time.Time
    From      Status
    To        Status
    Reason    string
    Detail    map[string]string   // never secrets
}

type Deployment struct {
    ID          ID
    Application string
    Environment Environment     // ADR-013
    CommitSHA   string
    ImageRef    string
    ImageDigest string           // immutable identity, not the tag
    ConfigDigest string

    Status      Status           // derived from Events; cached
    Events      []Event

    Decision    *EvaluationResult // why it was permitted (ADR-003)
    StartedAt   time.Time
    EndedAt     *time.Time

    SupersededBy *ID              // set when a later deployment replaces this
    Restores     *ID              // set on a rollback: which deployment it restores
}

func (d *Deployment) TransitionTo(to Status, reason string, at time.Time) error
```

`TransitionTo` validates against the table, appends an `Event`, and updates the cached `Status`. It never overwrites history.

`Restores` is what makes rollback honest: deployment 186 records that it restores the state of 184, and 184's own record is untouched. Rollback moves the system backwards and the ledger forwards.

### Rollback is a deployment

A rollback is a new `Deployment` that happens to target a previously recorded state. It gets a new ID, passes the same readiness gate, starts a candidate, health-checks it, and switches traffic through the same code path. It restores the recorded **image digest and configuration digest**, not the image tag — a mutable tag may now point somewhere else entirely, which is exactly the failure mode immutable digests exist to prevent.

History is never rewritten or deleted. There is no path in this state machine that removes a record.

### Time is injected

`TransitionTo` takes `at time.Time` rather than calling `time.Now()`. The domain package must remain pure (ADR-001), provenance records must be reproducible in tests, and a hash chain over records containing wall-clock time is untestable otherwise.

### The transition table is exhaustively tested

This is pure domain logic with no dependencies: every ordered pair of states should be asserted valid or invalid, including terminal states rejecting everything. It is a table-driven test of maybe forty lines and it should sit at or near 100% coverage. If any part of this codebase is going to be proved correct rather than merely tested, this is the part.

## Consequences

**Makes easy.** Every deployment can be narrated end to end from its own record — which is the demo. Interruption and rollback have real representations rather than being handled by convention. The provenance record (ADR-009) serialises the event log directly.

**Makes harder.** Records grow with events rather than staying fixed-size. Transitions require a reason string, which is slightly more typing and considerably more useful six months later.

**Rules out.** Status assignment outside `TransitionTo`. Deleting or editing history for any reason. A rollback that bypasses the health gate.

**Migration.** Add `Environment`, `ImageDigest`, `ConfigDigest`, `Events`, timestamps, `Restores`. Add the missing transitions. Either wire `RuntimeState` into the Docker adapter or delete it — it currently claims a fact the system does not track.

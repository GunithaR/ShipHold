# ADR-004: Error taxonomy and exit codes

## Status

Accepted

## Context

ShipHold's primary integration point is a CI step. In a CI step the **exit code is the API**: the decision text is for humans reading a log afterwards, and the exit code is the only thing the pipeline acts on.

The current implementation gets this exactly backwards. `internal/cli/check.go` uses `cobra.Command.Run` rather than `RunE`, and handles every failure with `fmt.Println("Error:", err)` followed by a bare `return`. A `BLOCK` decision prints and exits 0. So:

```bash
shiphold check && ./deploy.sh    # deploys regardless of the verdict
```

A deployment gate that cannot fail a build is not a gate. This is the highest-priority defect on the branch.

Separately, there is no error taxonomy at all. `ensureGitRepository` returns `fmt.Errorf("not a Git repository")` — a dynamically constructed error that cannot be matched with `errors.Is`, so no caller can react to it specifically. The DiffSage playbook identifies its `DiffSageError` hierarchy as what made every failure traceable to a cause, and calls bare exception handling out as a thing to fix from day one.

There is one distinction ShipHold needs that DiffSage did not: **a policy BLOCK is not an error.** It is a successful evaluation with a negative result. If `BLOCK` and "GitHub returned 500" travel the same code path with the same exit code, neither CI nor an operator can tell *"we checked and the answer is no"* from *"we could not check."* Those call for different responses — the first is a real signal to act on, the second is an outage to investigate — and conflating them will eventually cause someone to disable the gate.

## Options Considered

**Binary exit codes** (0 success, 1 anything else). Conventional and simple. Rejected: it erases the distinction above, which is the one thing this decision exists to preserve.

**Error strings inspected by callers.** Rejected as obviously fragile.

**Sentinel errors plus a typed wrapper, with a centralised exit-code map.** More setup, and it is what the rest of this system needs.

## Decision

### Errors are classified, not just wrapped

```go
package shiphold

type Kind string

const (
    KindConfig      Kind = "CONFIG"       // malformed or invalid config
    KindEvidence    Kind = "EVIDENCE"     // a source could not be reached
    KindPolicy      Kind = "POLICY"       // policy document itself is broken
    KindDeployment  Kind = "DEPLOYMENT"   // deployment execution failed
    KindProvenance  Kind = "PROVENANCE"   // persistence or integrity failure
    KindConflict    Kind = "CONFLICT"     // lock held, duplicate deployment
    KindInternal    Kind = "INTERNAL"     // a bug
)

type Error struct {
    Kind Kind
    Op   string   // "git.Collect", "docker.StartContainer"
    Err  error
    Hint string   // optional, actionable, human-facing
}

func (e *Error) Error() string { /* "<op>: <err>" */ }
func (e *Error) Unwrap() error { return e.Err }
```

Sentinels for conditions callers branch on:

```go
var (
    ErrNotAGitRepository = errors.New("not a git repository")
    ErrPolicyNotFound    = errors.New("policy file not found")
    ErrUnknownPolicyKey  = errors.New("unknown key in policy file")
    ErrDeploymentLocked  = errors.New("another deployment is in progress")
    ErrHealthTimeout     = errors.New("health check did not pass within timeout")
    ErrLedgerCorrupt     = errors.New("provenance chain verification failed")
)
```

Wrap at every layer boundary with `%w` so that `errors.Is` and `errors.As` work all the way up. Never return a bare `fmt.Errorf` from an adapter.

### A policy decision is not an error

`check.Service.Run` returns `(EvaluationResult, error)`. A `BLOCK` is a perfectly valid `EvaluationResult` with a `nil` error. The error return is reserved for *"the check could not be performed."* Rule-level unavailability is handled inside the evaluation as `RuleUnknown` (ADR-003) and does not surface as an error either.

### Exit codes

| Code | Meaning | Emitted when |
| --- | --- | --- |
| `0` | PASS | Evaluation completed; no violations. Deployment succeeded. |
| `1` | WARN | Evaluation completed; warnings only. **Non-zero by default.** |
| `2` | BLOCK | Evaluation completed; a blocking rule was violated or unknown. |
| `3` | Configuration error | Bad or missing config or policy; unknown key. |
| `4` | Evidence error | A required source could not be reached at all. |
| `5` | Deployment failure | Execution failed. Previous version preserved. |
| `6` | Conflict | Lock held; duplicate deployment. |
| `7` | Provenance failure | Write failed or chain verification failed. |
| `70` | Internal error | A bug. Stack trace, please file an issue. |

Two notes on this table.

**`WARN` exits 1 by default.** A warning that costs nothing is a warning nobody reads. The default is to fail the build, with `--warn-exit-code=0` available for teams that want otherwise. Defaulting a safety tool to permissive is the wrong direction to be wrong in.

**Codes 3–7 are the whole point.** A CI step can distinguish "the deployment was blocked, which is the system working" from "ShipHold could not reach GitHub, which is an outage" by exit code alone, without parsing anything:

```yaml
- run: shiphold check --env production
  # exit 2 → the gate worked, do not deploy
  # exit 4 → ShipHold is broken, escalate, do not treat as a pass
```

### Structure

- Every command uses `RunE`, never `Run`.
- No command calls `os.Exit` or prints an error. It returns one.
- A single wrapper in `internal/cli/` translates a returned error into (exit code, rendered message) exactly once.
- `cmd/shiphold/main.go` calls `os.Exit` in exactly one place and never panics. The current `panic(err)` produces a goroutine dump for what is usually a typo, and Cobra has already printed the error by then, so the user sees it twice.
- `SilenceUsage: true` and `SilenceErrors: true` on the root command — a usage dump on a runtime failure is noise.
- Internal errors print a stack trace; everything else prints one line plus a `Hint` where available.

### Errors carry hints

```text
Error: policy file not found: ./shiphold.yaml
Hint:  create one with `shiphold policy init`, or pass --config
```

Cheap to add at the point where the cause is known, and it is the difference between a tool that feels finished and one that does not.

## Consequences

**Makes easy.** CI can branch on outcome without parsing output. Operators can distinguish a working gate from a broken one. Tests assert on sentinel errors rather than strings. Adding a failure mode is a new `Kind` or sentinel, not a new special case at the call site.

**Makes harder.** Every adapter must classify its errors rather than returning whatever came back. This is the work, and it is the point.

**Rules out.** `panic` outside `main` for anything but genuine programmer error. Bare `fmt.Errorf` crossing a layer boundary. Any command exiting 0 on a negative verdict.

**Immediate actions.** Create `internal/shiphold/errors.go`. Convert `check` to `RunE`. Add the wrapper and the exit-code map. Remove the `panic` and the stray `fmt.Println("ShipHold")` from `main.go`. Document the exit-code table in `README.md` — it is part of the public interface and belongs where people will look for it.

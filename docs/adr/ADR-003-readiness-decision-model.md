# ADR-003: Readiness decision model

## Status

Accepted

## Context

`shiphold check` answers one question: *may this deployment proceed?* How that answer is shaped determines what the rest of the system can do with it — whether CI can act on it, whether provenance can record it, and whether a human can understand why.

The initial implementation has three problems that are cheap to fix now and expensive later.

First, the domain declares three decisions — `PASS`, `WARN`, `BLOCK` — and the evaluator can only produce two. `DecisionWarn` is unreachable, because `policy.Policy` fields are `bool`: a rule is required or it is absent, and there is no way to express "tell me, but do not stop me."

Second, `EvaluationResult.Reasons` is `[]string`. The evaluator computes a structured fact about each rule and then flattens it to prose at the moment of return. Everything downstream — JSON output, provenance records, tests — then has to work with sentences.

Third, there is no distinction between "this rule failed" and "this rule could not be evaluated." If the GitHub API is unreachable, the CI rule has not passed and has not failed; it is unknown. Reporting unknown as failure is wrong, and reporting it as success is dangerous.

## Options Considered

**Boolean gate.** `check` returns pass or fail. Simple, and it is what the exit code ultimately collapses to. Rejected because it discards the reason, and a gate that cannot explain itself gets disabled by the first person it inconveniences.

**Severity per rule, aggregated to a decision.** Each rule is configured with a severity and produces a status; the overall decision is the maximum severity observed.

**Numeric risk score.** Weight each rule, sum, threshold. Rejected outright. A deployment gate must be explainable and reproducible, and "your deployment scored 0.73" is neither. It also invites tuning the weights until the answer is the one you wanted.

## Decision

### Rule severity is tri-state, configured per rule

```yaml
rules:
  ci_passed:               block
  immutable_digest:        block
  healthcheck_configured:  warn
  signed_image:            off
```

`block` fails the gate. `warn` reports and does not fail. `off` skips the rule entirely — and skipping is recorded, so a policy that quietly disabled half its rules is visible in the provenance record rather than invisible.

### Every rule produces a structured result

```go
type RuleID string

type RuleStatus string

const (
    RuleSatisfied   RuleStatus = "SATISFIED"
    RuleViolated    RuleStatus = "VIOLATED"
    RuleUnknown     RuleStatus = "UNKNOWN"   // evidence unavailable
    RuleSkipped     RuleStatus = "SKIPPED"   // severity: off
)

type RuleResult struct {
    RuleID   RuleID
    Status   RuleStatus
    Severity Severity      // block | warn | off, as configured
    Message  string        // human-facing; free to reword
    Evidence []EvidenceRef // what this conclusion was drawn from
}
```

`RuleID` is stable and machine-readable. `Message` is for people and may change without breaking anything. Tests assert on `RuleID` and `Status`, never on `Message`.

### The decision is the maximum severity of violated rules

```text
any VIOLATED rule with severity block   →  BLOCK
else any VIOLATED rule severity warn    →  WARN
else any UNKNOWN rule severity block    →  BLOCK
else any UNKNOWN rule severity warn     →  WARN
else                                    →  PASS
```

**`UNKNOWN` on a blocking rule produces `BLOCK`.** This is the significant line in this ADR. A safety tool that cannot verify a required condition must refuse, not proceed. Failing closed on missing evidence is the whole posture of the product, and encoding it here rather than in an adapter's error handling makes it a property of the design instead of an accident of implementation.

The reason text distinguishes the two cases plainly: *"CI has not passed for commit 8f91a22"* versus *"CI status could not be determined: GitHub API returned 403."* Both block; they are not the same problem, and the operator needs to know which one they have.

### The full result

```go
type EvaluationResult struct {
    Decision      readiness.Decision
    Rules         []RuleResult
    PolicyVersion int
    PolicyDigest  string    // hash of the evaluated policy document
    EvaluatedAt   time.Time
    Environment   Environment
}
```

`PolicyDigest` matters for provenance: it records *which policy* permitted a deployment, so that a later relaxation of the rules is visible in the audit trail rather than retroactively silent.

### Evaluation is a pure function

```go
func Evaluate(ev readiness.Evidence, p Policy, at time.Time) EvaluationResult
```

No I/O, no clock reads, no logging, no error return. Given identical evidence and policy it returns an identical result, forever. This is what makes the decision reproducible from the provenance record months later — anyone can re-run the evaluation against the recorded evidence and get the recorded answer. That property is worth more than any amount of convenience inside the evaluator.

Rendering — text, JSON, colour, table — happens in `internal/cli/`. The evaluator never formats.

## Consequences

**Makes easy.** `WARN` becomes reachable and meaningful. JSON output falls out of the structure for free (ADR-014). Provenance records rule-level detail rather than a sentence (ADR-009). Tests are stable against message rewording. Adding a rule is a new `RuleID`, a pure function, and a config entry.

**Makes harder.** More types than a boolean. Policy files become slightly more verbose. Both are worth it; neither is close.

**Rules out.** Numeric scoring. Rules that perform I/O during evaluation — a rule needing new information means a new evidence collector (ADR-005), not a network call inside `Evaluate`.

**Migration from current code.** `policy.Policy`'s four `bool` fields become a severity map; `Evaluate` returns `[]RuleResult` instead of `[]string`; the CLI gains a renderer. Roughly a day's work now. After policies exist in other people's repositories it becomes a breaking change with a migration path, which is why the config carries a `version` field from the start (ADR-006).

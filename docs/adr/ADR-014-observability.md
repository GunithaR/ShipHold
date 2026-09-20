# ADR-014: Structured logging and machine-readable output

## Status

Accepted

## Context

ShipHold runs in two places with incompatible needs. In a terminal, a developer wants a short, legible verdict. In a CI runner, a pipeline wants a parseable result, and a human debugging that pipeline three days later wants a detailed log they can search.

The current code has neither. Output is `fmt.Println` from inside the command closure, and `cmd/shiphold/main.go` prints the literal string `"ShipHold"` before any command runs — a debug leftover that would corrupt any JSON output added later, because a JSON consumer receives `ShipHold{"decision":...}` and fails to parse.

There is a second problem worth separating out: **output and logs are being conflated.** The decision is a *result*; the story of how it was reached is a *log*. Sending both to stdout means the result cannot be piped into `jq` without the log lines breaking it.

## Options Considered

**`fmt.Println` throughout.** What exists. Rejected as soon as JSON output is needed.

**A third-party structured logger** (zerolog, zap). Faster, more features. Rejected: `log/slog` is in the standard library, its performance is irrelevant at ShipHold's volume (a handful of log lines per invocation, not thousands per second), and ADR-002 commits to keeping the dependency tree small in a tool whose subject is supply-chain discipline.

**`log/slog` for logs, a separate renderer for results.** Standard library, and it separates the two concerns properly.

## Decision

### 1. Logs go to stderr. Results go to stdout.

```text
stdout  →  the result. Text for humans, JSON with --output json.
           Parseable. Nothing else ever written here.
stderr  →  logs. Progress, diagnostics, warnings, errors.
```

This is the conventional Unix split and it makes the obvious thing work:

```bash
shiphold check --env production --output json | jq .decision
```

Nothing writes to stdout except the result renderer. The stray `fmt.Println("ShipHold")` in `main.go` is removed.

### 2. `log/slog`, with a handler chosen by output mode

```go
// text mode  → human-readable handler, stderr, level from --log-level
// json mode  → slog.NewJSONHandler, stderr
```

One logger, constructed in the composition root, passed down explicitly. No package-level global — a global logger is a hidden dependency that makes tests order-dependent and output capture awkward.

Levels used consistently: `Debug` for adapter-level detail (which config file was found, which HTTP call was made), `Info` for lifecycle milestones (container started, health passed, traffic switched), `Warn` for degraded-but-continuing (an evidence source unavailable), `Error` for the failure that ends the command.

Default level is `Info` in a terminal and `Debug` when `CI` is set — when something fails in CI, nobody gets to re-run it with more verbosity on the same conditions, so the verbose run should be the one that already happened.

### 3. `--output text|json` on every command that produces a result

The typed results from ADR-003 make this nearly free — the structure already exists, and the renderer chooses a representation:

```json
{
  "schema": "shiphold.dev/check/v1",
  "decision": "BLOCK",
  "application": "payments-api",
  "environment": "production",
  "commit_sha": "8f91a22c…",
  "policy_digest": "sha256:c41f…",
  "evaluated_at": "2026-09-18T11:04:22Z",
  "rules": [
    {
      "id": "ci_passed",
      "status": "VIOLATED",
      "severity": "block",
      "message": "CI has not passed for commit 8f91a22",
      "evidence": [{"source": "github-actions", "run_id": "1928374", "status": "failure"}]
    },
    {
      "id": "immutable_digest",
      "status": "SATISFIED",
      "severity": "block",
      "message": "resolved to sha256:9f2c1a…"
    }
  ]
}
```

The `schema` field is there from the first release. Output that other people's scripts consume is an interface, and an interface without a version is a promise you cannot keep.

### 4. Text output is designed, not incidental

```text
BLOCK   payments-api → production   commit 8f91a22

  ✗ ci_passed           CI has not passed for commit 8f91a22
                        run 1928374 · failure · 4m ago
  ✓ image_present       ghcr.io/acme/payments-api:v2.1.0
  ✓ immutable_digest    sha256:9f2c1a…
  ! healthcheck         GET /health configured, timeout 60s (warn)

  policy sha256:c41f… · 4 rules · 1 blocking
```

Colour when stdout is a TTY, plain otherwise; `NO_COLOR` respected. The verdict is on the first line because that is the line a person actually reads.

This is worth an hour of care. For a portfolio project the terminal output *is* the user interface, and it is what appears in the README screenshot and the recorded demo.

### 5. Deployments emit progress

`shiphold deploy` runs for a minute or more. Silence during a deployment is unnerving and, in CI, indistinguishable from a hang. Log each state transition (ADR-007) at `Info` with the deployment ID, and emit health-probe attempts at `Debug`:

```text
11:04:22 INFO  deployment requested  id=dpl_01HQ8X app=payments-api env=production
11:04:23 INFO  readiness passed      rules=4 decision=PASS
11:04:24 INFO  candidate started     container=shiphold-payments-api-01HQ8X
11:04:31 INFO  candidate healthy     attempts=4 elapsed=7.1s
11:04:31 INFO  traffic switched      from=dpl_01HQ8W to=dpl_01HQ8X
11:04:33 INFO  traffic verified      requests=10 failures=0
11:04:34 INFO  previous stopped      container=shiphold-payments-api-01HQ8W
11:04:34 INFO  deployment succeeded  id=dpl_01HQ8X elapsed=12.4s
```

That transcript is also, not coincidentally, the most convincing artefact this project can produce. It shows the safety model working, one line at a time.

### 6. Secrets are masked by construction

The `Secret` type from ADR-012 implements `slog.LogValuer`, so `slog` masks it automatically wherever it appears as an attribute. No call site has to remember.

### 7. Explicitly out of scope

Metrics (Prometheus), distributed tracing, and any log shipping. `vision.md` §6 is clear that ShipHold is not a monitoring platform, and structured logs to stderr are already collected by whatever runs the tool. Adding a metrics endpoint would mean ShipHold becomes a long-running process, which it is not.

## Consequences

**Makes easy.** `jq`-able output for CI. A searchable log when a deployment misbehaves. A demo transcript that explains the product without narration. Testable output, since renderers are pure functions over typed results.

**Makes harder.** Two renderers per result type to keep in step — mitigated by both consuming the same typed structure, so neither can drift far. JSON output is a compatibility surface from the moment it ships, which is what the `schema` field is for.

**Rules out.** `fmt.Println` outside the renderers. A package-level logger. Anything on stdout but the result. Unversioned machine-readable output.

**Immediate actions.** Week 1: remove the stray `fmt.Println("ShipHold")`; initialise `slog` in the composition root; add `--output` and `--log-level` as persistent flags on the root command. The renderers arrive with the typed results in Week 2.

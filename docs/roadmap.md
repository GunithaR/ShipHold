# ShipHold — Roadmap

**Last updated:** 2026-09-18
**Horizon:** ten working weeks, with a hard presentation checkpoint at Week 8.

This file replaces the calendar that previously lived in `architecture.md` §25. Milestones live here so that `architecture.md` can describe the system rather than the schedule.

---

## 1. The two deadlines

There are two dates and they are not the same date.

```text
Week 8   CV checkpoint      — the project must be presentable
Week 10  v1.0.0             — the project must be finished
```

**Week 8 is the one that governs every scoping decision in this file.** A project that is 70% complete and can be demonstrated end to end beats a project that is 95% complete and cannot be run. Everything in Weeks 1–8 is chosen so that at the end of Week 8 there is a working binary, a recorded demo, and a coherent set of documents.

Weeks 9–10 add no new capability. They harden, they test failure paths, and they write the narrative.

---

## 2. What "presentable at Week 8" means, concretely

By the end of Week 8, all of the following must be true. This is the acceptance test for the whole schedule.

```text
□ `shiphold check` returns PASS / WARN / BLOCK from real evidence
    and exits with the documented code
□ `shiphold deploy` performs a health-gated blue/green deployment
    against a real Docker + Traefik environment
□ A failed health check leaves the previous version serving traffic,
    and this is demonstrated, not asserted
□ Every deployment writes a provenance record
□ `shiphold history` and `shiphold inspect <id>` read those records back
□ `shiphold rollback <id>` restores a recorded known-good state
□ CI is green: build, vet, lint, race-enabled tests, coverage floor
□ README shows a real terminal transcript, not a planned one
□ ADR-001 through ADR-015 are committed and reflect actual decisions
□ One recorded end-to-end demo (asciinema or a short screen capture)
```

The last item is worth as much as several of the others. Most people reading a CV project will never clone it. A ninety-second recording of a deployment being blocked, then passing, then being rolled back, is the artefact that does the work.

---

## 3. Scope changes from the original twelve-week plan

The original plan assumed twelve weeks of largely full-time work. Ten weeks with a Week-8 checkpoint requires cuts. These are the cuts, with reasoning.

| Change | Rationale |
| --- | --- |
| **`shiphold diff` cut as a standalone command.** Diff output folds into `check`. | It had no domain model (see `REVIEW.md` §4.9) and would have expanded to fill Week 4. `check` already reports what evidence it found; showing "currently deployed digest vs candidate digest" inside that output delivers 90% of the value for 10% of the work. The standalone command moves to `extended-scope.md`. |
| **Registry adapter reduced to digest resolution.** | v1 needs one question answered: *does this tag resolve to an immutable digest, and what is it?* Full registry-client capability (listing, manifests, layer inspection) is not needed by any v1 policy rule. |
| **`DeploymentRepository` implemented file-first, PostgreSQL second.** | This is the only scope *addition*, and it saves more than it costs. A JSON-lines file repository takes about half a day and makes the entire test suite runnable without Docker. PostgreSQL then arrives as the second implementation of a proven interface rather than as the interface itself. It also satisfies the "prove an abstraction with two implementations" rule on the seam where it matters most. |
| **Testcontainers scoped to the PostgreSQL adapter only.** | Originally it was to cover PostgreSQL, container lifecycle, and "complete deployment topology where practical." The last of those is an E2E harness, which is Week 7 work and is better served by a Docker Compose fixture than by Testcontainers. |
| **CI provider limited to GitHub Actions REST, no GitHub App.** | Already the documented position; restated here so it does not creep. |
| **Kubernetes, AI, multi-host, signing: unchanged — all out.** | Already deferred. |

One thing was *not* cut: **the Week-1 Traefik spike.** It is the only item in the plan capable of invalidating the architecture, and the cost of discovering that in Week 6 instead of Week 1 exceeds the cost of the whole spike.

---

## 4. The plan

### Week 1 — De-risk and set the contracts

This week is deliberately not about features. It is about removing the two things that could waste a later week.

**Traefik traffic-switching spike.** Pass criteria, written down before starting:

```text
Given  Traefik + sample-app:v1 on a Docker network, v1 serving traffic
When   v2 starts, passes its health check, and the route is switched
Then   sustained requests through the proxy show zero 5xx and zero
       connection failures across the switch
And    v1 can then be stopped with no further errors

Given  the same setup
When   v2 starts and fails its health check
Then   the route is never switched, v1 continues serving,
       v2 is removed, and the failure is recorded
```

If option 1 (file provider with a weighted service) does not meet these, try option 2 (router priority flip) before changing anything else in the architecture. Record the outcome in ADR-008 and promote it from `Proposed` to `Accepted`.

**Contracts, in code:**

- `internal/shiphold/errors.go` — sentinel errors and the typed wrapper (ADR-004)
- Exit-code mapping centralised in `cmd/shiphold/main.go`, `RunE` everywhere
- `internal/cli/deps.go` — the composition root, before the second command exists (ADR-001)
- A shared command wrapper handling error→exit-code translation, so no command ever writes `try/print/return` again
- `log/slog` initialised, `--output text|json` flag on the root command (ADR-014)

**Housekeeping:** `.github/workflows/ci.yml` with a coverage floor; `go mod tidy`; `.gitignore` expanded; `LICENSE` chosen; `docs/adr/` created and ADR-000 moved into it; broken doc links fixed.

**Deliverable:** a validated traffic-switching approach and a CLI skeleton whose failure behaviour is correct even though it has almost no features.

---

### Weeks 2–3 — Readiness

The `check` command, properly.

- Domain: `Environment` added to the model (ADR-013); tri-state rule severity, `RuleResult`, structured `EvaluationResult` (ADR-003)
- Config: schema `version`, strict unknown-key rejection, `--config` with a documented discovery order, secret references never secret values (ADR-006, ADR-012)
- Evidence: narrow collectors plus an aggregator, with source unavailability captured as evidence rather than as a fatal error (ADR-005)
- Adapters: Git (done, needs hardening), GitHub Actions CI, registry digest resolution
- `check` renders human text and JSON from the same typed result

**Checkpoint:** `shiphold check --env production` returns a correct, explainable PASS / WARN / BLOCK against a real repository and a real GitHub Actions run, with the right exit code.

---

### Weeks 4–6 — Deployment

The largest block, and the one that most needs the Week-1 spike to have succeeded.

**Week 4:** the deployment state machine, corrected and complete — `DEPLOYING → FAILED`, `VERIFYING → ROLLING_BACK`, a distinct `ROLLED_BACK` terminal state, reachable `INTERRUPTED` (ADR-007). Events appended rather than status overwritten. Unit-tested exhaustively; this is pure domain logic and should reach near-total coverage without any infrastructure.

**Week 5:** the Docker deployment target and the traffic router adapter, behind their interfaces. Health probing with a defined timeout and failure threshold. The post-switch traffic verification predicate, defined and implemented (ADR-008).

**Week 6:** failure handling and the deployment lock. Duplicate-invocation rejection, interrupted-deployment recovery on startup, reconciliation of recorded intent against observed Docker and proxy state (ADR-011). Integration tests for the success path and at least three failure paths.

**Checkpoint:** `shiphold deploy` works, and — more importantly — `shiphold deploy` *refuses* to work correctly, on camera.

---

### Week 7 — Provenance

- The full provenance record: every field in `architecture.md` §15, including environment, policy version, rule-level results, and timings
- Canonical serialisation and the `prev_hash` / `record_hash` chain (ADR-009)
- `DeploymentRepository` — file-backed JSON-lines implementation first
- PostgreSQL implementation second, with Testcontainers integration tests (ADR-010)
- `shiphold history`, `shiphold inspect <id>`, `shiphold verify`

`verify` is a small command with an outsized effect: it is what makes "tamper-evident" a demonstrable property rather than a claim in a README.

---

### Week 8 — Rollback, and the CV checkpoint

- Rollback as a new forward deployment that restores a recorded state, never as a history rewrite (ADR-007)
- Rollback passes through the same health gate as a normal deployment
- Restores recorded configuration, not only the image digest
- **Then stop building.** Record the demo. Rewrite the README against the binary that now exists. Verify every item in §2 of this file.

If the schedule has slipped by this point, the honest fallback is to present Weeks 1–7 — check, deploy, provenance — and mark rollback as in progress with its ADR written. A blocked-then-passing deployment with a verifiable audit trail is already a strong project. Do not present a half-working rollback.

---

### Weeks 9–10 — Hardening and narrative

No new features. None.

**Week 9 — failure paths.** Deliberately test: failed CI, missing image, mutable-tag-only image, malformed policy, unknown policy key, Docker daemon down, proxy down mid-switch, health timeout, `SIGKILL` during deploy, repository write failure, failed rollback, corrupted ledger. Each one should produce a specific error, a specific exit code, and a specific provenance outcome. This week is where the project earns the word "safety."

**Week 10 — release.** Operations documentation (proxy setup, troubleshooting, recovery). An architecture-review pass against what was actually built, with ADR supersessions recorded where reality diverged from the plan — a superseded ADR is a feature, not an embarrassment. Release notes. `v1.0.0`.

---

## 5. Weekly discipline

Per-feature, unchanged from the original plan and worth keeping:

```text
Define behaviour → design interfaces → write tests → implement
  → unit tests → integration tests where warranted
  → update docs → update or add ADR → full CI → commit
```

Additionally, once per week:

- `go mod tidy && go vet ./... && golangci-lint run`
- Confirm every link in `README.md` and `docs/` still resolves
- Confirm no document describes a capability the binary does not have

That last check is the one that keeps this project honest. The current branch fails it in six places (`REVIEW.md` §4.11), and every one of them was cheap to prevent and slightly embarrassing to discover.

---

## 6. Explicit non-goals for this ten-week window

Restated so they do not creep back in: Kubernetes, GitOps reconciliation, multiple CI providers, multiple registries, AI in any role, a custom policy DSL, a GitHub App, multi-host deployment, artifact signing, monitoring, incident response.

Anything interesting in that list lives in `docs/extended-scope.md` with a rationale. That document exists so ideas can be written down and then left alone.

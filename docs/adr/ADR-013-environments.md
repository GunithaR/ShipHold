# ADR-013: Environments as a first-class domain concept

## Status

Accepted

## Context

`deployment.Deployment` currently has `ID`, `Application`, `CommitSHA`, `Image`, `Status`, and `RuntimeState`. It does not have an environment.

But essentially every question ShipHold exists to answer is environment-scoped:

```text
"What is running in staging?"
"Is this ready for production?"
"Roll production back to what it was on Tuesday."
"Production requires a passing CI run; staging does not."
"Has this exact digest already been promoted to production?"
```

None of these are expressible against a model where a deployment belongs only to an application. The last two are the ones that matter most: **differentiated policy per environment is the core use case for a deployment gate.** A gate that applies identical rules everywhere is either too strict for staging or too loose for production, and in practice it gets disabled for staging and then forgotten.

The cost asymmetry here is stark, which is why this ADR exists now rather than in Week 7. Adding `Environment` today is one field in one struct and one key in the config. Adding it in Week 7 means changing the domain type, the repository schema, the config format, every CLI command's flags, and — critically — every provenance record already written, which ADR-009 makes **immutable**. There is no migration path for an append-only ledger whose records lack a field. The old records would simply be environment-less forever.

## Options Considered

**Environment as a config-file selection only.** Point the tool at `production.yaml` and it deploys to production. Environment never enters the domain. Simple, and the provenance record cannot say where a deployment went, cross-environment queries are impossible, and "has this been promoted to production?" is unanswerable.

**Environment as a free-form string.** A `string` field on `Deployment`. Nearly free, and permits `prod`, `production`, `Production`, and `prod ` as four distinct environments — which will happen, and which will silently fragment the deployment history of the most important environment.

**A validated domain type, declared in configuration.** `Environment` is a defined type; valid values come from the policy document's `environments` block; anything else is rejected at load.

## Decision

**A validated `Environment` domain type, with the set of valid values declared in configuration.**

```go
package deployment

type Environment string

func (e Environment) Valid() bool { /* lowercase, [a-z0-9-], 1..32 chars */ }
```

Environments are **not** an enum in the source — a hardcoded `dev|staging|production` would be wrong for a team with `qa` or `preprod`. They are declared in the policy document (ADR-006), and any environment not declared there is rejected:

```text
Error: unknown environment "prod"
Hint:  declared environments are: staging, production
```

Rejecting an undeclared environment is what prevents the fragmentation problem, and it costs nothing because the declaration already exists for per-environment policy.

### What environment touches

**Domain.** `Deployment.Environment`, set at construction and never changed. A deployment belongs to exactly one environment for its entire life.

**Policy.** Rules, health-check settings, and the deployment target are all scoped per environment (ADR-006). Production requiring `immutable_digest: block` while staging accepts `warn` is the expected shape.

**Locking.** The lock key is `(application, environment)` (ADR-011). Deploying to staging while production is deploying is normal and must not block.

**Provenance.** `Record.Environment`, indexed, and part of the canonical hash. History, inspection, and rollback are all filtered by it.

**Rollback.** `LastSuccessful(app, env)` — rolling production back must never find a staging deployment.

**CLI.** `--env` on every command that acts on a deployment. Not defaulted:

```bash
shiphold check    --env production
shiphold deploy   --env staging
shiphold history  --env production
shiphold rollback --env production dpl_01HQ8X…
```

**No default.** `shiphold deploy` with no `--env` is an error, not a guess. A tool that quietly defaults to *some* environment will eventually deploy to the wrong one, and the whole product is an argument against that class of accident. `SHIPHOLD_ENV` may supply it in CI, where repeating the flag on every step is genuine friction.

### Deliberately out of scope

**Promotion as a first-class operation.** "Promote the digest currently in staging to production" is a natural next feature, and it needs decisions this ADR does not make — what evidence carries across, whether a fresh CI check is required, whether promotion can skip an environment. The model here supports it (the digest and the environment are both recorded, so the query is easy); the operation itself is deferred.

**Environment ordering or dependency.** No notion that staging precedes production. A policy rule such as `deployed_to_staging_first` could express it later, and it requires cross-environment queries that the recorded model makes possible. Not v1.

**Per-environment credentials.** ADR-012's reference scheme already allows `credentials: env:PROD_GHCR_TOKEN` under `environments.production`, which covers the realistic need without new machinery.

## Consequences

**Makes easy.** Differentiated policy, which is the central use case. Environment-scoped history and rollback. Concurrent deployments to different environments. Cross-environment questions become answerable from the recorded data whenever they are needed.

**Makes harder.** One more required flag on every command, and one more required field in the config. Both are small, and the flag being required is itself a safety property rather than a cost.

**Rules out.** Environment-less deployments. Undeclared environment names. A single global rule set. Any default environment.

**Immediate action.** Add the field now, in Week 2, before anything is persisted. Because ADR-009 makes provenance records immutable, this is the one item in the entire review with a genuinely hard deadline: after the first real record is written, every record before the change is permanently environment-less. That is the whole argument for doing it early, and it is sufficient.

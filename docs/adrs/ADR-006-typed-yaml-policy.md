# ADR-006: Typed YAML policy model

## Status

Accepted

## Context

Policies decide whether a deployment proceeds. They are written by humans, read in review, diffed in pull requests, and — critically — recorded in the provenance trail as part of the justification for a deployment.

The existing model is four booleans in a YAML file, parsed into a struct. It works, and it has four problems: no version field, no rejection of unknown keys, no per-environment scoping, and `bool` fields that make `WARN` inexpressible (ADR-003).

The unknown-key problem is the serious one. `yaml.Unmarshal` into a struct silently ignores keys it does not recognise. So:

```yaml
policy:
  require_ci_pas: true    # typo
```

parses without complaint, and the rule is silently off. **A policy typo currently fails open.** In a safety tool that is not a papercut; it is a correctness defect, and it is the kind that is discovered during an incident rather than during review.

## Options Considered

**Embedded policy language (Rego/OPA, CEL, Starlark).** Genuinely powerful and genuinely standard for this problem space. Rejected on three grounds: it is a large dependency and a second language for users to learn; it makes policy evaluation non-obviously terminating and non-obviously pure, undermining ADR-003's reproducibility guarantee; and ten weeks cannot absorb it. Worth revisiting if real users ever hit the ceiling of declarative rules — and the ceiling should be hit before the tool is built, not anticipated.

**Custom DSL.** Rejected without much thought. Writing a parser is a way to spend three weeks not building a deployment engine.

**Typed YAML with declarative rule severities.** Each rule is a named, implemented-in-Go predicate; the file configures which rules apply at what severity.

## Decision

### Shape

```yaml
version: 1

application:
  name: payments-api

environments:
  staging:
    target: docker
    health:
      endpoint: /health
      timeout: 60s
      interval: 2s
      failure_threshold: 3
    rules:
      ci_passed:              block
      image_present:          block
      immutable_digest:       warn
      healthcheck_configured: block

  production:
    target: docker
    health:
      endpoint: /health
      timeout: 120s
      interval: 2s
      failure_threshold: 3
    rules:
      ci_passed:              block
      image_present:          block
      immutable_digest:       block
      healthcheck_configured: block

registry:
  url: ghcr.io
  credentials: env:GHCR_TOKEN     # a reference, never a value
```

Four things this shape commits to:

**`version` is mandatory.** A file without it is rejected with a hint, not assumed to be version 1. The format will change; a version field costs one line now and makes migration possible later.

**Rules are scoped per environment.** Production requiring more evidence than staging is the central use case (ADR-013); a flat rule list cannot express it, and bolting it on later changes every policy file in existence.

**Rule names are stable identifiers.** `ci_passed` is a `RuleID` (ADR-003), matching a Go predicate. Renaming one is a breaking change requiring a version bump.

**Credentials are references.** `env:GHCR_TOKEN`, never the token itself (ADR-012). Policy files live in Git.

### Parsing is strict

```go
dec := yaml.NewDecoder(r)
dec.KnownFields(true)   // unknown key → error
```

Plus explicit validation after decoding: known severity values only, positive durations, a health endpoint that is a valid path, at least one environment defined, no rule referencing an unimplemented `RuleID`.

An unknown key produces `ErrUnknownPolicyKey` and exit code 3, with a hint naming the closest known key:

```text
Error: unknown key in policy file: environments.production.rules.ci_pas
Hint:  did you mean "ci_passed"?
```

Failing loudly on a typo is the entire justification for this section.

### Rules are Go code, registered by ID

```go
type Rule interface {
    ID() RuleID
    Evaluate(ev readiness.Evidence) (RuleStatus, string, []EvidenceRef)
}
```

Pure, no I/O, no clock. A rule needing information the evidence does not contain requires a new collector (ADR-005), not a network call inside evaluation. The registry is a map from `RuleID` to `Rule`, which means a policy file referencing an unknown rule fails at load time rather than being silently skipped.

### The policy document is hashed

On load, the canonicalised document is hashed and the digest is carried in `EvaluationResult.PolicyDigest` and recorded in provenance. This answers a question the audit trail must be able to answer: *under which version of the rules was this deployment permitted?* Without it, relaxing a policy retroactively rewrites the meaning of every past record.

### Discovery order

`--config`, then `SHIPHOLD_CONFIG`, then `./shiphold.yaml`, then `./.shiphold/config.yaml`. First match wins; the chosen path is logged at debug level and recorded in provenance. The current hardcoded `examples/policy.yaml` means the binary only works from the repository root.

## Consequences

**Makes easy.** Policy diffs are readable in review. Evaluation stays pure and fast. Policies are version-controlled alongside the code they gate. A typo is caught at load, not during an incident.

**Makes harder.** Genuinely novel policy logic requires a Go change and a release, rather than a config edit. This is deliberate: ShipHold's rules are code that has been reviewed and tested, which for a safety tool is a feature. When it becomes a real constraint for real users, that is the signal to revisit the embedded-language option — and it will be a much better-informed decision then.

**Rules out.** Arbitrary expressions, embedded scripting, and any rule whose behaviour is not determined by the Go source.

**Migration.** The current `policy.Policy` struct of four bools becomes an environment-scoped severity map. `internal/config` gains strict decoding, validation, discovery, and hashing. A day's work, and the last comfortable moment to do it.

# ADR-012: Secrets and credential handling

## Status

Accepted

## Context

ShipHold handles a GitHub token, container-registry credentials, a PostgreSQL connection string, and — if signing is ever added (`extended-scope.md` §1.1) — a private key.

Neither `architecture.md` nor `vision.md` mentions any of them. The DiffSage playbook lists this among the habits to carry over verbatim: *"Credentials kept out of application config, never logged, always masked in output."* DiffSage had a `CredentialsRepository`; ShipHold currently has nothing.

Three things about ShipHold make this sharper than usual.

**Policy files live in Git.** They are meant to be committed, reviewed, and diffed. A credential in a policy file is a credential in the repository's history, permanently.

**Provenance records are append-only and permanently stored.** ADR-009 means a record cannot be edited. A token that ends up in a provenance record cannot be redacted — only the whole ledger could be discarded, which defeats its purpose. This is a one-way door, and it is the reason this ADR needs to exist before the provenance work in Week 7, not after.

**The output is designed to be machine-readable.** `--output json` (ADR-014) dumps structured data into CI logs, which are frequently world-readable in open-source projects and broadly readable in most companies. Anything reachable by a JSON marshaller is effectively published.

## Options Considered

**Discipline.** Remember not to log credentials, review for it. Rejected — it fails on the first tired evening, and the failure is permanent because of the append-only ledger.

**A secret manager integration** (Vault, cloud KMS). Correct for a mature product, and too large for v1, and it presumes infrastructure the target user may not have.

**Reference-based configuration plus a type that cannot be printed.** Configuration holds references; the resolved value lives in a type whose formatting methods return a mask.

## Decision

### 1. Configuration holds references, never values

```yaml
registry:
  url: ghcr.io
  credentials: env:GHCR_TOKEN

ci:
  provider: github-actions
  token: env:GITHUB_TOKEN

provenance:
  backend: postgres
  dsn: env:SHIPHOLD_DSN
```

Supported reference schemes in v1: `env:NAME` and `file:/path/to/secret`. The resolver is an interface, so a future secret-manager scheme is an added implementation rather than a change to the config format.

A value that does not parse as a reference is rejected at config load with a specific error:

```text
Error: registry.credentials must be a secret reference, not a literal value
Hint:  use env:GHCR_TOKEN or file:/run/secrets/ghcr_token
       If this token is real, rotate it — it may be in your Git history.
```

Rejecting the literal is what makes this a control rather than a convention. The rotation hint is there because by the time this error appears, the value has usually already been committed.

### 2. `Secret` is a type that resists being printed

```go
package secret

type Secret struct{ v string }

func New(v string) Secret { return Secret{v} }

// Expose is the ONLY way to read the value. Deliberately awkward to write.
func (s Secret) Expose() string { return s.v }

func (s Secret) String() string                { return "***" }
func (s Secret) GoString() string              { return "***" }
func (s Secret) MarshalJSON() ([]byte, error)  { return []byte(`"***"`), nil }
func (s Secret) MarshalYAML() (any, error)     { return "***", nil }
func (s Secret) LogValue() slog.Value          { return slog.StringValue("***") }
```

Every credential in the codebase has this type. `fmt.Printf("%v")`, `%s`, `%+v`, `%#v`, `json.Marshal`, `yaml.Marshal`, and `slog` all produce `***`. Leaking one requires explicitly writing `.Expose()`, which is greppable, reviewable, and hard to do by accident.

`LogValue` matters specifically: `log/slog` (ADR-014) calls it automatically, so a secret passed as a log attribute is masked without the call site having to remember.

A lint rule or a CI grep for `.Expose()` outside `internal/infrastructure/` is worth adding — the only legitimate callers are the adapters that actually authenticate.

### 3. Nothing secret enters evidence or provenance

`readiness.Evidence` and the provenance `Record` are permanently stored (ADR-009). They record *that* authentication occurred and *what* it produced, never the material:

```text
recorded:      ci.authenticated = true
               ci.token_source  = "env:GITHUB_TOKEN"
               registry.auth    = "ghcr.io (token)"
not recorded:  the token
```

This is enforced structurally: no `Secret` field appears in any type reachable from `Record`. A compile-time guarantee beats a review checklist, and given that records cannot be edited afterwards, it is worth the small awkwardness of keeping the types separate.

### 4. Redaction at the output boundary, as a backstop

A final pass over rendered output replaces any resolved secret value found in it with `***`. This should never fire — if it does, something upstream is wrong. It exists because the cost of a leak here is unbounded and the cost of the backstop is negligible.

Error messages from third-party libraries are the realistic case: a `pgx` connection error may echo the DSN, including the password, and that error will travel up through ShipHold's own error wrapping (ADR-004) to the terminal. The backstop catches exactly this.

### 5. What is explicitly out of scope

- Secret-manager integrations. Deferred; the resolver interface makes them additive.
- Secret rotation. Not ShipHold's job.
- Signing keys. Deferred with signing itself (`extended-scope.md` §1.1). When it arrives it needs a real key-management decision, which is most of the work.
- Encryption at rest of the provenance ledger. Records contain no secrets by construction, so there is nothing to encrypt.

## Consequences

**Makes easy.** Policy files are safe to commit by construction, not by care. Logs and JSON output are safe by default. `.Expose()` gives a short, auditable list of every place a credential is actually read.

**Makes harder.** Adapters must call `.Expose()` at the point of use, which is slightly more verbose — and is exactly the friction intended. Config resolution gains a step, and secret resolution failures need their own clear error (`Error: GHCR_TOKEN is not set` with exit code 3).

**Rules out.** Literal credentials in configuration. `Secret` fields in any persisted type. Logging a credential without writing `.Expose()` deliberately.

**Immediate actions.** Create `internal/secret` in Week 1, before any adapter authenticates against anything. Retrofitting this after the GitHub and registry adapters exist means auditing every existing call site, and after Week 7 it means auditing a ledger that cannot be edited.

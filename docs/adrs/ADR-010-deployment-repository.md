# ADR-010: Deployment repository — two implementations

## Status

Accepted

## Context

Deployment provenance must be persisted. PostgreSQL was chosen early and is named in the project description, and it is a sound choice: transactional, queryable, constraint-enforcing, and a reasonable thing for a target team to already run.

But there is a problem with choosing it *first*, and the DiffSage playbook names it precisely: "an interface shaped by one implementation is a guess; shaped by two, it's tested." DiffSage's `BaseProvider` is well designed and has only Gemini behind it, and the playbook's instruction is to prove any adapter seam against two real implementations before anything else depends on it.

ShipHold has four such seams — CI provider, registry, deployment target, repository — and a ten-week budget that cannot build two adapters for each. So the question is not *whether* to follow the rule but *where* to spend it.

There is a second, more practical problem. If `DeploymentRepository` has only a PostgreSQL implementation, then every application-level test that touches persistence needs a database. Testcontainers makes that workable, but it means the test suite requires a running Docker daemon, takes tens of seconds instead of milliseconds, and cannot run on a machine where Docker is unavailable. Slow test suites get run less often, and a safety tool with a rarely-run test suite is a contradiction.

## Options Considered

**PostgreSQL only.** Simplest, and it makes the interface a PostgreSQL API with a generic name, and it makes Docker a prerequisite for testing anything.

**In-memory fake only, alongside PostgreSQL.** The usual approach, and it is what most projects do. It is weaker than it looks: an in-memory map is not a second *implementation*, it is a test double. It never exercises serialisation, never encounters a partial write, never has to think about ordering across a process restart. It does not test the interface; it tests around it.

**A file-backed implementation plus PostgreSQL.** A real second implementation with real persistence semantics — serialisation, durability, append ordering, crash behaviour — that happens to need no server.

## Decision

**Implement `DeploymentRepository` twice: file-backed first, PostgreSQL second.**

### The port

```go
// internal/application/provenance — declared by its consumer (ADR-001)

type DeploymentRepository interface {
    // Append is the only write. There is no Update, no Delete (ADR-009).
    Append(ctx context.Context, r Record) error

    Head(ctx context.Context) (Record, bool, error)
    Get(ctx context.Context, id ID) (Record, error)
    List(ctx context.Context, f Filter) ([]Record, error)
    LastSuccessful(ctx context.Context, app string, env Environment) (Record, error)

    // Walk yields records in Seq order, for chain verification.
    Walk(ctx context.Context, from int64, fn func(Record) error) error
}
```

Deliberately narrow. No transactions in the signature, no SQL concepts, no driver types. `LastSuccessful` exists because it is the one query rollback needs, and expressing it as a method rather than as a `Filter` keeps the rollback service from knowing how records are indexed.

`Walk` exists for `shiphold verify` and is a streaming interface specifically so that verification does not load the entire history into memory.

### 1. `FileRepository` — JSON Lines

One canonical JSON object per line, appended with `O_APPEND`. Half a day's work, and it buys:

- **A test suite with no infrastructure dependency.** Application and provenance tests run in milliseconds against a temp directory. This alone likely saves more time over ten weeks than the implementation costs.
- **Zero-dependency mode.** Someone evaluating ShipHold can run it without provisioning PostgreSQL. For a portfolio project this matters a great deal — the reviewer who has to set up a database will not set up the database.
- **A genuine test of the abstraction.** Two real implementations with different failure modes, per the playbook rule, on the seam where it is cheapest to satisfy.
- **A natural fit for the append-only ledger.** JSON Lines *is* an append-only log; the storage model and the domain model agree.

Limits, stated plainly: single-process only (guarded by the deployment lock in ADR-011), no indexed queries, `List` is a scan. All acceptable at the scale a single-host deployment tool operates at — a busy team might write a few thousand records a year.

### 2. `PostgresRepository`

The production implementation. Indexed queries, transactional appends, constraints enforced in the database rather than in application code:

```sql
CREATE TABLE deployment_record (
    seq             BIGSERIAL PRIMARY KEY,
    deployment_id   TEXT NOT NULL UNIQUE,
    application     TEXT NOT NULL,
    environment     TEXT NOT NULL,
    commit_sha      TEXT NOT NULL,
    image_digest    TEXT NOT NULL,
    policy_digest   TEXT NOT NULL,
    outcome         TEXT NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL,
    ended_at        TIMESTAMPTZ NOT NULL,
    body            JSONB NOT NULL,     -- canonical record
    prev_hash       TEXT NOT NULL,
    record_hash     TEXT NOT NULL UNIQUE
);

CREATE INDEX ON deployment_record (application, environment, seq DESC);

-- ADR-009: append-only, enforced by the database
CREATE FUNCTION deployment_record_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'deployment_record is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER no_update_or_delete
    BEFORE UPDATE OR DELETE ON deployment_record
    FOR EACH ROW EXECUTE FUNCTION deployment_record_immutable();
```

Driver: `pgx`. Migrations: plain numbered SQL files applied by a small embedded runner, not an ORM — the schema is small and explicit SQL is easier to review than generated DDL.

Tested with Testcontainers, in a build-tagged integration suite that the default `go test ./...` does not run.

### Both must pass the same suite

A shared contract test, run against both implementations:

```go
func TestRepositoryContract(t *testing.T, newRepo func(t *testing.T) DeploymentRepository) {
    // append and read back
    // Seq is dense and monotonic
    // Walk yields Seq order
    // hash chain links correctly across appends
    // Get on unknown ID returns a recognisable not-found
    // LastSuccessful ignores failed and rolled-back deployments
    // duplicate DeploymentID is rejected
}
```

This is what makes the two implementations meaningful rather than merely two. If a behaviour is only asserted against PostgreSQL, it is not part of the interface's contract — it is a PostgreSQL detail that happened to leak.

### Selection

```yaml
provenance:
  backend: postgres          # postgres | file
  dsn: env:SHIPHOLD_DSN      # a reference, never a value (ADR-012)
  path: ./.shiphold/ledger.jsonl
```

Default is `file`, so the tool works on first run with no setup. `postgres` is what a real deployment uses.

## Consequences

**Makes easy.** Fast, infrastructure-free tests for everything above the repository. A zero-setup evaluation path. A genuinely proven abstraction rather than a claimed one. The append-only model enforced identically in both backends.

**Makes harder.** Two implementations to keep correct — mitigated by the shared contract test, which is where the real specification lives. Some queries expressible in SQL are inefficient against a file scan; `List` filters in Go rather than in the storage layer.

**Rules out.** SQL in the application layer. Driver types crossing the port. Any write path other than `Append`.

**On the other three seams.** `CIProvider`, `RegistryProvider`, and `DeploymentTarget` each ship with one implementation in v1. Their ADRs should say so explicitly, and should name what the second implementation would be and which parts of the signature exist because of it. Designing against a hypothetical second implementation and *saying that is what you did* is a defensible position. Shaping an interface around one implementation and calling it an abstraction is not.

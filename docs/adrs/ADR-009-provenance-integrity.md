# ADR-009: Provenance record and ledger integrity

## Status

Accepted

## Context

ShipHold is described, in its own README and project description, as a **provenance engine** producing an **auditable trail**. That is a strong word, and as originally designed the architecture does not earn it.

The design says deployment history is persisted to PostgreSQL. As specified, that history is a set of rows that anyone with database access can edit, and an audit log that can be silently rewritten is a log, not an audit trail. The distinction matters precisely in the situation the feature exists for: a post-incident review, where the question is *what was actually deployed*, and where the honest answer must be defensible even if someone had an incentive to change it.

The DiffSage playbook anticipated this. Its suggested ADR-007 says storage design deserves an explicit decision, and that "given the domain, append-only or tamper-evident storage is worth deciding on explicitly rather than defaulting to a plain writable file."

The good news is that closing this gap is cheap — on the order of eighty lines of Go — and it converts a generic claim ("we store deployment history") into a specific, demonstrable one ("deployment history is tamper-evident, and here is the command that proves it").

## Options Considered

**Plain table, mutable rows.** What was implicitly planned. Cheapest, and it makes the word "auditable" unsupportable.

**Append-only by convention.** No `UPDATE` or `DELETE` in application code, enforced by database grants. Better, and it protects against the application, not against anyone with database access — which is the threat that matters.

**Hash-chained records.** Each record includes the hash of its predecessor, so any alteration invalidates every subsequent hash. Detection is a single pass.

**Hash chain plus signatures plus external anchoring.** The complete version. Signing gives authenticity on top of integrity; anchoring the chain head outside the database defends against a wholesale rewrite. Correct, and larger than a ten-week budget — signing is mostly a key-management project.

## Decision

**A hash-chained, append-only ledger in v1.** Signing and anchoring are deferred to `extended-scope.md` §§1.1–1.2 with their rationale recorded.

### The provenance record

One record per deployment, written once, never updated:

```go
type Record struct {
    Seq          int64        // dense, monotonic, gapless
    DeploymentID ID

    Application  string
    Environment  Environment
    CommitSHA    string
    Branch       string
    CIRunID      string
    CIRunURL     string

    ImageRef     string       // what was requested (may be a mutable tag)
    ImageDigest  string       // what was actually deployed — the identity
    ConfigDigest string
    PolicyDigest string       // which policy permitted this (ADR-006)

    Evidence     readiness.Evidence
    Decision     EvaluationResult   // rule-level, not a sentence
    Events       []deployment.Event // the full lifecycle log (ADR-007)

    Outcome      Status
    StartedAt    time.Time
    EndedAt      time.Time
    Actor        string       // who or what invoked it
    ShipHoldVersion string    // which build produced this record

    Restores     *ID          // set on rollback

    PrevHash     string       // hash of record Seq-1; "" for the genesis record
    RecordHash   string       // SHA-256(canonical(record without RecordHash) || PrevHash)
}
```

`ImageDigest` is the deployment's identity; `ImageRef` is only what the human typed. A mutable tag may point somewhere entirely different by the time anyone reads the record, which is the failure mode immutable digests exist to prevent.

`PolicyDigest` answers a question an audit trail must be able to answer: *under which version of the rules was this permitted?* Without it, relaxing a policy silently rewrites the meaning of every past record.

`ShipHoldVersion` matters more than it looks — when a record is surprising, the first question is whether the code that wrote it had a bug.

### Canonical serialisation

Hashing requires one byte sequence per record. Not `encoding/json` with map iteration, which is non-deterministic.

```text
- object keys sorted lexicographically
- no insignificant whitespace
- timestamps RFC 3339 in UTC, fixed precision
- explicit nulls, never omitted fields
- UTF-8, no escaping beyond what JSON requires
```

This is a stored format. Changing it invalidates every existing hash, so it gets a version and is covered by golden-file tests from the first commit. **This is the part most likely to be got subtly wrong**, and the part where a golden test written on day one saves a day later.

### The chain

```text
Record 1:  PrevHash = ""          RecordHash = H(canonical(r1) || "")
Record 2:  PrevHash = r1.Hash     RecordHash = H(canonical(r2) || r1.Hash)
Record 3:  PrevHash = r2.Hash     RecordHash = H(canonical(r3) || r2.Hash)
```

Editing record 2 changes its hash, which breaks record 3's `PrevHash`, and so on to the head. A single sequential pass detects it and names the first broken link.

### `shiphold verify`

```text
$ shiphold verify
Verified 186 records (seq 1–186).
Chain intact. Head: sha256:4a7e91c…

$ shiphold verify
✗ Chain broken at seq 185.
  Record 185 PrevHash:     sha256:9f2c1a…
  Record 184 RecordHash:   sha256:e81b04…
  Record 184 has been altered or replaced since it was written.
```

This small command is what makes the tamper-evidence claim demonstrable rather than asserted, and it is the most compelling thirty seconds of any demo of this project. Exit code 7 on failure (ADR-004).

### Append-only is enforced, not merely intended

- Repository interface exposes `Append` and reads. There is no `Update` and no `Delete`.
- PostgreSQL: a `BEFORE UPDATE OR DELETE` trigger that raises an exception, plus a grant that omits those privileges for the application role.
- File backend: opened `O_APPEND`, one JSON object per line.
- `Seq` is dense and gapless, so a *deleted* record is as detectable as an altered one. A chain alone does not catch truncation of the tail; comparing the head against a recorded high-water mark does.

### Honest limits

Worth being able to state precisely, because being clear about what a security property does *not* give you is more convincing than the property itself:

- **v1 detects record-level tampering. It does not prevent it.** Someone with write access can alter a record; they cannot do so undetectably.
- **A full-database rewrite is not detected.** Recompute every hash from record 1 and the chain is internally consistent again. External anchoring of the head (`extended-scope.md` §1.2) is what closes this, and it is deliberately out of v1.
- **There is no authenticity.** The chain proves records were not altered, not who wrote them. That requires signing (§1.1).

## Consequences

**Makes easy.** "Tamper-evident audit trail" becomes a defensible claim with a command behind it. Deployment history is reconstructable end to end. A recorded decision can be re-evaluated later from its recorded evidence and must produce the same result (ADR-003's purity requirement is what makes this work).

**Makes harder.** Records cannot be corrected. A record written with a bug stays, and the fix is a later compensating record — which is how ledgers work and is the right trade for an audit trail. Canonical serialisation must be exactly right and frozen.

**Rules out.** Mutable provenance rows. Non-deterministic serialisation. `time.Now()` inside record construction (injected clock, per ADR-005 and ADR-007). Secrets anywhere in a record — it is written once and permanently, so a leaked credential in a record cannot be redacted (ADR-012).

**Cost.** Roughly: canonical serialisation and golden tests (half a day), chain construction and append (half a day), `verify` (half a day), PostgreSQL triggers and grants (a few hours). Call it two days, in Week 7, for the property that makes the product name accurate.

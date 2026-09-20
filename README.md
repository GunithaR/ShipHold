# Architectural Decision Records

Each record documents one decision, the alternatives that were weighed, and the consequences accepted. Records are written when the decision is made, not reconstructed afterwards.

A record is never edited to reflect a change of mind. If a decision is reversed, the old record's status becomes `Superseded by ADR-NNN` and a new record explains what changed and why. A superseded ADR is evidence of learning, not a mistake to hide.

## Status values

| Status | Meaning |
| --- | --- |
| `Proposed` | Decided in principle, pending evidence (usually a spike). Do not build load-bearing work on it. |
| `Accepted` | In force. The codebase should reflect it. |
| `Superseded by ADR-NNN` | No longer in force. Kept for the record. |

## Index

| ADR | Title | Status |
| --- | --- | --- |
| [000](ADR-000-naming.md) | Project naming — ShipHold | Accepted |
| [001](ADR-001-layered-architecture.md) | Layered architecture and the dependency rule | Accepted |
| [002](ADR-002-go-and-cobra.md) | Go and Cobra for the CLI | Accepted |
| [003](ADR-003-readiness-decision-model.md) | Readiness decision model | Accepted |
| [004](ADR-004-errors-and-exit-codes.md) | Error taxonomy and exit codes | Accepted |
| [005](ADR-005-evidence-collection.md) | Evidence collection and provider ports | Accepted |
| [006](ADR-006-typed-yaml-policy.md) | Typed YAML policy model | Accepted |
| [007](ADR-007-deployment-lifecycle.md) | Deployment lifecycle and event log | Accepted |
| [008](ADR-008-traffic-switching.md) | Docker and reverse-proxy traffic switching | **Proposed** |
| [009](ADR-009-provenance-integrity.md) | Provenance record and ledger integrity | Accepted |
| [010](ADR-010-deployment-repository.md) | Deployment repository: two implementations | Accepted |
| [011](ADR-011-concurrency-and-recovery.md) | Deployment concurrency, locking and recovery | Accepted |
| [012](ADR-012-secrets.md) | Secrets and credential handling | Accepted |
| [013](ADR-013-environments.md) | Environments as a first-class domain concept | Accepted |
| [014](ADR-014-observability.md) | Structured logging and machine-readable output | Accepted |
| [015](ADR-015-ai-as-explanation-layer.md) | AI strictly as an explanation layer | Accepted |

## Template

```markdown
# ADR-NNN: Title

## Status
Accepted

## Context
What forces are at play. What makes this a decision rather than an obvious choice.

## Options Considered
Each option, with its real advantages — not strawmen.

## Decision
What was chosen, stated plainly.

## Consequences
What this makes easy, what it makes hard, and what it rules out.
```

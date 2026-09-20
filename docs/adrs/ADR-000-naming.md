# ADR-000: Project Naming — ShipHold

## Status

Accepted

## Context

The project was initially referred to internally as "Deployment Safety Engine" — a descriptive working title, not a product name. Before implementation begins, the project needs a real name for the repository, CLI binary, module path, and any public-facing artifact (README, demo, portfolio listing).

Naming criteria considered:

- Should suggest what the tool does without being literal or generic
- Should work naturally as a CLI binary and command prefix (`<name> check`, `<name> deploy`)
- Should be distinctive enough to search for and own
- Should be free of conflicts with existing DevOps/deployment tooling
- Should not misrepresent the project's scope (e.g. should not sound like a GitOps controller or Kubernetes platform, which this project explicitly is not — see vision.md Section 6)

## Options Considered

Several naming directions were explored before narrowing down:

- **Gate/guard metaphors** (Gatekeeper, Shipgate, Cleargate) — generic, mostly already used by existing tools
- **State/transition metaphors** (Groundstate, Stateshift) — conceptually strong but less intuitive as a CLI name
- **ShipLock** — a canal lock metaphor (a ship only passes when water levels equalize and conditions are verified). Strong conceptual fit, but `shiplock.io` is an existing registered product (an on-chain accountability platform for builders). Different domain, no functional overlap, but the namespace conflict was judged not worth the confusion for search visibility and interview credibility.
- **Deadbolt, Lockpass, Gatepass** — considered as lock-metaphor alternatives after the ShipLock conflict, not pursued further once ShipHold was proposed

## Decision

The project is named **ShipHold**.

The name carries two intentional, simultaneous meanings, both genuinely relevant to what the tool does:

1. **Hold as a verb** — a controlled pause before a deployment is allowed to proceed. A deployment does not go until the hold is cleared by the readiness gate. This maps directly to the `check` → `PASS`/`WARN`/`BLOCK` flow.
2. **Hold as a noun** — the cargo hold of a ship, where goods are secured, inspected, and accounted for before transit. This maps directly to the provenance and audit trail model — deployment records, immutable image digests, and known-good state are the "cargo" the system tracks.

A namespace search at the time of naming found no existing DevOps, CI/CD, or developer tooling products named ShipHold. The term appears only in unrelated maritime/cargo engineering contexts.

## Consequences

- CLI binary and command prefix: `shiphold` (`shiphold check`, `shiphold diff`, `shiphold deploy`, `shiphold status`, `shiphold history`, `shiphold inspect`, `shiphold rollback`)
- Repository, module path, and all documentation use ShipHold consistently from this point forward
- The tone the name signals — deliberate, controlled, safety-first — aligns with the product's actual positioning (Section 6 of vision.md) and should be preserved in README copy, demo framing, and any interview narrative. Avoid speed-centric framing ("fast," "instant deploy") that would undercut the name's meaning.
- No functional or legal conflict exists with the unrelated `shiplock.io` product from the earlier naming exploration, since that name was not chosen.

## Notes

This ADR is recorded primarily because the reasoning behind the name — not just the name itself — is a legitimate design decision worth being able to explain, the same way ADR-009 in DiffSage documented a repositioning decision rather than only a technical one.

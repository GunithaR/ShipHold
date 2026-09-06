# ShipHold

**A deployment safety and provenance engine for teams that want stronger deployment discipline without adopting a full Kubernetes platform.**

> Status: early development. This README will be updated as capabilities land.

---

## What ShipHold does

Before your code goes to production, ShipHold checks whether it's actually safe to deploy — CI status, image existence, configuration state, health-check readiness — and gives you a deterministic decision: `PASS`, `WARN`, or `BLOCK`, with the evidence behind it.

When a deployment proceeds, ShipHold starts the new version alongside the old one, verifies it's healthy, switches traffic only after that verification passes, and records exactly what was deployed — commit, image digest, configuration version, and outcome — so a rollback restores a known-good state instead of a guess.

```bash
shiphold check
shiphold diff
shiphold deploy
shiphold status
shiphold history
shiphold inspect <id>
shiphold rollback <id>
```

---

## Why ShipHold?

The name has two intentional meanings:

- **Hold as a verb** — a controlled pause before a deployment proceeds. The deployment remains held until its safety conditions are satisfied.
- **Hold as a noun** — the cargo hold of a ship, where goods are secured, inspected, and accounted for before transit.

The first represents the deployment gate.

The second represents deployment provenance and the audit trail.

See `docs/adr/ADR-000-naming.md` for the full reasoning.

---

## The Problem

Modern engineering teams already have tools for:

```text
Git
CI/CD
Docker
Container registries
Cloud infrastructure
Monitoring
```

But these tools do not automatically provide a coherent deployment safety layer.

Without ShipHold:

```text
Git
 ↓
CI
 ↓
Docker
 ↓
Production
```

The human has to connect the pieces.

With ShipHold:

```text
Git
 ↓
CI
 ↓
┌───────────────────────┐
│ Deployment Safety     │
│                       │
│ Readiness             │
│ Policy                │
│ Diff                  │
│ Deployment            │
│ Verification          │
│ Provenance            │
│ Rollback              │
└───────────┬───────────┘
            ↓
       Proxy → Docker
```

ShipHold becomes the coordination layer.

---

## Safe Deployment

ShipHold uses a reverse proxy to support a blue/green-style deployment flow.

```text
                    Public Endpoint
                          │
                          ▼
                   ┌────────────┐
                   │   Traefik  │
                   └─────┬──────┘
                         │
               ┌─────────┴─────────┐
               ▼                   ▼
          API v1 (active)      API v2 (candidate)
```

The sequence is:

```text
Start candidate
     ↓
Health check candidate
     ↓
Switch traffic
     ↓
Verify routed traffic
     ↓
Stop previous version
     ↓
Record deployment
```

If the candidate fails:

```text
Candidate fails
     ↓
Do not switch traffic
     ↓
Keep previous version active
     ↓
Remove candidate
     ↓
Record failure
```

Traefik is infrastructure used to perform traffic routing. ShipHold does not implement its own reverse proxy.

---

## GitHub Integration

GitHub/GitHub Actions provides CI evidence used by ShipHold's readiness and policy system.

Conceptually:

```text
Git commit
    ↓
GitHub Actions
    ↓
CI result
    ↓
ShipHold evidence
    ↓
Policy evaluation
    ↓
PASS / WARN / BLOCK
```

The core v1 architecture does **not** require a full GitHub App.

A GitHub App may be introduced later if capabilities such as repository installation, webhooks, or pull-request checks justify it.

---

## What ShipHold is not

ShipHold is not:

- a GitOps controller;
- a Kubernetes platform;
- a CI system;
- a container orchestrator;
- a reverse proxy;
- a monitoring platform;
- an incident-response/RCA tool;
- an AI system making deployment decisions.

It is a **deployment safety and provenance layer** around existing delivery infrastructure.

---

## Architecture

ShipHold is written in Go using a layered architecture:

```text
CLI
 ↓
Application Services
 ↓
Domain
 ↓
Ports / Interfaces
 ↓
Infrastructure Adapters
```

The core domain remains independent of Docker, GitHub, PostgreSQL, and vendor SDKs.

External systems are accessed through explicit interfaces so that infrastructure can be replaced without rewriting the core deployment logic.

See `docs/architecture.md` for the full design.

---

## Initial Technology Scope

```text
Go
Git
GitHub Actions
Container Registry
Docker / Docker Compose
Traefik
PostgreSQL
Testcontainers
GitHub Actions CI
```

The initial deployment target is a single-host Docker-based environment.

Kubernetes, additional CI providers, additional registries, and full GitOps functionality are future extensions rather than v1 requirements.

---

## Documentation

- `docs/vision.md` — product vision, philosophy, scope, and success criteria
- `docs/architecture.md` — technical architecture, domain model, deployment lifecycle, testing strategy, and milestones
- `docs/adr/` — architectural decision records
- `docs/development/` — development and setup documentation
- `docs/operations/` — deployment, rollback, troubleshooting, and operational procedures

---

## Development Philosophy

ShipHold follows:

```text
Design
  ↓
Define behavior
  ↓
Test
  ↓
Implement
  ↓
Integrate
  ↓
Document
  ↓
Verify through CI
```

A feature is not considered complete merely because its code works.

The relevant:

```text
Domain behavior
+
Implementation
+
Tests
+
Documentation
+
CI verification
```

must remain aligned.

---

## Installation

Not yet published.

Installation instructions will be added once the first CLI skeleton is available.

---

## License

TBD.

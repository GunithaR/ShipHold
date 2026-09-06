# ShipHold — Architecture

## 1. Purpose

ShipHold is a Go-based deployment safety and provenance system for small and medium teams using Git, CI, container registries, and Docker-based deployment environments.

The name reflects two intentional meanings: **hold** as a controlled pause before a deployment proceeds (the safety gate), and **hold** as the cargo compartment of a ship where goods are secured and inspected before transit (the provenance and audit trail). See ADR-000 for the naming rationale.

Its purpose is to make deployment state transitions:

- verifiable before deployment,
- controlled during deployment,
- observable after deployment,
- auditable over time,
- and reversible when necessary.

The system is intentionally designed as a foundation that can later expand into Kubernetes/GitOps adapters, additional CI providers, registries, and optional AI explanations without rewriting the core.

---

## 2. Core Engineering Principle

> A deployment is a state transition. Make that transition verifiable, auditable, and reversible.

The system separates:

1. **Evidence collection** — what is true about the proposed deployment?
2. **Evaluation** — what do policies say about that evidence?
3. **Decision** — PASS, WARN, or BLOCK.
4. **Execution** — perform the deployment only when permitted.
5. **Verification** — confirm that the resulting state is healthy.
6. **Provenance** — persist exactly what happened.
7. **Recovery** — restore a previously known-good deployment state.

---

## 3. Product Boundary

The product is not intended to replace GitHub Actions, Docker, Kubernetes, Terraform, Prometheus, or incident-management platforms.

It sits around existing delivery infrastructure as a deployment safety layer.

```text
Git
 │
 ▼
CI/CD
 │
 ▼
┌──────────────────────────┐
│ ShipHold │
├──────────────────────────┤
│ Readiness                │
│ Diff / Policy            │
│ Deployment               │
│ Verification             │
│ Audit / Provenance       │
│ Rollback                 │
└────────────┬─────────────┘
             │
             ▼
        Reverse Proxy
             │
             ▼
          Docker
             │
             ▼
       Running Service
```

The initial target is a single-host Docker-based deployment environment, optionally using Docker Compose, with a lightweight reverse proxy such as Traefik. Kubernetes is a future deployment adapter, not a v1 requirement.

GitHub/GitHub Actions is the initial CI integration. A full GitHub App is not required for the core v1 architecture. If future capabilities require repository installation, webhooks, PR checks, or similar functionality, a GitHub App can be introduced behind the appropriate provider boundary.

---

## 4. Architectural Goals

### 4.1 Primary goals

- Keep deployment decisions deterministic.
- Keep external systems behind explicit interfaces.
- Keep domain logic independent of Docker, GitHub, PostgreSQL, and vendor SDKs.
- Make deployment state durable and reconstructable.
- Preserve known-good state during failed deployments.
- Make failure handling explicit.
- Make the system easy to test without requiring real infrastructure for every test.
- Use real infrastructure integration tests where mocks are insufficient.
- Make future deployment targets and providers replaceable.
- Maintain documentation and architecture records throughout development.
- Keep the first release narrow enough to complete within three months of part-time development.

### 4.2 Non-goals

- Building a reverse proxy.
- Building a Kubernetes platform.
- Building a monitoring platform.
- Building an incident-response/RCA platform.
- Replacing CI/CD.
- Building an infrastructure-provisioning engine.
- Building a custom policy language.
- Making AI responsible for deployment decisions.
- Building a full GitHub App unless a concrete v1 capability requires it.

---

## 5. Initial Technical Decisions

The following decisions are deliberately made early because they remove major implementation uncertainty.

### 5.1 Docker traffic switching

A reverse proxy owns the public application endpoint.

The deployment engine does not expose each application version on the same host port.

Conceptually:

```text
                    ┌── API v1
                    │
Client → Proxy ─────┤
                    │
                    └── API v2
```

The proxy is responsible for routing.

The Go deployment engine is responsible for:

- starting the new version,
- verifying its health,
- switching the active route,
- verifying the switched traffic,
- stopping the old version,
- recording the deployment.

Traefik is the preferred initial implementation because of its Docker integration and label-based routing.

Caddy or another reverse proxy may be substituted if the technical spike shows a better fit.

The project does not implement its own proxy.

### 5.2 Policy engine

The initial policy system is deliberately simple.

Policies are represented by typed YAML configuration and evaluated by deterministic Go rules.

No custom DSL.
No generic expression language.
No complex AST parser.

Example:

```yaml
policy:
  require_ci_pass: true
  require_image: true
  require_immutable_digest: true
  require_healthcheck: true
```

Future policy complexity can be added only if real use cases justify it.

### 5.3 Integration testing

Unit tests remain fast and isolated.

Real infrastructure behavior is tested through integration tests using `testcontainers-go` where appropriate.

Initial integration targets include:

- PostgreSQL,
- Docker/container lifecycle behavior where practical,
- supporting test infrastructure.

External GitHub API behavior should use contract/integration tests carefully, avoiding dependence on a personal developer token for the normal test suite.

The default CI suite should run without requiring manually configured external credentials.

### 5.4 Deployment state and idempotency

Deployment is a stateful operation and must explicitly handle interrupted execution.

The design must consider:

- duplicate deployment requests,
- process interruption,
- Docker daemon failure,
- proxy failure,
- health-check timeout,
- persistence failure,
- partial deployment state.

Deployment operations should be designed to be as idempotent as practical.

---

## 6. Layered Architecture

The preferred architecture is:

```text
CLI
 │
 ▼
Application Services
 │
 ▼
Domain
 │
 ▼
Ports / Interfaces
 │
 ▼
Infrastructure Adapters
```

### CLI

Responsible for:

- parsing arguments,
- formatting output,
- mapping user commands to application services,
- translating application errors into user-facing messages.

The CLI must not contain deployment business logic.

### Application layer

Responsible for orchestration.

Examples:

- `CheckService`
- `DiffService`
- `DeploymentService`
- `RollbackService`
- `HistoryService`

The application layer coordinates domain operations and external ports.

### Domain layer

Contains the core concepts and rules:

- Deployment
- DeploymentDiff
- ReadinessCheck
- ReadinessResult
- ReadinessDecision
- DeploymentPolicy
- HealthCheck
- DeploymentEvent
- DeploymentArtifact
- RollbackRequest

The domain should not import Docker SDKs, GitHub clients, database drivers, etc.

### Infrastructure layer

Contains concrete implementations:

- local Git adapter
- GitHub Actions adapter
- GHCR adapter
- Docker adapter
- Traefik/proxy adapter or deployment-routing adapter
- PostgreSQL repository
- filesystem/configuration adapter

---

## 7. Proposed Repository Structure

The exact structure may evolve as implementation teaches us more.

The initial boundary should be:

> Module path is a placeholder until the repository is created: `github.com/<username>/shiphold`. Replace `<username>` throughout `go.mod` and import paths once the GitHub repo exists.

```text
shiphold/
├── cmd/
│   └── shiphold/
│       └── main.go
│
├── internal/
│   ├── domain/
│   │   ├── deployment/
│   │   ├── readiness/
│   │   ├── policy/
│   │   ├── configuration/
│   │   └── audit/
│   │
│   ├── application/
│   │   ├── check/
│   │   ├── diff/
│   │   ├── deploy/
│   │   ├── rollback/
│   │   └── history/
│   │
│   ├── infrastructure/
│   │   ├── git/
│   │   ├── github/
│   │   ├── registry/
│   │   ├── docker/
│   │   ├── proxy/
│   │   └── postgres/
│   │
│   ├── config/
│   └── logging/
│
├── tests/
│   ├── integration/
│   └── e2e/
│
├── docs/
│   ├── architecture/
│   ├── adr/
│   ├── development/
│   ├── operations/
│   └── decisions/
│
├── examples/
├── .github/
│   └── workflows/
├── README.md
├── go.mod
└── go.sum
```

**Do not create every package merely because it appears in this diagram.**

Packages should be created when the corresponding capability or boundary actually exists.

The repository structure is a design guide, not an implementation checklist.

---

## 8. Core Domain Model

The initial domain vocabulary should include:

```text
Deployment
DeploymentStatus
DeploymentResult

DeploymentArtifact
DeploymentConfiguration

ReadinessCheck
ReadinessResult
ReadinessDecision

DeploymentDiff
DeploymentPolicy

HealthCheck
HealthResult

DeploymentEvent
RollbackRequest
```

The proxy should generally remain an infrastructure concern. Domain logic should express concepts such as "activate deployment" rather than Traefik-specific labels.

### Readiness decision

The decision should be a domain type:

```text
PASS
WARN
BLOCK
```

The result should contain evidence and reasons rather than only a boolean.

Conceptually:

```text
ReadinessResult
├── checks
├── warnings
├── blocking reasons
├── policy evaluation
└── decision
```

---

## 9. Evidence → Evaluation → Decision

This is one of the most important architectural boundaries.

```text
External systems
      │
      ▼
Evidence
      │
      ▼
Policy / Evaluation
      │
      ▼
PASS / WARN / BLOCK
```

For example:

```text
Git commit = 8f91a22
CI status = PASS
Image exists = true
Image digest = sha256:abc...
Health endpoint configured = true
```

becomes evidence.

A policy may then evaluate:

```text
CI must pass       ✓
Image must exist   ✓
Digest required    ✓
```

and produce:

```text
PASS
```

The Docker deployment engine must not independently invent its own readiness logic.

---

## 10. Provider Interfaces

External systems should be represented through narrow interfaces.

### Git

```text
GitProvider
```

Initial implementation:

```text
LocalGitProvider
```

### CI

```text
CIProvider
```

Initial:

```text
GitHubActionsProvider
```

### Registry

```text
RegistryProvider
```

Initial:

```text
GHCRProvider
```

### Deployment

```text
DeploymentTarget
```

Initial:

```text
DockerDeploymentTarget
```

### Traffic routing

The deployment target may depend on a routing abstraction rather than directly manipulating Traefik.

Conceptually:

```text
TrafficRouter
└── TraefikRouter
```

This keeps the deployment service independent from the chosen proxy.

### Persistence

```text
DeploymentRepository
```

Initial:

```text
PostgreSQLDeploymentRepository
```

This prevents vendor implementations from spreading into business logic.

---

## 11. Docker + Reverse Proxy Deployment Model

The v1 deployment topology is:

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
          API v1 (blue)       API v2 (green)
```

Only the proxy owns the host/public port.

Application containers communicate through the Docker network and are not required to bind their application port directly to the host.

### Deployment sequence

```text
1. Validate readiness
2. Create deployment record
3. Start new container
4. Wait for startup
5. Run health/readiness checks
6. Activate new route
7. Verify traffic through proxy
8. Stop old container
9. Persist successful deployment state
```

### Failed deployment

```text
1. Start new container
2. Health check fails
3. Do not activate route
4. Remove failed container
5. Keep old container serving traffic
6. Record failure
```

This is the initial blue/green-style deployment strategy.

---

## 12. Reverse Proxy Technical Spike

Because traffic switching is a critical implementation dependency, it must be proven before the main deployment engine is built around it.

Early spike:

```text
Docker network
├── Traefik
├── sample-app:v1
└── sample-app:v2
```

Prove:

```text
v1 active
  ↓
start v2
  ↓
health-check v2
  ↓
route traffic to v2
  ↓
verify v2
  ↓
stop v1
```

Also prove failure:

```text
v2 starts
  ↓
health-check fails
  ↓
v1 remains active
  ↓
v2 removed
```

If Traefik introduces unacceptable complexity, the routing adapter can be changed before the deployment domain is coupled to it.

---

## 13. Deployment Lifecycle

The deployment lifecycle should be explicit.

```text
REQUESTED
    │
    ▼
VALIDATING
    │
    ▼
READY / BLOCKED
    │
    ▼
DEPLOYING
    │
    ▼
VERIFYING
    │
 ┌──┴────┐
 ▼       ▼
SUCCESS  FAILED
```

Additional operational states may be introduced when implementation requires them, such as:

```text
ROLLING_BACK
ROLLBACK_FAILED
INTERRUPTED
```

Do not prematurely model dozens of states.

---

## 14. Health-Gated Deployment

The deployment engine must never immediately replace the currently active version.

The proxy maintains the public endpoint.

```text
Active v1
   │
   ▼
Start v2
   │
   ▼
Health verification
   │
 ┌─┴────┐
 ▼      ▼
PASS   FAIL
 │      │
 ▼      ▼
Switch Keep v1
route   active
 │
 ▼
Verify routed traffic
 │
 ▼
Stop v1
```

The exact proxy configuration is an infrastructure detail.

The deployment domain only needs to know whether activation and verification succeeded.

---

## 15. Deployment Provenance

Every deployment should have a durable identity.

A deployment record should capture enough information to answer:

> What exactly was deployed?

At minimum:

```text
Deployment ID
Application
Git commit SHA
Branch
CI run
Image reference
Image digest
Configuration version
Readiness result
Policy decision
Start time
End time
Deployment result
Health verification
Events/errors
```

Mutable tags should not be treated as sufficient deployment identity when an immutable image digest is available.

---

## 16. Rollback Model

Rollback is a new operation that restores a previously recorded state.

It should not erase history.

Example:

```text
Deployment 184
      │
      ▼
Deployment 185
      │
      ▼
Rollback requested
      │
      ▼
Deployment 186
      │
      └── restores state represented by 184
```

Rollback should be validated and health-checked using the same safety mechanisms as a normal deployment.

The system should avoid assuming that an application image alone defines the complete deployment state.

---

## 17. Configuration and Policy

Deployment configuration should be typed and validated.

Conceptually:

```yaml
application:
  name: payments-api

deployment:
  target: docker

  health:
    endpoint: /health
    timeout: 60s

checks:
  ci:
    required: true

  image:
    required: true

policy:
  require_ci_pass: true
  require_immutable_digest: true
  require_healthcheck: true
```

The initial policy engine is deliberately rule-based.

Rules should operate on typed domain data.

Do not introduce:

- custom DSLs,
- embedded scripting,
- arbitrary expression evaluation,
- complex AST parsing.

Configuration differences between environments should not automatically be treated as errors.

The future configuration model should distinguish:

```text
Expected differences
vs.
Unexpected differences
```

---

## 18. Testing Architecture

Testing is a continuous engineering activity from Week 1 through Week 12.

It is not a final project phase.

Every feature should follow:

```text
Design
  ↓
Define behavior/tests
  ↓
Implementation
  ↓
Unit tests
  ↓
Integration tests where needed
  ↓
Documentation
  ↓
CI verification
```

### Unit tests

Cover:

- domain rules,
- policy evaluation,
- readiness aggregation,
- deployment state transitions,
- diff generation,
- configuration validation,
- rollback rules.

These should be fast and should not require Docker or network access.

### Application/service tests

Use fakes or mocks for:

- Git provider,
- CI provider,
- registry provider,
- deployment target,
- traffic router,
- repository.

These tests verify orchestration.

### Adapter tests

Test concrete infrastructure behavior separately.

Examples:

- Git adapter
- GitHub adapter
- registry adapter
- Docker adapter
- proxy/router adapter
- PostgreSQL adapter

### Integration tests

Use `testcontainers-go` where real infrastructure behavior matters.

Primary candidates:

- PostgreSQL,
- disposable Docker containers,
- complete deployment topology where practical.

Testcontainers should not replace unit tests. It should validate the boundary where mocks cannot give sufficient confidence.

### GitHub integration

The normal CI test suite should not depend on a personal GitHub token.

Use:

- fake providers for service tests,
- recorded fixtures or contract-style tests where practical,
- explicitly configured integration tests for real GitHub API behavior.

A full GitHub App is not required for the core v1 workflow.

### End-to-end tests

At least one complete scenario should exist:

```text
Git commit
  ↓
readiness checks
  ↓
Docker deployment
  ↓
proxy routing
  ↓
health verification
  ↓
audit persistence
  ↓
rollback
```

### Failure-path testing

The project should deliberately test:

- failed CI,
- missing image,
- invalid configuration,
- Docker failure,
- proxy failure,
- health failure,
- timeout,
- deployment interruption,
- persistence failure,
- failed rollback.

---

## 19. CI Testing Strategy

GitHub Actions should continuously run:

```text
Formatting
   ↓
Static analysis
   ↓
Unit tests
   ↓
Integration tests
   ↓
Build
```

As infrastructure integration grows, the CI environment should be able to start the required disposable services/containers automatically.

No manual machine configuration should be required for the default CI suite.

---

## 20. Documentation as a Continuous Activity

Documentation must evolve alongside implementation.

Do not postpone documentation to Week 12.

### `README.md`

Update whenever user-visible behavior changes.

### `docs/architecture.md`

Update when architecture or important boundaries change.

### `docs/vision.md`

Update when product direction or scope changes.

### ADRs

Create an ADR when an important architectural decision is made.

### Development documentation

Update:

- setup,
- testing,
- contribution,
- local environment requirements.

### Operations documentation

Update:

- deployment configuration,
- proxy setup,
- troubleshooting,
- rollback,
- failure recovery.

Rule:

> If implementation changes the answer to "how does this system work?", update the relevant documentation in the same development cycle.

---

## 21. Development Workflow

Each implementation increment should follow:

```text
1. Define problem
2. Update scope/design if necessary
3. Identify domain behavior
4. Design interfaces
5. Define tests
6. Implement
7. Run unit tests
8. Run relevant integration tests
9. Update documentation
10. Update ADR if architecture changed
11. Run full CI suite
12. Commit
```

This is intended to reproduce the strongest development practices used in DiffSage.

---

## 22. Architectural Decision Records

Likely ADRs include:

```text
ADR-001 Project Architecture
ADR-002 CLI Architecture
ADR-003 Deployment Domain Model
ADR-004 External Provider Interfaces
ADR-005 Docker as Initial Deployment Target
ADR-006 PostgreSQL for Deployment Provenance
ADR-007 Immutable Image Digests
ADR-008 Health-Gated Deployment
ADR-009 Reverse Proxy Traffic Switching
ADR-010 Deployment Idempotency and Recovery
ADR-011 Readiness Decision Model
ADR-012 Typed YAML Policy Model
ADR-013 AI as Explanation Layer
ADR-014 Kubernetes Deferred to Future Adapter
```

These are candidates, not a checklist.

Create an ADR when the decision is made and its tradeoffs matter.

---

## 23. Extension Strategy

The architecture should make future expansion additive.

### New deployment target

```text
DeploymentTarget
├── DockerDeploymentTarget
└── KubernetesDeploymentTarget
```

### New traffic router

```text
TrafficRouter
├── TraefikRouter
└── FutureRouter
```

### New CI provider

```text
CIProvider
├── GitHubActionsProvider
└── GitLabCIProvider
```

### New registry

```text
RegistryProvider
├── GHCRProvider
└── ECRProvider
```

### AI

An explanation provider can consume existing readiness results.

### GitOps

Future GitOps functionality can consume the existing deployment/provenance model rather than creating a second deployment model.

---

## 24. Architectural Quality Bar

The project should be considered well-founded when:

- domain logic can be tested without Docker;
- application services can be tested without GitHub;
- provider adapters can be replaced;
- traffic routing is isolated from deployment orchestration;
- deployment history has a stable domain representation;
- deployment failures have explicit states;
- rollback is modeled as a real operation;
- CLI commands remain thin;
- configuration is validated at boundaries;
- tests cover both success and failure paths;
- integration tests exercise important real infrastructure boundaries;
- documentation describes the current architecture;
- architectural decisions are recorded when they matter;
- the default CI suite is reproducible.

The objective is not to create a perfect architecture upfront.

The objective is to create an architecture that can evolve safely.

---

## 25. Three-Month Architectural Milestones

### Calendar mapping

The week numbers below are effort-based, not calendar weeks. They map onto real availability as follows:

```text
Now → mid-November (partial time)
   Week 1 only — spikes, Go fundamentals, domain design,
   ADR-001 through ADR-004 drafted incrementally.

Mid-November → late December (full commitment)
   Weeks 2–11 — this is where the bulk of implementation
   happens, compressed into ~6 weeks of full focus rather
   than spread thin across the calendar.

After that
   Week 12 — hardening and release, time permitting before
   applications are due.
```

If full commitment weeks run short, the honest fallback is to treat Weeks 2–6 (readiness through Docker deployment) as the interview-ready checkpoint, with provenance, rollback, and hardening documented as in-progress via their ADRs rather than rushed to completion.

### Week 1 — Foundation + Technical Spikes

- repository
- Go module
- CLI skeleton
- configuration
- logging
- error strategy
- test infrastructure
- CI
- initial documentation
- reverse proxy spike
- Docker deployment topology spike
- Testcontainers proof of concept

Deliverable:

A validated technical direction before the core deployment engine is built.

### Weeks 2–3 — Readiness

- domain models
- provider interfaces
- Git adapter
- CI adapter
- registry adapter
- readiness service
- policy model

Tests and documentation are updated alongside each feature.

### Week 4 — Deployment Diff and Policy

- deployment diff
- typed YAML policies
- deterministic policy evaluation
- CLI output
- tests
- documentation

### Weeks 5–6 — Docker Deployment

- deployment state machine
- Docker adapter
- proxy/router adapter
- blue/green-style health-gated deployment
- traffic switch
- failure handling
- integration tests
- end-to-end deployment tests
- operational documentation

### Weeks 7–8 — Provenance

- PostgreSQL schema
- repository interface
- persistence adapter
- history
- deployment inspection
- Testcontainers integration tests
- documentation

### Week 9 — Rollback and Recovery

- rollback model
- rollback service
- state reconstruction
- health verification
- interrupted-deployment handling
- idempotency checks
- failure tests
- documentation

### Weeks 10–11 — Hardening

This is not the beginning of testing.

Instead, these weeks focus on expanding confidence:

- failure-path coverage
- integration/E2E coverage
- proxy failure scenarios
- Docker failure scenarios
- persistence failure scenarios
- CLI quality
- error handling
- configuration validation
- documentation review
- architecture review
- ADR review

### Week 12 — Release

No major new feature.

- final test suite
- end-to-end demo
- README refinement
- architecture documentation
- operations documentation
- example configuration
- release notes
- v1.0.0

---

## 26. Definition of Foundation Complete

The foundation is complete when future work can be added through existing boundaries.

For example:

```text
                    Core Engine
                         │
        ┌────────────────┼─────────────────┐
        ▼                ▼                 ▼
   Readiness         Deployment         Provenance
        │                │                 │
   ┌────┴────┐      ┌────┴────┐       PostgreSQL
   ▼         ▼      ▼         ▼
 GitHub    Future  Docker   Kubernetes
 Actions   CI      Target     future
```

Future capabilities should primarily be additions to adapters, policies, application services, or presentation layers—not rewrites of the domain.

---

## 27. Definition of Done for Any Feature

A feature is not considered complete merely because its code works.

A feature is complete when the relevant parts of:

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

are present.

The exact test level depends on the feature.

The exact documentation depends on whether it changes architecture, configuration, operations, or user behavior.

This standard applies throughout the project, from Week 1 to v1.0.

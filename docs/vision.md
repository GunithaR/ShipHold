# ShipHold — Vision

## 1. Vision Statement

> Make software deployment a verifiable, auditable, and reversible state transition.

ShipHold exists to give small and medium engineering teams stronger deployment discipline without requiring them to adopt a full Kubernetes/GitOps platform.

It sits between existing development and deployment tools and answers three fundamental questions:

1. **Before deployment:** Is this deployment ready?
2. **During/after deployment:** Did the intended state actually become healthy?
3. **Afterward:** What exactly happened, and can we safely return to a known-good state?

---

## 2. The Problem

Modern software teams have mature tools for individual parts of the delivery lifecycle:

```text
Git
CI/CD
Docker
Container registries
Cloud infrastructure
Monitoring
```

But small teams often lack a coherent safety layer connecting those systems.

A deployment can therefore become a manual sequence:

```text
Check Git
   ↓
Check CI
   ↓
Check image
   ↓
Check configuration
   ↓
Run deployment
   ↓
Hope health checks pass
   ↓
Try to remember what was deployed
   ↓
Manually figure out how to roll back
```

The tools exist.

The missing piece is often **deployment state and coordination**.

---

## 3. Core Insight

A deployment should not be treated as:

> "Run a command that replaces the current application."

It should be treated as:

> "Transition the environment from one known state to another known state under explicit safety conditions."

Therefore:

```text
Current State
      │
      │
      ▼
Proposed State
      │
      ▼
Evidence
      │
      ▼
Policy Evaluation
      │
      ▼
Deployment Decision
      │
      ▼
Controlled Transition
      │
      ▼
Health Verification
      │
      ▼
Recorded State
```

---

## 4. Product Philosophy

### 4.1 Evidence before decisions

The system should collect evidence before making deployment decisions.

Examples:

- Git commit
- CI result
- image digest
- configuration
- health-check configuration
- dependency reachability

The system should show why it reached a decision.

### 4.2 Deterministic safety

Deployment gates should be deterministic.

Given the same:

```text
evidence
+
policy
```

the system should produce the same:

```text
PASS
WARN
BLOCK
```

AI should not silently change this behavior.

### 4.3 Provenance by default

Every successful deployment should answer:

> What exactly was deployed?

The system should preserve:

- source commit
- artifact
- immutable image identity
- configuration version
- checks
- decision
- deployment events
- health verification
- result

### 4.4 Reversibility

A deployment should not only move forward.

A known-good deployment should remain recoverable.

Rollback should restore recorded deployment state rather than guessing which image tag was previously used.

### 4.5 Explainability

The system should explain deployment decisions using the evidence that produced them.

For example:

```text
BLOCK

CI has not passed for commit 8f91a22.

The image exists and configuration checks passed,
but the required CI policy is unsatisfied.

No deployment was attempted.
```

---

## 5. Target Users

The initial target is:

### Small and medium engineering teams

Especially teams using:

```text
GitHub
GitHub Actions
Docker
Docker Compose / single-server deployments
Container registries
PostgreSQL
```

These teams may not need or want a full Kubernetes/GitOps platform, but they still benefit from:

- deployment gates,
- provenance,
- health verification,
- rollback,
- policy enforcement.

---

## 6. Positioning

The project should be positioned as:

> **A Git-centric deployment safety and provenance engine for teams that want stronger deployment discipline without adopting a full Kubernetes platform.**

It is not:

- another CI system,
- another container orchestrator,
- a reverse proxy,
- an incident-response system,
- a monitoring platform,
- an AI root-cause-analysis tool.

A reverse proxy such as Traefik is an infrastructure dependency used to enable safe traffic switching; the project does not compete with or implement the proxy.

---

## 7. Relationship to GitOps

The initial product is not a full GitOps controller.

However, it shares an important GitOps principle:

> Desired application state should be explicit, traceable, and reproducible.

The project can be thought of as providing some GitOps-style discipline to simpler Docker environments.

Future versions could evolve from:

```text
Git
 │
 ▼
ShipHold
 │
 ▼
Docker
```

toward:

```text
Git
 │
 ▼
ShipHold
 │
 ├── Docker
 │
 └── Kubernetes
```

The project should not require Kubernetes to prove its core value.

---

## 8. Core User Experience

A developer should be able to think:

> "I want to deploy commit 8f91a22."

The system should turn that into:

```text
1. What is changing?
2. Is the deployment ready?
3. Are policies satisfied?
4. What evidence supports the decision?
5. Start the new version safely.
6. Verify the new version before switching traffic.
7. Record exactly what happened.
8. Restore the previous state if necessary.
```

The CLI should expose this clearly.

Example:

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

## 9. Deployment Safety Model

The initial deployment model uses a lightweight reverse proxy.

The public endpoint belongs to the proxy:

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

This avoids host-port collisions when the new version is started alongside the old version.

The sequence is:

```text
Start candidate
     ↓
Health check candidate
     ↓
Switch proxy routing
     ↓
Verify routed traffic
     ↓
Stop previous version
     ↓
Persist deployment
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

The project does not build its own reverse proxy.

---

## 10. Why the Reverse Proxy Is in Scope

A reverse proxy is not a new product feature.

It is deployment infrastructure required to make the initial safety model technically sound.

Without it, simultaneous old/new containers can cause host-port collisions.

With it, the system can demonstrate a real blue/green-style transition:

```text
Old version
     │
     ├── still serving traffic
     │
New version starts
     │
Health verification
     │
Traffic switches
     │
Old version stops
```

This makes the deployment engine more realistic without requiring Kubernetes.

Traefik is the preferred first implementation because it integrates naturally with Docker. The proxy itself remains outside the core domain.

---

## 11. Policy Vision

The policy system should initially remain deliberately simple.

Example:

```yaml
policy:
  require_ci_pass: true
  require_image: true
  require_immutable_digest: true
  require_healthcheck: true
```

Policies are parsed into typed Go configuration and evaluated by deterministic rules.

The project does not initially attempt to build a general-purpose policy language.

The goal is to solve deployment safety, not to create another programming language.

---

## 12. Testing Philosophy

Testing is part of the product, not a final phase.

The project should continuously maintain confidence through:

```text
Unit tests
    +
Service tests
    +
Adapter tests
    +
Integration tests
    +
End-to-end tests
```

Mocks/fakes are appropriate for fast domain and application tests.

Real infrastructure should be used where mocks cannot validate meaningful behavior.

`testcontainers-go` will be used for disposable infrastructure integration where appropriate, especially PostgreSQL and container-based scenarios.

The normal CI suite should be reproducible without personal credentials or manually configured infrastructure.

---

## 13. Documentation Philosophy

Documentation evolves with the system.

The repository should continuously maintain:

- README
- architecture documentation
- vision
- ADRs
- development setup
- configuration reference
- operational/troubleshooting documentation
- deployment and rollback procedures

A feature is not complete simply because the implementation works.

The feature's relevant:

```text
Behavior
+
Tests
+
Documentation
+
CI verification
```

must remain aligned.

---

## 14. What Makes the Product Valuable

### Without the system

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

### With the system

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

The system becomes the coordination layer.

---

## 15. Initial Product Scope

The initial three-month product should include:

- Go CLI
- Git integration
- GitHub Actions integration
- container registry verification
- deployment readiness checks
- deployment diff
- typed YAML deployment policies
- deterministic policy evaluation
- Docker-based single-host deployment
- reverse-proxy-based traffic switching
- health-gated replacement
- PostgreSQL deployment history
- deployment provenance
- deployment inspection
- deterministic rollback
- deployment failure/recovery handling
- continuous automated testing
- continuously maintained documentation
- CI

---

## 16. Explicitly Deferred

### Kubernetes

Deferred because Kubernetes infrastructure would consume substantial development time without being necessary to validate the core product.

### ArgoCD / Flux

Deferred because the product is not initially a GitOps controller.

### Terraform

Deferred because infrastructure provisioning is a different product boundary.

### Multi-cloud

Deferred to avoid provider sprawl.

### Multiple CI providers

GitHub Actions is sufficient to validate the abstraction.

### Multiple registries

One registry is sufficient initially.

### Full monitoring

The system only needs enough health verification to determine deployment success.

### Incident RCA

Explicitly outside the product boundary.

### AI deployment decisions

Never part of the core safety model.

### Custom policy DSL

Explicitly deferred. Typed configuration plus deterministic Go rules is sufficient for v1.

### Custom reverse proxy

Explicitly deferred. Use an established open-source proxy instead.

### Full GitHub App functionality

Not required for the core v1 deployment-safety workflow. Introduce it only if repository installation, webhook-driven workflows, PR checks, or similar capabilities become part of the product.

---

## 17. AI Vision

AI is a future explanation layer, not the source of truth.

Potential future experience:

```bash
shiphold explain
```

The engine provides structured evidence:

```text
CI = PASS
Image = PASS
Configuration = WARN
Health dependency = PASS
Policy = WARN
```

AI can turn that into:

```text
The deployment is allowed with a warning because
configuration differs from the expected production
profile. CI, artifact, and dependency checks passed.

No blocking condition was detected.
```

The AI must not invent checks or override deterministic policy.

---

## 18. Long-Term Vision

The long-term architecture can grow from:

```text
                         Git
                          │
                          ▼
                 Deployment Safety
                       Engine
                          │
             ┌────────────┼────────────┐
             ▼            ▼            ▼
          Docker      Kubernetes    Other
                                      Targets
```

Additional capabilities could include:

- Kubernetes deployment adapter
- GitOps reconciliation
- additional CI providers
- additional registries
- environment comparison
- deployment analytics
- GitHub PR checks
- notifications
- AI explanations
- policy packs
- compliance reporting

These are extensions, not prerequisites for the core product.

---

## 18.5. Realistic Availability Model

The three-month product goal assumes a specific, uneven availability pattern rather than uniform full-time effort:

```text
Now → mid-November
   Partial time, alongside coursework and other commitments.
   Focus: Go fundamentals, technical spikes, early ADRs,
   domain design. Not full feature implementation.
       │
       ▼
Mid-November → late December
   Full commitment (holiday period).
   Focus: core implementation — readiness, policy,
   Docker deployment, provenance, rollback.
       │
       ▼
Remainder before applications
   Hardening, documentation polish, demo recording.
```

This is not a change to the product scope. It is an honest mapping of the same three-month goal onto real availability, so the plan stays credible rather than aspirational. If the full v1.0 milestone slips past the application deadline, the earlier phases (readiness gate, policy evaluation, provenance model) should still stand as a coherent, demonstrable subset — see Section 21's guiding principle: build the smallest reliable system with correct boundaries first.

---

## 19. Three-Month Product Goal

The goal is not to complete the entire long-term vision.

The goal is to establish a strong v1 foundation.

At the end of three months, the project should demonstrate:

```text
Developer
   │
   ▼
shiphold check
   │
   ▼
Evidence + Policy
   │
   ▼
PASS / WARN / BLOCK
   │
   ▼
shiphold deploy
   │
   ▼
Start candidate container
   │
   ▼
Health verification
   │
   ▼
Proxy traffic switch
   │
   ▼
Verify active state
   │
   ▼
Immutable deployment record
   │
   ▼
Known-good state
```

And:

```bash
shiphold rollback <deployment>
```

should be able to restore a previous recorded state.

---

## 20. Success Criteria

### Technical

- the core engine is written in Go;
- deployment decisions are deterministic;
- external systems are isolated behind interfaces;
- reverse-proxy routing is isolated from deployment orchestration;
- Docker deployment is health-gated;
- deployment state is persisted;
- rollback is modeled explicitly;
- important failure cases are tested;
- real infrastructure boundaries are covered by integration tests;
- CI continuously runs the test suite.

### Engineering

- architecture is documented;
- important decisions have ADRs;
- configuration is documented;
- tests are maintained throughout development;
- operational procedures are documented;
- the repository is understandable to another developer;
- no major architectural decision is hidden only in implementation code.

### Portfolio

A recruiter should be able to understand within minutes:

1. What problem it solves.
2. Why the problem matters.
3. How the architecture works.
4. Why Go was selected.
5. Why a reverse proxy is used.
6. How deployment safety is enforced.
7. How rollback works.
8. How the system is tested.
9. How the architecture can expand.

The project should demonstrate engineering depth rather than feature count.

---

## 21. Guiding Principle for Future Development

> **Build the smallest reliable system that establishes the correct architectural boundaries, then extend it without breaking those boundaries.**

A feature is not complete merely because its code works.

For each capability:

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

should remain aligned.

The three-month target is a strong foundation, not an artificially complete DevOps platform.

---

## 22. Final Vision

The ShipHold should eventually make deployment feel less like:

> "I hope nothing goes wrong."

and more like:

> "I know what is changing, I know why the deployment is allowed, I know the new version became healthy before traffic moved, I know exactly what was deployed, and I know how to return to the previous known-good state."

That is the product.

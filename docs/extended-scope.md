# ShipHold — Extended Scope

**Status:** none of this is in v1. Nothing here may borrow time from `docs/roadmap.md`.

This document exists so that good ideas can be written down and then left alone. Each entry states what it is, why it is worth doing eventually, what it depends on, and roughly what it would cost.

A documented post-v1 roadmap is an asset only if v1 actually ships. If Week 8 is tight, **delete entries from this file rather than pulling them forward.**

---

## 1. Provenance and trust

### 1.1 Signed provenance records

**What.** Each provenance record's `record_hash` is signed with a deployment key; `shiphold verify` checks signatures as well as chain integrity.

**Why.** The v1 hash chain (ADR-009) makes tampering *detectable by anyone holding a later hash*. Signing makes records *attributable* — it distinguishes "this record was not altered" from "this record was written by an authorised deployer." That is the difference between an integrity property and an authenticity property, and for a compliance story it is the more interesting one.

**Depends on.** ADR-009 shipped; a key-management decision that ADR-012 deliberately does not make.

**Cost.** Perhaps a week, most of it spent on key handling rather than on cryptography. Ed25519 signing itself is a handful of lines with the standard library.

**Risk of doing it early.** Key management is where this becomes a real project rather than a feature. Resist.

---

### 1.2 External anchoring of the ledger head

**What.** Periodically publish the current chain head (a single hash) somewhere outside ShipHold's own database — a Git commit in the application repository, a transparency log, an object-store write with versioning enabled.

**Why.** A hash chain stored entirely in one PostgreSQL database can be rewritten wholesale by anyone who can rewrite that database: recompute every hash and the chain is internally consistent again. Anchoring the head externally removes that attack, because a rewritten chain will not match the published head.

**Depends on.** ADR-009.

**Cost.** Small — a day or two for the simplest version (commit the head hash to a file in the repo on each deploy). The design thinking is the expensive part, not the code.

**Note.** This is the honest limit of the v1 claim, and it is worth being able to articulate it: *v1 gives tamper-evidence against record-level edits, not against a full-database rewrite; anchoring closes that.* Being able to state your own system's limits precisely is usually more impressive than the feature would have been.

---

### 1.3 Attestation-format export

**What.** Export provenance records in a standard software-supply-chain attestation format rather than only ShipHold's own JSON — so ShipHold's output can be consumed by tooling that already understands build provenance.

**Why.** It reframes ShipHold from "a tool with its own audit log" to "a tool that participates in the supply-chain ecosystem," which is a meaningfully larger claim.

**⚠️ Verify before building.** The relevant specifications are the in-toto Attestation Framework and SLSA provenance. I know both exist and roughly what they do, but I am **not confident enough in the current predicate schemas, version numbers, or envelope details** to have you build against my description of them. Read the specifications directly at `in-toto.io` and `slsa.dev` and design from the primary sources. Treat this entry as "a direction worth investigating," not as a specification.

**Depends on.** ADR-009; a stable provenance record shape.

**Cost.** Unknown until the specs are read. Plausibly one to two weeks including the mapping work.

---

### 1.4 Image signature verification as a policy rule

**What.** A `signed_image` policy rule that verifies a container image's signature (Cosign/Sigstore being the obvious ecosystem) before permitting deployment.

**Why.** It is the natural next policy rule after `immutable_digest`, and it closes the gap between "we deployed exactly this digest" and "this digest came from a build we trust."

**Depends on.** The policy rule framework from ADR-006 being genuinely extensible — which is a good way to *test* whether it is. Also on the registry adapter growing beyond digest resolution.

**Cost.** A week or so, largely spent on the verification toolchain rather than on ShipHold itself.

---

## 2. Deployment capability

### 2.1 Kubernetes deployment target

**What.** A second `DeploymentTarget` implementation.

**Why.** It is the strongest possible proof that the abstraction is real, because Kubernetes' deployment model is so different from single-host Docker's. If `DeploymentTarget` survives it without changing shape, the boundary was correctly drawn.

**Depends on.** v1 shipped with the Docker target and a stable interface.

**Cost.** Substantial — several weeks. Kubernetes already does health-gated rollout natively, so ShipHold's value there shifts from *performing* the deployment to *gating and recording* it. That shift is itself an architecture decision worth an ADR.

**Honest caution.** This is the item most likely to be started for CV reasons and abandoned half-finished. A half-finished Kubernetes adapter is worse than none, because it turns "deliberately scoped to Docker" into "tried Kubernetes and could not."

---

### 2.2 Canary deployment with traffic weights

**What.** Rather than a binary blue/green switch, shift traffic progressively — 5%, 25%, 50%, 100% — with health and error-rate gates between steps, and automatic revert on regression.

**Why.** If ADR-008 resolves in favour of Traefik's weighted-service approach, the mechanism is *already there*; v1 simply uses the weights 0 and 100. Canary becomes a scheduling and gating problem rather than a routing one.

**Depends on.** ADR-008 confirmed with weighted services; some source of error-rate signal, which ShipHold does not currently have and which brushes against the "not a monitoring platform" non-goal.

**Cost.** Two weeks, plus a genuine scope conversation about where the error-rate signal comes from without ShipHold becoming an observability tool.

---

### 2.3 Multi-host deployment

**What.** Coordinate a deployment across several hosts rather than one.

**Why.** It is the most common reason a real team would outgrow v1.

**Depends on.** Distributed locking that the single-host advisory lock in ADR-011 does not provide; partial-failure semantics (three hosts succeed, one fails — what is the deployment's status?).

**Cost.** High, and it changes the domain model rather than extending it. This is the point at which the honest answer becomes "use Kubernetes."

---

## 3. Integration and interface

### 3.1 GitHub App with pull-request checks

**What.** ShipHold as an installable GitHub App that posts a readiness check on pull requests — showing the PASS/WARN/BLOCK verdict before merge rather than at deploy time.

**Why.** It moves the gate left, which is where gates are most useful, and it is very demonstrable: a screenshot of a ShipHold check on a real PR communicates the product in one image.

**Depends on.** Webhook handling, app installation flow, and a hosted component — all of which turn a CLI into a service. `architecture.md` §3 explicitly holds this back for that reason.

**Cost.** Two to three weeks, most of it infrastructure rather than domain logic.

---

### 3.2 `shiphold diff` as a standalone command

**What.** The command cut from v1, restored: a full comparison between the currently deployed state and the candidate state — image digest, configuration, environment variables, policy version — with a structured diff output.

**Why.** "What exactly changes if I run this?" is a good question and the answer is currently folded into `check` output in abbreviated form.

**Depends on.** A decision about what is authoritative for "currently deployed state" — the provenance record, or the observed Docker/proxy state, and what it means when they disagree. That disagreement is genuinely interesting and is the real content of the feature.

**Cost.** A week, once the state-of-truth question is settled.

---

### 3.3 `shiphold policy test`

**What.** Evaluate a policy file against fixture evidence without touching any real system — `shiphold policy test --policy prod.yaml --evidence fixtures/failing-ci.json`.

**Why.** Policies are code. Code that cannot be tested in isolation does not get changed confidently. This is also the cheapest item in this entire document and arguably belongs in v1 if any week comes in under budget.

**Depends on.** Nothing that is not already in v1 — the evaluator is pure, and the evidence type is serialisable.

**Cost.** One to two days.

**Recommendation.** If you find slack anywhere in Weeks 2–8, spend it here rather than on anything else in this file.

---

### 3.4 Additional CI providers and registries

**What.** GitLab CI, CircleCI; ECR, Docker Hub, Artifact Registry.

**Why.** The second implementation is what proves `CIProvider` and `RegistryProvider` are abstractions rather than single-vendor APIs with generic names — the DiffSage lesson applied to the seams where v1 could not afford it.

**Depends on.** v1's interfaces being stable.

**Cost.** Small per adapter if the interfaces are right, and large if they are not — which is precisely the information the exercise produces.

---

## 4. Explanation layer

### 4.1 AI-narrated readiness and incident summaries

**What.** An optional `--explain` flag producing a natural-language summary of a decision, drawing only on the structured `RuleResult` set and the provenance record.

**Why.** It is the DiffSage pattern, applied where it belongs — and ShipHold's version is architecturally stronger, because the deterministic result exists independently and remains fully usable with the feature disabled or unavailable.

**Depends on.** ADR-015's boundary being honoured absolutely: the explanation layer reads the decision and never influences it. The moment an AI output can change a PASS to a BLOCK, the entire trust story collapses.

**Cost.** A week.

**Honest caution.** This is a small feature with a large risk of being misread. If it appears in the README above the deterministic engine, a reader will file ShipHold as "another AI DevOps tool," which is the exact positioning `vision.md` §6 spends a page avoiding. If it ships at all, it ships as a footnote.

---

## 5. Two things deliberately not on this list

**A web UI.** Deployment history is more legible in a browser, and building one would consume more time than every other item here while demonstrating nothing about the architecture that the CLI does not already demonstrate.

**A hosted SaaS version.** It changes the product category, the threat model, and the licence conversation all at once. If ShipHold ever warrants it, that starts with a new document, not a line in this one.

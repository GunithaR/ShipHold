# ADR-015: AI strictly as an explanation layer

## Status

Accepted

## Context

DiffSage establishes the pattern this project inherits: compute facts deterministically from evidence first, and only then let a language model narrate them. The model is never the source of truth.

ShipHold needs a stronger version of the same rule, for a reason specific to what it is. DiffSage's AI narrated a pull-request risk analysis — if the narration was wrong, a human read a misleading paragraph about a diff. ShipHold gates deployments and produces an audit trail. If a model influenced a decision, then the deployment record cannot be independently verified, because re-running the evaluation against the recorded evidence would not reliably reproduce the recorded outcome. That breaks the reproducibility property ADR-003 and ADR-009 are built on.

So the rule is not merely "AI should not decide." It is that **the trust chain must contain no non-deterministic component at all.**

`vision.md` §17 and `architecture.md` §4.2 both already state a version of this. It is written down here because it is the decision most likely to be eroded by a reasonable-sounding incremental suggestion, and a boundary defended by a stated principle holds better than one defended by a preference.

## Options Considered

**AI as a policy rule.** A model assesses whether a change looks risky and contributes to the verdict. Rejected — it makes the decision non-reproducible, unexplainable in the specific sense that matters (you cannot show *which rule* blocked), and unauditable. It also makes the gate's behaviour a function of a vendor's model version, which is not a property a safety tool should have.

**AI as a tie-breaker on `WARN`.** Superficially modest and materially the same thing: the model's output changes the outcome, so it is in the trust chain.

**AI summarises the deployment history.** Reads recorded provenance and produces prose. The records are unaffected, the summary is derived, and nothing gated depends on it. Acceptable.

**AI narrates a completed decision.** The deterministic engine produces the verdict; the model explains it in prose for a human reader. Acceptable.

**No AI at all.** Also entirely acceptable, and where v1 lands.

## Decision

**AI may only consume outputs of the deterministic system. It may never produce an input to it.**

```text
Evidence  →  Policy evaluation  →  Decision  →  Provenance record
                                       │              │
                                       └──────┬───────┘
                                              ▼
                                    [ optional explanation ]
                                              │
                                              ▼
                                     prose, for a human

  Everything above the dashed boundary is deterministic and reproducible.
  Nothing below it can affect anything above it.
```

Four constraints, all testable:

1. **No model output ever becomes evidence, a rule result, a decision, or any field of a provenance record.** Enforced structurally: the explanation service has read-only access to a completed `EvaluationResult` and returns a `string`. It has no access to the repository's write path, and no type reachable from `Record` has a field it can populate.

2. **Every feature works identically with AI disabled, unavailable, or failing.** An explanation that times out produces no explanation and changes nothing else. There is no code path where a model call failing alters an outcome. This is the constraint that makes the others verifiable — you can test it by running the whole suite with the feature off and asserting identical results.

3. **AI is off by default and opt-in per invocation** (`--explain`). A user who never enables it has a tool with no non-deterministic component anywhere in it.

4. **Explanations are never persisted into the ledger.** Provenance records contain evidence, rule results, and events. An explanation is a rendering of a record, regenerable at any time from the record itself, and generating it twice may produce different prose — which is fine for prose and disqualifying for an audit trail.

### Positioning, which is part of this decision

`vision.md` §6 spends a page establishing that ShipHold is not "an AI system making deployment decisions." That positioning is a real asset in a market where a great many tools lead with AI, and it is fragile in one specific way: if `--explain` appears prominently in the README, a reader will file the project under "AI DevOps tool" and stop reading before reaching the deterministic engine.

If an explanation feature ever ships, it appears in the documentation *after* the safety model, described as what it is — an optional convenience over records that stand on their own. It is never the headline, and it never appears in a demo before the deterministic verdict has been shown first.

### v1 contains no AI

No dependency, no interface, no flag, no configuration. `extended-scope.md` §4.1 describes what a future explanation layer would look like, at a week of effort, after v1 ships.

That is not a limitation to apologise for in an interview. "The decision engine is fully deterministic, every verdict is reproducible from the recorded evidence, and I deliberately kept the model out of the trust chain" is a stronger technical statement than any AI feature this project could ship in ten weeks.

## Consequences

**Makes easy.** Every decision is reproducible from its provenance record, forever. Tests are deterministic. There is no vendor dependency in the critical path, no API key required to gate a deployment, no cost per check, and no model-version drift changing what the gate does. The security and compliance story is clean: no evidence leaves the environment.

**Makes harder.** Explanations are limited to what deterministic rules can express — which, given that rule results carry structured evidence references (ADR-003), is a good deal more than it sounds. Some genuinely useful analysis ("this change pattern resembles last month's incident") is unavailable. That is the trade, and it is the right one for a gate.

**Rules out.** AI in evaluation, in evidence collection, in the decision, or in any persisted record. Any feature that does not work with AI disabled. Any explanation stored in the ledger.

**How this decision gets eroded, and how to notice.** The realistic failure is not someone proposing "let the model decide." It is a sequence of small steps: an explanation feature, then caching the explanation in the record "for convenience," then a model-generated risk annotation "for information only," then a policy rule that reads the annotation. Each step is reasonable and the fourth is disqualifying.

The test that catches it: **can this decision be recomputed from the recorded evidence, by anyone, with no network access, and produce exactly the recorded result?** If not, the boundary has moved. Run that question against any proposed change here.

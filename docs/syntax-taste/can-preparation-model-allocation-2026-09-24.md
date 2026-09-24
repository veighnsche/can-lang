# Models and reasoning effort for Can preparation

24 September 2026 · official documentation checked online on this date

This allocates models to the [preparation todo list](can-design-preparation-2026-09-24.md).
It does not execute the preparation, change the current task's model or change
account defaults. The assignments are engineering recommendations for this work,
not benchmark results proving an optimal configuration.

Use the [dependency and parallel launch plan](can-preparation-parallel-plan-2026-09-24.md)
to schedule these assignments. A stage lead integrates results; its existence
does not create a global wait between numbered stages.

## Models and supported settings

The current official GPT-6 family contains Astra, Sol and Luna. Astra targets the
most demanding work, Sol handles complex coding and general agent work, and Luna
suits focused, repeatable tasks. See the [official family guidance](https://developers.openai.com/api/docs/guides/latest-model).

| Model | Documented API reasoning efforts | Efforts exposed by this Codex host | Role in this plan |
| --- | --- | --- | --- |
| [GPT-6 Astra](https://developers.openai.com/api/docs/models/gpt-6-astra) · `gpt-6-astra` | `low`, `medium`, `high`, `xhigh`, `max` | Light, Medium, High, Extra High, Max, Ultra | Consequential design synthesis, technical disagreements and independent integration review |
| [GPT-6 Sol](https://developers.openai.com/api/docs/models/gpt-6-sol) · `gpt-6-sol` | `none`, `low`, `medium`, `high`, `xhigh`, `max` | Light, Medium, High, Extra High, Max, Ultra | Source audits, research, examples, experiments and detailed task planning |
| [GPT-6 Luna](https://developers.openai.com/api/docs/models/gpt-6-luna) · `gpt-6-luna` | `none`, `low`, `medium`, `high`, `xhigh`, `max` | Light, Medium, High, Extra High, Max | Bounded extraction, links, records and mechanical artifact handling |

The Codex column is corroborated by this session's task-dispatch tool schema;
availability can differ on another host. The [official Codex model guidance](https://learn.chatgpt.com/docs/models)
explains that Light maps to `low`, Extra High maps to `xhigh`, and Ultra uses
subagents. Luna does not support Ultra. Ultra is a Codex execution setting;
the GPT-6 API model pages do not list it as a `reasoning.effort` value.

| Effort | Treatment in this plan |
| --- | --- |
| None · `none` | API option for Sol/Luna; not exposed by this host for these tasks and not assigned |
| Minimal · `minimal` | Mentioned in general API schemas, but not listed as supported by these GPT-6 model pages; not assigned |
| Light · `low` | Record confirmed choices and handle simple artifact bookkeeping |
| Medium · `medium` | Bounded extraction, ordinary research, question preparation and focused review |
| High · `high` | Cross-file investigation, design trade-offs, experiment design and contract integration |
| Extra High · `xhigh` | Escalation only if a specific unresolved problem demonstrates benefit beyond High |
| Max · `max` | Reserve for an exceptionally difficult individual problem with a recorded reason; no default assignment |
| Ultra · `ultra` | Assign to the coordinated independent readiness audit in step 9, with the worker roles below |

These choices follow the [reasoning guidance](https://developers.openai.com/api/docs/guides/reasoning),
which describes the quality/latency trade-off and calls for evidence before using
Extra High routinely. `pro` is a separate API reasoning mode, not another effort
level or a fourth GPT-6 model; it is not selected by this Codex allocation.

## Stage leads

The lead integrates the stage's result. Each checkbox in the preparation list
also has an explicit worker assignment; the lead setting is not inherited by
every subtask. Labels Astra, Sol and Luna always mean the GPT-6 models above.

| Step | Preparation stage | Lead model | Effort | Reason |
| --- | --- | --- | --- | --- |
| 0 | Record the confirmed identity | GPT-6 Luna | Light | The user already made the decision; only recording is required. Complete—do not rerun. |
| 1 | Build the decision inventory | GPT-6 Sol | Medium | Requires complete extraction, distinction between facts and proposals, and careful deduplication. |
| 2 | Agree scope and evaluation criteria | GPT-6 Astra | High | Defines consequential trade-offs and what evidence counts as agent benefit. |
| 3 | Research facts and constraints | GPT-6 Sol | High | Trace behavior across specifications, compiler, runtime and primary sources. |
| 4 | Brainstorm and compare approaches | GPT-6 Astra | High | Open-ended language alternatives and interactions need strong synthesis. |
| 5 | Resolve technical questions with Jev | GPT-6 Astra | High | Construct fair decision packets, interpret disagreement and justify engineering choices. Jev remains the advisory classifier. |
| 6 | Resolve syntax without further user questions | GPT-6 Astra | Medium | Preserve the two recorded user choices; use Jev and evidence for remaining technical syntax choices under the later no-questions instruction. |
| 7 | Run design experiments | GPT-6 Sol | High | Build and analyze bounded probes against established criteria. |
| 8 | Consolidate specifications and acceptance | GPT-6 Astra | High | Reconcile semantics, grammar, contracts and evidence across the whole design. |
| 9 | Independently audit readiness | GPT-6 Astra | Ultra | Coordinate separate contract and product-flow reviews, reconcile findings and verify corrections. |
| 10 | Produce the implementation task list | GPT-6 Sol | High | Decompose the settled design and construct its dependency graph, with Astra Medium reviewing completeness. |

This allocation applies the user's [model-selection policy](/Users/vince/.codex/skills/model-selection/SKILL.md)
to delegated work. It uses the policy's normal Luna Light/Medium, Sol
Light/Medium/High and Astra Medium/High combinations. Official documentation also
allows other combinations; support alone does not make them necessary here.
The official [model-selection guidance](https://developers.openai.com/api/docs/guides/model-selection)
treats recommendations as starting points to evaluate on the actual workload.

## Worker assignments and escalation

Each checkbox receives a stable preparation-task ID such as `P3.2`, plus its
model and effort. These IDs identify this preparation checklist, not the later
language-finding inventory or implementation tasks. The two completed identity
items receive a suitable recording configuration retrospectively; this does not
claim that Luna performed the earlier work.

Use the listed setting for the specified task slice. Luna's extraction task in
step 1 operates on bounded source sections using an agreed extraction format;
Sol owns reconciliation and the completeness check. Preserve source links and
uncertainty instead of asking Luna to silently decide disputed semantics.

Escalate when evidence reveals contradictory contracts, subtle security or
ownership consequences, or ambiguity beyond the assigned task. Move routine
Luna work to Sol Medium, cross-file Sol work to Sol High, and major unresolved
design questions to Astra High. Record the reason. Extra High or Max requires a
specific reason that the normal levels are inadequate; neither is a blanket
upgrade because a document is long.

For Jev preparation, Sol High drafts the three semantically equivalent request
wordings and Astra Medium checks their equivalence. Astra High owns the evidence
framing and disagreement analysis. Jev is neither replaced by GPT-6 nor assigned
a GPT-6 reasoning effort.

Routine recording of a user-confirmed syntax choice belongs to Luna Light under
the stage lead's check. The user's later instruction ended further syntax
questions; difficult remaining choices require three fresh Jev consultations,
evidence and a recorded engineering judgment.

## Why step 9 uses Ultra

This is a defined coordinated review procedure, not a synonym for “think longer”:

1. An independent Astra High reviewer handles `P9.1`: identity, contract
   consistency, assumptions and evidence quality.
2. Separate Sol High reviewers cover the server and browser parts of `P9.2`.
   Run them in available concurrency slots; do not exceed the host's limit.
3. The Astra Ultra lead combines the findings and routes `P9.3` to Astra High
   for resolution of material blockers, including return to earlier preparation
   steps where required.
4. Sol Medium can verify recorded-choice agreement in `P9.4` alongside the
   independent reviews. It revalidates affected records after corrections; the
   relevant independent reviewers recheck corrected findings before the lead
   closes the gate.

Reviewers receive requirements, selected specifications and raw evidence without
the authors' prior review verdicts. They do not review their own authored sections.
Separate contract and product-flow scrutiny can expose different omissions;
the coordinator must reconcile them and obtain verification of corrections.
That independent work and coordination justify Ultra here. Other steps retain
their assigned effort even when an occasional bounded subtask can run in parallel.

Recheck model availability and the relevant official pages when this plan is
executed if the model catalog or host capabilities have changed.

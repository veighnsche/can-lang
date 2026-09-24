# Can design preparation todo list

24 September 2026

**Destination:** a complete implementation task list in dependency order, with
the necessary product, syntax and technical decisions already resolved.

**Current status:** this preparation round and the subsequent
[27-task execution round](can-implementation-task-list-2026-09-24.md) have
recorded completion. The [readiness audit](preparation/readiness-audit.md)
was a pre-implementation design check. The
[post-upgrade reconciliation](post-upgrade-reconciliation-2026-09-24.md)
now distinguishes delivered behavior, unfinished accepted requirements and
new proposals. The completed checklist below is historical preparation;
its future-tense statements are not the current implementation status.

The [recommendation program](can-recommendation-program-2026-09-24.md) supplies
the initial findings and proposed outcomes. Its gate numbers are recommendation
milestones; they are not yet a proven implementation dependency order. Existing
decisions remain authoritative until explicitly revised. The user-confirmed
identity below governs the evaluation of those recommendations.

**Model assignments:** every checkbox below has a GPT-6 model and reasoning
effort. Astra, Sol and Luna denote `gpt-6-astra`, `gpt-6-sol` and
`gpt-6-luna`; Light is `low`. The [allocation rationale and official sources](can-preparation-model-allocation-2026-09-24.md)
explain all supported efforts, stage leads, worker roles and escalation.
These assignments do not start any pending task or change account settings.

**Parallel execution:** the [dependency and launch plan](can-preparation-parallel-plan-2026-09-24.md)
maps every item below to its prerequisites, ready-to-launch packets and required
cross-topic agreements. The numbered stages organize the work; they do not
require all topics to finish a stage before any topic advances.

## 0. Freeze Can's identity

**Stage lead:** GPT-6 Luna · Light.

- [x] **P0.1 · Luna · Light** — Record the user's confirmed identity in the authoritative
  [design direction](decisions.md#design-direction): Can is made for AI coding
  agents. Human readability, familiarity and comfort are not design goals.
  Human-hostile syntax is acceptable when it serves agents better.
- [x] **P0.2 · Luna · Light** — Initially keep token efficiency open for investigation;
  the user subsequently selected it as a secondary measured goal under P2.4.

**Completed output:** a fixed audience and design priority. No syntax or
technical mechanism has been selected by this identity confirmation.

## 1. Build the complete decision inventory

**Stage lead:** GPT-6 Sol · Medium.

- [x] **P1.1 · Luna · Medium** — Extract every finding, recommendation, alternative and open question from
  the recommendation program, its two source reviews and the current decisions.
- [x] **P1.2 · Sol · Medium** — Give each item a stable tracking ID and link its source evidence and any
  earlier disposition. Consolidate duplicates without dropping distinct concerns.
- [x] **P1.3 · Sol · High** — Separate observed behavior, required outcomes, proposed solutions and
  assumptions. Flag human-oriented justifications for reassessment against the
  frozen identity.
- [x] **P1.4 · Sol · Medium** — Include the topics marked “later”; each must eventually be accepted,
  retained, rejected or explicitly deferred with a reason and reopening condition.

**Output:** a coverage ledger. Every source finding has a destination; inclusion
in the ledger does not approve its proposed solution.

**Completed output:** [decision inventory](preparation/decision-inventory.md),
covering all 23 crosswalk rows with stable IDs, evidence, current authority and
separate deferred items.

## 2. Agree the target and evaluation criteria

**Stage lead:** GPT-6 Astra · High.

- [x] **P2.1 · Astra · Medium** — Confirm the intended application scope and recommendation milestones,
  including server-driven SaaS and a Can-authored rich frontend. Distinguish
  intermediate milestones from the final target.
- [x] **P2.2 · Sol · High** — Record existing constraints, including no compatibility obligation,
  native JavaScript/Bun lowering and controlled platform capabilities. Surface
  conflicts explicitly rather than silently carrying forward incompatible rules.
- [x] **P2.3 · Astra · High** — Define how to evaluate benefit to coding agents. Decide which evidence
  will measure generation, understanding, editing, diagnosis and repair.
- [x] **P2.4 · Astra · High** — Investigate token efficiency across source, required
  context, generated output, diagnostics and repair iterations. The user chose
  total tokens per successful AI coding task as a secondary measured goal. Do
  not equate fewer characters with fewer tokens; finalize the exact protocol
  and weighting under P2.5.
- [x] **P2.5 · Astra · High** — Fix representative workloads and success criteria before comparing
  alternatives. Identify model/tokenizer and environment assumptions for any
  agent or token measurements.

**Output:** agreed scope and an evaluation protocol with explicit open research
questions. Human familiarity alone cannot justify a design choice.

**Completed artifacts:** [constraints](preparation/constraints.md) and the
[evaluation protocol](preparation/evaluation-protocol.md). P2.5 fixes task families, success criteria, model settings, repeated attempts
and case-registration rules. Exact candidate prompts/hidden fixtures must be
registered under implementation T01 **before** an agent comparison; none is
claimed to have run during preparation.

## 3. Research the facts and constraints

**Stage lead:** GPT-6 Sol · High.

- [x] **P3.1 · Sol · High** — Audit the relevant current compiler, runtime, catalogue, examples and
  specifications. Reproduce or qualify existing evidence where needed.
- [x] **P3.2 · Sol · Medium** — Research relevant approaches and platform behavior using primary sources;
  record versions, dates, limitations and applicability to Can's agent audience.
- [x] **P3.3 · Sol · High** — Establish the best current Can idiom for each problem and identify actual
  failures or measured costs. Separate a language gap from a tooling, library,
  example or deployment gap.
- [x] **P3.4 · Sol · Medium** — Record unknowns and counterexamples. Jev receives the researched context;
  it is not a research source and cannot inspect the repository itself.

**Output:** evidence packets sufficient to compare alternatives without relying
on assumptions about current behavior.

**Current artifacts:** [core](preparation/core-evidence.md),
[packages/assertions](preparation/packages-assertions-evidence.md),
[server](preparation/server-evidence.md), [browser](preparation/browser-evidence.md),
[lifetime/deployment](preparation/lifetime-deployment-evidence.md), and
[AI/support](preparation/ai-support-evidence.md) source audits, plus
[platform research](preparation/platform-research.md) and a reproducible
[two-caller generic-helper baseline](preparation/generic-helper-probe.md).
These establish current idioms and gaps; candidate comparisons and product
qualification still belong to P4/P7.

## 4. Brainstorm and compare approaches

**Stage lead:** GPT-6 Astra · High.

- [x] **P4.1 · Astra · High** — Develop alternatives for each open outcome, including retaining the
  current design where it can satisfy the agreed criteria.
- [x] **P4.2 · Astra · High** — Compare semantics, agent behavior, composition, refactoring, diagnostics,
  native lowering, tooling and implementation cost. Apply token criteria only
  after their definition and status have been agreed.
- [x] **P4.3 · Astra · High** — Expose interactions between topics and draft a design dependency map.
  Investigate browser constraints early enough to inform shared contracts;
  qualification milestone order does not decide research order.
- [x] **P4.4 · Sol · High** — Write representative programs and counterexamples for discussion. Mark
  all unselected syntax as illustrative.

**Output:** concrete alternatives with evidence, consequences and unresolved
questions. A favored alternative is not yet an accepted decision.

**Completed artifacts:** [core alternatives](preparation/core-alternatives.md),
[product alternatives](preparation/product-alternatives.md),
[supporting alternatives](preparation/support-alternatives.md), and the
[provisional design dependency map](preparation/design-dependency-map.md).
P4.2 is a qualitative comparison of semantics and plausible agent costs; it
does not claim measured agent advantage. The topic packets and disposable probes now supply representative source forms,
negative cases and counterexamples. Held-out agent fixtures are registered before
their later trials, not invented after observing candidate outcomes.

## 5. Resolve technical questions with Jev's advice

**Stage lead:** GPT-6 Astra · High.

- [x] **P5.1 · Astra · High** — Prepare technical decision packets with all relevant facts, constraints,
  exact examples and viable alternatives, including evidence against the favored
  choice. Ask Jev about technical details, not to choose the user's preferences.
- [x] **P5.2 · Sol · High** — For each consultation, prepare three fresh requests. Rewrite all
  explanatory prose—context, instructions, questions and option descriptions—
  while preserving facts, alternatives and exact technical identifiers/code.
- [x] **P5.3 · Astra · Medium** — Check semantic equivalence and full-request wording differences before
  sending. Save all requests and responses.
- [x] **P5.4 · Astra · High** — Investigate disagreements and missing evidence. Record the engineering
  judgment separately from Jev's classifications; agreement is advice, not proof.
- [x] **P5.5 · Astra · High** — Resolve technical dependencies needed to present honest syntax options.
  Keep any remaining assumptions visible for the next steps.

**Output:** supported technical decisions or precisely scoped experiments needed
to resolve them. No compiler behavior is established by a consultation alone.

**Completed consultation record:** three fresh requests each for
[pattern intent](preparation/jev-pattern/findings.md),
[generic-helper trial](preparation/jev-generic/findings.md),
[core contracts](preparation/jev-core-contracts/findings.md),
[product contracts](preparation/jev-product-contracts/README.md), and four
[supporting decision sets](preparation/jev-support-contracts/findings.md), plus
follow-up rounds for [error/action placement](preparation/jev-remaining-syntax/findings.md),
[owner records](preparation/jev-owner-record/findings.md),
[finite error sets](preparation/jev-finite-error-followup/findings.md), and
[fixtures](preparation/jev-fixture-followup/findings.md).
All exact requests/responses and disagreement analyses are saved. This is an
completed consultation record; subsequent route, owner, fixture, generic,
package, row, browser and JSON-action rounds followed new evidence and saved
all three requests/responses per difficult decision. Jev remained advisory.

## 6. Resolve syntax choices without further user questions

**Stage lead:** GPT-6 Astra · High.

- [x] **P6.1 · Astra · Medium** — Prepare concrete syntax alternatives in complete examples, with their
  semantic consequences, companion forms, diagnostics and relevant agent evidence.
- [x] **P6.2 · Sol · Medium** — Bundle contexts that share one syntax rule into a focused decision packet. Include
  affected construction, matching, ignoring, forwarding or error forms where relevant.
- [x] **P6.3 · Astra · High** — Honor the user's later instruction to stop asking questions:
  consult Jev three fresh times on remaining difficult syntax choices, then
  select by evidence and engineering judgment. Preserve the earlier user
  selections without reopening them.
- [x] **P6.4 · Luna · Light** — Record each selected choice and its full scope. Track
  unresolved forms and dependencies explicitly.

**Output:** selected syntax with its full stated scope. Revisit technical
analysis when a syntax choice changes the available semantics or implementation.

**Current answers and direction:** the user selected `bind name` for
ordinary-data captures and dependency-qualified source `uses` entries, then
directed us to stop asking questions and consult Jev for the rest. The earlier
answers remain in [confirmed syntax choices](preparation/confirmed-syntax-choices.md).
Engineering has since selected unnumbered application `error`, source `action`
and `owner record` for the planned design, with full scope and evidence in
[accepted technical decisions](preparation/accepted-technical-decisions.md).
The selected forms and boundaries are in [accepted technical decisions](preparation/accepted-technical-decisions.md)
and the [integrated action contract](preparation/integrated-action-contract.md).
Deferred optional syntax is excluded from the implementation list.

## 7. Run the necessary design experiments

**Stage lead:** GPT-6 Sol · High.

- [x] **P7.1 · Astra · High** — Design bounded probes, disposable prototypes and agent comparisons for
  unresolved claims. State what each experiment must establish before running it.
- [x] **P7.2 · Sol · High** — Exercise representative programs, negative cases, controlled edits and
  composition boundaries. Measure against the agreed baseline and criteria.
- [x] **P7.3 · Astra · High** — Return to research or Jev consultation and engineering syntax resolution when results
  contradict a proposed choice. Update affected evidence and decisions together.
- [x] **P7.4 · Luna · Light** — Keep experimental artifacts identified as such. A successful prototype
  does not silently become the production implementation.

**Output:** enough evidence to close design questions. Steps 3–7 form a bounded
decision loop for each topic; speculative choices do not become implementation
tasks merely to postpone resolving them.

## 8. Consolidate the complete design and acceptance contracts

**Stage lead:** GPT-6 Astra · High.

- [x] **P8.1 · Astra · High** — Reconcile accepted choices into the consolidated planned specifications and
  dispositions, identifying superseded current rules and linking supporting evidence.
- [x] **P8.2 · Astra · High** — Specify observable semantics, grammar where changed, type/checking rules,
  error behavior, runtime/native mappings and affected package/wire boundaries.
- [x] **P8.3 · Sol · High** — Cover diagnostics, assertions, tooling, generated artifacts, documentation
  and operational consequences wherever the chosen design affects them.
- [x] **P8.4 · Sol · High** — Write acceptance cases for positive behavior, rejected programs,
  refactoring, integration and failure paths. Distinguish design proof already
  obtained from production qualification still required during implementation.
- [x] **P8.5 · Astra · Medium** — Give every inventory item an explicit disposition. Record deferred items
  and their consequences for the final recommendation scope.

**Completed output:** [dispositions](preparation/design-disposition-ledger.md),
[accepted technical choices](preparation/accepted-technical-decisions.md),
[integrated action contract](preparation/integrated-action-contract.md) and
[observable product acceptance](preparation/product-acceptance-matrix.md).
Current `decisions.md` remains the implementation baseline; this planned design
is the source of truth for the later implementation tasks until those documents
are revised together. No hidden syntax decision is delegated to coding.

## 9. Audit readiness to plan implementation

**Stage lead:** GPT-6 Astra · Ultra.

- [x] **P9.1 · Astra · High** — Independently review identity alignment, source-finding coverage,
  cross-feature consistency, evidence quality and unresolved assumptions.
- [x] **P9.2 · Sol · High** — Check the complete server and browser scenarios against the design,
  including package composition, security boundaries and operational failures.
- [x] **P9.3 · Astra · High** — Close any unresolved choice that could materially change implementation
  architecture, public contracts, syntax or task ordering. Return to the relevant
  preparation step rather than disguising that choice as an implementation task.
- [x] **P9.4 · Sol · Medium** — Confirm that recorded earlier user syntax choices and later Jev-advised technical decisions match
  the consolidated specification; do not reopen confirmed choices without cause.

**Completed exit review:** the [readiness audit](preparation/readiness-audit.md)
checks inventory coverage, cross-feature boundaries, security, browser/Linux
scenarios and dependency order. Remaining work implements and qualifies the
planned design; measured agent comparisons may return a failed candidate to
design without silently widening the implementation.

## 10. Produce the implementation task list

**Stage lead:** GPT-6 Sol · High.

- [x] **P10.1 · Sol · High** — Derive tasks from the accepted design and acceptance cases, with traceability
  back to the coverage ledger.
- [x] **P10.2 · Sol · High** — Build an explicit dependency graph and order tasks by actual prerequisites.
  Identify parallel work and milestone completion conditions.
- [x] **P10.3 · Sol · Medium** — Give each task its purpose, scope, prerequisites, affected components,
  required generated outputs, validation and definition of done.
- [x] **P10.4 · Sol · Medium** — Include compiler/runtime/library/tooling work, documentation, examples,
  regeneration, platform packaging and end-to-end qualification where required.
  Include the repository's runtime lint, formatting, checking and relevant test
  requirements for tasks that touch authored runtime TypeScript.
- [x] **P10.5 · Astra · Medium** — Audit the list for omissions, circular dependencies, duplicated work and
  unapproved design choices. Use the recommendation gates as outcome checkpoints.

**Final deliverable:** the [complete 27-task implementation list](can-implementation-task-list-2026-09-24.md)
in dependency order, including parallel-ready lanes and Gate 1–5 exit checks.
That list was subsequently executed; its completion evidence and the current
reconciliation must be read together when preparing further work.

## Coverage check for the future inventory

The inventory in step 1 must account for at least every topic in the program's
crosswalk. This is a source coverage check, not a selection of mechanisms:

- Pattern intent and typo diagnostics.
- Package identities, import aliases and diagnostic error identities.
- Variant identity, specialization and leaf compatibility.
- Cross-package scenario ownership, lexical fixtures and queue behavior.
- Validated authored values, construction, observation, updates and codecs.
- Higher-order error preservation and public generic operation requirements.
- Capture binding, context records and refactoring.
- Routes, forms, validation, operation-bound authorization and UI targets.
- Safe HTML authoring and observable protocol/browser assertions.
- Fallible sequencing, resource ownership and explicit cleanup.
- SQL mutation results and the precise schema-checking promise.
- Webhook verification, idempotency, outbox delivery and uncertain commits.
- Service lifetime, cancellation, shutdown, scoped results and race observability.
- Linux packaging, installation and native behavior qualification.
- Can browser execution, shared contracts, capability separation and the rich grid.
- Browser accessibility, client capability coverage and component lifecycle.
- Native AI wrapper, batch and callable composition limits.
- Controlled catalogue integration and external SDK requirements.
- Bulk immutable collection construction.
- Billing/calendar rules and domain-library boundaries.
- Named-field construction, multiline layout, formatter and diagnostics.

# Experiments that would distinguish the brainstorm options

This is an experiment inventory, **not a report of agent trials**. No syntax
candidate has been implemented, benchmarked or selected. The central comparisons
concern expressing programs, composing abstractions and successfully changing them.
Existing regression cases provide supporting checks against incorrect semantics.

Start with the [brainstorm overview](README.md). The experiments below connect
ideas to actual Can pressures so that later selection need not rely on aesthetic
preference, token counts alone, or a classifier's vote.

## Questions and controls

Separate four kinds of comparison:

1. **Representation:** hold semantics, admitted programs and obligations fixed;
   change textual/prefix/tree/table presentation and editing interface.
2. **Semantics:** change one rule, such as success binding or argument order;
   explicitly change the oracle for that experiment and document the delta.
3. **Tooling:** hold language and behavior fixed; change contextual information,
   diagnostics, structured edits or formatter support.
4. **Expressivity:** attempt a genuinely unsupported task, such as a generic
   error-preserving combinator. Record missing capability separately from poor
   spelling, agent failure or an implementation bug.

A new language with better diagnostics versus old Can with worse diagnostics
does not isolate a syntax benefit. A graph editor that guarantees well-formed
trees versus raw text does not establish that a graph is better repository source.
Run a text/tree editing comparison on the same semantic language as well.

For representation comparisons use the same resolved semantics and trusted
acceptance vectors, with independently checked translations. A translator bug
is a harness failure. Matching the same buggy evaluator is insufficient: target
execution, semantic oracles and fault/effect traces must agree where applicable.
Known B2–B5 failures remain explicit until repaired, and cannot fairly be scored
as failures of a new notation alone.

## Workload and counterexample matrix

The rows are concrete proposed tasks. They include generation, maintenance,
diagnostic recovery and attempts that should be rejected. Paths identify real
source material, not a promise that every proposed extension works today.

The task IDs are references, not priorities or weights. Library construction,
data modeling, composition, module edits and application behavior are the main
comparisons. E13–E15 check evidence and build integrity; passing them does not
establish that a language helps agents write better programs.

Begin a comparison with E20 as a whole-program construction task, then use smaller
tasks to explain where candidates differ. This prevents success on isolated
mutation or rejection cases from defining what makes a language useful.

| ID | Task and source seed | Vary | Discriminator and independent check |
| --- | --- | --- | --- |
| E20 | From intent, build a new multi-module configuration-processing program: discover supplied JSON/collection APIs, model inputs, validate independent fields, transform valid entries and produce a structured report; then extract a reusable abstraction and accommodate a new requirement | Computation/result/error model, data/types, module structure, ordinary abstractions versus derivation/templates, and compiler context discovery | Independently check unseen inputs and complete errors; measure correct construction and evolution, API discovery, context retrieval, boilerplate, failed attempts and guidance required. Give each candidate equivalent library capabilities and references; record unsupported features separately. No current Can solution is provided to copy. |
| E01 | Extend [`success__map`](../../../sketches/success-values/success.can) from int-to-int to int-to-record; instantiate identity over every supported success kind | Success conventions, typed/contextual constructors, call/invoke, patterns | Exact whole value preserved; field named `value`, empty nominal record, brand, variant, Seq, Bytes and Fn remain distinct. Unit is a separately specified candidate. Count semantic repairs, not just parsed programs. |
| E02 | Repair the [F06 named-argument case](../workstream-a/evidence-argument-order.txt); then format and reorder labels | Optional/required/no labels; source/parameter order; explicit evaluation nodes | On empty sequence and zero divisor, evaluator and fresh target report the same specified first fault; each argument evaluated once; ordinary outcomes retain correct parameter binding. |
| E03 | Add one declared callback error to the E01 map operation | Concrete combinators, row-polymorphic relay, first-class outcomes | Preserve original payload/identity; handle or relay every admitted error; callbacks with empty rows still work. Storing an error as successful data must not accidentally flatten it or count as handling it. |
| E04 | Express collect/zip with two failing branches, using [`Validate__Report`](../../../std/quota/quota.can) as an existing all-failures pressure case | Single error, error union, product/vector of outcomes, explicit aggregate | Both independent failures survive with branch identity. Define ordering, aggregation and whether computation stops early; never score a fail-fast candidate against an all-failures oracle without declaring that semantic difference. |
| E05 | Refactor [`std__seq__map_from`](../../../std/seq/seq.can) through a named local intermediate and then a helper | Expression nesting, immutable blocks, explicit control graph; full/local types | Scope/capture remains correct; no new early exit; result, first faults, callback count and evidence obligations preserved. Measure how much unrelated source must enter the agent's context. |
| E06 | Add a nested variant case to the [`Json__Value`/frame workload](../../../std/json/json.can) | Nested/product patterns, qualified cases, decision relations, default policy | Exhaustiveness and payload identity survive; unreachable/overlapping rows are treated according to declared rules; every new source decision has valid evidence. Avoid counting formatter-generated internal arms as independent authored choices. |
| E07 | Express generic pairs nested in Seq/Map and a finite recursive JSON tree | Compositional types, explicit/contextual type arguments, nominal/structural representation | Existing [`Map__Pair<K,V>`](../../../std/map/map.can) nesting remains expressible; ambiguity rejected locally; recursive host cycles rejected under any selected validating boundary. Generic examples cannot masquerade as universal laws. |
| E08 | Move a helper into another module, rename an import, then inject duplicate declarations in both file orders | Global/scoped/owned/package IDs, explicit imports, visibility, authored/generated metadata | Exactly one intended owner; no order-dependent type selection; accidental shadowing/case collision diagnosed at definitions; test execution-mode changes explicit. Seed from [F02/F04](../workstream-a/README.md#current-findings-and-validation). |
| E09 | Extract a callback from a factory or record field, then attempt to capture private state and introduce an indirect cycle | Invocation target positions, named handles, explicit capture sets, richer callable types | Valid admitted callables work; unsound provenance/effect/cycle cases fail before execution. Unifying application spelling earns no credit for bypassing these rules. |
| E10 | Implement map/fold and then alter an index guard, based on seq and [`std__ratio__gcd_from`](../../../std/ratio/ratio.can) | Current recursion schemas, structural folds, state transitions, broader measures | Distinguish termination, bounds safety and runtime resource limits. Negative/zero inputs return declared behavior. The same admitted algorithm yields equivalent target behavior; no claim of constant stack or timer progress without evidence. |
| E11 | Move a pure helper across files and change a callee's actual output while its script remains unchanged | Default scripted tests, linked-pure execution, explicit evidence modes | Record exactly which evidence detects the mismatch; never invoke real externs at compile time. Missing/leftover exchanges fail; fresh stores remain isolated; mode distinctions appear in diagnostics. |
| E12 | Write then falsify a contract; add a clause outside the supported proof fragment | Inline/separate contracts, result binders, proof diagnostics, evidence rows | False statement rejected; unsupported/timeout/unknown not reported as proved; body and contract binders denote the same values. A passing row suite is insufficient. Use [`authority-evidence`](../workstream-a/authority-evidence.md) as the baseline inventory. |
| E13 | Replay B1's callback target/capture/revision/type/dependency and invocation-input edits, plus formatting-only controls | Source/core/graph canonicalization; explicit declaration IDs; normalized references | Relevant meaning changes remain distinguishable; formatting remains inert; executable-body changes stay separate from interface and accepted-value promises. Seed from [`canonical_test.go`](../../../compiler/canonical_test.go). |
| E14 | Add an unknown semantic node, read an old/incomplete baseline, and edit implementation plus pinned expectation together | Exhaustive serializers, schema versions, authority stores, evidence layout | No unknown placeholder becomes valid evidence; generated candidates are not automatically accepted; changes to independently accepted expectations are surfaced by the declared review policy. Hash equality does not authenticate acceptance. |
| E15 | Compile all stdlib and representative consumer subsets, then strict-check and execute fresh complete TS bundles | Module layout, emitter/core architecture, runtime helper packaging | Ownership conflicts and target typing errors exposed; no stale checked-in TS or casts erase the failure. Source checks, examples, target checks and execution remain separately reported. Seed from [F04/F05 logs](../workstream-a/logs/commands.json). |
| E16 | Pass getters, aliased mutable Bytes/records, noncanonical decimals, raw JS functions and forged brands across the boundary | Trusted direct API, data-only validation, checked handles, copy/transfer protocols | Score against the policy actually promised: validation establishes its chosen guarantees; a trusted API must not claim adversarial enforcement. Preserve exact numerics, scalar-string rules, ownership and mint/disclosure authority. Seed from the [host probe](../host-boundary.mjs). |
| E17 | Two child operations: both fail, one times out, one completes twice or mutates late; include failed cleanup and attempted root reentry | B05 A/B obligations, structured regions, outcome products, durable workflow candidates | Chosen customer contract states completion versus settlement; no failure or exclusive authority disappears. Some obligations differ between A and B and cannot be silently assumed common. Use deterministic controllable hosts and separately test progress assumptions. |
| E18 | Repair malformed literals, nesting and ambiguous constructors; resolve a stale concurrent edit without weakening evidence | Lexical forms, delimiter rules, formatter, text vs typed patch, contextual query support | Parse/format/parse preserves value/control/evidence; comments and raw payloads survive; stale edits rejected or explicitly rebased; diagnostics localize the actual defect. Test quotes, backslashes, braces in comments, Unicode scalars, exact decimal zero and nested generic closers. |
| E19 | Create and split a module containing record, variant, error, brand/opaque, constant, state, host binding and function declarations; seed from quota, success-values and host | Keyword declarations, uniform named slots, manifest/body cells; declaration order, exports and revision placement | Resolve owners and forward references consistently; preserve nominal/mint authority; distinguish constant, interface and body changes; reject duplicate clauses and host/source confusion. Compare whole-module editing, navigation and error recovery rather than isolated bodies. |

An additional boundary task should start from a **selected real async customer**,
such as an HTTP request that uses and releases a database resource. Define required
observations, error payloads, authority lifetimes and platform assumptions first.
E17 is a catalogue of potential conformance cases, not a decision that every
customer needs the strongest possible host contract.

## Small controlled comparisons within those tasks

Use explicit hypotheses that can lose:

| Hypothesis | Comparison | Evidence against it |
| --- | --- | --- |
| More explicit local information improves editing | Full local types versus uniquely determined locals with the same signature and diagnostics | Extra annotations increase incorrect edits or repair cost without reducing semantic failures |
| Uniform application helps agents | Split call/invoke versus one explicit application, same admitted targets | Agents confuse source versus host/capability obligations more often under the unified form |
| Required labels reduce accidental swaps | Same multi-argument task with positional, optional-label and required-label grammars | Label drift and rename failures offset fewer same-type swaps |
| Prefix forms reduce grammar mistakes | Keyword/infix and prefix forms with matching semantics and quality of tools | Fewer parser failures but worse scoping or behavior under the same budget |
| Stable edit handles improve refactors | Resolved-node patching versus text patches on the same repository representation | ID churn, stale patches, accidental identity preservation or context-fetch overhead reduce successful edits |
| Whole-value outcomes improve composition | Declared-shape, whole-value, and raw-total candidates on the success matrix | Simpler map code moves complexity into contracts, nested outcomes or total/fallible interface changes |
| Decision relations improve evidence alignment | Expression/graph and decision-table bodies on small and large product spaces | Product blow-up, overlap errors or correlated specification/implementation mistakes increase |
| Local deterministic inference is enough | Explicit, constructor-context-only and broader local inference | Common tasks still need nonlocal guesses, or a remote signature change alters meaning without useful diagnostics |

Keep other dimensions constant for each initial comparison. Then try a few
combined directions from the overview to detect interactions. Testing the full
Cartesian product of every idea would consume substantial effort without first
establishing which mechanisms matter.

## Agent-trial protocol to establish before running

1. Build or adapt a small executable surface for the specific question. Give every
   candidate its admitted semantics, validator and formatter. Label unsupported
   capabilities; do not let a stubbed compiler make one candidate look superior.
2. Create equivalent training examples and a compact reference for each syntax.
   Match instructional quality. Record initial familiarity and adaptation effort;
   judge any familiar syntax by agent outcomes, without giving human preference
   its own score.
3. Use independently specified task intent and held-out acceptance checks that
   agents cannot weaken. Authored examples may be edited where the task allows,
   but acceptance authority stays with the harness. Include intentionally wrong
   solutions that pass the visible examples.
4. Pin model versions, reasoning settings, tool availability, prompts, context
   budgets, wall-time/token budgets and start states. Use fresh task contexts,
   rotate presentation order, and avoid leaking successful answers between arms.
   Include more than one agent configuration before claiming broad benefit.
5. Compare paired tasks and several independent runs, with both greenfield and
   maintenance tasks and injected-error recovery. Run a pilot to estimate variance
   before choosing a sample size; a few appealing transcripts do not establish a
   winner. Preserve failed, timed-out and unsupported runs.
6. Record final artifacts, semantic deltas, diagnostics, tool exchanges, resource
   use and independent results. Classify harness/compiler defects separately,
   report exclusions and reruns, and retain raw traces with any sensitive
   workload data removed under an explicit policy.

## Measurements and selection limits

The primary result is independently correct completion within the stated budget,
including intended semantics and acceptance/authority obligations. A candidate
that silently drops an error, changes fault order, forges a brand, or weakens
trusted intent has not succeeded merely because it compiles.

Report at least:

- Correct completions, incorrect accepted results, unsupported tasks and timeouts.
- Attempts to first valid program and to independently correct completion.
- Fault, error, effect, ownership and evidence violations by category.
- Total context/output tokens, tool calls, wall time and diagnostic-repair rounds,
  including costs of failures; separate conditional-on-success summaries.
- Unrelated edit footprint, stale-edit/merge conflicts and declaration/evidence
  identity churn under equivalent refactors.
- Compiler/runtime costs relevant to the chosen mechanism: checking latency,
  specialization growth, emitted size, copying, stack depth and allocation.

Publish distributions and uncertainty, not just means or a composite score with
invented weights. First eliminate violations of required guarantees, then compare
correct-completion and cost tradeoffs. If candidates serve different workloads,
record that result rather than manufacture one universal winner. Coverage, proof
completion, source brevity and classifier confidence are supporting observations,
not substitutes for successful software changes.

## Exit from brainstorming

The brainstorm is ready for a later design stage when someone can choose a
specific question, locate its competing options, understand the semantic and
authority obligations, and run a discriminating task. This record provides those
inputs now. Choosing which experiments to run and selecting a final design remain
separate decisions; no requirement to build five full compilers is implied.

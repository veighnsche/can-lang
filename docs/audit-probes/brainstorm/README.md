# Can language brainstorm

Completed exploration, 2026-09-19. **Options and hypotheses; no language design
selected.** The central question is: **what language would best let coding agents
express, understand, compose and change programs?** The scope is the whole
language, informed by the [Workstream A audit](../workstream-a/README.md).

The user explicitly distinguished brainstorming from designing. Accordingly,
this record opens the option space, supplies examples and counterexamples,
examines combinations, and identifies experiments. It does not ratify a grammar,
rank imaginary benchmark results, or turn implementation checklists into approvals.
There is no compiler implementation in this brainstorm.

## Start here

| Record | What it contributes |
| --- | --- |
| This page | Central language questions, competing directions, common example and cross-feature interactions |
| [Language forms](language-forms.md) | Calls, outcomes, sequencing, patterns, types, data and lexical alternatives |
| [Authority and evidence](authority-evidence.md) | Modules, effects, identity, tests, contracts, acceptance and termination alternatives |
| [Boundaries and tooling](boundaries-tooling.md) | Host/wire/async/resource alternatives and agent editing mechanisms |
| [Discriminating experiments](experiments.md) | Concrete tasks, counterexamples, controlled comparisons and evidence to collect |

## Objective and freedom

Optimize for independently correct software changes by AI coding agents: local
understanding, reliable generation, precise edits, diagnosis and recovery,
preservation of intended behavior, and affordable verification. Human familiarity,
beauty and comfort have no independent weight. Human-unfriendly forms are fully
eligible. Making a form unpleasant is not itself a benefit to an agent.

There are zero external users. Every existing spelling, delimiter, source layout,
public ABI and compiler representation can be replaced. Engineering cost remains
real; backwards compatibility is not a product constraint. A chosen replacement
would migrate the repository and remove obsolete forms.

Verbosity, explicitness, uniformity and one canonical form are hypotheses about
how to achieve the objective. They need concrete interpretation. Repeating an
error row in five places could reveal intent or create five opportunities for
contradiction. A compact form could reduce context cost or conceal a crucial
decision. A grammar easy to parse could still be hard for an agent to edit.

The [current requirements](../../../REQUIREMENTS.md) and
[implemented-language dossier](../workstream-a/shared-model.md) are inputs. In
particular, apparent “simplicity” must be checked against exact values, nominal
authority, explicit outcomes, exhaustive decisions, evidence integrity and
termination. A candidate that changes a guarantee must identify that change and
its obligation. Existing parser accidents are not guarantees.

The user's fixed objectives are coding-agent effectiveness, no independent
preference for human comfort, and no compatibility obligation. Current language
rules are the comparison baseline, not a ban on brainstorming alternatives.
Nominality, inference, evaluation strategy, error/effect models, recursion and
concurrency can be questioned explicitly. Compare different spellings under the
same semantics first when isolating syntax effects; compare semantic changes as
separate proposals with their benefits, costs and changed guarantees stated.

No external literature results or JEV classifications are claimed as measurements
of these proposals. The independent brainstorming passes used repository evidence
and generated hypotheses. Existing audit scores do not select these new options.

## Central language questions

Begin with what an agent needs to express. Representation, compiler architecture
and evidence machinery follow from those questions; they do not determine the
answer in advance. The five source directions below are one axis of the
brainstorm. The semantic choices here are equally important.

| Language question | Alternatives to explore | What makes the choice consequential for agents? |
| --- | --- | --- |
| What is a computation? | Nested expressions; ordered immutable blocks; explicit dataflow; decision relations and transitions | Can an agent add a step, reuse a value or change a branch without restructuring unrelated code? |
| What does a function return? | Declared-shape successes; uniform whole-value success; raw values for total functions | Can the same abstraction work for scalars, records, variants, collections and functions without special cases? |
| How are failures composed? | Concrete error sets; generic rows; first-class outcomes; explicit continuations or relay operations | Can agents build map, recover, retry, zip and validation while preserving the intended failures? |
| How does application work? | Distinct named/indirect calls; one explicit application; ordinary application; argument records | How much must be remembered about call positions, arity, labels and evaluation order? |
| What can be named and reused? | Expression nesting; immutable bindings; named helpers; explicitly captured anonymous functions | Can agents factor a repeated computation without accidental capture or unnecessary declarations? |
| How is repetition eliminated at larger scales? | Ordinary generics/combinators; compiler-owned derivation; checked declaration templates; staged or hygienic syntax extensions | Can an agent express a family of related operations once without opaque generated behavior or another dialect to learn? |
| How are data and decisions expressed? | Nominal or structural records; qualified variants; nested patterns; decision tables; finite recursive data | Are domain models natural to construct and change, and are affected cases easy to find? |
| What must be spelled out? | Every type and parameter; explicit interfaces with deterministic local context; compiler-materialized annotations | Does information remain available where needed without repetitive, inconsistent edits? |
| How are programs organized? | Global qualified declarations; module scopes; package ownership; explicit versus derived metadata | Can agents navigate, extract, rename and combine code with a small reliable context? |
| How are iteration and state expressed? | Recognized recursion; structural folds; explicit state transitions; owned regions and protocol handles | Which algorithms are expressible, and how much proof or lifetime bookkeeping do ordinary edits require? |
| How do source and evidence relate? | Inline tests/contracts; separately owned declarations; referenced contracts; typed evidence relationships | Can intent remain visible and independently checkable as implementation changes? |
| How does the program interact with the world? | Explicit effect rows; capability arguments; host sessions; structured joins; resource protocols | Can agents express real operations, concurrency and cleanup with clear obligations? |
| What grammar supports reliable edits? | Keywords, punctuation, prefix forms, explicit delimiters, canonical trees or tables | Which forms reduce scoping mistakes, ambiguous partial edits and diagnostic cascades? |

These alternatives are developed in the three topic documents. Cross them rather
than treating a choice of punctuation or source representation as a complete
language. A prefix language could have conventional error rows; a compact text
language could expose every capability explicitly; a table-oriented language
could retain ordinary algebraic data and separate test cases.

## Explore combinations through real programs

Three workload lenses keep the brainstorm anchored in software construction:

- **A reusable collection library:** optional values, map/fold, functions returning
  records, nested collections and multiple errors. Compare whole-value versus
  declared-shape results, callable construction, error abstraction and iteration
  together. Ask how many special cases an agent must learn to generalize one
  working function.
- **A JSON/schema transformation:** model recursive data, validate fields,
  compose transformations, report failures and encode the result. Compare direct
  recursive values with explicit machines, nested patterns with decision tables,
  and explicit type arguments with bounded context. Ask what changes when a new
  case or field is added across the pipeline.
- **An application operation:** validate a request, read or update owned state,
  perform external work and produce a response. Compare effect declarations,
  capability parameters, sequencing, joins and resource lifetimes. Async and
  resource variants are proposed extensions; existing stdlib pieces provide
  inputs, not proof that the complete operation is already supported.

For each lens, explore both an explicitly decomposed form and a more compositional
form. Smaller grammar, shorter programs and more explicit source can pull in
different directions. None is the objective by itself. The question is which
combination helps agents construct and maintain the intended program.

## Five substantially different directions

These are combinations to explore, **not a shortlist of approved designs or a
ranking**. A mechanism can move between directions. Each has a distinct hypothesis
about the unit an agent should generate and edit.

| Direction | Unit the agent works on | Possible combination | Agent benefit hypothesis | Main way it could fail |
| --- | --- | --- | --- | --- |
| F1. Explicit canonical text | Declaration and expression | Keyword-led syntax, explicit boundary/local types, labelled fields, explicit matches, colocated evidence | Local text contains the facts needed for repair; small patches remain comprehensible | Repetition and deep nesting exhaust context; many redundant facts drift |
| F2. Regular prefix tree | One tagged expression | Prefix constructors/operators, uniform application, whole-value outcomes, explicit type/effect parameters | Fewer punctuation and precedence exceptions; syntax mirrors tree edits | Parenthesis/argument errors, longer paths and unfamiliar forms increase repair cost |
| F3. Compact source with compiler context | A typed hole or selected expression | Explicit interfaces, deterministic local elaboration, nominal constructors, compiler-supplied scope/effects/evidence context | Agent receives precise facts on demand without restating them throughout source | Nonlocal inference or stale context makes a small edit unexpectedly change meaning |
| F4. Explicit control/data graph | Typed node and ordered edge | Ordered argument nodes, outcome dispatch nodes, declaration handles, separate evidence edges, atomic graph patches | Evaluation, dependencies and edit scope are explicit and mechanically addressable | ID churn, graph fragmentation and invalid intermediate connections obscure whole behavior |
| F5. Decision relations and transitions | Row or state transition | Exhaustive disjoint input partitions, explicit calls within selected rows, transitions with measures, attached witnesses | Implementation structure aligns with decision-table tasks and coverage obligations | Cartesian explosion; authored examples accidentally become the entire specification |

F1 can retain separate call/invoke forms as an experimental control or use one
explicit call. F2 need not imply Lisp-like runtime semantics. F3 need not allow
implicit conversions, inferred authority or overload search. F4 is not permission
to run a cyclic graph or reorder effects. F5 is not general logic programming,
proof search or execution of arbitrary test examples.

Also vary what is canonical: authored text, serialized typed tree, or a single
semantic artifact with generated views. Multiple *views* of one artifact need not
create independently editable authorities. Multiple accepted source spellings do
create normalization and evidence questions. Compare those policies explicitly;
do not promise both as permanent compatibility modes.

## One workload expressed five ways

The real starting point is
[`success__map`](../../../sketches/success-values/success.can): map a checked
pure callback over an optional value, including an `int` to nominal-record result.
On None the callback is not invoked; on Some it is invoked exactly once. The
callback's declared error set is empty. Its ordinary domain success still differs
from primitive faults. The result is one optional value of the new type.

**Every block below is proposal notation, not compilable Can or a complete
function.** They are equivalent intended *body* sketches under a deliberately
shared whole-value success convention. This holds that choice fixed to expose
representation differences; it does not choose whole-value successes. Declarations,
module ownership, checked callback admission and evidence are shared prerequisites.

F1, explicit keyword tree:

```text
match value
  case Option<T>.Some(item = x)
    match call transform(argument = x)
      success(mapped: U)
        success Option<U>.Some(item = mapped)
  case Option<T>.None
    success Option<U>.None
```

F2, regular prefix tree:

```text
(match value
  (case (Option<T>.Some x)
    (match-outcome (apply transform x)
      (success mapped (success (Option<U>.Some mapped)))))
  (case (Option<T>.None)
    (success (Option<U>.None))))
```

F3, compact text with uniquely determined local types:

```text
match value
  Some(x) => match call transform(x)
    Ok(y) => Ok(Some(y))
  None => Ok(None)
```

Here the candidate elaborator must resolve `Some`, `None`, `x` and `y` uniquely
from explicit function and callback signatures, and expose those resolutions to
the agent. Constructor ambiguity is an error, not a search for a convenient type.

F4, control graph with block parameters:

```text
entry(value: Option<T>)
  dispatch value
    Some(x) -> present(x)
    None -> absent()
present(x: T)
  apply transform(x)
    success(y: U) -> mapped(y)
mapped(y: U)
  finish success Option<U>.Some(y)
absent()
  finish success Option<U>.None
```

These are structured control edges, not an arbitrary goto/cycle facility. Adding
a back edge would require an admitted termination rule. Ordinary immutable data
may be reused; this sketch does not make all values linear.

F5, decision relation:

```text
decision map_optional(value: Option<T>, transform: Fn<T,U,[]>): Option<U>
  partition       computation                  finish
  Some(x)         apply transform(x) as Ok(y)   success Some(y)
  None            none                         success None
```

The computation column executes only in the selected row. `as Ok(y)` here is a
full dispatch only because the callback has no domain errors; it cannot silently
discard an error if the signature expands. Overlapping partitions need a declared
priority rule or rejection. Table cells must have a compositional syntax when a
single expression becomes insufficient.

The immediate discriminators are editing the callback result from scalar to
record, introducing one declared callback error, adding a variant case, and
moving the callback across a module boundary. A pleasant-looking tiny example is
insufficient: each family must survive those edits and the larger JSON workload.

## Interactions that isolated spelling votes miss

| Pair of choices | Coupling or conflict to explore |
| --- | --- |
| Named arguments × formatting × evaluation | Parameter binding order and expression evaluation order are separate. Sorting labels can alter faults; safe reordering requires proof or explicit evaluation nodes. |
| Success convention × generics × patterns × contracts | The binder must denote the same value in body, example and ensures clause. A record field named `value` must not be confused with an outcome envelope. |
| Completed outcomes × error accounting | Storing a failure as data, handling it, relaying it and dropping it are distinct operations. A row union describes possible kinds, not two simultaneous failures. |
| Unified application × capabilities × termination | One spelling does not make arbitrary targets safe. A factory result or field-held callback still needs provenance, precondition, effect and indirect-cycle analysis. |
| Local inference × identity × diagnostics | Decide whether an annotation removal preserves evidence when elaboration is identical, and whether a distant signature edit can change elaboration. |
| General blocks × early exits × resources | Local naming can be added without hidden early return. Every introduced exit must define outcome accounting, branch evidence and cleanup. |
| Rich patterns × branch evidence | Desugaring product/nested/or patterns can multiply internal branches. Decide which authored decisions need which witnesses. |
| Source IDs × refactors × acceptance | Stable IDs can retain evidence across movement, but copying or reusing an ID must not transfer ownership or acceptance accidentally. |
| Evidence colocation × acceptance authority | A shared file is compatible with separate authority; separate files alone do not protect accepted expectations. |
| Host facade × nominal values × wire codecs | Shape, canonical value, ownership, minting permission and serializability are different obligations. |
| Async joins × effects × multiple failures | Scheduling cannot erase competing errors or lend the same exclusive capability to concurrent children. Timeout and actual settlement differ. |
| Canonical text × structural editing | A text view must round-trip without changing evaluation/evidence; a semantic patch needs stale-base detection and a reviewable diff. |
| More general recursion × exact arithmetic | Termination does not promise stack safety, bounded allocations or timer progress during large integer work. |

Combinations deserve exploration in both directions. Explicit prefix syntax
could use local inference. A compact textual language could require explicit
capability arguments. A decision relation could be a compiler-generated view of
ordinary matches. Graph patches could edit a canonical textual source through
resolved nodes without making the graph the repository's source of truth.

## Compiler and specification hypotheses

Three foundations remain plausible: a resolved source AST with disciplined shared
visitors; a typed core plus a source/evidence map; or a canonical typed graph with
generated text. Compare completeness of canonicalization, diagnostic localization,
lowering agreement and cost of a real semantic slice. The current type-string
representation creates pressure, but does not prove that a full compiler rewrite
must precede every useful change.

Likewise, consider an authored grammar with conformance fixtures, a schema that
generates parser/formatter/tool interfaces, or a small authoritative semantic core
with separately tested presentations. A shared generator can spread the same bug
to all consumers; independent differential checks remain useful. A single
intrinsic contract registry might align evaluator, emitter and proof admission,
but a descriptor declaring itself pure is not evidence of a correct implementation.

Machine-facing context could expose resolved interfaces, authority paths, allowed
next forms, missing obligations and the smallest relevant source slices. Compare
that with repeating all information in source. Neither approach should let an
agent edit compiler-derived facts into authority or treat an incomplete program
as runnable.

## Relationship to Workstream B

Correctness repairs supply regression cases for language alternatives. They do
not privilege the compiler structures they happen to repair. Unambiguous ownership,
composable libraries, valid target bundles and specified evaluation order matter
under any chosen syntax.

Brainstorming therefore supplies design questions to B without blocking its
remaining correctness repairs. A later design can replace an affected code path;
carry the corresponding invariant and regression into the replacement. Do not
implement a speculative new grammar just to close B, or invest in compatibility
adapters for a discarded one.

B1 is included as one completed repair relevant to evidence identity. Its
implementation is replaceable. [Supporting implementation notes](support/implementation-notes.md)
retain the observation and test receipts outside the language exploration.

## Coverage and completion

| Audit work | Where the brainstorm explores it |
| --- | --- |
| B1 / F01 | Authority/evidence alternatives; supporting implementation notes; regression tasks E13–E14 |
| B2–B5 / F02, F04–F06 | Ownership, target gates and evaluation interactions; experiments E02, E08, E15 |
| C1–C3 | Specification/core hypotheses; types and elaboration; source/evidence mapping |
| D1–D5 | Language forms: successes, calls, bindings, patterns, contextual typing |
| E1–E4 | Authority/evidence: names, ownership, effects, revisions, authored metadata |
| F1–F5 | Authority/evidence and boundaries: tests, proof, termination, faults, binders |
| G1–G3 | Language forms: errors, higher-order functions, compositional and nominal data |
| H / F03 | Boundaries/tooling: four contracts, trust, value/authority checks and wire data |
| I / B05 | Boundaries/tooling: competing async/resource models and customer obligations |
| J1–J3 | Language forms plus boundaries/tooling: lexical forms, formatter, diagnostics, intrinsics |
| K1–K3 | Real workloads and controlled experiments; migration implications, no migration performed |

This brainstorming pass is complete in the following bounded sense: all audited
language areas have alternatives, agent-benefit hypotheses, counterexamples and
discriminating tasks; radically different representations and cross-feature
combinations are included; implemented behavior and outstanding defects are accounted for. It does
not exhaust every possible programming language.

No winner has been measured or selected. No agent comparison, parser prototype,
new formal grammar, ABI migration or compiler redesign was performed. Those are
distinct subsequent experiments/design/implementation activities. The resulting
option space is ready for that work without treating any appealing sketch as the
answer in advance.

An independent review checked audit coverage, factual scope, example semantics
and premature selection. Its findings were addressed, including missing
declaration/contract/callable authoring alternatives and wire/runtime comparisons.
Document links and experiment identifiers were checked separately. These checks
establish the integrity of the brainstorm record, not the soundness or usability
of an unimplemented language candidate.

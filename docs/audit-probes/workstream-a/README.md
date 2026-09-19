# Workstream A — replacement language audit and JEV review

Completed 2026-09-19 against language/compiler baseline
`8bbcf13b33c74d26bc7bb0f9b72fba0d5415d70b`.

**Result: repair the confirmed defects and establish an accurate current
specification before treating the redesign proposals as requirements.** All
thirteen replacement JEV judgments completed with self-contained context. Six
selected `defer`; several others were split. The evidence supports correctness
work now, but does not ratify a new success convention, application syntax,
namespace model, test default, public ABI, error algebra or surface language.

This completes [TODO Workstream A](../TODO.md#a-redo-the-jev-review-correctly).
It does not complete the implementation workstreams. No compiler, language,
generated golden, accepted revision baseline or public ABI was changed. This is
a working-tree audit record; no implementation commit is claimed.

## Start here

- [Shared Can model](shared-model.md): goals, constraints, current semantics,
  authority and limits; supplied in full to every review.
- [Language core](language-core.md): values, outcomes, calls, patterns, generics,
  functions, recursion, literal/type/label rules and counterexamples.
- [Authority and evidence](authority-evidence.md): modules, effects, ownership,
  revisions, tests, contracts, proof, acceptance and documentation reconciliation.
- [Host and B05](host-async.md): exact TS layouts, trust, faults, four separate
  boundary layers, customers, and both competing async designs.
- [Method](method.md), [questions](questions.json), [packet manifest](packet-manifest.json):
  scope, balanced options, source hashes, complete-context selection and limits.

Each request contains actual source examples and implementation evidence relevant
to its decision. Repository paths serve as provenance, not missing-context links.
Current, proposed and historical statements are distinguished. The old JEV files
remain unchanged and withdrawn; their scores were excluded from all new inputs.

## Current findings and validation

| Finding | Current evidence | Interpretation |
| --- | --- | --- |
| F01 | [Core probes](evidence-core-probes.txt): changing a pinned callback target and factory body produces equal canonical expectations and no warning. Typed-Ok/pattern/invoke-argument forms also collide structurally. | Acceptance canonicalization is incomplete. The structural probes do not demonstrate a persisted proof-cache exploit. |
| F02 | [Core probes](evidence-core-probes.txt): conflicting `Shared__Record` declarations produce no diagnostic; swapping file contents changes the chosen shape from int to str. | Declaration ownership is ambiguous and source-order-dependent. |
| F03 | [Host probe](evidence-host-integration.txt): alias mutation observed, two getter executions, one raw JS callback effect. | Direct exported TS is a trusted embedding boundary. This is an ownership/provenance design gap, not a breach of a promised hostile-host sandbox. |
| F04 | [Raw whole-library log](logs/whole-stdlib.txt): incompatible ratio/scalar `math.nonterminating_decimal` schemas fail later at a scalar payload. | Per-module green rows do not establish stdlib composition. |
| F05 | [Fresh Can log](logs/fresh-json-can.txt): 785 rows pass and two modules emit. [Fresh strict TS](logs/fresh-json-ts.txt) fails on four imported/local aliases and widened JSON discriminants. [Repository TS](logs/repository-ts.txt) separately has missing provider paths. | Source checking, example execution, emission, strict target checking and target execution are different gates. |
| **F06 — new** | [Complete argument-order reproduction](evidence-argument-order.txt): on the same well-typed inputs, evaluator reports sequence-bounds failure first and fresh TS reports division by zero first. | Named-argument lowering changes primitive-fault priority. No wrong host type or side effect is needed. Ordinary non-faulting results agree. |

Normal baseline gates pass: `go test -count=1 ./...`, module check (65 current
modules), and grammar check. The known failing integration gates above remain
failures; no assertion or golden was weakened to hide them.
[Baseline record](baseline-checks.json), [exact integration commands and exit codes](logs/commands.json).

F06's trigger is concrete:

```can
match call order__take(right = xs[index], left = 1 / divisor)
  on Ok result => Ok(result.left, result.right)
```

The callee declares `left` before `right`. With `xs = Seq<int>[]`, `index = 0`
and `divisor = 0`, both argument expressions fault. The evaluator walks written
argument order, while emission produces:

```ts
order__take($canDivMod(1n, divisor)[0], $canSeqAt(xs, index))
```

JavaScript consequently evaluates division first. The probe executes the real
evaluator and fresh target, with four passing ordinary Can rows and matching
normal Node results as controls. The existing binder comment promises source
order; the current emitter contradicts it. Record the intended order explicitly
and add a differential regression before fixing the lowering. Audit constructors
and other argument-binding consumers for the same class of mismatch. The
temporary Go probes were removed; their full sources and outputs are retained.

## Replacement JEV results and audit disposition

Pinned model: **`jev-1.13.0`**. Thirteen successful HTTP responses, one per
context-specific question; no retry or resampling. Total reported usage:
**77,107 input tokens, 743 output tokens**. Each request contained 21,932–28,728
serialized UTF-8 bytes and reported 5,262–6,802 input tokens. All were below the
prechecked provider limits. [Provider limit evidence](provider-limits.txt).

`P(choice)` and confidence are separate returned fields. Neither is a soundness
proof, a usability measurement, a probability that the design is correct, or user
approval. JEV provided classifications, not prose rationale. The reasoning in
the last column below is the coordinating auditor's independent assessment.

**Accepted** means accepted as an audit recommendation with the stated scope.
It does not mean a new source rule or ABI has been ratified. **Unresolved** means
implementation must not use the judgment as a settled redesign requirement.

| Question / full receipt | JEV choice | P(choice) | Confidence | Auditor disposition and evidence |
| --- | --- | ---: | ---: | --- |
| [Sequencing](responses/sequencing.json) | repair_and_specify | 0.58 | 0.47 | **Accepted.** F01/F02/F04/F05 and F06 justify repairs and an accurate inventory/gate set. Choosing a new typed core versus a vertical semantic slice remains separate; `repair_then_core` retained 0.35. |
| [Successes](responses/successes.json) | declared_shape | 0.48 | 0.31 | **Unresolved.** `defer` is 0.46. Current B11 works across several success types but maintains a split convention. The uniform alternative still needs Unit, binder/contract rules and a separate public envelope. This near tie justifies neither permanent flattening nor an immediate ABI migration. |
| [Calls](responses/calls.json) | retain_distinct | 0.74 | 0.65 | **Unresolved long-term.** Present target restrictions support a bounded implementation, but do not prove two spellings are best. Do not widen callable positions before authority/cycle/evaluation-order rules exist. Preserve current behavior while fixing F06; compare the fully specified unification alternatives later. |
| [Modules](responses/modules.json) | strict_global | 0.60 | 0.50 | **Accepted as the immediate repair scope.** Reject duplicate identities and import one owned schema. F02/F04 do not by themselves prove that scoped names, packages or a visibility redesign are necessary. Long-term module identity/visibility remains unresolved. |
| [Pure-source tests](responses/pure_source_tests.json) | contract_boundary | 0.30 | 0.07 | **Unresolved.** Execute-pure is 0.29 and explicit modes 0.25. Scripted isolation and real composition have different benefits; linked mismatch tests show why both need visible evidence. Specify admission and coverage credit before changing the default. |
| [Host boundary](responses/host_boundary.json) | defer | 0.43 | 0.25 | **Unresolved.** F03 establishes the present trust contract, not the intended threat model. Choose controlled trusted embedding versus enforced public ingress first. A validated data-only facade and explicit trusted fast path deserve concrete customer/cost evaluation; TS readonly alone is insufficient. |
| [Error abstraction](responses/error_abstraction.json) | defer | 0.40 | 0.21 | **Unresolved.** Rows-first has 0.33. Specify finite-row operations/variance, relay versus recovery, failure consumption and catalogue accounting. A row union cannot alone preserve simultaneous failures; completed outcome products need their own representation and authority rules. |
| [B05 process](responses/b05_process.json) | contract_then_compare | 0.62 | 0.53 | **Accepted as a process.** Resolve source and host contracts against a real customer and common acceptance vectors before selecting a backend. This does not require two full implementations or select a Promise public facade. Prototype only the uncertainties requiring measurement; Resources remains separate. |
| [Contextual typing](responses/contextual_typing.json) | defer | 0.51 | 0.34 | **Unresolved.** Constructor-only context has 0.35. A compositional type model and explicit ambiguity/elaboration rules must precede omission. Compare it with fully explicit applications; never infer authority or laws from examples. |
| [Collection spelling](responses/collection_spelling.json) | defer | 0.54 | 0.38 | **Unresolved.** Regular application has 0.44. Compositional nesting is a semantic/compiler question; `Seq<T>`, `[T]` or `T[]` spelling needs independent usability evidence. Existing Map/Seq nesting disproves a blanket nesting ban. |
| [Literals](responses/literals.json) | defer | 0.70 | 0.60 | **Unresolved.** Current explicit families have 0.28. Exact decimals do not require a punctuation change. Measure escapes/raw-text and numeric edit errors before changing defaults; keep exact canonical values and no coercion. |
| [Boolean evaluation](responses/boolean_evaluation.json) | eager | 0.55 | 0.40 | **Unresolved as a permanent design choice.** Keep the current eager rule during correctness repairs. `defer` is 0.37; short-circuiting needs explicit branch-evidence/fault rules and measured agent trials. The audit does not authorize changing evaluation semantics. |
| [Argument labels](responses/argument_labels.json) | defer | 0.45 | 0.26 | **Unresolved.** Preserve-optional is 0.44. First repair/specify evaluation order (F06), then measure labels versus positional error rates. Label retention is a tooling decision distinct from correct parameter binding. |

The audit therefore accepts **three bounded process/correctness recommendations**
and leaves **ten design choices unresolved**. Retaining current behavior while a
decision is open is not a compatibility requirement or a permanent dual mode.
Once a replacement is chosen, migrate the repository and remove the old form.

Rejected as decision support or implementation requirements:

- The withdrawn scores, or any new probability, as approval of the previous
  whole-language redesign. Changed context and candidate sets also prevent a
  statistical before/after comparison of the distributions.
- “The current winner must be preserved forever.” The choices about current
  success/call/boolean forms establish no compatibility obligation.
- Treating the B05 proposals as interchangeable runtimes for already-settled
  source semantics. They differ on labels, singleton admission, host authority,
  rejection mapping, timeout versus cleanup, receipts and world failure. Both
  also predate current broadened success types.
- Treating a type cast, readonly annotation, freeze or shape check as callable
  purity, brand-minting authority or a complete host ownership proof.
- Treating green Can rows or unchanged TS goldens as strict target validity,
  universal proof, or evidence of all-stdlib composition.

## Decisions that must precede redesign implementation

For successes, settle the whole-value/binder matrix, empty records versus Unit,
contracts and nested outcomes, then the independently versioned public facade.
For calls, settle admitted target positions, argument order, explicit outcome use,
preconditions and indirect cycles. For modules/tests, settle ownership and the
meaning of each evidence mode rather than inferring them from file layout.

For host/errors/B05, settle trust and ownership, finite error composition and
simultaneous-failure representation, source/private/public/wire separation,
settlement and resource obligations. In particular proposal A's dynamic deadline
and declared rejection normalization are not proposal B's trusted T/S cleanup
and failed-world receipt protocol. A backend label settles none of those facts.

Surface choices need deterministic parser/formatter/elaboration rules plus actual
agent edit and diagnostic-recovery trials. No such usability trial or runtime
performance comparison was performed in this Workstream A audit. These missing
measurements are recorded as unresolved evidence, not fabricated from JEV scores.

## Completion evidence for Workstream A

| Checklist requirement | Completion evidence |
| --- | --- |
| Withdraw old reviews, preserve history | Original files unchanged; existing withdrawal retained; new review explicitly excludes old scores. |
| Research Can; goals, guarantees, non-goals, zero users | Shared model and all three dossiers; actual compiler/source evidence. |
| Values through host boundaries, including exact layouts/numerics | Core/authority/host dossiers; canonical decimal examples, bigint, tags, ownership and four-layer distinction. |
| Stdlib customers, counterexamples, F01–F05, both B05s | Dossiers and current reproductions, plus newly confirmed F06. |
| Resolve contradictory documentation before asserting facts | Dedicated reconciliation sections; current implementation distinguished from historical promises and proposals. |
| Balanced alternatives, consequences, obligations, migration, examples | Question definitions plus dossier decision sections; actual requests include full option descriptions and abstention. |
| Check limits; self-contained context without truncation/memory | Provider evidence, deterministic packet builder, manifest and all thirteen complete request bodies. |
| Repeat eight architecture and five surface questions | Thirteen HTTP 200 validated response receipts and original response bytes. |
| Preserve model, supplied context, answers, probabilities/confidence | Requests/responses, hashes, timestamps, model pin and reported usage. |
| Review against evidence, record accepted/rejected/unresolved | Disposition table and explicit rejected inferences above; no probability-to-approval rule. |

The next implementation work is the independently evidenced correctness repair
scope, including F06. Major redesign decisions remain explicit follow-up work;
this audit does not grant them implementation authority.

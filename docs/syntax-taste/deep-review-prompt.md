# Deep review of Can's language design

Review the complete Can language design in `/Users/vince/Projects/can-lang`. This is a critical design audit, not an implementation task or another syntax questionnaire. Do not assume approved decisions are good merely because they are approved. Challenge flawed decisions with evidence and concrete alternatives. Do not change language decisions, compiler code, or existing documents. Deliver a standalone review report.

## Authority and scope

Read `AGENTS.md` and the entire `docs/syntax-taste/decisions.md`, including examples and unresolved sections. That document records the current design; it is not proof that the design is coherent or complete. If it contradicts itself, report the conflict instead of choosing one interpretation silently.

Latest scope correction: **LLM tool calling is out of scope. We are not doing tools.** The decision document may still describe tools as deferred or required; flag that stale wording, but do not propose tool declarations, execution loops, or tool-calling features as missing requirements.

Read `docs/ASTRA_STDLIB.md` for intended capability coverage. Its old syntax, type assumptions, implementation strategies, and roadmap are not authoritative. Review other design documents, examples, and source code as historical evidence and sources of candidate requirements, not as approved decisions. Relevant older native-AI proposals (the a94–a97 notes, since removed) and related documents. Find their actual locations instead of assuming every reference is current.

All existing implementation, including uncommitted changes, may be old or based on rejected research. Do not infer approval from implementation status, commit status, a legacy test, or an old golden. Preserve unrelated work.

## Design priorities

- Can is primarily functional, immutable, and designed for AI coding agents. Human typing convenience is not the primary goal.
- Native probabilistic AI judgments are foundational, not an optional SDK wrapper. Text and user-shaped structured LLM responses are also in scope. Authors can transform generated data and supply it to later judgments; there is no compulsory LLM-to-Choice pipeline.
- Compile to TypeScript running on Bun, reusing equivalent native operations. Add only adapters required for Can's contracts and immutability. Do not hand-reimplement upstream algorithms without a demonstrated need.
- Prefer a simpler compiler, but evaluate specific tradeoffs. Do not turn Can into TypeScript merely to minimize compiler work.
- There are zero external users. Backward compatibility is not a product requirement.
- Explicit, coherent contracts matter. Avoid hidden coercions, implicit features, and accidental semantics.
- All authored functions are named and top-level. Existing callable values and `near` captures are deliberate. Identify where new executable constructs create tensions with those rules rather than silently treating them as exemptions.
- Approved syntax is the baseline for analysis, not immune to criticism. Every proposed revision must be clearly distinguished from a current decision.

## Review process

1. Build a concise map of the current rules and their dependencies. Distinguish settled decisions, provisional choices, deferred work, illustrative excerpts, and requirements without syntax. Check that every relevant section was read.
2. Trace realistic end-to-end programs through the proposed language. Do not settle for reviewing individual snippets in isolation. Where behavior or spelling cannot be derived, stop that trace and identify the exact missing rule. Do not invent a rule just to make an example work.
3. Compare interacting constructs across the whole design. Look for conflicts between declarations, invocation, scoping, typing, completions, assertions, lowering, and runtime behavior.
4. Check intended standard-library and application capabilities against the design. Identify requirements that cannot be expressed, are needlessly difficult, or require violating another rule.
5. Research external runtime/provider facts when needed, using current primary documentation. Distinguish verified facts from inference. Never assert that native TypeScript/Bun or TypeSafe behavior solves a Can-specific problem without explaining the adapter and its limits.
6. Produce a prioritized, bounded report. Consolidate findings with the same root cause. Avoid turning every unspecified implementation detail into a user decision.

## What to look for

Audit gaps, ambiguity, overlap, redundant mechanisms, contradictory rules, incompatible constructs, impossible examples, missing edge cases, accidental complexity, unsound assumptions, and poor tradeoffs. Pay particular attention to capabilities blocked by otherwise attractive local decisions, and opportunities where one focused improvement resolves several limitations.

Review at least these interactions, without treating this list as exhaustive:

### Grammar, types, and names

- Whether each example can be parsed unambiguously under the recorded indentation, punctuation, expression, and naming rules.
- Type/value namespaces, same-scope collision rules, shadowing, generated record names, option names, confidence/score bindings, and metadata fields.
- Explicit return types, field types, generics, callable contracts, nominal records, variants, and dynamic data.
- Contextual `%` versus binary remainder; its meaning and type in each AI form; numeric scales, confidence, thresholds, and conversions.
- Reusable `choice_arm` values, their captures and invocation context, and their relationship to the ban on anonymous/nested functions.
- Whether provisional syntax is consistently identified and superseded syntax has leaked into current examples.

### Calls, completion, and composition

- Ordinary calls, direct bindings, `match call`, `match chain`, forwarding, terminal completion, `do`, and unnecessary-local restrictions.
- Domain errors versus standard runtime failures, explicit `emits`, handler coverage, standard failure propagation, and generated aggregate failures.
- Callable capture timing, exact-name `near` captures, direct-call arguments, receiver capture, wrapper functions, and arrays of different callables.
- Whether a construct returns a payload, a completion, a generated record, a collection, or control flow—and whether those meanings compose consistently.
- Missing, empty, repeated, unreachable, or differently typed cases; do not reintroduce rejected empty argument slots or anonymous tuples.

### Async and coordination

- Automatic waiting with no authored promises/async/await, and how coordination avoids awaiting its inputs sequentially.
- The selected native mappings: `concurrent` to `Promise.all`, `concurrent with error` to `Promise.allSettled`, `race` to `Promise.any`, and `race with error` to `Promise.race`.
- Per-call handlers, shared handlers, `errors` groups, spread callable collections, handler execution order, handler failures, and collected result types.
- Empty collections, duplicate error kinds, all-failed races, mixed standard/domain failures, heterogeneous successes, and calls that never finish.
- Outstanding operations after early settlement, cancellation/lifetime boundaries, resource ownership, and observable side effects. Distinguish necessary contracts from optional new features.
- Sequential collection operations with async callbacks; immutable native-method lowering; places where direct native APIs do not preserve the chosen semantics.

### Native judgments and generation

- Separation of `judge` inputs and shared `state`; grouped state arguments at invocation; one batched request; dependencies between questions; binding and handler timing.
- Connection selection at both judge and question declarations; contradictory connection settings and provider compatibility.
- Noul threshold behavior and raw probability access; Choice winner versus full-record transformations; Score's weighted numeric result versus level probabilities; separate confidence semantics.
- Named generated record types, generated metadata fields, return-type constraints, and field collisions.
- Runtime-generated option sets versus statically named fields and exhaustive handlers. Description-only options versus executable reusable arms.
- User-defined LLM response shapes, schema validation, invalid outputs, supported record/type shapes, and ordinary data transformations into later questions or state.
- Request failures, validation failures, handler failures, and which layer owns each contract.
- No LLM tool calling: do not expand scope into it.

### Fetch, connections, and boundaries

- Named fetch declarations, typed decoded response records, ordinary invocation, query/header sections, input parameters, request bodies, response status/headers, and text/bytes requirements.
- Generic connection configuration, authentication sources, endpoint resolution, timeout units/scope, metadata ownership, and local compatible services.
- Reusing native transport while retaining native AI meaning; whether the design accidentally requires forbidden backend escape hatches.
- Browser and server feasibility, including credential placement and differences between Bun and browser APIs. Identify constraints without inventing a new deployment architecture.

### Assertions and verification

- Mandatory function assertions, call-site `when`, expected arguments, supplied outcomes, test context, and real execution.
- Repeated/dependent calls, multiple matching fixtures, concurrent call order, transitive tests, AI response fixtures, and malformed fixture handling.
- Whether a stubbed success is being confused with proof of real provider behavior or model quality.
- Where newly introduced declarations lack a coherent testing surface, and whether existing mechanisms can cover them before adding new syntax.

### Capability coverage

Trace representative flows such as: fetch data, request a typed LLM summary, transform its output into a runtime-defined question, batch independent judgments, inspect confidence, coordinate different database callables, handle all failures, and return a typed result.

Also check a small HTTP service and an immutable frontend state-update/render flow against the intended standard-library scope. Identify which gaps belong to the language, standard library, provider adapter, runtime, or merely documentation. Do not assume every library requirement deserves a new keyword.

## Finding format

For each substantive finding provide:

- A clear title and severity: blocker, major, moderate, or minor.
- Classification: contradiction, missing contract, syntax ambiguity, capability blocker, implementation mismatch, stale documentation, or optional improvement.
- Exact evidence from current decisions with file/line references; distinguish old-source evidence.
- A minimal concrete Can example or execution trace demonstrating the problem.
- Why it matters and which real capability it affects.
- The smallest coherent recommendation, plus an alternative if there is a meaningful tradeoff.
- Which approved decisions would change, if any, and the consequences for grammar, compiler complexity, runtime behavior, and native reuse.
- Whether it requires user-visible syntax approval, can be settled by technical design within existing authorization, or needs further evidence.
- Confidence in the finding and any assumptions needed to reproduce it.

Do not label a speculative concern as a proven contradiction. Do not report bare unsupported statements such as “this is ambiguous” without showing competing interpretations. Do not count formatting placeholders or explicitly incomplete excerpts as errors unless the required complete form is genuinely missing.

## Output

Deliver a single review report containing:

1. A short readiness assessment with the most consequential findings. Do not claim the design is complete just because its syntax has been selected.
2. Prioritized findings in the format above.
3. An end-to-end capability matrix: supported, supported with unresolved details, blocked, or intentionally out of scope.
4. A compact interaction map showing cross-cutting root causes.
5. A small prioritized repair plan, separating documentation cleanup, technical design, and genuinely new/revised surface syntax.
6. A bounded list of user decisions that are actually necessary. Give a count, but explain the audit boundary rather than claiming no other gaps can ever emerge. Where a syntax choice is needed, show three concrete alternatives using the same example and recommend one.
7. A list of deliberate exclusions and sections not fully verified.

No implementation, no silent rewrites, no automatic adoption of recommendations. Aim for a design that is coherent and capable, not a longer catalogue of features.

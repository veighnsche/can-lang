# Core-contract technical consultations

24 September 2026. These are five independent technical trial judgments for
DI-02, DI-05a/b, DI-06 and DI-07. They do not adopt semantics, select source
syntax or revise the canonical specifications. All three requests used
`jev-latest`; every response reports `jev-1.13.0`.

The [first request](request-1.json), [second request](request-2.json) and
[third request](request-3.json) each contain the complete evidence, constraints,
counterevidence and candidate contracts for all five questions. Their
[first](response-1.json), [second](response-2.json) and
[third](response-3.json) responses and HTTP metadata are saved separately.
The [wording audit](wording-audit.json) records manual semantic-equivalence
review and mechanical checks that all 33 explanatory fields differ in every
request pair. Technical identifiers and answer labels remain stable; earlier
Jev answers were not supplied. [The script](consult.py) reproduces the requests
and calls the API only with `--send`, reading `TYPESAFE_API_KEY` from the
environment without printing it.

## Method and evidence boundaries

The consultation followed the live TypeSafe [HTTP API](https://docs.typesafe.ai/api),
[state guidance](https://docs.typesafe.ai/concepts/state),
[Choice contract](https://docs.typesafe.ai/primitives/choice) and
[choice-consistency cookbook](https://docs.typesafe.ai/cookbooks/consistency_choice_cookbook).
Five independent questions share one state in each of three fresh calls. A
`defer_measure` option was available in every question. The cookbook's example
confidence threshold was not imported as a universal decision rule. The API
reported 10,436 input and 848 output tokens in total.

Evidence came from [core evidence](../core-evidence.md),
[packages/assertions](../packages-assertions-evidence.md),
[complete core alternatives](../core-alternatives.md),
[inventory](../decision-inventory.md), [constraints](../constraints.md) and
[evaluation protocol](../evaluation-protocol.md). Historical executed probes
and current source/test observations were distinguished in every request.
Existing [pattern](../jev-pattern/findings.md) and
[generic-helper](../jev-generic/findings.md) choices were not repeated.
This consultation ran no new compiler/runtime or comparative agent tests.

## Results

Each cell shows the highest-probability option, its probability, and the
response's distribution-derived confidence in parentheses.

| Question | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| DI-02 validated values | owner record 0.47 (0.30) | owner record 0.53 (0.37) | owner record 0.52 (0.35) |
| DI-05a package lookup | manifest handles 0.50 (0.32) | qualified imports 0.63 (0.51) | qualified imports 0.92 (0.89) |
| DI-05b error reporting | audit first 0.62 (0.49) | audit first 0.50 (0.33) | scoped numbers 0.40 (0.19) |
| DI-06 fixture ownership | exported seams 0.53 (0.38) | exported scenarios 0.39 (0.19) | lexical only 0.44 (0.26) |
| DI-07 variant policy | extensional leaves 0.84 (0.79) | baseline matrix first 0.65 (0.54) | extensional leaves 0.57 (0.43) |

Only DI-02 has an unchanged leading label, and its distributions are diffuse.
All other topics exhibit ranking disagreement. These are recommendations about
experiments, not evidence of implementation correctness, agent benefit or user
preference. Agreement neither proves the chosen contract nor removes framing
bias; fixed alternative order and shared contextual framing remain limitations.

## DI-02: boundary completeness precedes a favored prototype

**Jev advice:** owner-controlled record construction leads all three requests.
Opaque values received 0.34 in request 1; checked construction received 0.21 in
request 2. The replies do not establish that opacity or universal validation is
inferior.

**Engineering conclusion:** an owner-record prototype is a reasonable bounded
candidate, but the unexecuted exported-email counterexample should be built
first. Public type visibility must be separable from importer construction and
copy-update. Owner functions remain trusted to validate every public minting
route. Selected public projections and identity-only matching need explicit
rules, as do owner-defined equality and normalized values.

The decoder and fixture routes are part of the boundary, not later integration
work. Current generic document projection directly manufactures a nominal
record. Under this candidate, direct derived decoding of the protected type
must fail; decode a public wire representation and call the owner factory.
The consulted candidate also conservatively rejects direct derived schema of
the protected type, leaving wire-schema derivation available. Encoding uses an
owner projection. Supplied assertion completions must acquire owner-created
values through validated constants or factories; neither fixtures nor hidden
field patterns may fabricate or reconstruct the representation. Test all four
invalid creation paths: importer constructor, `with`, decode and fixture supply.
Frozen storage alone does not close any of those admission gaps.

Full opacity is a viable alternative when observations should also be owned;
universally checked construction is viable when public representation is
desirable but every creation route may fail through one declared validator.
The latter needs finite failures, normalization and validator recursion/effect
rules before it is a complete trial. Current transparent records plus careful
factories/revalidation remain a valid application baseline, but fail the
stronger claim that every inhabitant is owner-validated. A valid tenant ID must
still fail another tenant's protected invoice operation.

## DI-05a: both lookup candidates can pass; author renaming cannot count

**Disagreement investigation:** the first request divides probability between
manifest handles (0.50) and qualified imports (0.35); the other two favor
qualified imports. The evidence does not measure manifest-context cost versus
source qualification. Re-reading [package loading](../../../../compiler/internal/project/graph.go#L273)
and [import resolution](../../../../compiler/internal/resolve/symbols.go#L213)
confirms that the global short-name map rejects the libraries before file-local
aliases can help. Both candidate lookup contracts can remove that bottleneck;
the source evidence does not decide where authors should express the binding.

**Engineering conclusion:** compare qualified source imports and project-local
manifest handles on the same two unchanged `model` dependencies, alias edits,
direct/transitive visibility and relocated checkout. Keep dependency-relative
lookup, separate loaded-instance identity, reserved catalogue names, canonical
`internal` checks, locks and source maps in the test contract. The existing
loader also rejects one dependency key resolving to different directories;
support for independently loaded instances needs explicit graph rules rather
than an assumption that existing qualified strings already solve it.

The current globally unique short-name discipline fails the hard independent
composition case. Author-qualified declarations describe a coherent different
regime, but rewriting either bare-name dependency fails the **unchanged-library**
acceptance case. That alternative must not receive a passing score merely
because two rewritten declarations now differ. `defer_measure` is an evidence
step, not satisfying semantics. Jev's plurality does not choose import syntax.

## DI-05b: audit requirements before treating a report key as a design

**Disagreement investigation:** request 2 is nearly split between finishing the
audit (0.50) and scoped numbers (0.45); request 3 assigns 0.40 to scoped numbers,
0.28 to audit first and 0.27 to textual keys. The state consistently says that
external numeric and cross-release rename requirements are unknown. Scope is
a concrete way to fix collisions; audit-first avoids inventing requirements.
Neither proposition determines the other. Re-reading the
[checker](../../../../compiler/internal/check/errors.go#L25) and
[runtime admission](../../../../runtime/domain.ts#L69) confirms multiple numeric
consumers; these reads add no evidence for outside reporting requirements.

**Engineering conclusion:** complete the consumer/rename audit and establish
two-build reporting cases, then compare scoped numbers and canonical textual
keys; retain generated build-local numbers as a viable mapped-report option.
Do not decide by the absence of backward compatibility obligations. Nominal
matching must continue to distinguish declaration plus concrete arguments,
independently of any shared declaration-level report code.

Current global active/retired allocation fails independent duplicate-number
composition. Scoped numbers can retain unchanged authored source/registry
numbers while changing every consumer to understand owner scope. Textual keys
and generated codes are viable replacement contracts **after** a deliberate
source/registry migration; that migrated-input experiment must not be called
success on an unchanged-input test. Each new-contract composition test must
still allow independent declarations without globally coordinated allocation.
Include active/active and active/foreign-retired overlap, triggered distinct
errors, generic instances, and old report interpretation after graph growth.
Generated integers need an archived build identity and collision-free mapping;
a bare hash is not such a guarantee. The audit-first choice itself fixes no
collision.

## DI-06: the three rankings optimize different coverage contracts

**Disagreement investigation:** all three leading labels differ and confidence
is low. The exact [runtime selector](../../../../runtime/assert/fixtures.ts#L29)
uses the root short name and falls through to real execution when no row is
selected; the saved rename failure therefore has a direct mechanism. Full
queue identity is separate and is not the demonstrated defect. Rewording keeps
these facts and all alternatives, but emphasis on bounded scope versus
integrated coverage can change which experiment the classifier prefers. Its
responses contain no rationale, so that explanation is a hypothesis about the
decision tension, not a claim to know Jev's reasoning.

**Engineering conclusion:** first compare the best local-template/stub and
ordinary injected-callable baseline with the required integrated path. If the
acceptance case requires a caller to deliberately select simulated helper-body
behavior and preserve it through caller-label rename, exported helper scenarios
and typed exported seams are the viable linkage candidates. Scenarios hide
internal call structure behind a helper-owned configuration; seams expose
approved targets for finer caller control. Which burden is smaller remains
unmeasured.

Lexical-only fixtures can make a rename behaviorally stable by forbidding
foreign-root activation, and whole-helper stubs can preserve a supplied outer
completion. Those are coherent restricted contracts. They **do not pass the
stronger deliberately linked cross-package helper-body fixture requirement**
unless ordinary dependency injection separately supplies and tests that path.
Separate unit coverage must not be credited as equivalent integration coverage.
The unchanged ambient-label scheme fails the demonstrated rename requirement;
label qualification alone does not make activation explicit.

For any linkage trial, disappearing or changed requested scenarios/seams must
diagnose rather than invoke real code. Include two helper occurrences, nesting,
recursion, concurrent participants, duplicate display labels, unused links and
exact argument checks. Retain per-root/table/invocation FIFO allocation and
real/supplied/raw-provider evidence labels. No result justifies weakening those
guarantees or installing an unrestricted override registry.

## DI-07: transitive set inclusion versus documented path dependence

**Disagreement investigation:** requests 1 and 3 favor extensional leaves;
request 2 favors executing the complete baseline matrix first. This is mainly
an experiment-order disagreement. The repeated source check of
[`Assignable`](../../../../compiler/internal/types/compatibility.go#L9)
confirms the direct-specialization guard preceding leaf inclusion, so the core
direct/bridge contrast is established; broader option/codec and agent-edit
costs are still unmeasured. No request infers a memory defect or a nominal-brand
requirement.

**Engineering conclusion:** record and execute the complete matrix, then
compare an extensional compatibility experiment with today's documented rule.
For variants treated as specialized nominal leaf sets, inclusion is transitive:
if A's leaves are included in B and B's in C, A's are included in C. Equal
phantom sets therefore admit `tagged<int> -> tagged<str>` directly as well as
through `bridge` or `unit`. This removes the direct-only exception; it does not
make `some<int>` equal to `some<str>` or merge nominal generic records. A shared
`none` is a leaf of both option specializations, not permission to assign all
of `option<int>` to `option<str>`.

Retaining today's two-stage rule is coherent only as explicitly path-sensitive
admission. The direct step fails while both bridge steps pass, so it cannot
promise a transitive subtype relation or durable phantom branding. Better
documentation may help agents predict it, but does not remove refactor-sensitive
admission. If transitivity is made a hard requirement, documentation alone
fails that requirement; current authority has not yet adopted that requirement.

Nominal variant wrappers are a separate viable semantics when a demonstrated
brand requirement warrants them. Explicit injection, controlled unwrapping and
exhaustive conversion replace implicit cross-variant compatibility; deliberate
export-and-reinject remains a visible conversion. That trial needs wrapper
identity recovery in codecs and allocation/adapter measurements against today's
ordinary nominal record wrapper baseline. A flattened representation cannot
silently acquire an unbypassable wrapper guarantee through diagnostics.

## Resulting preparation work

The saved consultations support bounded comparisons, not a winning replacement
set. Build the missing owner-value bypass cases, the complete independent graph
and reporting cases, an honestly labeled fixture integration comparison, and
the full variant matrix. Register held-out creation/refactor/repair tasks and
whole-successful-task token accounting before claims of agent benefit. User
syntax decisions and canonical specification edits remain later work.

# Can: zero-compatibility syntax and ABI audit

Status: **audit and redesign recommendation, not an approved language spec**.

**2026-09-19 follow-up:** [Workstream A replacement audit](audit-probes/workstream-a/README.md)
reproduces F01–F05 at `8bbcf13`, adds F06 argument-order evidence, and records
all thirteen replacement JEV judgments. Its dispositions supersede use of the
withdrawn reviews: three bounded process/correctness recommendations accepted,
ten redesign choices unresolved. The proposals below remain unapproved.

Baseline: `8312d85`, including B11. No production compiler or language behavior
was changed during this audit. Existing design documents remain history, not
vetoes on the proposals below. B05 remains the async design identifier.

## Executive verdict

**Yes: simplify the language before adding more compatibility-shaped features.**
There are zero external users. Maintaining old source spellings, generated TS
layouts, or old goldens is not a product requirement. Migration work is real,
but it is implementation cost—not evidence that the current design is good.

Can's valuable identity is **explicit authority, exact values, declared failures,
exhaustive decisions, executable examples, and honestly scoped proofs**. Its
identity is not flattened record returns, globally prefixed identifiers,
file-location-dependent mocks, or restrictions inherited from string parsers.

B11 is a concrete example: `Ok<T>(x)` repaired expressivity by adding another
form around the old return convention. A clean design should make that form
unnecessary. I would replace it, not preserve my recent implementation merely
because it now exists.

The highest-value sequence is:

1. Fix confirmed identity, acceptance-evidence, and generated-code defects.
2. Establish a compositional type representation and typed core, plus explicit
   separation of source semantics, private execution ABI, public host ABI, and
   wire formats.
3. Make every success one value; unify call/binding/pattern semantics.
4. Introduce real module namespaces and location-independent pure-source tests.
5. Add finite error-set abstraction and complete higher-order typing in bounded
   steps, without weakening authority or termination guarantees.
6. Replace trusted-by-accident TS interop with an explicit embedding boundary;
   resolve the competing B05 backend proposals before fixing async linkage.
7. Migrate the stdlib and polish syntax with a real formatter.

This is not a recommendation for one giant rewrite. Each step should end with
one canonical supported form, a migrated repository, and green semantic gates.
There is no need for a permanent old/new compatibility mode.

## 1. Scope and evidence

Reviewed: current requirements and idioms; B00–B11 design/implementation context;
parser/AST, generic expansion, type/containment rules, call classification,
forwarding/chains, evaluator, TS emission, revision/acceptance machinery,
verification admission/composition, representative stdlib and host sources,
and both B05 proposals. Source references below name files and functions to
survive ordinary line-number drift.

This is a whole-language **architecture and design audit**, not a proof that
all checker paths or all host implementations are sound. It does not include
agent usability experiments, a runtime benchmark campaign, every historical
proposal, or a formal proof of compiler correctness. Those are explicit gates,
not claims hidden behind the word “audit.”

Baseline checks ran successfully:

- `go test -count=1 ./...`
- `go run ./tools/modcheck`: 65 current modules
- `go run ./tools/gramcheck`

Broader integration checks **failed**: compiling all stdlib sources as one world,
the repository-wide strict TS project, and strict TS on freshly emitted JSON
with its provider beside it. F04/F05 distinguish the causes. A green baseline
suite is not a claim that every emitted artifact passes strict TS.

Lexical corpus inventory, excluding Go-embedded fixtures:

| Corpus | `.can` files | Lines | Functions | Records | Variants | Externs | `match call` sites | `match invoke` sites |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| std | 13 | 7,483 | 401 | 108 | 5 | 7 | 486 | 14 |
| sketches, including negative examples | 52 | 2,411 | 128 | 64 | 6 | 12 | 73 | 17 |

65 of the 108 std record declarations have one field. **Not all are disposable**:
`Clock__Instant` and `Clock__MonotonicTick`, for example, carry useful domain
meaning. `Int__Value` and mechanically required result carriers are a different
category. Do not replace a semantic type just because it has one field.

Reproducible new probes are in `audit-probes/`. Their findings are described
below; they are audit probes, not permanent tests asserting that bugs should
continue to exist.

## 2. Findings requiring attention before the redesign

### F01 — Pinned callback expectations can change without the warning

**Confirmed defect; fix first.**

`compiler/revision.go:canonSmall` has no `fnref` case. Different references
canonicalize to `unknown-kind(fnref)`. That fallback is described as fail-closed
in a comment, but returning the same string is not rejection.

The probe builds two clean programs. The first factory's pinned expectation
and body return `fnref acc__a()`. The second returns `fnref acc__b()`, where the
target has different behavior. `CheckPinnedRows` reports no warning:

```text
old = one() => ctor(Ok)[value=unknown-kind(fnref)]
new = one() => ctor(Ok)[value=unknown-kind(fnref)]
warnings = []
```

Related canonicalization probes also show:

- `Ok<int>(value)` and `Ok<str>(value)` have equal `canonSmall` output;
- typed Ok patterns omit their type argument in `canonPattern`;
- changing `Node.InvokeArg` does not change `canonNode`.

The first item is an end-to-end acceptance-warning failure. The other three
are structural omissions; **this audit does not claim an exploited proof-cache
bypass**. Bodies are deliberately excluded from interface identity, and not
every canonicalizer currently feeds a persisted body certificate.

**Recommendation:** an exhaustive canonical typed-core encoding, with separate
interface/executable/evidence identities. Unknown nodes must return an error,
not a hashable placeholder. Include targets, revisions, captures, argument
expressions, and semantically relevant type arguments. Version the format and
refuse incomplete old evidence; do not silently regenerate an accepted baseline
from the candidate being reviewed.

**Gate:** changing a pinned callable target/capture must produce CAN6017;
changing an invocation argument must change executable identity; irrelevant
formatting must not. Keep interface identity separate from implementation churn.

### F02 — Duplicate record identities silently depend on source order

**Confirmed identity defect; fix first.**

`compiler/check.go:buildWorld` rejects duplicate functions, variants, and
constants, but uses first-wins handling for other declaration names.
`recordShapes` also selects the first record.

Two modules declaring `Shared__Record`, one with `value: int`, the other with
`value: str`, produce no error in `checkProgram`. Reordering their source
placement changes the resolved shape:

```text
diagnostics=[] shape=[[value int]]
diagnostics=[] shape=[[value str]]
```

Editor convenience for independent demos is not an acceptable identity rule
for a linked program. Identical spelling does not establish identical ownership.

**Recommendation:** immediately reject ambiguous identities in a linked world;
ultimately key symbols by package/module/declaration identity. Distinct scoped
records may share a short name. Reuse a shared error/type by import, not by
redeclaring a lookalike. Inspect the corresponding brand/error collision paths
in the repair; this probe established record behavior specifically.

**Gate:** source order cannot alter resolved types, owners, revisions, or host
bindings. CLI and editor must diagnose the same linked world.

### F03 — The public TS boundary admits mutable aliases and erased distinctions

**Confirmed behavior and design gap, not an exploit of a promised untrusted-host
sandbox.** The current design explicitly trusts hosts.

`compiler/emit.go:tsTypeB` emits `Seq<T>` as mutable arrays, Bytes as
`Uint8Array`, brands as their underlying string, and nominal records as
structural TS aliases. Exported generated functions are directly callable.

A Node probe against the B11 sketch:

```text
input = [1n]
output = success__id<Seq<int>>(input)
input[0] = 9n
output.value === input  -> true
output.value[0]         -> 9n
```

Record getters also execute when the emitted function reads fields. TS types
alone cannot establish immutable, side-effect-free Can data. Canonical decimals
are ordinary TS strings, so their representation invariant is not encoded by
`string` either. A further Node probe passes an ordinary effectful JS callback
into the exported optional mapper; it executes (`HOST_CALLBACK_EFFECTS 1`).
Source-level rejection of function-bearing extern signatures does not make
native JS function parameters into certified-pure Can callables.

**Recommendation:** specify which entry points accept trusted internal values
and which accept embedding/host values. Public ingress validates and acquires
ownership; exported TS types use readonly/opaque views. An explicitly trusted
zero-copy path may exist, but it must not be the accidental default contract.

**Gate:** adversarial alias/getter/prototype/cycle/invalid-decimal/tag tests,
including mutation after a host result completes. See §7 for the ownership
policy; neither a TS cast nor `Object.freeze` is an adequate universal solution.

### F04 — The stdlib does not compose as one checked world

**Confirmed integration failure, related to error ownership.**

Compiling all 13 stdlib sources together fails:

```text
scalars.can:1016: math.nonterminating_decimal field numerator: got dec, want int
```

`std/ratio/ratio.can` declares that error with `(numerator: int, denominator:
int)`; `std/scalars/scalars.can` declares the same identity with `(dividend: dec,
divisor: dec)`. Both cannot be the one global contract. The reported failure
occurs at a scalar test payload rather than at an explicit conflicting-identity
boundary.

**Recommendation:** choose one owned shared error schema or give genuinely
different errors distinct scoped identities. Add a whole-stdlib composition
check, not just individually green modules. Module namespacing and explicit
imports are functional requirements, not separator fashion.

**Gate:** one deterministic checked stdlib world and ordinary consumer subsets;
conflicting error payload declarations fail at their definitions. Never silently
coalesce payloads or weaken them to an untyped universal error.

### F05 — Passing Can tests does not currently guarantee strict-TS output

**Confirmed emitter/integration defects.**

The repository TS project fails on missing relative provider artifacts and
JSON typing. To separate packaging from generated-code correctness, I compiled
`std/json/json.can` together with `std/scalars/scalars.can` into a fresh flat
output directory. All **785 Can tests passed**, but strict TS on that fresh
output still failed:

- TS2440: imported `Bool__Value`, `Dec__Value`, `Int__Value`, and `Str__Value`
  conflict with local declarations;
- TS2322: nested variant discriminants in constructed sequence elements widen
  to `string` instead of the declared literal-tag union (first reported at
  generated `json.ts:308`, with additional sites).

The former reinforces F02's ownership problem. The latter is a concrete
code-generation typing issue, not a reason to loosen the declared variant type.
The global project's missing `./host`, `./quota`, and `./scalars` files are also
an artifact-layout/package gate problem; they should not obscure fresh-output
emitter failures.

**Recommendation:** strict-check complete freshly emitted dependency bundles
as a required compiler acceptance gate. Preserve contextual/literal types in
lowering; do not fix by inserting `any`, widening discriminants, or disabling
strictness. Distinguish checked source, tested source, emitted TS, and
strict-checked/executed target in build reports.

**Gate:** full stdlib and representative consumer bundles compile, strict-check,
and execute from their generated output directory without manually invented
imports. Goldens must be both intentional and valid, not merely unchanged.

## 3. Source semantics and syntax

### S01 — One success value, everywhere

**Replace the dual convention.**

Current: `compiler/result.go`, `success_value.go`, `types.go:checkCtor`, and
`emit.go` distinguish flattened records, one-value successes, and B11's explicit
whole-value path. Contracts still use envelope binders while typed body binders
can name T directly.

Recommended canonical form:

```can
Ok(value)
on Ok(result) => ...
```

`value` and `result` have the declared success type T. A record is one value:
`Ok(Pair(left, right))`. This applies to bodies, contracts, test expectations,
scripts, ordinary calls, and callbacks. Add a genuine Unit type/value for
no-payload success rather than proliferating named empty result records.

Retire multi-field Ok construction, success auto-flattening, `Ok<T>`, and
binder `.value` used only to unwrap an outcome. A data record's actual field
named `value` remains a field. Do not retain old spelling as overloads.

**Gate:** one matrix across every supported T and every producer/consumer
boundary, including empty records and records whose field is named `value`.
Migration must use resolved types, not global text replacement of `.value`.

### S02 — One application model, not named-call versus callback grammars

**Unify invocation; retain an explicit `call` as the working recommendation.**

Current: `parseCallHead` versus `parseInvokeHead`, `MatchCall` versus
`MatchInvoke`, and `invoke.go:resolveInvokeSites`. B11 can bind a Fn from a
factory but cannot directly invoke that new binder; it must pass it to another
function whose parameter is an admitted invocation head.

Recommended: `call target(args)` accepts a resolved named function or a typed
callable expression, including a field. Typing determines the signature;
lexical storage location does not. Keep argument evaluation order explicit and
single-evaluation. A unified AST still records static versus indirect targets
for authority, cycle analysis, tests, and emission.

The keyword itself is a lower-confidence taste choice. Unifying semantics is
more important than choosing `call f(x)` versus `f(x)`.

**Gate:** identical signatures produce identical application behavior for named,
parameter, field, and returned callables. No relaxation of preconditions,
capture authority, latent effects, or indirect-cycle checks is implied.

### S03 — Add immutable local binding and stop encoding sequencing as ladders

**Add expression blocks with immutable bindings.**

A function body remains an expression; a block has bindings followed by its
result expression. For an error-free operation, an irrefutable pattern is enough:

```can
let Ok(mapped) = call transform(input)
Ok(Option.Some(mapped))
```

That binding is allowed only when the call's declared error set is empty. A
fallible operation still needs exhaustive dispatch, or explicit transfer of a
completed outcome once that feature is specified. Do not sneak in implicit
`try`, early returns, or ignored failures.

This removes the reason for many total-call `match` ladders and capture-only
helpers. Current `match chain` and `forward call` are signs of the missing
compositional core, not features that must be immortalized. The latter is
late-parsed text and still fails generic stamping (`forwardcall.go`).

Ultimately, a whole-outcome tail call returns that outcome; error-preserving
forwarding need not have its own special syntax. Keep shared failure mapping
explicit rather than making a chain silently select or discard an error.

**Gate:** nonempty error sets reject irrefutable Ok bindings; argument and call
order, error payloads, state traces, and evidence attribution remain correct.

### S04 — One pattern language

**Unify constructor patterns, success patterns, and data patterns.**

Use constructor-shaped patterns with payload binding, e.g. `on Ok(x)` and
`on Option.Some(x)`. Permit record/tuple destructuring and ordinary wildcard,
constant, range, and or-patterns where their domains justify them. Use one
canonical arm spelling instead of different binding conventions for cases and
outcomes. Full-record binding still needs a way to retain nominal T.

Separate the pattern's meaning from its source name. Today `qualifyCase`
turns both `M__First.None` and `M__Second.None` into `M__None`; generic suffixes
then get appended to that flattened name. Case identity must include its parent
sum type, not just its domain prefix.

Preserve exhaustiveness, overlap diagnostics, first-match priority, and declared
finite failure handling. A generic error-preserving relay is not permission for
an untyped catch-all recovery. Keep eager multi-scrutinee evaluation explicit.

**Gate:** same-named cases in different parents coexist; nested patterns are
checked structurally; errors added to a closed handled set cannot disappear
behind an accidental catch-all.

### S05 — Explicit at boundaries; context where it is deterministic

**Replace “no inference” as a blanket slogan with a precise rule.**

Require public signatures, error/effect authority, and annotations for ambiguous
locals. Allow bidirectional checking where the expected type and lexical
bindings uniquely determine the answer. Constructor/pattern payloads should
not restate a generic parameter that is already known.

Never infer a signature, error set, effect capability, or generic law from
observed test rows. No implicit numeric conversions or structural coercion of
nominal types. No overload search with several plausible answers.

**Gate:** changing a local annotation from implicit to its uniquely resolved
explicit type is semantics-preserving; genuinely ambiguous empty collections,
polymorphic references, and applications require annotations.

### S06 — Keep a small surface; remove arbitrary lexical constraints

**Recommended changes, after semantic priorities:**

- Remove the blanket brace ban, especially in comments. It contributes no
  outcome/authority guarantee. Keeping indentation for blocks does not require
  rejecting punctuation in comments.
- Keep `name: Type` for declarations and `name = value` for named arguments.
  These express different things; visual uniformity is not a reason to merge them.
- Preserve useful argument labels. Stop error-linting a label away merely because
  it appears in declaration order. Use one binding rule everywhere.
- Prefer ordinary exact decimal tokens (`12.50` has type dec) and ordinary
  escaped strings with an explicit raw-string form. There is no binary float
  type whose inference must win. Keep exact canonical values; never normalize
  Unicode or round numerics implicitly. This is a preference to test, not proof
  that prefix literals are objectively wrong.
- Keep `Seq<T>` as a regular type application, alongside `Map<K,V>` and
  `Option<T>`. The important fix is compositional nesting, not replacing angles
  with another special collection grammar. B01's “dated spelling” argument is
  not evidence of a language problem.
- A `record` keyword could clarify the current `type`/`variant` distinction,
  but keyword renaming is low priority. Do not call it a semantic improvement.

**Boolean decision:** keep strict/eager `and`/`or` as the working recommendation
for the explicit-table language; require guarded `match` where evaluation can
fault. This is a modest-confidence preference, not a compatibility veto. If
agent trials favor short-circuit operators, model them as guarded control flow
and specify their evidence obligations before changing semantics. Short-circuit
operators are not the same as lazy function arguments. Avoid offering two
near-identical operator families merely to avoid making the decision.

**Gate:** a real formatter and grammar round trips, plus agent edit/error-rate
trials. Shorter files alone do not establish improved readability or correctness.

## 4. Types, generics, errors, and functions

### T01 — Replace type strings with a compositional type algebra

**Foundational compiler work with direct language benefits.**

`FnDecl.Ret`, fields, `fnTypeShape`, `sameType`, `seqElemName`, and much of
`expand.go` operate on type strings. Nesting exclusions and special rewriting
paths leak that representation into the source language.

Use typed nodes for primitive, nominal application, type parameter, sequence,
function, completed outcome, finite error row, and effect row. Resolve nominal
symbols to identities, not display strings. Give generic parameters kinds so a
data type, error set, and effect set cannot be confused.

Then admit nested generic **data** forms uniformly, including sequences of
variants and nested sequences, subject to real representation and containment
checks. Derive explicit constraints for operations such as equality/ordering;
per-instance success on a few examples is not a universal generic theorem.

**Gate:** nested applications, substitutions, cross-module identities, recursive
references, and signature equality all use one implementation. Unsupported
operations fail at the declared constraint, not deep in emitted code.

### T02 — First-class completed outcomes and finite error-set algebra

**The real stdlib unblocker after uniform successes.**

Introduce `Outcome<T,E>` for completed values, with finite declared error rows,
union/deduplication, and total mappings. Distinguish an error value from an
outcome carrying it. A coherent candidate is `Ok(T)` / `Err(E)`; returned errors
are values, not JS exceptions.

But this requires more than syntax. Specify what counts as handling,
preserving, transferring as data, or explicitly discarding an outcome. An
unused-variable warning is not sufficient to preserve today's no-silent-loss
claim when generic records can carry failures. The error catalogue must record
these distinctions. A generic identity relay can preserve any E; generic
recovery needs total handler coverage, not a universal unknown-error bucket.

Keep the success type in `-> T` plus one canonical error declaration in function
signatures; do not require both `-> Outcome<T,E>` and `emits E` for the same
function result. Explicitly returning an Outcome *as successful data* is a
separate nested type, not automatic flattening.

**Gate:** real generic `map`, `and_then`, `map_error`, recovery, zip and collect;
no loss of simultaneous failures or payloads; new error kinds trigger the
right handler/identity checks. Decide error-row variance explicitly rather than
silently loosening `sameFnErrs`.

### T03 — Ordinary higher-order source functions, with analyzable authority

**Replace positional bans with capabilities and latent contracts in stages.**

The current unary `Fn<A,R,[...]>`, named captures, data-only head/capture rules,
and no-required-preconditions admission are a bounded first implementation.
`types.go:captureDataKind` even distinguishes capture syntax kinds rather than
accepting every proven-pure data expression.

First unify application of already-admitted values. Then support source
callbacks in inputs/results/captures and general callable parameter lists,
with explicit capture sets, latent errors/effects, preconditions where admitted,
and sound indirect-call/termination analysis. Tuple or parameter-list argument
representation must be shared by declarations, Fn types, and application.

Do not add arbitrary host callbacks or anonymous effectful lambdas merely to
look conventional. Explicitly captured anonymous *pure* functions can be a
later convenience over the same core. A returned source callback is not the
same trust problem as a host callback.

**Gate:** cycle attempts, capability-carrying captures, preconditioned targets,
stateful/extern reachability, shadowing, escaping values, and target-revision
changes are tested across every newly admitted position.

### T04 — Preserve nominal meaning; stop using carriers as type-system patches

Keep nominal records, variants, and authority-bearing opaque types. Remove
wrapper records whose only purpose was the old return restriction. Preserve
semantic distinctions such as different clocks/units even if representations
match. Representation similarity is not identity.

Generalize opaque/brand representations beyond str when a real type requires
it—e.g. numeric identifiers or validated records—with module-owned smart
construction. Do not equate a brand tag with a verified predicate. A constructor
must establish whatever validation/authority the API claims.

Recursive algebraic data should be admitted where values are finite and the
recursion is well founded/guarded by containers, not through arbitrary cyclic
object graphs. This would allow a natural JSON tree instead of making every
application encode one through a bespoke frame machine. Traversal termination,
equality, host-cycle rejection, and resource costs still need separate checks.

**Gate:** compile a typed JSON AST and validated-record constructor without
hand-forging authority, infinite-size products, or recursive equality crashes.

## 5. Modules, errors, effects, and revisions

### M01 — Real namespaces and visibility

Use local short names and explicit qualified imports (`users.load`,
`Option.Some`). Introduce a genuine public/private distinction. `provides`
currently duplicates every function/type declaration, including helpers;
`emitModule` exports every source function. Generated implementation helpers
should not become a public API accidentally.

Resolve names through a module/package graph with deterministic identity, not
repository-wide search and global double-underscore conventions. Reject
ambiguous imports. Give errors one owner and import their declarations; retain
kernel identities separately from user symbols.

**Gate:** unrelated modules can both define `Value`, `Some`, and `parse`; moving
or reordering files does not change identity, seals, dependency ownership, or
emitted host resolution. Private helpers cannot be called from other modules.

### M02 — Delete unchecked duplicated authority lists

Module-header `emits` is not a checked union in the current compiler. An
unchecked list that looks like a contract is worse than a generated summary.
Remove it from authored source or make it a deliberately enforced export-level
contract. I recommend generating the module summary from checked declarations.

Likewise, derive the export list from explicit public declarations instead of
requiring parallel `provides` maintenance. Keep explicit imports, public error
contracts, and capabilities where those delimit real authority.

**Gate:** an authored formal line is either enforced, explicitly informational,
or absent. Diagnostics and summaries agree about the authority actually admitted.

### M03 — Effects are independent of failure and file location

An error-free extern can still observe time, randomness, network, or secrets.
`ExternDecl` currently has no effect field; current effect lists model Can cells,
not a general host footprint. Fn purity is guarded by separate graph rejection.

Add explicit host/capability footprints and carry them transitively through
source calls and callable types. Model pure deterministic kernels distinctly
from external observations, without bespoke source-level calling grammars for
every codec. A declaration of no errors does not mean purity.

Keep specialized brand-disclosure and asset-bridge authorization until a
replacement capability rule reproduces their guarantees. Do not replace them
with a general-purpose “cast to Bytes.”

**Gate:** zero-error impure operations cannot enter pure callbacks or be
reordered/eliminated as pure terms; lack of an annotation never certifies host
isolation. Capability grant ownership survives module renaming and ABI changes.

### M04 — Identity is essential; manual revisions on every helper are not

Retain distinct interface, executable, dependency, and acceptance identities.
Move exact artifact selection into a build/package manifest with immutable
content identities. Public API release/version policy can be explicit without
requiring a manual `rev 1` on every private helper and compiler-generated stamp.

Do not confuse removing source boilerplate with allowing same-interface contract
drift. Accepted interface baselines must still detect changed parameter types,
errors, effects, preconditions, or postconditions.

`REQUIREMENTS.md` promises that provider bumps never break pinned callers, but
the inspected resolver checks the supplied world's revision; that is not a
historical artifact store or multi-version linker. State the narrower current
guarantee. Add coexistence/version resolution only if implementing its semantics.

**Gate:** body edits invalidate executable evidence without pretending to be API
changes; API edits require accepted interface change; wrong/mixed artifacts
fail deterministically. No candidate file can appoint itself the accepted base.

## 6. Tests, contracts, control flow, and runtime faults

### E01 — Execute checked pure source independently of file location

Current `classifyCallee` separates same-file local from foreign source. Ordinary
tests execute the former and script the latter. `linked.go` and Fn invocation
already execute checked pure providers through another path.

Make checked pure-source execution the normal semantics on both sides of a
module boundary. Keep explicit mocks for deliberate contract-boundary tests;
script actual observations/externs. A refactor that moves a pure helper to a
module should not change what its callers' tests mean.

Keep request/response pairing, no missing/leftover exchanges, deterministic
fresh state, and transparent evidence provenance. Name effectful sites so scripts
and traces have stable identities rather than depending only on anonymous AST
position. Separate unit/mock evidence from integration execution and proof.

**Gate:** move a pure helper between files and obtain identical results and
source-arm obligations. Host operations never execute accidentally at compile
time. Integration runs do not manufacture local branch witnesses.

### E02 — Keep exhaustive decisions; distinguish proof from reachability examples

Tests are valuable executable examples, not “complete specifications” in the
universal sense. Continue distinguishing executed, certified, and uncovered.
Maintain hard witness requirements for admitted source branches unless an
explicitly authorized certificate discharges the obligation.

Attach obligations to stable source-level branch identities through lowering.
Do not force users to author extra tests only because a backend introduced
administrative dispatch, and do not erase a real user branch through optimization
before its evidence is recorded. Pinned acceptance should remain reviewer-owned;
warn-versus-error policy can be an explicit build/review mode, not syntax sugar.

**Gate:** equivalent lowering changes do not invent or remove source evidence.
False contracts and changed pinned expectations still fail or warn exactly as
the selected policy specifies.

### E03 — Termination proof should not dictate one spelling of an algorithm

Current recursion accepts named schemas and exact guard/step shapes in
`checkRecursion`. This is a legitimate bounded proof implementation, not the
ideal final syntax of all programs. Multi-scrutinee guards are not recognized
like their single-scrutinee equivalents.

Keep termination obligations. Add structural descent and well-founded measures
as the proof engine supports them; provide finite traversal constructs or
well-specified collection primitives rather than manual fuel plumbing everywhere.
No general unproved while loop is implied.

Lower tail recursion to loops/frames. `evApply` has a 1,024 depth backstop and the
TS emitter uses ordinary calls. Mathematical termination does not establish that
a valid long traversal fits either stack. Repeated immutable array append may
also be quadratic; benchmark instead of treating a passing small row as a cost
argument. A private builder optimization must not leak mutation into source.

**Gate:** large terminating inputs preserve results across evaluator and target;
resource exhaustion is reported honestly; new measures reject nondecreasing
cycles. Measure allocation/stack/time, not just emitted line count.

### E04 — Distinguish language outcomes from runtime-contract faults

Source has no exceptions, but emitted helpers throw for bounds faults and
unreachable dispatch; hosts and runtime resource limits can fail too.
`emit.go:strHelpers`, `seqHelpers`, and match defaults make this visible.

Keep declared errors in the ordinary outcome protocol. Put invalid host values,
broken runtime contracts, and resource failures in a separately specified
embedding-fault channel. Do not claim “every call returns a declared outcome”
without the checked-input/platform assumptions.

Provide explicit checked access (`Option` or a declared bounds error) and allow
unchecked indexing only with an admitted range proof. Define malformed Unicode,
canonical dec requirements, bigint bounds at platform conversion points, and
state/world isolation at ingress.

**Gate:** bad indices, wrong tags, host throws/rejections, invalid bytes/decimals,
and resource limits cannot be mislabeled as successful or ordinary undeclared
Can errors. Expected domain failures retain exact identities and payloads.

### E05 — Contracts and examples need one semantic model

`verify_admission.go` admits only a bounded int/bool/record fragment and excludes
several language features; source functions may be checked/tested without being
universally verified. Keep that distinction in interfaces and build output.

The uniform success/pattern semantics must also apply to ensures binders, not
leave the old envelope convention behind as another exception. Expand proof
support separately from syntactic admission. Do not silently omit unsupported
clauses, treat rows as proofs, or allow TS types to stand in for a solver theorem.

**Gate:** contract/body/evaluator/target agree on the same typed core. Every
unsupported proof feature is an explicit refusal or stated unverified status,
never a fabricated proof. Re-run all false-contract tests after ABI lowering.

## 7. Four separate contracts, not one thing called “the ABI”

### 7.1 Source value and outcome semantics

Nominal types, exact numerics, immutable values, completed outcomes, declared
errors/effects, and evaluation order. These are independent of JavaScript object
layout. The source must not know generated `$T$` identifiers or scheduler frames.

### 7.2 Private execution ABI

Compiler-controlled calls, closures, tags, stack/control frames, and specialization
symbols. Optimize these as a whole checked artifact. Internal record flattening
could later be an optimization, but never again change what `Ok(x)` means.

Require evaluator/target differential tests. Do not freeze every temporary or
validate already-checked internal values merely because public ingress validates.
The runtime cost must be measured, and internal optimization must preserve the
public/evidence contracts.

### 7.3 Public host/embedding ABI

Recommend a generated, versioned facade rather than exporting every implementation
function. A coherent completed-result candidate is:

```ts
type Outcome<T, E> =
  | { readonly $can_kind: "ok"; readonly value: T }
  | { readonly $can_kind: "err"; readonly error: E };
```

E is a closed union of nominal, tagged error values, preserving every declared
payload. Decide its exact tag/payload layout explicitly in the ABI spec. Closed
variant tags must identify the owning type/case, not inherit the compiler's
specialization name as an accidental protocol.

The facade specifies:

- explicit public exports and supported closed generic instantiations;
- schema/type/case identity, error/effect contracts, compiler/ABI format, and
  exact host binding identity in a machine-readable manifest;
- readonly record/sequence surfaces and nominal phantom/opaque TS distinctions;
- world-issued checked callable handles where a public signature carries Fn,
  rather than accepting any JS function as evidence of purity/termination;
- validation of bigint, canonical decimal strings, Unicode strings, object
  shapes, variants, errors, byte carriers, and recursive value structure;
- ownership on ingress and egress, plus a separate embedding-fault channel;
- a world instance for stateful execution rather than unaccounted global
  singleton state and arbitrary reentry.

A callback's purity, preconditions, and termination cannot be validated by
checking `typeof x === "function"`. Accept provenance-bearing source callables,
or a separately declared trusted host-callback contract—not an accidental
exception to the source containment rules.

Readonly TS is not runtime ownership. `Readonly<Uint8Array>` does not neutralize
mutating methods or shared buffers; freezing typed arrays is not a general
solution. Use copy/transfer/opaque owned views as specified for the boundary.
Reject accessors/cyclic or exotic objects where plain immutable data is required;
validation itself must not accidentally invoke arbitrary getters.

Do not auto-serialize or disclose opaque/secret brands. Validation of shape is
not authorization to create, declassify, or disclose a value.

**Trust tradeoff:** this is the recommended external boundary, not a claim that
all current trusted-host behavior is a compiler security bug. An explicitly
trusted internal ABI can remain cheaper. JEV's vote on this boundary was almost
a tie; the recommendation rests on the chosen ownership contract and observed
alias behavior, not model confidence.

### 7.4 Wire/persistence schema

Define serialization independently. Native bigint is not JSON; a canonical
decimal string is not an ordinary string merely because both use JS strings;
Fn/state/runtime handles are not data codecs. No generic mangled export name
should accidentally become the forever wire identity of a variant case.

Generate codecs only for admitted, authorized wire data with explicit schema
identities and evolution rules. Unknown/malformed wire inputs are validated
before entering the checked world. ABI format versions reject mixed builds;
with no users, they need not supply adapters for every historical format.

## 8. Async and resources: resolve B05, do not invent a competing B12

Both B05 documents are design-only and differ materially:

- `b05-async-design.md` §Q6 proposes a uniform completion-capable ABI, possibly
  `Promise<Result>`.
- `b05-async-design-pro.md` §D4 proposes a uniform private control-frame/machine
  ABI for a join-capable world, with a separate root embedding boundary.

They also differ in join presentation (labeled versus positional product arms).
Do not cite them as one settled implemented contract. Remove obsolete
record-success assumptions when reconciling them with the new value model.

Preserve structured ownership: all launched children settle, authority is
partitioned, no unjoined task escapes, failure products preserve simultaneous
failures, and timeouts do not falsely certify that effects have ceased.
Resources still need acquisition/use/release, ownership/lifetime, and cleanup
rules; await/join syntax alone does not unblock the whole HTTP/SQL/UI catalogue.

Choose the private backend and external root-completion facade **separately**.
A Promise facade does not require guest-visible Promise types; a private frame
machine is not evidence that arbitrary host work meets a real-time settlement
bound. Bigint work on a nonpreemptive event loop still matters.

**Gate:** trace/conformance tests for settlement, duplicates, reentry, mutation,
late effects, simultaneous failures, and host/platform progress assumptions.
No syntax-only async PR, and no preservation of synchronous goldens as a veto.

## 9. Compiler, tools, and specification

### C01 — Parse once; lower once; share the typed core

Use a token lexer, recursive type grammar, structured AST, and typed core. Do
not carry `forward call` as text for a later parser. Eliminate repeated string
splitting of the same Fn/type syntax and duplicated kind-dependent visitors.

Current `checkProgram` prechecks providers to prepare mutated ASTs before a
later per-module checking/execution pass; the LSP has corresponding preparation
logic. This is evidence that stage contracts need strengthening, not a reason
to declare one existing function “the pipeline” and stop.

Specify immutable parse data, resolved typed/elaborated data, proof/evidence
annotations, and lowering data. CLI, LSP, evaluator, proof admission, catalogue,
normalizer, revision identity, and emitter consume the appropriate shared form.
Unknown semantic variants must fail closed. Preserve source-origin maps through
sugar removal, specialization, and optimizations.

### C02 — Ship the canonical formatter before adding more style laws

A formatter is promised but not present as the promised complete language
surface. `canlc normalize` normalizes test outcomes, not source. Current lint
rules can make a compiler-clean file editor-errorful and recommend structural
rewrites rather than merely canonical formatting.

Separate syntax/type/authority errors from advisory refactors and selectable
review policy. The formatter owns whitespace, parentheses, deterministic
metadata layout, and printed resolved forms. It should not rewrite control flow
under the guise of formatting.

**Gate:** parse/format/parse equivalence, idempotence, comments and precise source
spans, literal round trips, and grammar/LSP alignment across the complete corpus.

### C03 — Replace the archaeology requirement with a current specification

Keep historical a/b documents, but publish one current semantic spec and one
status table: implemented, proposed, historical, deferred. Generate syntax,
primitive, intrinsic, and diagnostic inventories where possible.

Concrete drift found:

- root README reports status through a12 despite B11;
- REQUIREMENTS describes interface revision drift as code-content drift,
  Python evaluation, and no emitted match default; current code differs;
- can-idioms describes foreign externs as uncallable despite B02/B10;
- std/seq says bare sequence returns are rejected despite B10;
- B01 says `rev N` must remain, yet its brand example uses `@1` and its
  implementation plan says to parse `@N` headers;
- verification/AST comments still describe contracts as inactive although
  proof tests and the compiler actively enforce the admitted fragment.

These are not harmless for an agent-oriented language: they supply contradictory
instructions to its primary authors. Documentation coherence is part of the
language interface. Do not “fix” history by making an unimplemented proposal
sound shipped; label and link it.

### C04 — Move primitive and host contracts out of scattered special cases

A centralized intrinsic registry should declare signature, effects, evaluation,
proof admission, runtime lowering, and host/authority requirements. Current
`CalleeKind` and dedicated byte/asset/store branches demonstrate the surface
that must agree. Deterministic kernels need not pretend to be ordinary host
observations, but their source application/type rules should be ordinary.

Keep target runtime helpers versioned and shared where appropriate instead of
copying all support code into every module by default. Measure standalone-output
convenience against emitted size; do not sacrifice hermetic builds or exactness.

## 10. A coherent destination, not a promise that this parses today

**Illustrative proposed syntax only:** module/public notation and pattern/let
forms below require a grammar decision. This is deliberately not a `.can` fixture.

```can
module option

public variant Option<T>
  Some(value: T)
  None()

public fn map<T, U>(input: Option<T>, transform: Fn<T, U, []>) -> Option<U>
  emits []
  effects []
  tests
    // Concrete witnesses still belong here; omitted in this sketch only.
  match input
    on Option.Some(value) =>
      let Ok(mapped) = call transform(value)
      Ok(Option.Some(mapped))
    on Option.None() => Ok(Option.None())
```

Important properties, independent of keyword taste:

- no fake success carrier record;
- no `Ok<T>` and no binder-unwrapping `.value` ritual;
- no separate callback invocation grammar or parameter-location restriction;
- constructor patterns bind actual payload values;
- local binding does not hide fallible control flow;
- helper placement does not change pure test semantics;
- generic T/U are typechecked, not inferred from these examples;
- authority remains explicit; higher-order containment is expanded only when
  the checker can establish the corresponding obligations.

## 11. Migration plan and acceptance criteria

The **[master TODO checklist](audit-probes/TODO.md)** tracks all audit sections,
priorities, dependencies, acceptance criteria, and the replacement JEV review.

| Stage | Deliverable | Must not regress |
|---|---|---|
| 0 | Repair F01/F02/F04/F05; version canonical evidence; add whole-library and fresh-output TS gates | Accepted evidence ownership, identity uniqueness, deterministic resolution, valid target output |
| 1 | Type AST and core/stage contracts; authoritative spec and executable cross-boundary matrix | All existing meanings until an explicitly scoped breaking stage |
| 2 | Uniform single-value successes across bodies/contracts/tests/scripts/Fn/host; migrate all sources and goldens | Nominal types, exact values, full errors, authority, evaluator/Node agreement |
| 3 | Unified applications, typed patterns, immutable bindings; retire obsolete forwarding/chain forms | Effects/order, exhaustive failures, source witness accounting, cycles |
| 4 | Scoped modules/public boundary, manifest identities, pure-source test execution | Pins/artifact identity, brand ownership, hermetic host scripts |
| 5 | Finite error rows/completed outcomes and generic data composition; real stdlib combinators | No discarded failures, constraint checking, total recovery mappings |
| 6 | Expanded source higher-order capabilities, host facade/ownership, recursive data and iteration improvements | Preconditions, containment, indirect termination, data ownership |
| 7 | Resolve/implement B05 backend and resource design in their own bounded slices | Join/settlement/authority invariants and honest platform assumptions |
| parallel, after grammar choices | Formatter, spec, diagnostics, useful literal/name cleanup | Source-map quality, round trips, reproducibility |

Stages can be subdivided; public ABI design must be written before Stage 2,
even if complete hostile-host adapters arrive later. Correctness repairs do not
wait for the new parser. Do not throw away the evaluator while introducing a new
IR; it is an independent semantic oracle for backend work.

Goldens are observations of a selected representation, not a constitutional
ban on redesign. For each breaking slice:

1. Specify the intended semantic/ABI delta and write positive and negative tests.
2. Migrate with parsed/resolved structure; do not mass-replace overloaded text.
3. Compare behaviors and effect/error traces, not just TS text.
4. Regenerate affected TS/catalogues/manifests and review their deltas.
5. Run all Go, module, grammar, strict TS, Node, linked/LSP, false-proof,
   authority, and boundary gates. Commit the resulting single supported form.

Benchmarks should include large sequence/JSON work, generic instantiation growth,
build/LSP latency, allocation/stack behavior, and agent task success. A smaller
syntax is a hypothesis; measure successful edits and diagnostic recovery rather
than assuming token reduction alone makes agents more reliable.

## 12. JEV review record

**Correction: these reviews are withdrawn as decision support.** The supplied
Can context was insufficient. JEV cannot inspect the repository or research
missing facts; we must supply the complete context ourselves. The scores below
are historical records only, not support for this audit's recommendations.
Those recommendations remain the author's proposals. Reproducible compiler
findings remain valid independently of these calls. The replacement review is
tracked in [TODO section A](audit-probes/TODO.md#a-redo-the-jev-review-correctly).

`jev-1.13.0`, Choice; live HTTP/Choice documentation consulted. Full requests,
alternatives, distributions, and responses are in `audit-probes/jev-*.json`.
These are comparative design judgments, **not proofs, user approval, or empirical
usability results**. Narrow wins remain narrow. The JEV calls preceded the final
F04/F05 integration probes; those findings come from actual compiler/TS runs,
not from model judgments.

| Question | Selected | Probability | Confidence |
|---|---|---:|---:|
| Sequence | evidence_then_core | 1.00 | 1.00 |
| Success semantics | uniform_one_value | 0.97 | 0.96 |
| Application surface | unified_call | 0.57 | 0.42 |
| Modules | scoped_imports | 0.84 | 0.77 |
| Pure-source tests | execute_pure | 0.87 | 0.80 |
| Public host boundary | owned_validated | 0.50 | 0.26 |
| Error abstraction | finite_error_rows | 0.97 | 0.96 |
| Async ABI process | resolve_b05 | 1.00 | 1.00 |
| Local type checking | bidirectional | 0.64 | 0.46 |
| Sequence type spelling | uniform_generic | 0.96 | 0.93 |
| Literals | conventional_exact | 0.63 | 0.45 |
| Boolean evaluation | eager_explicit | 0.65 | 0.47 |
| Argument labels | preserve_labels | 0.83 | 0.76 |

In particular, the host decision was **0.50 versus 0.49 for trusted direct
interop**, and unified call spelling had meaningful alternatives. Neither should
be presented as a settled consensus. The ownership recommendation names the
intended external trust contract; the `call` keyword and literal conventions
should be tested with actual agent tasks before freezing a grammar.

## Bottom line

Preserve **guarantees**, not accidents of the first implementation. Break source
and generated output deliberately while there are no users. Keep tests as
migration evidence, repair incomplete identity machinery now, and stop charging
future programs for limitations that belong inside the compiler.

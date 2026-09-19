# Workstream A: authority and evidence dossier

Status: research dossier at `8bbcf13b33c74d26bc7bb0f9b72fba0d5415d70b`
(2026-09-19). This document reports the implementation and isolates decisions;
it does not approve a redesign. **Implemented** means present on ordinary compiler
paths now. **Proposed** means the audit/TODO asks for it. **Historical** means a
design record describes an earlier stage and may be superseded.

## Goals, non-goals, and trust boundary

The living `REQUIREMENTS.md` says Can is a small contract-first language for AI
agents: verbose, explicit, one canonical form, with behavior declarations visible
instead of inferred. Exact values, exhaustive decisions, declared failures,
compile-time decision tables, explicit authority, and honestly scoped proofs are
the durable goals. A weak table is not a universal specification; contracts and
review-owned acceptance evidence must carry obligations that examples cannot.

Current explicit non-goals include null/undefined, exceptions and hidden early
return, implicit conversions, default arguments, operator overloading, macros,
lazy evaluation, optional annotations, and a native backend in v0.1. The repository
has zero external users. Migration cost matters, but preserving old spellings,
ABI layouts, generated TypeScript shapes, and goldens does not.

The trust model is layered:

| Claim | Current authority |
|---|---|
| A source program is well typed/exhaustive and its rows pass | ordinary `canlc` checking and evaluator execution |
| Every source match arm has evidence | CAN4107: a passing row executed it, or the narrow local identity-relay certificate applies |
| A contract holds universally | active verifier, only for admitted `requires`/`ensures`; Z3 QF_LIA discharges each obligation |
| A revision interface did not drift | CAN6013, only when the caller supplies an accepted format-1 baseline |
| A pinned expectation did not weaken | CAN6017 warning against that accepted baseline; advisory, and subject to F01 |
| A cross-file pure composition behaves correctly | explicit Go `runLinkedPure` vectors; ordinary Can rows still use scripts |
| A host implementation honors its declaration | trusted adapter plus separate smoke/conformance tests; source errors are not an effect proof |

## Current module, ownership, and revision model

**Implemented.** Each input has a path-derived `Module.ID`, a source `mod` label,
and a generated output stem. Same basenames are disambiguated and state cells are
keyed by file identity. `provides` must exactly enumerate functions, types,
variants, brands, constants, and externs defined in the file. A `uses` entry has
the form `name@rev`, resolves a global bare name to a supplied provider, and checks
that provider's currently loaded revision. There is no historical artifact store:
if only rev 2 is loaded, a caller pinned to rev 1 fails.

The symbol world is not a namespace graph. Function/extern, variant, and constant
collisions are rejected, while record, error, and brand identities still share
global short-name registries with silent first-wins/coalescing behavior. Every
helper must appear in `provides`, and emission exposes implementation helpers.
Module-header `emits` is parsed but not checked against function declarations.

These compiler excerpts are the operative rule:

```go
// compiler/check.go, buildWorld
_, isFn := d.(*FnDecl)
...
} else if seenOther[name] {
    continue
} else {
    seenOther[name] = true
}
```

```go
// compiler/emit.go
func recordShapes(mods []*Module) map[string][][2]string {
    recs := map[string][][2]string{}
    ...
    if _, seen := recs[td.Name]; !seen {
        recs[td.Name] = td.Fields
    }
}
```

F02 is therefore an ownership failure, not only a diagnostic omission. Both
declarations below pass, and swapping their contents between sorted filenames
changes the selected shape from `int` to `str`:

```can
mod a
  provides [Shared__Record]
  uses []
  emits []
type Shared__Record rev 1 (
  value: int
)
```

```can
mod b
  provides [Shared__Record]
  uses []
  emits []
type Shared__Record rev 1 (
  value: str
)
```

Revision identities are keys such as `fn:m.m__go@1`, canonical interface content,
and a transitive shape closure. An accepted baseline is external review authority;
candidate generation writes `accepted: false` and cannot appoint itself. Comments,
locations, bodies, tests, and `given` are excluded from interface identity.
Contracts, errors, effects, field names, and nominal dependencies are included.
There is no proof cache; contracted declarations are re-proved on each compile.

## Effects and capabilities

**Implemented.** The source effect row currently describes only module-private
state cells. `state__get` and `state__put` are deterministic intrinsics, execute
against a fresh store per test, take no `given`, and require `.read`/`.write`.
Capabilities propagate through local and `uses`-pinned Can calls; undeclared and
stale entries fail before execution.

```can
mod m
  provides [m__bump, M__T]
  uses []
  emits []
state M__C: int = 0
type M__T rev 1 (
  total: int
)
fn m__bump(by: int) -> M__T rev 1
  effects [M__C.read, M__C.write]
  emits []
  tests
    three(3) => Ok(3)
  match call state__get(M__C)
    on Ok c => match call state__put(M__C, c.value + by)
      on Ok _ => Ok(c.value + by)
```

This is the complete positive fixture used by `compiler/effects_test.go`. The
limitation is structural: `ExternDecl` has params, return, errors, and revision,
but no effect/capability field. `std/host` correctly
describes wall clock as an observation in prose, yet source types encode only
possible errors:

```can
extern host__wall_now() -> Clock__Instant rev 1
  emits [clock.unavailable]
```

An empty-error extern is still recognized and rejected by linked-pure and Fn
target admission. The gap is that its signature cannot describe which host
authority it exercises: `emits []` is not an observation footprint. **Proposed:** effect
rows independent of error rows, transitive through named calls and callable types,
with distinct deterministic-kernel and external-observation classes. Existing
brand export and asset bridge certificates are real ownership controls and must
remain authority proofs rather than become casts.

## Tests, scripts, coverage, and location

**Implemented.** Every function must carry named, complete decision-table rows.
Each row gets a fresh state store. Local helpers execute real bodies. Externs and
cross-file Can functions consume call-site `given` exchanges; expected arguments
are checked, missing and leftover exchanges fail, and injected errors must belong
to the callee's declared `emits`. Script presence is not coverage. Passing execution
of a particular source arm, or the narrow local relay certificate, is coverage.

This ordinary cross-file test does not execute `leaf__copy`:

```can
mod middle
  provides [middle__copy]
  uses [leaf__copy@1]
  emits []

fn middle__copy(value: str) -> Encoding__Text rev 1
  emits []
  tests
    a("A") => Ok("A")
  match call leaf__copy(value)
    given
      a => [exchange args (value = "A") outcome Ok("A")]
    on Ok r => Ok(r.value)
```

The evaluator makes the mode split explicit:

```go
// compiler/eval.go, foreign-call branch
if ctx.Linked {
    val, err := evLinkedOutcome(ctx.Prog, fname, scrut, env, ctx, owner)
    ...
} else {
    val, err := evScriptOutcome(ctx.Prog, fname, scrut, node, env, ctx, owner)
    ...
}
```

`runLinkedPure` is **implemented** as a test-only Go helper. A committed vector
selects an explicit module set, root and revision, checked arguments, and expected
outcome. It first requires a clean world, walks every reachable branch, refuses
externs, state, nonempty effects, unresolved calls and cycles, executes real Can
bodies with no script fallback, uses fresh state, and discards its trace so it
cannot satisfy CAN4107. The current `TestLinkedPureMismatch` proves why this is
needed: all scripted unit rows are green while the real leaf returns `""` for
`"A"`; linked execution catches the mismatch in both module orders.

## Contracts, proof, and acceptance

**Implemented.** `requires` and outcome-indexed `ensures` are parsed and active on
ordinary CLI/LSP paths (the historical `a81` phrase “shipped, unwired” is
superseded by `a82`). The admission fragment supports booleans, linear integer
arithmetic, finite acyclic records of integers/booleans, path-sensitive body
reasoning, and verified callee summaries. Strings, brands, decimals, nonlinear
arithmetic, division, and unsupported bodies fail closed. Obligations are
`Requires ∧ Path ⇒ Ensures`; call sites prove callee preconditions. `given` rows
are never proof facts. A failed callee summary cannot prove its callers. Ordinary
uncontracted declarations are reported as “not universally verified.”

```can
fn m__max(left: int, right: int) -> M__Out rev 1
  emits []
  requires
    true
  ensures
    on Ok result
      result.value >= left
      result.value >= right
      match result.value == left
        true => true
        false => result.value == right
  tests
    ordered(1, 2) => Ok(2)
    reversed(2, 1) => Ok(2)
  match left <= right
    on true => Ok(right)
    on false => Ok(left)
```

**Implemented acceptance authority** is A-light. A row suffixed `pinned` executes
like any row, but its canonical expectation is recorded in an accepted baseline.
Changed, removed, or demoted rows produce advisory CAN6017 after the clean gate.
Unpinned rows churn silently. No stdlib row is currently pinned.

F01 demonstrates that the canonicalizer is not complete authority. `canonSmall`
falls back to `unknown-kind(<kind>)`; there is no `fnref` case. `canonPattern`
omits type arguments, and `canonNode` omits `MatchInvoke.InvokeArg`. Thus this
accepted row can change targets with no warning when the factory body changes too:

```can
fn acc__factory() -> Fn<int, int, []> rev 1
  emits []
  tests
    one() => Ok(fnref acc__a()) pinned
  Ok(fnref acc__a())
```

Current output is identical for `acc__a` and `acc__b`:
`one() => ctor(Ok)[value=unknown-kind(fnref)]`; warnings are `[]`. Typed-Ok,
typed-pattern, and invocation-argument structural probes also compare equal.
These narrower omissions are not evidence of a persisted proof-cache exploit.

## Termination and resource bounds

**Implemented.** Program-wide call cycles are rejected. Direct self-recursion is
admitted only through one of the recognized schemas: unit descent, Euclid, or
binary narrowing. The annotation must name integer parameters, must be used, each
self-call must have the exact admitted step, and recursion must sit below the
recognized false guard. Tests do not run while this proof is open.

```can
fn std__seq__concat_from<T>(left: Seq<T>, right: Seq<T>, pos: int, n: int, acc: Seq<T>) -> Seq__Values<T> rev 1
  decreases n
  emits []
  tests
    done<T=int>(Seq<int>[1], Seq<int>[2], 1, 0, Seq<int>[1, 2]) => Ok(Seq<int>[1, 2])
    step<T=int>(Seq<int>[1], Seq<int>[2, 3], 0, 2, Seq<int>[1]) => Ok(Seq<int>[1, 2, 3])
  match n <= 0
    true => Ok(acc)
    false => match pos < #right
      true => match call std__seq__concat_from<T>(left, right, pos + 1, n - 1, acc + right[pos])
        on Ok r => Ok(r.values)
      false => Ok(acc)
```

The proof establishes mathematical descent for admitted shapes, not practical
cost or target stack safety. The evaluator retains a 1024-call resource backstop;
TypeScript emits ordinary recursion. JSON uses explicit frame stacks and fuel
because mutual recursive data traversal is not expressible. **Proposed:** retain
termination obligations, add structural/well-founded measures and finite traversal
constructs, lower tail calls, and measure cost separately.

## Stdlib customers and present limits

At this HEAD, `std/` has 13 Can files, 6 revision pins, 16 `given` blocks, 63
`decreases` declarations, and 3 `effects` declarations. It has zero
`requires`, `ensures`, or `pinned` rows. These counts describe the checked-in
corpus, not language promises.

- `std/json` pins four scalar converters and scripts their pure cross-file calls;
  the comment that pure std code has no `given` is false for current composition.
- `std/quota` and `std/scalars` have Go-only linked-pure vectors that execute real
  provider bodies in both file orders.
- `std/host` scripts clock, randomness, hashing, secrets, environment, and logging
  externs, but source effect types do not name those observations.
- Sequence, text, ratio, scalar, JSON, and byte customers depend heavily on the
  recognized recursion schemas and manual fuel.
- Repeated carrier names such as `Int__Value`, `Str__Value`, and `Bool__Value`
  rely on the same global coalescing exposed by F02.
- Whole-stdlib compilation is not a current guarantee: incompatible declarations
  of `math.nonterminating_decimal` in ratio and scalars fail only later at a
  payload mismatch. Per-module green tests do not establish one composable world.

## Documentation contradictions to resolve before review

1. `REQUIREMENTS` R2 promises double-provided names are compile errors; F02 and
   `buildWorld` explicitly permit duplicate records/errors/brands.
2. R4 describes an `ai-lock.json` mapping and says a provider bump never breaks a
   pinned caller. The compiler instead takes an optional explicit accepted
   baseline, and it has no old-revision store; unavailable pins fail.
3. R6 says outside-world authority is explicit, but extern declarations cannot
   express effects. Error sets are not observation/capability sets.
4. R11 says compile-time evaluation is “Python today”; it is implemented in Go.
5. `can-idioms.md` says pure std code has no `given`; pure cross-file JSON/quota
   calls are scripted under ordinary semantics.
6. Historical `a81` says verification is unwired; `a82`, `compileAll`, and LSP
   activate it. The current fact is active verification for contracted code.
7. The title of `a87` still says draft/pre-decision, while its status and code say
   shipped/ratified. Its claim that pin drift warns across revisions is not the
   executable rule: rev-changed function identities are skipped by CAN6017 and
   handled first by CAN6013.
8. `std/README.md` calls admitted modules blessed and non-provisional, but there is
   no green all-stdlib composition gate. That label cannot imply joint ownership
   or target validity.
9. Audit M01/E01/F01/F02 recommendations remain **proposed**. The master TODO
   expressly says they are not approved specification.

## Module decision: balanced alternatives

### M-A — Owned module namespaces and qualified imports

Give each package/module a stable declared identity; key declarations by
`ModuleID + local name`; add explicit public/private declarations and qualified
imports or aliases. A file move is inert when its declared module identity is
unchanged. Two modules may both own `Record`, `parse`, or `Some`.

Semantic consequence: ownership becomes part of nominal equality, revision keys,
errors, brands, grants, host bindings, traces, and emitted symbols. This is the
cleanest authority model and the largest resolver/emitter migration. Authority
obligations: deterministic import resolution; no private access; ambiguous import
refusal; declaring-owner host binding; module identity independent of load order;
separate artifact identity from source path. With zero external users, migrate all
stdlib declarations structurally and delete the global-name model in one slice.

Illustrative before/after (proposed syntax):

```can
// before: both files declare global Shared__Record
type Shared__Record rev 1 (value: int)

// after
module accounting
public type Record rev 1 (value: int)

module display
import accounting.Record as AccountRecord
private type Record rev 1 (value: str)
```

### M-B — Keep globally qualified spellings, add explicit owner identity

Retain names such as `accounting__Record`, but require one owner in the registry
and resolve imports to an owner/declaration ID instead of a string. Privacy can be
metadata on declarations. Migration is smaller and emitted names stay readable,
but source names still encode hierarchy and rename/move policy remains coupled to
spelling. It must still reject duplicate owner-qualified identities at declaration
time and remove duplicate carrier/error redeclarations.

### M-C — One global namespace with strict uniqueness

Make every duplicate declaration an error and retain `provides`/`uses` as today.
This repairs F02 cheaply and is deterministic, but preserves long names, exports
helpers, prevents unrelated packages from sharing local vocabulary, and makes
ownership implicit in spelling. Its authority obligation is simple uniqueness,
yet it does not satisfy the audit's namespace/visibility goal. It is a valid small
language choice if that goal is rejected explicitly.

For all alternatives, migrate parsed/resolved identities rather than text search;
test both source orders, same local names in different owners, ambiguous imports,
private access, brand/error ownership, exact revision pins, and declaring-owner
host bindings. Do not keep a compatibility resolver.

## Pure-source testing decision: balanced alternatives

### T-A — Real pure bodies by default; explicit mocks

Ordinary rows execute checked pure Can across module boundaries. Only externs,
state/observations, or an explicitly marked contract mock consume scripts. Moving
a pure helper across files then preserves behavior and evidence obligations.
This best matches source semantics and the colocation goal, but changes existing
unit-test meaning and can expand test graphs and cost.

Before/after (the `mock` spelling is illustrative, not approved):

```can
// before: location makes a pure provider scripted
match call std__convert__int_to_str(value)
  given
    one => [exchange args (value = 1) outcome Ok("1")]
  on Ok text => Ok(text.value)

// after: real pure provider executes; deliberate injection is explicit
match call std__convert__int_to_str(value)
  on Ok text => Ok(text.value)

match mock call payment__authorize(request)
  given
    denied => [exchange args (request = request) outcome payment.denied()]
  ...
```

### T-B — Preserve unit scripts; add a source-level linked-test section

Keep current rows as hermetic contract tests and add explicit Can integration
vectors selecting roots/modules. This preserves failure injection and makes linked
evidence visible in the language artifact. It creates two test modes users must
understand, and refactoring a helper still changes ordinary-row meaning. Reports
must label mocked and linked evidence separately.

### T-C — Keep the current Go-only `runLinkedPure`

This has the smallest language surface, explicit roots, and a proven admission
walk. It also leaves composition outside the Can artifact, requires compiler-test
code for each customer, and cannot satisfy the strong reading of “tests and
business logic are the same artifact.” It is acceptable only if that goal is
narrowed to unit tables.

Every alternative needs the same authority obligations: resolve exact declaration
identities and revisions; compute transitive purity from effects including host
observations; inspect all reachable branches; preserve fresh state; prohibit
mixed real/script fallback; keep request/response pairing and missing/leftover
checks for mocks; give effect sites stable identities; refuse cycles/open
termination proofs; distinguish unit, linked, executed, certified, and uncovered
evidence; never let integration traces silently satisfy source-arm coverage.

Migration should first classify current `given` sites as pure-provider scripts or
true observation mocks, add parity vectors, then switch semantics and delete
obsolete tables in one coordinated stdlib change. Acceptance: extracting or moving
a pure helper changes neither values nor error/effect traces; a deliberately mocked
failure remains injectable; an impure zero-error extern never enters real-pure
execution; provider-body defects fail linked evidence; file order is inert.

## Validation performed

The F01/F02 safe probe passed while reproducing both findings; complete output is
in `evidence-core-probes.txt`. Targeted current tests also passed for linked-pure
mismatch/control, accepted-baseline authority and contract drift, pinned weakening,
effect propagation, contract admission/refutation, proof of `max`, and same-basename
module identity. No compiler source or temporary probe remains changed.

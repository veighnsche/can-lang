# Workstream A dossier: host boundary, target ABI, and B05 async process

**Snapshot.** Current facts below were inspected and reproduced at
`8bbcf13b33c74d26bc7bb0f9b72fba0d5415d70b` on 2026-09-19. **Current** means
implemented behavior at that snapshot. **Proposed** means a design with no
compiler/runtime implementation. **Historical** means useful provenance that is
not current authority. Can has zero external users, so a chosen redesign may
replace syntax and ABI outright; migration still needs semantic and acceptance
evidence, but no compatibility mode.

This packet is self-contained for review. Paths identify provenance only; all
facts needed for the host-boundary and B05 decisions are included inline.

## Shared language model

**Current goal.** Can is a contract-first language optimized for agents to read,
write, check, and emit as TypeScript. Functions declare return type, named error
set, effects, decision-table rows, and call-site observations. Errors are values,
matched exhaustively. Pure Can execution, scripted foreign observations, linked
execution, proof admission, TypeScript shape checking, and real host execution
are different evidence categories.

**Current source example.** The host shelf declares a nominal record, an extern,
and a wrapper whose tests script the extern instead of executing it:

```can
type Clock__Instant rev 1 (
  millis: int
)

extern host__wall_now() -> Clock__Instant rev 1
  emits [clock.unavailable]

fn std__clock__wall_now() -> Clock__Instant rev 1
  emits [clock.unavailable]
  tests
    ok() => Ok(1726920000000)
    down() => clock.unavailable()
  match call host__wall_now()
    given
      ok => [exchange args () outcome Ok(1726920000000)]
      down => [exchange args () outcome clock.unavailable()]
    on clock.unavailable _ => clock.unavailable()
    on Ok t => Ok(t.millis)
```

The decision table proves behavior under the listed observations. It does not
prove that the real clock is available, accurate, terminating, or isolated.
`std/host/host.externs.ts` is a separate implementation obligation, smoke-tested
under Node. Its actual synchronous boundary is:

```ts
export function host__wall_now():
  | { $can_kind: "ok"; millis: bigint }
  | { $can_kind: "clock.unavailable" } {
  return { $can_kind: "ok", millis: BigInt(Date.now()) };
}
```

## Current emitted layouts and exact numerics

The compiler rule is literal:

```go
var tsBase = map[string]string{
  "str": "string", "int": "bigint", "bool": "boolean",
  "dec": "string", "Bytes": "Uint8Array",
}
```

`Seq<T>` maps recursively to `T[]`; a brand maps to its underlying primitive;
a record maps to an exported structural object alias; a variant maps to an
exported `$can_kind` discriminated union; a callable maps to a JS closure over
the same result union as a named function. Extern signatures are statically
restricted to data-only types, but exported generated functions that accept a
Can `Fn` accept ordinary JS functions at runtime.

Concrete current output:

```ts
export type Success__Pair = { left: bigint; right: boolean };
export type Success__Option$T$int =
  | { $can_kind: "Success__Some$T$int"; value: bigint }
  | { $can_kind: "Success__None$T$int" };

export function success__id$T$Seq$L$int$G$(value: bigint[]):
  { $can_kind: "ok"; value: bigint[] } {
  return { $can_kind: "ok", value: value };
}

export function success__id$T$Success__Pair(value: Success__Pair):
  { $can_kind: "ok"; left: bigint; right: boolean } {
  return {
    $can_kind: "ok" as const,
    left: value.left,
    right: value.right,
  };
}
```

`int` is an unbounded mathematical integer in the evaluator (`*big.Int`) and a
JS `bigint` in emitted code. `+`, `-`, and `*` are exact. Integer division/modulo
use Euclidean `DivMod`; zero division is a loud fault. `dec` is an exact
canonical digit string such as `"0.3"`, evaluated with integer/rational
arithmetic and emitted through `$canDec*` helpers. It is not IEEE-754 and has no
implicit rounding. Its runtime TS type remains plain `string`, so the canonical
representation invariant is trusted at direct ingress. Canonical decimals retain
a dot and at least one digit on each side, strip redundant leading integer and
trailing fractional zeros, and fold negative zero to `"0.0"`; `d"0012.500"`
emits `"12.5"`, while `d"2.00"` emits `"2.0"` (`canonDec` and `$canDecNorm`).
`JSON.stringify` cannot
serialize `bigint` without an explicit codec/replacer.

Records and outcome objects are ordinary mutable JS objects. Sequences are
mutable arrays. Bytes are mutable `Uint8Array`. No emitted `readonly`, copy,
freeze, validation, prototype check, cycle check, or ownership transfer is
applied at ordinary exported function entry. Evaluator-side Bytes are owned
`[]byte`, and evaluator sequence append copies the outer array; those facts do
not make the public TS values owned.

## Four distinct boundaries

| Layer | Current or proposed contract | Identity and ownership |
| --- | --- | --- |
| Can source | **Current.** Nominal declarations, declared errors/effects, exhaustive outcomes, tests and observations. No JS values or promises are source values. | Static declaration/revision identity; cells are module-private. |
| Private generated ABI | **Current.** Direct TS calls and `$can_kind` unions. **Proposed B05 alternatives** replace or completion-lift this internal call convention. | Today values are ordinary JS references; internal helper names and generic stamps are compiler details. |
| Public embedding API | **Not currently separated.** Every exported generated TS function is directly callable, so trusted internal layout is also accidental ingress. | Today host callers can supply aliases, getters, prototypes, raw callbacks, and invalid decimal strings. A proposed validated public ingress would check values and acquire ownership; retaining an explicitly trusted embedding is another policy choice. |
| Wire/persistence schema | **Not supplied by generic TS layout.** JSON/std codecs are explicit library operations. | `bigint`, canonical-dec strings, nominal brands, state, callables, and runtime handles require explicit authorized codecs and stable schema identity. Mangled TS names are not wire identities. |

The design decision is therefore not “make all TS immutable.” A practical split
can retain a cheap, explicitly trusted internal ABI while giving public ingress
validated owned snapshots or opaque handles. A readonly annotation does not
stop an existing alias from mutating; shallow `Object.freeze` does not validate
prototypes, getters, cycles, nested arrays, decimal form, variant tags, brands,
or `Uint8Array` contents.

## Trust, authority, and faults

**Current checked facts.** Can checks the declared shape of calls and outcome
handling. Extern result unions contain only declared `Ok` and error members.
Source-level extern signatures cannot contain `Fn`. State effects are explicit
and transitive for Can cells. Foreign implementations are resolved from the
declaring module's `.externs` stem.

**Current trusted facts.** A real host adapter tells the truth, terminates as its
platform permits, returns the declared union, preserves nominal/decimal/byte
invariants, and does not retain harmful aliases. TypeScript checks source shape;
it does not prove host purity, liveness, ownership, exactly-once completion, or
absence of later effects. The current host shelf is synchronous and includes
clock, random bytes, hash, secret comparison, environment, and logging. It has
real Node implementations, but no general bounded async settlement protocol.

**Current faults are three categories.** Declared language errors are values in
`emits`. Primitive-domain faults are loud evaluator errors / emitted JS throws,
for example `int division by zero`, `str index out of range`, `str slice out of
range`, and `seq index out of range`. Resource failures are also outside
`emits`; the evaluator's local-call depth cap is 1024 and fails loudly. Exhaustive
emitted switches contain a defensive `throw new Error("unreachable")`. These
faults must not be silently converted into an ordinary declared error, nor may a
B05 runtime contract violation be reported as healthy Can success.

## Current confirmed findings F03-F05

Raw commands and exit codes are in `evidence-host-integration.txt`.

**F03 remains current.** The direct Node probe printed:

```text
ABI_ALIAS true 9
GETTER_INPUT { '$can_kind': 'ok', left: 7n, right: true } 2
HOST_CALLBACK_EFFECTS 1
```

The returned sequence is the same mutable array; record property access executes
host getters; a raw JS callback crosses an exported generated Fn parameter and
runs. This confirms an ownership/trust design gap. It is not an exploit of a
promised hostile-host sandbox: current documentation trusts host interop.

**F04 remains current.** Whole-stdlib compilation exits 1:

```text
canlc FAILED: scalars.can:1016: math.nonterminating_decimal field numerator: got dec, want int
```

The collision is concrete:

```can
// ratio.can
error math.nonterminating_decimal(numerator: int, denominator: int)

// scalars.can
error math.nonterminating_decimal(dividend: dec, divisor: dec)
```

The compiler coalesces a global short identity and reports the later payload
mismatch instead of rejecting the conflicting declarations at their definitions.

**F05 remains current.** The repository TS project exits 1 for missing relative
`./host`, `./host.externs`, `./quota`, and `./scalars` artifacts and JSON TS2322
discriminant widening. The isolated dependency-complete JSON build separates
those causes: canlc passes 785 rows and emits two modules, then strict TS exits 1
with four TS2440 imported/local type conflicts and TS2322 at `json.ts:308` and
later frame-construction sites. Passing source rows and byte-stable goldens do
not establish valid target output.

## Actual stdlib customers and demand

The host shelf exercises the current foreign boundary with exact representations:
clock milliseconds are `bigint`; random/hash payloads are `Uint8Array`; secret,
environment, and hash-profile brands erase to `string`; logging explicitly
converts its bigint level to decimal text before JSON serialization. Its Node
smoke test is evidence for those seven implementations, not a hostile-host proof.

JSON is the largest concrete target customer: 785 Can rows across JSON and
scalars, explicit raw numeric text in `Json__Num`, and a source-level recursive
frame machine because mutual recursion and variant sequences are restricted.
Its current strict-TS failure is direct evidence that source success and emitted
shape success are separate gates.

HTTP, SQL, and UI remain catalogue customers blocked on Async plus Resources
(UI also needs function support). B05 alone does not solve connection ownership,
transactions, streams, cancellation, acquisition/use/release, or cleanup.

## Documentation contradictions resolved for this packet

- The root README's “through a12” status is historical/incomplete. B00-B11 and
  later stdlib slices exist; feature status must come from current code and the
  audit inventory, not that sentence.
- `REQUIREMENTS.md` says emitted exhaustive matches use a bare switch with no
  default. Current emitter code deliberately adds a defensive unreachable throw
  for strict narrowing and loud emitter faults. The implementation is the current
  fact; the prose is stale.
- `contract verification: succeeded` does not mean every compiled function is
  universally verified. The same output listed `verified declarations: []` and
  the JSON/scalar declarations as uncontracted.
- Older B02-era claims that host implementations were unwritten are superseded:
  `std/host/host.externs.ts` has synchronous real implementations. What remains
  absent is bounded asynchronous settlement and isolation.
- Both B05 documents are **proposed design only** at the same inspected 8fccbb7
  baseline. Neither syntax, completion metadata, scheduler, or ABI is implemented.
- The earlier ASTRA join block is **historical illustrative direction**, not a
  parser contract. Its key obligations survive: account for every initiated
  child before dispatch, complete outcome products, authority partition, and
  explicit liveness assumptions; the two B05 proposals differ on host terminality.
- The two B05 docs disagree on join syntax (labeled outcome slots versus
  positional slots, descriptor/site spelling, singleton policy). This packet
  does not present either spelling as current or silently combine them.

## B05 proposal A: uniform completion-capable callable ABI

**Proposed in `b05-async-design.md`.** Source has `match join` with no descriptor,
pending value, `async`, or `await`. Branch labels recur in qualified arm slots
such as `left.Ok`; zero branches reject and a singleton Can or host branch is
legal. Declaration order fixes argument evaluation and launch. All generated Can
functions use one completion-capable TS convention, possibly `Promise<Result>`,
including functions that resolve immediately.

Representative lowering:

```ts
const leftArg = /* evaluate once */;
const rightArg = /* evaluate once */;
const leftRun = left__read(leftArg);
const rightRun = right__read(rightArg);
return Promise.all([leftRun, rightRun]).then(([left, right]) => {
  // exhaustive product dispatch over both terminal $can_kind values
});
```

This is close to platform completion primitives, but completion-lifts every call,
changes every function and adapter signature, and allocates Promise/closure work
on immediate paths. `Promise.all` is only collection machinery: all arguments
must be prepared before launch, every declared outcome collected, and the labeled
product dispatched without a first-error rejection or escaping host promise.

A join-reachable extern uses dynamic `completes <int parameter> else <declared
timeout error>` and maps throw/rejection through `rejects <declared error>`;
multi-branch host reachability also needs `concurrency isolated`. The adapter
reports a declared outcome by the supplied bound. A defines no separate cleanup
bound, private receipt protocol, or faulted-world state, so its declared timeout
does not certify B-style quiescence. Migration regenerates the whole emitted ABI,
goldens, adapters, imports, harnesses, and strict TS fixtures; zero users removes
any need for a dual direct/Promise ABI.

## B05 proposal B: uniform private control-frame machine

**Proposed in `b05-async-design-pro.md`.** Every function in a join-capable
linked world uses a private entry/control frame. An ordinary call pushes/enters
the callee machine; a join allocates a typed child vector and fixed terminal
slots; bounded extern receipts fill private slots; product dispatch runs only
after every slot is terminal. Frame pointers, queues, event identities, and
return positions are runtime control state, never Can values.

Its source head is `match join <local-site>`. Branch labels name declarations
and diagnostics, but arms are positional products of the real `Ok` or declared
error patterns; they do not invent label-qualified outcome names. Zero branches
reject. A singleton native join also rejects as redundant; the only singleton
form directly names a completion-contracted extern. This is a substantive
source and admission-policy alternative to proposal A, not merely another
lowering for the same settled syntax.

Conceptual example:

```text
root frame: pair__read(args, return-site)
  join site reads
    child[0] -> frame read__left(...), slot[0]
    child[1] -> frame read__right(...), slot[1]
  barrier waits for slot[0] and slot[1]
  dispatch (slot[0].tag, slot[1].tag) once
  retire both children and root frame
```

Registration, duplicate receipts, reentry, traces, and failed worlds have explicit
runtime homes; Promises can stay behind the host service. The price is frame
typing, scheduling, world admission, event addressing, state ownership, fault
reporting, and evaluator parity. A join-capable world cannot guess per-function
ABI. A wholly join-free artifact may retain the old emitter, but cross-ABI linkage
requires re-emission or an explicit bridge.

B resolves exact Can-state authority, treats all host access as one conflicting
`Host__World.use`, registers children before receipts, writes each slot once,
rejects unknown/duplicate/late receipts, forbids reentry, and initially admits one
active root per world. Authority remains owned through cleanup; contract failure
fails the world instead of manufacturing a declared error. Stable event identity,
not a racing script cursor, drives replay. Migration needs a new runtime, linked-
world ABI/executor, golden class, and host conformance harness; no legacy bridge
is owed.

## Proposed settlement contract and unresolved differences

Only the private-machine document, proposal B, specifies revision-bound `T/S`
settlement metadata like:

```can
extern net__fetch(key: str) -> Fetch__Value rev 2
  effects [Host__World.use]
  emits [net.down, net.timeout]
  completion trusted
    timeout_ms 2000
    settle_ms 2100
    timeout net.timeout
```

This is unimplemented and illustrative. In B, `T` fixes timeout disposition and
`S` bounds a receipt after cleanup. Settled means no authority for later effects,
buffer mutation, second receipt, or Can reentry. A timer racing an active worker
fails that contract. Shape and receipt validation are checkable; service liveness,
cleanup, and platform progress remain trust. Duplicate receipt, unexpected
rejection, reentry, mutable alias, or failed settlement is a world-failing host
fault, never a declared outcome. Bigint on a nonpreemptive event loop can delay
timers, so the bound needs independent progress or another explicit platform
assumption.

Proposal A binds timeout to an extern `int` parameter: non-positive immediately
returns the timeout error; positive asks the adapter to resolve by that bound.
`rejects` maps throw/rejection to a declared error and `concurrency isolated`
admits overlapping host calls. A has no `S`, receipt slots, one-root rule, or
specified late/duplicate world fault. Its result is not evidence of cleanup.

Any ownership claim should state when child authority retires, what later effects
are forbidden, and how violations reach the embedder. This is a candidate
obligation, not an agreed common contract. Review may adopt B, strengthen A, or
choose another checked mechanism with explicit guarantees and costs.

## Balanced decision and process options

Shared candidate goals are lexical joins, no source pending values, collection of
all reported outcomes, simultaneous-failure information, effect partition, no
source completion callback, and separate Resources. They disagree on terminality:
A uses bounded adapter results and declared timeout/rejection; B requires settled
receipts and faults the world on contract breach.

A has no site descriptor, admits native singletons, and labels arm slots. B has a
local site, rejects native singletons except direct bounded-host use, and uses
positional real-outcome slots. Labels expose branch identity; positions avoid a
second outcome namespace but bind meaning to declaration order. Neither is current.

Both documents were drafted before B07-B11 broadened success types. Their
success-record annotations and flat-record binder examples are historical design
assumptions, not present restrictions on all Can functions. A chosen join contract
must be reconciled with the current scalar/variant/Seq/Bytes/Fn success inventory
and any separately approved whole-value convention; neither document settles
that migration by itself.

Backend and public root facade must be separate decisions. A private frame
machine may expose `runRoot(...): Promise<Result>` to a JS embedder. A Promise
internal ABI may expose a validated root object rather than export every generated
function. Choosing Promise at the public edge does not create Can promises;
choosing frames internally does not prove host settlement.

Two processes remain credible. **Completion ABI first** migrates callables and
adapters, strict-checks them, then adds joins; ABI propagation becomes visible
early, but Promise scheduling can harden before receipt/reentry semantics exist.
**Machine contract first** specifies frames, receipts, world lifecycle, traces,
and ordinary-call parity before syntax; it prices ownership early but builds a
larger runtime before customer measurement.

Neither may land syntax-only. Both need once-only argument evaluation, all-branch
launch/collection, completion-order and simultaneous-failure cases, effect/final-
store parity, strict target shape, and a real adapter. A must test dynamic deadline,
declared rejection, and isolation without claiming cleanup. B additionally tests
duplicate/late/overwritten receipts, reentry, mutation, world failure, cleanup
before `S`, and deadline progress during bigint work. Making B's strength common
requires an explicit new contract. HTTP/SQL/UI still need Resources lifecycle.

## Questions for architecture review

1. Should generated modules expose only a validated root embedding API, with
   direct exported functions classified as private trusted ABI?
2. For public ingress, which values copy, transfer, validate recursively, or
   remain opaque, especially `Seq`, `Bytes`, branded strings, decimals, variants,
   and callbacks?
3. Which B05 internal ABI should be implemented: uniform completion-returning
   calls or a private frame machine? Which ownership/fault guarantees are required
   of both, and which remain proposal-specific?
4. Independently, what should the JS root-completion facade be, and which party
   owns root/world creation and failure disposal?
5. Should join product patterns be branch-labeled or positional? Show how a
   branch reorder changes diagnostics, canonical identity, and migration.
6. If proposal B's settlement strength is chosen, what exact trusted mechanism
   makes `S` honest for the first real host customer, and how does it progress
   while native bigint work is busy? If A is chosen, what does a timeout certify
   about any still-running operation?
7. Which Resources design supplies acquisition/use/release, cancellation,
   deadlines, shared protocol authority, and cleanup before HTTP/SQL/UI are
   called unblocked?

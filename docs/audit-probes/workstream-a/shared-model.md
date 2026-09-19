# Shared Can model for every replacement review

Research baseline: `8bbcf13b33c74d26bc7bb0f9b72fba0d5415d70b`, 2026-09-19.
This is a description of inspected implementation plus explicitly labeled design
questions, not a replacement language specification. The original review at
`8312d85` is withdrawn. Its choices and probabilities are excluded from the new
requests. Paths identify evidence for the human auditor; all facts needed for a
question must be present in the request itself. The reviewer cannot inspect the
repository, research links, execute programs, or remember another request.

## Purpose and authority

Can is a small contract-first language compiled by a Go compiler to TypeScript.
Its stated target is AI agents reading, writing and verifying programs. Explicit
interfaces, exact numerics, nominal values, producer-owned declared errors,
exhaustive decisions, executable decision tables, and bounded proof obligations
are central goals. A short spelling alone is not an improvement measurement.
There are zero external users: preserving old source, TS layouts, ABI or goldens
is not required. Migration still costs engineering work and must preserve chosen
semantic guarantees. There are no measured agent usability results in this audit.

Current non-goals/rules include no null/undefined source values, implicit numeric
conversion, operator overloading, default arguments, hidden exception propagation
or early return, and eager evaluation. Current policy favors explicit annotations
and canonical syntax; the requested review deliberately reopens some of those
surface rules. Any changed policy must be recorded as a proposal with obligations,
not described as already implemented. Source immutability does not establish
ownership of arbitrary JavaScript objects supplied by an embedding host.

## Implemented source and target model

Functions declare parameters, one success type, `rev N`, `emits [...]`, tests,
optional cell capabilities/contracts, and an expression body. Named calls are
outcome operations in `match call f(...)`; indirect calls use `match invoke cb
with input`. There is no general statement/block `let`. Chains and forwarding
cover some sequencing. Errors have qualified names and typed payloads; they are
returned, not thrown. Ordinary call dispatch covers the callee's finite declared
error set, including errors it need not actually produce. There is no generic
error-row parameter or first-class completed `Outcome<T,E>` type today.

Records and variants have nominal source types by global name, with the F02
declaration-identity defect described below. A brand is currently a nominal string
owned by its declaring file identity; sealing/disclosure has authority rules beyond value
shape. Records, Bytes and sequences are data values; callable containment has
additional restrictions. `int` is exact arbitrary-precision integer; `dec` is
exact finite decimal, no binary floating point or automatic int/dec conversion.
Integer `+`, `-`, `*` are exact; `/` and `%` use Euclidean division with loud
zero-divisor faults. Decimal `+`, `-`, `*` are exact; source decimal `/` and `%`
are rejected. Decimal division library operations instead declare their errors.
Current literals include `7`, `d"12.50"`, raw `"text"`, escaped `e"line\n"`,
`Seq<int>[1, 2]`. Canonical decimals always have a dot, trim leading integer
zeros and trailing fractional zeros, retain at least one digit on either side,
and fold signed zero to `"0.0"`: `d"0012.500"` stores `"12.5"`, while
`d"2.000"` stores `"2.0"`. Decimal canonicalization changes representation, not value;
there is no implicit Unicode normalization. Strings use Unicode scalar semantics.

Current successes depend on their declared type: record successes flatten fields,
other supported successes carry a `value` field. For a two-field record Pair,
`Ok(7, true)` is a success containing its fields; `Ok<Pair>(Pair(7, true))` is
the B11 whole-value source form but lowers to the same flattened success. For
an int result `Ok(7)` lowers to a `value` success. Untyped success binders keep
the old convention; `on Ok<T> result` binds exactly the T for supported success
types. A one-field record is not its field's scalar type. Empty nominal records
exist; a separately specified general Unit type/value is a design question.

Generic functions/records/variants are specialized at compile time, with explicit
type arguments and template-test bindings; type representation still relies
heavily on strings and specialized parsers. Some nested applications/containment
positions are deliberately refused. Successful instance examples do not prove
a generic law. Function values are unary `Fn<A,R,[errors]>` with explicit named
captures from source `fnref` targets. Purity, containment, preconditions and cycle
checks restrict eligible targets. Ordinary source factories can return Fn, but
invocation is still restricted to bare names of Fn-typed parameters of the
enclosing function; field paths and a bound factory result are not invocable
heads. Native
host callback certification is not implemented.

Generated TS represents int with `bigint`, dec and brands with strings, Bytes
with `Uint8Array`, sequences with arrays, records with structural object aliases,
variants with tagged objects, and callables with JavaScript functions. Success
tags are `$can_kind: "ok"`; errors use their qualified names and payload fields.
Public exports currently expose much of the internal layout directly, with no
general generated hostile-input validation, snapshots, or callable provenance
facade. Private layout, a public host calling contract, and wire/persistence
codecs are different design layers even where today's output conflates them.

## Modules, effects, tests, proof and revisions

Files have `mod`, `provides`, pinned `uses`, and module `emits` lists; symbols
use global prefixes such as `seq__map` and `Seq__Value`. Module `emits` is not
enforced as the union promised by older requirements. Function `emits` is checked.
There is no general public/private source declaration split. State cells are
module-private and use declared read/write capabilities propagated through calls.
Host calls are recognized as externs and refused by linked-pure/Fn-target
admission even with empty `emits`. Extern signatures carry no capability row,
so source `effects` cannot describe host authority independently of errors.
Brand minting/disclosure and bytes bridge certificates are additional authority.

Default compile-time tests execute same-file source helpers and script cross-file
source/extern calls. A script exchange pairs expected arguments with a permitted
complete success/error result; missing reached or leftover exchanges fail.
Linked-pure mode and checked source callback execution provide other paths;
linked integration evidence does not replace ordinary unit witness obligations.
Each test gets fresh state. Arbitrary hosts must not execute during compilation.
Moving a pure helper across files can change default testing obligations today.

Each source match arm needs executed or narrowly authorized certified evidence;
CAN4107 has no general proved-unreachable exemption.
Tests are finite examples, not universal contracts. `requires`/`ensures` checks
and universal verification have explicit admission restrictions; authored contracts
state intent, and an optional accepted revision baseline checks interface drift.
Unsupported proof clauses must not be partially treated as proved.
Normal source functions are not automatically universally verified. A source
`pinned` marker plus its entry in an accepted baseline designates trusted
acceptance; A-light records no pinner identity, warns after the clean gate,
and is incomplete under F01. Ordinary proposed rows are not acceptance authority.
The current stdlib contains no `requires`, `ensures`, or `pinned` rows.

Recursion admission is narrow: direct self-recursion under recognized guards
with integer measures. The schemas are unit descent (`p <= 0`, step `p - 1`),
Euclid (`b <= 0`, step `(b, a % b)`), and binary narrowing (`hi - lo <= 1`,
step `(lo, mid)` or `(mid, hi)` where `mid = (lo + hi) / 2`). Recursion occurs
under the false guard arm. Other cycles are refused.
This termination argument does not bound cost, allocation, or JavaScript stack
use. Runtime/embedding faults (such as invalid indexing, resource exhaustion or
host throws) are distinct from the declared domain-error set. A dedicated
embedding-contract fault protocol is proposed in B05, not implemented today.

Revisions check the supplied world's pinned interfaces and accepted baselines.
Bodies are excluded from interface identity. The current resolver does not
implement a historical artifact store or general side-by-side version linker;
older promises that old pins always keep working overstate the current mechanism.

## Current audit evidence and customers

The baseline normal Go, module and grammar checks pass. These do not imply
whole-library composition or strict validity of every generated TS bundle.
F01: pinned callback targets can change without an acceptance warning because
canonicalization omits `fnref` semantics; narrower typed-Ok/pattern/invoke probes
find structural omissions, not a demonstrated persisted proof-cache exploit.
F02: duplicate record identities select the first shape and change with file
order. F03: direct TS calls admit aliases, getters and ordinary effectful JS
callbacks; this is a trust/design gap, not a breach of a promised hostile-host
sandbox. F04: all-stdlib compilation fails on incompatible definitions of
`math.nonterminating_decimal` (ratio int numerator/denominator versus scalar dec
dividend/divisor). F05: fresh JSON plus scalars passes 785 Can rows but strict TS
fails on imported/local `Bool__Value`, `Dec__Value`, `Int__Value`, `Str__Value`
conflicts and widened JSON discriminants;
the repository TS project separately has missing dependency artifact paths.
The topic packets include the exact current reproductions relevant to decisions.

An additional current inspection finding concerns reordered named arguments:
the evaluator evaluates their expressions in written source order before binding
parameter slots; the TS emitter puts expressions into parameter order. Equal
non-faulting values do not establish order parity. Primitive-fault priority needs
an explicit differential check when both expressions can fault. This is separate
from F01-F05 and does not imply general source side effects in argument expressions.

Real customers include sequence/map/set/optional higher-order combinators,
JSON's explicit frame representation, schema migrations, scalar/ratio arithmetic,
text/Bytes/HTML disclosure, quota state and distinct clock types. Some single-field
records are mechanical result carriers, while clock instant/monotonic tick and
nominal authority wrappers have domain meaning worth preserving.

B05 has two unimplemented competing designs: a uniform completion-capable
callable ABI (Promise is one possible lowering) and a private machine/frame model.
Both seek joined child ownership and complete outcome products. They disagree
on source join syntax, singleton policy, host authority, rejection handling and
timeout/settlement guarantees, as well as private execution. The first proposes
dynamic deadlines and declared rejection errors without a separate cleanup bound;
the second proposes trusted timeout/settle bounds, receipts and world faults.
Resources and cancellation remain separate specifications. Public root completion
need not equal private execution representation. No async/resources capability
is claimed implemented.

## Review contract

Evaluate one supplied design decision as advice. Every option must be read with
its costs and preservation obligations. Do not infer approval, formal soundness,
measured usability, or missing implementation from a probability. Select the
defer/insufficient-evidence option when none of the offered directions is
supported. The coordinating auditor will independently accept, reject or leave
recommendations unresolved against the implementation evidence. No new syntax
or ABI becomes an implementation requirement merely from this review.

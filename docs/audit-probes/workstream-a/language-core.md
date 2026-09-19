# Workstream A dossier: language core

This is self-contained input for reviewing repository HEAD, not an approved
redesign. **Implemented** is enforced by compiler/tests; **proposed** is under
review; **historical** was superseded. Withdrawn JEV responses are not evidence.

Can is a small contract-first language that emits TypeScript. Its design values
are explicit authority, exact values, declared finite failures, exhaustive
decisions, executable examples, and honestly scoped proofs
(`README.md:1-6`, `docs/can-language-audit.md:11-23`). It targets AI authors, so
canonical spelling and local diagnosis matter more than terseness
(`REQUIREMENTS.md:8-24`). With no external users, source spelling, private TS
ABI, and old goldens have no compatibility claim. Migrate accepted changes once,
without aliases or legacy modes
(`AGENTS.md`; `docs/audit-probes/TODO.md:12-14`).

## Implemented value and numeric model

There is no null or undefined value and no implicit conversion. Source values
are `str`, unbounded `int`, `bool`, exact `dec`, nominal records, brands and
variants, `Seq<T>`, `Bytes`, and unary `Fn<A,R,[errors...]>` values. Runtime
values have distinct tags for these families (`compiler/eval.go:15-56`). The TS
backend maps `str -> string`, `int -> bigint`, `bool -> boolean`, `dec -> string`,
`Bytes -> Uint8Array`, sequences to arrays, brands to their underlying type, and
function values to closures (`compiler/emit.go:12-15,77-116`). This is a private
backend layout, not a public or wire schema.

Integers use `big.Int` and emit with an `n` suffix. Arithmetic and Euclidean `/`
and `%` are exact; zero division is a runtime/compiler fault, not a declared
error. Decimals come from `d"..."`, store canonical digits, compare as exact
rationals, and emit as strings. Scaled-integer `+`, `-`, and `*` are implemented;
decimal `/` and `%` are rejected (`compiler/parse.go:1003-1224`;
`compiler/eval.go:103-194`;
`compiler/emit.go:551-589,643-724`; `compiler/division_test.go`). For example:

```can
fn std__compare<T>(left: T, right: T) -> Int__Value rev 1
  emits []
  tests
    cmp_dec_gt<T=dec>(d"1.6", d"1.5") => Ok(1)
    cmp_str_lt<T=str>("a", "b") => Ok(-1)
  match left == right, left >= right
    true, _ => Ok(0)
    false, true => Ok(1)
    false, false => Ok(-1)
```

This stamps for `int`, `dec`, and `str`; `bool` needs a separate body because
ordering rejects it (`std/scalars/scalars.can:157-188`). `d"1.50" == d"1.5"` and
giant arithmetic are tested (`std/scalars/scalars.can:203-214`;
`compiler/arith_test.go:29-40,210-215`).

Current literal spellings are deliberately explicit:

* decimal digits are `int`; an exact decimal is `d"12.50"`; bare `12.5` is
  rejected with CAN6001 rather than becoming a binary float;
* ordinary `"..."` is raw. `e"..."` interprets exactly `\"`, `\\`, `\n`, `\r`,
  `\t`, and `\0` (`docs/a/a66-interpreted-strings.md:1-37`);
* booleans are `true` and `false`; `and`, `or`, and `not` are boolean-only;
* a sequence is `Seq<int>[1, 2]`; its element type remains explicit even for
  `Seq<int>[]`. Bytes are built from byte-valued integer sequences.

`and` and `or` are eager. Both operands use strict helpers, so
`false and (1 / 0 == 0)` and `true or (1 / 0 == 0)` still fault. Calls are not
allowed inside boolean operands. Precedence is `or`, `and`, `not`, comparison;
symbols `&&`, `||`, `!` are rejected (`compiler/bool_test.go:59-176,180-317`).
A risky computation must be guarded with `match`, making the skipped branch and
its evidence explicit.

## Implemented success and error outcomes

Every function declares one success type and a finite `emits` set. A call yields
an outcome and must dispatch its success plus every declared error. Errors are
named constructors with typed payloads. There are no exceptions, no first-class
`Outcome<T,E>` data type, and no generic error row. Host/runtime faults such as
integer division by zero are outside the declared error set.

The source and private TS success model currently has three paths:

1. A record success is flattened into the `Ok` envelope. If `Pair` has fields
   `left:int` and `right:int`, its TS success is
   `{ $can_kind: "ok"; left: bigint; right: bigint }`.
2. A primitive, brand, variant, `Seq`, `Bytes`, or `Fn` success has one `value`
   field, such as `{ $can_kind: "ok"; value: boolean }`.
3. B11's `Ok<T>(wholeValue)` and `on Ok<T> binder` bind `T` in source but retain
   path 1 or 2 in the ABI. A record is evaluated once, then projected flat.

The compiler rule is short and decisive (`compiler/result.go:12-60`):

```go
func valueSuccess(ret string, nominalValue bool) bool {
    _, seq := seqElemName(ret)
    _, _, _, fn := fnTypeShape(ret)
    return nominalValue || scalarSuccess(ret) || ret == "Bytes" || seq || fn
}
func successFields(ret string, records map[string][][2]string,
                   nominalValue bool) ([][2]string, bool) {
    if valueSuccess(ret, nominalValue) { return [][2]string{{"value", ret}}, true }
    fields, ok := records[ret]
    return fields, ok
}
```

Untyped `on Ok r` binds a value envelope (`r.value`) or a record payload
(`r.left`); typed `on Ok<T> r` binds `T`. Typed `Ok<T>` takes exactly one value of
the enclosing success type and is an outcome boundary, not data
(`compiler/success_value.go:8-47,75-117`). `$can_kind` is unspellable in Can
(`compiler/emit.go:169-204`). `Ok()` is a flat empty record; Unit does not exist.

An actual generic whole-value source exercises records, scalars, and sequences:

```can
fn success__id<T>(value: T) -> T rev 1
  emits []
  tests
    pair<T=Success__Pair>(Success__Pair(7, true)) => Ok(7, true)
    seq<T=Seq<int>>(Seq<int>[1, 2]) => Ok(Seq<int>[1, 2])
  Ok<T>(value)
```

See `sketches/success-values/success.can:31-44`. A callback returned this way
cannot be invoked at that binding site; it must be passed to a consumer with an
`Fn` parameter (`sketches/success-values/use.can:23-33,54-72`).

## Implemented calls, sequencing, and patterns

A named application is legal only in `match call f(args)` or special elaborated
syntax. Bodies are single expressions; there is no `let`. Foreign Can or extern
calls are pinned in `uses`, and tests supply named `given ... exchange` scripts.
Outcome arms are explicit and exhaustive (`compiler/check.go:731-790`).

One argument binder makes positional argument `i` bind parameter `i`, names bind
by name, and every parameter occur once (`compiler/bind.go:5-52`). The evaluator
evaluates reordered named arguments in source order, then assigns parameter slots
(`compiler/eval.go:1046-1069`). The emitter instead writes expressions in
parameter order (`compiler/emit.go:877-906`). Existing tests use constants: they
prove `subtract(right = 2, left = 9)` means 7 and emits `subtract(9n, 2n)`, but do
not prove evaluation-order parity (`compiler/bind_test.go:57-105`). The intended
order needs a decision and a fix for faulting expressions and before any future
effectful arguments. A current-HEAD reproduction with only well-typed values is
in `docs/audit-probes/workstream-a/evidence-argument-order.txt`: for source
`right = xs[index], left = 1 / divisor` and runtime `xs=[]`, `index=0`,
`divisor=0`, the evaluator reports the source-first sequence fault while emitted
TS reports parameter-first division. Non-faulting values return identically; the
probe shows neither side effects nor a normal result difference. Mixed calls are accepted,
but a positional argument after a named one is rejected
(`compiler/test_args_test.go:13-29`). Constructors use the same positional/named
idea. The linter nevertheless reports an in-declaration-order label as CAN3410
and its CLI exits nonzero, recommending positional spelling
(`compiler/lint.go:12-21,213-280`; `docs/can-idioms.md:225-236`).

Total calls still require a match ladder. `match chain` is sugar for nested call
matches with a success binder and optional pure guard per step, one `then`, and a
shared `else`; elaboration manufactures an arm for every declared error. Error
payloads are unavailable in the shared `else` (`compiler/chain.go:3-26,125-224,
227-310,367-414`). A real test is:

```can
match chain
  call chain__is_big(value) as b when b.value
    given
      big_odd => exchange args (value = 11) outcome Ok(true)
  call chain__is_odd(value) as o when o.value
    given
      big_odd => exchange args (value = 11) outcome Ok(true)
  then Ok(value)
  else chain.too_small(value)
```

This elaborates and executes as nested matches (`compiler/chain_test.go:70-105`).
`forward call f(args)` is a separate same-file-only shorthand expanded to a full
outcome match; its late-parsed representation still has generic-stamping gaps
(`compiler/forwardcall.go:3-89`).

Patterns are three separate languages:

* value matches admit boolean, string and integer constants, closed integer
  ranges, named constants, `_`, and `|` or-patterns. Multi-scrutinee values are
  evaluated eagerly from left to right; coverage and shadowing respect first
  match (`compiler/parse.go:2553-2668`; `compiler/eval.go:2512-2700`);
* variants use `on Case<T> binder` and bind the case payload as a whole; there is
  no nested constructor destructuring;
* call outcomes use `on Ok binder`, optional `on Ok<T> binder`, and one
  `on error.kind binder` per emitted error.

Single-boolean matches need both `true` and `false`, but a wildcard is valid in a
covered cell of a multi-scrutinee table. The stdlib itself uses `true, _` and
`false, _` (`std/scalars/scalars.can:173-188`), contradicting the broader wording
in `docs/can-idioms.md:30-45`. Variant identity also has a real limit:
`qualifyCase` derives a case from the domain prefix before `__`, so same-named
cases in two parent variants in one domain collide (`compiler/parse.go:146-161`;
`docs/b06-generic-variants.md:22-25`).

## Implemented contextual typing boundary

Public function signatures, record fields, generic parameters, and error sets
are explicit. Generic calls, constructors, patterns, and test rows supply all
type arguments. `Seq<T>` literals state `T`, including empty sequences. The
compiler does exact expected-type checking at a declared return boundary, but it
does not infer omitted generic applications or signatures from context or tests.
There are no locals to infer today. `REQUIREMENTS.md:26-39` summarizes this as
“no type inference”; the audit's bidirectional rule is **proposed**, not current.

Types are only partly compositional: declarations largely store strings and
helpers reparse `Seq` and `Fn` (`compiler/parse.go:291-321`;
`compiler/types.go:156-246`). Reliable omission needs structured identity,
substitution, nesting, and error/effect rows.

## Implemented generics, function values, and recursion

Generics use explicit compile-time monomorphization. Records, variants, functions,
and some nested fields are stamped. Parameters occur in signatures; applications
provide concrete arguments; direct recursive calls repeat their parameters
(`compiler/expand.go:261-337,580-760`). There are no kinds, generic constraints,
error parameters, or inferred laws: operators are checked per stamp. Nesting
limits are position-specific. The stdlib implements `Map<K,V>` as a sequence of
`Map__Pair<K,V>` records (`std/map/map.can:21-28,34-80`), while call type
arguments and some variant/sequence paths remain shallow
(`compiler/expand.go:485-508`).

`Fn<A,R,[errors...]>` is unary with exact, sorted concrete errors.
`fnref target(capture = value)` uses declaration-ordered named captures, leaves
one parameter unbound, and snapshots values. Its target is source-defined,
linked-pure, data-only in input/result, precondition-free, and pinned if foreign
(`compiler/bind.go:55-94`; `compiler/types.go:311-437,511-586`). Invocation is:

```can
fn use__run(cb: Fn<int, Ops__Quot, [ops.zero_divisor]>, n: int) -> Ops__Quot rev 1
  emits []
  tests
    run(fnref ops__divmod(divisor = 3), 7) => Ok(2, 1)
  match invoke cb with n
    on Ok r => Ok(r.quotient, r.remainder)
    on ops.zero_divisor _ => Ok(0, 0)
```

This is actual source (`sketches/fn-callback/use.can:14-27`). Despite the general
function type, `match invoke` resolves only a bare name that is an `Fn` parameter
of the enclosing function; a field, local success binder, or returned expression
cannot be the invocation head (`compiler/invoke.go:27-70`). Indirect cycles are
rejected.

Recursion is direct self-recursion under three syntactic termination schemas:
`decreases p` with `p <= 0` and `p - 1`; Euclid over two measures; or prescribed
midpoint narrowing. Mutual/cross-file cycles and near-equivalent unrecognized
forms are rejected (`compiler/loop_test.go`; `docs/can-idioms.md:82-96`). This is
syntactic admission, not a general termination proof. Example:

```can
fn fib__from(n: int, a: int, b: int) -> Fib__Value rev 1
  decreases n
  emits []
  match n <= 0
    true => Ok(value = a)
    false => match call fib__from(n - 1, b, a + b)
      on Ok r => Ok(value = r.value)
```

(`sketches/fibonacci/fib.can:31-41`). Negative measures take the base arm.

## Documentation contradictions and live customers

The following must not be blended into a fictional single current spec:

* `std/seq/seq.can:5-7` and `docs/a/a36-seq-typed-construction.md:127-132`
  say bare `Seq` returns are rejected. B10 later implemented them, and
  `compiler/seq_s1_test.go:470-487` accepts `-> Seq<str>`. The stdlib still uses
  `Seq__Values<T>` carriers, so its comment is stale and migration is unfinished.
* `docs/b06-generic-variants.md:50-64` preserves an older “bare variants rejected”
  boundary while acknowledging B07 superseded it. `docs/a/a93-generics-design.md`
  also predates generic variants and bare generic returns.
* `docs/b01-type-syntax-redesign.md:96-105` proposed `[T]`; it was never
  implemented. Current source and the new audit both retain `Seq<T>`.
* `REQUIREMENTS.md:31` states blanket explicit typing. Audit S05 proposes limited
  deterministic context, but no compiler change implements it.
* B10/B11 intentionally preserved older TS layouts and source modes. Under the
  present zero-user constraint, those compatibility choices are historical
  inputs, not reasons to retain the layers.
* `docs/can-idioms.md:30-45` overstates the boolean wildcard ban; stdlib product
  tables are counterexamples. Its label advice matches the linter, while audit
  S06 proposes preserving useful labels.

Live customers are `std/scalars` (numeric stamps/product patterns), `std/map`
(generic records nested through `Seq`), and `std/seq` (function values/carrier
workarounds). Migrate them and rerun executable rows, not only parser goldens.

## Decision packet: one success value

**A — uniform `Ok(value)` (audit proposal).** A producer returns exactly one
value of declared type `T`; `on Ok(result)` binds `T`. A record is constructed as
data, and no-payload success uses a real `Unit`. Consequence: source, checker, and
private TS share one outcome algebra. Obligations: specify Unit, contracts/tests/
scripts/callbacks, `value` fields, empty records, nested outcomes, and public/wire
mapping. Migration must resolve types while
rewriting `.value`; global text replacement would corrupt real data fields.

```can
// current record success and envelope binder
Ok(left, right)                 on Ok pair => pair.left
// proposed
Ok(Pair(left, right))           on Ok(pair) => pair.left
```

**B — keep the typed dual.** Preserve flat record envelopes, `value` envelopes,
and `Ok<T>` as an opt-in view. This minimizes churn but makes return type select
construction, binding, TS layout, and generic behavior; all need a permanent matrix.

**C — raw values for total functions.** Return `T` directly when `emits []` and
an outcome otherwise. Total code is lighter, but adding one error changes every
caller and function type; coercion/forwarding must be specified.

Because compatibility is not required, evaluate these by semantic regularity and
evidence obligations, not migration minimization.

## Decision packet: application and sequencing

**A — one explicit `call` plus immutable expression blocks (audit proposal).**
`call target(args)` accepts either a resolved named function or any typed callable
expression. The typed AST retains static/indirect identity for pins, authority,
cycles, scripts, and emission. A total call may bind irrefutably:

```can
// current
match call transform(input)
  on Ok r => Ok(Option__Some(r.value))
// proposed; legal only when transform emits []
let Ok(mapped) = call transform(input)
Ok(Option.Some(mapped))
```

Fallible calls remain exhaustive. Obligations: source-order/single evaluation,
signature/effects for every target form, test attribution, capture authority,
preconditions, and cycles. Equivalent tests must precede removing ladders,
`match chain`, and capture-only helpers.

**B — one bare application syntax.** `target(args)` has the same typed semantics
as A. It is terse but weakens the visible authority marker; tools still distinguish
construction and static/indirect calls without overload search.

**C — retain split call/invoke plus chain.** This keeps authority visible and
limits implementation work, but lexical storage location continues to decide
which grammar and AST apply; a returned callable remains unusable without a
consumer helper. The chain's shared `else` also cannot inspect error payloads.

**D — implicit propagation/early return.** This shortens fallible pipelines but
hides the leaving outcome unless transfer is visible. It needs error-row mapping
rules first and must not be smuggled in as `let Ok(...)`.

## Decision packet: patterns and contextual typing

For patterns, **A** is the audit's constructor-shaped language:
`on Ok(x)`, `on Option.Some(x)`, structural record/tuple subpatterns, constants,
ranges, `_`, and `|` where the domain supports them. It must preserve eager
scrutinee evaluation, first-match priority, coverage/overlap diagnostics, and
closed declared-error handling; variant identity must include the parent type.
Before/after: `on Ok<T> result` becomes `on Ok(result)` with `result:T`;
`on Map__Some payload` can become `on Map.Some(key, value)`. **B** retains the
three current systems and their contextual binders without solving case collision.
A catch-all error arm would let a new declared error bypass review.

For local/contextual types, **A** allows bidirectional omission only when an
expected type and lexical bindings determine one result. Public signatures,
error/effect authority, ambiguous empty collections, polymorphic references, and
generic laws remain explicit. Example: under an expected `Seq<int>`, `Seq[1,2]`
could elaborate uniquely, while an uncontextualized `Seq[]` remains an error.
Constructor and pattern payloads need not repeat a known `<T>`. Obligations are a
structured type algebra and deterministic elaboration. **B** keeps every
application explicit; it is predictable but perpetuates
`Ok<T>` and generic repetition whose type is already fixed. **C**, global inference
from bodies or tests, risks unstable APIs and mistaking sampled stamps for laws;
it conflicts with declared authority.

## Decision packet: collections, literals, booleans, and labels

**Collections.** Candidate A keeps `Seq<T>` as an ordinary type application and
fixes compositional nesting; source literals may remain `Seq<T>[...]` or omit `T`
only under the deterministic contextual rule above. Candidate B uses a bracket
type such as `[T]`; it saves characters but creates collection-specific grammar
without changing semantics. Candidate C rejects nesting; `Map<K,V>` already
demonstrates that real generic composition needs it. A migration must cover empty
literals, nested variants/records, type substitution, TS emission, and stdlib
carrier removal.

**Literals.** Candidate A uses ordinary `12.50` for exact `dec` (there is no float)
and ordinary escaped strings, with an explicit raw form. It improves familiarity
but must preserve exact source-to-canonical value rules, negative-token parsing,
the six escape meanings, Unicode byte behavior, diagnostics, and formatting.
Candidate B retains `d"12.50"`, raw `"..."`, and `e"..."`; it is unambiguous but
unusual. Candidate C makes bare decimals binary floats; that adds rounding,
comparison, serialization, and proof obligations absent today and should not be
treated as mere spelling. Before/after for A: `d"0.10" + d"0.20"` becomes
`0.10 + 0.20` while remaining exact and canonically equal to `0.3`.

**Booleans.** Candidate A keeps the implemented eager `and`/`or`; faults and
evidence are never skipped, and guarded `match` expresses conditional evaluation.
Candidate B makes them short-circuit control flow; it matches common languages but
changes faults, call/test obligations, and branch evidence. Candidate C offers
both strict and short-circuit families, increasing near-duplicate syntax and agent
choice. Decide one semantics through edit/error trials; short-circuit operators
must be specified as control flow, not as lazy function arguments.

**Argument labels.** Candidate A preserves every written label and uses the
existing single binder everywhere; `f(left = 1, right = 2)` stays accepted without
CAN3410. Labels document meaning and permit safe reordering, at the cost of a
canonical formatter choice. Candidate B deletes in-position labels, matching the
current lint but losing documentation and making reordering change style. Candidate
C requires all labels; it is maximally explicit but noisy for obvious unary and
mathematical calls. Constructors, calls, tests, exchanges, and `fnref` captures
must converge on a documented rule; current captures/exchanges already require
names while ordinary calls do not.

## Review constraints and acceptance evidence

Judge semantics separately from current implementation cost, private TS layout,
and future public/wire schema. Migrate once. Acceptance needs checker/evaluator/emitter parity plus focused
matrices for every success type, total and fallible calls, named argument order,
same-named variant cases, ambiguous type contexts, empty/nested sequences, exact
numeric corners, eager or short-circuit faults, function factories, and each
recursion schema. Existing tests establish examples; they do not prove generic
laws, termination beyond the admitted schemas, or correctness of a future host
boundary.

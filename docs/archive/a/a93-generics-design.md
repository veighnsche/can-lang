# a93: explicit generics — decided, G1 + compare pilot landed

Status: decided 2026-09-18. Order-8 part 1 of 3 (this
doc: generics; separate designs: function values,
first-class outcomes). Parent: `docs/archive/ASTRA_STDLIB.md`
§6 order 8 ("no inferred type or error parameters").
Verdict (principal, same day): Q1a per-instance
stamps, Q2b select+compare pilot (8→2). G1 implements.
Pilot outcome (same day, amendments 6–8): compare
3→1 (bool stays monomorphic), select reverted on the
v0 bare-return emit limit — generic `-> T` waits on
the outcomes design.

## 1. Ground facts (repo-verified)

- Types are strings: `Params [][2]string`, `Ret string`,
  `Seq<T>` by string convention (`seqElemName`).
  No variables, no substitution, no unification.
- Runtime is type-erased: `Value` is structural
  (`Kind`/`Dict`/`Arr`+`Elem`); brands erase. Expansion
  is purely check/emit-time — no runtime type passing.
- The one generic-like facility, `Seq<T>`, is handled
  structurally per site (literals, indexing, emit).
- Standing rejections: inferred generic parameters
  (`docs/archive/a/ASTRA_FSHARP_BORROW.md`); generics in the
  variant first cut (a72, "start without ... generics").
- Pilot surface is free: zero in-repo callers of
  `std__select__*` / `std__compare__*` outside their
  own file — unification needs no call-site migration.

## 2. Design: explicit monomorphic expansion

No inference anywhere. A generic decl names its type
parameters; every use names complete type arguments;
the compiler stamps one monomorphic copy per distinct
instantiation and checks, terminates, and emits each
as ordinary code. The template itself gets
well-formedness checks only (§4) — every operator in
every body is verified inside some stamped copy, so no
constraint/bound language is needed in any slice here.
(Consequence: `compare<T>` unifies in slice 1 despite
using `<` — each stamp checks its own operators.)

## 3. Syntax (proposed, not parsed)

```can
fn std__select<T>(condition: bool, when_true: T, when_false: T) -> T rev 1
  emits []
  tests
    pick_true<T=bool>(true, true, false) => Ok(true)

type Box<T> rev 1 (
  value: T
)
```

- Decl: `<P1, P2>` after the name, on `fn` (slice G1)
  and `type` (slice G2). `variant`/`brand`/`error`
  parameterization is not proposed.
- Params: `[A-Z][A-Za-z0-9]*`, must not collide with
  any declared type/record/brand/variant/error name in
  the program. Every param must appear in the
  signature (params or ret); unused params rejected.
- Instantiation, always complete and explicit: calls
  `select<str>(...)`, annotations `Box<str>`, test rows
  `name<T=str>(...)`. Partial application, defaults,
  and inference are rejected, not defaulted.
- Type arguments in G1/G2: any known monomorphic type
  (base, `Bytes`, record, brand, variant, `Seq` over
  those). Nested instantiation (`Box<Box<str>>`) and
  generic-typed `state` cells are rejected in these
  slices (expansion-termination and store-identity
  arguments owed separately).
- `emits` stays concrete kinds (no error-set params —
  outcome-generic fns await that separate design).
- Recursion: a generic fn's self-calls must repeat the
  caller's own type arguments exactly (checked at
  expansion); cross-instantiation recursion is
  rejected — it risks unbounded stamps.

## 4. Checking

- Template well-formedness at parse/check: params
  declared before use, all used in the signature, no
  collisions, concrete emits, instantiation syntax
  complete at every use site (calls, annotations,
  rows, `given` exchanges naming generic callees).
- Per-instance full checking: substitute args through
  the signature and body, then run the ordinary
  phases (types, exhaustiveness, termination,
  contracts-adjacent gates) on each stamped copy.
  Termination is per stamp — no new argument shape.
- Rows: same rule as all fns (rows required), each
  row pinning one complete instantiation. An
  uninstantiated generic (no rows, no calls) is
  rejected as an unchecked template.
- `uses` pins the base name + rev (`select@1`);
  instantiation lives in the call, never the pin.
  Mangled names never appear in source.

## 5. Emit (Q1)

- (a, recommended) Per-instance stamps: each stamped
  copy emits as its own TS function under a
  deterministic mangled name. What runs is what was
  checked; existing golden/parity gates keep their
  meaning. Mangling: base + (`$T$` + arg) per arg,
  with `<`/`>`/`,` escaped as `$L$`/`$G$`/`$C$`
  (`select<str>` → `select$T$str`). `$` never occurs
  in source identifiers (`\w+`), so stamps cannot
  collide with declared names.
- (b) Single TS-native generic per decl
  (`function select<T>(...)`). Less emit code, but the
  one artifact the checker never verified as a unit
  becomes the thing that runs; per-shape golden/parity
  evidence would need re-scoping.

## 6. Slices

- **G1: generic fns.** Parse, well-formedness,
  expansion, per-instance check/emit, `uses`
  base-name resolution (modcheck + compiler agree),
  editor highlighting for `<>`, golden + linked +
  parity for the pilot, LSP degrades gracefully.
- **G2: generic record types** (`Box<T>`; the shape
  `Schema<T>` will later reuse — no schema semantics
  here). Same gates.
- **Pilot (Q2):** unify `std__select__*` (4→1),
  optionally plus `std__compare__*` (4→1), in
  `scalars.can`, replacing the monomorphic fns (zero
  callers; no compatibility aliases per a89
  precedent). Decision tables move onto the generic
  with per-row instantiations.

## 7. Non-goals (all slices here)

Inference, defaults, partial application, nested
instantiation, generic variants/brands/errors/state,
error-set parameters, function values, first-class
outcomes, record traversal, TS-native generic emit
(unless Q1b), any claim the §1.5 row is delivered.

## Q1 — Emit strategy (verdict needed)

(a) per-instance stamps [recommended] or (b) single
TS-native generic per decl. Decides what the parity
harness executes and how goldens pin generic shapes.

## Q2 — Pilot scope (verdict needed)

(a) `select` family only (4→1) [recommended: smallest
proof] or (b) `select` + `compare` (8→2, same
machinery, stronger demonstration).

## Implementation amendments (G1 expansion, same day)

Found while building `compiler/expand.go`; all within
the verdict unless noted:

1. **Per-stamp coverage.** Decision-table coverage
   (CAN4107) applies per stamp: rows must cover every
   arm of every instance. Row routing splits evidence,
   so a two-arm generic needs both arms rowed per
   pinned shape. Unification preserves this naturally
   (moved rows already cover their stamp).
2. **Rowless stamps keep CAN3301.** An instance nobody
   rows is missing evidence for one shape; the message
   names base and args (`generic m__inner at <int>
   ships no tests`), not the mangled stamp. Pass-through
   composition works when the callee carries rows; the
   expansion fixpoint stays load-bearing for stamping
   (and precise erroring of) every called instance.
3. **Charset disjointness.** Params forbid `_` while
   every type name requires `__`: collision with the
   type namespaces is grammatically impossible, so no
   check exists for it (only value-param shadowing is
   rejected).
4. **`-> T` shape limits (v0, not generics).** One body
   text cannot serve bare and record instances (`Ok`
   constructs into record returns, identities bare
   ones), and callers cannot consume bare returns
   (`on Ok v` misbinds them — probed, pre-existing).
   The bare-caller gap belongs to the outcomes design,
   not G1. Consequence for Q2: `compare` (concrete
   record return) is certain; `select` is attempted in
   the same slice iff its all-bare stamps go green on
   rows alone (zero in-repo callers either way).
   Amendment verdict (principal, same day): proceed
   compare-first, select iff green.
5. **Headers expand with calls.** Expansion rewrites
   `provides` (base → sorted stamps) and per-module
   `uses` (base pin → used mangled pins, same rev) in
   AST only; source text keeps base names for
   text-level tools. Dead base pins warn CAN3401 like
   dead monomorphic pins. Missing-pin errors name the
   base, never the stamp.

## Pilot amendments (G1 gates + compare pilot, same day)

Found while unifying `std__compare__*` in
`std/scalars/scalars.can`; all within the verdict:

6. **Identity splice.** Stamping located each template
   by its pre-expansion decl index, which goes stale
   the moment a same-module sibling stamps first (map
   order) — stranding a template or dropping a stamp.
   The splice now locates the template by pointer, so
   the end state is order-independent; pinned by
   `TestGenericTwoTemplatesOneModule`. Single-generic
   modules never showed it.
7. **Compare is 3→1, not 4→1.** `>=` refuses bool
   operands (`cannot order bool with bool`), so no
   single body text serves all four instances;
   `std__compare__bool` keeps its monomorphic fn
   (`match left == right, left`). The generic carries
   the int/dec/str rows; golden + linked + parity pin
   all three stamps plus the bool fn.
8. **Select reverted (iff-green fails).** All four
   select instances are bare (`-> T`), and emit
   requires a declared record return
   (`declaredOkShape`: `returns unknown type bool`).
   A monomorphic `-> bool` fn fails identically, so
   this is a pre-existing v0 emit limit, not a
   generics defect; the checker accepts bare returns
   (identity) but emit cannot shape them. The bare
   wire shape belongs to the outcomes design (per
   amendment 4); the four monomorphic select fns are
   restored untouched. Consequence: generic `-> T`
   pilots stay blocked until outcomes lands — only
   concrete-record returns (`-> R`) unify today.

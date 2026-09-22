# a98: generic record types (G2) — decided, G2 + ratio pilot landed

Status: decided 2026-09-18. Order-8 part 1 of 3,
second slice (G1 generic fns landed per a93; this doc:
generic records; the shape `Schema<T>` will later
reuse — no schema semantics here). Parent:
`docs/a/a93-generics-design.md` (method + verdicts) and
`docs/ASTRA_STDLIB.md` §6 order 8.
Verdict (principal, same day): Q1a `Ratio__Value<T>`
3→1 pilot (+ first ratio golden/linked/parity), Q2a
unmentioned generic types rejected.
Outcome (same day): G2 landed per verdict — two-pass
expansion (fn, then type), `Ratio__Value<T>` 3→1 with
golden + linked + parity, all G1-mirroring gates green.
Findings: (1) type stamps needed the same naming
exemption as fn stamps (`m.GenericBase`); (2) the
chain-else gate went precise (`w<x>(` shape for all
fns, replacing the coarse `call `+`<` check and closing
a latent mono-fn gap); (3) the editor `<>` pattern
dropped its lookbehind so gramcheck (Go regexp) pins it
in-repo — the only cost is cosmetic highlighting of
unspaced `a<b>c` comparisons (zero corpus occurrences).

## 1. Ground facts (repo-verified)

- Types are strings in the AST (`Params[][1]`, `Ret`,
  `Fields[][1]`, `Ctor`, `seqlit.Elem`). The checker
  resolves them through the recs table
  (`compiler/types.go: newTycker`, `knownType`,
  `resolveRef`), construction through `checkCtor`, and
  emit through `recordShapes`/`tsTypeB`. Stamped
  records (`Box$T$str`) flow through all of it
  untouched — G2 needs no checker, eval, or emit rule.
- Patterns never destructure records (Pattern kinds:
  wild/bool/str/int/range/const/variant — `parse.go:67`).
  Records are consumed by projection (`v.field`) and
  whole-binders (`on Ok v`) only. No pattern work.
- Ctor heads are `[\w.]+` (`parse.go:1399`): `Box<str>(...)`
  does not parse today. Corpus scan: zero `w<x>(`
  expressions outside `Seq<...>[` (separately claimed)
  and G1's own decl line — claiming `Word<balanced>(`
  as a generic construction reinterprets no real code.
- Splitter precedent: `Seq<...>` heads are atoms in
  `findTop`/`findTopWord` (`parse.go:634,681,744`).
  G1's `isCallGenericHead` is `call`-keyword-gated, so
  G1 never needed the general form; G2 construction
  does (`Box<str>(v) == x` must not split at `<`).
- `substType` is shallow (bare param + `Seq<param>`
  only — `expand.go:647`); `closedArg` already encodes
  the argument rule (bare/closed/Seq, never nested —
  `expand.go:361`). Both are reused, not reinvented.
- Type references need no `uses` pins (only calls and
  consts have not-in-uses enforcement); pins, when
  present, must resolve with exact rev (`check.go:329`).
  Types resolve world-wide by name.
- `Seq<MonoRecord>` checks and emits today (probed:
  `M__I[]`), so `Seq<Box<str>>` follows by uniformity.
- Lint parses pre-expansion text (`lint.go:89`) with no
  type-name validation; G1 fns pass it unhandled.
  Generic types get a tolerance probe, not new rules.
- `std/ratio/ratio.can` (235 lines) is self-contained
  (no file outside it mentions `Ratio__`), compiles
  clean, and its golden is fresh but unpinned by any
  test. It declares `Ratio__Int/Bool/Dec`, three
  identical `value: <prim>` shapes returned across
  eleven fns — all constructions positional `Ok(...)`,
  all consumptions `.value` projections.

## 2. Design: two-pass monomorphic expansion

Mirror a93 §2. `expandGenerics` gains a type phase; order
is fixed: **fn pass, then type pass** — `Box<T>` inside a
generic fn is unstampable until the fn stamps. Each phase
keeps its own fixpoint (fn pass-through per a93 exists;
type pass-through `A<T>` → field `B<T>` iterates
`(base, args)` pairs to saturation). Nested user-generic
arguments (`Box<Pair<str>>`, `Seq<Box<str>>` as an *arg*)
are rejected by the existing `closedArg` shape; bare
mentions (`Box` with no args) fail downstream as unknown
types, and bare bases are excluded from the call-arg
registry so `f<Box>` fails at expansion naming the base.

A mention is any `Base<args>` in: fn params/rets, record,
variant-case, and error fields, const types and values,
extern sigs, ctor heads, and `seqlit` elems (including
`Seq`-wrapped). State cells reject mentions (base-only
rule, named at expansion, never a stamp leak). Chain-else
text extends G1's gate: closed `Name<...>` mentions are
rejected there (late-parsed regions never rewrite).

Template well-formedness mirrors fns: every param used in
≥1 field, no collisions (the a93 charset rule already
separates params from type names; field names share no
scope with params, so no shadow rule). Self-mention
(`L<T>` fielding `L<T>`) is always rejected — types have
no termination argument, unlike fn self-calls. Mutual
mention terminates by the known-pair fixpoint, and
`checkRecordCycles` sees stamps as ordinary records.

Stamps substitute fields, take the template rev, and
replace the template by identity splice (a93 amendment 6
applies to both phases). `provides` rewrites base →
sorted stamps; `uses` pins on generic types rewrite base
→ used stamps exactly like fn pins — this is forced, not
a choice: modcheck resolves base→base in source, so
anything but mirror-rewriting breaks text/compiler
agreement. Downstream phases see plain records.

## 3. Syntax (proposed, mostly already parses)

- Decl: `type Box<T> rev 1 (` … `value: T` … `)` —
  one `<>` group on the `type` line (params
  `[A-Z][A-Za-z0-9]*`, multi allowed, mirror fns).
- Annotations: `x: Box<str>` — no parse change (type
  strings already carry `Seq<...>`).
- Construction: `Box<str>(value = "a")` and positional
  `Box<str>("a")` — claims `Word<balanced>(` as a
  generic head. Consequence: `w<x>(y)` is always a
  generic head now, never a chained comparison (zero
  corpus occurrences; `w<x>[y]` untouched).
- Atom rule: the `Seq<...>` skip in every splitter
  generalizes to generic heads (`findTop`,
  `findTopWord`, match-scrutinee comma split), via one
  shared helper. `<>` commas inside parens already
  survive; only depth-zero splits need the rule.

## 4. Checking

Per-stamp there is nothing to check: stamps are plain
records, verified at use sites by the existing rules
(construction arity/types, projection, `Ok`-against,
exhaustiveness over values). G2 checking is therefore
template well-formedness (§2) + mention validity
(complete args, known closed types, admitted positions)
+ the existing suite running on stamps. Termination,
contracts, and coverage see stamps as ordinary decls.

## 5. Emit (decided by Q1a precedent, not a question)

Per-instance stamps: `export type Box$T$str = { ... }`;
constructions emit object literals, projections `.f`.
What runs is what was checked; `$` keeps stamps disjoint
from declared names. No TS-native `type Box<T>`.

## 6. Slices

- **G2 machinery:** parse (decl params, ctor-head
  claim, atom rule, `substType` generalization),
  two-pass expansion with fixpoints, header rewriting
  for type decls and pins, chain-else gate extension,
  registry exclusion for bare bases.
- **Gates (mirror G1):** modcheck agreement (no
  change + pinning test), grammar (covered by the G1
  `<>` pattern + the missing gramcheck scope rows),
  LSP same-position expansion (+ test), lint tolerance
  probe (+ test), golden + linked + parity for the
  pilot (Q1).
- **Pilot (Q1):** the only open scope.

## 7. Non-goals (this slice)

Inference, defaults, partial application, nested user
instantiation, generic variants/brands/errors/state,
bare-base mentions, methods, any S3 schema semantics,
and any claim beyond the pilot row.

## Q1 — Pilot scope (verdict needed)

(a) `Ratio__Value<T>` 3→1 in `std/ratio/ratio.can`
[recommended: real unification like the compare pilot
— eleven fns re-annotate, `.value` projections and
`Ok(...)` rows untouched — and it adds the first
golden + linked + parity pinning a live but unpinned
module] or (b) synthetic `Box<T>` + accessors in a
sketch [smallest proof, zero stdlib surface, weaker
evidence: no constructions in rows, no cross-fn
mentions].

## Q2 — Unmentioned generic type (verdict needed)

(a) reject as an unchecked template [recommended:
uniform with generic fns — every generic is
instantiated or deleted; fail-closed] or (b) allow
(uniform with monomorphic dead types, which carry no
check; well-formedness still enforced) [tolerant of
scaffolding, lets templates rot silently].

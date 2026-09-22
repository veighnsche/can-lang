# Prompt: design the a78 gap closures + implementation plan

Paste this into the designer bot with [docs/a/a78-classification-gaps.md](a78-classification-gaps.md)
attached plus the context files below. a78's investigation is
closed (all gaps verified); nothing is designed yet. This
prompt commissions the designs and the build plan. It
authorizes no compiler changes — output is design records
plus a phased plan, one per item.

Context to attach: `REQUIREMENTS.md` (R1–R11 + open
questions), `docs/a/a28-multi-scrutinee-match.md` (the
deferral item 3 reopens), `docs/a61-error-design.md` (removed)
item 2 (item 2 refines it, do not redesign it), `std/html/html.can`
(`html__url__scheme_token`, `html__url__authority`,
`html__attribute__value_from` — the exhibits),
`std/scalars/scalars.can` (`std__dec__round_half_even` —
the `let` exhibit, out of scope, see below),
`compiler/parse.go` (`parsePattern`, operator tables),
`docs/ASTRA_FSHARP_BORROW.md` (only to confirm `let` and
friends live elsewhere — do not design them here).

---

Design closures for the five a78 gaps — named constants,
forward arms (refine a61-2 to plan-ready), range arms,
or-patterns, boolean operators, unary minus — and return
an implementation plan per item. For each item, return:
decision record (options considered, chosen, rejected-
why, with file:line evidence for every claim about
current behavior); grammar delta (productions, one
canonical form — two spellings for one thing is
rejected by R-non-goals); proof-rule delta (how
exhaustiveness, test-per-arm A4107, and termination
interplay — new arm shapes are new coverage
obligations, spell them out); emit sketch (TS shape,
strict evaluation preserved); new diagnostic codes
(frozen-golden style, CLI and LSP share them);
TextMate + gramcheck sample updates (note the
documented dead weight: `when then and reach expect
mock case spec externals |` are grammar fiction —
reclaim or remove deliberately); migration diff sketch
on one named exhibit; test plan (suites, fixtures,
negative probes); risks (which existing theorem could
break and how the plan detects it). Counterexamples
welcome; unresolved items named with the exact missing
source. Rerunnable probes beat assertions.

Hard constraints (violating proposals are rejected,
not revised): single canonical form; no hidden control
flow (early return, exceptions, laziness — R-non-goals
plus strict/eager-only); exhaustive match stays total
with no catch-all; `decreases` proof burden never
lightened; formatter law (whatever is added has one
spelling the future formatter enforces); rev discipline
(R4 — state each item's hash/version semantics
explicitly, including the inertness requirement: a
renamed constant or reformatted arm is not a breaking
change); ai-lock canonicalization must treat new
surface like comments (see a78).

1. Named constants. Decl kind vs evaluable form?
   Providible across files (R2 — anything else fails
   the acceptance test)? Usable in tests, `given`,
   patterns, range bounds (composes with item 3 — do
   const bounds ride v1 or wait)? Types allowed
   (int/str/dec/bool only? records? brands?)?
   Forward references, shadowing, duplicate rules?
   Emit: TS `const`, or inline literal (duplication
   vs dependency — cite the project's standing
   answer)? Acceptance: the ten magic bounds in
   `scheme_token` become greppable names with zero
   call/test overhead, and proposals must cover test
   scope (round-2 finding: payoff is biggest there).
2. Forward arms. a61 item 2 stands — refine, do not
   redesign: exact syntax (name still TBD), binder
   shapes (`e` vs `_`, what binds what), check list
   (identical to a manual identity relay), coverage
   law (arm still exists, A4107 unchanged), emit
   (identical union construction to today's manual
   relay). Draw the line against mapping relays with
   evidence: `html__attribute__id` rebinds
   `html.nul_byte(value = value)` — show why that
   arm must stay manual under your rule. Migration:
   the 12 + 10 + 6 + 7 identity relays across the
   four worker exhibits.
3. Range arms. Reopen the a28 deferral deliberately:
   quote what a28 deferred and state what changed
   (the trigger). Arm syntax (`65..90`? bounds
   inclusive? const bounds v1 or literals-only? —
   depends on item 1's answer). Proof rules: how is
   exhaustiveness over int ranges computed
   (complement? mandatory `_` like string slots?)?
   Overlapping ranges: first-match-wins plus an
   unreachable-arm diagnostic (new code), or overlap
   rejected? Coverage interplay with teeth: is each
   range arm one A4107 obligation, and are boundary
   tests required or merely allowed? Emit shape.
   Migration: both ASCII ladders collapse; state the
   resulting arm/test counts.
4. Or-patterns. Same-RHS multi-pattern: separator
   (`|` is free in can surface and sits orphaned in
   the grammar — justify taking or refusing it)?
   Which pattern kinds combine (ranges? strings?
   variants? mixed — a28 forbids mixed slots, does
   that bind here)? Coverage: one obligation per arm
   or per branch (argue from A4107's purpose, not
   convenience)? Unreachable-branch detection?
   Migration: authority's 4x `prev == "-"` terminator
   arms become one — show it.
5. Boolean operators. Keywords (`and`/`or`/`not`) vs
   symbols (`&&`/`||`/`!`) — one form only, argue
   from the grammar's standing vocabulary and the
   dead-weight cleanup. Precedence table + strict
   left-to-right evaluation. THE decision: strict
   both-sides vs short-circuit. Short-circuit skips
   evaluation, which is laziness (banned) — but
   strict evaluation of `false and (1 / 0)` faults
   loud where short-circuit would not. Calls cannot
   appear as operands (CodeCallOutside stands), so
   operands are pure modulo loud faults: resolve
   against fault-contracts.md explicitly, do not
   hand-wave. Emit (`&&`/`||` with parens?).
   Composition target: stdlib char-class predicates
   (`alpha`/`digit`/`alnum` — absent today) become
   composable; state whether the predicate row rides
   this item or follows.
6. Unary minus. Prefix `-` (and `+`? decide) in the
   expression parser; precedence relative to `*` and
   binary `-`; literal behavior (`-3`, `d"-0.5"`)
   byte-unchanged; `dec` and `int` coverage; emit
   `-x`; kills the `0 - x` idiom. Smallest item —
   keep the design proportional (this one should fit
   in half a page).

Ordering and phasing. Order the five builds with
dependencies (const-before-const-bounds is the known
one; find the rest), each phased: design note →
grammar → parse → check/proof → eval → emit →
diagnostics → TextMate+gramcheck → goldens → stdlib
migration (the named acceptance proofs: two ladders,
`value_from` flatness, relay collapse, magic-bound
removal) → docs amendment (tagged, never silent
rewrite). Per phase: exact test gates
(`go test ./...`, `tsc`, modcheck, gramcheck, new
fixtures) and rollback shape. Confirm the scoped-out
list stays out (guards beyond ranges, multi-call,
higher-order map/fold, record-update, table lookup,
`let` — the last belongs to the F# borrow program,
cite don't duplicate); anything you want back in
needs a consumer and a paragraph.

## Designer-bot output bar

A plan this repo can execute has: no option left
open that a builder would have to invent (every TBD
is a named unresolved item, not a silent default);
every new diagnostic with code, message shape, and
golden fixture; every proof rule with the exact
obligation an agent must satisfy; migration diffs
sketched, not promised; and a risks section that
names which of exhaustiveness, termination,
test-per-arm, rev identity, or emit purity could
break, with the detection gate for each. Match the
house style: short lines, concrete over abstract,
file:line everything.

# a78: gap closures — final design record

Status: decided. This record consolidates the three
`a78-astra-design*.md` drafts (all three read in full;
design3 was newest but none alone suffices) and is the build
order for closing the a78 classification gaps. All three pins
are in-history (`7c33f7c`, `b5525eac`, `e87e5cb` = newest);
line anchors below were re-verified at HEAD `e05b07b`, and
draft anchors that drifted are noted as approximate.

Companion: [a78-classification-gaps.md](a78-classification-gaps.md)
(investigation; proposal direction superseded). Prior art:
a61 item 2 (forward arms), a28 (deferred OR-patterns/guards),
a77 (identity), a80–a82 (verifier: admission, obligations,
activation).

## Build order

Six closures: **constants → forward arms → ranges →
or-patterns → boolean operators → unary minus.** Ranges
depend on constants (v1 bounds are literals or visible
integer constants); or-patterns depend on ranges. Forwarding
keeps a61's design; no new error mechanism. Items graduate
separately, each probe-first with the established gates.
(Design1 ordered unary before boolean operators; rejected:
a78 ranked boolean operators above unary, and unary may ride
any item.)

## Prerequisite: LSP diagnostic transport fix

Verified against source (`compiler/lsp.go`,
`publishDiagnostics`): published items carry only
range/severity/source/message. The stable `code` and the a71
`Expected`/`Found`/`Hint` payloads exist on the internal
`Diag` but never reach the wire. Every new diagnostic in
this record — and the existing CAN4301–4306 and CAN6013
squiggles — loses its code in the editor until this is
fixed. Small transport slice with wire-level goldens, before
any item below ships a code. Comparing internal `Diag`
arrays alone does not satisfy CLI/LSP parity.

## 1. Named constants

A real, typed, revisioned, providible scalar-literal
declaration. V1 admits `int`, `str`, `dec`, `bool` only —
records, sequences, and brands are explicitly deferred
(design3 admitted them; the other two drafts and
blast-radius say narrow first). Initializers are literals;
computed initializers, aliases-as-initializers, and
zero-argument functions are rejected. No runtime
initialization, no import dependency at emit.

Spelling shape (all drafts agree):

```can
const std__ascii__COLON: int rev 1 = 58
```

**Naming decided here** — the drafts split three ways
(`Html__AsciiColon`, `html__ASCII_COLON`,
`ascii__COLON`), so this record decides on stdlib-majority
grounds: within stdlib, two modules (scalars, text) use
the shared `std__<domain>__<name>` convention against
one (`html__*`); active stdlib development uses it; and
a shared `std__` prefix cannot be squatted by an app
module the way a bare `ascii__` prefix could. Short
names stay SCREAMING (unanimous across drafts), which
also gives each declaration kind a distinct visual
namespace — lowercase for functions (`std__bool__and`),
CamelCase for types (`Bool__Value`), SCREAMING for
constants — keeping R5's grep-names rule unambiguous.
Pins read `std__ascii__COLON@1` like any `uses` entry.

The ASCII declarations live in a new shared provider
`std/ascii/ascii.can` (`mod ascii`, verified absent);
the magic-bound migration is a cross-file acceptance
proof. The ASCII predicates live beside the constants
(`std__ascii__is_digit`, `std__ascii__is_alpha`,
`std__ascii__is_alnum`), reusing scalars' `Bool__Value`
via an ordinary pinned `uses` entry rather than
minting a third duplicate: domain co-location is the
epic's own anti-sprawl logic, and the precedent
(`std__str__*` in text, `std__int__*` in scalars) puts
predicates with their domain. (Context: a boolean
combinator *library* already exists —
`std__bool__and/or/not` in scalars — so item 5 adds
keyword syntax for positions calls cannot occupy, not
a duplicate library.)

References work in bodies, tests, expected payloads,
`given` exchange arguments and outcomes, permitted
patterns, and contract expressions — test scope ships in
v1, since nameable values pay off most in tests. Repeated
scalars may be named inside existing nested constructors
and `seal` expressions, but those stay in their original
test context: **test-authorized brand construction never
becomes exportable production data.**

**Emit inline resolved literals, not exported TS
constants.** No runtime import, initialization order, or
shared mutable object. (The draft-cited R11 precedent does
not exist — R11 is the tsc re-check clause. Inline emit
stands on the stated reason: emit produces per-module flat
code with no runtime import mechanism.)

Details that prevent shortcuts (all verified as live
constraints, all carried):

- Evidence resolves constants in its **owning module**,
  not the ambient function environment.
- A constant used only in evidence still marks its import
  used (unused-import interplay).
- The termination checker stays syntactic: substitution
  must not make `n - One` another spelling of the
  certified `n - 1` step, and no relaxed `decreases`
  recognition is introduced.

### Constant identity (tagged R4 clarification)

R4's letter ("Changing code without bumping rev",
`REQUIREMENTS.md`) conflicts with shipped a77 practice
(bodies and tests excluded from interface identity). The
clarification: a constant's semantic content is its
expanded **typed value and nominal dependencies**, not its
spelling. Inventory comparison matches unchanged names,
then pairs remaining equal owner/revision/type/value
entries as renames — revision-inert, never bumped, never
guessed between equal-valued constants. A changed value
cannot hide behind a rename. Canonical formats advance
through the revision-format sequence with reviewed
baseline bridges; incompatible formats are refused, never
silently rehashed. The new `const` decl kind needs
canonicalization on these terms.

### Verifier touchpoints

- Admission (a80): constant references expand to literals
  before fragment classification — no whitelist change for
  int/bool constants; externally-sorted constants still
  fail as today.
- Identity (a77): `canonConst` per the clarification.
- Reporting (a82): `contractIdentities` is function-only;
  no change.

Acceptance: the ten magic bounds in `html__url__scheme_token`
become greppable names with zero call/test overhead.

### Closure: composite sorts (records, sequences, brands)

The deferral above is lifted: `const` now admits record,
non-nested `Seq`, and brand sorts per design3's closed-data
rule, implemented in `checkCompositeConst` (`compiler/const.go`).
Initializers stay ref-free (aliases-as-values included, so the
`TestConstLiteralOnly` pin and cycle-unrepresentability hold
verbatim); calls, arithmetic, matches, state, indexing, and
slicing are excluded; Bytes and variant-valued constants stay
inline. Validation reuses the shared tycker, so constructor
fields get row-identical unknown/repeated/missing/mistyped
rules (CAN6003), and eval/emit resolve through the existing
lazy-substitution paths with no new mechanism.

First consumer: `std/schema` decision tables migrate onto
eleven fixture consts (requests, entries, sites, snapshots,
policy seal). Row bytes 42,278 → 22,401 with zero outcome
change (`canlc normalize` byte-identical over all 122 schema
rows) and zero emit change (`schema.ts`, `errors.json`
byte-identical). Single-use hostile shapes (quote, control,
conflict, revoked, wrong-digest) stay inline where the bytes
are the point.

## 2. Forward arms

Exactly a61's spelling (no new error mechanism):

```can
on html.invalid_url e => forward e
on Ok r => forward r
```

`forward` is the entire immediate RHS of its own
call-outcome arm. The operand must be that arm's resolved,
named payload binder. `_`, outer binders, field
projections, constructors, inferred propagation, mapping,
retagging, and dropped fields are rejected; returning the
incoming union directly is rejected.

**Elaboration timing (decided here):** the checker
elaborates `forward e` into the ordinary complete
constructor during checking, so admission, the prover,
emit, and CAN4107 all see one shape:

```can
on html.invalid_url e => html.invalid_url(value = e.value)
```

Same payload, type, authority, and `emits` checks as
handwriting (a61's "same checks, same union
construction") — which is what forces check-time
elaboration rather than a downstream desugar: post-check,
a forwarded arm must be indistinguishable from its
manual reconstruction, so `relayStatus` applies one rule
to both and "identical evidence rules" holds by
construction instead of by parallel implementation.
For `Ok`, the source payload must hold
exactly the destination success fields with resolved
types; different wrapper-record names are not
automatically disqualifying — the operation stays
explicit field reconstruction, never a nominal record
cast. The emitted shape stays the existing flat tagged
union (`{ $can_kind: "html.invalid_url", value: e.value
}`), never an unchecked `return e`.

**CAN4107 is unchanged, and the boundary is precise:**
the certificate covers an unshadowed, bound **error arm
of a local call** with a complete unchanged
reconstruction (`relayStatus`, `compiler/lsp.go:425`).
`on Ok r => forward r` still needs a passing test to
take the arm. Untaken `Ok` arms and foreign-call error
arms acquire no certificates; certified arms are never
reported as executed.

### Migration warnings (verified against source)

Same-kind reconstruction census (full-corpus script,
`on KIND v => KIND(…)` per function): scheme-token 8,
authority 12, value_from 10, make_from 6,
**text_escape_from 8** — plus a long tail (153 corpus-wide,
mostly single-relay `el__*` builders, out of migration
scope). This closes **U1** mechanically: the 36-set
(scheme/authority/make_from/value_from) and design1's
38-set (authority/value_from/text-escape/scheme) counted
different worker sets; their union is **44 candidate
sites across 5 workers**. The older 35-receipt
(`12+10+6+7`) is the same four workers as the 36-set
with scheme undercounted 7-vs-8 — explained, not
competing. Scheme-token's eight are two first-character
recursive sites plus six continuation sites.

Eligibility is now audited, not assumed: a script
checked every same-kind site for own-binder projections
plus payload completeness against the declared type and
error fields. **All 44 migration candidates pass**
(44/44); 17 same-kind sites fail on values (matching
field sets but enclosing-value payloads — the trap
class, including the 438 arm below), plus 1
unresolvable kind (`render__utf8`, cross-module return
— slice-time check). So the 44 are eligible forwards,
not mere candidates; per-site confirmation against the
landed elaboration is still the migration slice's gate,
but there is no remaining inventory question.
Forwarding alone removes no arms, calls, or rows — only
retranscription.

This arm in `html__attribute__id` (std/html/html.can:438)
**must remain manual**:

```can
on html.nul_byte e => html.nul_byte(value = value)
```

It selects the enclosing input `value`, not `e.value` —
and sits directly above a true identity reconstruction
(`on html.invalid_identifier e =>
html.invalid_identifier(value = e.value)`), so a careless
migration forwards both. The negative probe uses
different strings for the two values. Matching test
values never establish structural identity, and no
dataflow-equality argument is introduced to claim
otherwise.

## 3. Range arms

Integer singletons and closed ranges:

```can
ascii__COLON => ...
ascii__UPPER_FIRST..ascii__UPPER_LAST => ...
```

Bounds inclusive. A range requires `lower < upper`;
equal endpoints use the singleton spelling. V1 bounds are
integer literals or visible integer constants — never
variables, calls, or expressions (hence after item 1).
Open/half-open ranges, decimal ranges, computed bounds,
implicit guards, and ranges as new termination guards are
rejected.

This extends a28's bool/string/wildcard slot fragment. It
does not reinterpret the deferral: a28 deferred
OR-patterns, guards, and multi-call; ranges widen the
admitted value domain, while item 4 separately reopens
the OR deferral.

### Proof rule

For every integer slot, collect exact arbitrary-precision
cut points `lower` and `upper + 1`: finite intervals plus
two unbounded tails, membership constant per atom. Feed
the atoms into the existing symbolic product-coverage
machinery (`compiler/eval.go:1900–2040,2056–2210` ranges
verified present) — no integer enumeration, no
machine-width narrowing, no ASCII assumption.

Per arm: `useful = space minus earlier arms`; per match:
`uncovered = full product minus union of arms`. Empty
useful is a static error; nonempty uncovered is an
exhaustiveness error with concrete witnesses (the
three-witness limit caps reporting, not proof). A `_` in
one product row proves nothing elsewhere. Partial
overlaps are legal under first-match semantics; fully
shadowed arms are rejected. The value-pattern `_` covers
a proved scalar remainder only — never call outcomes or
variant cases. The final-`else` separation holds: proof
results are consumed, never invented in emit.

**One range arm stays one CAN4107 obligation.** Boundary
and neighbor rows are mandatory acceptance fixtures for
the two stdlib migrations, written independently of the
range constants — but not promoted into a universal new
coverage law. The `n <= 0` guard and `n - 1` step keep
their termination standing; no new certificate.

### Verifier touchpoints

- Admission (a80): range arms are a new accepted pattern
  kind over `int` (own probes). Nothing else moves.
- Prover (a81): range arms become integer-interval path
  assumptions — QF_LIA-native, no new theory.
  Or-alternatives (item 4) split paths per alternative.

## 4. Or-patterns

`|` inside a single pattern slot, one RHS, one source
arm:

```can
ascii__SLASH | ascii__QUERY | ascii__HASH => ...
```

Ranges bind tighter than `|`; `|` binds tighter than the
comma between slots (the `|` operator token already
exists in the TextMate grammar — reclaimed, not
invented). V1 combines nonbinding scalar patterns: bools,
strings, integer singletons/ranges, suitable constants.
No wildcards inside alternative lists, no variant/error/
`Ok` alternatives, no binder unification, no mixed scalar
types per slot, no alternative tuple syntax.

**Coverage has two layers:** CAN4107 stays one obligation
per source arm, and every explicitly written alternative
in every slot needs a passing test selecting it in the
winning arm. Not Cartesian-product — but `1..10 | 5..15`
needs a witness in `11..15` for the second alternative
(`7` cannot credit both). An alternative whose effective
selection region is empty is rejected statically.
Handwriting shrinks; a mistyped delimiter stays visible
to the test law.

### Authority terminators (verified against source)

Authority's body (std/html/html.can:548–639) branches
`match n <= 0` first, then ladders `s[0] == 47/63/35/0`,
each closing a `prev == "-"` sub-check. The naive
flattening `match n <= 0, s[0]` evaluates `s[0]` on empty
input — behind the fuel guard EOF lives, while `/ ? #`
inspect a character. End-of-input also differs from
delimiter stops (empty `prev` rejects at end, not at the
delimiter arms). The migration is grouping **plus an
ordinary helper extraction**, not alternatives alone:

```can
html__ASCII_SLASH | html__ASCII_QUESTION | html__ASCII_HASH => match call html__url__authority_finish(orig, s, n, prev)
  on Ok r => forward r
  on html.invalid_url e => forward e
```

```can
match prev == "-"
  true => html.invalid_url(value = orig)
  false => Ok(value = orig, tail = s, n = n)
```

The EOF path calls the helper **after retaining the
existing empty-predecessor rejection** (the delimiter
paths never performed it). Cost explicit: two call
sites, four outcome arms, three helper rows. The helper
extracts as a named function (design2's
`authority_finish`): two call sites need one shared
block, and a match block cannot be shared without a
call — duplicating it inline would defeat the purpose.
The name is the migration author's call; the shape is
decided.

## 5. Boolean operators

Strict keywords (the grammar's orphaned `and` is
reclaimed; `or`/`not` ship with the item):

```can
and
or
not
```

`&&`, `||`, `!` are rejected. `!=` stays inequality.
Precedence, low to high: `or`, `and`, prefix `not`,
existing comparisons, binary `+ -`, `* / %`, prefix
numeric `-` (item 6), existing tighter length/postfix/
atomic forms. So `not a == b and c` is `(not (a == b))
and c`; comparison associativity is preserved. Both
operands must be bool. No calls inside operands, no
implicit truthiness, no symbol aliases.

Evaluation is left-to-right, once-only, and **eager**:
each operand evaluates exactly once, the right operand
whenever the left returns normally, whatever its value.
A left fault still stops execution before the right
operand — strictness is not execution after failure.
The well-typed fault cases must fault loudly, never
become typed `emits` outcomes or boolean results
(`docs/fault-contracts.md` typed-outcome vs
primitive-fault distinction, verified present):

```can
false and ((1 / 0) == 0)
true or ((1 / 0) == 0)
```

(The simpler `false and (1 / 0)` is ill-typed — integer
right operand, not a short-circuit test.) Hence no bare
`emit(left) && emit(right)` lowering: strict helper
calls through `$canBoolAnd`/`$canBoolOr` (emitted only
when used, following the established `$canDec*` helper
pattern), and the interpreter evaluates and stores both
child results before the truth table. Consequently `(n >
0) and (s[0] == 65)` is **not an indexing guard** —
domain-conditional evaluation still needs an explicit
exhaustive match. A generated-TS runtime parity gate is
mandatory: type-checking alone cannot catch an
accidental short-circuit.

The ASCII predicate row follows as three separately
gated stdlib slices — digit and alpha in either order
(leaves, no interdependency), alnum last — on the shared
constants with the existing `Bool__Value` wrapper,
reused from scalars by pin. Placement follows the
item-1 decision above (`std__ascii__*` beside the
constants). Alnum composes explicitly bound call
results; no operand-call exception, no multi-call
match:

```can
match call std__ascii__is_alpha(code)
  on Ok a => match call std__ascii__is_digit(code)
    on Ok d => Ok(value = a.value or d.value)
```

### Verifier touchpoint

Admission (a80) gains `and`/`or`/`not` as supported bool
combinators over bool operands (own probes) — the
whitelist has none today because the language has none.
The prover needs no change: eager semantics are
invisible to proof, and division is outside the fragment
anyway.

## 6. Unary minus

Prefix `-` for exact `int` and `dec`, binding tighter
than multiplication and binary subtraction. The parser
must handle `a * -b` and `a - -b`; a leading-minus
branch alone is insufficient. No unary plus. Existing
`-3` and `d"-0.5"` keep literal behavior and emitted
bytes; negated literal spellings normalize back to
those forms (literals-only today — parser comment
verified). Integers emit `(-value)`. **Decimals never
emit JavaScript `-x`** (canonical strings): reuse exact
subtraction from decimal zero, `$canDecSub("0.0",
value)` — the helper exists (`compiler/emit.go:506`);
the slice confirms the canonical-zero spelling against
the decimal canonicalizer rather than assuming it.

Migration is deliberately small: the `0 - scale`
arguments in `std__dec__divide_round_half_even_result`
become `-scale` (verified idiom, 3 sites), plus
`std__int__negate` (verified present at
std/scalars/scalars.can:394) keeping its three rows. Not
the round-half-even shared-tail/`let` exhibit. The
canonical-zero spelling is established, not pending:
`d"1.0" - d"1.0"` evaluates to `"0.0"`
(arith_test.go:36) and `d"0.0"` is used as a literal in
scalars.can — `$canDecSub("0.0", value)` needs no
further confirmation.

### Verifier touchpoint

None for integers: negation is QF_LIA-native. `0 - x`
fixtures keep working; migrating them is optional
churn. `dec` is outside the fragment either way.

## Migration receipt

Design2's table (most detailed, tied to named bodies).
Row accounting rule — this is what closes the row-count
method, with exact totals instantiating mechanically
from landed bodies in the migration slices: keep every
existing row that still selects the same arm; add per
replacement range arm witnesses for lower−1, lower,
upper, upper+1 at each decision boundary, one witness
per `|`-alternative in its exclusive region, and one
per gap value between adjacent admitted classes. The
receipt's arm counts are the structural decisions; the
row totals above follow the rule once bodies land.

| Exhibit                 | Reviewed source arms | After ranges / flatness | After alternatives | Test rows |
| ----------------------- | -------------------: | ----------------------: | -----------------: | --------- |
| `scheme_token`          |                   40 |                      23 |                 11 | Keep 14, add 20 boundary → **34** |
| `authority`             |                   54 |                      40 | 28, plus 2 in `authority_finish` | Keep 26, add 12 boundary → **38**, plus **3** helper rows |
| `value_from`            |                   22 |                      19 |          Unchanged | Keep all **12** |
| `attributes__make_from` |                   18 | Forward RHS changes only |         Unchanged | Keep its **3** own rows and existing transitive evidence |
| `text_escape_from`      |             8 eligible sites | Forward RHS changes only |         Unchanged | Arm/row migration numbers TBD by slice (inventoried + eligible) |

Design1's higher row estimates (62 / 56 / 19) stand as
the upper bound until the migration slices re-derive
exact rows. Boundary rows preserve consumer-specific
behavior (`/` terminates authority with unchanged tail;
`:` continues scheme but not authority) — never
generated by assuming outside-alphanumeric means false.
Value-from's five-arm string match migrates on already
available syntax (the `name` 2-arm flat idiom proves
it) — an explicit acceptance task with fuel and NUL
guards intact, not a range credit.

## Diagnostics, tooling, gates

Nine codes, reconciled from the drafts' 6/9/13-way
split. All verified free of collisions:

| Code | Rule | Source |
| ---- | ---- | ------ |
| `CAN2003` | Constant naming | design2 |
| `CAN2104` | Unknown constant | design2 |
| `CAN2105` | External constant missing from `uses` | design2 |
| `CAN2205` | Duplicate same-module constant | all three |
| `CAN6014` | Nonliteral constant initializer | design1+2 |
| `CAN3011` | Invalid forward placement/binder | all three |
| `CAN4110` | Invalid integer range bounds | all three |
| `CAN4111` | Earlier arms fully cover this arm | all three |
| `CAN4112` | Or-alternative contributes no remaining space | design1+2 |

Dropped: design3-only `CAN2401`–`CAN2405` (const family
with no agreed semantics — the item-1 slice respawns
codes only if five distinct const rules materialize)
and `CAN4113`/`CAN4114` (folded into `CAN4112`'s
alternative-usefulness rule). `4112` takes the
design1/design2 meaning (alternative contributes
nothing), not design3's (unreachable arm — covered by
`4111`).

Each code ships message templates, payload strings,
source anchors, minimal violations, and golden paths in
its item slice; fixtures are draft inputs, never
captured compiler output. Existing type, pin, coverage,
termination, and fault codes are reused.

Every item runs design → grammar → parse → proof/check
→ eval → emit → diagnostics → editor grammar → goldens
→ migration → tagged-docs phases, with rollback. Gates:

```sh
go test -count=1 ./...
go run ./tools/modcheck
go run ./tools/gramcheck
(cd tscheck && npm ci --no-audit --no-fund &&
  ./node_modules/.bin/tsc -p tsconfig.json)
go test -count=1 ./compiler -run '^TestA78[CFROBN]'
go test -count=1 ./compiler -run '^TestA78.*Runtime'
```

Runtime suites execute actual generated output, never
just type-check it, and fail — never skip — when
tooling is missing. Negative probes are required for
ranking shortcuts, hidden calls, unsupported canonical
nodes, false coverage certificates, and fault-order
changes. The mutation gate runs both consumers: a
digit-57→58 extension is masked in scheme by its
earlier colon arm and caught only through authority's
colon rejection.

Guards beyond ranges, multi-call, higher-order
operations, record update, table lookup, and `let` stay
out. Readability refactors under current rules stay
out. A const design that is secretly a fn or secretly a
cell fails item 1.

## Corrections log

Reviewed claim-by-claim against source at `e05b07b`;
kept what holds, cut or fixed what does not:

- **Cut: the R11 precedent.** R11 is the tsc re-check
  clause. Inline emit stands on the stated reason (no
  runtime import mechanism in emit).
- **Corrected: `case`.** a78's bonus finding said
  `case` survives only as a `parseFields` label argument.
  Wrong: `case` heads live variant-declaration rows
  (`reVariantCase`, `VariantCase`, `compiler/parse.go`
  region verified). Cleanup preserves `case`, removes
  the genuinely obsolete vocabulary, reclaims `|` and
  `and` only with their items.
- **Corrected: int predicates.** a78's "established"
  `is_even/odd/multiple` overstates: `is_zero`,
  `is_positive`, `is_negative`, `is_prime` (plus text
  `is_whitespace`) exist; even/odd/multiple do not.
- **Reconciled and closed: relay counts (U1).**
  Full-corpus census: the 36-set and design1's 38-set
  counted different workers; the union is 44 sites
  (scheme 8, authority 12, value_from 10, make_from 6,
  text_escape_from 8) plus a 153-strong long tail
  outside migration scope. The 35-receipt is the 36-set
  with scheme undercounted 7-vs-8. The eligibility
  audit then passed 44/44 (own-binder projections,
  complete payloads) and isolated the 17-site trap
  class plus 1 cross-module unknown — so U1 leaves no
  inventory question, only slice-time confirmation
  against the landed elaboration.
- **Decided: naming and predicate home.** No draft
  consensus existed (three spellings); decided here on
  stdlib-majority grounds (`std__` spans scalars+text
  against one `html__` module), prefix-squatting
  resistance, and kind-distinct visual namespaces:
  `std__ascii__COLON`, predicates beside the constants
  as `std__ascii__is_*`, scalars' `Bool__Value` reused
  by pin. Pre-existing `std__bool__and/or/not` noted
  so item 5 is never misread as duplicating them.
- **Reconciled: const types.** Narrow v1 wins 2–1
  (design1+2 over design3): scalar literals only;
  records/sequences/brands deferred.
- **Reconciled: order.** Boolean operators fifth,
  unary sixth (design2+3 and a78's ranking over
  design1's inversion).
- **Reconciled: diagnostics.** Nine-code set above;
  design3-only extras dropped with reasons.
- **Reconciled: bool helper name.** `$canBoolAnd`
  (design1+2) over design3's `$canAnd`.
- **Reconciled: dec negation.** Concrete
  `$canDecSub("0.0", value)` (design1+2, helper
  verified present) over the abstract sign-toggle.
- **Reconciled: OR migration.** Design2's extracted
  `authority_finish` (costed: 2 sites, 4 arms, 3 rows)
  over the inline sketches.
- **Demoted: designer-executed checks.** Model-vector
  counts (1.1M×2 classifiers, 170k/64k worker
  comparisons, product tables, JS probes) ran in the
  designers' sandboxes against inputs not in this
  repository. Kept as designer evidence with per-slice
  rerun requirements — never as established fact.
- **Removed: dead sandbox links.** All `sandbox:/…`
  links pointed at designer environments, not the repo.
- **Noted: pin currency.** All three pins
  (`7c33f7c`, `b5525eac`, `e87e5cb`) are in-history;
  draft line-anchors predate HEAD and may have drifted
  — anchors above are re-verified, not inherited.

## Verification appendix (reviewer-executed)

- `publishDiagnostics` items: range/severity/source/
  message only — no `code`, no payloads. Prerequisite
  real.
- Nine reconciled codes + four dropped codes: no
  collisions outside the drafts.
- Full-corpus census (`on KIND v => KIND(…)` per
  function): scheme 8, authority 12, value_from 10,
  make_from 6, text_escape_from 8 = 44; 153
  corpus-wide including the single-relay tail.
  Eligibility audit over the same sites (binder +
  completeness vs declared fields): 44/44 pass, 17
  impure-value manuals (incl. the 438 arm), 1
  cross-module unknown (`render__utf8`).
- Canonical zero `"0.0"` established
  (arith_test.go:36, scalars.can usage). Module
  casing: all `mod` names lowercase; stdlib uses
  `std__<domain>__<name>` (scalars, text) and
  `html__*` (html) — the naming decision follows
  the two-module majority.
- Manual arm `std/html/html.can:438` in situ, true
  identity arm directly below.
- Authority 548–639: `n <= 0` base, `s[0] ==
  47/63/35/0` ladders each closing `prev == "-"`.
- a61 item 2 present. `relayStatus`
  (`compiler/lsp.go:425,440,450`) present.
  `Bool__Value` present (division, scalars).
  `std/ascii` absent. No `let` in parser. Relay
  concept in eval. `std__dec__divide_round_half_even_result`
  present. `$can_kind`, `$canDecSub` present.
  `std__int__negate` (scalars.can:394) present.
  `docs/fault-contracts.md` present.
- `TestDivZeroLoud` pins loud faults. Parser comment
  pins literals-only unary minus. `0 - scale` at 3
  sites. Variant `case` rows live
  (`reVariantCase`, `VariantDecl`/`VariantCase`).
- Grammar keyword regex holds orphaned
  `when|then|and|reach|expect|mock` (highlighted,
  unparseable); `|` an operator token.
- Anchor ranges verified present: eval.go (2264
  lines), check.go (2059), lsp.go (784), parse.go
  (1774), emit.go (2334).
- ai-lock.json: aspirational mentions only
  (`REQUIREMENTS.md`, a02 non-goal). "Future lock"
  framing stays future.

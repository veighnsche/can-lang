# v0.4 — Type discipline (spec)

Status: shipped. Panel third place, first feature that changes
which programs are legal: the float ban plus branded secrets.

Two halves, one rule underneath: **no implicit conversions, ever.**
A value of one type never silently becomes another. The compiler already
banned inference that hides information (R-requirements non-goals); this
extends the ban to values. Every conversion is a named, greppable,
compiler-checked boundary.

## Half 1 — Float ban and `dec`

IEEE floats are ungrammatical. A dotted-number literal (`0.5`, `12.34`,
`1e3`) is a compile error, `CAN6001`, at the token, naming the legal
spelling:

- `float 0.5 has no spelling: write d"0.5"`

Today `0.5` parses as a dotted `ref` and dies at test-execution time
with an unbound-name error; the ban replaces confusion with one code.
Floats are rejected in every value position (test args, expectations,
given stubs, bodies, scrutinee args): the scanner parses them as
`Kind "float"` and `checkStatic` flags each one, so the squiggle is
precise and shared between editor and CLI. The evaluator refuses
`float` nodes outright (defense in depth; unreachable past the gate).

The legal spelling for fractional quantities is the `dec` literal,
`d"12.34"`: one canonical form, no new delimiters, no suffixes, no
constructor calls. Canonical shape is `^-?\d+\.\d+$`; anything else in
the quotes (`d"abc"`, `d"1"`) is `CAN6001` with the same message.
Values normalize at parse (trailing fractional zeros stripped, so
`d"1.50"` and `d"1.5"` are the same value) and compare exactly:
proofs evaluate `dec` with `big.Rat`, so `=>` in a test still means
exact equality. This is the whole payoff — agent-written tests die on
float equality (`0.1 + 0.2 != 0.3`), and exactness is load-bearing for
normalize goldens and decision tables.

Comparisons (`==`, `!=`, `>=`, `<=`, `>`, `<`) work on `dec` exactly like `int`.
Mixed `int`/`dec` comparison is `CAN6003`: no conversions, not even
widening. There is no arithmetic in v0 (`+` does not exist), so `dec`
needs none; arithmetic is future work and will be exact when it lands.

TypeScript mapping: `dec` emits as `number`. Documented boundary: canlc
proofs are exact over arbitrary precision; the TS mapping is exact for
up to 15 significant digits, which covers every v0 program (v0 has no
arithmetic to accumulate error). A decimal runtime arrives with
arithmetic, not before. `normalize` renders `dec` canonically
(`d"1.5"`).

> Amendment (a10): superseded — see `a10-numerics.md`. `int` emits as
> `bigint` (unbounded, exact) and `dec` as canonical-digit strings
> through exact `$canDec` helpers; no `number` mapping and no
> significant-digits boundary remain.

## Half 2 — Branded secrets

`pw: str` and `pw_hash: str` are both `str` today, and confusing them
is exactly the mistake an agent makes at 2am-context-window. Sensitive
strings get brand declarations:

```
brand Auth__Password is str rev 1
```

One underlying type in v1 (`str` only; int-backed brands are future).
Brands are nominal: `Auth__Password` is not `str`, and
`Auth__Password` is not `Auth__Hash`. Brands follow R3 naming
(`CAN2002` on violation), carry `rev` like types, and are listed in
`provides` like types (they are module surface; `checkModIntegrity`
covers both sides, errors stay exempt).

Branding is proof, not runtime: at runtime a branded value is a plain
string. The compiler tracks the brand; the emitter forgets it (`tsType`
maps every brand to `string`). Taint tracking as syntax, with zero
runtime cost.

Raw strings become branded through exactly one gate, `seal`:

```
pw_hash = seal Auth__Hash("secret")
```

`seal Brand("literal")` is an expression form legal in every value
position (test args, expectations, stubs, bodies). The argument must
be a string literal — sealing a computed string is laundering, and it
is `CAN6003`. The brand must be declared (`CAN6002`). There is no
unseal: a branded value leaves its brand only through a declared
declassifier (below). `grep seal` is the complete audit of where
secrets enter the program.

The sink rule falls out of nominal typing with no separate taint pass:
a branded value flows only into a position annotated with the same
brand. `reason = pw` where `reason: str` is `CAN6003`. Comparing
`pw == user.pw_hash` across two brands is `CAN6003`. Error fields,
return fields, call args, test expectations are all just positions
with declared types; the checker enforces all of them. Same-brand
equality is allowed (exact, leaks nothing, result is `bool`).

### Declassifiers are `extern`

A brand that can never leave is useless: password checking must
compare a password against a hash. That boundary is a foreign function,
declared, never defined:

```
extern auth__check_pw(pw: Auth__Password, hash: Auth__Hash) -> Auth__Verdict rev 1
  emits [auth.mismatch]
```

`extern` is the reserved foreign-import form from R2, now sketched.
Rules:

- Module-local: declared in the file that calls it, listed in that
  file's `provides` (it is emit surface: the TS file imports it by
  name). No `uses` entry (there is no provider module to pin; the
  `rev` on the decl is the pin). Another module that wants it
  declares its own.
- No body, no tests, no decision table. Optional `emits` row
  (defaults to empty); the row exists so call-site `given` tables can
  script foreign outcomes — stubs are validated against the extern's
  `emits` exactly like can calls.
- Callable only as a match scrutinee (`match call auth__check_pw(..)`),
  with total `given` tables like any call. The evaluator consumes
  stubs for externs without a body; nothing else changes, because
  stubs were always values, never executions.
- Foreign predicates return `Ok` or an error, never `bool`: the check
  above returns `Ok(Auth__Verdict)` on match and raises
  `auth.mismatch` otherwise, so `on Ok ok` / `on auth.mismatch _`
  arms reuse the entire proof, coverage, and catalog machinery
  untouched. `bool`-returning calls remain unscriptable in v0.
- TS emit: `import { auth__check_pw } from "./auth.externs"` (R11
  already promised typed imports; the file itself is host-provided,
  outside canlc).

`type Auth__Verdict rev 1 ()` is an empty record type: the verdict
carries no data. Empty field lists are legal for types (dual of empty
error payloads like `db.down()`); `Ok()` constructs them.

The error catalog lists an extern's `emits` kinds as raised by the
extern (`raised_by` includes `auth.auth__check_pw` for
`auth.mismatch`): the `emits` row is the foreign code's contract, and
the catalog is the index of contracts.

## The checker (first real type checker)

`checkTypes` is new; until now annotations were documentation. It runs
in `checkSem` after `checkEmits`, before test execution, and is shared
by editor and CLI. Environment: params plus match-bound variables
(`on Ok user` binds the callee's `Ret` record; shadowing copies the
map down, no diagnostic — the shadowing question stays open).

Universe: `str`, `int`, `bool`, `dec`, record type names, brand names.
Error kinds are not value types.

- Unknown name in any annotation (param, ret, type field, error
  field, seal) → `CAN6002` at the name.
- `==`, `!=`, `>=`, `<=`, `>`, `<` require identical operand types → else
  `CAN6003` at the operator row, naming both types. (Dynamic `==`
  across kinds used to be false; statically it is now an error.
  Intended: the discipline bites.)
- Positional call arg `i` against callee param `i` (can or extern);
  arity mismatch → `CAN6003`. (Positional calls stay legal per open
  question 2; now they are checked.)
- `Ok(...)` fields against the enclosing `Ret` record (test
  expectations and bodies); named error ctors against their
  `ErrorDecl`; unknown record ctor → `CAN6002`.
- Record field access (`user.pw_hash`) walks `TypeDecl` fields;
  fields on brands or scalars → `CAN6003`.
- Test args against params; `given` stub fields against callee
  contract (kinds were already checked; now fields too).
- `seal` per above.

Codes: `CAN6001` float/dec spelling, `CAN6002` unknown type, `CAN6003`
mismatch. Family `CAN6xxx` types (`CAN5xxx` is world/tooling).
`allCodes` gains all three; the code.go family comment is updated.

## Consequences (accepted before building)

- `docs/archive/sketches/auth-login/db.can`: declares `brand Auth__Hash is str
  rev 1`; `Db__User.pw_hash` becomes `Auth__Hash`; constructions use
  `seal` in tests and arms. Field-type change is breaking:
  `Db__User` 3 → 4, `db__get_user` 4 → 5 (body bytes changed).
- `docs/archive/sketches/auth-login/auth.can`: declares `brand Auth__Password is
  str rev 1`, `type Auth__Verdict rev 1 ()`, `error auth.mismatch()`,
  `extern auth__check_pw`; `pw` becomes `Auth__Password`; the
  `pw == user.pw_hash` bool match becomes `match call
  auth__check_pw(pw, user.pw_hash)` with `on Ok ok` /
  `on auth.mismatch _` arms; every test scripts the new call node
  (unreached tests write `-`); `auth__login` 2 → 3; uses re-pinned to
  `db__get_user@5`, `Db__User@4`, plus `Db__Hash@1` (db-domain prefix
  per R3; the spec draft said `Auth__Hash`, corrected at build).
- Goldens regenerate: `auth.ts` gains the extern import and the
  mismatch union arm; `db.ts` is unchanged at runtime (brands erase
  to strings); `errors.json` gains `auth.mismatch`;
  normalize outcomes are unchanged (same decisions, proven a new
  way). The flagship's core logic is now a foreign call wrapped in
  proof — the honest shape of a password check.
- `tools/modcheck` learns `brand`/`extern` decls; the TextMate
  grammar gains the two keywords plus `seal` and `d"..."`, or
  `gramcheck` fails the gate. `broken-login/` demos must keep
  exactly their titled squiggles: every fixture is verified
  type-clean under the new checker.
- Every new diagnostic gets an `lsp_test.go` case following the
  existing `TestDiagnose*` shape; `TestCodesUnique` and
  `TestGoldenJSONDiags` cover the new codes.

## Open decisions (do not block)

- Whether `seal` on a non-literal should ever be legal (leans: no;
  explicit laundering review is a future lint, not a language form).
- Whether externs need cross-module imports or stay module-local
  (leans: local; a shared extern module is just a module whose
  surface is all extern, and it works today).
- Whether `normalize` should redact branded values (leans: no;
  fixtures contain test secrets by design; secrecy is about program
  flow, not committed tables).
- Full decimal runtime and arithmetic (waits for the first `+`).
- int-backed brands.

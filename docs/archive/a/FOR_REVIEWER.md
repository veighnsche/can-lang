# FOR_REVIEWER — the can-lang language, by example

Scope: the language only. How it parses, what it proves, what it
refuses, and what shipped programs look like. Toolchain internals
(compiler stages, LSP plumbing, checkers) are out of scope; one
short verification pointer closes the doc.

Thesis in one line: tests and business logic are the same artifact.
Every function ships its decision table, every effect is a handled
call, and anything the compiler cannot prove before running is a
compile error, not a runtime surprise.

## 1. Reading a file: `db.can` end to end

The smallest complete program (`docs/archive/sketches/auth-login/db.can`):

```
mod db
  provides [db__get_user, Db__User, Db__Hash]
  uses []
  emits [db.user_not_found, db.down]

error db.user_not_found(id: str)
error db.down()

brand Db__Hash is str rev 1

type Db__User rev 4 (
  id: str
  pw_hash: Db__Hash
  failed_attempts: int
)

fn db__get_user(id: str) -> Db__User rev 5
  emits [db.user_not_found, db.down]
  tests
    known_user(id = "u_01") => Ok(id = "u_01", pw_hash = seal Db__Hash("secret"), failed_attempts = 0)
    unknown_user(id = "u_99") => db.user_not_found(id = "u_99")
=
  match id
    "u_01" => Ok(id = "u_01", pw_hash = seal Db__Hash("secret"), failed_attempts = 0)
    _ => db.user_not_found(id = id)
```

Read it top-down: the `mod` header is the dependency manifest
(what this file defines, needs, and can produce). `error` lines
declare scoped failure values. `brand` wraps `str` in a nominal
type with exactly one gate in (`seal`). `type` declares records.
`fn` declares the signature, its `emits`, and its decision table —
then `=` and one single-expression body. There are no statements
anywhere in the language: no `return`, no `let`, no early exit.

Reviewer check: `provides` must name exactly what the file defines
(nothing more, nothing less); every `uses` entry pins a provider
rev (`db__get_user@5`); every test must execute and pass at compile
time or the build fails.

## 2. Decision tables: `tests` plus `given`

A test row is a call specification: `name(inputs) => expected`.
An expectation is either an `Ok(...)` record or a complete error
construction (`=> auth.login_failed(user_id = "u_99")`, compared
kind and payload — a12). Bare error kinds are refused.

Foreign calls are scripted at the call site through `given` — one
exchange sequence per test (`docs/archive/sketches/auth-login/auth.can`):

```
  match call db__get_user(id)
    given
      happy => [exchange args (id = "u_01") outcome Ok(id = "u_01", pw_hash = seal Db__Hash("secret"), failed_attempts = 0)]
      flaky => [exchange args (id = "u_01") outcome db.down()]
      missing => [exchange args (id = "u_99") outcome db.user_not_found(id = "u_99")]
      ...
    on db.user_not_found _ => auth.login_failed(user_id = id)
    on db.down _ => ...
    on Ok user => ...
```

Each row proves "this request received this permitted response":
arg names resolve through the callee signature and every expected
arg must arrive equal, with nothing unexpected.

Four rules a reviewer can check by eye:

- Totality: every test appears in every reachable table.
  Unreachable tests are written `-` explicitly — omission is an
  error, never a default.
- Consumption: each evaluation takes the head of that test's list.
  Calling with an empty list is an error; leftover non-`-` entries
  at test end are an error.
- Honesty: stubs must be `Ok(...)` or an error the callee declares
  in its own `emits`. Invented outcomes are errors.
- Shape: every row is an `exchange` with args and outcome;
  outcome-only rows are errors.

A site hit several times in one test takes a multi-element list —
the retry loop's
`later => [exchange args () outcome net.down(), exchange args () outcome Ok(body = "hi")]`
scripts two hits of one site in order.

## 3. Types: exact, nominal, no conversions

Base types: `str`, `int`, `bool`, `dec`. Records: `Name(field = value)`,
empty record `Name()`. Construction and calls share one application
syntax; `:` declares, `=` binds; `[]` enumerates and nothing else;
no curly braces anywhere, including comments.

- `dec` is exact (`d"12.34"` literals; floats are ungrammatical).
  `+`, `-`, `*` take same-type operands only and yield the operand
  type — `1 + 2.5` is an error, never a coercion. Division has no
  rule and is rejected.
- Brands are nominal: `Auth__Password` is not `str` and not
  `Db__Hash`. Raw secrets enter only through `seal B("lit")`, and
  the only way out is a declared `extern` declassifier. `grep seal`
  is the whole audit: in `auth.can` the password brand and the hash
  brand never meet in can code — the comparison happens across the
  foreign predicate `auth__check_pw`.
- Errors are values, never thrown: `auth.login_failed(user_id = id)`
  constructs one; `on` arms handle them. A function raising anything
  outside its `emits` is an error. `emits` is a conservative upper
  bound (a12): entries need not be raised, but every entry must name
  a declared error — no consumer stub can manufacture provider
  honesty.

## 4. `match`: exhaustiveness is syntactic, coverage is measured

Every `match` must cover every outcome — a missing arm is a compile
error, not a warning. Three shapes:

- Booleans need exactly `true` + `false` (the lockout check
  `match failed_attempts >= 3`).
- Call matches need exactly the callee's `emits` plus `Ok`
  (three arms over `db__get_user`: `user_not_found`, `down`, `Ok`).
- Value matches close with `_`.

Beyond exhaustiveness, the test-per-arm law is measured: every arm
must execute across the decision-table run, over green tests only.
A test suite that never takes an arm fails the build even when every
test passes.

## 5. The three callee kinds

This is the load-bearing distinction. All calls happen as `match`
scrutinees, but what the call *means* depends on locality:

1. Foreign — an can function from another file (via `uses` pin) or
   a module-local `extern`. Never executed; the `given` stub is the
   outcome. Example: `match call db__get_user(id)`.
2. Local helper — a function defined in the same file. Executed,
   never stubbed: the call site carries no `given` table because a
   deterministic call has nothing to script. The helper's own inner
   tables script every test that can reach them, keyed by the
   caller's test names flowing through. Example: the two retry arms
   both end in `match call auth__verify(...)` with no table, while
   the helper's single `check_pw` table scripts all twelve tests.
3. Store operations — `state__get(C)` / `state__put(C, v)`. Executed
   against the test's private store, no `given` table.

Same-file callees take no pin and no `rev` reference; a same-file
`uses` entry is rejected as resolving nowhere.

## 6. Helpers: the shrink, shown

Before (each retry arm carried its own copy of the verify logic);
after, both arms end identically (`auth.can` lines 82–89):

```
        on Ok user => match call auth__verify(user.id, user.failed_attempts, pw, user.pw_hash)
          on auth.login_failed _ => auth.login_failed(user_id = user.id)
          on auth.account_locked _ => auth.account_locked(user_id = user.id)
          on Ok s => Ok(user_id = s.user_id, remaining_tries = s.remaining_tries)
    on Ok user => match call auth__verify(user.id, user.failed_attempts, pw, user.pw_hash)
      ...
```

and the decision lives once in `auth__verify` (lines 91–115): the
lockout check, one `check_pw` table, three helper-owned tests
(`vh_ok`, `vh_bad`, `vh_locked`). Reviewer check: the helper's
`emits` must be covered exactly at both call sites, and its arms
must be taken across helper tests plus caller flow-through.

## 7. Loops: recursion with a visible halting proof

The only loop is direct self-recursion admitted by a `decreases`
line naming one `int` param; every self-call site must pass
exactly `p - 1` and sit under the false arm of the canonical
`p <= 0` guard (a11). Recursion across files is refused
program-wide; only direct self-recursion is admitted
(`retry.can` lines 23–43):

```
fn retry__fetch(fuel: int) -> Retry__Doc rev 1
  decreases fuel
  emits [retry.exhausted]
  tests
    now(fuel = 2) => Ok(body = "hi")
    later(fuel = 2) => Ok(body = "hi")
    never(fuel = 2) => retry.exhausted
    empty(fuel = 0) => retry.exhausted
=
  match fuel <= 0
    true => retry.exhausted()
    false => match call net__fetch()
      ...
      on net.down _ => match call retry__fetch(fuel - 1)
```

Trace `never` with fuel 2: down, recurse with 1; down, recurse
with 0; base arm. Each step decrements by one under the positive
branch, so every chain — including negative entries, which take
the base arm immediately — returns a declared outcome, and the
proof runs before anything executes. Delete the `decreases` line
and the identical file is a cycle error: proof-gated admission
(a11), not a diverted hang — the trace is finite either way.
Mutual recursion stays refused, same-file or cross-file; `p - 0`,
larger steps, computed steps, unchanged `p`, and unguarded sites
are all rejected.

## 8. State: private cells, declared authority, isolated tests

One cell, two builtins, one annotation (`counter.can` quoted whole —
38 lines):

```
state Count__total: int = 0
...
fn count__bump(by: int) -> Count__Tally rev 1
  effects [Count__total.read, Count__total.write]
  ...
  match call state__get(Count__total)
    on Ok c => match call state__put(Count__total, c.value + by)
      on Ok _ => Ok(total = c.value + by)
```

- Cells are module-private, base-typed, literal-initialized, and
  absent from `provides`. `get` yields `Ok(value = ...)`; `put`
  yields empty `Ok`, whose bound variable carries nothing — using
  it is an error.
- Authority is declared per function and transitive through local
  calls with no inference: `count__twice` declares both capabilities
  because its helper bumps. Undeclared use and stale capabilities
  are errors; unproven authority blocks execution.
- Each test starts from init: `three` and `three_again` feed the
  same input and agree — under a shared store the second would print
  6. The tables themselves prove sequential tests cannot interfere.

## 9. Modules, revs, pins

`uses [db__get_user@5, Db__User@4, Db__Hash@1]`: every cross-file
edge pins an exact rev; unpinned use is an error. Changing code
without bumping rev is an error; bumping without changing code is
an error. `provides`/`uses` integrity is exact on both sides.
`state__get` / `state__put` are reserved names.

## 10. Rejected programs (the language saying no)

A reviewer should confirm each of these fails, with one precise
error per mistake:

- A `match` missing one callee outcome (e.g. no `on Ok` arm).
- A `given` table missing a test, or a key naming no test.
- A stub outcome the callee never declared.
- `1 + 2.5`, or any `/`.
- A self-call with no `decreases` line; a `decreases` line with no
  self-call; a self site passing `fuel` unchanged or `fuel - 0`.
- A `state__put` of a `str` into an `int` cell.
- Using the variable bound by a `put`'s `Ok` arm.
- An effect exercised but not declared; a capability declared but
  never exercised; a cell named but never declared.
- A bare error kind in a test expectation; an outcome-only script
  row; an `emits` entry naming no declared error.
- `broken-login/` holds one titled file per squiggle class.

## 11. Boundaries, stated not smuggled

- Division: no rule (exactness of `/` on `dec` needs its own).
- `int` is unbounded and exact in proofs; TS emit preserves it
  (`bigint`). `dec` is exact in proofs and in TS emit
  (canonical-digit strings plus exact `$canDec` helpers, a10).
- Prod cells persist across calls while tests prove per-scenario
  behavior from init; prod recursion depth is host-limited.
- No bool-returning calls, no cross-module externs, no non-literal
  `seal`, no int-backed brands, no async — each waits for a real
  program blocked without it.

## 12. Reviewer checklist

Per function: decision table present; every arm taken; `emits` a
conservative upper bound (every entry names a declared error);
args and `Ok` shapes agree with the record; error expectations
complete. Per call: right kind (foreign/local/store), right
evidence (`given` iff foreign, every row an exchange), right
authority (`effects` iff store, transitive). Per file: `provides`
exact, pins exact, no reserved-name collisions. Per program:
goldens byte-identical, `broken-login/` titles unchanged.

## 13. Verifying a claim (toolchain pointer, short)

`go test ./...` (goldens + diagnosis suites), `go run
./tools/modcheck` (uses/provides/given-keys), `go run
./tools/gramcheck` (editor grammar). Open any `docs/archive/sketches/*/*.can`
file with the editor extension: clean files show zero diagnostics,
`broken-login/` shows exactly its titled squiggles.

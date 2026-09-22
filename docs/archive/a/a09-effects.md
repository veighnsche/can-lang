# v0.9 — Effects: mutation and state (spec)

Status: shipped. Expressiveness item 8: mutation and state
extending R6's purity seed. The item 7 audit closed with no build
(no real program was blocked), items 1–6 were shipped, and the
build below was approved and landed with the counter sketch.
Refinement backlog (post-ship re-review): put-payload uses are now
rejected statically; foreign callees are named as uses-pins in
transitive messages; malformed store sites count as capability
use so one mistake yields one squiggle.

## The seed (R6, quoted)

"Pure by default; effects only via `call` + exhaustive outcome
handling. A function touching the outside world says so in `uses`."

The extension keeps every clause: state operations ARE calls,
handled through `match` like any other outcome, and effect use is
declared in the signature where R9 readers can see it. What changes
is that some callees are neither foreign (stubbed) nor local
(executed): they are store operations, deterministic per test, and
need no `given` table — the a07 principle extended to a third
callee kind.

## Shape: cells, two builtins, one annotation

A cell is module-private named storage for one base-type value:

```
state Retry__fuel: int = 0
```

`:` declares, `=` binds, per R1. Cell names follow R3 Type naming.
The type is a base type only (`str`, `int`, `bool`, `dec`);
brands, records, and cells holding cells are out. The init is a
literal only. Cells are private: they appear in no `provides`
list, take no `uses` pin, and are visible only in their own file
(a07 locality precedent). Cells carry no `rev`: they are runtime,
not surface, and an init change re-proves itself by re-running the
decision tables.

Two blessed call names operate on cells, taking the cell as first
argument:

- `match call state__get(Retry__fuel)` yields `Ok(value = ...)`
  with the cell's current value.
- `match call state__put(Retry__fuel, expr)` stores `expr` (exact
  type match, no conversions, per house rule) and yields `Ok()`.

Both are infallible, so their matches carry single `on Ok` arms.
The match is the point: every effect is a handled call, and a
future fallible op adds arms without changing shape. No new
expression syntax; `state`, `effects`, and cell refs ride the
existing grammar.

Authority to touch state is declared per function, beside `emits`:

```
effects [Retry__fuel.read, Retry__fuel.write]
```

`get` needs `.read`, `put` needs `.write`. Capabilities are
transitive through local calls: a caller declares everything its
helpers may do — no inference, per the project goal. A foreign
can callee's effects are readable in its own file, so the caller
declares a superset of them. Extern bodies are host-side and out
of the proof (existing boundary: the host owns the implementation).

## The isolation argument (written down, not waved at)

R7 runs every test in a hermetic sandbox. Each test evaluates with
a fresh store built from the decl literals; `put`s thread through
that test's evaluation only, including flow-through into helpers.
Sequential tests cannot interfere, and no test can observe another
test's writes. Values are immutable trees — bindings hold values,
never references — so there is no aliasing by construction, and a
`put` never disturbs an already-bound variable.

Store operations are therefore deterministic per test (arguments
plus the test's own prior puts), which is why they take no `given`
table: there is nothing to script, exactly the a07 reasoning. A
test that never reaches a `put` leaves the store untouched; a cell
no test reads is dead weight the tables will expose.

The documented boundary: prod emit holds cells in module-scope
`let`s shared across calls, while tests prove per-scenario
behavior from init. Cross-call histories in prod are outside the
tables — same class as the a04 int53 boundary: documented, not
solved. Async stays distant (a05): single-threaded store, no
ordering questions.

## Static rules (one rule, one code)

- `state__get`/`state__put` need no `uses` entry (same-file
  builtins, a07 precedent). Naming an undeclared cell is
  `CAN3001`: the call names something undeclared.
- Using an effect the signature does not declare is `CAN3107`
  (foreign-raise precedent: the boundary was crossed silently).
- A declared capability nothing uses — directly or through
  callees — is stale, `CAN3108` (stale-emits precedent: proof
  text that proves nothing misleads).
- Put-value and cell-type mismatch is `CAN6003`, like every other
  conversion that does not exist. A `get` payload binds `on Ok v`
  with exactly one field, `value`, typed as the cell type.
- Test-per-arm law applies to effect matches unchanged: an
  unreached `put` is dead effect code and fails coverage, which is
  the point.
- All effect errors block test execution in the prove-first gate:
  effects that are not proven do not run.

## Unchanged machinery (proof by non-interference)

- Parse gains one decl (`state Name: T = lit`) and one metadata
  line (`effects [...]`); duplicates and malformed lines are
  parse errors, per house precedent.
- Exhaustiveness: infallible ops want exactly `{ok}` through the
  same table. Coverage, `given` totality, `uses`/`provides`
  resolution, and the error catalog are untouched (store ops
  produce no errors and take no stubs).
- `modcheck`: untouched (expression-level; verified by running
  at build time).
- Grammar: `state` and `effects` join the control-keyword rule;
  `gramcheck` gains a sample, per a06 precedent.

## TS emit (inside a documented boundary)

Cells emit as module-scope `let Cell: T = init`. A `get` match
reads the cell into a synthetic `{ kind: "ok", value: T }`
union; a `put` match assigns then proceeds. The module ok-shape
agreement is unchanged: accessor payloads never flow into it.
Prod recursion-plus-state composes as ordinary statements; prod
stack and shared-state boundaries stay documented, not solved.

## Implementation (for the build proposal)

- `parse.go`: `StateDecl` (`Name`, `Type`, `Init`, `Line`);
  `effects [...]` in the fn-metadata switch.
- `types.go`: `state__get`/`state__put` bypass `callee()` with
  cell rules (declared cell, put-value type exact, `value`-field
  binding typed from the cell).
- `check.go`: capability checking with transitive local-call
  union plus foreign-callee superset; staleness; cell-decl
  validation through the existing field rules.
- `eval.go`: `Store map[string]*Value` on `Ctx`, forked fresh
  per test from decl literals; get/put evaluate strictly in the
  caller's environment.
- `lsp.go`: the `blocked` gate extends to the new codes.
- `code.go`: `CAN3107`/`CAN3108` in the registry.

## Consequences (accepted by writing this down)

- No build starts without approval of this doc; ratings and
  plans are not specs (a05 consequences still bind).
- The build ships a stateful demo as its own sketch (a bounded
  counter with per-test isolation proved by tables that would
  interfere under a shared store), with goldens mirroring the
  retry-loop shipment. `auth-login/` and `retry-loop/` stay
  untouched; `broken-login/` keeps exactly its titled squiggles.
- Tests mirror the a08 shipment: diagnose cases per new code,
  entry-shape negatives, transitive-capability cases, and the
  isolation claim checked (same scenario twice in one file, no
  cross-talk), plus registry coverage.

## Open decisions (do not block)

- Cross-module cell sharing (leans: no — accessor functions;
  cells stay private, calls cross).
- Fallible store ops such as bounded quotas (leans: no — the
  single-arm shape is the feature).
- Cell types beyond base (leans: no — records and brands wait
  for a program that needs them, per the item 7 rule).
- Prod shared-state test coverage, e.g. scripted call sequences
  (leans: separate proposal; per-test freshness is the v1
  semantics).

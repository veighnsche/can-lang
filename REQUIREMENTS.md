# can-lang — Requirements (v0.1 freeze) — HISTORICAL

> This frozen predecessor contract is superseded and no longer
> enforced: the shipped compiler retired decision tables,
> given/call evidence, revision pins, `e"..."` literals, and CAN
> codes, and its sketch programs were deleted. The body below is
> preserved untouched per its own no-silent-rewrite rule; current
> authority lives in [tasks](docs/implementation/tasks.md),
> [coverage](docs/implementation/coverage.md), the package READMEs,
> and [evidence](docs/implementation/evidence/2026-09-21/).

Status: living — v0.1 freeze plus ratified amendments, each tagged
with its version (a04–a09). Rules are never rewritten silently.
Live shape: `sketches/auth-login/` + `sketches/retry-loop/retry.can` +
`sketches/counter/counter.can`. Map + reading order: `docs/README.md`.

## Goal

A programming language optimized for AI agents to read, write, and verify —
even where that is actively unpleasant for humans. Verbose, explicit, single
canonical form. If a behavior can be declared, it must be declared; nothing
is inferred that an agent would otherwise have to guess.

Core thesis: tests and business logic are the same artifact, not two files.
Every function ships its decision table (`tests`) and its branch evidence
(`given` at each `call`); writing the behavior and proving it are one act
(see R7, R8). There is no separate test suite to forget, drift, or infer.
A decision table anchors intended behavior but cannot constrain all admitted
behavior: a weak table is satisfied by a wrong implementation, so the
specification must rule out the constant function, not merely bless examples.
Amendment (a87): within one table, rows differ in authority — `pinned`
rows are acceptance and report weakening loudly (`CAN6017`); unmarked
rows are proposed evidence.

## Non-goals (explicit outs)

- No `null` / `undefined`. No hidden control flow (early return, exceptions).
- No operator overloading, no implicit conversions, no default args.
- No macros or user-defined dialects. No lazy evaluation: strict/eager only.
- No type inference that hides information; annotations are mandatory.
- No syntactic sugar with two spellings for one thing. One canonical form,
  enforced by formatter (formatter TBD). Amendment (a66): interpreted
  `e"..."` literals are the versioned exception. Ordinary `"..."` stays
  raw and a decoded `e"..."` may denote the same value; the two forms
  are not interchangeable sugar (raw cannot spell LF without a
  physical line break, `e"..."` admits only six escapes), and
  interpretation never depends on module, revision, or formatter
  mode.
- No native/asm backend for v0.1 (revisit on measured hot paths; WASM first).

## R1 — Delimiters

- No curly braces outside string literals, including comments.
  (a45: `{`/`}` inside `"..."` literals are data — JSON, CSS,
  templates — never delimiters. The ban scans string-aware; quotes
  inside comments never open a string.)
- `()` = application and records: calls `db__get_user(id)`, construction
  `Ok(id = "u_01")`, empty record `db.down()`, grouping.
- `[]` = enumerations only: mod lists, `given` outcome sequences.
- `:` declares (type fields `(id: str)`). `=` binds (record fields
  `(id = "u_01")`).
- Match arms are single expressions; nesting is indentation, including an
  arm expression continuing on the next line.

## R2 — Modules

- Every file opens with
  `mod <domain> provides [...] uses [...] emits [...]`.
- `provides` names what the file defines; `uses` names external can items;
  `emits` is the union of errors the module can produce.
- Every `uses` entry must resolve to exactly one provider module. Unresolved
  or double-provided = compile error. (`go run ./tools/modcheck` enforces
  this over sketches today.)
- can-to-can dependencies are never re-declared in the consumer. `extern`
  is reserved for true foreign (non-can) imports, which are otherwise
  unsketched.
- Shared types are provided/used like functions.
- `state` cells (`state Name: T = lit`, a09) are module-private storage
  and never appear in `provides`.

## R3 — Naming

- Functions: `domain__verb`. Types: `Domain__Name`. Compiler-rejected
  otherwise. No version affixes, no abbreviations rule (open: fixed
  dictionary or free words?).
- Identifiers never change when versions change (see R4).
- State cells: `Domain__Name`. `state__get` / `state__put` are reserved
  for store operations (a09).

## R4 — Versions

- `rev N` (positive integer) is declaration metadata on types and functions,
  never part of the name. Integers count breaking-change generations;
  semver-style minor/patch promises are out.
- The compiler records rev -> content hash in `ai-lock.json`. Changing code
  without bumping rev = error. Bumping rev without changing code = error.
- `uses` must pin provider revs: `uses [db__get_user@4, Db__User@2]`.
  Unpinned use = error. No floating versions.
- Provider bumps never break pinned callers; upgrading a pin is an explicit
  one-line diff.
- Constants (`const name: TYPE rev N = literal`) carry `rev` like types
  and functions. A constant's semantic content is its expanded typed
  value, not its spelling (a78 slice 1): changing the value without
  bumping rev = error, and renaming with owner/revision/type/value
  unchanged is revision-inert.

## R5 — Errors

- Errors are scoped values, declared as `error <domain>.<name>(fields)`,
  e.g. `error auth.login_failed(user_id: str)`.
- No hand-picked global numbers. The compiler assigns stable ids recorded
  in `ai-lock.json`. Agents grep qualified names, never magic numbers.
- Every function declares `emits [...]`. Raising anything else = error.
  Entries are a conservative upper bound (a12): unrealized entries are
  allowed, but every entry must name a declared error.
- Errors are returned as values in expression position, never thrown.
  Exhaustive `match`/`on` handling of every declared outcome is required;
  no catch-all.

## R6 — Functions

- Pure by default; effects only via `call` + exhaustive outcome handling.
  A function touching the outside world says so in `uses`.
- State authority is declared beside `emits`: `effects [C.read, C.write]`
  (a09), transitive through local calls with no inference. Undeclared
  use and stale capabilities are errors; unproven authority blocks test
  execution like an open termination proof.
- Bodies are single expressions. No statements, no `return`.
- `match` is always exhaustive; a missing arm is a compile error, not a
  coverage warning. A value match may take several scrutinees at once
  (a28) (`match x, y`, arms `p1, p2 => …`): one `bool`/`"str"`/`_`
  pattern per slot, same arity on every arm, exhaustiveness proven over
  the product space. Arms win top-to-bottom; scrutinees evaluate exactly
  once, left to right. Call matches keep a single `call` scrutinee;
  multi matches take no `given`. Coverage stays one obligation per
  reachable source arm, not per product cell.
- Amendment (a90): an arm may relay a same-file local call
  directly (`=> forward call f(args)`), elaborated at check
  time into the full dispatched match; refused shapes are
  CAN3013, undeclared forwarded kinds CAN4001.

## R7 — Compile-time tests

- Each function carries `tests` in its signature, written as call specs:
  `name(inputs) => expected`, e.g.
  `happy(id = "u_01", pw = "secret") => Ok(user_id = "u_01")`.
- The compiler executes all tests in a hermetic sandbox during the build
  (no network, clock, random except via stubs). Any failure fails the build.
- Iteration must provably halt before anything runs or emits (a08,
  a11): direct self-recursion only, program-wide — same-file and
  cross-file cycles are refused (`CAN3005`). Admitted recursion
  declares `decreases p` over an `int` param, sits under the false
  arm of the canonical `p <= 0` guard (`CAN3009` otherwise), and
  passes `p - 1` at every self site (`CAN3008` otherwise). Cycles,
  unguarded recursion, and unproven decreases block execution;
  negative entries take the base arm and return a declared outcome.
- Each test evaluates with a fresh store built from `state` inits (a09);
  sequential tests cannot interfere.
- `ai build --prod` strips `tests` and `given` to zero shipped bytes.
- Amendment (a87): a row suffixed `pinned` is trusted acceptance;
  unmarked rows are proposed evidence and churn freely. The accepted
  baseline records pinned expectations; weakening, removing, or demoting
  one warns (`CAN6017`) after the clean gate, never blocking emit.
  Table-level pinning is refused: authority is per-row.

## R8 — External stubs (call-site `given`)

- Stubs live at the `call` site, not in a per-test map. Each `call` carries
  a `given` table: one exchange sequence per test, e.g.
  `flaky => [exchange args (id = "u_01") outcome db.down()]`.
- Every row is an exchange binding expected call args to one permitted
  outcome (a12): the table proves "this request received this permitted
  response", not merely the next response. Outcome-only rows are errors
  (`CAN3109`). Arg names resolve through the callee signature (positional
  call args included); every expected arg must arrive equal and the call
  must supply nothing unexpected, else the test fails.
- Error expectations are complete constructions
  (`=> auth.login_failed(user_id = "u_99")`), compared kind and payload.
  Bare error kinds prove nothing about the payload and are errors (a12).
- Tables are partial (a91): a test with no entry claims non-reach;
  reaching the call without a script fails the test. Unknown keys
  are errors. The retired `-` spelling is an error.
- Each evaluation consumes the head of that test's list. Calling with an
  empty list = error. Leftover entries at test end = error.
- Stub outcomes are restricted to the callee's declared `emits`, resolved
  through `uses` -> provider file. Invented outcomes = error.
- Same extern called twice = two tables (retries, sequences). Same site hit
  twice (loops) = multi-element list of exchanges.
- Three callee kinds (a07, a09): foreign calls (can via `uses`, externs)
  are stubbed through `given`; same-file helpers execute with no table;
  `state__get` / `state__put` execute against the test's store, also with
  no table. All three are deterministic per test. Helper-internal
  tables need scripts only for tests that reach them (a91); caller
  rows that never arrive need no entry.

## R9 — Retrieval surface

- An agent must be able to judge a file from formal lines only: `mod`
  header + signatures + `emits` + test-case names. No separate spec prose
  (Given-When-Then layer cut in review: unverifiable duplication).
- Layout contract (to enforce): types + signatures before bodies, so a
  reader can stop early and still know what the file does.

## R10 — Tooling (shipped with the language)

- Canonical formatter with zero options; formatting is law, not style.
- Editor grammar with a distinct scope per syntactic class
  (`editors/vscode/`, TextMate today; verified by `go run ./tools/gramcheck`).
- Committed executable checks for every rule above, earliest form first
  (today: `tools/modcheck`, `tools/gramcheck`, `go test ./...`).
- Real parser with source positions lives in `compiler/` (AST carries
  1-based lines on every decl, test, arm, and match).
- `canlc lsp` serves editor squiggles over stdio. Every keystroke
  re-runs: parse, naming (R3), rev pins (R4), provides/uses integrity (R2:
  provides names exactly what the file defines, unused uses warn),
  decision-table shapes (duplicate test names, unknown/missing args),
  call resolution (unknown callee, not-in-uses, calls outside a match
  scrutinee, proven self-recursion via `decreases` (`CAN3006`–`CAN3008`),
  local-cycle refusal (`CAN3005`)), given/test cross-checks (R8: missing
  given table, script no test selects, retired `-` spelling
  (`CAN3111`), stub outside
  callee emits, no table on deterministic calls (`CAN3106`)),
  emits integrity (R5: raising outside emits, unknown error kinds,
  undeclared emits entries), store authority (a09: undeclared `CAN3107`, stale
  `CAN3108`, unknown cell `CAN3001`), exhaustiveness proof (every
  violation, not the first), test runs, missing-tests warnings,
  unused params. World errors
  (bad uses, double definitions) report per-line and suppress only the
  execution-dependent checks, so one broken line never hides the rest.
  Each squiggle spans its exact token (callee name, test name, given key,
  emits entry, offending kind) in UTF-16 columns, not the whole line;
  only parser failures stay line-wide. Every squiggle carries a stable
  `CANnnnn` code (`compiler/code.go`); `canlc --format json` prints the
  same diagnostics as JSON lines, and the CLI enforces the full suite,
  so "no squiggles" and "compiles" are one gate. `canlc normalize` prints
  every decision-table outcome in canonical form, one sorted
  `mod.fn/test => value` line per test. Every build writes
  `errors.json` beside the emit: each kind with its fields, raisers,
  handling arms, and hitting tests. Test-per-arm law: every match arm
  must execute across the decision-table run (`CAN4107`) or carry the
  authorized structural certificate (certified identity relay);
  reporting distinguishes executed, certified, and uncovered —
  certified is never reported as taken or reachable. `emits` is a
  conservative upper bound, so unrealized entries are allowed (a12);
  coverage is assessed over green tables only. Type discipline (`CAN6xxx`): floats are ungrammatical
  (`CAN6001`, write `d"12.34"`); `dec` compares exactly in proofs;
  brands (`brand B is str`) are nominal with one gate in
  (`seal B("lit")`) and no way out except declared `extern`
  declassifiers, which are module-local, scripted through `given`
  like can calls, and imported by the TS emit from
  `./<stem>.externs`. Unknown types are `CAN6002`, any other mismatch
  `CAN6003`. Arithmetic (`+`, `-`, `*`, same-type operands only, exact
  (a10: unbounded ints, no overflow mode), no division) extends the
  `CAN6003` rule and yields the operand type. The checker runs in `checkSem` before test execution,
  so CLI and editor share it.
  The Cursor/VSCode client in `editors/vscode/` (`client/` + bundled
  `bin/canlc`) shows them; demo files live in `sketches/broken-login/`.
  TextMate regex remains for coloring only; semantic tokens are future.

## R11 — Compilation target: TypeScript (v0.1, decided)

- v0.1 emits TypeScript: not JavaScript (erasure would drop our proofs),
  not native (runtime/GC/FFI cost unjustified pre-product).
- Mapping: `emits` unions -> discriminated unions; proven-exhaustive
  `match` -> bare `switch` (no `default`, no `never` scaffolding — shipped
  code is pure business logic); externs -> typed TS imports; `state` cells
  -> module-scope `let`s with synthetic ok-unions at access sites (a09);
  numerics -> `bigint` ints and canonical-digit-string decs with exact
  `$canDec` helpers emitted inline only where used (a10: no imports,
  needs BigInt/ES2020; host `bigint` handling is an adapter contract).
  Exhaustiveness
  is proven by canlc's verify phase before anything runs or emits; `tsc`
  re-checks types and contracts, not coverage.
- Compile-time test evaluation stays inside the compiler (Python today),
  hermetic; prod emit strips `tests` + `given`. Reference implementation:
  `compiler/` (Go, subset: auth-login shape; `go run ./compiler`).
  Golden outputs: `sketches/auth-login/db.ts` + `auth.ts`,
  `sketches/retry-loop/retry.ts`, `sketches/counter/counter.ts`
  (each beside its `errors.json`), all gated by `go test ./...`.
- Source stays backend-agnostic. Go/Rust emitters are future options
  (Go: fast ops, clunky match emit; Rust: 1:1 mapping, slow builds),
  not v0.1 scope.

## Open questions (not decided)

1. Can two revs of one name coexist at runtime, or must the world upgrade
   in lockstep? Module-level releases?
2. Full named-arg unification for calls (`db__get_user(id = id)`), or keep
   positional calls beside named construction?
3. Helper-extraction rules — decided (a07): same-file callees need no
   pin and no `given`; the flagship shrunk via `auth__verify`.
4. Multi-call stub scripts beyond single-site lists (stateful externals,
   clocks, ordering across two different externs).
5. Type-version migration rules for shared types.
6. Prod-strip semantics (textual vs IR) and verifiability of stripped output.
7. `mod emits` vs union of `fn emits`: checked relation or free?
8. Formal grammar productions for `tests` / `given` / `match` (today prose).

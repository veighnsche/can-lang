# v0.2 — Machine-native artifacts (spec)

Status: all three shipped (codes + `--format=json`, `normalize`,
error catalog). Winner of the six-feature judge panel
(synthesis ref in session log 2026-09-16): the only candidate that closes
a gap between written spec and reality instead of adding surface.

## Goal

Every compiler output an agent consumes must be parseable without
squinting. Today the compiler speaks only prose (`canlc FAILED: ...`) while
REQUIREMENTS promises machine-checkable artifacts (R4 rev→hash lock, R7
prod strip, error catalogs). This bundle delivers three of them. Prose
messages stay byte-identical; JSON is added beside them, never instead.

Non-goals: `ai-lock.json` enforcement (still out), changing any human
message, changing emit output, new language syntax.

## Artifact 1 — JSON diagnostics (`--format=json`)

`canlc --out DIR --format=json file.can [...]` behaves exactly like the
default build, except compile failure prints one JSON object per line on
stdout (not stderr) and still exits 1:

```json
{"code":"CAN3201","sev":"error","file":"auth.can","line":20,"start":4,"end":9,"msg":"test extra has no script at the call to db__get (line 22)"}
```

- `file/line/start/end/sev/msg` mirror the `Diag` struct the LSP already
  serves (`compiler/lsp.go`); `start/end` are UTF-16 columns, `end <= start`
  means whole-line, same contract as `publishDiagnostics`.
- `code` is stable forever. Families:
  - `CAN1xxx` parse (grammar, braces ban, rev presence)
  - `CAN2xxx` declarations (naming R3, rev pins R4, provides/uses R2,
    double definitions)
  - `CAN3xxx` calls and decision tables (call resolution, given/test
    cross-checks R8, test shapes, missing tests)
  - `CAN4xxx` errors and proof (emits R5, exhaustiveness, test runs)
  - `CAN5xxx` world/tooling (siblings, unused uses/params)
- Implementation: add `Code string` to `Diag`; each check site sets it
  (new file `compiler/code.go` owns the registry + a uniqueness test).
  `run()` gains `--format` parsing; a `reportDiags` helper renders the
  same `[]Diag` the LSP path produces. CLI and editor therefore cannot
  diverge: one check suite, two renderings (this also closes the
  CLI/editor asymmetry — `compile()` must run the full suite, not just
  parse→world→proof→tests).
- Golden test: `TestGoldenJSONDiags` runs the compiler over
  `docs/archive/sketches/broken-login/` and byte-compares normalized JSON per file.

## Artifact 2 — Canonical values (`canlc normalize`)

`canlc normalize file.can [...]` runs every decision table and prints one
line per test, sorted, in canonical form:

```
db.db__get_user/known_user => Ok(failed_attempts = 0, id = "u_01", pw_hash = "secret")
```

Canonical grammar over `compiler/eval.go`'s `Value`:

- `str` → `"..."` with exactly `\"`, `\\`, `\n` escapes; `int` → decimal;
  `bool` → `true`/`false`.
- `ok`/`rec` → `Ok(k = v, ...)` with keys sorted byte-wise, never
  declaration order (declaration order is not stable across edits).
- `err` → `err(dotted.kind)` plus sorted fields when present.
- No whitespace except the single spaces shown; trailing newline per line;
  output ends with newline. Byte-identical reruns or it is a bug.

Implementation: `normalizeValue(v *Value) string` beside `describe()`
in `compiler/eval.go`; `canlc normalize` reuses the `parse →
buildProgram → runTest` path and prints `mod.fn/test => value` (test
expectations are not printed, outcomes are). Golden test over
`docs/archive/sketches/auth-login/`.

## Artifact 3 — Error catalog (`errors.json` per build)

Every successful `canlc --out DIR ...` also writes `DIR/errors.json`:

```json
{"kind":"db.down","fields":[],"raised_by":["db.db__get_user"],
 "handled_by":[{"fn":"auth.auth__login","arm":"on db.down _"}],
 "hit_by_tests":["auth.auth__login/flaky","auth.auth__login/down"]}
```

- `raised_by`: functions whose bodies construct the kind (reuse the
  `checkEmits` walk, minus diagnostics).
- `handled_by`: call-match arms pattern-matching the kind, with the arm's
  source row for `on` text.
- `hit_by_tests`: decision-table tests whose stub or expectation names
  the kind (reuse `stubbedKinds` + expectation scan).
- Sorted arrays, stable keys. Golden test: `DIR/errors.json` for
  `docs/archive/sketches/auth-login/` byte-compared in `TestGoldenAuthLogin` (extend,
  do not fork).

## Touch list

- `compiler/code.go` (new): code registry + `TestCodesUnique`.
- `compiler/lsp.go`: `Diag.Code`, `reportDiags`, `--format` in `run()`.
- `compiler/eval.go`: `normalizeValue`; `compile()` runs the full
  `diagnose`-equivalent suite (shared helper, not a copy).
- `compiler/check.go`: set codes at existing sites; expose the
  raise/stub/handle scans for the catalog (no logic change).
- `compiler/*_test.go`: JSON goldens, normalize goldens, catalog goldens.
- `REQUIREMENTS.md` R10: one line per artifact when shipped.

## Rollout order

1. Codes + `--format=json` (unlocks agent parsing; forces CLI/editor
   unification as a side effect).
2. `normalize` (needs nothing from 1, but lands cleaner after).
3. Catalog (reuses the walks hardened by 1–2).

## Open decisions (do not block 1)

- Whether `errors.json` should also ship in prod emit or stay a
  build-side artifact (leans: build-side; prod stays pure logic per R7).
- Whether `--format=json` should include passing tests as `"ok"` lines
  (leans: no; silence is success, catalog covers the positive inventory).

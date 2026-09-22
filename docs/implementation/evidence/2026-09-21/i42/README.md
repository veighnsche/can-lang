# I42 acceptance — complete admission applications

Closed 2026-09-21. The four manifest-backed example applications build
freshly from staged release layouts, assert their real ordinary Can
computation, and run every documented interaction end to end: staged
HTTP, a loopback provider stub with separately labelled raw-provider
results, disposable PostgreSQL, and the pinned browser harness.
The only compiler change is pool scope elision for assertion rows,
mirroring transaction handles.

Baseline `5c19ece` (I41) plus the I42 worktree. Apple M4 (Mac16,12)
darwin/arm64, macOS 27.0, Go 1.27.1, Bun 1.4.2 (`744846f84`, pinned
archive sha256 `90987a3a…6be1`, hash-verified), TypeScript 7.0.2 with
`@types/bun` 1.4.2 and `@types/node` 24.13.6 for the strict-TS leg,
Playwright 1.55.1 with Chromium 140.0.7339.186, disposable
PostgreSQL 17.11 (Debian 17.11-1.pgdg13+2, aarch64) over loopback TCP.

## Design consultations

[i42-jev](../i42-jev/decision.md): three fresh Jev consultations (all
prose rewritten, same facts/options). Unanimous for elide_pool (0.94,
0.48, 0.44): assertion rows omit `sql::pool` inputs while the harness
splices its scope token, mirroring transaction handles; the round-2
hedge toward review_needed was investigated and followed on the
merits, since no shipped fixture declares a pool input and
unprovisioned pool use still fails closed at the denied live
boundary. Majority for multi_module (0.59, 0.85, 0.80): each
application splits into `src/` modules behind provides/uses.
Unanimous for split_by_observability (1.0, 0.97, 1.0): wire-visible
rows over staged raw HTTP, swap rows in the pinned browser, one
native witness per outcome row. Judgments are advice; the checks
below are the proof.

## Implementation

- Compiler, `check/sql.go` + `check/http.go` + `emit/expressions.go`:
  `isPoolScopeRequest` admits the opaque pool as a scope request, so
  assertion rows elide pool inputs, `near` capture keeps its
  contract, and emission splices the harness scope token. Pool
  operations a row does not when-supply fail at the denied live
  boundary exactly like transaction handles.
- `check/sql_test.go`: `TestPoolInputsElideInAssertions` pins the
  elision — a `near sql::pool` loader with a pool-less assert row
  checks, and the `when` row still names the pool explicitly.
- `check/transaction_test.go`: the `callback scope input` reject now
  expects `does not fit expected type` instead of `call arity
  mismatch`. This is the intended consequence of elision: the
  pool-taking callback's zero-argument row is arity-correct, so the
  misuse surfaces where the pool reaches `transaction_execute`,
  matching the sibling `wrong callback target` case. Rejection is
  preserved; only the site moved.
- `examples/native-ai/`: plan → ordinary transformation → dynamic
  options → one batched judge call → explicit 0.7 confidence gate →
  concurrent coordinated reads → typed JSON report. Provider
  endpoints stay placeholders; handlers and error mapping are
  authored Can. Failure rows cover invalid plans
  (`llm::invalid_response`), invalid answers (`ai::invalid_answer`),
  DB outage (`unavailable`), and close failure (`unclosed`).
- `examples/account-search/`: descriptor-bound search over a
  `near`-captured pool, immutable row rendering, safe HTML, packaged
  assets, plus a validation form and a clock poll on the same page.
- `examples/form-validation/`: repeated/optional form values with
  422 feedback; no database.
- `examples/dashboard/`: polling with coordinated reads.
  `recent_count` races two identical `read_recent` reads through the
  pool — the first success wins with the same rows either way, and
  only `all_failed` maps the count to unknown. Each `when` outcome
  is queued twice, one per branch; the loser's lease settles late
  with no effect on the result.
- `tests/integration/testdata/applications/`: shared `seed.sql`
  (7 accounts incl. hostile/Unicode names, 3 dashboard rows,
  2 triage rows) plus the `seed-driver.ts` setup/teardown runner.
  No runtime migration API.
- `tests/integration/browser/{forms,accounts,dashboard}.mjs`: three
  pinned scenarios modelled on the I34 harness. Each aborts every
  non-localhost request and records a `report.json` plus screenshot.
- `tests/integration/applications_test.go`: staged legs (assert,
  double build with byte-identical IDs, usage entry), staged forms,
  the loopback provider stub with valid/invalid modes, live legs
  against disposable PostgreSQL, and the browser driver. Reports
  and screenshots are archived only when `CAN_BROWSER_EVIDENCE_DIR`
  is set; the committed `i42/{forms,accounts,dashboard}/` copies
  below come from the final green run.

## Verification

- `go test -count=1 ./...`: all 16 packages pass, including
  `tests/integration` (245s) with `CAN_BUN` 1.4.2,
  `CAN_BUN_ARCHIVE` (hash-verified), `CAN_TSC` (TS 7.0.2), and
  `CAN_TEST_POSTGRES_URL` on the disposable container.
- Staged: account-search 23, form-validation 14, dashboard 16, and
  native-ai 20 assertions, every one `real-can`; each rebuild is
  byte-identical and each entry prints usage without a database.
- Staged forms: 7 checks — CSP/pinned-script page, 200 and 422
  swaps, hostile-name escaping, 404, and 405 on the wrong method.
- Stubbed native-ai: valid plan plus exactly one batched judgment
  (`Draft`/`Pick`/`Likely`/`Rate` all observed once) with pool
  refusal surfacing `sql::connection_failed`; an output-less plan
  surfaces `llm::invalid_response` without advancing to the judge;
  an out-of-range answer surfaces `ai::invalid_answer`. All three
  are loopback fixture data, labelled raw-provider in the log.
- Live: 17 checks — account search over real rows (incl. hostile
  escaping, repeated/blank/missing query 422s, validate 200/422,
  summary, 404, 405), dashboard data (`3 accounts 3 recent`), the
  full triage report (`vip`/`ship`/0.9/2/2), and dropped-table 503s
  with reseed recovery on every leg.
- Browser (Chromium 140.0.7339.186, loopback only):
  `forms/` 8 checks (422/200 swaps, escaping, CSP, no inline
  handlers), `accounts/` 9 checks (live search/validate swaps,
  hostile escaping, summary poll), `dashboard/` 5 checks (live poll
  swap, repeated polls). Every request stayed on the serving
  loopback base; the only executing script is the pinned HTMX.
- `bun test runtime/test/`: 850 pass, 0 fail, 63,285 expectations
  (142 files) — unchanged, proving no runtime regression.
- `cataloguegen --check`, `modcheck` (42 fixtures), `gramcheck`:
  all pass.
- `gofmt`/`go vet` clean on every touched file (4 remaining `gofmt`
  findings are pre-existing in untouched files).
- Negative cases: malformed/blank/hostile forms → 422/escaped
  swaps; invalid plan/answers → typed exit-1 provider failures with
  no retry or advance; `all_failed` → unknown count → 503;
  dropped tables → 503/`unavailable` with recovery; pool refusal →
  `sql::connection_failed`; close failure → `sql::close_failed`;
  unknown routes/methods → compiler-owned 404/405; SIGTERM shuts
  every server down cleanly with the pool closed.

## Limitations

- Provider evidence is loopback stub data, never a live provider
  call; no shipped verdict depends on provider quality.
- The `unclosed` close-failure row is assert-level (when-supplied);
  no live leg forces a real close timeout.
- The dashboard race is redundancy, not a deadline: both branches
  issue the same read and the loser always settles.
- The `tscheck` project gate over `std/`/`sketches/` goldens still
  fails on pre-existing predecessor rot; I43/I44 own that cleanup.

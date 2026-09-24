# UP01 acceptance registry

Named positive (+) and negative (−) cases for every Fix finding and the
invoice gate. Status values: **pass** (observed green at baseline),
**fail** (observed failing/skipped at baseline — recorded separately, never
counted as completion), **unimplemented** (new behavior with no passing run;
no claim may rely on it).

Current-failure evidence below is baseline characterization, not completion
credit. New-behavior implementation and proof belong to UP02–UP23; this
registry only names the cases and their baseline state.

## B01 — structural browser audit (proof: UP10/11/15/23)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| B01-P1 quoted host words | `"Bun."`, `"node: introduction"`, `"require("` in string data compile | fail — `browser-audit-results.json` records literal rejection |
| B01-P2 full closure audit | reachable runtime modules resolve structurally with origin/span diagnostics | unimplemented |
| B01-N1 real host operation | actual `Bun`/`process`/`require` use rejects with location evidence | fail — current scan skips runtime bodies, concatenation bypass exists |
| B01-N2 server-only import | server SQL/env/fs capability reachable from browser `main` prevents publication | unimplemented |

Baseline evidence: `evidence/2026-09-24/post-upgrade-review-961f921/browser-audit-results.json`,
`compiler/internal/browser/browser.go` (raw-substring scan).

## B02 — Unicode regex advancement (proof: UP03/23/24)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| B02-P1 empty Unicode matches | `😀x` empty pattern with `u`/`v` → starts `[0,2,3]` | fail — saved starts are `[0,0,0,0,0]` |
| B02-P2 non-Unicode parity | same probe without Unicode flag → `[0,1,2,3]` | fail — same probe output |
| B02-P3 nonempty/caps/captures | nonempty matches, capture defaults, zero/one/max caps keep behavior | pass (existing `runtime/test/text.test.ts`) |
| B02-N1 invalid input | bad pattern/flag/budget keep named failures; no unbounded materialization | pass (existing suite) |

Baseline evidence: `post-upgrade-review-961f921/core-probes/regex-output.txt`,
native expectation in `selected-behavior/regex-native-probe.md`, current
`runtime/text.ts` `lastIndex++` iteration.

## U01 — shared handler-free action declaration (proof: UP05/08/16/19/20/21)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| U01-P1 one locked package | server and browser import the same action symbols from one locked `invoice_contract` | unimplemented — grid mirrors declarations with stubs |
| U01-P2 handler-free parse/check | the three selected declarations parse, check, expose canonical metadata | unimplemented — grammar requires `handles` |
| U01-N1 `handles` rejected | `handles` clause in shared declaration fails with span | fail — `handles` is currently required (`syntax/native.go`) |
| U01-N2 no mirrors/stubs | mirrored local action, missing/extra case, computed symbol diagnosed | unimplemented |

Baseline evidence: `examples/invoice/src/web/web.can:5-36`,
`examples/invoice-grid/src/web/web.can:11-49` (mirrors + `save_stub`/`load_stub`),
`compiler/internal/syntax/native.go:402` (`action requires a handles clause`).

## U02 — request-aware mount and services (proof: UP06/08/09/16/19/22)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| U02-P1 explicit pool/origin | one startup pool + validated origin captured by mount callables; handlers receive `http::request` | unimplemented — handlers take only captures/body |
| U02-P2 protected operation | each GET/save rechecks session, membership, tenant/invoice, revision in-transaction | unimplemented |
| U02-P3 token revocation | retained header/body/upgrade/snapshot use after drainage fails with resource-state failure | fail — WeakMaps retain snapshots after `abandonRequest` |
| U02-N1 wrong binding | missing/mistyped pool/origin, wrong request position, generic handler rejected | unimplemented |
| U02-N2 denied actor | foreign/missing invoice and revoked membership commit nothing, disclose nothing | unimplemented |

Baseline evidence: `selected-behavior/experiments/request-lifetime/findings.md`,
`examples/invoice/src/web/web.can:46-53` (per-request pool, body session token).

## U03 — declared routes and contract edits (proof: UP08/09/13/19/20/21/22)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| U03-P1 direct captured route | declared GET reaches the live authenticated handler; no test rewrite | fail — API serves query spelling, captured path 404s (`gate5_frontend_test.go:976`) |
| U03-P2 route edit moves both | literal change rebuilds both targets; old path 404/new path 200 live | unimplemented |
| U03-P3 field/leaf/limit edits | capture/field/leaf/status/limit/body-mode edits propagate or diagnose with spans | unimplemented |
| U03-P4 replay semantics | authorized same-ID replay returns recorded result; changed payload conflicts | unimplemented |
| U03-N1 pre-handler classes | malformed capture/JSON, wrong media, over-limit stay 400/415/413, never app `invalid` | unimplemented |
| U03-N2 revoked replay | revoked actor replay gets nondisclosing 403, no old result | unimplemented |

Baseline evidence: `tests/integration/gate5_frontend_test.go:437,869,976-993`
(query rewrite, served query spelling, `L-route`/`L-wire`/`L-unlink` limits).

## U04 — supported browser delivery (proof: UP04/07/10/11/13/15/18/20/23/25)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| U04-P1 empty app boots | zero-arg `main` runs once after DOM readiness from compiler-owned entry | unimplemented — grid `main(str[] args)` needs test boot |
| U04-P2 canonical query keys | valid/absent numeric `tenant`/`invoice` reach the expected view | unimplemented (`browser::query_parameter` does not exist) |
| U04-P3 paired publication | server serves report-selected digest JS/map/table under CSP; same-input rebuild identical | unimplemented (`--browser-manifest` does not exist) |
| U04-P4 runtime parity | failures/diagnostics/equality/int64/async/disposal/coordination in named Chromium+WebKit | fail — shims + sync ALS stand in for profile |
| U04-N1 bad bootstrap | `main(str[] args)`, malformed `%ZZ`, duplicate alias, oversized query fail finitely | unimplemented |
| U04-N2 tamper/mismatch | tampered/unmanifested asset, missing map, mismatched manifest block publication | unimplemented |

Baseline evidence: `tests/integration/browser/build-grid.mjs` (boot + 4 shims),
`examples/invoice-grid/src/web/web.can:1024-1033` (`main(str[] args)`),
`selected-behavior/experiments/browser-owner/findings.md`,
`selected-behavior/query-native-probe.md`.

## U05/S02 — cancel policy and fault reporting (proof: UP11/13/20/23)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| U05-P1 sync cancel | matching key/submit prevented during native dispatch; `defaultPrevented` immediate | unimplemented (`browser::on_cancel_*` do not exist) |
| U05-P2 ordinary events | mismatched key, noncancelable, ordinary subscriptions dispatch once, uncanceled | unimplemented |
| S02-P1 one sanitized report | failing callback yields one located diagnostic; later events/disposal still work | fail — settlement discards completions/rejections |
| U05-N1/S02-N1 boundaries | unadmitted kind rejected; disposed view cannot cancel; no double dispatch; no raw Event leak | unimplemented |

Baseline evidence: `runtime/platform/browser.ts` event settlement,
`selected-behavior/experiments/event-cancel/findings.md`.

## S01 — checked HTML guards (proof: UP08/12/19/22/23)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| S01-P1 admitted swaps | declared 200/422/409/403/503 swap `inner` into exactly the admitted connected target | fail — 503 does not swap; only 422 excepted |
| S01-P2 missing target | absent/disconnected target and in-flight loss report `action::missing_target`, status preserved | unimplemented |
| S01-N1 unsafe shapes | OOB/partial/extra/redirected tasks and response-control headers mutate nothing | unimplemented |
| S01-N2 post-commit fault | committed write + render/swap failure stays uncertain 500, never rollback | unimplemented |

Baseline evidence: `runtime/platform/html.ts` HTMX policy,
`selected-behavior/experiments/htmx-guard/findings.md`.

## S03 — grid focus/notice/pending (proof: UP20/23)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| S03-P1 attached focus | add/move/reorder focuses an attached control by stable key; verified via `activeElement` | fail — recorded `L-focus-row` (pre-attachment focus) |
| S03-P2 durable notice | blocked-save notice survives rerender, announced in live region | fail — recorded `L-notice` (erased notice) |
| S03-P3 pending state | saving visible until its own attempt settles | fail — recorded `L-flight` (absent indicator) |
| S03-N1 late/disposed | late reply, double save, removed view cannot clobber newer edits or focus detached nodes | unimplemented |

Baseline evidence: `tests/integration/gate5_frontend_test.go:12`
(`L-focus-row`, `L-notice`, `L-flight`).

## P03 — public generic composition (proof: UP02/14/17/21/24)

| Case | Expectation | Baseline status |
| --- | --- | --- |
| P03-P1 identity emits | public `identity<T>` checks **and** fully emits; no symbolic type in emitted graph | fail — symbolic `types.Parameter` reaches `NativeTypeDeclarations` |
| P03-P2 helper chains | `pass`/`wrap`/`nested` (incl. `nested<T> → identity<box<T>>`) compile across locked packages | fail — opaque arguments rejected outside self-recursion |
| P03-P3 atomic SCC | stationary/permutation mutual recursion validates atomically; `nested<int>` calls `identity<box<int>>` | unimplemented |
| P03-N1 illegal bodies | callee arithmetic on opaque formals fails at declaration, not at use | unimplemented |
| P03-N2 growth/false proof | expanding `T[]` cycle, private-template borrow, failed-component reuse rejected with chain evidence | unimplemented |

Baseline evidence: `selected-behavior/experiments/generic-recursion/findings.md`,
`compiler/internal/check/specialize.go` (opaque-argument rejection),
six-fixture probe referenced by UP02/UP14/UP17.

## Invoice gate rows (proof: UP19–UP23, UP25)

| Gate | Baseline status |
| --- | --- |
| Shared source (one locked package, no mirrors/stubs/handlers) | fail — duplicated declarations, stubs, per-target wires |
| Direct routes (captured GET/POSTs enter bound handlers, no rewrite) | fail — query spelling served, captured 404 |
| Explicit service and actor (one pool/origin, per-request auth, revocation) | fail — per-request pool, body token, retained snapshots |
| Finite transport outcomes (declared cases vs 400/404/405/413/415/500) | fail — 503 no-swap, no checked case tables |
| Protected effects and uncertainty (once-only mutation, replay, 7-day ledger, revisions) | unimplemented |
| Browser application (zero-arg boot, query keys, grid flows, focus/notice/pending, cancel, reports) | fail — boot/shim path, `L-*` limits |
| Browser/runtime closure (structural audits, named Chromium+WebKit parity, CSP pairing) | fail — substring audit, Chromium-only harness |
| Contract-edit table (6 rows: route, capture rename, field rename, leaf rename, status/limit, body mode) | unimplemented |

No row above is green at baseline except the B02 retained-behavior cases
(B02-P3, B02-N1). A skipped required leg at UP24/UP25 is not a pass.

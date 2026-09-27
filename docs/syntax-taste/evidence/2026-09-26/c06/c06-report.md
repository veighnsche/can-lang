# C06 report — Qualify W1 across both apps and three browsers (2026-09-26)

## Scope

W1 acceptance legs on the final shared controls/actions and the H06
generation policy: positive (both apps check/emit/run × 3 browsers),
negative (breaking control-API edit diagnoses the non-edited app),
capture-edit (route/capture/wire/body/result-leaf edits rebuild both
targets or diagnose), failure-path (late reply after unmount, disposal
mid-save, IME + normalization caret preservation), integration
(truthful denied/not-found paired reads), plus the generation-policy
legs (old-page/new-server and rollback refusal, blocking refresh).

## Slices (all [C06]-tagged)

- Design: three fresh Jev consultations (lowering shape, pin scope,
  htmx channel) — unanimous; see `jev/`.
- H06 E slice, server (E-merge candidate: no E worker is active):
  `runtime/platform/action-routes.ts` (mount-mode threading,
  handshake vocabulary, exact-409 builder, exact-shape test, slot
  read), `router.ts` (dispatch-time refusal before action logic),
  `assets.ts` (once-per-startup manifest pin for paired builds),
  `server.ts` (pin, explicit-table refusal, slot splice).
  Tests: `runtime/test/generation.test.ts` (12).
- H06 E slice, client (E-merge candidate): `action-json.ts` (slot
  header, 409 pre-classification, `generation_mismatch` outcome
  lowered to `http::transport_failed` phase `generation`),
  `htmx-guard.ts` (`htmx:config:request` stamping) + guardgen and all
  integrity re-pins (new guard sha384
  `sha384-C5A6uJ5FDnfm4n0j9TRlnVP+OIz//4VRs7OcvEYgUzCRevIwXmktURVNyPxYvOpQ`).
  Tests: `runtime/test/generation-client.test.ts` (4) + guard
  stamping legs.
- H06 C slice: shared `stale_notice` + `generation` kind arms in the
  notices library (relocked into both apps); grid `stale_state`
  (prompt survives edits, blocks saves/replays, resolves through
  observed agreement or refresh); phase-branching failure arms in
  `send_save`/`load_pressed`. Grid 394/394, compare 221/221,
  server 347/347 asserts green.
- W1 check/build legs: `tests/integration/c06_test.go`
  (`TestC06Positive`, `TestC06NegativeAPIBreak`,
  `TestC06CaptureEdits`).
- Served legs: `/invoice-compare` shell route on the invoice server;
  harnesses `compare.mjs` (13), `w1-grid.mjs` (12), `drift.mjs` (4);
  `TestC06ServedMatrix` (compare + grid + rollout + rollback × 3).

## Design record (from Jev + contract analysis)

- Lowering: a distinct `generation_mismatch` fetch outcome lowered
  into the declared bound as `http::transport_failed{phase:
  "generation"}`. Runtime-only; follows the E04 timeout/cancelled
  phase precedent. No catalogue/checker/emitter change, so no A
  slice is needed.
- Scope: pin on paired builds only (unpaired servers keep
  bit-identical behavior for curl/tests/non-browser callers);
  enforce on JSON + form mounts; document mounts exempt
  (navigations cannot carry headers and always load the current
  document; no program mounts a document action today).
- htmx channel: the owned guard stamps `detail.ctx.request.headers`
  on `htmx:config:request` from the live slot (htmx 4.0 verified to
  expose the mutable headers record at that event).
- 409 classification: the exact mismatch shape maps ahead of any
  declared 409 domain case; domain 409s (e.g. `grid_conflict`) keep
  their path. Refusals never retry and never report (expected
  failure, no E06 occurrence).
- Noted refinement: 409 classification now observes the body, so a
  wire fault while reading a 409 body reports honestly
  (timeout/aborted/transport) instead of `unexpected_status`. Exact
  preservation would have lied about caller cancel.

## Honest N/A cells (not passes)

- Compare makes no server calls (C04 seeds-only design, kept): no
  paired reads, no mismatch outcomes, no refresh prompt exist in
  that app. Its C-slice consumption is the shared notice vocabulary
  (asserted in-consumer). Adding an unreachable mapping would be
  dead code.
- The form-page (htmx) mismatch path refuses with the exact 409,
  never swaps (4xx noSwap policy), and never retries; the blocking
  *prompt* there is a follow-up (no Can browser code runs on that
  page to render one).
- No document-mode GET action exists in any program, so captured
  document reads have no live consumer to qualify; the W1
  "captured HTML reads" leg is evidenced on the served read surface
  that exists (captured JSON loads + query-addressed pages through
  paired builds).

## Follow-ups (out of scope, not smuggled in)

- E04 checker patch (mandatory caller bounds, cancel-signal source,
  server budget config): no W1 leg needs caller bounds, and
  mandatory bounds would break every existing program. Needs its
  own design task.
- Converting the query-addressed form page to a captured
  document-mode GET (the UP19 deferral C03 unblocked at the checker
  level): unowned; no 2026-09-26 task covers it.
- Form-page blocking refresh prompt mechanism.
- `createJsonActions().mount` (test-only legacy exact-route mounts)
  bypasses the handshake by construction; emitted programs use
  `$canActionRoutes` bindings and are covered.

## Evidence

- `jev/`: 3 requests + responses + wording audit + findings.
- `legs/`: per-leg harness reports + screenshots (see `c06-legs.json`
  for the per-app per-browser matrix with versions and hashes).
- Runtime: `bun run check:runtime` green; affected suites green
  (list in the matrix section).
- Regression: `TestGate5ServedMatrix` at close (see below).

## Served matrix

`TestC06ServedMatrix` green in 643s (2026-09-26, MacBook Air,
`CAN_BUN_ARCHIVE` pinned zip, container Firefox 141 over
`CAN_FIREFOX_WS`, loopback forwarders; chromium 140.0.7339.186,
webkit 26.0, firefox 141.0 — versions from the leg reports). 99
served checks, all passing, no limitations, no aborted requests, no
page faults:

| Suite | Legs × browsers | Checks |
| --- | --- | --- |
| compare (second app boots, edits, review, caret, composing input, mirror, drop-left disposal, silence after drop, ledger clean) | 3 | 13 × 3 = 39 |
| w1-grid (boot, slot/header proof, missing-id + page denials, 404, caret, save/load refusal prompts, refresh resolves) | 3 | 12 × 3 = 36 |
| drift rollout (V1 page, V2 server: real 409, prompt, no retry, never ran, refresh resolves) | 3 | 4 × 3 = 12 |
| drift rollback (V2 page, V1 server: same, symmetric) | 3 | 4 × 3 = 12 |

Per-leg reports + screenshots: `legs/`; machine-readable matrix:
`c06-legs.json` (versions, checks, request counts, sha256).

Check/build legs (same worktree, same day): `TestC06Positive`
(grid 394/394, compare 221/221, server 347/347 asserts, all
real-can; compare browser rebuild-stable; compare pairing verified),
`TestC06NegativeAPIBreak` (dropped `parse_qty` fails both consumers
with `fields::parse_qty is private` + consumer file/region),
`TestC06CaptureEdits` (route rebuilds both targets; capture/wire/
body/leaf diagnose in both targets with located messages).

Runtime suites: `bun run check:runtime` green (lint + format +
typecheck); `generation.test.ts` 12/12, `generation-client.test.ts`
4/4, `htmx-guard.test.ts` 35/35, plus the server/router/mount/fetch
regression files (124 + 48, all passing).

Regression: `TestGate5ServedMatrix` green in one run at close
(1179s, 2026-09-27, same worktree/host/pinned toolchain): grid
32×3, conformance 17×3, empty 4×3, invoice 20×3 — 219 served
checks, all passing (see execution README row). Two earlier post-fix attempts died in the shared
build stage on per-root 5s supervision timeouts under host load
(load avg 41 from GUI renderers + canlc workers); failing roots
differed per attempt and the same builds pass directly, so the
attempts were discarded as environmental, not code, failures.

## Main-integration re-qualification (2026-09-27, main `932924f7`)

After integrating all 10 slices onto main (design `3286e5cc`
through record `932924f7`, interleaved with concurrent-lane commit
`85566db4`), the full C06 battery was re-run on main with the same
pinned toolchain (`CAN_BUN_ARCHIVE=/tmp/bun-darwin-aarch64.zip`,
container FF141 via `CAN_FIREFOX_WS`, Playwright 1.55.1):

- `TestC06Positive` PASS (517s): grid 394, compare 221; compare
  pairing `bc1af0119dcb`.
- `TestC06NegativeAPIBreak` PASS (8s).
- `TestC06CaptureEdits` PASS (816s): route/capture/wire/body/leaf
  5/5 sublegs.
- `TestC06ServedMatrix` PASS (942s): compare 13×3, w1-grid 12×3,
  drift rollout 4×3 + rollback 4×3 = 99/99 across Chromium
  140.0.7339.186, WebKit 26.0, Firefox 141.0. Served pairings:
  server 347 assertions, compare `e01fe43690e1`, grid
  `e2bfe00b9f1a`, drift `6993bc8ced10` vs `a3c447785bf6`.

Command: `go test ./tests/integration/ -run 'TestC06' -count=1
-timeout 90m -v` (the default 600s `go test` timeout kills this
battery mid-run; the 90m timeout is required, not a code issue).
Result: `ok ... 942.100s`, EXIT=0, zero failures. Counts match the
worker-run matrix exactly; no drift from the concurrent-lane
interleave. Full log retained by the coordinator at
`/tmp/c06-main-qual.log` for this session.

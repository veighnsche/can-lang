# I32 acceptance — HTTP values, forms and exact router

Closed 2026-09-21. All 22 I32 catalogue operations are admitted, checked,
emitted, and verified end to end. Server lifetime operations
(`make_server_config`, `server_start`, `server_wait`, `server_stop`) stay
excluded for I33.

## Design consultations

- `../i32-jev/decision.md` (prior): snapshot responses, collision errors.
- `../i32-compiler-jev/decision.md`: mounts are ordinary checked calls; the
  lowered `$canOwnCallable` wrapper serves directly as `MountedCallback`;
  no `ir/http.go`, no second pipeline. Paths require string literals at
  check time with runtime mount validation authoritative (2-1 split on
  paths resolved by the `staticInputs` contract; dissent was low-confidence).
- `../i32-handler-assert-jev/decision.md`: assertion rows elide
  `http::request` root inputs; the runner supplies one harness token per
  root; body readers must carry `when`-supplied outcomes (elision over an
  explicit constructor, 2-1 with one near-tie).
- `../i32-opaque-assert-jev/decision.md`: bare `ok` is allowed in assertion
  expectations only, for C-excluded opaque/callable results only
  (unanimous). The runner passes undefined-expected against opaque actuals.
  Staged loopback verifies response content.

## What changed

Check: new `compiler/internal/check/http.go` (generic JSON/form
specialization, fixed route contracts, literal-path validation, mount
enforcement); gate admission plus specialization dispatch in
`check/program.go`/`check/specialize.go`; route checks in
`check/completions.go`; route-reference rejection in `check/callables.go`;
root-argument elision plus bare-opaque expectation gates in
`check/assertions.go`; `syntax.ScopeExpr` (synthesized, never parsed);
`ir.ScopeRequest`; `BareOpaque` completion context honored only for
`Opaque`/`Callable` results.

Emit: `$canHTTPRequests` (`createRequests<ConcreteHeader>`),
`$canHTTPResponses` (`$canCreateHTTPResponses`, distinct from the AI
`$canCreateResponses` factory), `$canRouter` state, bindings,
per-specialization descriptor closures, `$canHTTPKinds` domain membership,
and `$canScopeRequest` lowering in program and assertion roots.

Runtime: `denyLiveBoundary` on all eight readers (`scoped` = supplied
boundary over a harness-provided scope); `scopeRequest` harness token in
`assert/context.ts`; undefined-expected acceptance for opaque tokens in
`assert/runner.ts`; `FormSchema.kind` broadened to `string` (codec
`SchemaNode` precedent) so embedded descriptors check strictly.

## Verification

- `go test ./compiler/...`: all packages pass.
- `bun test runtime/`: 254 pass, 0 fail, 21,752 expectations (50 files).
- Strict `tsc` (isolated 7.0.2 install): runtime drafts, runtime tests,
  and the staged `entry.ts` all clean.
- `tests/integration/http_test.go`: staged bundle asserts 13 Can rows
  (`supplied-completion` + `real-can` evidence), runs, builds, strict-checks,
  then drives 19 loopback requests with 34 response assertions through
  compiled handlers via a private test bridge (kept separate from the I33
  server API): method echo,
  query present/missing/repeated/plus/percent, JSON roundtrip plus media
  and malformed mappings, form present/empty/absent/malformed mappings,
  generic-inferred and wrapper handlers, 404, 405 with `Allow`, and HEAD
  rejection.
- Shared `runtime/test/http-paths.json` corpus: 19 accept + 25 reject
  verdicts agree between the Go checker and the runtime mount; lone
  surrogates and invalid bytes pinned inline on each side.
- Full `tests/integration/` suite passes with the staged gates enabled.
- Source programs exercise every admitted operation; no TS casts hide
  incompatible contracts; no predecessor HTTP code is referenced.
- Pinned Bun 1.4.2 archive and executable hashes verified against
  `distribution/target.json` before staged runs.

## Documented interpretations and limits

- References to `route_get`/`route_post` themselves are rejected: static
  paths must be visible at the direct mount call.
- Route callbacks must be `callable` references resolving to exactly
  `(http::request) -> http::server_response` with empty emits; locals and
  call results are rejected as unnamed.
- Valued `ok` expectations for opaque/callable results are rejected at
  check time (they could never compare); bare `ok` states the C exclusion
  explicitly. Failure expectations are unaffected.
- Rows for scope-elided functions use fixed arguments or literal spreads;
  runtime-length spreads cannot match gapped inputs. Variadic scope tails
  are never elided.
- One harness token per assertion root; opaque when-row arguments compare
  by token identity.
- 204/205/304 handlers follow the bool-wrapper execution pattern at Can
  level; wire body exclusion is proven at runtime level.

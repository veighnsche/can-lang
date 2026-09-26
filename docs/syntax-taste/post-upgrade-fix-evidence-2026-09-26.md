# Post-upgrade fix evidence: 11 rows to source and passing tests

26 September 2026 · code candidate `08904291`
(plus the UP23 verdict step and this documentation; no product-code
change after the candidate).

This is the UP26 evidence record for the 11 **Fix** findings selected
in the [disposition ledger](post-upgrade-dispositions-2026-09-24.md).
Each row links the shipped source and the proving tests. Case IDs
(B01-P1, …) come from the [UP01 acceptance
registry](evidence/2026-09-24/implementation-up01/acceptance-registry.md).
The [selected-behavior packet](post-upgrade-selected-behavior-2026-09-24.md)
is the contract these rows implement; the [reconciliation](post-upgrade-reconciliation-2026-09-24.md)
is the pre-fix baseline (its "still present / remain open" notes are
superseded row by row below).

**Evidence status.** Staged gates (UP21–UP24) ran on this machine
(darwin/arm64, pinned Bun 1.4.2, Chromium 140.0.7339.186, WebKit 26.0,
PostgreSQL 17.11): six sharded full-tree logs at the qualified
commit, archived with digests under
[evidence/2026-09-26/shards/](evidence/2026-09-26/up24/README.md#shards--full-tree-log-set)
— exit 0 on every shard, zero failures. Exact commands in the
[clean-build recipe](post-upgrade-clean-build-2026-09-26.md#full-staged-qualification).
Installed-target evidence (UP25) is deferred to the x86 Linux machine;
no installed run is claimed here. Every proving test below is committed
and rerunnable, so the evidence is independently verifiable.

**Boundaries.** The 19 deferred proposals and 9 retentions in the
disposition ledger are unchanged: none of their behavior is described
as shipped, and the completion audit re-verifies each exclusion. No new
measured performance, token or portability claim is made.

## B01 — structural browser audit

Quoted host words (`"Bun."`, `"node: introduction"`) compile; actual
forbidden imports/operations and reachable runtime dependencies reject
with location evidence.

- Shipped source: [structural audit](../../compiler/internal/browser/audit.go),
  [script lexer](../../compiler/internal/browser/script.go),
  [module graph scan](../../compiler/internal/browser/browser.go),
  [bundle closure audit](../../compiler/internal/driver/browser_bundle.go).
- Proving tests: `TestBrowserBuildTarget`
  (`tests/integration/browser_build_test.go`, quoted-host acceptance
  plus rejection legs) and the `TestGate5ServedMatrix` audit legs
  (`tests/integration/gate5_frontend_test.go`, full-closure
  server-import/secret scan of served bytes).
- Cases: B01-P1 quoted host words, B01-P2 full closure audit, B01-N1
  real host operation, B01-N2 server-only import.

## B02 — Unicode regex advancement

`text::matches` over `😀x` with an empty Unicode pattern reports
starts `[0,2,3]`; non-Unicode, nonempty, capture and cap cases keep
their behavior.

- Shipped source: [capped native `matchAll` loop](../../runtime/text.ts)
  (code-point steps under `u`/`v`, code-unit steps otherwise; UTF-16
  start/end; absent captures read `""`).
- Proving tests: `runtime/test/text.test.ts` ("empty matches advance
  like native matchAll…", plain-flags parity leg, retained
  nonempty/caps/capture/invalid-input legs), run under the pinned Bun
  by the Go suite's runtime leg.
- Cases: B02-P1 empty Unicode matches, B02-P2 non-Unicode parity,
  B02-P3 nonempty/caps/captures, B02-N1 invalid input.

## U01 — shared handler-free action declaration

Server and browser import the same action symbols from one locked
package; no `handles` row, mirror declaration or stub handler remains.

- Shipped source: [shared contract](../../shared/invoice-contract/src/invoice_contract/invoice_contract.can)
  (three handler-free actions), vendored identically into
  [invoice](../../examples/invoice/vendor/billing/src/invoice_contract/invoice_contract.can)
  and [grid](../../examples/invoice-grid/vendor/billing/src/invoice_contract/invoice_contract.can);
  [checked declaration](../../compiler/internal/check/actions.go);
  [`handles` removal diagnostic](../../compiler/internal/syntax/native.go)
  ("action handles was removed; bind handlers with action::mount").
- Proving tests: `TestGate3ContractEdits`
  (`tests/integration/gate3_matrix_test.go`, shared-identity setup and
  handler-free parse/check) and `TestInvoiceGridPagePaired`
  (`tests/integration/invoice_test.go`, one locked package served to
  both pages), plus the no-mirror source/asset audits in the
  gate3/gate5 suites.
- Cases: U01-P1 one locked package, U01-P2 handler-free parse/check,
  U01-N1 `handles` rejected, U01-N2 no mirrors/stubs.

## U02 — request-aware mount and services

One startup pool and one validated public origin are explicitly bound;
every handler derives the actor from server-controlled request state
and rechecks authorization at the protected operation.

- Shipped source: [`action::mount` checking](../../compiler/internal/ir/action.go)
  and [emission](../../compiler/internal/emit/action_bindings.go);
  [server dispatch with token abandon/revoke](../../runtime/platform/server.ts);
  [exact-Origin policy](../../runtime/platform/csrf.ts);
  [JSON/form adapters](../../runtime/platform/action-json.ts).
- Proving tests: `TestGate3ServerMatrix` (explicit pool/origin,
  protected operation, denial legs),
  `TestGate3Lifecycle` (one pool, revocation, startup/shutdown),
  `TestGate3Concurrency` (protected-operation races),
  `TestGate4FaultMatrix` (denied-actor legs).
- Cases: U02-P1 explicit pool/origin, U02-P2 protected operation,
  U02-P3 token revocation, U02-N1 wrong binding, U02-N2 denied actor.

## U03 — declared routes and contract edits

The declared path reaches the live authenticated handler directly;
route/capture/field/leaf/body edits rebuild both targets from the one
contract or diagnose; replay keeps its exact semantics.

- Shipped source: [route builder and
  validation](../../runtime/platform/action-routes.ts), [checked
  action references](../../compiler/internal/emit/actions.go),
  [browser client projection](../../runtime/platform/action-client.ts).
- Proving tests: `TestGate3ServerMatrix` (direct captured route, no
  rewrite), `TestGate3RouteRebuildLive` (route edit moves both
  targets), `TestGate3ContractEdits` (field/leaf/limit edit table),
  `TestGate3AdapterMatrix` (pre-handler 400/415/413/404/405 classes),
  `TestGate4FaultMatrix` + `TestGate3ReplayExpiry` (disconnect/replay,
  retention, revoked replay).
- Cases: U03-P1 direct captured route, U03-P2 route edit moves both,
  U03-P3 field/leaf/limit edits, U03-P4 replay semantics, U03-N1
  pre-handler classes, U03-N2 revoked replay.

## U04 — supported browser delivery

The empty app and invoice grid boot through the maintained
compiler-owned build/bundle path; the server publishes the selected
content-addressed same-origin asset under its CSP from a verified
pairing.

- Shipped source: [browser bundle
  stage](../../compiler/internal/driver/browser_bundle.go),
  [pairing verification](../../compiler/internal/driver/pairing.go)
  (`--browser-manifest`), [browser target](../../compiler/internal/driver/browser.go),
  [query bootstrap](../../runtime/platform/browser.ts)
  (`browser::query_parameter`).
- Proving tests: `TestGate5ServedMatrix` (empty-app, grid,
  conformance and invoice legs on Chromium + WebKit; pairing,
  retention, tamper and closure legs), `TestInvoiceGridPagePaired`
  (paired publication), `TestBrowserBuildTarget` (bootstrap
  rejections).
- Cases: U04-P1 empty app boots, U04-P2 canonical query keys, U04-P3
  paired publication, U04-P4 runtime parity, U04-N1 bad bootstrap,
  U04-N2 tamper/mismatch.

## U05 — cancelable admitted events

An admitted keyboard/submit event can request native `preventDefault`
through a registration policy; ordinary events never cancel and
disposed views cannot cancel stale events.

- Shipped source: [cancel-policy
  registration](../../runtime/platform/browser.ts)
  (`browser::on_cancel_key` for `keydown`/`keyup` plus an exact key,
  `browser::on_cancel_event` for `submit`; synchronous
  `preventDefault` before the single immutable-snapshot dispatch).
- Proving tests: `tests/integration/browser/conformance.mjs` "cancel"
  legs via the gate5 conformance suite (matching-key/submit
  prevention, mismatched-key and noncancelable dispatch-once,
  disposal legs).
- Cases: U05-P1 sync cancel, U05-P2 ordinary events, U05-N1
  unadmitted/disposed boundaries.

## S01 — visible HTML 503

A real 503 keeps status 503 and visibly replaces the specified target
feedback; every admitted status swaps exactly its connected target,
and guard rejections mutate nothing.

- Shipped source: [checked swap
  policy](../../runtime/platform/html.ts) (page-level noSwap plus
  exact `hx-status` exceptions from the action case table),
  [compiler-owned guard](../../runtime/platform/htmx-guard.ts)
  (missing-target, response-control and task-shape policy with
  `action::missing_target` / `action::protocol` occurrences).
- Proving tests: `TestInvoiceFormLive` + `TestInvoiceHTMLFragments`
  (`tests/integration/invoice_test.go`, admitted swaps and served
  fragment bytes/policy), `TestGate3RendererFault`
  (post-commit fault stays an uncertain 500),
  `TestInvoiceBrowserGuardDOM` (UP23 verdict legs: five admitted
  swaps, missing-target before/during, OOB/partial/control-header
  rejection, remount stability).
- Cases: S01-P1 admitted swaps, S01-P2 missing target, S01-N1 unsafe
  shapes, S01-N2 post-commit fault.

## S02 — observable browser handler faults

A deliberately failing admitted handler yields one sanitized,
source-located diagnostic and does not silently claim success; later
events still run and disposal still works.

- Shipped source: [callback settlement with
  reporting](../../runtime/platform/browser.ts) via
  [the browser reporter](../../runtime/browser/diagnostics.ts)
  (category/phase/file/line/column; native cause, raw input, stack and
  secret bytes omitted; frozen record to `console.error`).
- Proving tests: `runtime/test/browser-dom.test.ts` ("failed handler
  completions report once without breaking later events", plus the
  timer variant), gate5 disposal legs.
- Cases: S02-P1 one sanitized report, S02-N1 disposal boundaries.

## S03 — grid focus, notice and pending-save state

Add/move/reorder focuses an attached intended control after the
complete tree is attached, blocked-save notice survives rerender and
is announced, pending save stays visible until settlement, and late
replies/disposal cannot clobber newer edits.

- Shipped source: [grid web
  sources](../../examples/invoice-grid/src/web/) (immutable
  notice/pending/focus state; attach-then-focus render order).
- Proving tests: gate5 grid-matrix focus/notice/pending legs
  (`tests/integration/browser/grid.mjs` via `TestGate5ServedMatrix`)
  and the late/dispose legs plus the `conformance.mjs` "dispose" leg.
- Cases: S03-P1 attached focus, S03-P2 durable notice, S03-P3 pending
  state, S03-N1 late/disposed.

## P03 — public generic composition

Public identity/helper/known-composite chains check across packages
and emit only valid concrete calls; illegal callees fail at their
declaration; no false parametric proof is reusable.

- Shipped source: [symbolic public-callee
  proof](../../compiler/internal/check/specialize.go)
  (`symbolicCall`: public validated callees only, private templates
  excluded, committed-proof visibility, expanding-edge rejection),
  [SCC-atomic validation](../../compiler/internal/check/specialize.go),
  [concrete emission](../../compiler/internal/emit/) (reached
  instances become direct calls; declaration-only symbolic types
  leave the emitted graph).
- Proving tests: `TestCurrentBundledGenericChain`
  (`tests/integration/generics_test.go`: cross-package chain,
  `nested<T> → identity<box<T>>`, stationary/permutation SCC,
  `nested<int>` direct concrete call, failed-component and
  failed-component-reuse legs),
  `TestExportedGenericRejectsPlus`,
  `TestExportedGenericRejectsRepresentationOperations`,
  `TestExportedGenericSymbolicProofIsolation`,
  `TestPrivateGenericTemplatePreserved`
  (`compiler/internal/check/exported_generics_test.go`).
- Cases: P03-P1 identity emits, P03-P2 helper chains, P03-P3 atomic
  SCC, P03-N1 illegal bodies, P03-N2 growth/false proof.

Representation-dependent operations on a bare type parameter still
need an explicit named callable or dictionary input; no traits or
error-set parameters were added (P04 stays deferred).

## Invoice acceptance workload

The existing invoice server/grid is the cross-target workload proving
the rows above together: shared source, direct routes, explicit
service plus actor, finite outcomes, protected effects, browser
application and closure, and the contract-edit table. Staged evidence
is `TestGate3*` + `TestGate4*` + `TestGate5*` + `TestInvoice*` in the
full-tree log; installed evidence is the deferred UP25 rerun on x86.


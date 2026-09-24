# UP01 file ownership and write-set reservation

Lanes per `post-upgrade-implementation-tasks-2026-09-24.md`: **C** compiler,
**R** server/application, **B** browser/delivery, **I** coordinator.
A task's write set includes its focused tests. No two tasks in a wave may
claim the same write-set label.

## Exclusive boundaries (standing, all waves)

| Lane | Owns | Must not touch |
| --- | --- | --- |
| C | `compiler/internal/{syntax,resolve,types,check,ir,emit}` for UP02/05/08/11/14/17; contract-mutation suite (UP21); docs (UP26) | `compiler/internal/browser`, build driver, browser tooling (B); server/platform runtime (R) until the UP18 handoff narrows it |
| R | `runtime/platform/{http,router,server,action-routes,action-json,form,html}.ts` through UP12; invoice shared dependency + server files; live server tests (UP22) | browser runtime/profile, `compiler/internal/browser`, build driver, grid (B); common compiler (C) |
| B | browser runtime/profile modules, `runtime/platform/browser.ts`, `compiler/internal/browser`, build-driver/browser tooling, grid sources; browser tests (UP23) | common compiler before the UP18 handoff; invoice shared contract (R, frozen before UP19/20) |
| I | catalogue generation + mirrors, `runtime/modules.json`, lock publication, shared fixture integration, CI changes, `integration/*` branches, this `implementation-up01` directory, UP24/25/27 gates | worker-owned sources except via integration merges |

## Per-task write sets (from the ledger)

UP01 integration; UP02 compiler; UP03 regex; UP04 browser-core;
UP05 compiler+catalogue; UP06 server-runtime; UP07 browser-core;
UP08 compiler+catalogue; UP09 server-runtime; UP10 browser-audit;
UP11 compiler+catalogue; UP12 server-html+other-apps;
UP13 browser-core+browser-api; UP14 compiler; UP15 build-driver+distribution;
UP16 invoice-contract+invoice-domain; UP17 compiler+generic-tests;
UP18 build-driver+distribution+server-runtime+server-assets+compiler;
UP19 invoice-server+invoice-render+invoice-domain; UP20 invoice-grid;
UP21 contract-tests; UP22 server-tests; UP23 browser-tests;
UP24/25/27 integration; UP26 documentation.

## Ordered handoffs (ownership moves only here)

1. UP02 → UP07/11/14: symbolic/concrete model boundary (C internal).
2. UP04 → UP11: generated-call context interface (B→C, interface only).
3. UP07 → UP10/11: profile inventory + runtime exports (B→B/C, interface only).
4. UP08 → UP09/12: callback + per-case HTML guard metadata (C→R/B).
5. UP11/13 → UP15: call/entry interfaces + real operations for bundling.
6. UP12 → UP18: server/asset integration (R→B); UP16 → UP19/20: frozen shared
   contract snapshot (locks updated only through I afterwards).
7. UP17 → UP18: common emitter ownership narrows to asset/entry inputs (C→B).
8. UP18 → UP19/22: server file ownership released (B→R).

## Coordinator-held shared outputs

`runtime/catalogue.ts`, generated Go/catalogue mirrors, the pinned HTMX
asset, `runtime/modules.json`, dependency locks, `compiler/testdata`
mirrors, and CI workflows. Workers propose changes through their owning
task's authored inputs; I regenerates and checks mirrors after each
handoff. Runtime formatting runs only in the worker's isolated checkout
with unrelated hunks removed before handoff.

## Known temporary breakage rule

An incompatible API cutover may leave dependent examples unbuildable on the
integration branch until their migration tasks land. I records those known
failures; the application gate is not marked green and no compatibility
shim is added. UP24 is the required complete-tree green barrier.

# P08 dependency map and concurrent lanes

## P08.1 component/contract map + prerequisites

| Work | Components | Interface | Hard prerequisites |
|---|---|---|---|
| A surface + lowering | checker, emitter, formatter, warnings, fixtures/goldens, bulk construction, `runtime/collections/*` | C-A, C-I | None (starts immediately) |
| B failure conventions | Can libs, docs, X-R06-1/X-R07-1 | C-B | None (pure authoring; no shared generated files) |
| C browser + 2nd app | catalogue browser section, browser.ts, grid, new app, actions checker (R03) | C-D, C-E | A for examples migration format (formatter); otherwise starts immediately |
| D host discrimination | isolated prototypes, adapter/companion assignment, W2 | C-D, C-G | C's library proof (R02→R01 edge); DOM primitives can start alongside C |
| E lifetime/observation/storage | server.ts, transport, sql pool/tx, s3.ts, catalogue lifecycle section, request-policy spec | C-C, C-F (operands) | None for the policy spec; operands after X-R04-1/X-R15-1 probes |
| F data + companion pair | sql descriptors/dialects/rows, PG app, MySQL legs, webhook example, companion service, C-G policy, R16-03 envelope bullet | C-F, C-G, C-C (vocabulary), C-A (batch mirror), C-E (endpoint status honesty) | C-C honesty vocabulary published by E; X-R10-1 probe outcome for claims |
| G editor sequence | lsp.go, diagnostics, format/hover/ref/complete/rename | C-I, C-D (queries) | Q3 packet (done); rename impl after A's `with` lands; rest start immediately |
| H deploy/AI/docs | dist scripts, UP25, AI budget/eval, R16 fixes, story | C-H, C-G | R16 fixes start now; story/AI/UP25 follow B–G outputs |

Shortest dependent chain: A(Q3 `with`) → G(rename); C(library proof) →
D(discrimination); E(C-C vocabulary) → F(protocol); all → H(story/UP25).
Nothing else serializes.

## P08.2 shared foundations

Task-writer follow-up: the [R14 supplement](blocker-resolution/README.md)
adds F-owned durable reserve/fence/settle/reconcile services to C-G; H's native
guard consumes them, and E integrates shared context/transport. F04 qualifies
the backend ledger, so H08 consumes F04 evidence. G03/G05 now include local
bindings by user 1A. These are slices of the same eight lanes and retain the
shared-file owners below; no new ownership cycle is introduced.

Foundations (build once, depend many): request-policy spec + C-C vocabulary
(E), failure conventions (B), `with` grammar + formatter (A), paired-build
identity (exists; H extends). Everything else is lane-local until
integration.

## P08.3 lanes with ownership and handoffs

| Lane | Owner | Inputs | Outputs | Handoffs |
|---|---|---|---|---|
| A surface/lowering | surface owner | C-A, C-I | checker/emitter/formatter/lint, migrated examples, W4 lowering | `with` impl → G; formatter → C; lowering → W4 legs |
| B conventions | conventions owner | C-B | X-R06-1/X-R07-1, convention docs, W3 legs | conventions → all Can-authoring lanes |
| C browser/apps | browser owner | C-D, C-E, A formatter | native additions, shared libs, 2nd app, captured reads, W1 legs, Firefox provisioning, R16-01 invoice fix | library proof → D; C-D/C-E → D, G; R03 patches → E (server boundary) + A (syntax/emission) |
| D host decision | host owner | C proof, C-D, C-G | discrimination record, tier assignment, W2 integration | assignment → H (packaging/release scope) |
| E lifetime | lifetime owner | C-C, probe results | policy spec, contracts, operands, hook, S3 resolution, W5 legs | C-C vocabulary → F; hook → H (ops docs) |
| F data/pair | data owner | C-F, C-G, C-C, X-R10-1 | PG app, MySQL parity, recipes, protocol+auth, companion, pair qual | protocol → H; claim outcome → E (budget interplay) |
| G editor | editor owner | C-I, C-D, A `with` | format→hover→refs→completion→rename, R09 acceptance | rename → B/C authors (dogfood) |
| H release | release owner | all lane outputs, F→H identity handoff | UP25 qual, AI budget/eval, R16-02/R16-04, supported story, W6 legs, provisioning register | story → user; blockers → preparation |

Shared-file conflict owners (one owner each; others hand off patches):
`compiler/internal/catalogue/catalogue.json` merge → E owner (append
per-package sections only: browser→C, lifecycle→E, map/set→A, sql→F);
`runtime/platform/browser.ts` → C;
`runtime/platform/{server,transport,sql,s3,action-json,action-routes,html,htmx-guard,action-client}.ts` → E
(C hands the R03 server-boundary patch to E; action-client deadline stays
with E — no split ownership);
`compiler/internal/check/{completion_matches,locals,callables,array}.go` → A;
`compiler/internal/check/{actions,action_bindings,browser}.go` → C
(E hands the R04-operand patch to C);
`compiler/internal/emit/action*.go` + `compiler/internal/syntax/*` R03 slice → A
(C hands R03 emission/grammar patches to A);
`compiler/internal/sql/*` + descriptors → F; `runtime/collections/*` → A;
`compiler/lsp.go` + `diagnostics.go` → G; `examples/webhook/**` + companion → F;
`examples/invoice*/**` → C; `distribution/**` → H.
Single-lane files: `runtime/ai/*`, `examples/native-ai/**`, top-level
`README.md`, `distribution/README.md` → H; `tests/<area>` follows the lane
of the code under test, integration matrix legs follow the owning workload
lane. Generated `runtime/catalogue.ts` and pinned vendor files: regenerated
only, never hand-edited. `runtime/` + `tools/runtime/` edits carry
lint-fix, format, runtime-check, and relevant-test duties.
(P09-m6: if C authors before A's formatter lands, C re-runs the formatter +
rechecks before handoff; A owns the formatter, C owns re-migration.)

## P08.4 integration checkpoints + failure handling

- **IC1 interfaces published:** C-A..C-I implemented behind their owners;
  cross-lane contract tests (edit-propagation, status honesty, unknown-write
  vocabulary) green. Blocks: dependent lane completion claims.
- **IC2 workloads W1–W6:** each workload's positive/negative/failure-path/
  integration legs green in the owning lane's environment. Blocks: H release
  assembly.
- **IC3 pair + matrix:** companion pair qual, widest browser/DB matrix,
  UP25 Linux, AI eval — green. Blocks: supported-story publication.
- **Failure handling:** a failed checkpoint returns only the affected
  contract to preparation with a named issue (contract, evidence, options);
  unrelated lanes continue. An invalidated shared interface (C-*) pauses its
  consumers until re-published. No lane silently relaxes a contract.

## P08.5 workers and shared environments

Logically parallel (A–H) ≠ simultaneously runnable. Shared resources:

- PostgreSQL: one server, separate databases per lane/run; no shared tables.
- S3-compatible storage: disposable buckets / unique key prefixes per run;
  never the same key across concurrent runs.
- Browsers: pinned Chromium/WebKit/Firefox runners; parallel-safe via
  separate profiles/ports.
- x86 Linux (UP25): single queue — H schedules exclusive windows.
- AI eval: env credentials + spend caps; H serializes live-model runs.
- Provisioning register (P09-M4, owned by H — explicit prerequisites, not
  implicit leg dependencies): disposable S3 bucket + creds (gates E's
  X-R15-* and W5 S3 legs); live PG/MySQL instances + creds (gates E's
  X-R04-1, F's X-R10-1, W6 data legs); Firefox runner provisioning (lane C
  executes, gates W1/W6 matrix legs — P09-M3: gate5 runs Chromium+WebKit
  only today); AI provider creds + spend approval (gates R14 eval);
  x86 access scheduling (gates UP25). Unavailable live evidence is not a pass.
- This MacBook Air: no x86 emulation ever; no long full-tilt runs without
  asking; prefer minimal-scope reruns (F-R13-04).
- Generated outputs/packaging state: only the owning lane mutates; IC gates
  serialize publication.

Actual concurrency = min(logical readiness, workers, environment slots).
Lane count follows the design; with fewer workers, order by the critical
path: A → C → D, E → F, B/G anytime, H last.

## P08.6 cycle/bottleneck check

Edges: A→C (formatter), A→G (`with`), C→D (proof), E→F (vocabulary),
B→* (conventions, non-blocking), *→H. No cycles. Hidden prerequisites
surfaced: X-R04-1/X-R15-1/X-R10-1 probes gate their consumers' completion
(but not their start — conditioned branches let work begin). Bottlenecks:
catalogue.json merge (mitigated: append-only sections + E owner), UP25 x86
window (mitigated: H schedules late, everything else proceeds), AI spend
(mitigated: budgets + stub-first). Critical-path candidates: C→D host chain
and E→F data chain; no durations promised without evidence.

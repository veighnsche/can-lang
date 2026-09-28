# Generated startup attribution checklist

Status: G25-G30 Muse-reported complete; R5 independently found correction
requirements. Initial owner and all scratch are retired; correction tasks ready.
Design: [startup attribution](generated-startup-plan.md). Workspace:
`/Users/vince/Projects/can-lang`. Executor: sole Muse Spark 1.3 Contributor/MAX,
scoped `--yolo`, no arbitrary total step cap. Campaign:
[master tasks](performance-improvement-tasks.md).

## Monitoring, prerequisites and ownership

Codex owns research/design/three saved consultations and independent review.
Muse owns all tools/tests/corrections and shared progress here. Codex-owned
monitoring record is `.performance/performance-push-20260928/continuation.json`;
The next iteration runtime record is
`.performance/performance-push-20260928/muse-run-9-owner.json`; the continuation
links it. The existing same-chat `exhaust-supported-performance-fixes` heartbeat
runs every 15 minutes. Session/process/prompt identities and confirmed startup belong in
the monitoring record before Codex yields. No second implementation executor.

Prerequisites: reviewed forwarding commit and all old groups/prompts/scratch
retired; source-supported stage design and equivalent Jev responses saved.
No global defaults, compiler/runtime implementation, vendor or publication edits.
No optional catalogue comparison. This packet authorizes its bounded busy-host
attribution runs only after correctness qualification; the correction below
authorizes one fresh replacement for the provisional initial run.

| Lane | Owned files | Inputs | Parallel safety and join |
| --- | --- | --- | --- |
| A | new `tools/performance/startup-attribution.py`, `startup-attribution.ts` | G25 frozen CLI/AST manifest/event schema | Coordinator implements after G25; no test-file edits until G27 author releases it |
| B | new `tools/performance/test_startup_attribution.py` | G25 interface and design, synthetic + actual-emission acceptance requirements | Native Muse subagent authors tests without running builds/tests or editing shared progress; joins at G28 |

Use native implementation agents for these disjoint ready lanes; no extra
worktrees/copies/private caches. Only the coordinator runs commands and updates
this checklist. Serialize all integration, build, test and sampling work. Save
compact commands/exits/timestamps/hashes/cleanup in
`generated-packet-5-muse-evidence.json`; keep raw attribution records separately
as `generated-startup-attribution.json`. Do not claim a gain or commit code.

## Shared interface

- [x] **G25 — freeze opt-in preparation/probe contract**
  - Prerequisites: acceptance and retired ownership above.
  - Owner: Muse coordinator, both new attribution tool interfaces.
  - Changes: defined below; default validation only; six independent trial
    processes, two excluded warmups, seven accepted batches, one fresh child each.
  - Acceptance: real state-body/import inventory inspected (evidence below);
    metadata root, prepared graph roots and instrumented identity separated; no
    static Can preloads; fresh fd-3 per child via existing driver helper only.
    Contract frozen before lanes A/B.
  - Evidence: G25 validation 2026-09-28 (HEAD `77731b3d`, forwarding `4589122c`
    inside; runtime.py `96f3022a`, bench `53995d55`, transpile `250c056e` match
    packet-4 for reused drivers). One-build perfemit emitted 281 artifacts over
    271 zero-byte placeholders (verified 0 bytes before execution), then runtime
    symlink + node_modules link. `program/state.ts` 187723 bytes
    `5f43d077`, entry `98f1e584`, adapter 2792 bytes `e86b0e83` with 14/14
    bindings incl. doubled/sum/frequency/generic_pass/captured_map. No
    `diagnostics/` in perfemit output, so diagnostics stages are honestly skipped
    (modules) / N/A (bundle) this packet. Installed TS 7.0.2 sync/AST API parses
    under node v24.21.0 (type-stripping; crashes under Bun, so probe runs on
    `node` from PATH): `$canInitialize` has 57 top-level statements, all
    Variable(const)/Expression, 0 await/yield, 0 syntactic diagnostics; state has
    59 imports. Owner: coordinator (tools + checklist + commands); native agent
    owns only `tools/performance/test_startup_attribution.py` content, no
    commands/shared writes. Scratch `/tmp/can-startup-muse8-AQMyJW` registered in
    `muse-run-8-owner.json`, retired at end.

### Frozen G25 contract (normative for G26/G27; no drift without coordinator edit)

Files: `tools/performance/startup-attribution.py` (driver),
`tools/performance/startup-attribution.ts` (AST probe, runs under `node`),
`tools/performance/test_startup_attribution.py` (tests load hyphenated modules
via importlib spec, as `drivers/test_runtime_driver.py` does). No other new
files. No production/compiler/runtime/harness-default/vendor/publication edits.

Probe CLI: `node <repo>/tools/performance/startup-attribution.ts <cmd>`.
Uses only installed `typescript` (`typescript/unstable/sync`,
`typescript/unstable/ast`) + `node:fs`. Commands:
`inventory --state S --manifest M` and
`instrument --state S --out O --manifest M --probe-specifier P`
(`P` frozen as `./startup-probe.ts`). Exit 0 writes manifest (+ `O` for
instrument); any failure exits 2, writes nothing, stderr one line
`startup-probe-error <code> <detail>`, code in {`parse_error`,
`has_diagnostics`, `initializer_missing`, `initializer_ambiguous`,
`unsupported_statement`, `async_expression`, `missing_semicolon`,
`already_instrumented`, `io_error`}.
Rules: exactly one `$canInitialize` FunctionDeclaration; body is a Block with
>=1 statement; every top-level statement is VariableStatement (const-only,
every declarator has an initializer, plain Identifier names) or
ExpressionStatement; no Await/Yield/NewTarget-async anywhere in the body (New
expressions allowed); source text of each statement ends with `;`; spans
strictly increasing/non-overlapping/in order; source must not already contain
`__canStartupMark`.
Transform: prepend one import line
`import {__canStartupMark as __canStartupMark} from "./startup-probe.ts";`
then for each statement i (span [start,end), start = first non-trivia char,
end after `;`, applied last-to-first) emit
`__canStartupMark(i,0);\n` + original bytes verbatim +
`\n__canStartupMark(i,1);`. No lexical blocks, no reordering, original bytes
preserved. Multiple calls per statement = one inclusive observation.
Manifest schema `can.startup-probe-manifest/1`: {schema, state_path,
state_sha256, state_bytes, initializer:{name,start,end,statement_count},
probe_specifier|null, instrumented_sha256|null,
statements:[{index,start,end,bytes,sha256,kind,binding,factories}]}.
Binding: VariableStatement -> comma names; `a = ...` -> left identifier text
(or `complex:<kind>`); else `call:<callee>`. Callee: Identifier text |
`base.name` for PropertyAccess | `complex`. factories: ordered unique callees
(New targets as `new:X`).
Probe helper (driver-written into diag graph as `program/startup-probe.ts`,
transpiled with the graph): exports `__canStartupMark(index, phase)` pushing
`[index, phase, performance.now()]` onto
`globalThis.__canStartupEvents` (created once). Event schema: array of
`[int index, 0|1 phase, float ms]`. Coverage rule: exactly 2N events, ordered
`[i,0,t0],[i,1,t1]` per statement, `t1 >= t0`; cost = `t1-t0` inclusive only.

Driver CLI:
`startup-attribution.py --repo R --output O [--validate-only | --measure]
[--scratch-parent D] [--trials 6] [--warmups 2] [--batches 7]
[--child-timeout 60] [--sampling-bound 300] [--prepared-out DIR]
[--trial-run --trial-index I --prepared P --rows F]`.
Default (no `--measure`) is validation only. Loads existing
`drivers/runtime.py` via importlib; uses its `command` (fd-3 boundary) and
`adapter` (binding discovery) as module globals `driver_command`,
`driver_adapter` (patchable by tests); never duplicates fd logic. Invokes
existing `drivers/runtime-transpile.ts` via bun for the prepared-graph
transform (records its `transpilation.json`).
Modes: validate-only = one-build preparation + preflight, no timing;
`--measure` = validate, then sequentially self-spawn `--trials` independent
`--trial-run` OS processes, collect rows, compute report, always clean scratch
(try/finally + SIGINT/SIGTERM handlers). `--trial-run` is internal (one trial).
`--prepared-out DIR` (validate-only only; refused with `--measure`): DIR must
not exist, becomes the scratch, kept on success for coordinator gates
(coordinator retires it), retired by tool on failure/interruption.
Exits: 0 complete; 1 failed (record written, `reason` set); 2 usage.
Profiles (frozen): `ordinary-modules`, `ordinary-bundle`, `minimal` control,
`diagnostic-modules` (separate label, always last). Per trial, ordinary order
= rotate([ordinary-modules, ordinary-bundle, minimal], trial_index % 3)
(`trial_order(i)`); 2 warmup + 7 accepted batches per profile, one fresh Bun
child per batch via `driver_command` (fresh fd-3 each).
Prepared layout in owned scratch: `perfemit`, `runtime-inventory/`,
`generated/` (runtime/ placeholders verified then replaced by symlink to
`<repo>/runtime`), `node_modules` symlink, `generated-js/` (ordinary graph),
`generated-js-diag/` (symlinks to ordinary files except real instrumented
`program/state.js` + `program/startup-probe.js`), `startup-ordinary-bundle.js`
(`bun build generated-js/startup-facade.js --target=bun --format=esm`),
`startup-launcher.js`, `manifest.json`, `transpilation.json`.
Frozen writers: `write_facade` (`startup-facade.ts` in generated/ root before
transpile: re-exports `initialize` from `./program/state.ts`; `verify()`
checks doubled/frequency/sum/generic_pass/captured_map with frozen-result and
input-unchanged assertions using in-graph runtime), `write_minimal`
(`startup-minimal.ts`: empty initialize/verify, zero imports),
`write_probe_module`, `write_launcher` (plain JS, ZERO static import
statements; argv `--root URL --mode ordinary|diagnostic --diagnostics skip|run
--diag-module URL --metadata-root DIR --entry-url URL --statements N`; dynamic
import root (stage import), optional diagnostics import + configure (only when
`run`), sync `initialize()` with no await in timer, untimed `verify()`,
single-line JSON `{status:"ok", stages, events?}`).
Diagnostics rule: `run` iff `<generated>/diagnostics/source-index.json` exists
AND profile is module-graph; bundle -> `not_applicable_bundled`; minimal or
absent metadata -> `skipped_no_metadata`. Stage statuses: `measured` |
`skipped_no_metadata` | `not_applicable_bundled`.
Child acceptance `accept_launch(returncode, stdout, mode, statement_count)`:
raises `ValueError("launch rejected: <code> ...")`, code in {`exit`, `output`,
`json`, `status`, `unexpected_events`, plus event codes}; diagnostic coverage
via `validate_events(events, n)` -> per-statement ms list or
`ValueError("startup events invalid: <code> ...")`, code in {`count`, `order`,
`index`, `phase`, `time`}. Any rejection fails the trial, failing the run
(failure is evidence).
Pure/testable API: `parse_args`, `median`, `mad`, `trial_order`,
`validate_events`, `accept_launch`, `check_manifest_coverage` (True or
`ValueError("manifest coverage invalid: <code> ...")`, code in {`count`,
`order`, `span`, `hash`, `kind`}), `check_caps` (raises
`ValueError("scratch cap exceeded ...")` over 64 MiB total or 500 compiled
`.js` real files), `mirror_runtime_inventory`, `verify_placeholders`,
`replace_runtime_link`, `install_node_modules_link`, `make_scratch`,
`cleanup_scratch` (refuses paths outside allocation record),
`owned_scratch(parent, keep)` context manager (cleans on exit/exception incl.
KeyboardInterrupt), `PROFILES`, `STAGES`, `STATUSES`, `STATEMENT_MARK`,
`EVENTS_KEY` constants.
Output schema `can.startup-attribution/1` (both modes share header; measure
adds trials/rows/activity/report): header {schema, mode, status, reason?,
repo, commit, timestamps, preparation{perfemit_ms, emit_ms, transpile_ms,
bundle_ms, instrument_ms}, manifest, checks[], scratch{bytes, compiled_files,
retired}, prepared_out?}; measure adds {trials, warmups, batches,
profile_orders, rows[{trial, batch, profile, warmup, parent_wall_ms,
stages:{import, diagnostics_import, configure, initialize: {status, ms|null}},
statement_ms?|events?}], activity{before, after (ps without args/env)},
report{per-trial medians + across-trial median/range/MAD per profile x stage,
units ms}}. No p95, no speedup claims, no summed instrumented rows, no median
subtraction. Parent wall separate from child stages. Warmups retained,
excluded from medians. Activity via `ps -eo pid,ppid,pcpu,pmem,comm` only.

## Lane A — actual preparation and stage attribution

- [x] **G26 — implement bounded opt-in driver and AST probe**
  - Prerequisites: G25. Owner: coordinator, attribution tools only.
  - Changes: implemented frozen contract: one-build perfemit, zero-byte
    inventory/link, prepared module graph + ESM bundle + minimal control, oracle
    facade, staged launcher, TS-7 AST probe, separately labelled diagnostic
    graph. Additive coordinator notes: `check_manifest_coverage` pure validator;
    two-pass existing-script transpile (script skips the runtime symlink);
    atomic probe writes; non-TS extension guard.
  - Acceptance: met at G28 (no production edits/copies/installs; branding
    coherent via in-graph runtime; 64 MiB/500 caps enforced; 60 s child timeout;
    300 s sampling bound; cleanup on success/failure/interruption).
  - Evidence: `tools/performance/startup-attribution.py` (877 lines),
    `startup-attribution.ts` (242 lines); G28 gates below.

## Lane B — independent test authoring

- [x] **G27 — meaningful transformation and lifecycle tests**
  - Prerequisites: frozen G25; independent of G26. Owner: native Muse agent,
    new test file only; no commands/shared writes.
  - Changes: 81-test file as specified (transformation roundtrips + all 9 probe
    codes, event/launch/manifest validators, links, defaults, caps, lifecycle,
    writer contents, env-gated actual-state inventory). First spawn failed
    silently; retry authored and released the file; no other files touched.
  - Acceptance: met at G28 (81/81 pass, 0 skips with env set; failures drove 3
    real probe/driver fixes; no production test touched).
  - Evidence: `tools/performance/test_startup_attribution.py` (1216 lines);
    joined at G28 with zero test-content fixes needed.

## Integration and bounded evidence

- [x] **G28 — join and qualify**
  - Prerequisites: G26/G27 released. Owner: coordinator, tools/test integration.
  - Changes: fixed 3 real defects via owned probe/driver (adjacent-span order,
    server-crash parse_error, non-TS guard) plus diagnostic transpile second
    pass; ran 81 new tests, validation-only preparation with kept graph, strict
    TS, 24 oracles; verified identities; retired kept graph.
  - Acceptance: met 2026-09-28T14:54:48Z: 81/81 pass 0 skips; validate-only
    complete 13/13 checks + 4/4 preflights; strict tsc exit 0; 24/24 oracles,
    stderr 0; fresh emission matches G25 hashes; 41.8 MiB/298 files under caps;
    kept graph retired; no timing before gates; runtime untouched.
  - Evidence: `generated-packet-5-muse-evidence.json` (G28 section);
    `g28-validate.json` compact record in scratch until final retirement.

- [x] **G29 — one qualified busy-host attribution run**
  - Prerequisites: G28. Owner: coordinator, existing new opt-in tool.
  - Changes: one explicit `--measure` run as specified (6 independent trial
    processes sequential/counterbalanced, diagnostic last, 2 warmups + 7 batches
    x 1 fresh child, ps activity without args/env, raw rows + median/range/MAD).
  - Acceptance: met 2026-09-28T14:55:48Z: 216/216 launches accepted, 0
    rejections; non-isolated/busy-host labelled; no p95/speedup/summed rows;
    scratch retired (42.1 MiB/298 files at measure, retired after).
  - Evidence: `generated-startup-attribution.json` (679 KiB raw) and
    `docs/performance/generated-startup-result.md` (report).

- [x] **G30 — source-supported next handoff**
  - Prerequisites: G29 terminal result. Owner: coordinator, read-only result note.
  - Changes: dominant stages identified (import/eval ~18.7 ms; init ~3.1 ms led
    by $canDomain/$canText); source usage/side effects inspected; candidates
    inventoried; nothing implemented.
  - Acceptance: met 2026-09-28: scope reconciled with production
    entry/assertions; counts-vs-timings and bundle caveats recorded; per-site
    invoke, numeric JSON and all twelve dispositions preserved; no exhaustion
    declared; no pruning/lazy/invoke code written.
  - Evidence: `docs/performance/generated-startup-next-investigation.md`;
    terminal handoff below. R5/Q5 left to Codex.

## R5 corrections — same checklist, updated Muse skill

The [R5 correction design](generated-startup-plan.md#r5-correction-design-independent-review-2026-09-28)
amends the original frozen G25 contract where necessary. Preserve the initial
G25-G30 progress as Muse-reported history; those ticks are not Codex acceptance.
Original raw and evidence files stay unchanged. Independent review/validation/
negative reproductions are the three `generated-packet-5-independent-*.json`
records named in the design. Production compiler/runtime regions and existing
shared driver/storage/isolation implementations stay unchanged.

The sole coordinator uses the updated Muse skill in an owned interactive tmux
TUI, Contributor/MAX, scoped yolo, full saved handoff as one positional argument.
Runtime identities, normal/read-only viewing commands, viewer state and cleanup
ownership live only in the linked Codex monitoring record. No arbitrary total
step limit, second executor, private cache, worktree, source/dependency copy or
retained bundle. Use the same checklist in one sustained session. After finishing,
record a terminal handoff and explicitly release all coordinator/agent file
writers; an idle live TUI is allowed. Do not tick R5/Q5 or commit code.

| Correction lane | Ownership | Parallel prerequisite / join |
| --- | --- | --- |
| A | Coordinator: startup-attribution.py/.ts, result/next-investigation notes and shared progress | G28a interface freeze; parallel with native lane B; G28d joins |
| B | Native Muse agent: test_startup_attribution.py only | G28a frozen contracts; no build/test commands or shared progress writes; release file and report to coordinator |

- [ ] **G28a — freeze corrected ownership, identity and rejection contracts**
  - Prerequisites: original owner/child/groups/prompt/scratch independently
    retired, independent findings reviewed.
  - Owner: Muse coordinator; this checklist interfaces and tool design decisions.
  - Changes: read linked correction design and all three independent records;
    select bounded reuse of existing storage/process patterns without global
    edits; specify durable scratch/child registration and recovery, strict
    measurement membership/status/numeric checks, source UTF-16/AST coverage,
    safe output writes and complete identities/freeze checks. Record concrete
    helper interfaces/error evidence before native test authoring begins.
  - Acceptance: original stage/oracle/native/fd-3 semantics preserved; failure
    paths specify bounded group retirement and truthful cleanup failure; no
    fabrication of absent historical identities. Disjoint ownership frozen.
  - Evidence: fill actual decisions/interfaces and native lane assignment here.

- [ ] **G28b — repair allocation, process and measurement implementation**
  - Prerequisites: G28a; independent of G28c authoring.
  - Owner: coordinator, startup-attribution.py/.ts only; no test-file edits until
    native owner releases it.
  - Changes: implement all correction design requirements: durable PID/start/
    inode owner markers, safe recovery and kept-graph transfer, refusal of foreign
    or replaced resources, complete process-group lifecycle including preparation,
    finite/stage/profile/membership/event validation, UTF-16 whole-source and full
    AST coverage, preexisting/alias output refusal and identity-safe rollback,
    actual source/driver/prepared graph/resolved dependency inventory and freeze.
  - Acceptance: each reproduced defect fails closed; genuine cleanup errors stay
    visible with retirement false; no global/default/runtime production edits.
  - Evidence: actual diff, commands, timestamps and compact identities in
    generated-packet-5-corrective-muse-evidence.json.

- [ ] **G28c — author meaningful negative and lifecycle regressions**
  - Prerequisites: G28a; native agent owns only test_startup_attribution.py.
  - Changes: cover the independently reproduced bad stages/events, incomplete/
    duplicate/unknown batches and warmups, profile status mismatch, source identity
    and omitted statement, astral Unicode spans, preexisting output preservation,
    input/output alias, unregistered prefix allocation and root/marker replacement,
    active-owner refusal, safe abandoned recovery and cleanup-failure reporting.
    Include real bounded controller-plus-descendant timeout and SIGTERM retirement,
    with immediate test-owned cleanup. Detect source/prepared/dependency identity
    drift and missing inventory; no test that only mirrors implementation.
  - Acceptance: tests exercise actual rejection/lifecycle behavior, preserve prior
    meaningful oracles and zero-skip actual-state coverage. Agent runs no commands,
    releases file before coordinator integration and reports completion evidence.
  - Evidence: exact new coverage and native release handoff here.

- [ ] **G28d — join and independently meaningful qualification**
  - Prerequisites: G28b/G28c terminal and all native writers released.
  - Owner: coordinator; serialized tool/test integration and commands.
  - Changes: run applicable new/updated tests including real lifecycle checks,
    actual emitted state inventory with zero skips, validation-only preparation,
    strict TS and existing 24 actual generated/native cases. Reuse one build/graph
    within this invocation, retain compact full source/module/driver/dependency
    identities and verify cleanup. Repair real failures through the same tasks.
  - Acceptance: all bounded gates pass; no leaked groups, kept graphs or cleanup
    failures; original raw/evidence hashes unchanged; no competing task-owned jobs.
  - Evidence: corrected compact qualification commands/exits/oracles/hashes/real
    timestamps/owner retirement; do not retain execution tree or broad audit logs.

- [ ] **G29a — one newly qualified complete busy-host attribution run**
  - Prerequisites: G28d complete and no test/build/native task remains live.
  - Owner: coordinator, corrected opt-in tool and result report.
  - Changes: make one fresh bounded run to
    .performance/performance-push-20260928/generated-startup-attribution-corrected.json
    with six independent sequential driver trials, two excluded warmups and seven
    accepted batches/profile, counterbalanced ordinary profiles, diagnostic last,
    one fresh startup/batch, 300-second sampling bound and 64 MiB/500-file cap.
    Record observed activity without idle gate or secrets; full frozen identities
    and exact completeness checks required. Retire all scratch/groups immediately.
  - Acceptance: incomplete/failed/drifted trials fail the run, no merged partial
    data; original record unchanged and explicitly provisional. Recompute reports
    from corrected data with median/range/MAD, separate preparation/sampling times,
    inclusive instrumentation caveats, no p95/causal speedup claim. Explicitly
    correct original trial-number prose discrepancy and missing-identity limits.
  - Evidence: corrected raw record, arithmetic/report and cleanup evidence.

- [ ] **G30a — reconcile follow-up and release writers**
  - Prerequisites: G29a terminal; coordinator owns result/next-investigation notes.
  - Changes: update notes from corrected evidence, preserve original limitations;
    qualify Intl.Segmenter no-throw assumption and nested callee inventory versus
    executed work. Retain pending per-site invoke/numeric/all-twelve queue without
    authoring any production remedy. Supply terminal handoff with source/evidence
    hashes, actual timestamps, remaining uncertainty and explicit all-writer release.
  - Acceptance: no unsupported semantic or performance claim, no exhaustion claim,
    no implementation outside this correction; R5/Q5 left unticked for Codex.
  - Evidence: fill handoff here and compact corrective evidence; TUI may remain
    idle for review/viewing and later same-session continuation.

## Independent acceptance and continuation (Codex leaves Muse unticked)

- [ ] **R5 — independent review and checkpoint commit**
  - Prerequisites: G25-G30 plus G28a-G30a terminal, all writers released and
    execution child groups retired; TUI viewer lifecycle separately recorded.
  - Owner: Codex; inspect actual diff, transformation, controls/sampling/identities
    and raw report calculations. Run bounded meaningful correctness independently;
    do not repeat optional sampling merely for acceptance. Verify cleanup and
    promptly commit coherent accepted tools/docs with exact owned paths. Genuine
    defects return as concrete assignments to Muse in this checklist.
- [ ] **Q5 — next supported fix and final queue discipline**
  - Prerequisite: R5. Owner: Codex; reconcile all twelve slices, design the next
    source/evidence-supported remedy, make required three fresh consultations and
    save ordered Muse tasks. No time cap or manufactured work. Retire the campaign
    heartbeat only when the independent final review finds no ready supported fix.

## Muse terminal handoff (G25-G30 complete, 2026-09-28)

Coordinator + one native test author (retry after a silent first-spawn failure);
no other executors, worktrees, installs, caches or competing jobs. Owned
deliverables: `tools/performance/startup-attribution.py` (`4b36c148`),
`startup-attribution.ts` (`2cb99861`), `test_startup_attribution.py`
(`ba652ef3`, 81/81 pass, 0 skips), `docs/performance/generated-startup-result.md`
(`e311c370`), `generated-startup-next-investigation.md` (`3bec12cb`), this
checklist, `.performance/performance-push-20260928/generated-packet-5-muse-evidence.json`
and `generated-startup-attribution.json` (679 KiB, 216/216 accepted). No
production/compiler/runtime/harness/vendor/publication edits; no commit; no
speedup claim; no exhaustion declared. Busy-host medians: import ~18.7 ms
modules / ~12.9 ms bundle, init ~3.1 ms ($canDomain 0.93, $canText 0.87);
diagnostics unresolved by absent metadata. Owned scratch fully retired (see
evidence); no cleanup failures. R5/Q5 and all next design belong to Codex.

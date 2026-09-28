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

- [x] **G28a — freeze corrected ownership, identity and rejection contracts**
  - Prerequisites: original owner/child/groups/prompt/scratch independently
    retired, independent findings reviewed.
  - Owner: Muse coordinator; this checklist interfaces and tool design decisions.
  - Changes: read linked correction design and all three independent records;
    selected bounded driver-local analogues of storage.py patterns (marker,
    dev/ino identity, pid/start liveness, group record, descriptor-safe delete)
    with zero edits to storage.py/isolation.py/drivers/runtime.py or any
    production file; specified durable scratch/child registration and recovery,
    strict measurement membership/status/numeric checks, source UTF-16/AST
    coverage, safe output writes and complete identities/freeze checks below.
    Recorded before native test authoring began.
  - Acceptance: original stage/oracle/native/fd-3 semantics preserved; failure
    paths specify bounded group retirement and truthful cleanup failure; no
    fabrication of absent historical identities. Disjoint ownership frozen.
  - Evidence: frozen contract immediately below; native lane B assigned
    test_startup_attribution.py only, no commands/shared writes, release to
    coordinator before G28d. Toolchain HEAD `c62d259`, Bun 1.4.2, Go 1.27.1,
    Node 24.21.0, TS 7.0.2; runtime bare imports are node:/bun: builtins only
    (playwright/bun:test are test-only under runtime/test/).

### Frozen G28a correction contract (normative for G28b/G28c; amends G25 where noted)

Ownership (driver-local; prefix never proves ownership):
`OWNER_MARKER=".startup-owner.json"`, `OWNER_KIND="can.startup-scratch"`,
`OWNER_SCHEMA=1`. Marker JSON: kind, schema_version, root, pid,
process_start (ps lstart or null), created_at, root_identity/marker_identity
[dev,ino], process_group pgid|null, launched_children bool, transferred bool,
keeper_pid|null, state active|retained|cleaned. Helpers:
`_fs_identity(st)->[dev,ino]`, `_process_start(pid)->str|None`,
`_pid_alive(pid,group=False)->bool` (ValueError on invalid pid),
`_owner_is_live(meta)->bool` (pid alive AND start matches when recorded),
`_read_owner(root)->dict` or `ValueError("scratch ownership invalid: <code>
...")` codes {missing, foreign, replaced, active_owner, active_group},
`make_scratch(parent)->Path` (mkdtemp + immediate O_EXCL/O_NOFOLLOW marker,
fsync, _ALLOCATED), `cleanup_scratch(path)->bool` (False for None/missing-
unregistered; else durable verify, bounded group retirement, descriptor-safe
delete; ANY failure retains marker+_ALLOCATED and raises
`ValueError("scratch cleanup failed: ...")` with the actual error),
`recover_scratch(path)->bool` (explicit abandoned-owned retirement; refuses
active/foreign/replaced), `owned_scratch(parent,path,keep)` (keep+success =>
retained/transferred, keeper_pid=self, coordinator retires later via
cleanup_scratch and records it; every other exit cleans incl.
KeyboardInterrupt; cleanup failure propagates, callers record retired False).
Call-site scratch record: {bytes, compiled_files, retired, cleanup_error?};
retired True only after verified deletion (kept graphs: transfer + explicit
later retirement, never silent success).

Process groups (fd-3 helper preserved; bun launches stay on driver_command):
`_kill_process_group(pgid,term_grace=2.0,kill_grace=1.0)->{pgid,termed,killed,
retired}` (TERM, bounded poll, KILL, bounded poll, killpg verify; ValueError
only for invalid pgid type; termed True when TERM alone emptied the group).
`run_supervised(args,cwd,timeout,term_grace=2.0,kill_grace=1.0)
->CompletedProcess` (Popen start_new_session, text capture; timeout =>
bounded group retirement first, then raise TimeoutExpired; nonzero =>
CalledProcessError). run_measure spawns every --trial-run controller via
run_supervised, records its pgid in the owner marker, clears after each
trial; handled SIGINT/SIGTERM during sampling retires the active controller
group bounded before cleanup. Prep keeps driver_command (go/bun prep
children spawn no detached descendants; documented).

Measurement (stage/oracle/profile semantics unchanged):
`accept_launch(returncode,stdout,mode,statement_count,profile=None)` keeps
4-arg behavior; new: every measured ms must be finite non-bool number >= 0,
every unmeasured ms None, else existing `launch rejected: output ...`
(detail says "finite milliseconds"); profile given adds: ordinary-bundle =>
diagnostics both not_applicable_bundled; minimal => both skipped_no_metadata;
ordinary-modules/diagnostic-modules => both measured or both skipped
(consistent pair, never bundled/mixed); import+initialize always measured;
violations => `launch rejected: output ...` (detail names "profile <name>");
unknown profile => same code. `validate_events(events,n)`: each time finite
non-bool >= 0 and globally nondecreasing across the 2N sequence, all under
existing `startup events invalid: time ...`. `check_row_membership(rows,
trials,warmups,batches)->True` or `ValueError("row membership invalid:
<code> ...")` codes {count, unknown, warmup, duplicate, omission}: exact
trials x 4 profiles x (warmups+batches) rows, unique complete
(trial,profile,batch), warmup == (batch < warmups), no bools-as-numbers.
`check_trial_rows(rows,trial_index,warmups,batches)` same codes for one
controller payload. `build_report(rows,trials,probe_manifest,warmups=2,
batches=7)`: membership defects => `_Fail("report rejected: ...")`;
parent_wall_ms and measured ms finite >= 0; diagnostic rows carry
statement_ms length == statement_count, finite >= 0;
accepted_batches_per_profile_trial must equal batches exactly. Report shape
unchanged (median/range/MAD, no p95).

Source/AST (spans stay TS UTF-16 offsets, instrumentation-correct):
`_utf16_to_python_index(text,utf16_offset)->int` (surrogate-aware; bad
offset => `manifest coverage invalid: span ...`).
`check_manifest_coverage(manifest,state_text)` adds whole-source
state_sha256/state_bytes check => new code `source`, and initializer
name/start/end check (start==statements[0].start, end==last.end) => new code
`initializer`; per-statement slicing via UTF-16 mapping; existing
count/order/span/hash/kind meanings unchanged.
`check_manifest_agreement(authoritative,candidate)->True` or
`ValueError("manifest agreement invalid: <code> ...")` codes {source,
initializer, count, span, hash} (ordered factories equality). prepare() runs
`inventory` then `instrument` and requires agreement before transpile; no
partial execution.

Safe writes: probe refuses pairwise state/out/manifest canonical alias =>
`startup-probe-error path_alias ...`, and preexisting out/manifest (lstat
ENOENT required) => `startup-probe-error output_exists ...`; codes join the
frozen 9 (total 11); rollback unlinks only invocation-created files.
Python `_write_output(path,record)`: refuses preexisting target
(`ValueError("output refuses preexisting path: ...")`), atomic
pid-unique tmp + os.replace, tmp cleaned on failure. Rows/prepared/manifest
inside fresh owned scratch use the same helper.

Identities/freeze: `collect_tool_versions()->{bun,go,node,typescript}`;
`collect_prepared_identities(scratch,repo)->dict` {tool_versions, drivers
sha256 (runtime.py, runtime-transpile.ts, startup-attribution.py/.ts),
sources (facade/minimal/probe/launcher frozen-content sha256, state
sha256/bytes), prepared {ordinary/diagnostic: sorted real .js
[{path,bytes,sha256}] + symlinks [{path,target}], bundle {sha256,bytes}},
dependencies {node_modules realpath target, package_json sha256|None,
bun_lock sha256|None, bare_specifiers sorted}}; no node_modules walk, no
copies. `check_identity_drift(before,after)->True` or
`ValueError("prepared identity drift: ...")`. run_measure freezes after
preflight, stores record["identities"], re-collects after sampling and
fails the run (no report) on drift; validate-only records without drift
check. Original raw/evidence bytes untouched; absent historical identities
are never reconstructed.

- [x] **G28b — repair allocation, process and measurement implementation**
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
  - Evidence: implemented 2026-09-28T15:34:41Z (probe behaviors verified above;
    full gates at G28d): durable `.startup-owner.json` marker (pid/start,
    root/marker dev/ino, group record) with missing/foreign/replaced/
    active_owner/active_group refusals; cleanup_scratch truthful failure
    (retains marker+allocation, `scratch cleanup failed`); recover_scratch
    abandoned-owned path with bounded group retirement; owned_scratch keep
    transfer (retained/transferred/keeper_pid); _kill_process_group
    TERM/KILL/reap + own-group suicide guard; run_supervised new-session
    controllers with on_start marker tracking and retirement-before-raise;
    accept_launch finite-nonneg ms + profile status rules (optional profile);
    validate_events finite-nonneg + global nondecreasing; check_row_membership/
    check_trial_rows count/unknown/warmup/duplicate/omission; strict
    build_report (exact batches, finite walls/costs); UTF-16 span mapping +
    source/initializer coverage codes; check_manifest_agreement with
    inventory-before-instrument in prepare(); probe path_alias/output_exists
    refusals + identity-checked rollback; atomic _write_output with
    preexisting refusal; collect_prepared_identities/check_identity_drift
    frozen before trials and verified after sampling. Zero edits outside the
    two owned tool files. Diff/commands/hashes finalized in corrective
    evidence at the G28d join.

- [x] **G28c — author meaningful negative and lifecycle regressions**
  - Prerequisites: G28a; native agent owns only test_startup_attribution.py.
  - Changes: covered the independently reproduced bad stages/events, incomplete/
    duplicate/unknown batches and warmups, profile status mismatch, source identity
    and omitted statement, astral Unicode spans, preexisting output preservation,
    input/output alias, unregistered prefix allocation and root/marker replacement,
    active-owner refusal, safe abandoned recovery and cleanup-failure reporting.
    Included real bounded controller-plus-descendant timeout and SIGTERM retirement,
    with immediate test-owned cleanup. Detects source/prepared/dependency identity
    drift and missing inventory; no test that only mirrors implementation.
  - Acceptance: tests exercise actual rejection/lifecycle behavior, preserve prior
    meaningful oracles and zero-skip actual-state coverage. Agent ran no commands,
    released the file before coordinator integration and reported completion evidence.
  - Evidence: native lane-B terminal 2026-09-28 (two silent subagent_spawn
    failures with zero file changes, then Workflow single-agent success; 17 tool
    calls, no commands/shared writes, only test_startup_attribution.py touched).
    62 new tests in 12 classes (AcceptLaunchFinite 8, ValidateEventsFinite 4,
    RowMembership 7, TrialRows 7, BuildReport 3, ManifestSource 7, AstralSpan 2,
    ProbeSafeWrite 8, WriteOutput 2, Ownership 7, Lifecycle 5, Identity 2) for
    143 total; file `ed00ea56` (2275 lines, compiles). One existing expectation
    updated as frozen: ScratchTests foreign cleanup now expects
    `scratch ownership invalid`. ActualStateTests env-gated coverage preserved.
    Agent explicitly released the file; coordinator owns all integration from here.

- [x] **G28d — join and independently meaningful qualification**
  - Prerequisites: G28b/G28c terminal and all native writers released.
  - Owner: coordinator; serialized tool/test integration and commands.
  - Changes: ran applicable new/updated tests including real lifecycle checks,
    actual emitted state inventory with zero skips, validation-only preparation,
    strict TS and existing 24 actual generated/native cases. Reused one build/graph
    within this invocation, retained compact full source/module/driver/dependency
    identities and verified cleanup. Repaired one real failure at its root cause
    (zombie-visible process groups; reap added to the killer, tests untouched).
  - Acceptance: all bounded gates pass; no leaked groups, kept graphs or cleanup
    failures; original raw/evidence hashes unchanged; no competing task-owned jobs.
  - Evidence: 2026-09-28T15:56:06Z validate-only complete 13/13 checks + 4/4
    preflights (57 statements, state `5f43d077`/187723 B matching G25, bundle
    `27ebf9cf`/907741 B matching independent validation, 41.8 MiB/298 files,
    identities recorded, graph transferred); 143/143 tests 0 skips at
    15:58:40Z (first run 141/143 exposed unreaped-zombie retirement blindness,
    fixed in `_kill_process_group` via waitpid reap, verified harmless to
    Popen.wait/communicate); strict tsc exit 0; 24/24 oracles stderr 0 B via
    fd-3 helper; kept graph retired through the tool transfer path, owned
    parent removed and verified absent, no stray processes or /tmp residue;
    original raw `b10fee1b…` unchanged. Full compact record in
    generated-packet-5-corrective-muse-evidence.json; execution tree retired.

- [x] **G29a — one newly qualified complete busy-host attribution run**
  - Prerequisites: G28d complete and no test/build/native task remains live.
  - Owner: coordinator, corrected opt-in tool and result report.
  - Changes: made one fresh bounded run to
    .performance/performance-push-20260928/generated-startup-attribution-corrected.json
    with six independent sequential driver trials, two excluded warmups and seven
    accepted batches/profile, counterbalanced ordinary profiles, diagnostic last,
    one fresh startup/batch, 300-second sampling bound and 64 MiB/500-file cap.
    Recorded observed activity without idle gate or secrets; full frozen identities
    and exact completeness checks required. Retired all scratch/groups immediately.
  - Acceptance: incomplete/failed/drifted trials fail the run, no merged partial
    data; original record unchanged and explicitly provisional. Recomputed reports
    from corrected data with median/range/MAD, separate preparation/sampling times,
    inclusive instrumentation caveats, no p95/causal speedup claim. Explicitly
    corrected original trial-number prose discrepancy and missing-identity limits.
  - Evidence: 2026-09-28T16:00:01Z-16:00:11Z exit 0, status complete, 216/216
    accepted (48 warmup + 168 accepted, 0 rejections), exact 7 accepted
    batches/profile/trial, frozen identities == post-sampling (no drift),
    counterbalanced orders with diagnostic last, 553/553 processes observed
    (WindowServer+Chrome, no secrets), prep 3025/362/178/460/23 ms, scratch
    retired (42.5 MiB/298 files, verified absent, no strays). Corrected raw
    `ef7c4861…` (799854 B); original `b10fee1b…` byte-identical, never merged.
    Medians: import 17.466 modules / 12.854 bundle, init 2.998 / 2.901
    ($canDomain 0.932, $canText 0.883). Report recomputed in the result note
    with the original trial-4/trial-3 prose correction and provisional limits.

- [x] **G30a — reconcile follow-up and release writers**
  - Prerequisites: G29a terminal; coordinator owns result/next-investigation notes.
  - Changes: updated notes from corrected evidence, preserved original limitations;
    qualified Intl.Segmenter no-throw assumption and nested callee inventory versus
    executed work. Retained pending per-site invoke/numeric/all-twelve queue without
    authoring any production remedy. Supplied terminal handoff with source/evidence
    hashes, actual timestamps, remaining uncertainty and explicit all-writer release.
  - Acceptance: no unsupported semantic or performance claim, no exhaustion claim,
    no implementation outside this correction; R5/Q5 left unticked for Codex.
  - Evidence: result note rewritten from corrected data only (`d80c1c6e`, 7144 B)
    with original-limits section and trial-number prose correction; follow-up
    note updated (`d303102a`, 5700 B) qualifying the Segmenter no-throw claim
    (native capability/first-use timing needs investigation, precedence must
    hold) and the nested-callee inventory-vs-execution caveat; candidates,
    per-site invoke/numeric queue and all-twelve dispositions retained; zero
    production edits; no commit. Terminal handoff below. ALL coordinator/native
    file writers released; TUI remains idle at composer for review/viewing.

## Independent acceptance and continuation (Codex leaves Muse unticked)

- [x] **R5 — independent review and checkpoint commit**
  - Prerequisites: G25-G30 plus G28a-G30a and G28e-G30b terminal, all writers released and
    execution child groups retired; TUI viewer lifecycle separately recorded.
  - Owner: Codex; inspect actual diff, transformation, controls/sampling/identities
    and raw report calculations. Run bounded meaningful correctness independently;
    do not repeat optional sampling merely for acceptance. Verify cleanup and
    promptly commit coherent accepted tools/docs with exact owned paths. Genuine
    defects return as concrete assignments to Muse in this checklist.
  - Acceptance: 2026-09-28, after G30d explicit ALL-writer release, one bounded
    review passed 224/224 actual-state tests with zero skips, one validate-only
    graph/13 gates, actual replacement-preservation probe and nine durable
    metadata-group observations. All review groups/keeper/scratch retired;
    465 current runtime/emitter/fixture input hashes remain unchanged. Earlier
    strict TS/24 oracles and 216-row/69-summary arithmetic reused, not repeated.
    Evidence: `generated-packet-5-independent-bounded-acceptance.json`; exact
    rebuilt module inventories were not retained by the independent runner,
    so complete content continuity remains Muse evidence, with independent
    actual validation and unchanged source inputs. Accepted within builtin-only
    attribution scope, no performance improvement or speedup claim. Auxiliary
    lane frozen per latest user direction; no successive hardening packet.
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

## Muse terminal handoff (G28a-G30a correction complete, 2026-09-28T16:01:47Z)

Coordinator + one native test author (two silent subagent_spawn failures with
zero file changes, then Workflow single-agent success); no other executors,
worktrees, installs, private caches or competing jobs. All G28a-G30a ticked;
R5/Q5 left unticked for Codex. Changed paths (hashes sha256, 8-char prefix):
`tools/performance/startup-attribution.py` (`1cc077aa`, 1636 lines),
`startup-attribution.ts` (`299ae139`, 270 lines),
`test_startup_attribution.py` (`ed00ea56`, 2275 lines, 143/143 pass 0 skips),
`docs/performance/generated-startup-result.md` (`d80c1c6e`, corrected only),
`generated-startup-next-investigation.md` (`d303102a`, qualified),
this checklist,
`.performance/performance-push-20260928/generated-packet-5-corrective-muse-evidence.json`
and `generated-startup-attribution-corrected.json` (`ef7c4861…`, 799854 B,
216/216 accepted, 16:00:01Z-16:00:11Z). Original raw (`b10fee1b…`) and
original evidence (`661dadd6…`) byte-identical, never merged. No
production/compiler/runtime/harness-default/vendor/publication edits; no
commit; no speedup claim; no exhaustion declared. Busy-host corrected
medians: import ~17.5 ms modules / ~12.9 ms bundle, init ~3.0 ms
($canDomain 0.93, $canText 0.88); diagnostics unresolved by absent metadata.
Remaining uncertainty: busy-host noise (trial-5 bundle outlier retained);
per-module import split still unmeasured; Segmenter lazy timing unproven.
All owned scratch/groups retired and verified absent; no cleanup failures.
ALL coordinator/native file writers released. TUI idle at composer for
review/viewing; no further tasks started.


## Second independent correction: G28e–G30b (ready after released handoff)

Supporting design: [complete supervision/input identity correction](generated-startup-plan.md#second-independent-review-complete-phase-supervision-and-input-identity).
Independent evidence and three fresh consultations are linked there. G28a–G30a
completion records are preserved as Muse reports; R5 is still not accepted.
Only the coordinator writes progress here. Runtime/session/goal/custody values
remain in the [authoritative Codex record](../../.performance/performance-push-20260928/muse-run-9-owner.json).

- [x] **G28e — freeze the complete lifecycle and identity interfaces**
  - Prerequisites: previous explicit all-writer release, R5 rejection evidence
    and linked design read. Owner: sole coordinator, checklist/interface notes.
  - Changes: inspected native goal support (three-way negative, see evidence);
    no `get_goal`/`create_goal` exists to reuse or create, so no delegation
    goal can be established and no prose substitute is offered. Defined concrete
    contracts for preparation/trial workers, retirement result and
    failed-retirement custody, registration rollback, explicit kept-graph
    keeper, required input/resolved-dependency records, phase timestamps and
    test seams. Followed the linked design, not a new broad planning exercise.
  - Acceptance: known four lifecycle defects have direct contracts, fd-3 helper
    and Bun-local parent timer retained, complete identities specified, production
    and shared helpers untouched; native goal blocker given with evidence.
  - Evidence: frozen contract immediately below with requirement-to-regression
    mapping. Goal blocker: exposed tool schemas contain no
    get_goal/create_goal/report_progress/update_goal; `muse --help` lists no
    goal subcommand; `muse schema` wire schema has zero goal mentions
    (verified 2026-09-28T16:31:17Z). Delegation-gated G28g cannot start until
    Codex resolves goal availability; coordinator-owned G28e/G28f proceed.

### Frozen G28e contracts (normative for G28f/G28g; amends G28a where noted)

All-exit supervised spawn. `run_supervised(args,cwd,timeout,term_grace=2.0,
kill_grace=1.0,drain_timeout=5.0,on_start=None)` retires the worker group on
EVERY exit path (zero, nonzero, timeout, external signal), then drains pipes
bounded. New `RetirementFailed(Exception)` with attrs args/returncode/
timed_out/interrupted/retirement/detail and message prefix `phase retirement
failed`: zero exit + verified retirement => CompletedProcess (as before);
zero exit + unverified => RetirementFailed; nonzero + verified =>
CalledProcessError (as before, actual code preserved); nonzero + unverified
=> RetirementFailed with returncode kept; timeout => bounded retirement then
TimeoutExpired when verified, else RetirementFailed(timed_out=True);
external signal during wait => retire bounded, re-raise the original signal
when verified, else RetirementFailed(interrupted=True) chained from it.
Post-kill drain uses communicate(timeout=drain_timeout); on expiry pipes are
closed, pipes_drained False, retirement unverified. Retirement dict gains
`pipes_drained` and `refused` (None normally). Marker/on_start registration
failure still retires the new group first, then raises. Existing 143-test
call shapes keep passing (no-descendant fixtures retire trivially).

Preparation worker. New internal `--prepare-run --scratch S --child-timeout T`
mode binds the driver, runs prepare() into S, writes S/preparation-result.json
via _write_output ({status, manifest?, prepared?, checks, timings, bytes,
files, error?}), exits 0/1. run_measure/run_validate spawn it via
run_supervised (one registered group enclosing Go build, emission, inventory,
probe, transpile, bundle, preflights and helper children); the worker uses
driver_command internally so fd-3 is unchanged. Parent validates the result
(checks all pass, manifest schema, re-runs check_manifest_coverage/agreement
from scratch files) and maps worker/timeout/retirement failures to _Fail with
full detail. Parent SIGINT/SIGTERM cancels preparation and trials alike.
Version/resolution inspection commands run as transient supervised probes via
the same helper with marker record/clear.

No premature clearing. run_measure clears active pgid and marker ONLY after
verified retirement; on RetirementFailed/unverified the marker keeps the
group, scratch is retained, and the record carries recoverable custody
{owner pid/start, pgid, leader pid/start, root identity, reason}; later
cleanup refuses with active_group until a verified retirement or a guarded
recovery. `_kill_process_group(...,leader=None)`: leader={pid,start} triggers
pre-signal verification (leader alive, start matches, ps pgid matches);
mismatch or dead-unreaped leader with no continuity proof => refused (no
signal). Fresh-call paths (run_supervised, measure-finally) pass no leader
(temporal locality, reuse infeasible in bounded graces); recover_scratch
passes the recorded leader, so stale markers can never signal a recycled
foreign pgid. Marker gains group_leader {pid,process_start}|None;
_set_process_group(root,pgid,leader=None).

Registration rollback. make_scratch/explicit allocation captures the new
inode identity immediately; on marker-write failure `_rollback_new_directory
(path,expected_identity)` removes it only if identity matches and the
directory is empty, else leaves it and reports. Failure raises
`ValueError("scratch registration failed: <marker error>[; rolled back
<path> | ; rollback failed: <error> (cleanup pending: <path>)]")`. Foreign,
replaced or non-empty victims are never touched.

Keeper custody. Marker gains keeper None|{pid,process_start,asserted_at}
(keeper_pid kept for compat, keeper object authoritative; schema 1 or 2
accepted on read, new markers written 2, missing keeper normalizes to None).
take_custody(root): live caller asserts keeper after ownership verify;
refuses live-foreign keeper/owner; idempotent for keeper-self.
release_custody(root): keeper-self clears to None. Live-foreign keeper =>
new `active_keeper` code under `scratch ownership invalid` in _read_owner,
blocking cleanup_scratch and recover_scratch; dead keeper never blocks.
owned_scratch keep-transfer sets keeper to the live producer; the exited
producer never protects a graph another live consumer holds.

Complete identities. collect_prepared_identities stays total (None allowed)
and gains: inputs {fixture tree hashes, emitter {module-owned go-list
dependency package files, go.mod, binary sha/bytes}, build {gomaxprocs,
build args}},
runtime_ts {actual linked repo/runtime **/*.ts hashes, scope-labelled
conservative}, emitted {adapter + generated/packages **/*.ts hashes},
executables {bun/go/node realpath/version/dev/ino/size} (swap-detecting
identity, not 100 MB content hashes; documented), builtins {bun
version+revision}, resolved {computed True, roots, reachable_files count,
reachable_bare [{specifier, importers, resolution builtin {runtime} | file
{path, sha256, bytes}}] resolved with bun createRequire from the importer
directory, unreachable_bare list with best-effort resolutions (informational,
no completeness requirement)}. Reachability walks relative and
static-string dynamic imports transitively from the ordinary/minimal/diag
facade roots plus the bundle artifact. check_identity_complete(inv) rejects
missing/empty required data (`ValueError("identity incomplete: <what>")`;
enforced at freeze and revalidate call sites in run_measure/run_validate, so
empty equal dicts never pass at run level). check_identity_drift stays a pure
comparator to preserve the existing meaningful drift unit tests without test
edits (test lane is delegation-blocked). Required reachable resolution
failure or non-file non-builtin
resolution fails the run. Freeze before use, revalidate after; drift fails.

Bounds and timer purity. record.bounds={overall,preparation,sampling} each
{start,end} ISO; record.timestamps stays overall. Sampling spans first trial
spawn to last rows validation; preparation spans the worker; prose uses each
label exactly. parent_wall_ms stays in the trial worker around the Bun spawn;
supervisor/identity work never enters it.

G28a preserved: all foreign/active/replaced/alias/output refusals, truthful
cleanup outcomes, fd-3 helper, stage/oracle/profile/membership/event/AST
contracts, report shape, no production/shared-helper edits.

Requirement-to-regression mapping. Success-with-leftover =>
RetirementFailed + custody retention (G28g fixture via run_supervised; G28h
real trial path). Nonzero/timeout/signal => exit-specific contracts above
(G28g fixtures incl. self-SIGTERM thread; G28h real interruption retired).
Registration failure => rollback + message (G28g forced marker failure +
_rollback unit cases; G28h allocation hygiene). Failed retirement =>
marker kept, active_group refusal, guarded stale-marker recovery refusal
(G28g forged/foreign groups; G28h no live groups at cleanup). Keeper =>
take/release/active_keeper refusal/released recovery (G28g live-sleep
keeper; G28h transfer + coordinator retirement). Identity completeness =>
complete-record acceptance, missing/wrong-importer/drift rejection (G28g
synthetic trees + real bun resolution; G28h full inventory on one reused
build). Bounds/timer => sampling-vs-preparation split asserted on real
G28h/G29b records (integration evidence, not unit mirrors).

- [x] **G28f — implement supervised phases, safe registration and identities**
  - Prerequisite: G28e; may run alongside G28g only after interface freeze.
  - Owner: coordinator, `tools/performance/startup-attribution.py` and `.ts` only.
    Test file stays with native author until release; no shared helper edits.
  - Changes: enclose preparation and all spawned tooling in one owned worker
    group; retain existing per-trial workers and unchanged fd-3 inside. Bound
    retirement and pipe drainage on success/error/timeout/signal; prove all groups
    absent before clearing markers/deleting graphs. Preserve failed-retirement
    marker and report recoverable owner/error. Register/roll back allocation on
    marker failure for generated/explicit paths; transfer retained graphs to a
    real live keeper and reject active or mismatched ownership. Record and validate
    exact relevant source/runtime/build inputs, all required drivers/graphs and
    runtime-aware resolved dependencies/executables; freeze before use, check drift
    after, reject missing data. Separate actual preparation and sampling bounds.
  - Acceptance: all design contracts enforced without broad source/dependency
    walks/copies; no unbounded waiting or false cleanup success; no production edits.
  - Evidence: driver `16aefcf3` (2449 lines, was 1636) 2026-09-28T16:51:38Z;
    probe .ts untouched (final review raised no probe defect). Implemented:
    RetirementFailed + all-exit retirement/bounded drain in run_supervised;
    --prepare-run worker with parent result/coverage revalidation; _phase_tracker
    verified-only clearing with retire_pending; retained-custody records;
    chained _failure_reason; leader-verified _kill (recover passes recorded
    leader, fresh paths pass none); registration rollback; take/release_custody
    + active_keeper (keeper_pid + schema-1 read compat preserved);
    supervised version/resolution probes; emitter go-list inputs; runtime_ts,
    emitted, executables, builtins revision; reachability walk + single bun
    resolve probe; check_identity_complete at freeze/revalidate (drift stays
    pure for the 143). Smoke proof 16:48Z: worker validate-only complete 13/13,
    119 reachable files, 13 reachable bare all builtin, playwright
    file-resolved unreachable, 14 emitter pkgs; first smoke correctly failed
    closed on unprefixed bundle externals (bun build strips node: prefixes),
    fixed via runtime builtinModules. 143/143 zero-skip on the new code;
    transfer retired via tool; tree verified absent; no strays. Both earlier
    raw/evidence records preserved (hashes re-verified at handoff). Native
    author notes: success-with-leftover observable via descendant death (pid
    file fixture); RetirementFailed attrs frozen; forced marker failure raises
    `scratch registration failed`; check_identity_complete rejects synthetic
    gaps (build complete synthetic trees per G28e mapping).

- [x] **G28g — independently meaningful lifecycle/identity regressions**
  - Prerequisite: G28e; native agent owns only `test_startup_attribution.py`,
    no commands/shared progress writes. Independent of G28f implementation.
  - Changes: add actual bounded controller/descendant tests for success leaving
    a child, nonzero failure, timeout, external SIGTERM during preparation and
    during trials; exercise the complete run paths, not just the kill helper.
    Cover registration failure, failed-retirement marker retention/deletion refusal,
    marker-write allocation rollback on generated/explicit paths, keeper-active
    recovery refusal and safe released recovery. Cover missing required identity,
    wrong-importer resolution, dependency/input drift and complete record acceptance;
    distinguish sampling bounds from preparation. Preserve previous meaningful
    143 tests, real zero-skip AST/oracle coverage and safe test-owned cleanup.
    Controlled short-lived subprocess fixtures may replace expensive preparation
    work, while using the real phase-supervision/cancellation code path.
  - Acceptance: known defects fail before repair, valid behavior passes, tests
    leave no owned process/tree and do not merely mirror implementation details.
  - Evidence: case-to-defect mapping, author release and bounded command plan;
    coordinator alone integrates and runs tests after native writer release.
    COORDINATOR NOTE 2026-09-28T16:52:42Z: delegation BLOCKED — native goal
    tools (get_goal/create_goal) unavailable (three-way verification in G28e
    evidence); skill forbids prose substitution. No native writer spawned;
    zero writers outstanding. G28h/G29b/G30b downstream-blocked pending Codex
    goal resolution. G28e contracts + G28f implementation + smoke proof above
    are complete and reviewable as-is.
    RESOLVED 2026-09-28T17:46Z in fresh retained-state continuation (normal
    start, no --no-session-log): native goal created, single workflow test
    author owned test file only (no commands/shared writes) and released 48
    regressions in 9 classes (191 total, 143 preserved byte-meaningful).
    Test file `28b9c926` 3859 lines. Case-to-defect mapping in
    generated-packet-5-final-corrective-muse-evidence.json (g28g).

- [x] **G28h — join and qualify the whole corrected execution contract**
  - Prerequisites: G28f/G28g terminal, native writer released.
  - Owner: coordinator, sequential integration/checks and shared progress.
  - Changes: run all applicable attribution tests with actual generated state,
    zero skips, new complete-path lifecycle cases and validated resolved input
    inventory. Reuse one bounded validation build/prepared graph within this
    invocation for actual AST/probe/control checks, strict TS and existing 24
    actual emitted/native oracles; retire transferred graphs with real custody.
    Repair genuine test failures in owned files without weakening contracts.
  - Acceptance: correct actual sampling/launch semantics, valid complete identities,
    original hashes unchanged, all groups retired before scratch cleanup, no leaks.
    No competing owned commands or successful broad audit repeated without cause.
  - Evidence: met 2026-09-28T17:58:00Z: validate-only complete 13/13 + 4
    preflights (57 statements, state `5f43d077`/187723 B matching G25, bundle
    `27ebf9cf`/907741 B matching G28d, 119 reachable files, 13 bare builtin,
    14 emitter pkgs, 41.8 MiB/298 files, transfer recorded); first suite run
    183/191 exposed 3 genuine driver defects, all repaired at root cause in
    owned driver with zero test edits (communicate-first wait blocked on
    descendant-held pipes -> wait + retire + bounded drain; RetirementFailed
    `.args` tuple coercion -> property + `__str__`; blank identity strings
    accepted -> blank rejection); rerun 191/191 0 skips at 17:57:27Z; strict
    tsc exit 0; 24/24 oracles stderr 0 B via fd-3 helper; kept graph retired
    via tool transfer path (True), owned parent removed and verified absent,
    no strays; all four prior raw/evidence hashes re-verified unchanged.
    Full compact record in
    `generated-packet-5-final-corrective-muse-evidence.json` (g28h).

- [x] **G29b — one complete replacement run with real input identities**
  - Prerequisites: G28h passed, all qualification/native work retired.
  - Owner: coordinator, sequential optional busy-host tool execution and result note.
  - Changes: run once to
    `.performance/performance-push-20260928/generated-startup-attribution-final.json`
    because previous source/resolved-dependency identities are irrecoverably absent.
    Freeze complete inputs/modules/drivers/resolutions before use and after, record
    actual separate preparation/sampling bounds, observe activity without secrets.
    Preserve 6 trials, 2 excluded warmups, 7 accepted batches/profile, 4 profiles,
    fresh startup each, ordinary counterbalance and diagnostic last, 300-second
    sampling/64 MiB/500-file caps; reject incomplete/drifted/failed trials.
  - Acceptance: preserve both earlier raw/evidence files unchanged/provisional,
    no merges/retries merely for status/reassurance, immediate verified group/tree
    retirement. Report median/range/MAD from final accepted membership with honest
    busy-host/inclusive/root-scope limits, no p95 or causal speedup claim.
  - Evidence: met 2026-09-28T17:59:09Z-17:59:20Z exit 0, status complete,
    216/216 accepted (48 warmup + 168 accepted, 0 rejections), exact 7+2
    batches/profile/trial, frozen == after (no drift), counterbalanced orders
    with diagnostic last, bounds overall 09-20 / preparation 09-13 / sampling
    13-19, 595/592 processes observed (WindowServer+Chrome, no secrets), prep
    2689/502/200/333/24 ms, scratch retired (42.5 MiB/298 files, verified
    absent, no strays). Final raw `fb060ec5` (1092771 B); priors `b10fee1b`
    and `ef7c4861` byte-identical, never merged. Medians: import 16.942
    modules / 12.369 bundle, init 2.974 / 2.840 ($canDomain 0.907, $canText
    0.855). Report recomputed in the result note with provisional limits.

- [x] **G30b — reconcile and release for independent review**
  - Prerequisite: G29b terminal (or concrete blocker with safe owned retirement).
  - Owner: coordinator, result/next-investigation/checklist and compact evidence.
  - Changes: preserve prior discrepancies and provisional records, update only
    supported conclusions and pending startup/authored-invoke/numeric/twelve-slice
    queue. Keep R5/Q5 untouched. Use native progress aligned with evidence; reserve
    100% and native `update_goal` completion for this final reviewable handoff.
    Supply exact changed paths/hashes, commands/times, unresolved limits and explicit
    ALL coordinator/native writers released; remain idle in this same TUI.
  - Acceptance: no production remedy or unsupported performance/exhaustion claim;
    cleanup failures/retained custody stated truthfully. Goal unavailable is a
    blocker, not a prose-only replacement. Codex owns independent acceptance/commit.
  - Evidence: met 2026-09-28T18:01Z: result note rewritten from final data
    only (`d6dc1386`, 7880 B) with both-priors-provisional limits section and
    trial-5-hot table; follow-up note updated (`622c1cf8`, 5692 B) with final
    medians, Segmenter qualification and inventory-vs-execution caveat
    retained; candidates, per-site invoke/numeric queue and all-twelve
    dispositions retained; zero production edits; no commit. Terminal handoff
    below. ALL coordinator/native file writers released; TUI remains idle
    for review/viewing. Native goal
    `goal-9747f76d-5eef-4f6d-8276-beeb4561490f` completed at handoff.


## Codex native-goal capability resolution (after G28f writer release)

The same TUI's supported `/goal` status command actually replied **goals are
unavailable for this session**. CLI help and MSP schema alone were insufficient:
the installed executable includes all four exact native goal tool names. This
session was launched with `--no-session-log`; the [official configuration
documentation](https://dev.meta.ai/docs/muse-code/configuration) confirms that
mode disables retained-session features. The [interactive documentation](https://dev.meta.ai/docs/muse-code/interactive)
documents `/goal` and native progress tracking. Relevant user settings have no
runtime capability override. Disabled retained state is the leading cause;
actual normal-start capability/invocation must still be verified, not inferred
from a binary string or documentation. Compact evidence is
`generated-native-goal-investigation.json` in the campaign evidence directory.

G28f's explicit ALL-writer release and no native agent/child group permit a
controlled fresh continuation. Retire only the old idle coordinator process,
preserving the same owned tmux socket/session/pane and attached read-only
viewer. Start exactly one Contributor/MAX/YOLO coordinator with the entire
owned saved handoff as its positional argument, normal native retained state,
no arbitrary step/token budget, no extra worktree or installed software.
Resume this SAME checklist at G28g–G30b; inspect native get_goal and create a
matching corrective goal before delegation when supported. If unavailable,
report a genuine blocker with writer release; do not waive or emulate it.
Necessary active native event/goal state is task-owned, bounded and temporary:
register actual UUID/path/inode, monitor storage, preserve compact goal/check
facts, and retire exact owned session logs after writers/coordinator/viewers
release. Do not export or retain bulky transcripts or delete shared session
indexes/caches. This resolution does not accept G28f production/tool code,
change G28g's regressions or authorize extra timing beyond G29b.

## Muse terminal handoff (G28e-G30b correction complete, 2026-09-28T18:01Z)

Coordinator (sole Contributor/MAX/YOLO, fresh retained-state continuation)
+ one native workflow test author for G28g (test file only, no commands,
released 17:46Z after a first silent spawn failure); no other executors,
worktrees, installs, caches or competing jobs. Owned deliverables:
`tools/performance/startup-attribution.py` (`77f29d04`, 2466 lines),
`startup-attribution.ts` (`299ae139`, untouched),
`test_startup_attribution.py` (`28b9c926`, 3859 lines, 191/191 pass 0
skips), `docs/performance/generated-startup-result.md` (`d6dc1386`),
`generated-startup-next-investigation.md` (`622c1cf8`), this checklist,
`.performance/performance-push-20260928/generated-packet-5-final-corrective-muse-evidence.json`
and `generated-startup-attribution-final.json` (`fb060ec5`, 1092771 B,
216/216 accepted). No production/compiler/runtime/harness/vendor/publication
edits; no commit; no speedup claim; no exhaustion declared. Busy-host
medians: import ~16.9 ms modules / ~12.4 ms bundle, init ~3.0 ms
($canDomain 0.91, $canText 0.86); diagnostics unresolved by absent
metadata. G28h repaired 3 genuine driver defects at root cause with zero
test edits (wait-first supervision, RetirementFailed `.args`, blank
identity rejection). Owned scratch fully retired (kept graph via tool
transfer path, parents removed and verified absent, no strays); both
prior raw/evidence records byte-identical, never merged. ALL
coordinator/native file writers released. R5/Q5 and all next design belong
to Codex. Native goal `goal-9747f76d-5eef-4f6d-8276-beeb4561490f`
complete.


## Third independent review corrections (G28i-G30c)

Supporting design: [third review](generated-startup-plan.md#third-independent-review-narrowly-repair-remaining-failure-branches).
All affected coordinator/native writers explicitly released before this revision.
Authoritative current runtime/native goal/cleanup custody: [Codex monitor](../../.performance/performance-push-20260928/muse-run-10-owner.json).
These are narrow unmet-contract repairs; all earlier evidence remains historical.
The sole existing Muse coordinator owns these assignments sequentially; no extra
agent/executor is required for this small corrective packet. Inspect the native
goal and create the matching new corrective goal if the prior one is complete.
Use native report_progress and complete only at explicit all-writer handoff.
Routine long-command inspection waits 1-5 minutes, default2, initial native bash
yield120000ms; preserve the command deadline and await delivered completion.
No repeated unchanged file/status polling or busywork to fill goal wakeups.

- [x] **G28i — complete allocation registration rollback**
  - Prerequisite: read exact third-review ENOSPC probe and frozen ownership contract.
  - Owner/files: sole coordinator, startup-attribution.py and test_startup_attribution.py.
  - Change: roll back exact new allocation for actual marker write/fsync OSError
    and handled interruption, not only wrapped ValueError; handle short writes.
    Protect replaced/foreign marker/root identities and report retained cleanup.
  - Checks: actual os.write ENOSPC and fsync failure injections; interrupted
    registration; short-write handling; both auto and explicit allocation paths;
    rollback refusal on replacement/nonempty foreign content remains safe.
  - Evidence: met 2026-09-28T18:26:56Z: 6 new RegistrationFailureTests, all
    failed pre-fix exactly as probed (3 OSError escapes, 1 interrupt leftover
    dir, 1 truncated marker, 1 KeyboardInterrupt instead of pending report);
    post-fix 30/30 with RegistrationRollback/Scratch/Ownership suites.
    Driver `bd19769a`, tests `af26a1c7`. Compact record in
    `generated-packet-5-third-corrective-muse-evidence.json` (g28i).

- [x] **G28j — consume command output without pipe-capacity deadlock**
  - Prerequisite: G28i complete; read real 1 MiB output probe and third-review design.
  - Owner/files: sole coordinator, same two Python files.
  - Change: concurrently drain both pipes, detect leader exit separately from EOF,
    retire descendant group on every exit branch, bound final drain/reaping, and
    enforce explicit combined4MiB capture overflow failure. No extra files/agents.
  - Checks: real 1 MiB stdout and stderr successful exact capture; nonzero output;
    leader exits with descendant-held pipe; timeout/signal/on_start-failure and
    overflow retirement. A successful large-output command must not time out
    merely because the pipe fills. No weakened or implementation-mirroring tests.
  - Evidence: met 2026-09-28T18:32:25Z: 7 new SupervisedCaptureTests, all
    failed pre-fix (6 TimeoutExpired pipe stalls, 1 truncated 64 KiB
    partial); post-fix 21/21 with all supervision suites, no strays.
    Driver `85296306`, tests `9df0f49e`. Compact record in
    `generated-packet-5-third-corrective-muse-evidence.json` (g28j).

- [x] **G28k — reject unsupported complete dependency identities**
  - Prerequisite: G28j complete; read real two-importer resolution proof.
  - Owner/files: sole coordinator, same two Python files; update scope note only.
  - Change: fail closed on reachable file dependencies lacking correct per-importer
    ESM/transitive closure; positive completeness remains this actual builtin-only
    packet. Do not claim first-importer entry-file hash covers another importer.
    Separate actual reachable importer lists from syntactic unreachable inventory.
  - Checks: real two-context same-specifier/different-content fixture cannot pass
    full completeness; incomplete file contexts rejected; actual13 builtin roots
    continue to qualify with runtime revision and correct reachable inventory.
  - Evidence: met 2026-09-28T18:38:11Z: 5 new ReachableClosureTests + updated
    g28g contract tests pass 21/21 with identity/resolution/bounds suites;
    two-importer fixture records unsupported with 2 distinct context paths
    and rejects naming probe-dep; builtin-only tree passes full
    completeness; orphan stays unreachable-only. Driver `3ebeff0c`, tests
    `f92ba86f`. Compact record in
    `generated-packet-5-third-corrective-muse-evidence.json` (g28k).

- [x] **G28l — bounded corrective qualification and identity continuity**
  - Prerequisites: G28i/G28j/G28k complete, no other native writers.
  - Owner/files: sole coordinator, applicable tests and compact evidence.
  - Checks: new contracts plus applicable lifecycle/identity regressions; use one
    bounded validate-only build with actual AST state for zero-skip qualification
    and immediate keeper/group/scratch retirement. Record exact current module,
    source and driver identities; compare ordinary/diagnostic/bundle content
    identities to unchanged final raw record. Repeat strictTS/24 generated/native
    gates only if their relevant production/probe/emission identities changed,
    explaining the new reason; do not repeat sampling.
  - Evidence: met 2026-09-28T18:39:18Z: one validate-only build complete
    13/13 + 4 preflights; 209/209 tests 0 skips at 18:38:52Z; ordinary 282
    + diagnostic 10 + bundle `27ebf9cf` + state `5f43d077` + 13 builtin
    reachable specs all byte-identical to `fb060ec5`; strict TS / 24
    oracles not repeated (probe `299ae139`, `drivers/runtime.py`
    `96f3022a`, runtime/ unchanged; emission identical; no new reason);
    kept graph tool-retired, parent removed verified absent, no strays;
    no new sampling. Repaired driver `3ebeff0c` recorded separately in
    `generated-packet-5-third-corrective-muse-evidence.json` (g28l).

- [x] **G30c — corrective handoff and truthful historical producer notes**
  - Prerequisite: G28l complete or concrete safe blocker.
  - Owner/files: coordinator, this checklist, generated-startup-result.md and
    generated-startup-next-investigation.md if necessary, compact evidence only.
  - Change: describe final raw as historical77f29d04 producer evidence pending
    Codex acceptance; record repaired-tool qualification separately. No rewritten
    hashes/retroactive inputs, extra timing, performance gain or exhaustion claim.
    Preserve next startup/authored-invoke/numeric/all-twelve queue.
  - Evidence: met 2026-09-28T18:40Z: result note (`2cde090e`, 8453 B) names
    `77f29d04` as the historical producer, records Codex arithmetic
    verification alongside withheld tool acceptance, and points to
    separately qualified repaired driver `3ebeff0c`; all reported numbers
    unchanged; follow-up note needed no change (queue/candidates intact);
    no commit. Terminal handoff below. ALL coordinator/native file writers
    released; TUI remains idle for review/viewing. Native goal
    `goal-3564d2ce-1c59-4825-88eb-6040bf2683c5` completed at handoff.

## Muse terminal handoff (G28i-G30c correction complete, 2026-09-28T18:40Z)

Sole Contributor/MAX/YOLO coordinator, same TUI, no extra executor.
Repaired three Codex-reproduced contract defects in owned Python only:
allocation rollback for write/fsync/interruption + short writes (G28i),
selector-driven concurrent pipe drain with leader-exit observation and
4 MiB overflow failure (G28j), per-importer reachable resolution with
builtin-only fail-closed completeness (G28k). Deliverables:
`tools/performance/startup-attribution.py` (`3ebeff0c`),
`test_startup_attribution.py` (`f92ba86f`, 209/209 pass 0 skips),
`docs/performance/generated-startup-result.md` (`2cde090e`, numbers
unchanged, historical `77f29d04` producer named), this checklist,
`generated-packet-5-third-corrective-muse-evidence.json`. G28l qualified
on one validate-only build (13/13, byte-identical ordinary/diagnostic/
bundle/state/13-builtin continuity vs `fb060ec5`, retired verified
absent); strict TS / 24 oracles not repeated (probe, shared driver and
emission identities unchanged — no new reason); no new sampling. Final
raw `fb060ec5` and all earlier records byte-identical, never merged or
relabelled. No production/compiler/runtime/default/vendor/shared-driver
edits; no commit; no gain/exhaustion claim. ALL file writers released;
no groups/keeper/scratch retained. R5/Q5 and all next design belong to
Codex. Native goal `goal-3564d2ce-1c59-4825-88eb-6040bf2683c5` complete.

## Fourth independent review corrections (G28m-G30d)

Supporting design: [fourth review](generated-startup-plan.md#fourth-independent-review-finish-marker-and-metadata-custody).
Explicit ALL-writer release and native goal completion were independently verified
before this revision. Runtime identities, current goal, viewers and cleanup custody
remain solely in the [Codex monitor](../../.performance/performance-push-20260928/muse-run-10-owner.json).
The original ENOSPC and 1 MiB stream failures now pass independent checks; two
remaining custody failures are reproduced in `generated-packet-5-independent-fourth-review-probes.json`.
These complete existing requirements, with no new architecture or sampling.
The same sole Muse coordinator owns this small packet sequentially; no new agents.
Inspect native get_goal and create the matching corrective goal if the previous
goal is complete; report_progress follows evidence and update_goal complete only
at reviewable ALL-writer handoff. Keep R5/Q5 unchecked. Preserve current Muse
long-command cadence: initial native bash yield120000ms, routine reinspection
1-5 minutes/default2, retained handle/deadline, delivered completion, no busy polls.

- [x] **G28m — preserve replaced marker on failed registration**
  - Prerequisite: read fourth-review probe and design, verify writer release.
  - Owner/files: sole coordinator, startup-attribution.py and test_startup_attribution.py.
  - Change: retain original descriptor and root device/inode proof; failure unlink
    only the exact created marker in the unchanged root. Preserve replacements,
    foreign contents and replaced roots, report cleanup pending honestly. Ordinary
    real write/fsync/interruption failures still roll back exact new empty roots.
  - Checks: actual marker replacement while original fd stays open, then actual
    write/fsync/interruption failure; replacement bytes and inode survive and no
    success/rolled-back claim. Both automatic/explicit paths, root replacement,
    ordinary ENOSPC/short writes and existing refusals remain covered.
  - Evidence: met 2026-09-28T18:57:57Z: 5 new ReplacementPreservationTests,
    all failed pre-fix (false rolled-back + destroyed replacements, 1
    KeyboardInterrupt instead of pending); post-fix 35/35 with
    registration/ownership suites. Driver `2c0edc8a`, tests `3e424664`.
    Compact record in
    `generated-packet-5-fourth-corrective-muse-evidence.json` (g28m).

- [x] **G28n — durably register identity metadata process groups**
  - Prerequisite: G28m complete; freeze the small existing-tracker executor interface.
  - Owner/files: sole coordinator, same Python files; no shared-driver changes.
  - Change: use existing phase-tracker custody for versions/revision, emitter
    go-list, runtime resolver and builtin inventory commands in every actual
    before/after identity path. Register real child group/leader in the marker,
    clear only verified retirement, retain pending custody and propagate genuine
    registration/retirement failures. No fresh worker architecture or recursive
    supervision of tracker bootstrap process-status reads.
  - Checks: real metadata child alive with matching durable group/leader; meaningful
    coverage of each command family and actual collection caller, normal/error/
    timeout/handled interruption and failed-retirement marker retention. Never
    silently turn custody failure into optional missing metadata.
  - Evidence: met 2026-09-28T19:06:15Z: frozen `_tracked_command` executor
    (busy-guard, durable register, verified-only clear, custody retention
    on RetirementFailed, ownership failures propagate); all five metadata
    families routed through it with required tracker; 10 new
    MetadataCustodyTests pass 27/27 with tracker/identity/resolution/
    bounds suites, no strays. Driver `f8f8bc51`, tests `3d77778e`.
    Compact record in
    `generated-packet-5-fourth-corrective-muse-evidence.json` (g28n).

- [x] **G28o — bounded qualification and historical content continuity**
  - Prerequisites: G28m/G28n complete, no other writers.
  - Owner/files: sole coordinator, applicable tests and new compact evidence.
  - Checks: new regressions plus relevant custody/lifecycle contracts, one reused
    bounded validate-only graph for all actual AST tests with zero skips and full
    generated ordinary/diagnostic/bundle/state/builtin identity continuity against
    unchanged final raw. Retire graph keeper/groups/scratch with custody evidence.
    Strict TS/24 actual oracles repeat only for an explained relevant identity
    change. No sampling, installs, copies, private caches or retained bundles.
  - Evidence: met 2026-09-28T19:07:08Z: one validate-only build complete
    13/13 + 4 preflights; 224/224 tests 0 skips at 19:06:55Z; ordinary 282
    + diagnostic 10 + bundle `27ebf9cf` + state `5f43d077` + 13 builtin
    reachable specs all byte-identical to `fb060ec5`; strict TS / 24
    oracles not repeated (probe `299ae139`, `drivers/runtime.py`
    `96f3022a`, runtime/ unchanged; emission identical; no new reason);
    kept graph tool-retired, parent removed verified absent, no strays;
    no new sampling. Repaired driver `f8f8bc51` recorded separately in
    `generated-packet-5-fourth-corrective-muse-evidence.json` (g28o).

- [x] **G30d — truthful corrective handoff and writer release**
  - Prerequisite: G28o complete or concrete safe blocker.
  - Owner/files: coordinator, this checklist, startup result/next-investigation
    notes only if necessary and new compact corrective evidence.
  - Change: preserve historical raw/producer hashes, provisional status and
    separate repaired-tool qualification. Keep startup/authored-invoke/numeric/
    all-twelve queue and all prior negative evidence; no gain/exhaustion claim.
  - Evidence: met 2026-09-28T19:08Z: result note (`1a53c568`, 8742 B) keeps
    `77f29d04` historical producer with `3ebeff0c` repair history and adds
    fourth-review custody gaps plus separately qualified `f8f8bc51`; all
    numbers unchanged; follow-up note needed no change; no commit.
    Terminal handoff below. ALL coordinator/native file writers released;
    TUI remains idle for review/viewing. Native goal
    `goal-ccfd36a2-c65b-4899-bb45-f9728af43b90` completed at handoff.

## Muse terminal handoff (G28m-G30d correction complete, 2026-09-28T19:08Z)

Sole Contributor/MAX/YOLO coordinator, same TUI, no extra executor.
Finished two Codex-reproduced custody gaps in owned Python only:
replaced marker/root preserved on failed registration via opened-fd
plus root/current-path identity proof (G28m), and durable phase-tracker
custody for all identity metadata commands via one frozen
`_tracked_command` executor with required tracker (G28n). Deliverables:
`tools/performance/startup-attribution.py` (`f8f8bc51`),
`test_startup_attribution.py` (`3d77778e`, 224/224 pass 0 skips),
`docs/performance/generated-startup-result.md` (`1a53c568`, numbers
unchanged, `77f29d04`/`3ebeff0c` history plus `f8f8bc51` qualification),
this checklist,
`generated-packet-5-fourth-corrective-muse-evidence.json`. G28o qualified
on one validate-only build (13/13, byte-identical ordinary/diagnostic/
bundle/state/13-builtin continuity vs `fb060ec5`, retired verified
absent); strict TS / 24 oracles not repeated (probe, shared driver and
emission identities unchanged — no new reason); no new sampling. Final
raw `fb060ec5` and all earlier records byte-identical, never merged or
relabelled. No production/compiler/runtime/default/vendor/shared-driver
edits; no commit; no gain/exhaustion claim. ALL file writers released;
no groups/keeper/scratch retained. R5/Q5 and all next design belong to
Codex. Native goal `goal-ccfd36a2-c65b-4899-bb45-f9728af43b90` complete.

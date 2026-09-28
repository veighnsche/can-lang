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

- [ ] **R5 — independent review and checkpoint commit**
  - Prerequisites: G25-G30 plus G28a-G30a and G28e-G30b terminal, all writers released and
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

- [ ] **G28e — freeze the complete lifecycle and identity interfaces**
  - Prerequisites: previous explicit all-writer release, R5 rejection evidence
    and linked design read. Owner: sole coordinator, checklist/interface notes.
  - Changes: inspect native `get_goal`; reuse the matching goal or `create_goal`
    before delegating, without a token budget or overwriting unrelated work.
    Define concrete contracts for preparation/trial workers, retirement result
    and failed-retirement custody, registration rollback, explicit kept-graph
    keeper, required input/resolved-dependency records, phase timestamps and
    test seams. Follow the linked design, not a new broad planning exercise.
  - Acceptance: known four lifecycle defects have direct contracts, fd-3 helper
    and Bun-local parent timer retained, complete identities specified, production
    and shared helpers untouched; native goal success evidenced or blocker given.
  - Evidence: exact frozen interfaces and requirement-to-regression mapping;
    compact native goal reference/progress evidence in the handoff.

- [ ] **G28f — implement supervised phases, safe registration and identities**
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
  - Evidence: exact diff/hashes and compact focused reproductions of each old
    failure plus genuine cleanup failures if any; preserve both earlier records.

- [ ] **G28g — independently meaningful lifecycle/identity regressions**
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

- [ ] **G28h — join and qualify the whole corrected execution contract**
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
  - Evidence: commands/exits/timestamps/test counts/source/module/driver/resolution
    hashes, retirement and keeper handoff/cleanup, in new compact
    `generated-packet-5-final-corrective-muse-evidence.json`.

- [ ] **G29b — one complete replacement run with real input identities**
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
  - Evidence: new raw/identities/control/event/order data, independently reviewable
    arithmetic and explicit real cleanup outcomes; no execution tree retained.

- [ ] **G30b — reconcile and release for independent review**
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
  - Evidence: checklist and visible terminal handoff plus compact native goal
    lifecycle references, full execution retirement or exact pending cleanup owner.

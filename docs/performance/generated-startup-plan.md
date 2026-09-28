# Generated startup attribution

Status: ready design, 2026-09-28. Forwarding checkpoint `4589122c` and its
independent review are accepted. This packet attributes a supported startup
question before choosing a production change; it does not claim a speedup.
Implementation follows [G25-G30](generated-startup-tasks.md). The
[master campaign](performance-improvement-plan.md) continues afterward.

## Evidence and decision

`runtime_core.go:programImports` supplies broad Bun bindings to authored
packages. `program_state.go:stateImports` adds domain/platform constructors.
`emitStateModule` emits a synchronous `$canInitialize` with domain, bytes/CLI,
assets, SQL/crypto, utilities/core, platform groups, collections, AI/codecs and
checked value initialization. The actual function-body statement sequence is
authoritative; import ordering does not establish factory execution ordering.
`program_entry.go` configures diagnostics before initialization inside
`runEntry`; diagnostics reads real emitted source-index/maps. Runtime environment
evaluation consumes the private fd-3 snapshot once.

Existing startup measurements combine process launch, prepared JavaScript
loading/evaluation, initialization, correctness and exit. Bun build/transpile
duration cannot attribute loading. Neither broad imports nor historical launch
medians prove a safe reachability or lazy-initialization rewrite. The browser
profile differs and is outside this Bun attribution packet.

Three fresh completely reworded equivalent consultations are saved under
`.performance/performance-push-20260928/generated-startup-request/response-[1-3].json`.
The equivalence record checks every prose leaf and identical factual/code inputs.
All choose separate actual stages and temporary AST instrumentation. No selection
disagreement arose; shared framing still limits advice. Independent source and
execution checks, not classifier agreement, determine acceptance.

## Separate opt-in workflow

Add companion attribution tools under `tools/performance/`, without changing
production emit/runtime files, existing driver behavior or global defaults.
Default operation is validation only; timing requires an explicit `--measure`.
Codex authorizes one bounded timing run for this packet under the campaign-only
quiet waiver. Do not poll idle, control apps or power settings, or launch any
other task-owned build/test during sampling.

Use one shared-cache `perfemit` build with `GOMAXPROCS=2` and `-p=2`; emit the
actual runtime performance fixture with a zero-byte runtime path inventory.
Verify placeholders and replace them with the checkout runtime link before any
execution. Link existing `node_modules`; never copy source/dependencies. Generate
prepared JavaScript only as temporary build output from that actual emitted
graph and its referenced runtime inputs. Record TypeScript transformations and
relative import rewrites, and retain package/external dependency resolution.
Do not transpile the entire checkout or create a private cache.

Prepare an ordinary JavaScript module graph and a Bun ESM bundle using the same
small oracle facade over the actual generated adapter. Its `initialize` calls the
real initializer; its untimed verifier executes doubled, generic/captured map,
fold and frequency checks with their matching runtime data/completion modules.
Keep runtime branding coherent in the bundle: do not compare bundled carriers
using a separately instantiated external runtime. Keep the module and bundle
root/facade identity explicit; this is adapter/facade-root startup, not total
production entry startup. A minimal native module is a process/import control.

Record Go emission, JS transformation and bundling times separately as one-time
preparation observations. They are not startup samples or before/after evidence.

## Stages and instrumentation

Each measured child is a fresh Bun process. Its launcher must not statically
import Can runtime before its dynamic-import timer. Supply the existing private
`{}` environment snapshot on fd 3 through the established driver command helper.
Report parent launch-to-exit wall duration separately from child stages:

1. Dynamic import and evaluation of the prepared selected root.
2. Separate diagnostics import when applicable, without charging it twice.
3. `configureDiagnostics` against the actual unmodified emitted metadata and
   original generation entry URL; record that metadata root precisely. This
   probe does not claim transformed JS stack-map accuracy.
4. Synchronous `initialize()` duration. No `await` inside its timer.
5. Correctness after all timers, with child exit/output verification.

Retain ordinary module and bundle stage totals as primary observations. An
additional diagnostic module-graph phase may instrument only the owned generated
state output. Parse the actual `$canInitialize` body with the installed TypeScript
AST, enumerate all top-level executable statements, and place before/after
observations at exact spans without adding a lexical block or changing order,
argument evaluation, variables or control flow. Preserve original statement
bytes/hash, position, binding labels and factory-call identities. Include
non-factory initialization statements so coverage cannot silently omit work.
Multiple calls in one statement remain one inclusive statement observation.

Freeze this transform interface before parallel test authoring. Fail closed on
unsupported control/syntax, missing or repeated marks, unknown graph resolution,
unmatched statement order or failed correctness. Do not execute a partially
transformed initializer. Record the instrumented variant's distinct identity;
never overwrite checkout sources. A compact transformation record and original
state hash suffice; do not retain source copies. Statement observations include
argument preparation and observer overhead. They are not isolated factory times,
and their sum does not replace an ordinary unprofiled stage total. Do not subtract
independently measured medians to manufacture exclusive costs.

## Sampling and evidence

Use six independent driver-process trials per profile, two excluded warmup
batches and seven accepted batches, one fresh child startup per batch. Preserve
the existing standard sampling structure rather than treating 54 observations
as 54 independent headline trials. Counterbalance profile order by trial, run
sequentially, and retain excluded warmups, raw child stages and parent durations.
The diagnostic instrumented phase is separate from ordinary profiles. Every
accepted launch must pass its oracle, exact exit/output and complete expected
event coverage. Do not merge partial failures into a successful comparison.

All results are explicitly **non-isolated/busy-host**; record lightweight process
activity before/after trials without idle gating or command arguments/environment
secrets. OS/Bun caches remain uncontrolled. Report per-trial medians and across
trial median/range/MAD, units, exact scope and identities, with no p95 and no
causal speedup claim. Small or noisy stages remain unresolved when data cannot
support a remedy. Preserve raw rows and hashes compactly, not execution artifacts.

The report must distinguish import inventory from module evaluation attribution:
file count/bytes are context, not time per module. Bun bundle tree shaking is part
of that prepared profile; it does not prove a production emission rewrite safe.

## Correctness, storage and follow-up

Validate the unmodified emitted TypeScript with strict TS and the existing 24
actual generated/native oracles. Validate prepared module/bundle and instrumented
routes against real generated results and successful domain/error-plan creation.
Pin AST inventory coverage to actual emitted state, and test const scope,
statement order, repeated/missing events and unsupported control rejection.
Existing production origins, fresh failures, assertions, owners/leases, native
algorithms and publication code remain untouched. Runtime lint/format/check is
required if a justified authored runtime edit occurs; none is authorized here.

Every scratch allocation immediately records owner PID/inode and installs cleanup
for success, failure and handled interruption. Bound scratch to 64 MiB and 500
compiled files, bounded command timeouts and a 300-second sampling phase. This is
an individual measurement bound, not a campaign stopping cap. Retire child groups
before cleanup, delete temporary compiled graphs/bundles/prompts, preserve only
compact evidence (target at most 2 MiB) and report cleanup failure. Reuse installed
Bun and shared Go cache; no installs, disposable checkouts or retained bundles.

Codex independently verifies the implementation and evidence, commits accepted
tools promptly, then designs the next supported fix from actual attribution and
source usage/side effects. Per-site authored invoke proof and secondary numeric
JSON retention remain ready investigations. All twelve slices stay dispositioned;
the final exhaustion review is still pending.


## R5 correction design (independent review, 2026-09-28)

The initial Muse run is retired and its source is not independently accepted.
Independent validation passed all 81 tests with actual state inventory enabled,
strict TypeScript and 24 actual emitted/native cases. Recalculation matches all
216 original rows. That does not establish the missing failure or identity
contracts. Compact independent evidence is in
`.performance/performance-push-20260928/generated-packet-5-independent-preliminary-review.json`,
`generated-packet-5-independent-validation.json` and
`generated-packet-5-independent-negative-probes.json`.

Reproductions: negative/NaN/infinite stages and nonfinite events are accepted;
six accepted batches can produce a report intended for seven; an unregistered
prefix-shaped directory is deleted; a bad manifest output causes deletion of a
preexisting instrumented output; a whole-source identity mismatch and an omitted
statement are accepted; TypeScript UTF-16 spans fail Python validation when an
astral character precedes the initializer. The original scratch is gone, so its
missing prepared graph/dependency/driver identities cannot be reconstructed as
historical evidence. Keep the original raw record and Muse evidence unchanged,
explicitly provisional; a fresh corrected run is authorized for this concrete
reason, not to repeat successful sampling for reassurance.

The correction remains inside these opt-in tools and their tests/notes. Reuse
existing storage/process patterns where suitable, without modifying their global
contracts or adding a quiet gate. Freeze correction interfaces before disjoint
test authoring. Preserve exact stage/oracle/profile semantics and existing fd-3
helper; no production or default changes.

- Immediately register every allocation with durable PID/start and root/marker
  device/inode identity. Path prefixes and a memory-only set are not ownership.
  Refuse foreign allocations, symlink/replacement drift and active owners/groups.
  Recover abandoned owned work with an explicit bounded retirement path. Kept
  validation graphs transfer cleanup ownership to a recorded coordinator and
  must have a concrete retirement step. On any cleanup failure retain ownership,
  record the actual error and mark retirement false; never silently claim success.
- Own and supervise preparation/trial process groups. On command timeout,
  sampling deadline or handled SIGINT/SIGTERM, bounded TERM/KILL/reap must retire
  descendants before scratch deletion. Do not rely on subprocess.run killing
  only its direct child. Preserve the established fresh private fd-3 boundary.
- Reject nonfinite/negative measured milliseconds and parent durations; require
  null for unmeasured stages and correct profile-specific stage statuses. Validate
  exact trial/profile/batch identity, warmup designation and unique complete
  membership at collection and report boundaries. Reject omissions, duplicates,
  unknown IDs, booleans as numbers and malformed counts. Diagnostic event times
  must be finite/nonnegative and globally nondecreasing in exact paired order.
- Validate whole-source hash/byte count and UTF-16 span boundaries, initializer
  extent and complete ordered coverage against the actual AST inventory. Do not
  accept a caller-adjusted count as proof that every statement was included.
  Preserve original statement bytes, lexical scope and instrumentation identity.
  Refuse output/input aliases and preexisting outputs before writes; rollback
  only files created by this invocation, never preexisting or replaced files.
- Save a complete compact prepared JavaScript graph inventory (ordinary and
  diagnostic), source/runtime input identities, facade/launcher/probe/driver
  hashes, installed compiler/runtime/tool versions, and exact resolved dependency
  identities actually used. Do not hash unrelated dependency trees or copy them.
  Freeze these identities before trials and verify after sequential sampling;
  drift fails the run. Include the inventory in raw evidence before cleanup.

Qualify the corrections with meaningful negative tests and real bounded lifecycle
checks (controller plus descendant, timeout and handled interruption), actual
state inventory with zero skips, strict TS and 24 existing generated/native
oracles using one reused build. Then make one fresh corrected busy-host run with
six independent trials, two warmups and seven accepted batches/profile. Preserve
original data separately; do not merge partial or historical trials. Regenerate
current reports only from corrected raw data, with explicit old-record limits,
actual preparation/sampling timestamps, median/range/MAD and inclusive statement
caveats. Correct the old report's trial 4/trial 3 prose discrepancy explicitly.

The next-investigation note must qualify the assertion that fixed Intl.Segmenter
arguments cannot throw. Native capability/constructor effects and first-use
failure timing need source/contract investigation before a lazy rewrite. AST
callee labels inside nested function bodies are syntactic inventory, not proof
those factories executed. No production remedy is authorized in this correction.


## Second independent review: complete phase supervision and input identity

Codex's independent review after the G28a–G30a all-writer release rejects R5.
The 216 rows and all 57 statement summaries recompute correctly, but that does
not establish lifecycle safety or missing execution identities. The actual
controlled success-path probe left a live descendant after `run_supervised`
returned; marker-write failure left an unregistered allocation. Root retired
that owned group and temporary tree immediately. See
[compact independent evidence](../../.performance/performance-push-20260928/generated-packet-5-independent-final-review-probes.json).
Source review also finds preparation still calls unsupervised `driver_command`,
`run_measure` clears the group on every path before proving retirement, and
post-kill `communicate()` can wait without a bound. The dependency record has
specifier names/package-lock/link identity but no resolved module contents or
actual source/runtime input inventory. Existing total timestamps include
preparation while the result labels them sampling. Both prior raw runs and
Muse evidence remain unchanged and provisional; no missing historical identity
can be filled using a later reconstruction.

Three fresh equivalent Jev Choice requests, responses, equivalence review and
reconciliation are `generated-supervision-request/response-[1-3].json`,
`generated-supervision-equivalence.json` and
`generated-supervision-reconciliation.json` under the same evidence directory.
All choose phase workers (probabilities .93/.98/.93); differing strength is
advice only. Independent reasoning selects a local preparation controller plus
existing per-trial controllers rather than introducing wrappers around every
Bun launch: established fd-3 transport and the parent timer around the real Bun
spawn stay inside the worker. No shared driver or global default change.

The next correction must meet these requirements:

1. One durably registered owned process group encloses the entire preparation
   phase: Go compilation/emission, inventory, probe, transpilation, bundle and
   preflights, including helper subprocesses. Trial controllers use the same
   lifecycle contract. Commands that inspect versions/resolution also belong to
   a supervised phase. Registration failure must retire the newly spawned group.
   All normal, nonzero, timeout and handled-signal exits perform bounded
   TERM/KILL/reap and bounded pipe drainage. A successful controller with live
   descendants cannot hand off as retired. Preserve the actual child's failure
   while reporting retirement failures explicitly. Never clear an active group
   marker or delete its scratch until retirement is proved; keep recoverable
   PID/start/group/inode evidence when it cannot be proved. Parent cancellation
   covers preparation as well as trials. Do not signal a mismatched/reused owner.
2. Register allocation ownership before fallible work, or safely roll back the
   exact newly allocated inode if marker creation fails. Cover generated and
   explicit paths; do not delete foreign/replaced roots. Retained validation
   graphs need explicit custody by the live consuming coordinator/test owner,
   with its PID/start identity; the exited producer is not the keeper. An active
   keeper prevents abandoned recovery. Preserve all existing output/alias,
   foreign/active/replaced-path refusal and honest cleanup-error contracts.
3. Freeze complete real input identities before use and revalidate after use:
   relevant Can fixture/compiler/emitter/build inputs, actual linked runtime TS,
   tooling/drivers/instrumentation/launcher, prepared ordinary/diagnostic/bundle
   graphs, actual installed executable identity and built-in runtime identity,
   and actual resolved external module/package content identities. Resolve with
   the actual runtime and importer context; capture reachable dependencies and
   resolution links/manifests, without copying code or walking unrelated
   node_modules. Distinguish resolved/reachable imports from syntactic labels;
   reject incomplete identity records, ambiguous required resolution and drift.
   Empty equal dictionaries, unresolved bare-name lists, missing required hashes
   or null required source identity are not sufficient. Use bounded inventory;
   report a real blocker rather than silently relaxing completeness. A complete
   conservative inventory may include unused inputs if its scope is labelled.
4. Separate preparation and sampling start/end from overall wall bounds. Keep
   parent launch timing inside trial controllers around actual Bun invocation;
   supervisor/identity work is not part of that interval. Retain ordinary
   counterbalance, six trials, two excluded warmups, seven accepted batches per
   profile and one fresh startup per batch, diagnostic last; complete finite
   stage/events/AST/oracle/membership checks and inclusive timing interpretation.
5. Preserve original and first-corrected raw/evidence bytes. Their arithmetic
   remains useful; neither has newly complete identities. After lifecycle and
   identity qualification, one replacement busy-host run to a new `-final.json`
   is authorized specifically because the missing identities cannot be repaired
   retrospectively. Never merge trials or repeat sampling simply for reassurance.
   Keep 300-second sampling, 64 MiB/500 compiled-file scratch limits and a compact
   evidence target of 2 MiB per execution record; scope is adapter/facade-root,
   not total production `runEntry`. No p95 or causal speedup claim.

This is a correction of the new opt-in tool, not a production remedy. Shared
helpers/runtime/compiler/defaults/vendor/publication remain outside scope.
Use one saved checklist and the existing released interactive Muse session.
Before delegation use Muse's native `get_goal` and reuse a matching active goal,
or native `create_goal` for these repairs if none is active; never overwrite an
unrelated goal. No arbitrary goal token budget. Align native `report_progress`
with task evidence, and complete `update_goal` only at the all-writer release
and reviewable handoff. If native goal tools are unavailable/rejected, report
that blocker before delegation. Actual session/goal references belong in the
Codex monitoring record and compact handoff, not copied mutable checklist state.


## Third independent review: narrowly repair remaining failure branches

Codex verified explicit ALL-writer release and native goal completion before this
revision. `generated-packet-5-independent-third-review-probes.json` reproduces
ENOSPC at actual marker `os.write`: the marker is removed but the new unregistered
root survives because registration only catches ValueError. A real child writing
1 MiB to stdout stalls at the 64 KiB pipe buffer under wait-first supervision,
times out, and returns only that prefix. Both controlled groups/trees were retired.
`generated-packet-5-independent-third-review-resolution.json` additionally proves
that two importer-local packages named alike resolve to different files, whereas
the collector claims one first-importer file identity for both contexts.

These violate the existing registration, bounded supervision and complete-identity
contracts; they require narrow corrections, not a new architecture or another
measurement packet. Keep phase workers and actual Bun-local parent timing. Drain
stdout/stderr concurrently while observing leader exit independently of pipe EOF;
on leader exit, retire descendants before bounded final drainage. Use standard
library pipe multiplexing without extra files, helper processes or persistent
threads. Preserve exact output within a 4 MiB combined capture bound; excess is an
explicit failed command with verified retirement, never silent truncation or a
complete trial. Registration must roll back the exact newly allocated directory
for real write/fsync errors and handled interruption, preserving foreign/replaced
paths and reporting rollback failures. Detect/retry short marker writes.

This attribution packet's actual reachable bare dependencies are thirteen runtime
builtins. Builtin identity may be coalesced under the actual Bun revision; a file
dependency cannot be declared complete from first-importer resolution or a single
entry-file hash. Fail closed on reachable nonbuiltin file dependencies until
importer-specific ESM conditions and their transitive content closure are supported.
Do not expand this repair into package graph implementation. Distinguish actual
reachable importers from the wider syntactic test-file inventory; unsupported file
contexts must not yield a complete frozen identity/report. Unreachable resolutions
remain explicitly informational.

Codex independently recomputed all 69 profile/stage/statement summaries, exact
216-row membership (48 excluded warmups, 168 accepted batches), counterbalance,
bounds and equality. All 456 current runtime/emitter input hashes and all producer
driver hashes match; the four earlier records remain unchanged. Evidence is
`generated-packet-5-independent-third-review-arithmetic.json`. These checks validate
the calculations, not the whole tool. Preserve final raw fb060ec5 with its actual
77f29d04 producer hash; never relabel it as produced by repaired code. No additional
sampling is authorized by this correction: the failures do not disprove successful
small-output trials over the thirteen builtin roots. Revalidate the generated graph
identities in one bounded correctness build, record the repaired driver separately,
and leave independent acceptance to Codex. Broader successful generated/native/TS
checks need repetition only if their production/probe/driver identities change.

## Fourth independent review: finish marker and metadata custody

After the explicit G28i-G30c ALL-writer release and native goal completion,
Codex rechecked the real failure paths. Actual ENOSPC now leaves no allocation,
and actual 1 MiB stdout and stderr commands return their full output successfully.
The controlled probes and all their groups were retired. Compact evidence:
`generated-packet-5-independent-fourth-review-probes.json`.

Two requirements from the existing ownership design remain unmet. During a real
marker write failure, replacing the marker while its original descriptor remains
open causes `_write_owner_marker` to unlink the replacement, then report the
allocation rolled back. Failure cleanup must compare the originally opened
descriptor's device/inode and the original root identity with the current paths
before unlinking anything. Retain a replacement or foreign/nonempty root intact
and report cleanup pending; remove only the exact newly created marker and empty
allocation on an ordinary write/fsync/interruption failure. Do not use a path
name or the intended JSON payload as proof of custody.

The actual `bun --version` identity probe launches a new process group with no
`on_start` callback; the live marker's process_group stays null. The version,
revision, emitter go-list, resolver and builtin inventory commands execute after
the preparation worker retires. Their in-memory supervision does not give later
abandoned recovery durable knowledge of a still-live child if the parent dies.
Apply the existing `_phase_tracker` registration/verified-retirement contract to
these sequential metadata commands, including before/after identity collection.
Freeze one small owned executor interface shared by those helpers, retaining the
same single active-group marker. Registration/retirement errors must propagate
as ownership failures, not become optional missing metadata or clear the marker.
Do not introduce another worker architecture, package resolver, helper process,
cache or timing profile. Keep necessary bounded process-identity/bootstrap status
reads distinct from the substantive metadata jobs; avoid recursive supervision
of the process-status operations that establish the tracker itself.

New regressions must exercise actual replacement during write/fsync/interruption
failure and inspect the durable marker while real metadata children are alive.
Cover each substantive metadata command family, normal/error/timeout/handled
interruption and failed retirement, with controlled children and exact cleanup.
One reused bounded validate-only graph qualifies actual AST tests and generated
content continuity. Existing strict TS/24 oracles need repetition only if their
relevant identities changed. Preserve all historical raw/evidence bytes and the
actual fb060ec5/77f29d04 producer relationship; no further measurement is authorized.
This small correction stays with the same released Muse coordinator, sequentially,
under a new matching native corrective goal. Independent acceptance and the
startup/authored-invoke/numeric/all-twelve queue remain Codex-owned.

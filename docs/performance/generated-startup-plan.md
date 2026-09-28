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

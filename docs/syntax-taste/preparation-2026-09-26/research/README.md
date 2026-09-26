# P03 research record — 2026-09-26 round

## Coverage

| File | Topics | Findings |
|---|---|---|
| [r05-r08-core-library.md](r05-r08-core-library.md) | R05 iteration/collections, R06 failure composition, R07 assertion setup, R08 authoring policies | F-R05-01..05, F-R06-01..05, F-R07-01..04, F-R08-01..04 |
| [r01-r03-r09-browser-host.md](r01-r03-r09-browser-host.md) | R01 host integration, R02 browser controls/UI, R03 captured routes, R09 editor | F-R01-01..05, F-R02-01..07, F-R03-01..02, F-R09-01..03 |
| [r04-r12-r15-lifetime-storage.md](r04-r12-r15-lifetime-storage.md) | R04 budgets/cancellation, R12 failure observation, R15 S3 | F-R04-01..06, F-R12-01..03, F-R15-01..04 |
| [r10-r11-r13-r14-r16-backend-ops.md](r10-r11-r13-r14-r16-backend-ops.md) | R10 relational, R11 workers/outbound, R13 deployment, R14 AI, R16 docs | F-R10-01..05, F-R11-01..04, F-R13-01..04, F-R14-01..03, F-R16-01..04 |

Every packet reports per topic: best current idiom + limitation (file:line
anchored), platform facts with versions, probes run or why none, facts vs
uncertainties vs counterexamples, guarantees to preserve, and open questions
split into technical decisions vs user syntax choices.

## Method and delegation

Four bounded read-only research subagents ran in parallel, one per topic group
above. No repo writes; disposable probes permitted only under `/tmp`.
Model-selection rationale (skill: choose lowest fitting combination; Sol-Medium
is the normal technical default; escalate for cross-file/uncertain/risky work):

- Core/library (R05–R08): cross-file checker/runtime/spec analysis with
  material trade-offs → Sol-High equivalent.
- Browser/host (R01–R03, R09): cross-file + native DOM/build semantics →
  Sol-High equivalent.
- Lifetime/storage (R04, R12, R15): cross-file, uncertain failure semantics,
  data-loss-adjacent defect → Sol-High equivalent.
- Backend/ops (R10, R11, R13, R14, R16): bounded scans + docs verification →
  Sol-Medium equivalent.

The coordinator verified the highest-risk anchors independently before accepting
the packets: S3 `scrub()` end+delete vs catalogue line 12399, Boolean-order /
local-forwarding / near-capture checker sites, LSP capability advertisement,
static-route capture rejection, invoice `env::required("")` startup. All match.

## Probe policy outcome

**No reruns were needed.** Every disputed-or-insufficient claim in the review
already has retained executed evidence at the identical revision and toolchain
(Bun 1.4.2, Go 1.27.1), or is a source-verified fact re-read at HEAD:

- Recursion 100-vs-20k: retained source + emitted code + logs + harness.
- Result-data 5-root pass + `emits [failure]` rejection: retained logs.
- Owner-assertion rejection: retained compile log.
- DOM semantics: retained Chromium probe outputs pinning exact adapter calls.
- Race-drain: retained probe + corroborating `coordination.test.ts:537-583`.
- S3 discrepancies: textual catalogue-vs-implementation contradiction + own comments.
- Staged qualification: UP26/UP27 shards at `be95d009` (docs-only delta to HEAD).

Per P02.3, rerun only on changed code, missing coverage, or uncertainty — none
applies. This also honors the machine constraints (no long full-tilt runs on
this Air). New probes are specified instead where the *remedy* (not the finding)
needs evidence — see P04 experiments.

## New findings beyond the review (to carry into P04/P05/P07)

1. **SQL native foothold:** `Bun.SQL Query.cancel()` exists in pinned
   `bun-types@1.4.2` (`sql.d.ts:454-470`); Can never retains the handle.
   Server-side effect per dialect unstated — needs isolated qualification.
2. **S3 remedy candidate:** `NetworkSink.end(error?)` + "Writer will
   automatically abort multipart upload on error" (`s3.d.ts:69-91,574-583`).
   Whether it releases the event-loop pin without completing the key is
   unstated — highest-value question for isolated storage qualification.
3. **Worker-claim SQL question:** whether `SELECT ... FOR UPDATE / SKIP LOCKED`
   passes the PG descriptor backend is untested — needs a descriptor probe at
   implementation time, not a syntax change.
4. **Carrier-auth doc gap:** webhook `/outbox/pending` + `/outbox/ack` take no
   secret (any port holder can list/ack); README "stated limits" omits it.
   Minimal correction identified (bind to loopback / trusted carrier).
5. **Stale Linux sentences persist in two places:** top-level `README.md:99` and
   `distribution/README.md:196` ("No Linux support is claimed") — browser half
   already fixed. Minimal rewording identified.
6. **R16-01 minimal fix identified:** replace `match call env::required("")`
   with bare `env::invalid_name("")` completion; `startup_window` arms unaffected.
7. **Shutdown nuance:** `server_wait` signal path awaits `native.settled` with
   no wait bound (`server.ts:488-511`); `shutdownMs` applies to `server_stop`
   only. Catalogue wording ("timeout does not revoke ownership") is honest.
8. **SELECT breadth correction:** CTE/UNION/JOIN/subselect/OFFSET/ILIKE/aggregates
   all admitted — the R10 gap is mutations/identity/schema, not SELECT power.
9. **Unused foothold:** `ownerSignal` integration in transport deadline exists
   but no production caller passes it — a future ingress-lifetime signal point,
   not a live policy.

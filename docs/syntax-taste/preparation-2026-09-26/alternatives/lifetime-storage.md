# P04 alternatives — lifetime/storage/observation (R04, R12, R15)

## R04 budgets, cancellation, ownership

Layering (from the prior reconciliation, retained): operation contracts
first, under a written request-policy spec. The three review-level options:

- **O1 operation deadlines first.** Add effective native SQL/connection
  deadlines and compose operation budgets within the existing drain policy.
  Native footholds: `Bun.SQL Query.cancel()` (pinned types; server-side
  effect unqualified), transport `Deadline` (already aborts fetch),
  `process::run` SIGTERM→SIGKILL (already true termination). Needs: per-adapter
  interruption contracts (what cancel means for SQL/S3/streams), budget
  composition (serial 5s+5s+5s vs one 5s budget — open), disconnect/SIGTERM
  propagation into the policy. Preserves leases, failure identity, honest
  unknown-write outcomes by construction.
- **O2 scope policy.** Explicit cooperative cancellation + supervised loser
  ownership at request/scope boundaries (losers move to a supervisor instead
  of blocking the response). Needs: supervisor lifetime/shutdown contract,
  late-failure diagnostics, cleanup/effects rules. Addresses the "race does
  not bound response latency" finding directly; must never release a live
  lease because a timer won.
- **O3 deployment limits (retain).** Keep semantics; rely on database/proxy/
  process limits with a documented external latency boundary. Cost: docs +
  operator recipes. Fails W5's bounded-behavior acceptance unless the user
  scopes W5 down.

Possible surface after evidence: caller deadline/cancellation operands on
SQL and on browser `action::request/post` — only once the native contract
each maps to is qualified. No syntax established as necessary yet.

Distinguishing evidence (implementation time): a real HTTP hedge measurement
comparing O1 vs O2 on response time, lease preservation, late-failure
diagnostics, client disconnect, cleanup, effects, and shutdown; plus
isolated qualification of `Query.cancel()` per dialect and of any Bun
serve-stack disconnect signal. Bun-types notes: `close({timeout})` takes
seconds (Can converts correctly); `reserve({signal})` aborts only queued
reservation, not running queries.

Failure cases: stalled SQL mid-transaction (commit uncertainty must survive);
headers sent then body stalls; client disconnects mid-handler; overlapping
requests sharing a pool; SIGTERM with nonsettling work (supervisor SIGKILL
boundary documented, not wished away).

## R12 failure observation

- **O1 retain manual-only.** Authors catch standard failures and log safe
  details per handler. Cost zero; fragility retained (every app must add
  wrappers correctly; faults in wrappers/adaptation/dispatch/drain stay
  silent).
- **O2 built-in redacted reporter (favored).** A small automatic hook at the
  server request boundary (`serveOuter`, possibly dispatch/adapter layers)
  reusing main/late-owner/browser machinery: redacted payloads, declaration
  identity for domain errors, category + occurrence for standard failures,
  correlation/source identity per request. Client responses stay fixed 500s
  (no leakage). Exactly-once-per-occurrence discipline shared with existing
  reporters. Design points: correlation ID source, redaction rules for
  request context (reuse `entry.ts` policy), layer placement.
- **O3 author-registered hook.** Authors register a callback receiving the
  redacted report. More flexible; the boundary must survive a throwing
  callback (cf. browser `settle`). Needs a registration/lifetime rule.

No syntax established as necessary for any option. Failure case: a handler
that throws *and* whose logging wrapper throws must still produce exactly one
redacted server-side record with its correlation identity.

## R15 S3 cancellation and deadlines (defect resolution, not taste)

### Cancellation/deletion discrepancy (F-R15-01/02)

- **O1 fix code to match catalogue (strongly indicated).** Make cancel truly
  "release the sink without completing, so the key never materializes; never
  deletes the key." Prerequisite: verify a no-complete release path —
  prime candidate `sink.end(new Error(...))` ("Writer will automatically
  abort multipart upload on error" per pinned `s3.d.ts`), which current code
  never tries. If verified: cancel-before-first-byte stays safe (lazy sink),
  cancel-after-first-byte aborts without publish or delete.
- **O2 re-specify the catalogue (disfavored).** Declare cancel destructive
  with caller-visible warnings. Only if no no-complete release exists in the
  pinned Bun API. Even then, a silent `delete()` of caller data under a
  "cancel" name is unacceptable — the contract must name the destruction and
  its window honestly.

Either way, terminal-state guards (`upload_closed` after finish/cancel) and
honest unknown-outcome reporting stay.

### Deadline semantics (F-R15-03/04)

- **O1 effective deadline.** Race the pump awaits against a timer with a
  native-abort or abandonment contract that bounds never-settling reader,
  writer, flush, completion, and metadata awaits.
- **O2 documented between-awaits bound (retain).** Keep the loop-top check
  and document honestly that it cannot bound a hung await. Fails W5 unless
  scoped out.

### Isolated storage qualification (implementation time, disposable bucket, env credentials)

(a) cancel-while-replacing an existing key: seed known bytes+etag →
begin/write/cancel → assert original bytes+etag preserved (distinguishes
preservation from absence); (b) concurrent-reader observer visibility during
cancel/abort (never observes partial/transient bytes); (c) never-settling
`source.read`/writer/`end()`/stat under deadline (bounded return asserted);
(d) `end(Error)` pin-release + non-materialization; (e) service-failure
injection at each `scrub` await; (f) orphaned-multipart-part accounting
(AbortMultipartUpload is never issued today). Local fakes cannot establish
(a)/(b). No live deletion experiment was performed in preparation.

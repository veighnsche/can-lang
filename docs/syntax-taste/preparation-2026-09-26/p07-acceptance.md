# P07.4 acceptance cases + P07.5 assumption validation

Positive, negative, failure-path, and integration cases per workload (W1–W6).
Each case names its checking-vs-execution layer and whether mocked evidence
or live qualification proves it. Measurements follow P02.2. Production tests
requiring implementation are specified here, not claimed as passed.

## W1 second application (R02/R03/R09)

- Positive: second app imports shared keyed-table + field/error controls
  unmodified; both apps check, emit, and run in Chromium/WebKit/Firefox.
- Negative: a breaking control-API edit fails the non-edited app's check
  with a located diagnostic (no silent drift).
- Capture edit: route/capture/wire/body/result-leaf edits rebuild both
  targets or diagnose (extends the U03 edit-propagation legs).
- Failure-path: late reply after unmount; disposal mid-save; IME +
  normalization caret preservation — all covered per the R02 case list.
- Integration: captured HTML reads render through paired builds with truthful
  denied/not-found statuses.

## W2 second integration (R01/R11)

- Positive: second vendor added with no vendor-specific compiler patch and
  no hand-edited generated code; capability rejection + lifecycle tests green.
  Conditioned (P09-M1): if X-R01-1 selects catalogue-only for the widget
  class, generic catalogue additions via the standard reviewed path are
  permitted — the bar is *no vendor-specific patch*, not *no compiler change*.
- Negative: an unadmitted capability in the integration fails the browser
  closure with location evidence.
- Integration: adapter-tier and companion-tier conformance suites (as
  assigned by X-R01-1) run in CI shape.

## W3 reusable infrastructure (R06/R07)

- Positive: one retry/trace helper serves two unrelated callback error sets;
  oracle verifies attempt counts + failure-then-success ordering.
- Negative: adding an error to one caller requires no edits to unrelated
  consumers (verified by the edit-and-recheck leg).
- Owner setup: handler helper extracted with factory fixtures; useful tests
  retained; setup plumbing measured (X-R07-1) against the Q4 gate.
- Failure-path: unexpected fixture rejection fails closed with a located
  standard fault (never a forged owner value).

## W4 large iteration (R05)

- Positive: 100k-step state machine + growing aggregation + bounded worker
  batch complete; stack/memory measured per P02.2 (100/20k/100k).
  Under companion scope the "bounded worker batch" is the companion batch
  shape mirroring C-A step/failure reporting (lane-F task; P09-m5).
  Lowered-loop failures carry step index + occurrence linkage (P09-M8).
- Negative: non-lowerable recursion keeps current behavior + "not lowered" note.
- Failure-path: injected fault at step 60k reports step + declared failure
  (no native overflow); aggregation preserves order/failure identity.

## W5 controlled failures (R04/R12/R15)

- Stalled SQL/headers/bodies, disconnect, overlap, SIGTERM: bounded
  user-visible behavior + defined escalation + no post-disposal use +
  honest unknown-write outcomes (live legs, not assertion examples).
  Conditioned (P09-B7): cancel-verified branch (X-R04-1/X-R04-3 positive) →
  abort-path legs; cancel-absent branch → boundary-returns-at-budget legs
  with owned-until-settlement + unknown-write + supervisor escalation.
- Failure hook: one redacted correlated report per unexpected failure;
  client responses stay fixed 500s; double-throwing wrapper still reports once.
- S3, conditioned per branch (P09-B8, isolated bucket + env creds):
  O1 (cancel verifies) → cancel-while-replacing preserves original
  bytes+etag + concurrent readers never observe partial bytes +
  never-settling awaits return boundedly. O2 (destructive) →
  `cancel_upload` is REMOVED from the catalogue and replaced by explicitly-
  named `discard_upload` with a destructive contract + unknown-outcome
  reporting; legs assert the name removal, the explicit call sites, and the
  honest outcomes — never preservation. O2 orphaned parts: best-effort
  abort attempted, residual orphan risk measured + documented with an
  operator lifecycle recommendation (P09-m4).

## W6 production operation (R10/R11/R13/R14)

- Real PG app: generated identities, precise values, nullable audit times,
  JSON, concurrent replay, migration recipe — plus MySQL parity legs
  (widest matrix). "Migration recipe" = the operator-owned procedure
  (P19 deferral stands), not versioned migrations (P09-M6).
- Companion pair: bounded concurrency, retries/backoff, poison, crash
  recovery qualified against the versioned protocol; carrier endpoints
  authenticated; destination policy enforced.
- Paired Linux deploy on x86 (UP25): credentials, rollout/rollback,
  retained assets, old-browser-open scenario.
- AI: tenant budgets/correlation enforced + measured; model-change eval
  protocol executed for one in-scope feature; quality/cost/latency reported.
  The [R14 supplement](blocker-resolution/README.md) fixes combined token
  counting and immediate pre-dispatch typed rejection. Qualify complete-request
  bounds, parallel atomic admission, durable unknown holds, idempotent/late
  settlement, fixed-epoch transitions, direct-call guard coverage and safe
  context/failure reporting. X-R14-1 must qualify at least one real profile for
  the registered support-ticket-triage feature; all-profiles-unqualified blocks
  W6-AI despite correct unavailable-profile rejection tests.

## Authoring/tooling acceptance (R08/R09)

- Q1: true-first programs check; formatter rewrites to false-first;
  fixpoint + overlay validation; other match modes unaffected (negative legs).
- Q2: C8-shape programs check; lint flags the shape; formatter keeps source;
  type/error behavior identical (differential leg over existing fixtures).
- Q3: `with` bindings check/emit/capture correctly; unknown/duplicate/
  mistyped bindings error with CAN-CHECK-CAPTURE; fallback lookup unchanged;
  rename updates `with` sites.
- LSP: format/hover/references/completion/rename each demonstrated on the
  acceptance flow (extract/rename callback, change shared record, inspect
  contract, repair callers).
- Local references/rename (user 1A): all uses of a function-local binding are
  found/renamed; shadowed bindings and same-spelled names in different scopes
  remain untouched. Include captured references to that binding where they
  resolve to it; retain selected `with` parameter behavior and validated edits.

## Docs/examples (R16)

- Invoice startup uses a meaningful authored error; webhook limits include
  carrier auth; Linux sentences accurate; supported story consistent with
  all selected contracts; `canlc assert` green on touched examples.

## P07.5 unresolved-assumption validation

Two selected-design assumptions cannot be probed in preparation (no
provisioned services or credentials on this machine) and are therefore
CONDITIONED, not hidden — each implementation task carries its probe as
first-class acceptance with both branches specified:

1. `Bun.SQL Query.cancel()` server-side effect (X-R04-1) → R04 contracts
   specify per-adapter behavior for cancel-verified vs cancel-absent
   outcomes; no caller operand ships without a verified native meaning.
2. `NetworkSink.end(Error)` no-complete release (X-R15-1) → R15 contracts
   specify fix-to-catalogue vs honest-destructive branches; silent delete is
   rejected under both.

All other selected designs rest on source-verified facts or retained
executed evidence at the identical revision. No implementation task may read
"figure this out" — conditioned branches above are the complete list.

## P07.6 consolidation check

Specs (`p07-contracts.md`), decisions (`p06-answers.md`, `jev/findings.md`,
`p07-dispositions.md`), acceptance (this file), and shared interfaces
(`p07-reconciliation.md`) were cross-read for consistency: every CHANGE has
a contract + acceptance + interface owner; every RETAIN names its preserved
rule + tests; every DEFER/REJECT names its condition; Q1–Q3/Q7–Q8 surfaces
appear in exactly one contract each; conditional syntax (Q4–Q6) appears only
as gates. Required docs, examples, editor support, generated artifacts
(regeneration, never hand-edited catalogue/vendor), and operational
consequences are attached to their owning contracts.

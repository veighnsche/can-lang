# P07.2 preparation-returns log — contracts returned from qualification

When qualification (H10/H11) returns a contract to preparation, the return
is recorded here with its evidence, scope, and resolution. Only the
returned contract re-opens; nothing else in P07 is disturbed.
Format per entry: returned contract, returning evidence, scope boundary,
resolution options, resolution (once decided), IC2 effect.

## R1 — W5 S3-deadline scope (X-R15-3 negative) — RETURNED, awaiting user scope call

- Returned contract: the W5 "every hung read/write/flush/end/stat await
  bounded" acceptance scope for S3 (task-list experiment row X-R15-3).
- Returning evidence: E07 H1–H5 pin the remainder unbounded;
  E09 W5-S5b hung `source.read` pins unbounded on final adapters
  (`docs/implementation/evidence/2026-09-26/e09-w5.md:102-107`).
  H11 finding F1.
- Task-list mandate (line 933): "Honest between-awaits documentation is
  required but **W5 fails**. Return storage scope/contract to preparation
  for explicit resolution; no silent reduced scope or green IC2."
- Scope boundary: S3/storage deadline semantics only. S1–S4 preservation
  legs, discard/unknown-outcome vocabulary (X-R15-1), and all non-S3 W5
  legs are unaffected and stay green.
- Resolution options:
  - (a) Accept the between-awaits bound as the supported contract:
    hung S3 awaits documented unsupported; W5 verdict becomes
    pass-with-documented-exclusion; H14 carries the exclusion.
  - (b) Keep W5 failed: IC2 stays blocked behind a future S3
    deadline implementation (new scope, new qualification).
- Resolution: USER-DECIDED 2026-09-27 — requirement RETAINED, neither
  option taken as offered. The user does not accept indefinite waits or
  a between-awaits-only exclusion: storage waits must be bounded. The
  failed acceptance condition is engineering work to implement and
  verify (bounded hung-await behavior + live MinIO validation).
- IC2 effect: blocking until the bounded-wait implementation is
  qualified; no waiver, no exclusion.

## R2 — W5-L3 overrun-tx peer release (E04 tx-ownership semantics) — RETURNED, awaiting scope call

- Returned contract: the at-budget peer-response ownership rule for
  overrun transactions (which scope owns the tx callback at bound
  expiry).
- Returning evidence: handler returns `sql::commit_unknown` at 155ms
  against the 150ms bound, but the peer receives `unknown:commit` at
  2009ms (tx-callback settlement, not bound expiry) because dispatch
  `drain` waits for callback tasks inside the request scope
  (`e09-w5.md:54-86`). E09: "Detaching tx ownership from the request
  scope is an E04 semantic change with disposal/commit implications,
  not a W5 leg fix." The L3 leg pins the split (handler <1500ms,
  peer ≥1800ms and <8000ms) and fails loudly on change. H11 finding F2.
- Scope boundary: tx-callback/request-scope ownership only. Pool-lease
  legs (L1/L2/L6/L7), handler-boundary behavior, and the escalation
  vocabulary are unaffected and stay green.
- Resolution options:
  - (a) Document the owned-until-settlement peer rule as the
    supported contract (peer releases at callback settlement for
    overrun tx; `stop` stays bounded via `shutdownMs`); file the
    E04 detach as deferred follow-up scope with the L3 leg as the
    trip wire.
  - (b) Commission the E04 semantic change now (detach tx ownership
    from request scope with disposal/commit design + full
    re-qualification of L1–L10 and the E04 suite).
- Resolution: USER-DECIDED 2026-09-27 — bounded waiting applies to the
  client response too; internal timeout detection while holding the
  response is insufficient. The ownership and cleanup changes need
  proper design and validation. A timeout must report uncertainty
  honestly — it must not imply the write was cancelled or rolled back.
  Engineering work commissioned (design via Jev consultation, then
  implement + validate; L3 leg pins flip to bounded peer response).
- IC2 effect: blocking until the ownership/cleanup implementation is
  qualified; no waiver.

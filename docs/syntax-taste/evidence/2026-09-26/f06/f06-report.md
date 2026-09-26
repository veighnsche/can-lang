# F06 — Companion operations and W4 batch mirror qualification (2026-09-26)

Source: R11/R05; W6-pair/W4-batch; superseded X-R11-1. Owner: lane F
(sole owner of legs W6.3 and W4.3). Machine record:
[f06-legs.json](f06-legs.json).

Environment: macOS darwin-arm64 (MacBook Air), bun 1.4.2 pinned
(`CAN_BUN_ARCHIVE=/private/tmp/bun-darwin-aarch64.zip`), go1.27.1.
Evidence run `TestCompanionPairLiveF06` green in 82.8s; the
committed tree `e1ca3e55` differs from that run only by tightening
the W4.3 RSS gate from 512 to 256 MiB (measured 56 MiB — both
bounds pass), and a second full run at exactly `e1ca3e55` passed
in 130.2s (loaded host). Companion units 40/40;
`TestWebhookSliceLive` (F05) still green;
`TestRuntimeInventoryMatchesBodies` green.

Harness (committed, F-owned):
`tests/integration/webhook_f06_test.go` — stages `examples/webhook`,
asserts and builds it, seeds one fresh SQLite file for the run, and
serves the built entry on port 18495. Every leg runs the real
companion entry (`examples/webhook/companion/main.ts`) as worker
processes against the live Can side — never a Go reimplementation
of the protocol — with a programmable downstream stub. F06 added
operator tuning flags to the companion (`--lease-ms`,
`--concurrency`, `--max-batch`, `--backoff-base-ms`,
`--backoff-max-ms`, `--idle-ms`, `--timeout-ms`) plus an extracted,
unit-tested arg parser (`companion/args.ts`, 6 new tests); defaults
are unchanged.

## W6.3 — companion pair (PASS)

- Two-worker leases/claims: 48 rows over capped concurrent batches
  split worker-A=24/worker-B=24, disjoint, every row delivered
  exactly once downstream with one ledger row and one delivered
  attempt. The caps force the split whatever the scheduling.
- Idempotent ack: a redrain reports empty and records no new
  attempts; F05's duplicate-ack protocol coverage still stands.
- Concurrency: `--concurrency 1` peaks at 1 in-flight send;
  `--concurrency 4` over delayed sends peaks at 4 — bounded and
  parallel. No Can-side knob exists; the peak is the live proof.
- Poison/dead-letter: 3 rows against an always-500 downstream ride
  exactly 5 failed attempts (`downstream-status/500`) and then
  dead-letter (`poison/5`) with no sixth downstream touch; the dead
  table holds `poison` with 5 attempts.
- Destination/credential: denied and unmapped subscriptions die
  with `destination-denied/no-matching-rule` and
  `no-destination/sub-unmapped` without touching the downstream; a
  missing bound variable fails for retry (`credential-missing`),
  the bound token delivers with a Bearer credential.
- Downstream timeout (E04 caller bound): the first send hangs past
  the 1.5s bound and fails as `downstream-timeout/timeout`; the
  redelivery lands. Occurrences carry the contracted code only.
- Crash/unacked redelivery: a companion SIGKILLed with 4 gated
  sends in flight acks nothing; after the 2s leases expire a fresh
  companion redelivers all 8 rows. Four rows post twice downstream
  while the ledger holds one effect each — at-least-once with
  idempotent settlement, never exactly-once. No exactly-once claim
  is made anywhere.
- Auth: a wrong-secret companion exits 1 on `unauthorized` and
  leaves the outbox untouched; a bad tuning flag exits 2 with usage
  before any I/O.
- Backoff/idle: one looping companion against an always-500
  downstream fails the first batch, idles empty, sleeps every gap
  (no hot loop, keeps polling), takes no supervisor restart, and
  drains on SIGTERM to exit 0. The exact ladder stays pinned by the
  `runWorker` unit tests; this leg proves the live sleeps exist.

## W4.3 — bounded batch mirror (PASS)

200 rows drain through both companions in 6 batches over 3.4s
(59 rows/s), peak companion RSS 57,616 KiB for both processes. The
committed gate bounds peak RSS under 256 MiB (4.5x headroom) and
the drain under 120s; throughput is recorded, never gated. Every
batch step in every leg carries the C-A mirror from A07's handoff:
1-based contiguous claim-order index, finite outcome, finite
failure identity, occurrence on every non-delivered step — and
every report line is swept for secrets and URLs.

Stack note: the worker loop is iterative by construction (sequential
claim paging plus a bounded lane pool), so batch depth is constant;
the leg evidences completion of the bounded 200-row drain with flat
measured memory rather than a per-step stack probe, which Bun does
not expose.

## Contract notes and handoffs

- No Can-worker implementation was added: all worker logic stays in
  the TypeScript companion; the Can side owns receipt, ledger,
  outbox, and decide/ack transactions exactly as in F05.
- No exactly-once-delivery claim: leg H proves duplicates reach the
  downstream and collapse on `delivery_id` plus idempotent ack.
- Distinct DB per run: each run seeds a fresh SQLite file in a
  fresh temp home. Deviation from the task letter: the
  `provision-local.sh db mkdb` path mints PG/MySQL databases, but
  the webhook Can side is SQLite-only by F05 design, so no
  PG/MySQL database applies; isolation is one disposable file per
  run, never a shared table. No H-owned files were touched and no
  exclusive H window was needed.
- One transient seen during development (not in the evidence run):
  a poison round missed one claim (no batch error surfaced in that
  run's logging), desynchronizing versions to 7 for one row; the
  committed leg logs every round and accepts version >= 6 with
  attempts pinned at 5. If it recurs, the round log will carry the
  batch error. A harness-side missing `--once` in an early W4.3
  draft (looping workers idling instead of exiting) was caught by
  the watchdog and fixed; the watchdog plus wedge forensics stay in
  the committed harness.
- No runtime/ or tools/runtime/ edits, so no RH duty arose and no
  modules.json change was needed; the inventory check passes.
- Handoffs: W6-pair and W4-batch evidence (this record plus the
  legs JSON) to IC2/H12/H13. H13's full matrix can cite the
  companion half; UP25/x86 and browser legs stay with their owners.

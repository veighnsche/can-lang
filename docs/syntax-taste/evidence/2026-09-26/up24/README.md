# UP24 staged qualification evidence

26 September 2026 · code candidate `08904291`, qualified at
`be95d009` (test/CI-only delta, see the [completion
audit](../../../post-upgrade-completion-audit-2026-09-26.md#candidate-match);
UP23 verdict step plus `TestHeavySlotCap` isolation).

Browser evidence below was captured via `CAN_BROWSER_EVIDENCE_DIR`
during the sharded qualification (six shards, `-parallel 2`, one
shard at a time; the single-log `-parallel 4` shape thrashed this
16 GB host — see `shards/shard1-gate3-killed-p4.log`). Digests for
every file: [`SHA256SUMS`](../SHA256SUMS).

## `reports/` — gate/browser artifacts, all passing

17 `report.json` files, every one with `"passed": true`:

- `s2-contract/` (shard2 `TestInvoiceContract*`): 6 scenario
  reports × 3 checks — route-save, limit-budget, capture-save,
  leaf-invalid, status-save, field-save — each with its
  `screenshot.png`.
- `s3-live/` (shard3 application legs): `forms` (8 checks),
  `dashboard` (5 checks), `accounts` (9 checks), each with its
  `screenshot.png`.
- `s4-browser/` (shard4 gate5/browser):
  - `conformance-chromium` / `conformance-webkit`: 17 checks each
    (includes the U05 "cancel" legs and the S03 "dispose" leg).
  - `grid-chromium` / `grid-webkit`: 32 checks each (S03
    focus/notice/pending legs).
  - `invoice-chromium` / `invoice-webkit`: 20 checks each;
    the webkit report records the expected `L-redirect-webkit`
    limitation (no same-origin 302 producible under WebKit
    interception; the redirect branch is qualified on Chromium).
  - `empty-chromium` / `empty-webkit`: 4 checks each.
  - `codec-chromium` / `codec-webkit`: `parity.json` plus
    `vectors.js` (wire-codec parity vectors, Chromium
    140.0.7339.186 and WebKit 26.0).
  - Every `report.json` directory carries its `screenshot.png`.

Shard1 (gate3) and shard5 (rest) emit no browser evidence, so no
`s1`/`s5` directory exists.

## `../up23-verdict.json` — UP23 guard verdict

Written by the standalone `CAN_UP23_OUT=/tmp/up23-verdict.json go
test ./tests/integration -run TestUP23WriteVerdict` step (101 s):
11/11 legs pass on Chromium 140.0.7339.186 and WebKit 26.0, bound
to commit `be95d009180437d361682a6fd37a9b8c442cb86a`, consumed by
the shards via `CAN_UP23_RESULTS`. (The `TestUP23WriteVerdict` row
skips inside shard5: it requires `CAN_UP23_OUT`.)

## `../shards/` — full-tree log set

- `shard0-units.log` — non-integration packages: 18 ok, 4
  no-test-files, 0 failures.
- `shard1-gate3.log` — `TestGate3`: 8/8 PASS in 929.7 s.
- `shard1-gate3-killed-p4.log` — the superseded `-parallel 4`
  attempt (killed under swap exhaustion; kept as history).
- `shard2-contract.log` — `TestInvoiceContract*`: 9/9 PASS in
  762.8 s.
- `shard3-live.log` — invoice-live + gate4 + applications: 13/13
  PASS in 695.7 s.
- `shard4-browser.log` — gate5 + browser: 6/6 PASS in 478.9 s.
- `shard5-rest.log` — remainder: 65 PASS / 5 SKIP / 0 FAIL in
  599.7 s.

Skips: the two Linux installed-artifact tests (deferred UP25),
`TestCurrentMySQLPersistence` and `TestCurrentS3Objects`
(unoperated external services), `TestUP23WriteVerdict` (needs
`CAN_UP23_OUT`; the verdict itself is archived above).
`TestMySQLDifferential` skips in shard0 (same MySQL reason).
Together the six shards execute all 106 `tests/integration`
tests plus every unit package exactly once.

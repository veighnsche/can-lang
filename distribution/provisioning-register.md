# Provisioning and evidence register (H01)

H-owned live-dependency register for the Can implementation program.
Each provision tracks readiness independently: one blocked provision never
blocks unrelated lanes. Logical lanes (A–H) are parallel workstreams, not
runnable capacity — the capacity notes below state what can actually run
at once. Unavailable live evidence is never a pass.

Source: R13; P08.5; S-platforms
([contracts](../docs/syntax-taste/preparation-2026-09-26/p07-contracts.md),
[lanes](../docs/syntax-taste/preparation-2026-09-26/p08-lanes.md)).
Provisioning tasks H02–H05 and C01 update their own rows; H keeps this
register. Credential values never appear here — env names only.

Evidence ID scheme: `EV-<task>-<nnn>` (e.g. `EV-H02-001`), recorded in
the row when evidence lands. No IDs are recorded yet.

## DB-PG — live PostgreSQL

- Status: **READY** (H02, 2026-09-26; operator record
  [provision-local.md](provision-local.md)).
- Owner: H (provisioning task H02).
- Consumers: E02, F02, F04, F06 (each uses an isolated database; no
  shared mutable tables).
- Pins: `postgres:17@sha256:f4c66b…91232`, observed `17.11
  (Debian 17.11-1.pgdg13+2)` aarch64 — exact match.
  Native `linux/arm64`, loopback `127.0.0.1:55433`, container `can-pg17`.
- Credential env names: `CAN_TEST_POSTGRES_URL` (per-database URL).
- Assigned resources: one server; databases `can_e02`, `can_f02`,
  `can_f04`, `can_f06` plus per-run `can_<lane>_run<k>` via `db mkdb`.
- Evidence IDs: `EV-H02-001` (psql version-pin + isolation roundtrip).
- Capacity: one server, separate databases per lane/run.

## DB-MySQL — live MySQL

- Status: **READY** (H02, 2026-09-26; operator record
  [provision-local.md](provision-local.md)).
- Owner: H (provisioning task H02).
- Consumers: F02, F04, F06 (MySQL parity legs).
- Pins: `mysql:8.4@sha256:0744…93fb8d`, observed `8.4.11`.
  Native `linux/arm64`, loopback `127.0.0.1:3307`, container `can-mysql84`.
  SQLite remains included and needs no provisioning.
- Credential env names: `CAN_TEST_MYSQL_URL` (per-database URL);
  `CAN_TEST_MYSQL_CONTAINER` (kill/restart legs).
- Assigned resources: one service; namespaces `can_b1_03` (fixed live
  harness shape), `can_f02`, `can_f04`, `can_f06`, plus per-run via
  `db mkdb`.
- Evidence IDs: `EV-H02-002` (`mysql.test.ts` 12/12, `mysql-tx.test.ts`
  5/5 incl. kill/restart leg).
- Capacity: one service, isolated namespaces per lane/run.

## S3 — disposable object storage

- Status: **READY, S3-protocol scope** (H03, 2026-09-26; operator
  record [provision-local.md](provision-local.md)).
- Owner: H (provisioning task H03).
- Consumers: E07, E09 (exclusive owners of run keys and the
  qualification harness).
- Pins: MinIO `RELEASE.2025-09-07T16-13-09Z`
  (`quay.io/minio/minio@sha256:14ce…bd8936e`), container `can-minio`,
  endpoint `http://127.0.0.1:9000`, region `us-east-1`. Real
  S3-wire-protocol server, not AWS S3: qualification covers S3-protocol
  behavior; an AWS-real leg would need a user-supplied bucket (no
  standing ask: none specified).
- Credential env names: `CAN_TEST_S3_ENDPOINT`, `CAN_TEST_S3_REGION`,
  `CAN_TEST_S3_BUCKET`, `CAN_TEST_S3_ACCESS_KEY`,
  `CAN_TEST_S3_SECRET_KEY`.
- Assigned resources: isolated bucket `can-b1-10`; per-run prefixes
  owned by the E harness; never the same key across concurrent runs.
- Evidence IDs: `EV-H03-001` (SigV4 roundtrip; `s3.test.ts` 19/19 live).
- Capacity: one bucket; per-run prefixes. Local fakes cannot close the
  S3 gate.

## BROWSERS — Chromium, WebKit, Firefox

- Status: **PARTIAL** — Chromium and WebKit pinned; Firefox runner
  **BLOCKED** pending lane-C provisioning.
- Owner: C executes browser provisioning (task C01); H retains this
  register row.
- Consumers: C02–C07 gate legs, H12 paired-deploy qualification.
- Pins: Playwright 1.55.1 (`tests/integration/browser/bun.lock`);
  Chromium 140.0.7339.186; WebKit 26.0; Firefox pin assigned by C01
  (accepted by `invoice-contract.mjs`, absent from the gate5 matrix).
- Credential env names: none (no secrets; runners are local).
- Assigned resources: separate Playwright profiles/ports per parallel
  run (C01 records the Firefox runner when provisioned).
- Evidence IDs: none recorded.
- Capacity: parallel-safe via separate profiles/ports.

## AI-CAP — capped AI evaluation access

- Status: **BLOCKED** — no provider credentials and no spend approval yet.
  Exact ask recorded in [ai-eval-access.md](ai-eval-access.md).
- Owner: H (provisioning task H04).
- Consumers: H08 (live evaluation gate; H04 spend caps stay separate
  from the H07 tenant token budget).
- Pins: provider/model selection assigned by H04.
- Credential env names: unassigned — H04 assigns env names (values never
  recorded in artifacts or evidence files).
- Assigned resources: none yet. H04 records capped run authorization
  and accounting availability; no paid evaluation without its required
  spend approval.
- Evidence IDs: none recorded.
- Capacity: H serializes live-model runs.

## X86-WINDOW — native x86 UP25 access

- Status: **BLOCKED** — no machine access or exclusive window arranged yet.
  Exact ask recorded in [x86-window.md](x86-window.md).
- Owner: H (provisioning task H05).
- Consumers: H12 (native x86 UP25 qualification).
- Pins: Debian 13+ amd64/glibc host; target
  `bun-1.4.2-linux-amd64-v1`
  ([target-linux-amd64.json](target-linux-amd64.json),
  [provenance](linux/provenance.json)); Go 1.27.1 toolchain; pinned
  qualification tools recorded by H05.
- Credential env names: `CAN_BUN_ARCHIVE` (pinned bun-linux-x64.zip),
  `CAN_LINUX_INSTALL_ROOT` and `DATABASE_URL` (PG roundtrip leg).
- Assigned resources: none yet. H05 arranges exclusive access to the
  user's Debian 13+ amd64/glibc machine and matching tool pins.
- Evidence IDs: none recorded.
- Capacity: single queue — H schedules exclusive windows. No Docker
  `linux/amd64`, no QEMU/Rosetta-Linux emulation, no long Air
  saturation without asking.

## Gate summary

| Provision | Status  | Unblocking task |
|-----------|---------|-----------------|
| DB-PG     | READY   | H02 (done)      |
| DB-MySQL  | READY   | H02 (done)      |
| S3        | READY*  | H03 (done; S3-protocol scope) |
| BROWSERS  | PARTIAL | C01 (Firefox)   |
| AI-CAP    | BLOCKED | H04             |
| X86-WINDOW| BLOCKED | H05             |

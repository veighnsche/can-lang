# Local live provisioning (H02 databases, H03 object storage)

Operator record for the loopback-bound disposable services on this MacBook Air.
Credential values never appear here — env names only. Runtime secrets live in
`$CAN_PROVISION_DIR` (default `$HOME/.can-provision`, mode 700/600) and in
process env. All images are native `linux/arm64`; no emulation anywhere.

Tooling: [provision-local.sh](provision-local.sh),
[provision-s3-bucket.mjs](provision-s3-bucket.mjs). Both are idempotent.

## H02 — databases (READY)

| Item | PostgreSQL | MySQL |
|---|---|---|
| Image (digest-pinned) | `postgres:17@sha256:f4c66…91232` | `mysql:8.4@sha256:0744…93fb8d` |
| Observed version | 17.11 (Debian 17.11-1.pgdg13+2), aarch64 | 8.4.11 |
| Container | `can-pg17` | `can-mysql84` |
| Endpoint | `127.0.0.1:55433` | `127.0.0.1:3307` |
| Credential env | `CAN_TEST_POSTGRES_URL` (per-database URL) | `CAN_TEST_MYSQL_URL` (per-database URL); `CAN_TEST_MYSQL_CONTAINER` (kill/restart legs) |
| Namespaces | `can_e02`, `can_f02`, `can_f04`, `can_f06` + per-run `can_<lane>_run<k>` via `db mkdb` | `can_b1_03` (bun live legs), `can_f02`, `can_f04`, `can_f06` + per-run via `db mkdb` |

Notes:

- The MySQL URL shape (port 3307, database `can_b1_03`, app password) is
  fixed by the committed live harness (`runtime/test/mysql.test.ts` lookup);
  the provision script encodes exactly that shape.
- PG has no URL-shape constraint: Go integration drivers take any disposable
  per-run database URL.
- Validation (2026-09-26): real `psql` connection + version pin match;
  per-run isolation roundtrip (two probe DBs, table visible in one only,
  both dropped); `bun test runtime/test/mysql.test.ts` 12/12;
  `bun test runtime/test/mysql-tx.test.ts` 5/5 including the docker
  kill/restart recovery leg.

## H03 — object storage (READY, S3-protocol scope)

- Service: MinIO `RELEASE.2025-09-07T16-13-09Z`, image
  `quay.io/minio/minio@sha256:14ce…bd8936e`, container `can-minio`,
  endpoint `http://127.0.0.1:9000`, region `us-east-1`.
- Bucket: `can-b1-10` (isolated; per-run prefixes owned by the E harness:
  `s3t/<ts>/` in bun legs, `s3i/<nano>-<pid>/` in Go legs).
- Credential env: `CAN_TEST_S3_ENDPOINT`, `CAN_TEST_S3_REGION`,
  `CAN_TEST_S3_BUCKET`, `CAN_TEST_S3_ACCESS_KEY`, `CAN_TEST_S3_SECRET_KEY`.
- Scope honesty: MinIO is a real S3-wire-protocol server, not an in-process
  fake — but it is not AWS S3. Qualification here covers S3-protocol
  behavior. If lane E specifies a leg needing AWS-real server semantics, a
  user-supplied bucket/credentials become required (no standing ask: none
  specified yet).
- Cleanup ownership: harness legs delete their own keys (best effort);
  the bucket is disposable and H-owned.
- Validation (2026-09-26): SigV4 PUT-bucket + PUT/GET/DELETE roundtrip;
  `bun test runtime/test/s3.test.ts` 19/19 live, including multipart,
  abandon-never-materializes, cancel-on-reader-fail, and timeout legs.

## Operator runbook

```sh
distribution/provision-local.sh db up   # start PG + MySQL, ensure DBs/users
distribution/provision-local.sh s3 up   # start MinIO, ensure bucket
eval "$(distribution/provision-local.sh db exports)"  # operator shell only
eval "$(distribution/provision-local.sh s3 exports)"  # operator shell only
distribution/provision-local.sh db mkdb can_f02_run7        # per-run isolation
distribution/provision-local.sh db mkdb can_e02_run3 pg     # pg only
distribution/provision-local.sh db down # stop (volumes kept)
distribution/provision-local.sh s3 down # stop (volume kept)
```

Containers use `--restart unless-stopped` and named volumes, so provisioned
state survives reboots; re-running `up` after a `down` resumes service.

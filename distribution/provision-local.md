# Local live provisioning (H02 databases, H03 object storage, C01 browser runner)

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

## C01 — container Firefox runner (READY)

Native Firefox is impossible on this macOS 27 box: Firefox 141 launches
but its juggler pipe never connects (sandbox-extension EPERM plus
RenderCompositorSWGL framebuffer failure, unaffected by sandbox flags,
TMPDIR, or headed mode), and Firefox 155 dies instantly because the
kernel denies creation of exactly `~/Library/Application
Support/Firefox` (any case; even root; even a fresh user). Chromium and
WebKit stay native; only Firefox runs in a container.

- Service: Playwright 1.55.1 `firefox.launchServer` serving one Firefox
  141.0 (build v1490) over a tokened websocket.
- Image (digest-pinned):
  `mcr.microsoft.com/playwright:v1.55.1-noble@sha256:2f29…03ad1c`,
  native `linux/arm64`, container `can-ff`, loopback
  `ws://127.0.0.1:18783/<token>`.
- Credential env: `CAN_FIREFOX_WS` (full ws endpoint including the
  per-start token; process env only — re-eval `browser exports` after
  every `browser up`, since each server start mints a fresh token).
- Optional env: `CAN_FIREFOX_HOST_ALIAS` (container-to-Mac loopback
  alias the firefox legs use; default `host.docker.internal`).
- Mechanics: the container's main process is `sleep infinity`; `browser
  up` starts the container, installs the pinned `playwright@1.55.1`
  client plus the launchServer entry into the persisted `can-ff-srv`
  volume exactly once, then ensures the ws server is running and
  reachable from the host. `--restart unless-stopped` keeps the
  container across reboots; re-running `up` after a `down` restarts the
  server (fresh token) and resumes service.
- Consumers: gate5 firefox legs (`grid`, `conformance`, `empty`,
  `invoice` on ports 18651–18654) connect via `CAN_FIREFOX_WS` when set
  and launch natively otherwise (CI macos-15 path). Each leg opens a
  fresh browser context, so legs stay isolated on the shared server.
- Validation (2026-09-26): `browser up` idempotent + `down`/`up`
  restart resumes with a fresh token; ws connect reports Firefox 141.0;
  container Firefox loads a 127.0.0.1-bound Mac server through the host
  alias with 200 + title. Gate5 matrix legs land with the C01 harness
  wiring (see evidence index).

## Operator runbook

```sh
distribution/provision-local.sh db up   # start PG + MySQL, ensure DBs/users
distribution/provision-local.sh s3 up   # start MinIO, ensure bucket
distribution/provision-local.sh browser up  # start the container Firefox ws server
eval "$(distribution/provision-local.sh db exports)"  # operator shell only
eval "$(distribution/provision-local.sh s3 exports)"  # operator shell only
eval "$(distribution/provision-local.sh browser exports)"  # operator shell only
distribution/provision-local.sh db mkdb can_f02_run7        # per-run isolation
distribution/provision-local.sh db mkdb can_e02_run3 pg     # pg only
distribution/provision-local.sh db down # stop (volumes kept)
distribution/provision-local.sh s3 down # stop (volume kept)
distribution/provision-local.sh browser down # stop (volume kept)
```

Containers use `--restart unless-stopped` and named volumes, so provisioned
state survives reboots; re-running `up` after a `down` resumes service.

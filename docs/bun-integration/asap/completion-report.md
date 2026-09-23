# ASAP B1 completion report (2026-09-23)

All thirteen Bun-to-Can capabilities B1-01 through B1-13 are
implemented, verified and committed on pinned Bun 1.4.2. The
machine-readable queue shows every capability task plus ASAP-FINAL
done; this report reconciles the thirteen contracts and records the
final suite evidence with no hidden skips.

## Target and services

- Bun: 1.4.2, archive `bun-darwin-aarch64.zip` size 25377591,
  sha256 `90987a3a…be1d12f` (`distribution/target.json`,
  target `bun-1.4.2-darwin-arm64-v1`).
- Catalogue: 34 packages, 87 types, 102 errors, 254 operations,
  10 native modes (`make catalogue-check` clean).
- Live services: MinIO (`can-b1-10-minio`, 127.0.0.1:9000, bucket
  `can-b1-10`) and MySQL 8.4 (`can-b1-03-mysql`,
  127.0.0.1:3307, database `can_b1_03`).

## Final suites (this report)

| Suite | Result |
|---|---|
| `bun test runtime/test/` (service env set) | 530 pass / 0 fail, 69 files |
| `go test ./compiler/...` | all packages ok |
| `go test ./tests/integration/` (archive + services) | ok, 723s |
| `go test ./...` (archive + services) | exit 0: 17 packages ok, 0 failures |
| Fresh-emit strict `tsc` gate (all maintained examples) | exit 0 |
| `make catalogue-check` | clean |
| `go run ./tools/modcheck` | 85 maintained sources ok |

Service legs ran, not skipped: S3 and MySQL integration legs execute
against the containers above (targeted re-runs: S3+MySQL 20s,
Markdown 8s; skips return instantly). The tsc tree includes the new
`examples-markdown` emit, which imports the markdown runtime.

## Per-capability evidence

| Capability | Contract | Tests | Notes |
|---|---|---|---|
| B1-01 files/paths/glob | `b1-01/contract.md` | `files.test.ts`, `files/` example | `cd38d53` |
| B1-02 SQLite + SQL boundary | `b1-02/contract.md`, G-SQL gate | sqlite/MySQL integration, `sql/` fixtures, `sqlite` example | dialect-aware descriptors; stale emit needles repaired in `9eeed7e` (red since `173ba9e`) |
| B1-03 MySQL | `b1-03/contract.md` | live persistence, `mysql` example | transactions, grammar backend |
| B1-04 bounded processes | `b1-04/contract.md` | process tests + `process` example | `214b1e3` |
| B1-05 streams/lifecycles | `b1-05/contract.md`, G-EVENT pull | pull-reader suites, `stream` example | shared-event pull, no new grammar |
| B1-06 HTTP + bodies | `b1-06/contract.md` (48 rows) | http-request/path/transport suites, `http/{main,server,tls}.can` | 12 commits; true incremental multipart out of scope (buffered `formData` path only) |
| B1-07 WebSockets | `b1-07/contract.md` (48 rows), Jev audit | loopback lifecycle, `ws/socket.can`, `websocket` example | reader surface; .07/.08 vacuous (primitive lost, pub/sub excluded) |
| B1-08 crypto/passwords | `b1-08/contract.md` | crypto suite + example | 4 commits |
| B1-09 cookies/CSRF | `b1-09/contract.md` (35 rows), Jev audit | loopback login, `cookies/session.can`, `cookies` example | first-wins lookup; false-vs-config distinction |
| B1-10 S3 | `b1-10/contract.md` (55 rows), Jev audit | staged MinIO lifecycle, `s3/objects.can` | bound presigned method; no hand-rolled signing |
| B1-11 document formats | `b1-11/contract.md`, Jev audit | formats suite, staged CLI, file-backed consume | shared projector; JSONL locations are record-index paths (byte offsets unrepresentable — recorded gap); YAML/JSON5 last-wins documented |
| B1-12 Markdown + G-HTML | `b1-12/contract.md`, two Jev audits | 10-test safe corpus, live CLI + negatives, `markdown` example | strict links, soft breaks, full list/table fidelity; `render_with` deferred (sync/async boundary) |
| B1-13 common utilities | `b1-13/contract.md` | utilities fixtures + example | 4 commits |

Contracts live under `docs/implementation/evidence/2026-09-23/b1-NN/`;
consultation audits under
`docs/bun-integration/asap/evidence/consultations*/`; exit records in
[decisions.md](decisions.md); queue in [execution-queue.json](execution-queue.json).

## Repairs folded into the final run

- `test(b1-02)`: three integration emit needles updated to the
  dialect-first descriptor shape (missed by `173ba9e`).
- `test(b1-12)`: catalogue operation count 252 → 254, trivia
  round-trip inventory 54 → 55, `b1-12/contract.md` evidence dir.
- No capability code changed for these; all are expectation
  maintenance against intended behavior.

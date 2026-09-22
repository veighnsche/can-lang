# Preparation evidence

Observed 2026-09-22. These are standalone native probes and read-only code observations, not proof that the proposed Can APIs are implemented. All preparation changes stay in `docs/bun-integration/`; no compiler/runtime/spec/LF implementation files were edited and no message was sent to the current implementer.

## Native target and reproducibility

The binary matches the pinned `distribution/target.json`: Bun **1.4.2**, revision `744846f844374847c902b5e7fd59b4342a51ef99`, darwin-arm64, SHA-256 `35d20dd0263e5c950194434b925454fdfa9ba6e4467da960410fa05b08a7a5b5`.

The existing isolated qualification bundle supplied the executable; this task did not download, build or modify Bun. [Source observation](source-observation.json) stores its location and per-file hashes. The checkout was concurrently changing, so those hashes do not describe an atomic source snapshot. [Run metadata](probe-run-metadata.json) records final probe/result hashes.

Run with an explicitly selected executable whose hash matches the target:

```sh
python3 docs/bun-integration/asap/evidence/run-probes.py --bun /absolute/path/to/qualified/runtime/bun
python3 docs/bun-integration/asap/evidence/validate.py
```

The runner rejects another binary, uses a temporary working directory and minimal environment, disables package installation, captures outputs and applies host deadlines of 30 and 20 seconds. The probes create/remove their own temporary trees and use only loopback network endpoints. They do not connect to MySQL, PostgreSQL, S3 or user services. Temporary test credentials are hardcoded nonproduction literals; no secret environment is passed to children.

## Observed results

| Area | Native observation | Consequence |
|---|---|---|
| API availability | SQL, password, CSRF, Cookie/CookieMap, S3Client, TOML/YAML/JSON5/JSONL and Markdown symbols exist. | Presence is not complete capability qualification. |
| Files | Write, rename, read, directory listing and glob returned the expected temporary file. | Native filesystem route exists; limits, symlinks and ownership still require Can tests. |
| SQLite Bun.SQL | `bigint:true` alone returned 9007199254740992 for 9007199254740993n. | Do not reuse PostgreSQL options without dialect-specific tests. |
| SQLite exact option | Adding `safeIntegers:true` returned the exact bigint. Direct bun:sqlite safeIntegers was exact too. | Prefer qualified shared API; do not prematurely introduce a second runtime backend. |
| SQLite lifecycle | A transaction committed before/after an await, a thrown-error transaction rolled back, and a new file connection recovered committed exact values. | These native behaviors were verified after consultation; Can ownership/commit-failure/locking still need tests. |
| Processes | Literal `$(echo unintended)` arrived as an argument; stdout/stderr and exit 7 were captured. | Array arguments avoid shell reinterpretation in this case. No process-tree proof is claimed. |
| Streams | Cancel invoked native cancel and releasing the reader removed the lock. | Reuse native lifecycle hooks; pending-read and race tests remain. |
| Password | Correct password verified true; wrong password false. | Native password route verified, not all malformed-hash/cost cases. |
| WebCrypto | AES-GCM round-trip worked with a nonextractable key. | Initial key/encryption path exists; signature suite still unqualified. |
| Cookies/CSRF | Cookie attributes serialized; token accepted matching session and rejected another session. | Keep explicit session binding and test full attribute/expiry policy later. |
| TOML | Large integer raised a lossless-representation SyntaxError. | Document the native rejection; do not assume every parser rounds the same way. |
| YAML/JSON5/JSONL | Large integer became 9007199254740992. | Projection cannot recover original tokens; unsafe parsed integers must be rejected for Can int. |
| JSONL malformed suffix | parse returned valid prefix; parseChunk returned done=false and a SyntaxError. | Full-input checks are required. The final saved result retains error name/message explicitly. |
| Markdown | Default output retained script and javascript: URL; disabling raw spans escaped a simple tag. | Neither default rendering nor raw-HTML suppression alone proves safe HTML. |
| URL/text/date | Native relative URL, repeated query keys, UTF-8 encoding and epoch ISO conversion worked. | Fill missing Can APIs around native operations. |
| HTTP/WebSocket | Loopback streaming Response concatenated to onetwo; client/server echoed a text message. | Native entry points work; this is not backpressure, TLS or disconnect qualification. |

Read [native results](native-probe-results.json), [probe source](native-probe.ts), [SQLite lifecycle results](sqlite-lifecycle-results.json) and its [source](sqlite-lifecycle-probe.ts). Errors intentionally elicited by probes are observations, not tool failures or failed Can tests. The validator checks these expected observations explicitly.

No live MySQL/S3 acceptance, full SQL dialect compiler, complete Can event examples, safe Markdown renderer or production adapter was implemented during preparation.

## Primary sources read

Official documentation was checked on the preparation date; it is live and can change independently of the pinned binary. Runtime experiments decide claims about the actual target. Capability plans link their specific sources.

- [Bun API inventory](https://bun.sh/docs/runtime/bun-apis), [file I/O](https://bun.sh/docs/runtime/file-io), [glob](https://bun.sh/docs/runtime/glob).
- [SQL](https://bun.sh/docs/runtime/sql), [SQLite](https://bun.sh/docs/runtime/sqlite): unified dialect API and SQLite-specific options.
- [Processes](https://bun.sh/docs/runtime/child-process), [streams](https://bun.sh/docs/runtime/streams), [HTTP server](https://bun.sh/docs/runtime/http/server), [WebSockets](https://bun.sh/docs/runtime/http/websockets).
- [Hashing/passwords](https://bun.sh/docs/runtime/hashing), [cookies](https://bun.sh/docs/runtime/cookies), [CSRF](https://bun.sh/docs/runtime/csrf), [S3](https://bun.sh/docs/runtime/s3).
- [TOML](https://bun.sh/docs/runtime/toml), [YAML](https://bun.sh/docs/runtime/yaml), [JSON5](https://bun.sh/docs/runtime/json5), [JSONL](https://bun.sh/docs/runtime/jsonl), [Markdown](https://bun.sh/docs/runtime/markdown).
- [TypeSafe index](https://docs.typesafe.ai/llms.txt), [API](https://docs.typesafe.ai/api.md), [Choice](https://docs.typesafe.ai/primitives/choice.md). Browser fetch failed for TypeSafe; direct HTTPS retrieval succeeded and the live request contract was read.

## Consultations

Three fresh requests assessed four independent decisions. Read the [pre-dispatch semantic/wording audit](consultations/pre-dispatch-audit.md), requests/responses 1–3 and their metadata. The [decision ledger](../decisions.md) gives outcomes, probability variation and limitations. No classifier-generated rationale is invented: Jev returned choices, confidence and distributions, not prose explanations.

Native SQLite transaction/reopen evidence was gathered after these requests. Requests correctly describe what was known when sent; they were not edited afterward to claim later evidence was supplied.

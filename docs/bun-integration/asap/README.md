# Bun → Can ASAP implementation handoff

Prepared 2026-09-22. This directory turns all **13 ASAP capabilities** into an implementation handoff with specific files, proposed contracts, native lowering, ordered steps, acceptance cases and fixture behavior. It leaves the active LF01–LF21 implementer and shared implementation files untouched.

Start with [agent instructions](agent-prompt.md), follow the [ordered execution queue](execution-queue.md), and apply the [shared integration contract](integration-contract.md). The [decision ledger](decisions.md) distinguishes settled recommendations from bounded gates. [Evidence](evidence/README.md) includes actual pinned-Bun probes and three fresh Jev consultations.

**Commit your own work regularly:** deliver small, verified commits containing only your B1 changes. Preserve other agents' changes and isolate overlapping work when needed. See the [commit policy](integration-contract.md#commit-policy); earlier B1 prompts prohibiting commits are superseded.

## Readiness

**Not all thirteen designs are closed or free of blockers.** The [readiness audit](readiness.md) lists every capability, what can start, known blockers, unresolved scope and required proof. SQL grammar, event syntax, streaming multipart, process-tree cleanup, safe Markdown and real-service qualification need particular attention.

## File boundaries

Follow the [required filetree](filetree.md): feature-owned Go and runtime modules, an emitter extraction prerequisite, protected orchestration hubs and an executable changed-file size guard. The per-capability tables below identify integration areas; they do not permit appending every feature to existing large files.

## Capability plans

| Task | Detailed plan | Primary implementation area |
|---|---|---|
| B1-01 | [Files, directories, paths and globbing](b1-01-files.md) | `runtime/platform/files.ts` |
| B1-02 | [SQLite integration and the shared multi-dialect SQL boundary](b1-02-sqlite-and-sql-boundary.md) | `compiler/internal/project/manifest.go` |
| B1-03 | [MySQL integration](b1-03-mysql.md) | `runtime/platform/sql.ts` |
| B1-04 | [Bounded child-process execution](b1-04-processes.md) | `runtime/platform/process.ts` |
| B1-05 | [Streams and reusable lifecycle contracts](b1-05-streams-and-lifecycles.md) | `runtime/transport/stream.ts` |
| B1-06 | [Broader HTTP and incremental request/response bodies](b1-06-http.md) | `runtime/platform/http.ts` |
| B1-07 | [WebSocket client and server integration](b1-07-websockets.md) | `runtime/platform/websocket.ts` |
| B1-08 | [Password and broader cryptographic operations](b1-08-crypto.md) | `runtime/platform/crypto.ts` |
| B1-09 | [Cookies and CSRF primitives](b1-09-cookies-csrf.md) | `runtime/platform/cookies.ts` |
| B1-10 | [S3-compatible object storage](b1-10-s3.md) | `runtime/platform/s3.ts` |
| B1-11 | [Typed TOML, YAML, JSON5 and JSONL codecs](b1-11-document-formats.md) | `runtime/codec/formats.ts` |
| B1-12 | [Markdown rendering and structured processing](b1-12-markdown.md) | `runtime/platform/markdown.ts` |
| B1-13 | [Common URL, text, byte and time utilities](b1-13-common-utilities.md) | `runtime/platform/url.ts` |

## What preparation established

- Verified the existing pinned Bun 1.4.2 binary SHA against `distribution/target.json`; no runtime was downloaded or changed.
- Exercised native files/globs, SQLite, argument-array processes, stream cancellation, password verification, AES-GCM, cookie/CSRF, Markdown, URL/text/date and loopback HTTP/WebSocket behavior.
- Found and reproduced SQLite's `bigint:true` precision problem; `safeIntegers:true` preserves the tested exact integer. A subsequent native probe verified awaited SQLite transactions, rollback and file reopening.
- Reproduced unsafe integer conversion in YAML/JSON5/JSONL, TOML's rejection, JSONL prefix acceptance and unsafe default Markdown HTML.
- Consulted Jev three times with independently rewritten prose on backend preference, event sequencing, format guarantees and HTML trust. Results are advice, not acceptance proof.

## What is intentionally still an implementation gate

SQL dialect grammar admission needs a concrete backend/corpus decision. Event primitives need complete Can program comparisons. Markdown safe rendering needs proof that native callback boundaries satisfy Can's existing safe HTML contract. These have named owners, outputs and exit conditions; they do not block unrelated capability work.

MySQL and S3 need real local service qualification. The preparation does not claim those services were tested. Native samples do not prove Can integration, resource escape analysis or complete runtime conformance.

All proposed APIs are interface sketches, not newly approved grammar. All implementation tasks remain pending. The three-tier parent roadmap still contains 36 tasks; this handoff expands only its 13 ASAP tasks and does not narrow or replace them.

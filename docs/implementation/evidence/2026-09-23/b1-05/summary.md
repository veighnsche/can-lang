# B1-05 evidence: pull streams and reusable lifecycle contracts

Date: 2026-09-23. Pinned target: bun-1.4.2-darwin-arm64-v1 (revision
744846f844374847c902b5e7fd59b4342a51ef99).

## Decision

G-EVENT compared nine complete programs (file/JSONL scan, process
drain with deadline, socket session) across shared-library,
per-domain and shared-event-pull surfaces; see
`docs/bun-integration/asap/evidence/b1-05-comparison/README.md`. All
nine parse, so no surface needs new grammar. Three fresh Jev rounds
preferred the library surface 3/3; the decision audit in
`docs/bun-integration/asap/evidence/consultations-b1-05/decision-audit.md`
overrides on contract rows the packets did not carry (B1-06.02
incremental reuse, B1-05.07 same-path fixtures, caller-visible
acceptance rows, library+fifo incoherence) and selects pull with
deferred consume sugar, drain-by-demand and repeated-selector FIFO
transcripts. Exit recorded in the G-EVENT row of
`docs/bun-integration/asap/decisions.md`.

## Contract

Catalogue package `stream` (6 operations, 4 errors 1316-1319, opaque
`stream::reader<T>` and `stream::writer`) plus 3 file producers on
`files`. Full operation/error table and lifecycle rules live in
`contract.md` beside this file and the generated catalogue mirror.
Reader generics resolve per call site through a stream specialization
hook that mirrors the collections machinery; handles are
harness-supplied scope values in assertions, like pool/transaction
handles. Catalogue additions only: no parser/checker/IR grammar files.

## Native qualification (pinned binary)

- `ReadableStream` pull is demand-driven: zero pulls before
  `getReader`, pulls matching reads (`stream_pull`).
- Cancel settles a pending read as done with no value and releases the
  lock (`stream_abort`); the adapter maps interrupted reads onto
  `cancelled` and pre-cancels native waits because owner close holds
  for live leases.
- Chunk objects are distinct per read; `.slice()` detaches a boundary
  copy; streaming `TextDecoder` reassembles split UTF-8 (`stream_alias`).
- `FileSink.write` reports accepted bytes but write-after-end is
  silent, so ended-state enforcement is adapter-side via owner close
  (`stream_filesink`).
- Child pipe reads are not message-aligned: two writes arrived as one
  chunk (`stream_process_reader`); producers frame explicitly.

## Verification

- `runtime/test/streams.test.ts`: 11 owned-stream cases (order/end,
  line framing incl. forced split UTF-8, caps, cancel race, close
  twice/use-after-close/foreign handles, error/end precedence,
  boundary copies both directions, short writes, close failures,
  acquisition errors, shared drain).
- `tests/integration/streams_test.go`: staged-bundle bun suite plus
  pump transcripts through real dispatch, a live file-stream program,
  transcript overrun/underrun violations and four rejection shapes.
- `examples/stream`: maintained manifest-backed example covered by
  `TestStdlibMaintained`, including cross-assertion transcript reuse.
- `go test ./...`, `make catalogue-check`, `go run ./tools/modcheck`,
  `tests/conformance/native.test.ts`, strict `tsc` over the fresh emit
  tree, and the ASAP layout size guard all pass; see the slice commit
  for identifiers.

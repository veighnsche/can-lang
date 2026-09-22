# B1-02 — SQLite integration and the shared multi-dialect SQL boundary

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: B1-01. Surface: Library.

Deliver SQLite memory/file connections and a genuine dialect-aware SQL contract, preserving PostgreSQL. Prefer Bun.SQL with safeIntegers:true for SQLite, supported by the native probe. bigint:true alone is insufficient on this target. Retain bun:sqlite as a fallback to investigate only if required native behavior fails; the direct synchronous API is not automatically compatible with Can asynchronous transaction bodies.

## Where to implement

**Required layout:** follow [filetree and module boundaries](filetree.md). Its feature modules supersede the coarse existing-file locations below; do not append B1 implementation bodies to existing orchestration hubs.

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `compiler/internal/project/manifest.go` | existing | Admit explicit postgres/mysql/sqlite connection and descriptor dialects. |
| `compiler/internal/sql/parser.go` | existing | Separate the PostgreSQL parser from the dialect-neutral entry point. |
| `compiler/internal/sql/descriptors.go` | existing | Dialect-tagged descriptor validation, parameter segments, cardinality and limits. |
| `compiler/internal/check/sql.go` | existing | Prove descriptor/pool dialect agreement and parameter/result contracts. |
| `compiler/internal/ir/sql.go` | existing | Carry checked dialect and descriptor evidence. |
| `compiler/internal/emit/sql.go` | existing | Emit dialect-aware execution from checked evidence. |
| `runtime/platform/sql.ts` | existing | Native connection options and per-dialect value/error mapping. |
| `runtime/platform/sql-descriptor.ts` | existing | Immutable descriptor materialization and native template bindings. |
| `runtime/platform/transaction.ts` | existing | SQLite transaction lifetime, rollback and async body handling. |
| `tests/integration/sql_test.go` | existing | SQLite persistence and PostgreSQL regression tests. |
| `tests/integration/sql_descriptors_test.go` | existing | Dialect grammar, placeholders and source spans. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
connection dialect = postgres | mysql | sqlite
sqlite config = memory | file(path, mode, busy_timeout)
existing typed SQL declarations + explicit descriptor dialect
existing one / optional / many / execute cardinalities
existing transaction body and error propagation
result metadata = dialect-appropriate exact affected_rows / optional insert_id
```

## Implementation sequence

1. Create a three-dialect contract table covering placeholders, null, integers, decimals, booleans, text, blobs, dates, affected rows and insert IDs. Reject unsupported mappings; never silently cast an exact Can integer through Number.
2. Introduce dialect into manifest and checked descriptor identity. Keep PostgreSQL parsing in its backend. Do not feed MySQL backticks or SQLite syntax through libpg_query and call the result dialect validation.
3. Complete the SQL admission gate in decisions.md before declaring frontend support. Compare a pinned maintained dialect parser with native prepare qualification; preserve compile-time syntax guarantees or explicitly reconcile the contract. Runtime-only prepare does not prove offline build validation. A regex/semicolon splitter is forbidden as a substitute parser.
4. Continue static-string/parameter segment binding through native tag templates. No user values in SQL text. Prove repeated and out-of-order parameters, comments/literals and parameter-like text. Preserve row-limit/cardinality contracts per dialect instead of appending PostgreSQL-specific syntax blindly.
5. Open Bun.SQL with explicit sqlite adapter, filename and safeIntegers:true; qualify all numeric conversions. Implement memory lifetime, file open modes and close. File DB tests must reopen with a new connection. Never infer rollback from an in-memory disappearing connection.
6. Implement transaction begin/commit/rollback on the owned connection; a body can await. Test domain failure, standard failure, cancellation and commit failure independently, preserving primary occurrence and cleanup diagnostics.
7. Qualify lock contention, busy timeout and event-loop behavior. Bound connection acquisition and release. If the target cannot interrupt a long native SQLite operation, describe that limitation and require an enforceable execution design before promising hard query deadlines.
8. Complete PostgreSQL plus SQLite evidence, then pass the same descriptor/value tests to B1-03. Do not mark the three-engine milestone done before MySQL.

## Acceptance evidence

- Success: memory and persistent DB, typed CRUD, one/optional/many/execute, NULL mapping, exact ±64-bit edge values, Unicode and blobs, transaction commit/rollback.
- Rejected: pool/descriptor dialect mismatch, missing/extra bind values, multiple statements where one is required, malformed per-dialect grammar, unsupported row shape and loss-prone conversion.
- Runtime: safeIntegers regression at 9007199254740993n; failure after first write really rolls back; await inside transaction; closing pool during leased operation; busy database; failed commit provenance.
- Generated TS: native bindings remain separated from literals; no new SQL engine or interpolated values; target options include safeIntegers:true for SQLite.

## Fixtures and local assertions

Local attached tests own schemas and fixture sequences. Pure row-projection tests use immutable row data; native SQL request fixtures verify descriptor, dialect, parameters and transaction ordering. Real SQLite runs are mandatory and offline. Reusable SQL fixtures do not make an untested caller count as locally tested.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const db = new Bun.SQL({adapter: "sqlite", filename, safeIntegers: true});
const rows = await db`SELECT ${9007199254740993n} AS n`;
// Compiler-owned static template segments + separately supplied values remain bound.
await db.close();
```

## Gates and limitations

The SQL grammar backend remains an explicit implementation gate, not a secretly decided new dependency. A probe verifies one integer, not every type or transaction. Native prepare against a database may depend on schema and cannot casually replace offline compiler parsing. Preserve exact diagnostics and existing approved statement restrictions.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/sql)
- [Official Bun documentation](https://bun.sh/docs/runtime/sqlite)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.

## Additional native evidence gathered after consultation

The [SQLite lifecycle probe](evidence/sqlite-lifecycle-results.json) verified commit across an await, rollback after a deliberate exception and exact committed data after file reopening. These strengthen the Bun.SQL preference; Can ownership, lock contention, failed commit and interruption remain implementation acceptance requirements.

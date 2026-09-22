# B1-03 — MySQL integration

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: B1-02. Surface: Library.

Implement MySQL through Bun.SQL using the shared dialect boundary. This is a required ASAP deliverable, not a later optional adapter. Do not copy PostgreSQL assumptions about placeholders, booleans, datetime, decimal values, insert IDs or error codes.

## Where to implement

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/platform/sql.ts` | existing | MySQL config, mapping and known operational errors. |
| `runtime/platform/transaction.ts` | existing | Native MySQL transaction ownership and rollback. |
| `compiler/internal/project/manifest.go` | existing | MySQL connection configuration and descriptor dialect. |
| `compiler/internal/sql/descriptors.go` | existing | MySQL frontend dispatch and binding evidence. |
| `tests/integration/mysql_test.go` | new | Real service qualification and shared three-engine workload. |
| `tests/integration/testdata/mysql` | new | Local schema, seeds, dialect cases and driver harness. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
mysql connection(host, port, database, credentials, TLS, pool_limits)
SQL typed descriptors declare mysql dialect
query cardinalities retain the common Can contract
execute -> affected_rows plus optional exact insert_id
transaction callback retains its calculated error set
```

## Implementation sequence

1. Reuse B1-02 frontend/value tables; implement MySQL-specific options with explicit TLS and credentials. Never log credential-bearing URLs.
2. Use Bun.SQL native prepared bindings. Distinguish identifiers from values; dynamic identifiers require a separate checked operation, not interpolation into a parameter slot.
3. Qualify BIGINT signed/unsigned extremes and DECIMAL. If the driver emits decimal strings, validate and preserve them through an explicit exact representation; do not coerce to floating point.
4. Define DATETIME without timezone separately from instants; set test session timezone explicitly. Exercise bit/boolean, binary and NULL. Reject unsupported native result objects before exposing data.
5. Classify native errors by verified codes for connection/auth/constraint/deadlock/timeout; preserve private cause. Unknown or adapter-programming errors remain standard failures, not generic mysql errors.
6. Run shared typed CRUD/rollback workload across all three engines and dialect-specific cases against a provisioned local MySQL service. Do not add automatic retries to non-idempotent statements.
7. Add clean setup/teardown, version reporting and test-service configuration. Missing service is a visible qualification blocker, never a passing integration result.

## Acceptance evidence

- Success: exact bindings containing quotes/comment markers, affected rows, generated IDs, nullable fields, transaction commit and rollback, prepared statement reuse.
- Rejected: SQLite/PostgreSQL descriptor on MySQL pool, invalid grammar and unsupported types, unsigned integer outside Can conversion policy.
- Runtime: disconnect mid-query, authentication failure, unique-key conflict, concurrent transactions, deadlock error, rollback failure does not replace original failure.
- No fabricated evidence: service version and execution logs saved; fixture tests alone cannot close this task.

## Fixtures and local assertions

Request fixtures verify dialect, descriptor identity, typed values and transaction boundaries. Real tests use an isolated database name and a dedicated low-privilege test account. Do not discover production credentials. PostgreSQL regression evidence remains separate from MySQL evidence.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const db = new Bun.SQL(mysqlURL, {adapter: "mysql", bigint: true});
// Qualify exact MySQL behavior; option names alone are not evidence of every mapping.
const rows = await db`SELECT ${value} AS value`;
```

## Gates and limitations

No MySQL service was exercised during this preparation. SQL modes, timezone, server version and TLS configuration affect behavior and belong in the qualification record. Bun is the wire driver; no npm MySQL client enters the runtime.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/sql)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.

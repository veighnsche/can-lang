# Native Can migration ledger: sql-history

Status: **proposed replacements; implementation and parity evidence pending**.

[Overview](../../native-can-tests-migration-ledger-2026-09-30.md) · [Machine-readable rows](sql-history.json)

Every deletion condition is conjunctive with the overview's common gate. It covers **all** protected facets, variants and delegated oracles, even where a row's short replacement sentence mentions only its first facet. Row IDs name coverage obligations, not one-to-one implementation files.

## HISTORY-001

**TestCurrentSQLQueries** — [tests/integration/sql_test.go](/Users/vince/Projects/can-lang/tests/integration/sql_test.go:46)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-001`. Evidence: pending.

**Protects**

- Nine supplied SQL query assertion roots pass; optional/one/many/execute descriptors bind correctly.
- Repeated and relocated builds have the same identity; emitted SQL operations and row shapes are present semantically; optional strict TypeScript check passes.
- Four deliberately invalid descriptor variants are rejected: wrong cardinality, missing parameter type, row type mismatch, undeclared descriptor.

**Current observations:** assert report, build IDs, emitted program files, compile failures

**Variants:** negative manifest: cardinality negative manifest: parameter type negative manifest: row type negative manifest: undeclared

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_TSC optional

**Capabilities:** SUITE CAN BUILD DIAG WORK

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself.

## HISTORY-002

**TestCurrentSQLQueriesLive** — [tests/integration/sql_test.go](/Users/vince/Projects/can-lang/tests/integration/sql_test.go:186)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-002`. Evidence: pending.

**Protects**

- Compiled CLI against PostgreSQL 17 returns exact one/optional/many/execute and bytes/nullable rows; row-missing, row-count, row-limit, duplicate, hostile bound input and refused connection remain distinct.
- Credential enters through fd 3 snapshot rather than argv/environment; setup checks six accounts and two cover rows; teardown follows success/failure.

**Current observations:** compiled child exit/stdout/stderr and setup-driver counts

**Variants:** by_id present/missing by_term present/none/ambiguous/hostile rows limited/full add first/duplicate cover bytes and nullable refused port

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_TEST_POSTGRES_URL disposable PostgreSQL 17

**Capabilities:** SUITE CAN PROC CHILD DB WORK ENV

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/queries-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/queries-driver.ts); [tests/integration/testdata/sql/queries.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/queries.sql)

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself.

## HISTORY-003

**TestCurrentSQLTransactions** — [tests/integration/sql_test.go](/Users/vince/Projects/can-lang/tests/integration/sql_test.go:375)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-003`. Evidence: pending.

**Protects**

- Twelve supplied transaction assertion roots pass; emitted transaction/SQL wiring and repeatable relocated build identity hold; optional strict typecheck passes.
- Wrong cardinality, missing parameter type, wrong row type and undeclared descriptor each reject.

**Current observations:** assert report, build IDs, emitted program and negative build diagnostics

**Variants:** four negative manifest mutations

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_TSC optional

**Capabilities:** SUITE CAN BUILD DIAG WORK

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself.

## HISTORY-004

**TestCurrentSQLTransactionsLive** — [tests/integration/sql_test.go](/Users/vince/Projects/can-lang/tests/integration/sql_test.go:518)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-004`. Evidence: pending.

**Protects**

- Separate compiled CLI calls prove commit persistence and rollback absence; sequence IDs 7/8/9/10 distinguish committed, conflicting and rolled-back writes.
- Invalid codec input, nested transaction fallback rollback and refused connection produce the expected failures; credential uses fd 3.

**Current observations:** child exit/stdout/stderr and seed setup report

**Variants:** fetch present/missing commit and duplicate rollback and sequence gap nested transaction invalid input refused port

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_TEST_POSTGRES_URL disposable PostgreSQL 17

**Capabilities:** SUITE CAN PROC CHILD DB WORK

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/transactions-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/transactions-driver.ts); [tests/integration/testdata/sql/transactions.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/transactions.sql)

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself.

## HISTORY-005

**TestCurrentSQLDescriptors** — [tests/integration/sql_descriptors_test.go](/Users/vince/Projects/can-lang/tests/integration/sql_descriptors_test.go:45)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-005`. Evidence: pending.

**Protects**

- Supplied descriptor root and ordinary run pass; repeat/relocated build identity and semantic descriptor emission hold; optional strict generated TypeScript passes.
- Twelve malformed descriptors reject: unbounded select, RETURNING, multiple statements, parameter gap/extra/reused limit, literal limit, field order, non-scalar/unknown type, execute limit and syntax.

**Current observations:** assert/run exit, build IDs, emitted state, diagnostic text

**Variants:** 12 descriptor negative cases

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_TSC optional

**Capabilities:** SUITE CAN BUILD DIAG WORK

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself.

## HISTORY-006

**TestCurrentSQLDescriptorsLive** — [tests/integration/sql_descriptors_test.go](/Users/vince/Projects/can-lang/tests/integration/sql_descriptors_test.go:184)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-006`. Evidence: pending.

**Protects**

- Built descriptor state is exercised against PostgreSQL 17 with a neutral fd-3 snapshot and four-statement create/truncate/insert/drop fixture; a missing or failing check cannot pass.
- Template expansion repeats a bound $1, ignores $9 inside a SQL comment, and keeps quoted, Unicode and hostile strings as values; stacked injection does not drop the table.
- Search, repeated binding, by-id present/missing, insert count and readback, comment literal and bound LIMIT produce their specified rows.

**Current observations:** driver checks array with ok/detail, server version

**Variants:** repeat/comment binding like, quote, Unicode OR and stacked injection with survivor count by-id present/missing insert roundtrip comment literal bound limit

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_TEST_POSTGRES_URL disposable PostgreSQL 17

**Capabilities:** SUITE CAN DB NATIVE WORK

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/descriptors-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/descriptors-driver.ts); [tests/integration/testdata/sql/descriptors.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/descriptors.sql)

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself.

## HISTORY-007

**TestF02LiveLockingAndReturning** — [tests/integration/sql_f02_test.go](/Users/vince/Projects/can-lang/tests/integration/sql_f02_test.go:21)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-007`. Evidence: pending.

**Protects**

- PostgreSQL SKIP LOCKED/NOWAIT/ordinary and NO KEY locking semantics under contention; 55P03 classification inside and outside transaction.
- Eight identical concurrent inserts are unidentifiable by refetch while raw RETURNING identifies one row; 16 keyed writers refetch own row.
- MySQL optional leg rejects INSERT RETURNING; cost timing is recorded, not thresholded.

**Current observations:** raw and runtime driver report plus Go verdicts

**Variants:** PG locking modes ambiguous 8 writers keyed 16 writers raw RETURNING optional MySQL timing evidence only

**Environment/selection gates:** CAN_TEST_POSTGRES_URL bun on PATH CAN_TEST_MYSQL_URL optional

**Capabilities:** SUITE CAN DB NATIVE WORK CLOCK METRIC

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/f02-live-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/f02-live-driver.ts)

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself. F02 timing is descriptive evidence only; do not turn a cost threshold into an unrequested performance gate.

## HISTORY-008

**TestF03LiveRelationalSlice** — [tests/integration/sql_f03_test.go](/Users/vince/Projects/can-lang/tests/integration/sql_f03_test.go:24)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-008`. Evidence: pending.

**Protects**

- Raw PostgreSQL RETURNING single/multi/conflict rows and bigint identity; runtime SQL error vocabulary for duplicate/missing/multiple/wide rows.
- Poisoned transaction reports 25P02 and subsequent replay works; eight lookup-first writers settle with one winner and no poisoned connection.
- Nullable audit, money minor units, epoch milliseconds, JSON and int64 edges round-trip; SQLite RETURNING rows; optional MySQL RETURNING rejection and LAST_INSERT_ID writer ownership.

**Current observations:** raw native driver report plus Go field verdicts

**Variants:** PG RETURNING error vocabulary aborted transaction/replay eight-writer race encoding edges SQLite optional MySQL

**Environment/selection gates:** CAN_TEST_POSTGRES_URL bun on PATH CAN_TEST_MYSQL_URL optional

**Capabilities:** SUITE CAN DB NATIVE WORK

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/f03-live-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/f03-live-driver.ts)

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself.

## HISTORY-009

**TestCurrentMySQLPersistence** — [tests/integration/mysql_test.go](/Users/vince/Projects/can-lang/tests/integration/mysql_test.go:43)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-009`. Evidence: pending.

**Protects**

- Offline Can assertions and built production CLI run against disposable MySQL through fd-3 credential.
- Separate processes prove persisted insert/read, missing/duplicate classification, commit/abort visibility and invalid input rejection.

**Current observations:** setup server version/empty notes, child exit/stdout/stderr

**Variants:** ping add/read missing/duplicate commit/abort invalid codec

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_TEST_MYSQL_URL

**Capabilities:** SUITE CAN PROC CHILD DB WORK

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/mysql-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/mysql-driver.ts); [tests/integration/testdata/sql/mysql-seed.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/mysql-seed.sql)

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself.

## HISTORY-010

**TestCurrentSQLitePersistence** — [tests/integration/sqlite_test.go](/Users/vince/Projects/can-lang/tests/integration/sqlite_test.go:43)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-010`. Evidence: pending.

**Protects**

- Offline Can assertions and built production CLI run against an owned SQLite file with fd-3 empty snapshot.
- Separate processes prove file persistence, missing/duplicate classification, commit/abort visibility and invalid input; in-memory check succeeds.

**Current observations:** setup library version/empty notes, child exit/stdout/stderr

**Variants:** memory add/read missing/duplicate commit/abort invalid codec

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN PROC CHILD DB WORK

**Proposed Can replacement:** Can case stages the source and manifest, checks offline assertions/negative builds when present, runs the production candidate or native DB observation with ordinary Can helpers, and compares raw facts in Can; replace textual emitted-layout pins with behavior/provenance checks.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/sqlite-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/sqlite-driver.ts); [tests/integration/testdata/sql/sqlite-seed.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/sqlite-seed.sql)

**Notes:** Proposed only. Where the old Bun driver checks outcomes, move both sequence and oracle into Can; retain independent raw SQL observation rather than checking the Can SQL adapter against itself.

## HISTORY-011

**descriptors-driver.ts** — [tests/integration/testdata/sql/descriptors-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/descriptors-driver.ts:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-011`. Evidence: pending.

**Protects**

- Applies four-statement PostgreSQL setup and drop fixture; imports emitted descriptor state, expands repeated placeholders while ignoring comment text, and executes the exact search, hostile-input, insert and limit observations.
- Current driver embeds named expected rows, counts and pass/fail decisions in TypeScript; these must become Can-owned.

**Current observations:** native SQL rows, errors, version and script-generated JSON

**Variants:** None recorded.

**Environment/selection gates:** disposable database; see related Go case gates

**Capabilities:** SUITE DB NATIVE WORK

**Proposed Can replacement:** Can case selects and applies fixture, sequences native/raw observations, checks server/version/seed and expected rows in Can; native adapter may return SQL facts and errors only.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/descriptors.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/descriptors.sql)

**Notes:** Independent raw-driver channel is required for observations the production Can SQL adapter cannot independently verify.

## HISTORY-012

**f02-live-driver.ts** — [tests/integration/testdata/sql/f02-live-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/f02-live-driver.ts:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-012`. Evidence: pending.

**Protects**

- Builds runtime-style descriptor table, controls PG lock contention, RETURNING/refetch races and optional MySQL; computes checks and timing report.
- Current TypeScript script contains scenario-specific fixture shape/version checks or expected-result decisions.

**Current observations:** native SQL rows, errors, version and script-generated JSON

**Variants:** None recorded.

**Environment/selection gates:** disposable database; see related Go case gates

**Capabilities:** SUITE DB NATIVE WORK CLOCK METRIC

**Proposed Can replacement:** Can case selects and applies fixture, sequences native/raw observations, checks server/version/seed and expected rows in Can; native adapter may return SQL facts and errors only.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Independent raw-driver channel is required for observations the production Can SQL adapter cannot independently verify. F02 timing is descriptive evidence only; do not turn a cost threshold into an unrequested performance gate.

## HISTORY-013

**f03-live-driver.ts** — [tests/integration/testdata/sql/f03-live-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/f03-live-driver.ts:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-013`. Evidence: pending.

**Protects**

- Builds runtime-style descriptor table; executes raw PG/SQLite/MySQL RETURNING, error vocabulary, abort/replay, lookup-first race and encoding checks.
- Current TypeScript script contains scenario-specific fixture shape/version checks or expected-result decisions.

**Current observations:** native SQL rows, errors, version and script-generated JSON

**Variants:** PG raw RETURNING runtime SQL error vocabulary abort/replay eight-writer lookup race blessed encoding roundtrips optional MySQL LAST_INSERT_ID and RETURNING rejection SQLite raw RETURNING

**Environment/selection gates:** disposable database; see related Go case gates

**Capabilities:** SUITE DB NATIVE WORK

**Proposed Can replacement:** Can case selects and applies fixture, sequences native/raw observations, checks server/version/seed and expected rows in Can; native adapter may return SQL facts and errors only.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Independent raw-driver channel is required for observations the production Can SQL adapter cannot independently verify.

## HISTORY-014

**mysql-driver.ts** — [tests/integration/testdata/sql/mysql-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/mysql-driver.ts:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-014`. Evidence: pending.

**Protects**

- Applies two-statement MySQL seed and checks empty notes/server version.
- Current TypeScript script contains scenario-specific fixture shape/version checks or expected-result decisions.

**Current observations:** native SQL rows, errors, version and script-generated JSON

**Variants:** None recorded.

**Environment/selection gates:** disposable database; see related Go case gates

**Capabilities:** SUITE DB NATIVE WORK

**Proposed Can replacement:** Can case selects and applies fixture, sequences native/raw observations, checks server/version/seed and expected rows in Can; native adapter may return SQL facts and errors only.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Independent raw-driver channel is required for observations the production Can SQL adapter cannot independently verify.

## HISTORY-015

**queries-driver.ts** — [tests/integration/testdata/sql/queries-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/queries-driver.ts:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-015`. Evidence: pending.

**Protects**

- Applies/tears down ten-statement PG query seed and checks PG 17 and row counts.
- Current TypeScript script contains scenario-specific fixture shape/version checks or expected-result decisions.

**Current observations:** native SQL rows, errors, version and script-generated JSON

**Variants:** None recorded.

**Environment/selection gates:** disposable database; see related Go case gates

**Capabilities:** SUITE DB NATIVE WORK

**Proposed Can replacement:** Can case selects and applies fixture, sequences native/raw observations, checks server/version/seed and expected rows in Can; native adapter may return SQL facts and errors only.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/queries.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/queries.sql)

**Notes:** Independent raw-driver channel is required for observations the production Can SQL adapter cannot independently verify.

## HISTORY-016

**sqlite-driver.ts** — [tests/integration/testdata/sql/sqlite-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/sqlite-driver.ts:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-016`. Evidence: pending.

**Protects**

- Applies two-statement SQLite seed and checks empty notes/library version.
- Current TypeScript script contains scenario-specific fixture shape/version checks or expected-result decisions.

**Current observations:** native SQL rows, errors, version and script-generated JSON

**Variants:** None recorded.

**Environment/selection gates:** disposable database; see related Go case gates

**Capabilities:** SUITE DB NATIVE WORK

**Proposed Can replacement:** Can case selects and applies fixture, sequences native/raw observations, checks server/version/seed and expected rows in Can; native adapter may return SQL facts and errors only.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Independent raw-driver channel is required for observations the production Can SQL adapter cannot independently verify.

## HISTORY-017

**transactions-driver.ts** — [tests/integration/testdata/sql/transactions-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/transactions-driver.ts:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-017`. Evidence: pending.

**Protects**

- Applies/tears down five-statement PG transaction seed and checks PG 17/account count.
- Current TypeScript script contains scenario-specific fixture shape/version checks or expected-result decisions.

**Current observations:** native SQL rows, errors, version and script-generated JSON

**Variants:** None recorded.

**Environment/selection gates:** disposable database; see related Go case gates

**Capabilities:** SUITE DB NATIVE WORK

**Proposed Can replacement:** Can case selects and applies fixture, sequences native/raw observations, checks server/version/seed and expected rows in Can; native adapter may return SQL facts and errors only.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/integration/testdata/sql/transactions.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/transactions.sql)

**Notes:** Independent raw-driver channel is required for observations the production Can SQL adapter cannot independently verify.

## HISTORY-018

**descriptors.sql** — [tests/integration/testdata/sql/descriptors.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/descriptors.sql:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-018`. Evidence: pending.

**Protects**

- Static database schema/seed input for the named SQL scenario; exact statement shape and expected starting rows must remain accounted for.

**Current observations:** fixture bytes consumed by driver

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** FILES DB

**Proposed Can replacement:** Can selects this SQL input and applies it through generic database mechanics, then independently checks starting state.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Data fixture, not executable harness. Exact bytes need preservation only while protecting an identified contract.

## HISTORY-019

**mysql-seed.sql** — [tests/integration/testdata/sql/mysql-seed.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/mysql-seed.sql:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-019`. Evidence: pending.

**Protects**

- Static database schema/seed input for the named SQL scenario; exact statement shape and expected starting rows must remain accounted for.

**Current observations:** fixture bytes consumed by driver

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** FILES DB

**Proposed Can replacement:** Can selects this SQL input and applies it through generic database mechanics, then independently checks starting state.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Data fixture, not executable harness. Exact bytes need preservation only while protecting an identified contract.

## HISTORY-020

**queries.sql** — [tests/integration/testdata/sql/queries.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/queries.sql:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-020`. Evidence: pending.

**Protects**

- Static database schema/seed input for the named SQL scenario; exact statement shape and expected starting rows must remain accounted for.

**Current observations:** fixture bytes consumed by driver

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** FILES DB

**Proposed Can replacement:** Can selects this SQL input and applies it through generic database mechanics, then independently checks starting state.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Data fixture, not executable harness. Exact bytes need preservation only while protecting an identified contract.

## HISTORY-021

**sqlite-seed.sql** — [tests/integration/testdata/sql/sqlite-seed.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/sqlite-seed.sql:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-021`. Evidence: pending.

**Protects**

- Static database schema/seed input for the named SQL scenario; exact statement shape and expected starting rows must remain accounted for.

**Current observations:** fixture bytes consumed by driver

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** FILES DB

**Proposed Can replacement:** Can selects this SQL input and applies it through generic database mechanics, then independently checks starting state.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Data fixture, not executable harness. Exact bytes need preservation only while protecting an identified contract.

## HISTORY-022

**transactions.sql** — [tests/integration/testdata/sql/transactions.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/transactions.sql:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-022`. Evidence: pending.

**Protects**

- Static database schema/seed input for the named SQL scenario; exact statement shape and expected starting rows must remain accounted for.

**Current observations:** fixture bytes consumed by driver

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** FILES DB

**Proposed Can replacement:** Can selects this SQL input and applies it through generic database mechanics, then independently checks starting state.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Data fixture, not executable harness. Exact bytes need preservation only while protecting an identified contract.

## HISTORY-023

**TestOwnerSetupGreen** — [tests/failure-conventions/owner_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/owner_test.go:16)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-023`. Evidence: pending.

**Protects**

- Owner helper extraction through private fallible factory, trap kind/message probes and all pre-extraction roots green (14 roots).

**Current observations:** assert report, real-Can provenance

**Variants:** positive owner project

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-024

**TestOwnerNegativeFailsClosed** — [tests/failure-conventions/owner_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/owner_test.go:49)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-024`. Evidence: pending.

**Protects**

- Unexpected factory rejection fails only tainted row with standard fault frames at trap binary span and tainted call site (15 roots).

**Current observations:** assert report, status and located frames

**Variants:** tainted row

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-025

**TestOwnerForgeRejected** — [tests/failure-conventions/owner_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/owner_test.go:89)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-025`. Evidence: pending.

**Protects**

- Foreign owner-record constructor is rejected at check phase, with no assertion report.

**Current observations:** compiler rejection diagnostic

**Variants:** forged owner source

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-026

**TestFactoryPrivateToPackage** — [tests/failure-conventions/owner_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/owner_test.go:107)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-026`. Evidence: pending.

**Protects**

- Foreign package cannot call private fixture_id factory.

**Current observations:** compiler rejection diagnostic

**Variants:** injected sneak package

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-027

**TestFactoryRepairGuided** — [tests/failure-conventions/owner_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/owner_test.go:197)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-027`. Evidence: pending.

**Protects**

- Adding ids::retired first yields missing completion arm; repaired factory yields 16 green roots while tier_for helper source remains identical.

**Current observations:** negative diagnostic, assert report, source snapshot

**Variants:** incomplete/complete repair

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-028

**TestRetryConventionGreen** — [tests/failure-conventions/retry_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/retry_test.go:14)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-028`. Evidence: pending.

**Protects**

- Shared result-data retry helper handles profile and billing failure sets, capture/re-raise/traces, standard faults and correct FIFO attempts across 35 roots.

**Current observations:** real-Can assert report

**Variants:** profiles/billing 35 roots

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-029

**TestRetryFixedGreen** — [tests/failure-conventions/retry_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/retry_test.go:31)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-029`. Evidence: pending.

**Protects**

- Per-domain fixed-bound retry comparison with explicit emits and layered consumer has 15 green roots.

**Current observations:** real-Can assert report

**Variants:** 15 roots

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-030

**TestRetryNoRetryMutantFails** — [tests/failure-conventions/retry_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/retry_test.go:49)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-030`. Evidence: pending.

**Protects**

- No-retry mutant makes exactly eleven two-attempt oracle roots fail unused-fixture while all other roots pass.

**Current observations:** assert report root IDs/violations

**Variants:** eleven named roots unused fixture

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-031

**TestRetryOverAttemptFails** — [tests/failure-conventions/retry_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/retry_test.go:91)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-031`. Evidence: pending.

**Protects**

- Three attempts against two queued rows produce missing-fixture in exhausted consumer roots while other roots remain green.

**Current observations:** assert report root IDs/violations

**Variants:** mutated retry count

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-032

**TestAddErrorIsolationResultData** — [tests/failure-conventions/retry_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/retry_test.go:285)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-032`. Evidence: pending.

**Protects**

- Adding profiles::suspended to result-data design leaves unrelated app/retry/billing files byte-identical and produces 40 green roots.

**Current observations:** source snapshots, assert report

**Variants:** add-error surgery

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-033

**TestAddErrorIsolationFixed** — [tests/failure-conventions/retry_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/retry_test.go:411)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-033`. Evidence: pending.

**Protects**

- Same added error in fixed-bound comparison leaves unrelated files identical and produces 18 green roots.

**Current observations:** source snapshots, assert report

**Variants:** add-error surgery

**Environment/selection gates:** CONV_BUNDLE or CAN_BUN_ARCHIVE

**Capabilities:** SUITE CAN FAULT DIAG WORK

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/failure-conventions/retry](/Users/vince/Projects/can-lang/tests/failure-conventions/retry); [tests/failure-conventions/retry-fixed](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-fixed); [tests/failure-conventions/retry-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/retry-negative); [tests/failure-conventions/owner-setup](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup); [tests/failure-conventions/owner-negative](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-negative); [tests/failure-conventions/owner-forge](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-forge)

**Notes:** For mutation tests retain the exact failing root/violation sensitivity, not only a green baseline.

## HISTORY-034

**TestClosePreservesParentAndConcurrentCache** — [tests/support/tempcache/cache_test.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache_test.go:40)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-034`. Evidence: pending.

**Protects**

- Closing one owned cache removes only that cache, preserving its parent and concurrent cache.

**Current observations:** filesystem path existence

**Variants:** two concurrent caches

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE WORK FILES FAULT NATIVE

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/support/tempcache/cache.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache.go)

**Notes:** Owned-resource semantics must survive replacement even if cache implementation changes.

## HISTORY-035

**TestRecoveryRequiresInactivityAndPreservesForeignPaths** — [tests/support/tempcache/cache_test.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache_test.go:67)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-035`. Evidence: pending.

**Protects**

- Recovery deletes only old dead unlocked owned cache; preserves fresh, live, locked, foreign-marker and symlink paths.

**Current observations:** filesystem/liveness/lock checks

**Variants:** abandoned/fresh/live/locked/foreign/symlink

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE WORK FILES FAULT NATIVE

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/support/tempcache/cache.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache.go)

**Notes:** Owned-resource semantics must survive replacement even if cache implementation changes.

## HISTORY-036

**TestUnavailableLivenessRetainsOrphan** — [tests/support/tempcache/cache_test.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache_test.go:111)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-036`. Evidence: pending.

**Protects**

- Inconclusive liveness probe retains orphan rather than deleting it.

**Current observations:** path existence after probe failure

**Variants:** lsof unavailable

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE WORK FILES FAULT NATIVE

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/support/tempcache/cache.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache.go)

**Notes:** Owned-resource semantics must survive replacement even if cache implementation changes.

## HISTORY-037

**TestRecoveryRejectsSymlinkMarker** — [tests/support/tempcache/cache_test.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache_test.go:122)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-037`. Evidence: pending.

**Protects**

- Symlinked owner marker is not followed or offered to liveness check.

**Current observations:** liveness callback and path existence

**Variants:** symlink marker

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE WORK FILES FAULT NATIVE

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/support/tempcache/cache.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache.go)

**Notes:** Owned-resource semantics must survive replacement even if cache implementation changes.

## HISTORY-038

**TestRecoveryWorkIsBounded** — [tests/support/tempcache/cache_test.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache_test.go:152)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-038`. Evidence: pending.

**Protects**

- Recovery processes at most recoveryLimit candidates and leaves overflow for later.

**Current observations:** candidate count and remaining directory

**Variants:** limit+1 abandoned

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE WORK FILES FAULT NATIVE

**Proposed Can replacement:** Can-authored case stages the relevant ordinary Can fixture or resource state, invokes the candidate/owned-resource operation, and checks the specified positive and deliberately broken result with Can comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/support/tempcache/cache.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache.go)

**Notes:** Owned-resource semantics must survive replacement even if cache implementation changes.

## HISTORY-039

**TestMain** — [tests/failure-conventions/harness_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/harness_test.go:142)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-HISTORY-039`. Evidence: pending.

**Protects**

- Suite-owned bundle cache is allocated once, reused by key, recovers abandoned owned work, and closes after m.Run; CONV_BUNDLE bypasses build.

**Current observations:** cache identity, cleanup and process exit

**Variants:** None recorded.

**Environment/selection gates:** CONV_BUNDLE CAN_BUN_ARCHIVE CAN_TEST_CACHE

**Capabilities:** SUITE BUILD WORK ENV

**Proposed Can replacement:** Can suite policy selects bundle/build reuse and reports cleanup; native supervisor owns lifetime and stale recovery.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/support/tempcache/cache.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache.go)

**Notes:** 

## HISTORY-040

**staging/assert-report helpers** — [tests/failure-conventions/harness_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/harness_test.go:1)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-HISTORY-040`. Evidence: pending.

**Protects**

- Stage Can projects, run candidate assertions, require valid green/real-Can reports, snapshot/mutate sources and compare helper-local surgery.

**Current observations:** process status, report data, file snapshots

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE CAN PROC DIAG FILES WORK

**Proposed Can replacement:** Implement ordinary reusable Can helper functions for staging, assertions, report decoding, mutations and source comparisons.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** 

## HISTORY-041

**tempcache implementation** — [tests/support/tempcache/cache.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache.go:1)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-HISTORY-041`. Evidence: pending.

**Protects**

- Per-run owned cache, parent protection, marker and lease validation, inactivity/liveness checks, bounded abandoned recovery and close behavior.

**Current observations:** filesystem identity, marker/lease and process liveness

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** WORK FILES NATIVE

**Proposed Can replacement:** Move lifecycle policy and its tests to Can; native resource/supervisor may enforce process-safe ownership and cleanup mechanics.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** External host/conformance caller also imports this package and must be rewired before removal.

## HISTORY-042

**TestBaselineToolchainMatchesFreeze** — [tests/baseline/baseline_test.go](/Users/vince/Projects/can-lang/tests/baseline/baseline_test.go:68)

Kind: `go_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-042`. Evidence: pending.

**Protects**

- Compares current Bun version/revision and Go minimum to frozen baseline; frozen compiler revision must be full SHA and ancestor of HEAD.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Pinned historical T01 toolchain values may be obsolete; current provenance/required platform checks must be assigned elsewhere before retirement.

## HISTORY-043

**TestRegistryComplete** — [tests/baseline/baseline_test.go](/Users/vince/Projects/can-lang/tests/baseline/baseline_test.go:155)

Kind: `go_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-043`. Evidence: pending.

**Protects**

- T01 registry has unique complete cases, prompts, current fixtures, pending candidates, held-out checks, and frozen model/effort/attempt settings.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Registered trial protocol is dated; determine whether a live authoring policy still consumes it.

## HISTORY-044

**TestHarnessSmokeRepeatable** — [tests/baseline/baseline_test.go](/Users/vince/Projects/can-lang/tests/baseline/baseline_test.go:231)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-044`. Evidence: pending.

**Protects**

- Baseline shell smoke runs in disposable temp outside source, returns all passing steps and zero failures; test explicitly retains then cleans report directory.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Current resource/report semantics must survive even if T01 historical gates retire.

## HISTORY-045

**TestT26ProductGuideCoversContracts** — [tests/baseline/t26_product_guide_test.go](/Users/vince/Projects/can-lang/tests/baseline/t26_product_guide_test.go:30)

Kind: `go_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-045`. Evidence: pending.

**Protects**

- Guide names supported targets, browser/SQL/action limits and failure/replay/shutdown claims, and cited implementation anchors retain markers.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Dated target/version and wording pins may retire; live guide consistency and current contract claims need replacement evidence.

## HISTORY-046

**TestT26AgentComparisonReportConsistent** — [tests/baseline/t26_product_guide_test.go](/Users/vince/Projects/can-lang/tests/baseline/t26_product_guide_test.go:150)

Kind: `go_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-046`. Evidence: pending.

**Protects**

- T26 report covers registry cases, honestly records zero unrun agent trials/no significance, explicit dispositions and deferred candidate files.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Historical report integrity can remain as record; do not claim unrun trials passed.

## HISTORY-047

**TestT26GeneratedArtifactsFresh** — [tests/baseline/t26_product_guide_test.go](/Users/vince/Projects/can-lang/tests/baseline/t26_product_guide_test.go:211)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-047`. Evidence: pending.

**Protects**

- Catalogue generator check passes and runtime/catalogue.ts exists nonempty.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Fresh generated artefacts remain a current gate independent of old T26 report.

## HISTORY-048

**TestT27DIInventoryComplete** — [tests/baseline/t27_release_audit_test.go](/Users/vince/Projects/can-lang/tests/baseline/t27_release_audit_test.go:35)

Kind: `go_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-048`. Evidence: pending.

**Protects**

- All DI-01..DI-23 audit rows have exact IDs and dispositions, disambiguating 09/09b and 14/14a.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Dated decision inventory; confirm current audit process before retirement.

## HISTORY-049

**TestT27AcceptedItemsMapToTestsAndDocs** — [tests/baseline/t27_release_audit_test.go](/Users/vince/Projects/can-lang/tests/baseline/t27_release_audit_test.go:83)

Kind: `go_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-049`. Evidence: pending.

**Protects**

- Accepted/retained DI items map to existing proving tests and implementation/doc anchors; four runtime unit suites exist.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Current behavior tests remain independent; mere existence of foreign runtime suites cannot count as migrated /tests coverage.

## HISTORY-050

**TestT27NoDeferredMechanismPresent** — [tests/baseline/t27_release_audit_test.go](/Users/vince/Projects/can-lang/tests/baseline/t27_release_audit_test.go:166)

Kind: `go_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-050`. Evidence: pending.

**Protects**

- No unauthorized numbered errors, RETURNING, finally/markup or bulk/billing/calendar surface; explicit parser rejection and deferral anchors persist.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Some negative syntax/catalogue exclusions may be current product contracts; decide each separately rather than blanket-retiring dated audit.

## HISTORY-051

**TestT27GateEvidenceLinked** — [tests/baseline/t27_release_audit_test.go](/Users/vince/Projects/can-lang/tests/baseline/t27_release_audit_test.go:303)

Kind: `go_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-051`. Evidence: pending.

**Protects**

- Audit links extant nonempty compiler/integration/browser evidence and baseline/consultation records.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Historical audit links should be preserved as provenance, not credited as live test execution.

## HISTORY-052

**TestT27GeneratedArtifactsFresh** — [tests/baseline/t27_release_audit_test.go](/Users/vince/Projects/can-lang/tests/baseline/t27_release_audit_test.go:334)

Kind: `go_test`. Proposed disposition: `consolidate`. Replacement obligation: `CAN-HISTORY-052`. Evidence: pending.

**Protects**

- Catalogue generator check passes and generated Go/TS files bear DO NOT EDIT markers.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Consolidate with T26 freshness gate only after both generator and marker obligations are covered.

## HISTORY-053

**TestT27RecommendationScoped** — [tests/baseline/t27_release_audit_test.go](/Users/vince/Projects/can-lang/tests/baseline/t27_release_audit_test.go:355)

Kind: `go_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-053`. Evidence: pending.

**Protects**

- Dated release recommendation limits qualified Bun/browsers/Linux, excludes Firefox and claims no offline durability/queue.

**Current observations:** file/registry/document contents and, where used, command exit/output

**Variants:** None recorded.

**Environment/selection gates:** Bun/Go/Git for T01 smoke or generator when applicable

**Capabilities:** SUITE DOC FILES PROC BUILD WORK

**Proposed Can replacement:** Can case checks current retained contract from structured files and product commands; if dated-only, retain its record and record evidence for retirement.

**Old harness may be deleted when:** Only after each current sub-obligation has Can evidence or a documented dated-record retirement decision, historical files remain available as records where required, and callers stop invoking this Go test.

**Notes:** Scope statements are evidence history; current qualification must be independently recorded.

## HISTORY-054

**baseline smoke/full shell runner** — [tests/baseline/run.sh](/Users/vince/Projects/can-lang/tests/baseline/run.sh:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-054`. Evidence: pending.

**Protects**

- Smoke collects toolchain provenance, gofmt, compiler build and version; full mode additionally runs vet, catalogue, grammar, runtime checks and all Go/Bun tests, recording steps/report.

**Current observations:** shell command statuses and baseline-report.json

**Variants:** --smoke --full KEEP_DIR retention

**Environment/selection gates:** local Bun/Go/Git; full is broad and currently deferred

**Capabilities:** SUITE DOC PROC BUILD WORK ENV

**Proposed Can replacement:** Can suite defines selectable current gate profiles, expected status and report policy; CI launcher may invoke Can and propagate exit; historical T01 comparison may remain a record.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No broad full run authorized in this documentation round. Must not implement full mode as Can wrapping former Go/Bun host suites.

## HISTORY-055

**baseline.json** — [tests/baseline/baseline.json](/Users/vince/Projects/can-lang/tests/baseline/baseline.json:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-055`. Evidence: pending.

**Protects**

- Frozen T01/T26 trial, deferral or comparison record; current assertions may read it but its dates and unrun trials are not new execution evidence.

**Current observations:** stored historical bytes/provenance

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as labelled historical evidence or explicitly map any still-current data vector to a Can case.

**Old harness may be deleted when:** Archive/retire only after provenance, live callers and current obligations are reconciled; no automatic deletion from dated name.

**Notes:** Do not count historical records as passing native tests.

## HISTORY-056

**registry.json** — [tests/baseline/registry.json](/Users/vince/Projects/can-lang/tests/baseline/registry.json:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-056`. Evidence: pending.

**Protects**

- Frozen T01/T26 trial, deferral or comparison record; current assertions may read it but its dates and unrun trials are not new execution evidence.

**Current observations:** stored historical bytes/provenance

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as labelled historical evidence or explicitly map any still-current data vector to a Can case.

**Old harness may be deleted when:** Archive/retire only after provenance, live callers and current obligations are reconciled; no automatic deletion from dated name.

**Notes:** Do not count historical records as passing native tests.

## HISTORY-057

**t26-agent-comparison-2026-09-24.json** — [tests/baseline/reports/t26-agent-comparison-2026-09-24.json](/Users/vince/Projects/can-lang/tests/baseline/reports/t26-agent-comparison-2026-09-24.json:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-057`. Evidence: pending.

**Protects**

- Frozen T01/T26 trial, deferral or comparison record; current assertions may read it but its dates and unrun trials are not new execution evidence.

**Current observations:** stored historical bytes/provenance

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as labelled historical evidence or explicitly map any still-current data vector to a Can case.

**Old harness may be deleted when:** Archive/retire only after provenance, live callers and current obligations are reconciled; no automatic deletion from dated name.

**Notes:** Do not count historical records as passing native tests.

## HISTORY-058

**t26-agent-comparison-2026-09-24.md** — [tests/baseline/reports/t26-agent-comparison-2026-09-24.md](/Users/vince/Projects/can-lang/tests/baseline/reports/t26-agent-comparison-2026-09-24.md:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-058`. Evidence: pending.

**Protects**

- Frozen T01/T26 trial, deferral or comparison record; current assertions may read it but its dates and unrun trials are not new execution evidence.

**Current observations:** stored historical bytes/provenance

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as labelled historical evidence or explicitly map any still-current data vector to a Can case.

**Old harness may be deleted when:** Archive/retire only after provenance, live callers and current obligations are reconciled; no automatic deletion from dated name.

**Notes:** Do not count historical records as passing native tests.

## HISTORY-059

**README.md** — [tests/baseline/candidates/T26-native-ai/README.md](/Users/vince/Projects/can-lang/tests/baseline/candidates/T26-native-ai/README.md:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-059`. Evidence: pending.

**Protects**

- Frozen T01/T26 trial, deferral or comparison record; current assertions may read it but its dates and unrun trials are not new execution evidence.

**Current observations:** stored historical bytes/provenance

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as labelled historical evidence or explicitly map any still-current data vector to a Can case.

**Old harness may be deleted when:** Archive/retire only after provenance, live callers and current obligations are reconciled; no automatic deletion from dated name.

**Notes:** Do not count historical records as passing native tests.

## HISTORY-060

**candidate.json** — [tests/baseline/candidates/T26-native-ai/candidate.json](/Users/vince/Projects/can-lang/tests/baseline/candidates/T26-native-ai/candidate.json:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-060`. Evidence: pending.

**Protects**

- Frozen T01/T26 trial, deferral or comparison record; current assertions may read it but its dates and unrun trials are not new execution evidence.

**Current observations:** stored historical bytes/provenance

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as labelled historical evidence or explicitly map any still-current data vector to a Can case.

**Old harness may be deleted when:** Archive/retire only after provenance, live callers and current obligations are reconciled; no automatic deletion from dated name.

**Notes:** Do not count historical records as passing native tests.

## HISTORY-061

**TestPrototypeCapabilitiesUnadmitted** — [tests/host-discrimination/admission_test.go](/Users/vince/Projects/can-lang/tests/host-discrimination/admission_test.go:29)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-061`. Evidence: pending.

**Protects**

- Catalogue source and generated runtime contain no storage/clipboard/chart prototype identities; prevents self-admission.

**Current observations:** catalogue/owned-tree/manifest source inspection

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE DOC FILES BUILD

**Proposed Can replacement:** Can case checks prototype admission status against current catalogue and tracked source; if prototypes are admitted or retired, require explicit decision and new guard before deleting old check.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Guard against unintended prototype leakage is potentially current even though prototype measurements are historical.

## HISTORY-062

**TestPrototypesUnreferenced** — [tests/host-discrimination/admission_test.go](/Users/vince/Projects/can-lang/tests/host-discrimination/admission_test.go:66)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-062`. Evidence: pending.

**Protects**

- Owned production trees do not reference host-discrimination prototype paths; reverse imports remain allowed.

**Current observations:** catalogue/owned-tree/manifest source inspection

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE DOC FILES BUILD

**Proposed Can replacement:** Can case checks prototype admission status against current catalogue and tracked source; if prototypes are admitted or retired, require explicit decision and new guard before deleting old check.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Guard against unintended prototype leakage is potentially current even though prototype measurements are historical.

## HISTORY-063

**TestReviewManifestCoversPrototypes** — [tests/host-discrimination/admission_test.go](/Users/vince/Projects/can-lang/tests/host-discrimination/admission_test.go:126)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-063`. Evidence: pending.

**Protects**

- Review manifest is closed, unreviewed status and each prototype TS file appears exactly once without dangling entries.

**Current observations:** catalogue/owned-tree/manifest source inspection

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE DOC FILES BUILD

**Proposed Can replacement:** Can case checks prototype admission status against current catalogue and tracked source; if prototypes are admitted or retired, require explicit decision and new guard before deleting old check.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Guard against unintended prototype leakage is potentially current even though prototype measurements are historical.

## HISTORY-064

**"create, select, update, dispose roundtrip"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:33)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-064`. Evidence: pending.

**Protects**

- create, select, update, dispose roundtrip; operations dispose, update

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-065

**"invalid specs reject before host contact"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:44)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-065`. Evidence: pending.

**Protects**

- invalid specs reject before host contact; operations create; expected leaves chart::invalid_spec; no host contact or wire send on rejected input, pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-066

**"unavailable SDK fails closed without host contact"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:64)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-066`. Evidence: pending.

**Protects**

- unavailable SDK fails closed without host contact; operations create; expected leaves chart::unavailable; no host contact or wire send on rejected input, pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-067

**"native SDK failure maps to unavailable with no native text"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:73)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-067`. Evidence: pending.

**Protects**

- native SDK failure maps to unavailable with no native text; operations create; expected leaves chart::unavailable; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-068

**"deep copies: caller mutation cannot reach the widget or callbacks"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:84)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-068`. Evidence: pending.

**Protects**

- deep copies: caller mutation cannot reach the widget or callbacks; operations protocol/packaging; frozen/opaque output

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-069

**"callback throw is sealed: one diagnostic, SDK never sees it"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:97)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-069`. Evidence: pending.

**Protects**

- callback throw is sealed: one diagnostic, SDK never sees it; operations protocol/packaging; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-070

**"nested events serialize in dispatch order"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:110)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-070`. Evidence: pending.

**Protects**

- nested events serialize in dispatch order; operations protocol/packaging

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-071

**"reentrant update applies immediately during dispatch"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:122)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-071`. Evidence: pending.

**Protects**

- reentrant update applies immediately during dispatch; operations create, update; pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-072

**"dispose poisons: late events dropped, ops fail, double dispose ok"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:134)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-072`. Evidence: pending.

**Protects**

- dispose poisons: late events dropped, ops fail, double dispose ok; operations dispose, update; expected leaves chart::disposed

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-073

**"dispose during callback completes the callback then drops the rest"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:151)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-073`. Evidence: pending.

**Protects**

- dispose during callback completes the callback then drops the rest; operations create, dispose

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-074

**"tokens are opaque: SDK handles never leak to callers"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:167)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-074`. Evidence: pending.

**Protects**

- tokens are opaque: SDK handles never leak to callers; operations protocol/packaging; frozen/opaque output

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-075

**"surface census: three ops, three failure leaves, one callback"** — [tests/host-discrimination/adapters/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.test.ts:175)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-075`. Evidence: pending.

**Protects**

- surface census: three ops, three failure leaves, one callback; operations create, dispose, update; expected leaves chart::disposed, chart::invalid_spec, chart::unavailable

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-076

**"roundtrip: write then read returns the text"** — [tests/host-discrimination/adapters/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.test.ts:4)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-076`. Evidence: pending.

**Protects**

- roundtrip: write then read returns the text; operations readText, writeText

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-077

**"empty clipboard reads the empty leaf"** — [tests/host-discrimination/adapters/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.test.ts:10)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-077`. Evidence: pending.

**Protects**

- empty clipboard reads the empty leaf; operations readText; expected leaves clipboard::empty

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-078

**"invalid writes reject before host contact"** — [tests/host-discrimination/adapters/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.test.ts:17)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-078`. Evidence: pending.

**Protects**

- invalid writes reject before host contact; operations writeText; expected leaves clipboard::invalid_text; no host contact or wire send on rejected input, pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-079

**"denied permission fails closed before clipboard contact"** — [tests/host-discrimination/adapters/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.test.ts:28)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-079`. Evidence: pending.

**Protects**

- denied permission fails closed before clipboard contact; operations readText, writeText; expected leaves clipboard::denied

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-080

**"unavailable target fails closed without host contact"** — [tests/host-discrimination/adapters/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.test.ts:42)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-080`. Evidence: pending.

**Protects**

- unavailable target fails closed without host contact; operations readText, writeText; expected leaves clipboard::unavailable; no host contact or wire send on rejected input, pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-081

**"native denial and native failure map with no native text"** — [tests/host-discrimination/adapters/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.test.ts:55)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-081`. Evidence: pending.

**Protects**

- native denial and native failure map with no native text; operations readText, writeText; expected leaves clipboard::denied, clipboard::unavailable; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-082

**"reentrancy: concurrent writes settle independently in host order"** — [tests/host-discrimination/adapters/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.test.ts:75)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-082`. Evidence: pending.

**Protects**

- reentrancy: concurrent writes settle independently in host order; operations readText, writeText

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-083

**"surface census: two ops, four failure leaves, zero callbacks"** — [tests/host-discrimination/adapters/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.test.ts:85)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-083`. Evidence: pending.

**Protects**

- surface census: two ops, four failure leaves, zero callbacks; operations readText, writeText; expected leaves clipboard::denied, clipboard::empty, clipboard::invalid_text, clipboard::unavailable

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-084

**"roundtrip: set, get, remove, missing reads null"** — [tests/host-discrimination/adapters/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.test.ts:9)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-084`. Evidence: pending.

**Protects**

- roundtrip: set, get, remove, missing reads null; operations localGet, localRemove, localSet

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-085

**"invalid keys and values reject before host contact"** — [tests/host-discrimination/adapters/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.test.ts:17)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-085`. Evidence: pending.

**Protects**

- invalid keys and values reject before host contact; operations localGet, localSet; expected leaves storage::invalid_key; no host contact or wire send on rejected input, pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-086

**"quota breach maps to quota_exceeded with no native text"** — [tests/host-discrimination/adapters/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.test.ts:32)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-086`. Evidence: pending.

**Protects**

- quota breach maps to quota_exceeded with no native text; operations localSet; expected leaves storage::quota_exceeded; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-087

**"unavailable target fails closed without host contact"** — [tests/host-discrimination/adapters/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.test.ts:48)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-087`. Evidence: pending.

**Protects**

- unavailable target fails closed without host contact; operations localGet, localRemove, localSet; expected leaves storage::unavailable; no host contact or wire send on rejected input, pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-088

**"native security failure maps to unavailable with no native text"** — [tests/host-discrimination/adapters/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.test.ts:59)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-088`. Evidence: pending.

**Protects**

- native security failure maps to unavailable with no native text; operations localGet; expected leaves storage::unavailable; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-089

**"immutable copies: results are frozen primitives with no aliasing"** — [tests/host-discrimination/adapters/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.test.ts:71)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-089`. Evidence: pending.

**Protects**

- immutable copies: results are frozen primitives with no aliasing; operations localGet, localSet; frozen/opaque output

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-090

**"surface census: three ops, three failure leaves, zero callbacks"** — [tests/host-discrimination/adapters/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.test.ts:83)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-090`. Evidence: pending.

**Protects**

- surface census: three ops, three failure leaves, zero callbacks; operations localGet, localSet; expected leaves storage::invalid_key, storage::quota_exceeded, storage::unavailable

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts)

**Notes:** prototype adapter is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-091

**"render, select, release roundtrip"** — [tests/host-discrimination/companions/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.test.ts:48)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-091`. Evidence: pending.

**Protects**

- render, select, release roundtrip; operations release, render, select; frozen/opaque output

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-092

**"invalid specs reject client-side without sending"** — [tests/host-discrimination/companions/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.test.ts:63)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-092`. Evidence: pending.

**Protects**

- invalid specs reject client-side without sending; operations render; expected leaves chart::invalid_spec; no host contact or wire send on rejected input

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-093

**"select-after-release expires; double release is idempotent"** — [tests/host-discrimination/companions/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.test.ts:84)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-093`. Evidence: pending.

**Protects**

- select-after-release expires; double release is idempotent; operations release, render, select; expected leaves chart::expired

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-094

**"out-of-range index rejects against a live render"** — [tests/host-discrimination/companions/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.test.ts:100)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-094`. Evidence: pending.

**Protects**

- out-of-range index rejects against a live render; operations render, select; expected leaves chart::invalid_spec

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-095

**"denied destination, bad credential, and offline fail closed"** — [tests/host-discrimination/companions/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.test.ts:109)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-095`. Evidence: pending.

**Protects**

- denied destination, bad credential, and offline fail closed; operations render; expected leaves companion::destination_denied, companion::transport_failed, companion::unauthorized; no host contact or wire send on rejected input, native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-096

**"version mismatch fails closed both directions"** — [tests/host-discrimination/companions/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.test.ts:159)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-096`. Evidence: pending.

**Protects**

- version mismatch fails closed both directions; operations protocol/packaging; expected leaves companion::version_mismatch

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-097

**"auth parity: same policy both sides; secret never logged"** — [tests/host-discrimination/companions/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.test.ts:168)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-097`. Evidence: pending.

**Protects**

- auth parity: same policy both sides; secret never logged; operations render; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-098

**"envelope bytes are deterministic and measured"** — [tests/host-discrimination/companions/chart-widget.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.test.ts:183)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-098`. Evidence: pending.

**Protects**

- envelope bytes are deterministic and measured; operations protocol/packaging; pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-099

**"pastebin roundtrip: write then read returns the text"** — [tests/host-discrimination/companions/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.test.ts:45)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-099`. Evidence: pending.

**Protects**

- pastebin roundtrip: write then read returns the text; operations readText, writeText

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-100

**"empty companion cell reads the empty leaf"** — [tests/host-discrimination/companions/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.test.ts:51)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-100`. Evidence: pending.

**Protects**

- empty companion cell reads the empty leaf; operations readText; expected leaves clipboard::empty

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-101

**"denied destination and bad credential fail closed"** — [tests/host-discrimination/companions/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.test.ts:58)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-101`. Evidence: pending.

**Protects**

- denied destination and bad credential fail closed; operations readText; expected leaves companion::destination_denied, companion::unauthorized; no host contact or wire send on rejected input

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-102

**"version mismatch and transport failure map honestly"** — [tests/host-discrimination/companions/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.test.ts:93)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-102`. Evidence: pending.

**Protects**

- version mismatch and transport failure map honestly; operations readText; expected leaves companion::transport_failed, companion::version_mismatch; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-103

**"auth parity: same policy both sides; secret never logged"** — [tests/host-discrimination/companions/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.test.ts:118)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-103`. Evidence: pending.

**Protects**

- auth parity: same policy both sides; secret never logged; operations writeText; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-104

**"semantic gap: companion cell and device cell are disjoint"** — [tests/host-discrimination/companions/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.test.ts:133)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-104`. Evidence: pending.

**Protects**

- semantic gap: companion cell and device cell are disjoint; operations readText, writeText; expected leaves clipboard::empty

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-105

**"envelope bytes are deterministic and measured"** — [tests/host-discrimination/companions/clipboard.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.test.ts:150)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-105`. Evidence: pending.

**Protects**

- envelope bytes are deterministic and measured; operations protocol/packaging; pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-106

**"roundtrip: set, get, remove, missing reads null"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:44)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-106`. Evidence: pending.

**Protects**

- roundtrip: set, get, remove, missing reads null; operations localGet, localRemove, localSet

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-107

**"invalid keys reject client-side without sending"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:52)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-107`. Evidence: pending.

**Protects**

- invalid keys reject client-side without sending; operations localGet; expected leaves storage::invalid_key; no host contact or wire send on rejected input

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-108

**"denied destination fails closed without sending"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:77)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-108`. Evidence: pending.

**Protects**

- denied destination fails closed without sending; operations localGet; expected leaves companion::destination_denied; no host contact or wire send on rejected input

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-109

**"bad credential maps to unauthorized on both sides"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:98)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-109`. Evidence: pending.

**Protects**

- bad credential maps to unauthorized on both sides; operations localGet; expected leaves companion::unauthorized

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-110

**"version mismatch fails closed both directions"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:123)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-110`. Evidence: pending.

**Protects**

- version mismatch fails closed both directions; operations protocol/packaging; expected leaves companion::version_mismatch

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-111

**"transport failure maps to transport_failed"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:134)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-111`. Evidence: pending.

**Protects**

- transport failure maps to transport_failed; operations localGet; expected leaves companion::transport_failed; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-112

**"malformed response maps to protocol_error"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:153)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-112`. Evidence: pending.

**Protects**

- malformed response maps to protocol_error; operations localGet; expected leaves companion::protocol_error

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-113

**"server quota passes through as quota_exceeded"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:169)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-113`. Evidence: pending.

**Protects**

- server quota passes through as quota_exceeded; operations localSet; expected leaves storage::quota_exceeded

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-114

**"auth parity: same policy admits on both sides; secret never logged"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:192)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-114`. Evidence: pending.

**Protects**

- auth parity: same policy admits on both sides; secret never logged; operations localSet; native exception or secret text does not leak

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-115

**"envelope bytes are deterministic and measured"** — [tests/host-discrimination/companions/storage.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.test.ts:207)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-115`. Evidence: pending.

**Protects**

- envelope bytes are deterministic and measured; operations protocol/packaging; pinned byte/count observation

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FAULT

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts)

**Notes:** prototype companion is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-116

**"prototype bundles are byte-identical across builds"** — [tests/host-discrimination/packaging.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/packaging.test.ts:24)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-116`. Evidence: pending.

**Protects**

- prototype bundles are byte-identical across builds; operations protocol/packaging; pinned byte/count observation, six prototype bundle entries, repeatability/size observations

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FILES FAULT BUILD

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts); [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts); [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts); [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts); [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts); [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype packaging is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-117

**"bundle sizes are pinned"** — [tests/host-discrimination/packaging.test.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/packaging.test.ts:33)

Kind: `bun_test`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-117`. Evidence: pending.

**Protects**

- bundle sizes are pinned; operations protocol/packaging; six prototype bundle entries, repeatability/size observations

**Current observations:** Bun expect assertions over fake host/server calls, result records and observed bytes

**Variants:** all values/host faults enumerated in this declaration

**Environment/selection gates:** Bun prototype test environment

**Capabilities:** SUITE NATIVE FILES FAULT BUILD

**Proposed Can replacement:** If this experiment remains a current gate, Can-authored scenario recreates its specific fake-host or companion protocol actions and compares the observed outcomes; otherwise preserve measured result as historical evidence.

**Old harness may be deleted when:** Only after a per-test admission/current-contract review distinguishes live product requirements from dated tier measurements, retained obligations have Can evidence, and prototype callers are retired.

**Related source:** [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts); [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts); [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts); [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts); [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts); [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts)

**Notes:** prototype packaging is explicitly unadmitted; old bundle-size/byte pins are candidate historical measurements, while semantic failures may still express a current intended contract.

## HISTORY-118

**REVIEW-MANIFEST.json** — [tests/host-discrimination/REVIEW-MANIFEST.json](/Users/vince/Projects/can-lang/tests/host-discrimination/REVIEW-MANIFEST.json:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-118`. Evidence: pending.

**Protects**

- Records unadmitted prototype inventory, tier comparison and selection context.

**Current observations:** stored document/manifest

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve record; Can admission guard may read manifest if still current.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No prototype is admitted solely by test migration.

## HISTORY-119

**x-r01-1.md** — [tests/host-discrimination/x-r01-1.md](/Users/vince/Projects/can-lang/tests/host-discrimination/x-r01-1.md:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-119`. Evidence: pending.

**Protects**

- Records unadmitted prototype inventory, tier comparison and selection context.

**Current observations:** stored document/manifest

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve record; Can admission guard may read manifest if still current.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No prototype is admitted solely by test migration.

## HISTORY-120

**t1-sketches.md** — [tests/host-discrimination/catalogue/t1-sketches.md](/Users/vince/Projects/can-lang/tests/host-discrimination/catalogue/t1-sketches.md:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-120`. Evidence: pending.

**Protects**

- Records unadmitted prototype inventory, tier comparison and selection context.

**Current observations:** stored document/manifest

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve record; Can admission guard may read manifest if still current.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No prototype is admitted solely by test migration.

## HISTORY-121

**chart-widget.ts** — [tests/host-discrimination/adapters/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/chart-widget.ts:1)

Kind: `support`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-121`. Evidence: pending.

**Protects**

- Unadmitted adapter/companion prototype implementation exercised by associated Bun test declarations; must not be mistaken for production Can coverage.

**Current observations:** fake host or protocol implementation

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** NATIVE DOC

**Proposed Can replacement:** Retain as historical prototype only with explicit role, or replace any admitted production behavior with reviewed native mechanics and Can-owned tests.

**Old harness may be deleted when:** After prototype admission/retirement decision, no active caller, and migration of any still-current semantic obligation.

**Notes:** Source under tests is executable implementation, not a passive JSON fixture.

## HISTORY-122

**clipboard.ts** — [tests/host-discrimination/adapters/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/clipboard.ts:1)

Kind: `support`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-122`. Evidence: pending.

**Protects**

- Unadmitted adapter/companion prototype implementation exercised by associated Bun test declarations; must not be mistaken for production Can coverage.

**Current observations:** fake host or protocol implementation

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** NATIVE DOC

**Proposed Can replacement:** Retain as historical prototype only with explicit role, or replace any admitted production behavior with reviewed native mechanics and Can-owned tests.

**Old harness may be deleted when:** After prototype admission/retirement decision, no active caller, and migration of any still-current semantic obligation.

**Notes:** Source under tests is executable implementation, not a passive JSON fixture.

## HISTORY-123

**storage.ts** — [tests/host-discrimination/adapters/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/adapters/storage.ts:1)

Kind: `support`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-123`. Evidence: pending.

**Protects**

- Unadmitted adapter/companion prototype implementation exercised by associated Bun test declarations; must not be mistaken for production Can coverage.

**Current observations:** fake host or protocol implementation

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** NATIVE DOC

**Proposed Can replacement:** Retain as historical prototype only with explicit role, or replace any admitted production behavior with reviewed native mechanics and Can-owned tests.

**Old harness may be deleted when:** After prototype admission/retirement decision, no active caller, and migration of any still-current semantic obligation.

**Notes:** Source under tests is executable implementation, not a passive JSON fixture.

## HISTORY-124

**chart-widget.ts** — [tests/host-discrimination/companions/chart-widget.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/chart-widget.ts:1)

Kind: `support`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-124`. Evidence: pending.

**Protects**

- Unadmitted adapter/companion prototype implementation exercised by associated Bun test declarations; must not be mistaken for production Can coverage.

**Current observations:** fake host or protocol implementation

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** NATIVE DOC

**Proposed Can replacement:** Retain as historical prototype only with explicit role, or replace any admitted production behavior with reviewed native mechanics and Can-owned tests.

**Old harness may be deleted when:** After prototype admission/retirement decision, no active caller, and migration of any still-current semantic obligation.

**Notes:** Source under tests is executable implementation, not a passive JSON fixture.

## HISTORY-125

**clipboard.ts** — [tests/host-discrimination/companions/clipboard.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/clipboard.ts:1)

Kind: `support`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-125`. Evidence: pending.

**Protects**

- Unadmitted adapter/companion prototype implementation exercised by associated Bun test declarations; must not be mistaken for production Can coverage.

**Current observations:** fake host or protocol implementation

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** NATIVE DOC

**Proposed Can replacement:** Retain as historical prototype only with explicit role, or replace any admitted production behavior with reviewed native mechanics and Can-owned tests.

**Old harness may be deleted when:** After prototype admission/retirement decision, no active caller, and migration of any still-current semantic obligation.

**Notes:** Source under tests is executable implementation, not a passive JSON fixture.

## HISTORY-126

**storage.ts** — [tests/host-discrimination/companions/storage.ts](/Users/vince/Projects/can-lang/tests/host-discrimination/companions/storage.ts:1)

Kind: `support`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-HISTORY-126`. Evidence: pending.

**Protects**

- Unadmitted adapter/companion prototype implementation exercised by associated Bun test declarations; must not be mistaken for production Can coverage.

**Current observations:** fake host or protocol implementation

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** NATIVE DOC

**Proposed Can replacement:** Retain as historical prototype only with explicit role, or replace any admitted production behavior with reviewed native mechanics and Can-owned tests.

**Old harness may be deleted when:** After prototype admission/retirement decision, no active caller, and migration of any still-current semantic obligation.

**Notes:** Source under tests is executable implementation, not a passive JSON fixture.

## HISTORY-127

**"qualified native APIs and behaviors"** — [tests/conformance/native.test.ts](/Users/vince/Projects/can-lang/tests/conformance/native.test.ts:9)

Kind: `bun_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-127`. Evidence: pending.

**Protects**

- Pinned runtime identity, every required native API and behavior probe together produce no failures.

**Current observations:** mutated global descriptors or identity and qualification report

**Variants:** None recorded.

**Environment/selection gates:** pinned Bun runtime/native host

**Capabilities:** SUITE NATIVE FAULT ENV BUILD

**Proposed Can replacement:** Can selects each missing-API/identity fault, obtains raw native qualification observations, and checks failure reason and positive result in Can.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/conformance/native.ts](/Users/vince/Projects/can-lang/tests/conformance/native.ts); [distribution/target.json](/Users/vince/Projects/can-lang/distribution/target.json); [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py)

**Notes:** Loop-generated Bun declaration; parameterized title must remain explicit in eventual coverage. No Can wrapper around bun test qualifies.

## HISTORY-128

**`refuse missing ${name} without fallback`** — [tests/conformance/native.test.ts](/Users/vince/Projects/can-lang/tests/conformance/native.test.ts:13)

Kind: `bun_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-128`. Evidence: pending.

**Protects**

- Removing each required native API descriptor makes qualification fail with the exact missing API reason, without fallback.

**Current observations:** mutated global descriptors or identity and qualification report

**Variants:** JSON.rawJSON Array.fromAsync node:util.types.isProxy node:util.types.isNativeError Bun.Transpiler.prototype.scanImports Bun.Transpiler.prototype.transformSync

**Environment/selection gates:** pinned Bun runtime/native host

**Capabilities:** SUITE NATIVE FAULT ENV BUILD

**Proposed Can replacement:** Can selects each missing-API/identity fault, obtains raw native qualification observations, and checks failure reason and positive result in Can.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/conformance/native.ts](/Users/vince/Projects/can-lang/tests/conformance/native.ts); [distribution/target.json](/Users/vince/Projects/can-lang/distribution/target.json); [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py)

**Notes:** Loop-generated Bun declaration; parameterized title must remain explicit in eventual coverage. No Can wrapper around bun test qualifies.

## HISTORY-129

**`refuse different ${key}`** — [tests/conformance/native.test.ts](/Users/vince/Projects/can-lang/tests/conformance/native.test.ts:29)

Kind: `bun_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-129`. Evidence: pending.

**Protects**

- Changing each runtime identity field makes qualification fail with field-specific mismatch.

**Current observations:** mutated global descriptors or identity and qualification report

**Variants:** name version revision sha256 platform architecture

**Environment/selection gates:** pinned Bun runtime/native host

**Capabilities:** SUITE NATIVE FAULT ENV BUILD

**Proposed Can replacement:** Can selects each missing-API/identity fault, obtains raw native qualification observations, and checks failure reason and positive result in Can.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/conformance/native.ts](/Users/vince/Projects/can-lang/tests/conformance/native.ts); [distribution/target.json](/Users/vince/Projects/can-lang/distribution/target.json); [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py)

**Notes:** Loop-generated Bun declaration; parameterized title must remain explicit in eventual coverage. No Can wrapper around bun test qualifies.

## HISTORY-130

**"refuse missing AsyncLocalStorage without fallback"** — [tests/conformance/native.test.ts](/Users/vince/Projects/can-lang/tests/conformance/native.test.ts:36)

Kind: `bun_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-130`. Evidence: pending.

**Protects**

- Injected missing AsyncLocalStorage availability makes qualification fail closed.

**Current observations:** mutated global descriptors or identity and qualification report

**Variants:** None recorded.

**Environment/selection gates:** pinned Bun runtime/native host

**Capabilities:** SUITE NATIVE FAULT ENV BUILD

**Proposed Can replacement:** Can selects each missing-API/identity fault, obtains raw native qualification observations, and checks failure reason and positive result in Can.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/conformance/native.ts](/Users/vince/Projects/can-lang/tests/conformance/native.ts); [distribution/target.json](/Users/vince/Projects/can-lang/distribution/target.json); [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py)

**Notes:** Loop-generated Bun declaration; parameterized title must remain explicit in eventual coverage. No Can wrapper around bun test qualifies.

## HISTORY-131

**native qualification implementation** — [tests/conformance/native.ts](/Users/vince/Projects/can-lang/tests/conformance/native.ts:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-HISTORY-131`. Evidence: pending.

**Protects**

- Checks exact Bun identity, executable hash/layout and minimum OS; required API presence and behavior probes include async context, raw JSON token/negative zero, ordered fromAsync, nonthenable boxing, strict equality, immutable arrays, Unicode, fatal UTF-8, set order and I/O handles.
- Current file contains test-specific assertions and computes passed/failures report.

**Current observations:** direct Bun/Node observations and in-script assert exceptions

**Variants:** manifest API list behavior probes OS/platform/architecture

**Environment/selection gates:** distribution/target.json qualified Bun runtime offline isolation

**Capabilities:** SUITE NATIVE ENV FILES FAULT

**Proposed Can replacement:** Can-authored qualification case supplies vectors/expected behavior and verdict; narrow native probe mechanics return raw traces for values ordinary Can cannot construct; provenance and executable digest remain independently observed.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py); [distribution/target.json](/Users/vince/Projects/can-lang/distribution/target.json)

**Notes:** This implementation is a host harness, not merely a fixture. Its full executable policy must move to Can.

## HISTORY-132

**README.md** — [tests/authoring-policies/README.md](/Users/vince/Projects/can-lang/tests/authoring-policies/README.md:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-132`. Evidence: pending.

**Protects**

- Registered authoring-policy comparison inputs and recorded baseline/reference evidence; agent trials explicitly remain unrun.

**Current observations:** documents/registry entries

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as historical record and map any currently enforced policy to a new Can-authored case; never manufacture trial passes.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Comparison result is not a current Can suite result.

## HISTORY-133

**registry.json** — [tests/authoring-policies/registry.json](/Users/vince/Projects/can-lang/tests/authoring-policies/registry.json:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-133`. Evidence: pending.

**Protects**

- Registered authoring-policy comparison inputs and recorded baseline/reference evidence; agent trials explicitly remain unrun.

**Current observations:** documents/registry entries

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as historical record and map any currently enforced policy to a new Can-authored case; never manufacture trial passes.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Comparison result is not a current Can suite result.

## HISTORY-134

**x-r08-comparison-2026-09-26.md** — [tests/authoring-policies/x-r08-comparison-2026-09-26.md](/Users/vince/Projects/can-lang/tests/authoring-policies/x-r08-comparison-2026-09-26.md:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-134`. Evidence: pending.

**Protects**

- Registered authoring-policy comparison inputs and recorded baseline/reference evidence; agent trials explicitly remain unrun.

**Current observations:** documents/registry entries

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as historical record and map any currently enforced policy to a new Can-authored case; never manufacture trial passes.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Comparison result is not a current Can suite result.

## HISTORY-135

**README.md** — [tests/authoring-policies/candidates/README.md](/Users/vince/Projects/can-lang/tests/authoring-policies/candidates/README.md:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-135`. Evidence: pending.

**Protects**

- Registered authoring-policy comparison inputs and recorded baseline/reference evidence; agent trials explicitly remain unrun.

**Current observations:** documents/registry entries

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as historical record and map any currently enforced policy to a new Can-authored case; never manufacture trial passes.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Comparison result is not a current Can suite result.

## HISTORY-136

**main.can** — [tests/authoring-policies/boolean/baseline/src/main.can](/Users/vince/Projects/can-lang/tests/authoring-policies/boolean/baseline/src/main.can:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-136`. Evidence: pending.

**Protects**

- boolean baseline Can source is a comparison subject for authoring-policy baseline/reference leg.

**Current observations:** Can source text and compiled behavior where current comparison actually ran

**Variants:** boolean baseline

**Environment/selection gates:** None recorded.

**Capabilities:** CAN DOC

**Proposed Can replacement:** Can case may compile/inspect this source if the comparison remains current; otherwise preserve with historical report.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No executable runner exists in this family and agent trial results are unrun.

## HISTORY-137

**main.can** — [tests/authoring-policies/boolean/reference/src/main.can](/Users/vince/Projects/can-lang/tests/authoring-policies/boolean/reference/src/main.can:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-137`. Evidence: pending.

**Protects**

- boolean reference Can source is a comparison subject for authoring-policy baseline/reference leg.

**Current observations:** Can source text and compiled behavior where current comparison actually ran

**Variants:** boolean reference

**Environment/selection gates:** None recorded.

**Capabilities:** CAN DOC

**Proposed Can replacement:** Can case may compile/inspect this source if the comparison remains current; otherwise preserve with historical report.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No executable runner exists in this family and agent trial results are unrun.

## HISTORY-138

**main.can** — [tests/authoring-policies/locals/baseline/src/main.can](/Users/vince/Projects/can-lang/tests/authoring-policies/locals/baseline/src/main.can:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-138`. Evidence: pending.

**Protects**

- locals baseline Can source is a comparison subject for authoring-policy baseline/reference leg.

**Current observations:** Can source text and compiled behavior where current comparison actually ran

**Variants:** locals baseline

**Environment/selection gates:** None recorded.

**Capabilities:** CAN DOC

**Proposed Can replacement:** Can case may compile/inspect this source if the comparison remains current; otherwise preserve with historical report.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No executable runner exists in this family and agent trial results are unrun.

## HISTORY-139

**main.can** — [tests/authoring-policies/locals/reference/src/main.can](/Users/vince/Projects/can-lang/tests/authoring-policies/locals/reference/src/main.can:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-139`. Evidence: pending.

**Protects**

- locals reference Can source is a comparison subject for authoring-policy baseline/reference leg.

**Current observations:** Can source text and compiled behavior where current comparison actually ran

**Variants:** locals reference

**Environment/selection gates:** None recorded.

**Capabilities:** CAN DOC

**Proposed Can replacement:** Can case may compile/inspect this source if the comparison remains current; otherwise preserve with historical report.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No executable runner exists in this family and agent trial results are unrun.

## HISTORY-140

**main.can** — [tests/authoring-policies/near/baseline/src/main.can](/Users/vince/Projects/can-lang/tests/authoring-policies/near/baseline/src/main.can:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-140`. Evidence: pending.

**Protects**

- near baseline Can source is a comparison subject for authoring-policy baseline/reference leg.

**Current observations:** Can source text and compiled behavior where current comparison actually ran

**Variants:** near baseline

**Environment/selection gates:** None recorded.

**Capabilities:** CAN DOC

**Proposed Can replacement:** Can case may compile/inspect this source if the comparison remains current; otherwise preserve with historical report.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No executable runner exists in this family and agent trial results are unrun.

## HISTORY-141

**main.can** — [tests/authoring-policies/near/reference/src/main.can](/Users/vince/Projects/can-lang/tests/authoring-policies/near/reference/src/main.can:1)

Kind: `support`. Proposed disposition: `retain_fixture`. Replacement obligation: `CAN-HISTORY-141`. Evidence: pending.

**Protects**

- near reference Can source is a comparison subject for authoring-policy baseline/reference leg.

**Current observations:** Can source text and compiled behavior where current comparison actually ran

**Variants:** near reference

**Environment/selection gates:** None recorded.

**Capabilities:** CAN DOC

**Proposed Can replacement:** Can case may compile/inspect this source if the comparison remains current; otherwise preserve with historical report.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** No executable runner exists in this family and agent trial results are unrun.

## HISTORY-142

**x-r02-1.md** — [tests/browser-controls/x-r02-1.md](/Users/vince/Projects/can-lang/tests/browser-controls/x-r02-1.md:1)

Kind: `historical`. Proposed disposition: `preserve_record`. Replacement obligation: `CAN-HISTORY-142`. Evidence: pending.

**Protects**

- Historical browser-control experiment records checker/build observations; it does not supply live browser matrix evidence.

**Current observations:** recorded document

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Preserve as historical context; current browser control scenarios are mapped in the browser ledger fragment.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Notes:** Must not retire current tests/integration/browser_controls_test.go through this record.

## HISTORY-143

**distribution native qualification caller** — [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py:67)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-HISTORY-143`. Evidence: pending.

**Protects**

- Stages pinned runtime, chooses network isolation, invokes native.test.ts and native.ts, checks exit/report and writes overall qualification verdict.

**Current observations:** archive hash, executable hash, subprocess statuses and qualification JSON

**Variants:** darwin sandbox linux unshare or caller isolation

**Environment/selection gates:** pinned upstream archive distribution/target.json

**Capabilities:** SUITE NATIVE ENV FILES WORK

**Proposed Can replacement:** CI/launcher retains necessary isolation/provisioning and propagates Can suite result; all case-specific qualification sequencing, comparisons and verdict aggregation move to Can.

**Old harness may be deleted when:** After the named obligations have passing Can-owned evidence, negative controls and required environments are covered, all active callers are rewired, and the old host harness is shown unused.

**Related source:** [tests/conformance/native.ts](/Users/vince/Projects/can-lang/tests/conformance/native.ts); [tests/conformance/native.test.ts](/Users/vince/Projects/can-lang/tests/conformance/native.test.ts)

**Notes:** External script is in scope because it supplies /tests conformance coverage; cannot just point it at new files while keeping the oracle.

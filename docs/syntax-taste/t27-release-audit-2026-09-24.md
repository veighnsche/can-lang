# T27 release audit: DI dispositions, gate evidence and supported recommendation

**Status correction after the upgrade (24 September 2026):** this is the
historical T27 audit, followed below by its original evidence and verdicts.
Use the [current reconciliation](post-upgrade-reconciliation-2026-09-24.md) and
updated [T26 product guide](t26-product-guide-2026-09-24.md) for present status.

- The preparation directory was absent from the T27 commit but was committed
  in `961f921`. Its [selection ledger](preparation/design-disposition-ledger.md)
  now supplies DI-11 browser scope, DI-13 companion effect tests, DI-14 SQL
  boundaries and DI-16 Linux target decisions. The original “unattested” rows
  below describe the audit's then-available inputs; they are not current
  absence or rejection decisions.
- The [final acceptance follow-up](can-implementation-task-list-2026-09-24.md#final-acceptance-2026-09-24-head-5d96ac9)
  records a repaired test-local timeout and passing installed-artifact Linux
  Gate 4 rerun under emulation. The earlier failure and requirement for a
  native (non-emulated) linux/amd64 run below are superseded as descriptions
  of that blocker. No new native-host run is claimed here.
- Chromium/WebKit passes establish the tested Can-authored grid path with
  its route rewrite and test bundler/shims. They do not qualify general
  browser runtime delivery or fulfill shared action linkage. Firefox is not
  qualified. SQLite is implemented and exercised by the examples; the original
  blanket exclusion of non-PostgreSQL databases below is too broad.
- Shared request-aware actions, supported browser delivery and specified UI
  observations remain unfinished accepted requirements. Original gate PASS
  entries are preserved as historical results, not as closure of these gaps.

24 September 2026. Final release audit for the
[implementation task list](can-implementation-task-list-2026-09-24.md)
(T01–T27). This document records the disposition of every DI
inventory row, maps each accepted item to its tests and docs, shows
that no deferred mechanism slipped into the implementation, links
the Gate 1–5 evidence, and publishes the supported recommendation
scoped strictly to the platforms and browser versions that passed.

It changes no normative contract: where this audit and
[decisions](decisions.md), [technical-spec](technical-spec.md),
[AI/I/O](ai-io-spec.md), [coordination](coordination-spec.md),
[platform/testing](platform-testing-spec.md) or the
[T26 product guide](t26-product-guide-2026-09-24.md) disagree, the
contract wins and the disagreement is a defect to report. Nothing
here is inferred from Jev advice; the consultations in
[jev-consultations](jev-consultations/findings.md) are advisory and
§6 records them as such.

The machine-checked companion is
`tests/baseline/t27_release_audit_test.go`: it pins the inventory
table below, the accepted-to-test/doc map, the deferred-absence
scans, generated-artifact freshness, gate-evidence links and the
recommendation scope. All commands in §8 ran at the audit commit.

## 1. DI inventory dispositions

The following is the original audit table. Read it with the status correction
above; current reconciliation uses the subsequently committed preparation inputs.

Source of truth is the committed task list (its per-task DI grants
plus its "Deliberately outside this implementation graph"
section). The `preparation/` packet directory referenced by the
task list was never committed, so four identifiers (DI-11, DI-13,
DI-14 base, DI-16) have no committed description; they are
recorded as unattested, own no build task, and are held
not-accepted — any future use needs a new disposition first.

| ID | Disposition | Implementing task / note |
| --- | --- | --- |
| DI-01 | accepted | T02 `bind` capture at every depth |
| DI-02 | accepted | T05 `owner record` |
| DI-03 | deferred | finite error-set parameters; no mechanism shipped |
| DI-04 | accepted | T07 exported generics under symbolic type variables |
| DI-05a | accepted | T03 canonical package-instance graph, `uses` |
| DI-05b | accepted | T04 unnumbered errors, `can.error.v2` identity |
| DI-06 | accepted | T08 same-owner fixtures, `scenario` links |
| DI-07 | accepted | T06 extensional closed-variant leaf sets |
| DI-08 | deferred | capture-binding syntax; no mechanism shipped |
| DI-09 | accepted | T11 typed path captures, route table |
| DI-09b | accepted | T12 keyed-row forms, HTML action adapter |
| DI-10 | unselected | declarative markup and typed DOM-target scopes; safe builders retained instead |
| DI-11 | unattested | no committed description, no task, no mechanism; not accepted |
| DI-12 | retained | T18 keeps selected race/ownership semantics |
| DI-13 | unattested | no committed description, no task, no mechanism; not accepted |
| DI-14 | unattested | base item undescribed in committed sources; only DI-14a is named; not accepted |
| DI-14a | deferred | mutation `RETURNING`; checker rejects it |
| DI-15 | retained | T18 keeps selected shutdown semantics |
| DI-16 | unattested | no committed description, no task, no mechanism; not accepted |
| DI-17 | deferred | native AI wrapper/batch/callable forms; T26 candidate slot records the deferral |
| DI-18 | retained | current escape semantics kept by T18; static escape/consumed-fault additions deferred |
| DI-19 | deferred | unrestricted SDK imports; declared imports only |
| DI-20 | deferred | bulk immutable collections; no catalogue op shipped |
| DI-21 | deferred | billing/calendar; later policy-explicit library, not a gate blocker |
| DI-22 | deferred | named-field constructors; no mechanism shipped |
| DI-23 | deferred | multiline grammar; triple-quote form rejected by the parser |

## 2. Accepted and retained items map to tests and docs

Each row names one proving test and one proving doc or
implementation anchor; the audit test checks every cell exists.

| ID | Test | Doc / implementation anchor |
| --- | --- | --- |
| DI-01 | `TestPatternBindCapturesAtEveryDepth` (`compiler/internal/check/pattern_bind_test.go`) | `type BindPattern struct` in `compiler/internal/syntax/ast.go` |
| DI-02 | `TestOwnerRecordBoundaryPositives` (`compiler/internal/check/owner_record_test.go`) | T26 guide §5 "owner-only" boundary |
| DI-04 | `TestExportedGenericStructuralComposites` (`compiler/internal/check/exported_generics_test.go`) | `compiler/internal/check/exported_generics.go` |
| DI-05a | `TestQualifiedImportsRejectUndeclaredAndPrivate` (`compiler/internal/resolve/instances_test.go`) | `Dependency *Token` in `compiler/internal/syntax/ast.go` |
| DI-05b | `TestFormerNumericCollisionComposesWithQualifiedIdentity` (`compiler/internal/project/error_identity_test.go`) | `ReportIdentityVersion = "can.error.v2"` in `compiler/internal/project/lock.go` |
| DI-06 | `TestScenarioLinkResolves` (`compiler/internal/check/scenario_test.go`) | `Scenario *Token` in `compiler/internal/syntax/ast.go`; decisions.md DI-06 section |
| DI-07 | `TestGenericVariantLeafSetCompatibility` (`compiler/internal/types/types_test.go`) | "extensional named leaf sets" in `compiler/internal/types/compatibility.go` |
| DI-09 | `runtime/test/action-routes.test.ts` (13 pass) | T26 guide §2.3 captures/dispatch |
| DI-09b | `runtime/test/form-rows.test.ts` (12 pass) | T26 guide §2.2 `lines_order` adapter |
| DI-12 | `runtime/test/shutdown.test.ts` (6 pass) | `docs/implementation/shutdown.md` race/ownership policy |
| DI-15 | `runtime/test/shutdown.test.ts` (6 pass) | `docs/implementation/shutdown.md` exactly-once close |
| DI-18 | `runtime/test/shutdown.test.ts` (6 pass) | `docs/implementation/shutdown.md`; no `finally` keyword in `compiler/internal/syntax` |

JSON POST-save / bodyless GET-load (T13) and the browser target
(T21–T24) are qualification work under the integrated action
contract rather than numbered DI rows; their evidence is the
`runtime/test/action-json.test.ts` suite (14 pass) and the Gate 3/5
suites in §5.

## 3. No deferred mechanism slipped in

Each check below is executed by the audit test, not eyeballed:

- Numbered errors (DI-05b removal): no `\berror\s+[0-9]`
  declaration survives in any `.can` file or non-test Go file
  under `compiler/`, `examples/`, `std/` or `tests/`. The only two
  matches repo-wide sit in rejection tests:
  `TestFileParserRejectsObsoleteAndMalformedGrammar` and
  `TestChecksRejectsNumberedRedeclaration`.
- Mutation `RETURNING` (DI-14a): the checker pins the rejection —
  `compiler/internal/sql/cardinality.go` reports `RETURNING is not
  admitted`.
- Consumed-fault channel (DI-18 additions): no `finally` keyword
  in non-test `compiler/internal/syntax` sources.
- Declarative markup (DI-10): no `Markup` AST node in
  `compiler/internal/syntax/ast.go`.
- Bulk collections, billing/calendar, markup catalogue surface
  (DI-20, DI-21, DI-10): no `name` or `identity` in any
  `packages`/`types`/`errors`/`operations` entry of
  `compiler/internal/catalogue/catalogue.json` matches
  bulk/billing/calendar/markup (the word "bulk" occurs only in
  text-normalization adapter prose). `std/` ships no
  billing/calendar package.
- Multiline grammar (DI-23): the parser rejection suite
  `TestFileParserRejectsObsoleteAndMalformedGrammar` pins the
  triple-quote rejection.
- Declared imports only (DI-19): undeclared transitive and
  private imports fail per
  `TestQualifiedImportsRejectUndeclaredAndPrivate`.
- Native AI forms (DI-17): `tests/baseline/candidates/T26-native-ai/`
  still records `deferred` in both `README.md` and
  `candidate.json`, with the current idiom retained.
- DI-03, DI-08, DI-22: no implementing task exists in the graph
  and no syntax, checker or catalogue surface implements them;
  the inventory (§1) records the deferral.

## 4. Generated artifacts are fresh

The catalogue source of truth
(`compiler/internal/catalogue/catalogue.json`) regenerates
byte-identical outputs: `cataloguegen --check` passes, covering
`compiler/internal/catalogue/generated.go` and the generated
`runtime/catalogue.ts` (both `DO NOT EDIT`). The audit test runs
this check. No other generator exists in the tree (no
`go:generate` directives outside catalogue docs).

## 5. Gate 1–5 evidence

| Gate | Commit | Evidence | Status |
| --- | --- | --- | --- |
| Gate 1/2 core contract integration | `9fcc1ae` T09 | `compiler/internal/check/core_integration_test.go`, `compiler/internal/emit/core_integration_test.go`, `compiler/internal/syntax/core_integration_test.go` — owner across generic/package seam, variant leaf across imports, fixture link after alias rename, error identity after lineage change | PASS, reproducible via `go test ./compiler/...` |
| Gate 3 server product matrix | `7333359` T17 | `tests/integration/gate3_matrix_test.go` — staged `CAN_BUN_ARCHIVE=… go test ./tests/integration/ -run TestGate3` 3/3 PASS in 128s on darwin/arm64 Bun 1.4.2; known limitation: rebuilt page still posts to the old path (stale post answers 404) | PASS on darwin/arm64; skips without `CAN_BUN_ARCHIVE` by design |
| Gate 4 fault suites | `3b1306a` T20 | `tests/integration/gate4_fault_test.go` — staged darwin PASS in 80s (invoice 176/176, webhook 67/67, stream 7/7, shutdown suites 71 pass); installed-artifact Linux run (Debian 13 amd64, `--network none`, emulated): invoice/webhook/stream legs PASS but the shutdown-suite leg FAILS at pre-existing `runtime/test/transport-late.test.ts` (25ms budget races fetch+read+decode under emulation; passes with 2000ms, in isolation, and 5/5 on darwin — harness timing, not product) | NOT fully qualified: needs a native (non-emulated) linux/amd64 run |
| Gate 5 frontend gate | `6aedb0f` T25 | `tests/integration/gate5_frontend_test.go` plus `tests/integration/browser/grid.mjs`, `tests/integration/browser/build-grid.mjs`, `tests/integration/browser/sha256-shim.mjs` — staged 2/2 PASS in 412s: qualified chromium 140.0.7339.186 and webkit 26.0 (27 checks each); firefox unavailable (Playwright launcher timeout, best-effort per design) | PASS on chromium/webkit; firefox not qualified |

Cross-cutting: `go test ./...` all ok, `gofmt`/`go vet` clean,
`bun run check:runtime` (lint, format, typecheck) clean at the
audit commit — see §8 for the exact commands and results. The T01
freeze (`tests/baseline/baseline.json`,
`tests/baseline/registry.json`) still anchors the toolchain and
the held-out registry with no revisions.

## 6. Consultations and agent comparisons

Jev SystemOne consultations
(`docs/syntax-taste/jev-consultations/`, three fresh requests to
`jev-latest` answered by `jev-1.13.0`) remain advisory
classifications: unanimous 1.00 selections for single locked
integration (T09), same-transaction ledger (T14) and strict
transitive browser closure (T21/T22), investigated in
`docs/syntax-taste/jev-consultations/findings.md` as a coherence
check against the task list's own acceptance rules. The retained engineering positions follow from
the failure-mode arguments and are demonstrated by the §5 gate
suites, not by the vote count. The consultation changed no
task-list selection.

Held-out AI-agent creation/refactor/repair comparisons ran
mechanically under the T01 protocol
(`tests/baseline/reports/t26-agent-comparison-2026-09-24.md`): no
live model trials, tokens reported only where measured, no
significance claimed. The native-AI candidate stays deferred per
§3.

## 7. Supported recommendation

The recommendation below is the original T27 verdict. Its Linux blocker,
database exclusion and general browser implication are superseded by the
status correction above and the current reconciliation.

Only the platforms and browser versions that passed the §5 gates
are recommended. Anything else is explicitly out of scope until it
passes its gate.

### 7.1 Narrower server-driven path (recommended)

Server-rendered HTMX invoice form (§2.2 of the T26 guide) plus
JSON POST-save / bodyless GET-load actions, on:

- Bun 1.4.2 (`744846f844374847c902b5e7fd59b4342a51ef99`),
  development target `bun-1.4.2-darwin-arm64-v1`
  (darwin/arm64, archive sha256
  `90987a3a16d7db556d886ac3d551e7b6d3edf0a1cf43acaed622e8676be1d12f`).
- Go minimum 1.25; TypeScript leg pinned by `package.json`.
- PostgreSQL 17.11 where a server database is required.

Linux target `bun-1.4.2-linux-amd64-v1` (Debian 13, glibc 2.41)
is pinned and reproducible but NOT yet recommended: Gate 4
needs a native (non-emulated) linux/amd64 run (§5). An emulated
run cannot fully qualify Gate 4.

### 7.2 Broader Can-authored path (recommended, qualified browsers only)

The Can-authored invoice grid (`examples/invoice-grid/`,
`canlc build --target browser`) against the §7.1 server, only on:

- chromium 140.0.7339.186 — qualified (27 checks).
- webkit 26.0 — qualified (same 27 checks).

Firefox is not qualified and not recommended. The grid keeps an
in-view draft while offline with explicit identical-ID replay or
authorized reconciliation; there is no reload durability and no automatic queue.
No worker profile, no authored JS/TS in browser
code, no server secrets/SQL/process access in browser code
(compile-time transitive closure).

### 7.3 Explicitly out of scope

Firefox, non-PostgreSQL databases, database migrations, offline
reload durability, worker execution, arbitrary authored JS/TS,
raw markup/scripts, all §1 deferred mechanisms, and any platform
or browser version not named in §7.1–§7.2.

## 8. Audit commands

Run at the audit commit on go1.27.1 darwin/arm64 with Bun 1.4.2:

- `go test ./tests/baseline/ -v -count=1` — audit suite plus T01/T26 suites PASS.
- `go test ./...` — all packages ok, 0 fail.
- `gofmt -l` on touched files and `go vet ./tests/baseline/` — clean.
- `go run ./compiler/internal/catalogue/cmd/cataloguegen --check` — clean (also asserted by the audit test).
- `bun run check:runtime` — lint, format check and typecheck clean.

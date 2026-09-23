# Capability admission and evolution guide — 2026-09-22

Status: current. This is the single linked guide for proposing, admitting,
testing, evolving, and retiring Can capabilities (LD33/AE33). There is no
backwards-compatibility promise: admission never freezes an operation's
contract, and retirement removes entries outright. There is no
project-authored foreign-binding mechanism; every callable operation resolves
to the closed catalogue.

## 1. Admission checklist (the review gate)

A proposal is admitted only when every row has a linked artifact. A typed
signature alone is an incomplete proposal and fails review; see
`examples/2026-09-22/lf19/cases/incomplete-proposal/PROPOSAL.md` and §6.

| # | Requirement | Artifact |
| --- | --- | --- |
| 1 | Complete Can signature: name, inputs, generic constraints, result, emits | Catalogue entry in `compiler/internal/catalogue/catalogue.json` |
| 2 | Native mapping: named Bun/JS primitive plus the minimal adapter | Emit-table row (`compiler/internal/emit/program.go`) and runtime adapter |
| 3 | Error translation: every native failure mode mapped to a declared emit or a total contract | Catalogue `emits` + adapter code + negative fixture rows |
| 4 | Immutability/lifetime: what is frozen, who owns handles, close/drain rules | Guide section + opaque-type entry (`constructible: false` for handles) |
| 5 | Fixture and conformance: substitution rows, raw-adapter evidence, target evidence, kept separate | `when`/`using raw` fixtures plus staged tests; evidence labels in assert reports |
| 6 | Distribution/registry update | Regenerated `std/catalogue` mirror (`cataloguegen --check` green) and error-registry entries |
| 7 | Retirement note | Named removal path: entry deletion plus error-kind retirement |

Reproduce the bounded checks with the LF19 corpus:

```sh
python3 docs/implementation/examples/2026-09-22/lf19/run.py \
  --bundle /path/to/can-<version>-<target> --out results.json
```

## 2. Worked pure capability: `text::from_int`

Catalogue identity `can.std.text@1::from_int` (`compiler/internal/catalogue/catalogue.json`).

- Signature: `fn str from_int(int value)`, `emits []`, no generics, no callbacks.
- Native mapping: `String` over the `bigint` argument. The emit table binds
  the identity to `$canNumbers.fromInt` (`compiler/internal/emit/program.go`),
  implemented as `success(String(value))` (`runtime/number.ts`). The adapter is
  empty beyond the call: no coercion at source boundaries, because the checker
  only admits `int` inputs (mixed `int + float` is rejected: `operator +
  requires identical operand types`).
- Error translation: total function; there are no failure modes to translate.
- Immutability/lifetime: pure. The `bigint` input is a value; no state, no
  handle, no lifetime beyond the call.
- Fixture/conformance: `cases/pure-capability` runs 3 rows
  (`9007199254740993`, `-42`, `0`) with `real-can` evidence — genuine execution,
  not a mock. The maintained `std/scalars` project covers the same operation in
  the staged stdlib suite.
- Registry artifacts: catalogue entry (`assertion: real`, ref `C6`), the
  generated `runtime/catalogue.ts` mirror, and the `std/catalogue` package
  mirror (revision 1, target `bun-1.4.2-darwin-arm64-v1`).

## 3. Worked resource capability: `sql::pool_open` / `sql::query_one` / `sql::pool_close`

Catalogue identities `can.std.sql@1::pool_open`, `can.std.sql@1::query_one`,
`can.std.sql@1::pool_close`.

- Signatures: `pool_open(str connection_variable, int max_connections) ->
  sql::pool` emitting `[http::credentials_missing, sql::connection_failed]`;
  `query_one<P: sql_parameters, R: sql_row>(sql::pool, str descriptor, P) -> R`
  with a static descriptor input; `pool_close(sql::pool, int timeout_ms) ->
  void` emitting `[sql::close_failed]`.
- Native mapping: `Bun.SQL` pools (`runtime/platform/sql.ts`, `bigint: true`,
  validated max), `Bun.SQL` tagged templates with parser-derived static
  segments and server-side binds, and `Bun.SQL.close` with lease drain.
- Error translation: missing credentials normalize to
  `http::credentials_missing`; connection, query, constraint, row-shape, and
  close failures map to the declared `sql::*` emits. No raw provider error
  escapes untranslated.
- Immutability/lifetime: `sql::pool` is opaque and non-constructible, so only
  `pool_open` mints handles and no source literal can forge one. The owner
  closes with `pool_close`; a timed-out close stays owned rather than leaking.
  `cases/forged-pool` shows the checker rejecting a record passed as a handle
  (`expression type does not fit expected type`).
- Fixture/conformance, kept separate: `cases/resource-capability` runs offline
  with `when` substitution rows and reports mixed `real-can` +
  `supplied-completion` evidence — the project logic really executes while the
  PostgreSQL answers are supplied. Target conformance is the live suite in
  `tests/integration/sql_test.go`, gated on `CAN_TEST_POSTGRES_URL`; supplied
  rows are never presented as target verification.
- Registry artifacts: catalogue entries (`assertion: supplied`, refs `P6`,
  `P12`), the `std/catalogue` mirror, and the manifest `sql` descriptor
  contract (`compiler/internal/project/manifest.go`: postgresql-only,
  named parameters, fully qualified nominal types, trailing limit parameter).

## 4. Rejection boundary

- Unknown catalogue operation: `cases/unknown-operation` calls `text::bogus`
  and the checker reports `no eligible call text::bogus`. Catalogue-only
  access means anything absent from `catalogue.json` is uncallable.
- Unsupported project-authored native binding: `cases/authored-binding`
  declares a `connection` with protocol `acme_custom_v1` and the checker
  reports `unknown protocol`. Only the two closed profiles
  (`typesafe_systemone_v1`, `openai_responses_v1`) exist
  (`compiler/internal/check/connections.go`); projects cannot add bindings.

## 5. Evolution and retirement

- Add an operation by adding a catalogue entry plus every checklist artifact,
  then regenerate mirrors with `go run
  ./compiler/internal/catalogue/cmd/cataloguegen` and verify with `--check`
  (Go tests reject stale mirrors).
- Change an operation by changing its entry and adapters together; no
  compatibility alias is kept. Consumers re-check against the new contract.
- Retire an operation by deleting its catalogue entry, moving its error kinds
  from `active` to `retired` in the error registry, regenerating mirrors, and
  rerunning the gates. Retired names stay rejected like any unknown operation.

## 6. Incomplete-proposal classification

`cases/incomplete-proposal/PROPOSAL.md` proposes `text::word_count` with a
typed signature only. Against the checklist: rows 2–7 have no artifacts, so
the verdict is **incomplete** and the proposal fails the review gate. This is a
documentation review outcome, not a compiler test: no manifest or binding
mechanism is invented to express the missing evidence.

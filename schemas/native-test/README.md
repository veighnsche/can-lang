# Native-test trust and evidence wire schemas (version 1)

Status: **frozen by P01**. These schemas pin the wire fields shared by the
reference toolchain, native owner, Can suite and qualification consumers.
They do not implement any component or qualify any behavior.

## Roles

| Role | Meaning | Identified by |
| --- | --- | --- |
| `R` | Reference toolchain: compiler, emitter, runtime/catalogue, Bun, trusted bindings used to build and run the suite | Compiler/runtime/Bun/binding digests in the acceptance manifest |
| `N` | Native infrastructure: outer supervisor and owned resource/driver services | Executable digest, source/build provenance, capability record |
| `S` | Suite source: Can source, fixtures, options, coverage manifest, expected vectors | Source closure and snapshot digests |
| `A` | Suite artifact: the R-built immutable nonpublishing generation | Generated artifact digest |
| `C` | Candidate: compiler/runtime/inputs under test and the subject artifacts it produces | Compiler, runtime and subject artifact digests |
| `T` | Transitional bootstrap parent/witness for first R/N acceptance | Executable digest; outside every killable test subtree |
| `observer` | Independent observation probe (native, browser host-effect, DB) | Observer implementation digest plus the runtime observed |

R builds the judges; C produces subjects only. N enforces limits, owns
lifetimes and reports execution/cleanup facts; it never authors scenarios,
expectations or verdicts. Can owns plans, scenarios, comparisons and reports.
Candidate code never enters the trusted controller/worker realm.

## Execution versus qualification modes

Every scoped document carries both fields:

- `mode`: `execute` runs cases; `list` and `plan` are nonexecuting and supply
  no execution or qualification receipt.
- `purpose`: `execution` is a development run with explicit trust limits;
  `qualification` claims acceptance evidence and additionally requires
  accepted R/N/S/A identities, complete observations and clean receipts.

A `qualification` purpose never downgrades missing identities, partial
evidence or failed cleanup into a pass. Downstream tasks bind the exact
accepted manifests; these schemas only reserve the fields.

## Files

| Schema | Validates |
| --- | --- |
| `identity.schema.json` | One trusted identity: role, name, digest |
| `scope.schema.json` | Run/case/variant/attempt scope with mode, purpose, host profile, plan digest |
| `operation.schema.json` | N request envelope or result envelope (`completed`, `rejected`, `failed`, `deadline`, `indeterminate`) |
| `limit.schema.json` | Finite run/case/resource budgets; every field required and positive |
| `completeness.schema.json` | Collection/interval completeness: state, counts, truncation, gaps, watermark |
| `report.schema.json` | Can report: verification, per-unit admission/execution/behavior/checks, body terminal |
| `receipt.schema.json` | N completion/cleanup receipt binding the report digest and listing released/remaining/forced resources |
| `correction.schema.json` | Append-only qualification correction invalidating a prior report/receipt pair |

Each file is self-contained (shared `id`, `digest` and clock shapes are
repeated per file) so Go, TypeScript and Can consumers can adopt one
envelope without a cross-file resolver.

## Wire conventions

- Every wire field is `snake_case`. Can identifiers are lowercase-only and
  typed JSON decode requires exact key matches, so any camelCase spelling
  would be unimplementable in ordinary Can; snake_case is the single wire
  spelling, chosen once with no legacy form.
- `schema_version` is the constant `"1"` on every envelope. Unknown versions
  are rejected; there is no legacy spelling.
- `run_id`, `operation_id`, `case_id`, `variant_id`: `^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`.
- Digests: `sha256:<64 lowercase hex>`. Paths alone are never identities.
- Clock values: `{clock, ms}` with `clock` in `n-monotonic` (N's enforcing
  clock) or `wall-utc`, and `ms >= 0`.
- Objects reject unknown fields (`additionalProperties: false`) except the
  explicitly bounded per-operation `facts`/`partial` maps, which later
  capability slices type per operation.
- Limits are always finite: every budget field is a required positive
  integer. Omitted, zero, negative or `"unlimited"` limits are rejected.

## Semantic rules (checked by `validate.py` beyond shape)

1. `operation` result: `completed` requires `kind: ok`; any other outcome
   requires a non-`ok` mechanical kind.
2. `limit`: `final_cleanup_reserve_ms <= run_wall_ms` and `prepare_ms <= run_wall_ms`.
3. `completeness`: `returned_count <= total_count`; `sealed` additionally
   requires `truncated: false`, empty `gaps` and `returned_count == total_count`.
4. `report`: `execute` mode requires at least one unit; `matched` behavior
   requires `completed` execution, `body_terminal: true` and every check
   `matched`; `blocked` admission requires `not-started` execution and
   `undetermined` behavior.
5. `receipt`: `clean` cleanup requires empty `remaining`.

## Checks

Run the fixture suite (valid documents accepted, invalid documents
rejected with the expected reason):

```sh
python3 schemas/native-test/validate.py
```

`fixtures/valid/*.json` are raw wire documents. `fixtures/invalid/*.json`
are raw wire documents that must be rejected; `fixtures/invalid/manifest.json`
maps each file to its schema and a required error substring. The validator
implements a documented JSON Schema subset (see its header) over the
standard library only, and fails loudly on unsupported schema keywords.

# std — package dispositions and the maintained example inventory

Each directory holds one package README with its current disposition
plus, until I44 deletes them, the retired predecessor sources it
replaces. Four packages ship maintained current-language projects
under `current/`; the closed native catalogue itself is generated
into [`catalogue/`](catalogue/) from
`compiler/internal/catalogue/catalogue.json` — explanations and
domain examples live in the owning package, never as a second
handwritten declaration source.

## Package dispositions

| Package | Current disposition | Maintained demonstration |
|---|---|---|
| `ascii/` | Superseded; native `text::to_*` strict grammar | `scalars/current` |
| `catalogue/` | Generated mirror (see below) | `cataloguegen --check` |
| `division/` | `number::divmod`, `euclidean_divmod` | `ratio/current` |
| `host/` | Finite `clock`/`random`/`crypto`/`env`/`log` catalogue | `cli` fixtures, admitted applications |
| `html/` | Catalogue-owned safe constructors | `html` fixtures, admitted applications |
| `json/` | Exact typed `codec` JSON | `codec` fixtures, native-ai report |
| `map/` | Native immutable maps and sets | `map/current` (shared with `set/`) |
| `quota/` | Ordinary Can validation pattern | form-validation application |
| `ratio/` | Exact-amount divmod and half-even rounding | `ratio/current` |
| `scalars/` | I22 numeric/conversion catalogue | `scalars/current` |
| `schema/` | Static build-time asset approval | `assets` fixtures, admitted applications |
| `seq/` | Native array catalogue | `arrays` fixtures, admitted applications |
| `set/` | Native immutable maps and sets | `map/current` (shared with `map/`) |
| `text/` | Native text and Unicode catalogue | `text/current` |

No new maintained project was authored: every domain above is
already demonstrated by a current program, so per-package
replacements would duplicate coverage for no behavioral gain.

## Maintained example inventory

These eight projects are the complete maintained set. Each is a
current manifest-backed project (`can.project.json`,
`can.errors.json`, `src/`, README, mandatory assertions, explicit
imports) that builds fresh from a staged layout with output only in
owned dist. `tests/integration/stdlib_test.go` discovers them by
walking for manifests, so no project can be silently omitted from
fresh compilation; `modcheck` covers every maintained `.can` file.

- `std/map/current`, `std/ratio/current`, `std/scalars/current`,
  `std/text/current`
- `examples/native-ai`, `examples/account-search`,
  `examples/form-validation`, `examples/dashboard`

## Coverage row mapping

Every included row of the coverage operation inventory resolves to
its catalogue operations, its source assertions, and its integration
evidence. The machine gate (`TestCatalogueInclusionInventory`)
walks all 144 catalogue operations and requires each owning task to
hold implementation evidence; the table below records the same
mapping per coverage row for review.

| Coverage row | Catalogue operations | Source assertions | Integration evidence |
|---|---|---|---|
| int/float/bool arithmetic, equality/ordering | native operators | `scalars/current` (40 rows) | I07/I22 evidence, `number`/`text` runtime tests |
| records/errors/variants, arrays, copy-update, field/index/slice/length | native values | `arrays` fixtures | I06/I07 evidence |
| `text::from_int/from_float/from_bool/to_int/to_float/to_bool` | `text::*` | `scalars/current` | I22 evidence |
| `number::int_to_float/float_to_int/bool_to_int/int_to_bool` | `number::*` | `scalars/current` | I22 evidence |
| `number::floor/ceil/trunc/round/is_finite/is_nan` | `number::*` | `scalars/current` | I22 evidence |
| `number::divmod/euclidean_divmod/round_ratio_half_even` | `number::*` | `ratio/current` | I23 evidence |
| array `.map/.filter/.for_each/.fold/.find/.some/.every/.sort_by` | native methods | `arrays` fixtures | I21 evidence |
| `.slice/.concat/.to_reversed`, prelude `append` | native methods | `arrays` fixtures | I07/I21 evidence |
| str `.includes/.starts_with/.ends_with/.to_lower_case/.to_upper_case/.trim/.slice/.split/.replace_all`, `text::join` | native methods, `text::join` | `text/current` (36 rows) | I24 evidence |
| `text::scalars/from_scalars/graphemes/normalize_nfc` | `text::*` | `text/current` | I24 evidence |
| `collections::empty_map/get/insert/replace/remove/entries` | `collections::*` | `map/current` | I25 evidence |
| `collections::empty_set/contains/add/union/intersection/difference` | `collections::*` | `map/current` | I25 evidence |
| `option::value<T>`, none/some | `option::*` | fixtures | I06/I46 evidence |
| `bytes::buffer/empty/from_ints/to_ints/from_utf8/to_utf8`, `.length` | `bytes::*` | `bytes` fixtures | I13 evidence |
| `codec::encode_json<T>/decode_json<T>`, schema descriptors | `codec::*` | `codec` fixtures, native-ai | I14 evidence |
| domain catalogue, `standard_failure`, `all_failed` | prelude/catalogue | `coordination` fixtures, dashboard race | I47/I49/I19 evidence |
| connection, named fetch, body/envelope/header/query data | `http::*` fetch ops | `fetch` fixtures, native-ai stub | I15/I26 evidence |
| Noul/Choice/Score, static/record/dynamic forms, judge state | `ai::*` | `native` fixtures, native-ai stub | I16/I17/I27 evidence |
| text/record LLM, ordinary transformations | `llm::*` | `native` fixtures, native-ai stub | I28/I42 evidence |
| `io::stdin_bytes/stdin_text/stdout_write/stderr_write`, main args | `io::*` | `cli` fixtures, native-ai | I11/I29 evidence |
| `clock::wall_millis/monotonic_millis/sleep_millis` | `clock::*` | account-search, dashboard | I30/I42 evidence |
| `random::secure_bytes/uuid_v4`, `crypto::sha256` | `random::*`, `crypto::*` | `cli` fixtures | I30 evidence |
| `env::required/optional`, `log::write_info/write_error` | `env::*`, `log::*` | `cli` fixtures | I29/I30 evidence |
| `html::make_tag/text/parse_url/text_attribute/url_attribute/element/fragment/text_fragment/stylesheet/meta_viewport/document` | `html::*` | `html` fixtures, admitted applications | I31 evidence |
| `htmx::get/post/target_id/target_attribute/swap_inner/swap_outer/trigger_change/trigger_input_changed/trigger_every/indicator_id/disable_this/runtime_head` | `htmx::*` | admitted applications, pinned browser | I31/I34 evidence |
| `http::request_method/request_path/query_one/query_all/request_headers/request_body/request_json<T>/request_form<T>` | `http::*` | `http` fixtures, admitted applications | I32 evidence |
| `http::make_status/make_body_status/status_ok/status_unprocessable/status_internal/status_unavailable/make_server_headers/empty_server_headers` | `http::*` | admitted applications | I32 evidence |
| `http::response_empty/response_bytes/response_text/response_html/response_json<T>` | `http::*` | admitted applications | I32 evidence |
| `http::route_get/route_post/make_router/make_server_config/server_start/server_wait/server_stop` | `http::*` | admitted applications | I32/I33 evidence |
| `asset::url`, static asset manifest, local HTMX route | `asset::*` | `assets` fixtures, admitted applications | I34 evidence |
| `sql::pool_open/pool_close/query_one/query_optional/query_rows/execute` | `sql::*` | admitted applications | I35/I37 evidence |
| `sql::with_transaction/transaction_query_one/transaction_query_optional/transaction_query_rows/transaction_execute`, decision/commit/rollback | `sql::*` | transaction fixtures | I38 evidence |
| ordinary named app validation/view/update functions | ordinary Can | admitted applications | I06/I08/I21/I42 evidence |

## Intentional P14 exclusions

Deferred or excluded capabilities fail cleanly as unknown names or
typed errors; none reaches a legacy kernel or fallback. Recorded
distinctly from included rows: decimal primitives and conversions,
GCD/LCM/roots/primality/combinatorics, bool ordering and implicit
numeric conversion, validator-descriptor builders, full casefold and
locale formatting, record keys and hash callbacks, group/zip/dedup
and comparator sort beyond `sort_by`, standalone hex/base64 codecs,
host externs/effects, filesystem/subprocesses, timing-safe compare,
broader crypto/timezone surfaces, template keywords and raw encoders,
cookies/redirect builders/captures/wildcards/middleware/WebSockets,
query ping/batch/dynamic builders/streaming, raw transaction
handles and isolation dials, migrations, client components and
authored browser JS, LLM tools, proof/termination/effect inference,
automatic retries, and rollback guarantees after cancellation.

## Checks

- `go run ./compiler/internal/catalogue/cmd/cataloguegen --check`
  (mirrors, including this `catalogue/` directory)
- `go test ./compiler/internal/catalogue/`
  (`TestCatalogueInclusionInventory`: every operation resolves to
  owning-task evidence)
- `go run ./tools/modcheck` (retired shapes stay out of maintained
  sources; `uses` resolve to catalogue or tree packages)
- `go test ./tests/integration/ -run TestStdlibMaintained`
  (every maintained project asserts and builds fresh from staging)

Old sources beside these READMEs are retired artifacts; the I44
deletion list is every `std/*/*.can` and `std/*/*.ts` outside
`current/` and `catalogue/`, the twelve `std/*/errors.json` files
beside them, `std/host/host.externs.ts`,
`std/host/platform.d.ts`, and all twenty `docs/archive/sketches/*/` programs.
`std/html/HISTORY.md` stays as a labelled historical document.

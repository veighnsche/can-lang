# Coverage and admission ledger

**Historical implementation round:** this record describes I01–I50. Later LF01–LF21 and T01–T27 added or revised capabilities, including browser Can. Read the [post-upgrade reconciliation](../syntax-taste/post-upgrade-reconciliation-2026-09-24.md) for current status; earlier scope exclusions below are historical.

This ledger maps approved contracts to their owning tasks and required evidence. Completion status is recorded in [tasks](tasks.md): all fifty tasks are checked with evidence as of 2026-09-21. The [remaining task plans](remaining-tasks.md) now record every plan closed. C/Q/A/P refer to the linked current specs there. A task's absence from the first runnable milestone does not defer it from the final initial distribution.

## Traceability

Every row below resolves through its owning tasks to passing evidence: each task row in [tasks](tasks.md) links its acceptance evidence, which records source, runtime, and integration proof with exact pins. Catalogue operations additionally resolve mechanically — `TestCatalogueInclusionInventory` walks all 144 operations to owning-task evidence — and the [std inventory](../../std/README.md) maps each included row to its operations, assertions, and integration suite. The chain ends at the admitted release report: [I45 acceptance](evidence/2026-09-21/i45/README.md) plus the [distribution release notes](../../distribution/README.md#release-notes). Intentional P14 exclusions are listed in the std inventory and fail cleanly; they never reach a legacy path.

## Every specification section

| Section | Owning tasks / final evidence |
|---|---|
| C1 authority/scope | I01,I44,I50 |
| C2 grammar | I03,I04,I16,I41 |
| C3 names/packages | I05,I06,I09,I41 |
| C4 types/generics/callables | I06,I08,I46 |
| C5 expressions/completions | I07,I10,I19 |
| C6 primitive/native semantics | I07,I13,I22,I24 |
| C7 collections/text | I21,I24,I25,I43 |
| C8 exact amounts/init/local rule | I23,I48 |
| C9 domain/standard failures | I47,I49,I40 |
| C10 primitive/callback traces | I07,I08,I11,I21,I45 |
| C11 end-to-end closure | I27,I28,I38,I42 |
| C12 readiness boundary | I12,I45,I50 |
| Q1 participants/regions | I10,I19 |
| Q2 preparation | I08,I19 |
| Q3 native promise settlement | I10,I19 |
| Q4 result types | I06,I19,I46 |
| Q5 arm coverage | I10,I19,I49 |
| Q6 all_failed and inference | I19,I46 |
| Q7 domain/standard precedence | I19,I49 |
| Q8 handler failure | I10,I19 |
| Q9 empty expansions | I19,I18 |
| Q10 outstanding resources | I20,I33,I38 |
| Q11 async collection callbacks | I21 |
| Q12 mandatory traces | I18,I19,I42,I45 |
| A1 authority/native identity | I16,I44,I50 |
| A2 data/error catalogue | I06,I13,I47,I49 |
| A3 declarations/signatures/visibility | I08,I16 |
| A4 connection/transport | I15 |
| A5 named fetch | I26 |
| A6 exact typed JSON | I14 |
| A7 judge phases | I17,I27 |
| A8 provider profile | I17,I27 |
| A9 arms/dynamic options/generated records | I08,I27,I46 |
| A10 generation profile | I28 |
| A11 native/provider evidence | I01,I18,I45 |
| A12 complete traces | I26,I27,I28,I42 |
| P1 authority/platform classification | I01,I43,I50 |
| P2 catalogue/linkage/manifests | I05,I09,I34,I37,I47 |
| P3 assertion execution | I12,I18,I41 |
| P4 evidence and fixtures | I12,I18,I45 |
| P5 deterministic harness | I18,I19 |
| P6 opacity/resources | I06,I20,I31,I33,I35,I38 |
| P7 platform errors | I47,I49 plus platform tasks |
| P8 CLI/utilities | I11,I29,I30 |
| P9 safe HTML/HTMX | I31,I34 |
| P10 HTTP/router/server | I15,I26,I32,I33 |
| P11 assets/browser boundary | I34,I42 |
| P12 PostgreSQL | I36,I37,I35,I38 |
| P13 account/search trace | I42 |
| P14 capability inclusion/exclusion | I43,I44,I50; tables below |
| P15 verification/release | I01,I02,I39,I40,I45,I50 |

The current decisions file supplies the selected surface for these section contracts, including SURFACE-069 supersession. I04/I16's grammar corpus and I42's complete programs must cover those current examples; historical illustrative fragments are not required to compile unchanged. The existing eight-topic Jev reconciliation is linked by the current specs and preserved, not replaced by this implementation consultation.

## Historical findings F01–F15

| Finding | Current repair contract | Implementation closure |
|---|---|---|
| F01 completion regions | C5,A3,Q1/Q8 | I10,I16; nested handlers/terminal/value tests |
| F02 coordination algebra | Q2–Q9 | I19,I46; all four combinators and aggregates |
| F03 async native callbacks | C7,Q11 | I21; sequential, failure-stop and thenable tests |
| F04 exact external codec | A6 | I13,I14; adversarial wire corpus |
| F05 batching and connection | A4,A7,A8 | I15,I17,I27; one POST, validation before handlers |
| F06 callable/arm compatibility | C4,A3,A9 | I08,I27,I46; captures/invariance/wrappers |
| F07 fetch/LLM capability | A5,A10 | I26,I28; bodies/envelopes/typed output |
| F08 fixture identity/evidence | P3–P5 | I12,I18; queue and evidence-label tests |
| F09 early-settlement ownership | Q10,P6 | I20,I33,I38; late work/drain/timeout |
| F10 service/browser execution | P8–P13 | I29,I31–I34,I42; server-rendered flows |
| F11 trusted provenance | P6,P9,P12 | I06,I20,I31,I35,I38; opaque forgery/lease tests |
| F12 value semantics | C6–C8 | I07,I21–I25; native conformance |
| F13 grammar/scopes | C2,C3,A3 | I03–I06,I16,I41; section/name/collision tests |
| F14 local elimination | C8 | I48; finite AST predicate positives/negatives |
| F15 stale wording/examples | C1,P14 | I42–I44,I50; current examples and historical labels |

## Complete initial catalogue operation inventory

Names are current catalogue spellings, not aliases for the old stdlib. Where the spec uses a family of generic signatures, I47 must instantiate/check that family without inventing additional operations. Every row is included in I43's final inventory gate in addition to its primary tasks.

| Surface/capability | Owning task | Required observable corner |
|---|---|---|
| int `+ - * / % ** & \| ^ ~ << >>`; float arithmetic; bool operations; equality/ordering | I07,I22 | bigint truncation/signs, negative exponent, shift direction, float NaN/infinity/±0, no coercion |
| records/errors/variants; array literals; copy-update; field/index/slice/length | I06,I07 | nominal identities, immutable aliasing, code-unit access, huge bigint bounds |
| `text::from_int`, `from_float`, `from_bool`, `to_int`, `to_float`, `to_bool` | I22 | strict parse grammar versus native explicit formatting; no percentage/locale surprise |
| `number::int_to_float`, `float_to_int`, `bool_to_int`, `int_to_bool` | I22 | exactness/finite/integral and0/1 checks |
| `number::floor`, `ceil`, `trunc`, `round`, `is_finite`, `is_nan` | I22 | native round tie direction and nonfinite behavior |
| `number::divmod`, `euclidean_divmod`, `round_ratio_half_even`; division/rounded records | I23 | zero denominator, negative signs, exact remainder identity |
| array `.map`, `.filter`, `.for_each`, `.fold`, `.find`, `.some`, `.every`, `.sort_by` | I21 | sequential order, empty cases, stop on first failure, no async comparator |
| array `.slice`, `.concat`, `.to_reversed`; prelude `append` | I07,I21 | native copies; source and aliases unchanged |
| str `.includes`, `.starts_with`, `.ends_with`, `.to_lower_case`, `.to_upper_case`, `.trim`, `.slice`, `.split`, `.replace_all`; `text::join` | I24 | native Unicode casing/whitespace, literal replacement, nonempty delimiters |
| `text::scalars`, `from_scalars`, `graphemes`, `normalize_nfc` | I24 | scalar validation, bounded chunking and pinned ICU behavior |
| `collections::empty_map`, `get`, `insert`, `replace`, `remove`, `entries`; `entry<K,V>` | I25 | absent/duplicate keys, insertion order, scalar-key constraint |
| `collections::empty_set`, `contains`, `add`, `union`, `intersection`, `difference` | I25 | both set-size paths preserve Can order |
| `option::value<T>`, none/some records | I06,I46 | ordinary nominal variants; no null leakage |
| `bytes::buffer`, `bytes::empty`, `from_ints`, `to_ints`, `from_utf8`, `to_utf8`, buffer `.length` | I13 | no mutable view exposed, fatal invalid decode |
| `codec::encode_json<T>`, `decode_json<T>` and compiler schema descriptors | I14 | duplicates, exact numbers and every finite budget shared across consumers |
| domain catalogue + standard_failure + all_failed | I47,I49,I19 | exact ID/kind/specialization/payload and original occurrence order |
| connection + named fetch/body/envelope/header/query data | I15,I26 | auth capture, full deadline, repeated metadata and status policy |
| question Noul/Choice/Score, static/record/dynamic forms; `choice_option`/`choice_arm`; judge grouped state | I16,I17,I27 | request-wide validation, metadata context, one batch and ordered handler effects |
| text/record LLM with ordinary transformations | I28,I42 | closed schema, refusal/truncation/invalid distinction and later independent judge |
| `io::stdin_bytes`, `stdin_text`, `stdout_write`, `stderr_write`; main args | I11,I29 | bounded reads, awaited writes, application-only argv |
| `clock::wall_millis`, `monotonic_millis`, `sleep_millis` | I30 | declared int/float units,0–2147483647 duration, deterministic substitution |
| `random::secure_bytes`, `uuid_v4`; `crypto::sha256` | I30 |0–65536 bytes; randomness supplied in assertions, SHA real |
| `env::required`, `optional`; `log::write_info`, `write_error` | I29,I30 | exact validated name, no enumeration/mutation; exactly two JSON string fields |
| `html::make_tag`, `text`, `parse_url`, `text_attribute`, `url_attribute`, `element`, `fragment`, `text_fragment`, `stylesheet`, `meta_viewport`, `document` | I31 | exact closed inventories, contextual native escaping, head/body and child rules |
| `htmx::get`, `post`, `target_id`, `target_attribute`, `swap_inner`, `swap_outer`, `trigger_change`, `trigger_input_changed`, `trigger_every`, `indicator_id`, `disable_this`, `runtime_head` | I31,I34 | range/ID/same-origin validation and sole trusted pinned script |
| `http::request_method`, `request_path`, `query_one`, `query_all`, `request_headers`, `request_body`, `request_json<T>`, `request_form<T>` | I32 | normalized path, cached bounded snapshot, strict form rules |
| `http::make_status`, `make_body_status`, `status_ok`, `status_unprocessable`, `status_internal`, `status_unavailable`, `make_server_headers`, `empty_server_headers` | I32 | excluded body statuses, invalid/reserved headers |
| `http::response_empty`, `response_bytes`, `response_text`, `response_html`, `response_json<T>` | I32 | typed body/status and immutable complete native Response |
| `http::route_get`, `route_post`, `make_router`, `make_server_config`, `server_start`, `server_wait`, `server_stop` | I32,I33 |404/405/Allow/no implicit HEAD, native stop plus lease drain |
| `asset::url`, static asset manifest and local HTMX route | I34 | digest/name/MIME allowlist, reserved path,422swap/204no-swap |
| `sql::pool_open`, `pool_close`, `query_one`, `query_optional`, `query_rows`, `execute` | I35,I37 | driver establishment, native parameterization, typed rows and bounded cardinality |
| `sql::with_transaction`, `transaction_query_one`, `transaction_query_optional`, `transaction_query_rows`, `transaction_execute`; decision/commit/rollback records | I38 | scoped handle, drain before decision, private rollback, commit_unknown |
| ordinary named app validation/view/update functions | I06,I08,I21,I42 | error records, immutable state and safe server-rendered results; no hidden purity promise |

## ASTRA_STDLIB intent and explicit limits

All sections of the historical catalogue were read. Its old NOW/dependency labels and sequenced roadmap are superseded. Inclusion below means current equivalent capability, not restoration of every old helper name or error schema. A convenient function expressible as ordinary Can is not automatically a new native intrinsic.

| Historical section/family | Current disposition |
|---|---|
| §1.1 abs/negate/sign/min/max/clamp/distance/square/pow/bounded arithmetic | Native arithmetic/comparisons and ordinary domain validation patterns, I07/I22/I43. No old blanket helper-name/API commitment. Decimal counterparts/lerp are not admitted as a dec type. |
| §1.2 division/mod/multiple/parity | Native bigint `/`/`%` plus C8 divmod/Euclidean adapter, I23. GCD/LCM/integer-root/primality/power-of-two/factorial/binomial and convenience combinatorics are advanced catalogue work deferred by P14. No fuel algorithms ported. |
| §1.3 decimal parts/scale/exact divide/round/sqrt; ratios | No decimal primitive. Integer minor units and exact half-even ratio rounding cover initial exact amounts, I23. General normalized fractions/decimal conversion/algebra require later specific contracts. |
| §1.4 bool/compare/predicate/select | Ordinary operators, match and named Can functions, I07/I43. No bool ordering primitive or implicit numeric conversion added from old examples. |
| §1.5 scalar validation and report aggregation | Ordinary Can records/errors/match and arrays, I06/I21/I42. Internal boundary schemas are I14/I37; user schema builders and arbitrary validator descriptors deferred. |
| §1.6 conversion | Current explicit C6 int/float/bool/text catalogue, I22. Decimal conversions excluded; no compatibility aliases. |
| §1.7 optional/outcome combinators | Ordinary option/record/variant composition plus completion handling, I06/I10/I46. Generic stored completion/outcome monad catalogue unnecessary initially; users can materialize data variants. |
| §1.8 text | C7 search/case/trim/split/join/replace/scalar/grapheme/NFC, I24. Full casefold, locale-rich formatting and additional text algorithms deferred; not substituted by lowercase. |
| §1.8 collections | C7 array/map/set catalogue, I21/I25. Arbitrary hash/equality callbacks, record keys, new group/zip/dedup/sort comparator APIs beyond C7 deferred. |
| §1.8 encoding/schemas | Owned bytes/strict UTF-8/exact typed JSON and internal type descriptors, I13/I14. Old standalone hex/base64/base64url or authored schema catalogues are not silently ported without current admission contracts. |
| §1.8 host shelf | Finite CLI/clock/random/SHA/env/log catalogue, I29/I30. Host externs/effects, runtime filesystem/subprocesses and broader crypto/timezone catalogues excluded/deferred. |
| §2 typed fragments/documents/attributes/URLs/templates | Safe HTML and named functions/arrays, I31. No template keyword, raw context encoder, user sealing or arbitrary script/style URL injection. Head stylesheet and static assets have P9/P11 contracts. |
| §3 web request/response/router/server/client | P10 server plus named fetch, I26/I32/I33. Cookies, redirects as a general library builder, route captures/wildcards/middleware, arbitrary cancellation contexts, TLS/HTTP2 policy, WebSockets and streaming deferred. Native platform internals do not imply authored APIs. |
| §4 query declarations/binding/rows/pools | Static manifest PostgreSQL descriptors and native Bun.SQL, I35–I37. No revisioned schema attestation surface; row validation is not migration/schema equivalence proof. Dedicated ping/batch helpers, dynamic query building, arbitrary dialects and streaming deferred. |
| §4 transactions | Scoped callback returning commit/rollback data, I38. No authored raw begin/commit/rollback handles, adjustable isolation/access schema or claimed rollback after unknown commit. Migrations/planning/apply deferred. |
| §5 view/update/render/form/list/table | Ordinary immutable server-side functions plus HTML/HTTP/HTMX, I31–I34/I42. Typed forms validate at server boundary. Lists/tables render through arrays and safe constructors. |
| §5 component/children/slots/events/commands/mount/dispatch/unmount/hydrate/history | Browser Can/client component framework and authored JS/TS are out of scope. HTMX handles approved requests/swaps/triggers. No Preact runtime, browser callback compilation, keyed reconciliation algorithm or hydrate protocol. |
| §6 old roadmap | Replaced by M0–M5. Native AI is the second executable slice, never an optional layer after the historical frontend roadmap. |

Additional exclusions: LLM tools, provider tool execution loops, proof/termination/effect inference, affine ownership syntax, dynamic backend imports, arbitrary provider adapters, automatic retries and transaction rollback guarantees after cancellation. Linux requires a future full target gate; Windows/browser Can are outside this initial plan. Runtime headers/env secrets remain on the server, never in static HTMX output.

## Final closure rule

I50 cannot close unless every included row links to passing implementation evidence, every C/Q/A/P and F row is accounted for, every exclusion remains intentional, and all mandatory P15 gates pass on the packaged target. An unimplemented supported row is a blocker, not a documentation footnote. A deferred row may not be smuggled into the runtime as an untested legacy fallback.

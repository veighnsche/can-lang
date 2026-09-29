# G02 migration map

Date: 2026-09-29. Baseline HEAD `4393a89e`. Frozen while the old parser
still accepts old source. Method: temporary in-module helper
(`compiler/internal/syntax/zz_migration_map_test.go`, deleted after this
freeze) drove `syntax.Parse` + `syntax.Lex` over every active `.can` file
and recorded single-character delimiter edits at lexer token spans. Every
edit offset was verified to hold the expected old byte. Constructor
classification used a reviewed declaration/catalogue map: authored
`ErrorDecl`/`RecordDecl`/`QuestionDecl`-generated records per declaring
package (with per-file import-alias and self-package resolution) plus all
110 catalogue errors and 57 catalogue records from
`compiler/internal/catalogue/catalogue.json`.

## Totals

- Files walked: 196 (183 active + 13 frozen `implementation-gaps` projects).
  Parsed: 194. Skipped: 2 (classified below).
- Span edits: 3470 (`migration-edits.txt`, one per line):
  `path:line:col off=N "old"->"new" role`.
  - `error-decl`: 28 (14 declarations: `error E(...)` parens).
  - `emits-bound`: 2396 (1198 finite bounds, incl. `emits []` and nested
    callable/choice-arm bounds; each bound's own brackets only).
  - `error-ctor`: 296 (148 non-terminal error constructions, incl.
    success-data `ok wrapper(E(...))`, arrays, bindings, nested generics).
  - `error-ctor-terminal`: 750 (375 terminal `FailureBody` completions).
  - `record-ctor-kept`: 5729 constructor positions verified as records
    (authored, generated-record, and catalogue) and excluded from edits.
  - Unknown constructors: 0 after alias/self-package resolution.
- `emits calculated` bounds (wrap/main.can lines 17, 28, 86) produced no
  edits. `emits [][]` (queues.can:107) maps only the bound pair
  (off=2263/2264); the array suffix is untouched, preserving precedence.
- Calls, records, arrays, patterns, strings, comments, JSON, package
  headers, and `[_]` catches are excluded by construction: only
  `ErrorDecl`, `ErrorBound`, and `ConstructorExpr` AST nodes classified as
  errors yield edits.

## Coverage classes (all present in the map)

- Nullary: terminal/data `E()` reviewed in new F02/N01 tests; no nullary
  authored `error E()` decl exists in the active corpus (catalogue
  nullaries such as `collections::key_absent` are constructed in tests).
- Generic: `all_failed<F>` catalogue error (aggregate-composition.can,
  coordination/main.can) with nested `codec::invalid_data{...}` payloads.
- Qualified: `contract::*` alias-resolved errors across invoice projects;
  `codec::invalid_data`, `http::*`, `text::*`, `number::*` catalogue ctors.
- Nested: `all_failed<a_failure>{[codec::invalid_data{"a", "type"}]}` and
  deeply nested retry `traced_down` assertions.
- Catalogue: 110 catalogue errors in the classification map.
- Success-data: 148 non-terminal `error-ctor` edits (stored values).
- Terminal: 375 `error-ctor-terminal` edits reaching `FailureBody`.

## Skipped files (2)

- `compiler/testdata/current/lexer/core.can` [df8f3e49feab]: bare
  `Parse` fails at line 19 (a parenthesized literal is not a valid
  assertion head); the 59-file round-trip walker skips it as a negative
  fixture.
  M01 migrates its single line-15 `emits []` bound by hand and keeps the
  line-19 failure intact.
- `docs/.../implementation-gaps/projects/diagnostic-parse/src/main.can`
  [4a878d8fcbf2]: intentionally invalid (`ok @`); not referenced by any
  Go test. Classified historical; left untouched.

## Executed frozen projects (3 of 13)

Only `passing`, `failing-assertion`, and `pending-assertion` are copied by
`tests/integration/verified_build_test.go` (needs `CAN_BUN_ARCHIVE`).
Those three migrate under M02. The other ten frozen projects are
unreferenced dated evidence; their map rows are recorded for review but
their files stay historical.

## Out of span-map scope (lane-owned)

- Go-embedded Can (F/N/M03/P04 suites): migrated by each lane with its
  tests; every `strings.Replace` mutation must assert it matched.
- Executable docs probes: `technical-spec.md` C10/Consumer blocks and
  `decisions.md` `## Packages` examples migrate with P01 (window 2);
  `tools/performance` generators migrate under M02.
- Intentional negatives (`owner-negative`, `retry-negative`,
  `gallery-failing`, rejection tests): keep old spellings only as named
  rejection cases; M02/M03 preserve each test's intent.

## Per-file hashes and edit counts

Format: `path [sha256:12] edits=N roles=map[...]`. Files with `edits=0`
parsed cleanly and need no delimiter change.

compiler/testdata/current/arrays/contracts.can [315a6e40fdec] edits=68 roles=map[emits-bound:60 error-ctor-terminal:8]
compiler/testdata/current/arrays/main.can [bfd784379ed2] edits=86 roles=map[emits-bound:86]
compiler/testdata/current/assertions/basic.can [61dc84ce828e] edits=4 roles=map[emits-bound:4]
compiler/testdata/current/assertions/native-slice.can [04b2156b996f] edits=4 roles=map[emits-bound:4]
compiler/testdata/current/assertions/queues.can [2c88439717b8] edits=52 roles=map[emits-bound:52]
compiler/testdata/current/assets/page.can [dbf9215b79ad] edits=20 roles=map[emits-bound:20]
compiler/testdata/current/bytes/roundtrip.can [759d6faae060] edits=30 roles=map[emits-bound:22 error-ctor-terminal:8]
compiler/testdata/current/callables/captures.can [876c23e4f3ad] edits=30 roles=map[emits-bound:30]
compiler/testdata/current/checks/main.can [eca084689ce6] edits=12 roles=map[emits-bound:8 error-ctor-terminal:4]
compiler/testdata/current/cli/echo.can [71c41bdb5861] edits=2 roles=map[emits-bound:2]
compiler/testdata/current/cli/environment.can [e8db6ea90cfa] edits=2 roles=map[emits-bound:2]
compiler/testdata/current/cli/input-text.can [e43b91f77587] edits=2 roles=map[emits-bound:2]
compiler/testdata/current/cli/input.can [1d59079a4d1a] edits=2 roles=map[emits-bound:2]
compiler/testdata/current/cli/utilities.can [6a6ccb1d54fe] edits=10 roles=map[emits-bound:10]
compiler/testdata/current/codec/formats.can [e13e5a52dfc0] edits=28 roles=map[emits-bound:14 error-ctor-terminal:14]
compiler/testdata/current/codec/roundtrip.can [07e72680eb64] edits=20 roles=map[emits-bound:14 error-ctor-terminal:6]
compiler/testdata/current/cookies/session.can [f99afaf1a396] edits=28 roles=map[emits-bound:24 error-ctor-terminal:4]
compiler/testdata/current/coordination/aggregate-composition.can [7369d0ccfc41] edits=304 roles=map[emits-bound:44 error-ctor:200 error-ctor-terminal:60]
compiler/testdata/current/coordination/heterogeneous.can [88699083154e] edits=12 roles=map[emits-bound:12]
compiler/testdata/current/coordination/main.can [d68e7f705995] edits=28 roles=map[emits-bound:20 error-ctor:2 error-ctor-terminal:6]
compiler/testdata/current/crypto/commands.can [063edcde6523] edits=10 roles=map[emits-bound:10]
compiler/testdata/current/fetch/main.can [a273c5c9d906] edits=94 roles=map[emits-bound:26 error-ctor:32 error-ctor-terminal:36]
compiler/testdata/current/generics/chain-core.can [697f01bd81bc] edits=10 roles=map[emits-bound:4 error-ctor-terminal:4 error-decl:2]
compiler/testdata/current/generics/chain-helpers.can [0995f60f7056] edits=10 roles=map[emits-bound:8 error-ctor-terminal:2]
compiler/testdata/current/generics/chain-mail.can [8a58e148de71] edits=4 roles=map[emits-bound:4]
compiler/testdata/current/generics/chain-main.can [e7512c035239] edits=4 roles=map[emits-bound:4]
compiler/testdata/current/generics/definitions.can [f2bee750e0a9] edits=4 roles=map[emits-bound:4]
compiler/testdata/current/generics/finite-field-chain.can [176f82e7a590] edits=4 roles=map[emits-bound:4]
compiler/testdata/current/generics/finite-transition.can [3209499abc96] edits=4 roles=map[emits-bound:4]
compiler/testdata/current/generics/helper.can [3a8be5677a95] edits=2 roles=map[emits-bound:2]
compiler/testdata/current/generics/main.can [a087a7dd111f] edits=4 roles=map[emits-bound:4]
compiler/testdata/current/generics/spread-inferred.can [eac430025d0f] edits=4 roles=map[emits-bound:4]
compiler/testdata/current/html/main.can [5000974a5465] edits=10 roles=map[emits-bound:6 error-ctor-terminal:4]
compiler/testdata/current/http/main.can [a4118e5ce3cd] edits=34 roles=map[emits-bound:28 error-ctor-terminal:6]
compiler/testdata/current/http/server.can [7a32d9186310] edits=14 roles=map[emits-bound:14]
compiler/testdata/current/http/tls.can [c60fd9f8c1e0] edits=10 roles=map[emits-bound:10]
compiler/testdata/current/lexer/literals.can [6cc3ceebf3a6] edits=0 roles=map[]
compiler/testdata/current/markdown/main.can [a847622d08d3] edits=12 roles=map[emits-bound:10 error-ctor-terminal:2]
compiler/testdata/current/native/declarations.can [ef418fc7102a] edits=20 roles=map[emits-bound:20]
compiler/testdata/current/native/generation.can [b8b5facdbecd] edits=10 roles=map[emits-bound:10]
compiler/testdata/current/native/noul.can [9748f25a8d11] edits=8 roles=map[emits-bound:8]
compiler/testdata/current/native/questions.can [ba9731bb3abd] edits=40 roles=map[emits-bound:40]
compiler/testdata/current/parser/offline.can [072c42bf513c] edits=2 roles=map[emits-bound:2]
compiler/testdata/current/project/src/alpha/second.can [749eb3451efa] edits=2 roles=map[emits-bound:2]
compiler/testdata/current/project/src/alpha/shared.can [47fb128e2228] edits=0 roles=map[]
compiler/testdata/current/project/src/beta/shared.can [4324daf198d4] edits=0 roles=map[]
compiler/testdata/current/project/src/main/main.can [35be38f53bc1] edits=2 roles=map[emits-bound:2]
compiler/testdata/current/project/vendor/sample/src/shared.can [90db561d342f] edits=2 roles=map[error-decl:2]
compiler/testdata/current/s3/objects.can [76154ecfaacb] edits=68 roles=map[emits-bound:32 error-ctor-terminal:36]
compiler/testdata/current/sql/descriptors.can [1f4acb4c0469] edits=2 roles=map[emits-bound:2]
compiler/testdata/current/sql/mysql.can [99e192e8c2f1] edits=16 roles=map[emits-bound:16]
compiler/testdata/current/sql/queries.can [a242dd2ad463] edits=20 roles=map[emits-bound:16 error-ctor-terminal:4]
compiler/testdata/current/sql/sqlite.can [06ed519840f7] edits=16 roles=map[emits-bound:16]
compiler/testdata/current/sql/transactions.can [8750f6e59c49] edits=22 roles=map[emits-bound:20 error-ctor-terminal:2]
compiler/testdata/current/templates/helpers.can [1dadbe1c8a08] edits=0 roles=map[]
compiler/testdata/current/templates/main.can [deedc8693611] edits=26 roles=map[emits-bound:26]
compiler/testdata/current/wrap/main.can [ea327ea3b642] edits=42 roles=map[emits-bound:16 error-ctor:18 error-ctor-terminal:8]
compiler/testdata/current/ws/socket.can [a91252175105] edits=32 roles=map[emits-bound:26 error-ctor-terminal:6]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/cpu-assertion/src/main.can [fc5b6acd51a0] edits=4 roles=map[emits-bound:4]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/diagnostic-resolve/src/main.can [d9ecafbd137b] edits=4 roles=map[emits-bound:4]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/diagnostic-semantic/src/main.can [7bda55549313] edits=4 roles=map[emits-bound:4]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/failing-assertion/src/main.can [b40041d7c686] edits=4 roles=map[emits-bound:4]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/passing/src/main.can [fbfd46e7a08d] edits=4 roles=map[emits-bound:4]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/pending-assertion/src/main.can [f83d955313b0] edits=6 roles=map[emits-bound:6]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-array/src/main.can [c43281b8579a] edits=4 roles=map[emits-bound:4]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-closure/src/main.can [b7b100d46223] edits=8 roles=map[emits-bound:8]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-direct/src/main.can [51d9360f2829] edits=4 roles=map[emits-bound:4]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-record/src/main.can [fe4856f78287] edits=4 roles=map[emits-bound:4]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-transaction-array/src/main.can [3d9db7fe900c] edits=6 roles=map[emits-bound:6]
docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/resource-transaction-record/src/main.can [ef2f29f35b15] edits=6 roles=map[emits-bound:6]
examples/account-search/src/model/model.can [24d41fc4b804] edits=10 roles=map[emits-bound:8 error-ctor-terminal:2]
examples/account-search/src/records/records.can [d2a56aae821e] edits=0 roles=map[]
examples/account-search/src/render/render.can [e448665bc8e8] edits=8 roles=map[emits-bound:8]
examples/account-search/src/web/web.can [56b607839e25] edits=22 roles=map[emits-bound:20 error-ctor-terminal:2]
examples/cookies/src/main.can [efaa18083604] edits=28 roles=map[emits-bound:24 error-ctor-terminal:4]
examples/crypto/src/main.can [76100458bee2] edits=12 roles=map[emits-bound:10 error-ctor-terminal:2]
examples/dashboard/src/model/model.can [e0b28c27d21d] edits=12 roles=map[emits-bound:6 error-ctor-terminal:6]
examples/dashboard/src/records/records.can [87b379470572] edits=0 roles=map[]
examples/dashboard/src/render/render.can [86da2fb127ce] edits=6 roles=map[emits-bound:6]
examples/dashboard/src/web/web.can [13361147f00a] edits=10 roles=map[emits-bound:10]
examples/files/src/main.can [04f2ba3b9b11] edits=12 roles=map[emits-bound:8 error-ctor-terminal:4]
examples/form-validation/src/model/model.can [446741195582] edits=2 roles=map[emits-bound:2]
examples/form-validation/src/records/records.can [0bc008994999] edits=0 roles=map[]
examples/form-validation/src/render/render.can [6f5af1ca5769] edits=6 roles=map[emits-bound:6]
examples/form-validation/src/web/web.can [174a1e0720b9] edits=14 roles=map[emits-bound:14]
examples/gallery/src/01-hello-world.can [be47e5d79c28] edits=2 roles=map[emits-bound:2]
examples/gallery/src/02-cli-argument.can [10a69e4e016b] edits=2 roles=map[emits-bound:2]
examples/gallery/src/03-arithmetic.can [c2009816988b] edits=2 roles=map[emits-bound:2]
examples/gallery/src/04-temperature.can [f6f439e0a851] edits=2 roles=map[emits-bound:2]
examples/gallery/src/05-leap-year.can [02af27d4bae5] edits=2 roles=map[emits-bound:2]
examples/gallery/src/06-fizzbuzz.can [82edc084e811] edits=2 roles=map[emits-bound:2]
examples/gallery/src/07-fibonacci.can [718fe613a550] edits=2 roles=map[emits-bound:2]
examples/gallery/src/08-factorial.can [50e5961a9b80] edits=2 roles=map[emits-bound:2]
examples/gallery/src/09-gcd.can [079aa760f904] edits=2 roles=map[emits-bound:2]
examples/gallery/src/10-prime-check.can [440684be764d] edits=4 roles=map[emits-bound:4]
examples/gallery/src/11-palindrome.can [57c9b50a7a67] edits=2 roles=map[emits-bound:2]
examples/gallery/src/12-word-character-count.can [26e6314bf407] edits=4 roles=map[emits-bound:4]
examples/gallery/src/13-boolean-match.can [a34a3b1580a7] edits=2 roles=map[emits-bound:2]
examples/gallery/src/14-record-variant-match.can [8b5ff0c5120f] edits=2 roles=map[emits-bound:2]
examples/gallery/src/15-exhaustive-match.can [c5e9d2a959ea] edits=2 roles=map[emits-bound:2]
examples/gallery/src/16-optional-values.can [594aa3cd3901] edits=2 roles=map[emits-bound:2]
examples/gallery/src/17-named-error-recovery.can [9fdfb0595791] edits=2 roles=map[emits-bound:2]
examples/gallery/src/18-relay.can [1c00ef8cb75e] edits=4 roles=map[emits-bound:2 error-ctor-terminal:2]
examples/gallery/src/19-immutable-record-update.can [e0eb7bff93c8] edits=2 roles=map[emits-bound:2]
examples/gallery/src/20-array-map.can [41df65dd3525] edits=4 roles=map[emits-bound:4]
examples/gallery/src/21-array-filter.can [3b3477957785] edits=4 roles=map[emits-bound:4]
examples/gallery/src/22-array-fold.can [97480702a13c] edits=4 roles=map[emits-bound:4]
examples/gallery/src/23-array-sort.can [2772b74dc60e] edits=4 roles=map[emits-bound:4]
examples/gallery/src/24-map-word-frequencies.can [97bbfd8ab458] edits=4 roles=map[emits-bound:4]
examples/gallery/src/25-generic-function.can [1ea6a76841bd] edits=2 roles=map[emits-bound:2]
examples/gallery/src/26-callable-argument.can [84f1e3252f8d] edits=6 roles=map[emits-bound:6]
examples/gallery/src/27-recursion-vs-collection.can [502a417e6473] edits=6 roles=map[emits-bound:6]
examples/gallery/src/28-attached-assertions.can [48b6113007c0] edits=2 roles=map[emits-bound:2]
examples/gallery/src/main.can [b3c3ada5d67a] edits=2 roles=map[emits-bound:2]
examples/invoice-compare/src/model/model.can [d340b79e4ca9] edits=44 roles=map[emits-bound:44]
examples/invoice-compare/src/records/records.can [43c580d4ef91] edits=0 roles=map[]
examples/invoice-compare/src/web/web.can [618ad08ee17c] edits=52 roles=map[emits-bound:38 error-ctor-terminal:14]
examples/invoice-compare/vendor/controls/src/fields/fields.can [bd03c05a84f9] edits=14 roles=map[emits-bound:14]
examples/invoice-compare/vendor/controls/src/keyed/keyed.can [96b6cad171ae] edits=16 roles=map[emits-bound:16]
examples/invoice-compare/vendor/controls/src/notices/notices.can [59348072a13e] edits=30 roles=map[emits-bound:30]
examples/invoice-grid/src/model/model.can [5b62d0c3f895] edits=128 roles=map[emits-bound:128]
examples/invoice-grid/src/records/records.can [73e22ba79f94] edits=0 roles=map[]
examples/invoice-grid/src/web/web.can [8aa350cb1abb] edits=104 roles=map[emits-bound:70 error-ctor-terminal:34]
examples/invoice-grid/vendor/billing/src/invoice_contract/invoice_contract.can [f6909d4a9230] edits=0 roles=map[]
examples/invoice-grid/vendor/controls/src/fields/fields.can [bd03c05a84f9] edits=14 roles=map[emits-bound:14]
examples/invoice-grid/vendor/controls/src/keyed/keyed.can [96b6cad171ae] edits=16 roles=map[emits-bound:16]
examples/invoice-grid/vendor/controls/src/notices/notices.can [59348072a13e] edits=30 roles=map[emits-bound:30]
examples/invoice/src/model/model.can [5b700a38bcd4] edits=166 roles=map[emits-bound:124 error-ctor-terminal:42]
examples/invoice/src/records/records.can [94faa6566c93] edits=0 roles=map[]
examples/invoice/src/render/render.can [fc67e50b9db4] edits=14 roles=map[emits-bound:10 error-ctor-terminal:4]
examples/invoice/src/web/web.can [190716a30804] edits=70 roles=map[emits-bound:44 error-ctor-terminal:26]
examples/invoice/vendor/billing/src/invoice_contract/invoice_contract.can [f6909d4a9230] edits=0 roles=map[]
examples/language-site/src/site.can [6799a4dbb615] edits=20 roles=map[emits-bound:20]
examples/markdown/src/main.can [3f9cb773b217] edits=12 roles=map[emits-bound:10 error-ctor-terminal:2]
examples/mysql/src/main.can [3034d6119ebc] edits=8 roles=map[emits-bound:8]
examples/native-ai/src/app/main.can [847e6a8fc5a2] edits=8 roles=map[emits-bound:4 error-ctor-terminal:4]
examples/native-ai/src/model/model.can [dc9a1b422f75] edits=14 roles=map[emits-bound:10 error-ctor-terminal:4]
examples/native-ai/src/oracles/oracles.can [8a7c8d301b87] edits=24 roles=map[emits-bound:16 error-ctor-terminal:8]
examples/native-ai/src/records/records.can [a0711951450d] edits=0 roles=map[]
examples/process/src/main.can [b50adb084463] edits=6 roles=map[emits-bound:6]
examples/sqlite/src/main.can [a08274014518] edits=8 roles=map[emits-bound:8]
examples/stream/src/main.can [c85cf677a4e8] edits=34 roles=map[emits-bound:4 error-ctor-terminal:30]
examples/utilities/src/main.can [b71e0fc1ffe0] edits=14 roles=map[emits-bound:12 error-ctor-terminal:2]
examples/webhook/src/model/model.can [b78d02a19144] edits=54 roles=map[emits-bound:26 error-ctor-terminal:28]
examples/webhook/src/records/records.can [34fefd86afd6] edits=0 roles=map[]
examples/webhook/src/web/web.can [94233dcd3ffa] edits=102 roles=map[emits-bound:56 error-ctor-terminal:46]
examples/websocket/src/main.can [e43a03a68341] edits=32 roles=map[emits-bound:26 error-ctor-terminal:6]
shared/grid-controls/src/fields/fields.can [bd03c05a84f9] edits=14 roles=map[emits-bound:14]
shared/grid-controls/src/keyed/keyed.can [96b6cad171ae] edits=16 roles=map[emits-bound:16]
shared/grid-controls/src/notices/notices.can [59348072a13e] edits=30 roles=map[emits-bound:30]
shared/invoice-contract/src/invoice_contract/invoice_contract.can [f6909d4a9230] edits=0 roles=map[]
std/map/current/src/main.can [b378c8de57b0] edits=36 roles=map[emits-bound:28 error-ctor-terminal:8]
std/ratio/current/src/main.can [d65e2fcc6d58] edits=22 roles=map[emits-bound:16 error-ctor-terminal:6]
std/scalars/current/src/main.can [a546cc9e697d] edits=52 roles=map[emits-bound:38 error-ctor-terminal:14]
std/text/current/src/main.can [c1662b662daa] edits=80 roles=map[emits-bound:60 error-ctor-terminal:20]
tests/authoring-policies/boolean/baseline/src/main.can [2e53818b4481] edits=4 roles=map[emits-bound:4]
tests/authoring-policies/boolean/reference/src/main.can [8fdb5af2f567] edits=4 roles=map[emits-bound:4]
tests/authoring-policies/locals/baseline/src/main.can [55353c9cc2dc] edits=4 roles=map[emits-bound:4]
tests/authoring-policies/locals/reference/src/main.can [c006c6ecede4] edits=4 roles=map[emits-bound:4]
tests/authoring-policies/near/baseline/src/main.can [be7effd4b8df] edits=8 roles=map[emits-bound:8]
tests/authoring-policies/near/reference/src/main.can [c3127481f239] edits=8 roles=map[emits-bound:8]
tests/failure-conventions/owner-forge/src/app/main.can [c6e49d53f855] edits=2 roles=map[emits-bound:2]
tests/failure-conventions/owner-forge/src/handler/handler.can [60bc5405c46d] edits=12 roles=map[emits-bound:10 error-ctor-terminal:2]
tests/failure-conventions/owner-forge/src/ids/ids.can [59b7a17614ab] edits=10 roles=map[emits-bound:4 error-ctor-terminal:4 error-decl:2]
tests/failure-conventions/owner-negative/src/app/main.can [c6e49d53f855] edits=2 roles=map[emits-bound:2]
tests/failure-conventions/owner-negative/src/handler/handler.can [a44c3be2171d] edits=12 roles=map[emits-bound:10 error-ctor-terminal:2]
tests/failure-conventions/owner-negative/src/ids/ids.can [59b7a17614ab] edits=10 roles=map[emits-bound:4 error-ctor-terminal:4 error-decl:2]
tests/failure-conventions/owner-setup/src/app/main.can [c6e49d53f855] edits=2 roles=map[emits-bound:2]
tests/failure-conventions/owner-setup/src/handler/handler.can [7844a9adfd29] edits=12 roles=map[emits-bound:10 error-ctor-terminal:2]
tests/failure-conventions/owner-setup/src/ids/ids.can [59b7a17614ab] edits=10 roles=map[emits-bound:4 error-ctor-terminal:4 error-decl:2]
tests/failure-conventions/retry-fixed/src/app/main.can [c6e49d53f855] edits=2 roles=map[emits-bound:2]
tests/failure-conventions/retry-fixed/src/billing/billing.can [7a462ae9c630] edits=24 roles=map[emits-bound:4 error-ctor-terminal:18 error-decl:2]
tests/failure-conventions/retry-fixed/src/profiles/profiles.can [c05f9eb343d7] edits=36 roles=map[emits-bound:6 error-ctor-terminal:26 error-decl:4]
tests/failure-conventions/retry-negative/src/app/main.can [c6e49d53f855] edits=2 roles=map[emits-bound:2]
tests/failure-conventions/retry-negative/src/billing/billing.can [6e8f0077f2c0] edits=46 roles=map[emits-bound:8 error-ctor:10 error-ctor-terminal:26 error-decl:2]
tests/failure-conventions/retry-negative/src/profiles/profiles.can [4226aa99b19b] edits=80 roles=map[emits-bound:16 error-ctor:12 error-ctor-terminal:48 error-decl:4]
tests/failure-conventions/retry-negative/src/retry/retry.can [fae70103ad40] edits=8 roles=map[emits-bound:8]
tests/failure-conventions/retry/src/app/main.can [c6e49d53f855] edits=2 roles=map[emits-bound:2]
tests/failure-conventions/retry/src/billing/billing.can [6e8f0077f2c0] edits=46 roles=map[emits-bound:8 error-ctor:10 error-ctor-terminal:26 error-decl:2]
tests/failure-conventions/retry/src/profiles/profiles.can [4226aa99b19b] edits=80 roles=map[emits-bound:16 error-ctor:12 error-ctor-terminal:48 error-decl:4]
tests/failure-conventions/retry/src/retry/retry.can [f25d0fdf8986] edits=8 roles=map[emits-bound:8]
tests/integration/testdata/browser-conformance/src/main.can [452666fdb0cf] edits=48 roles=map[emits-bound:48]
tests/integration/testdata/browser-controls/src/main.can [fdfc7b1fe5b6] edits=36 roles=map[emits-bound:36]
tests/integration/testdata/formats/main.can [378cdaa92296] edits=14 roles=map[emits-bound:14]
tests/integration/testdata/gallery-failing/src/main.can [2ab0518dcd5a] edits=4 roles=map[emits-bound:4]
tools/performance/fixtures/apps/src/invoice.can [a4aa40d82c88] edits=8 roles=map[emits-bound:8]
tools/performance/fixtures/apps/src/main.can [d0a990b0387d] edits=2 roles=map[emits-bound:2]
tools/performance/fixtures/apps/src/map.can [41df65dd3525] edits=4 roles=map[emits-bound:4]
tools/performance/fixtures/runtime/src/doubled.can [41df65dd3525] edits=4 roles=map[emits-bound:4]
tools/performance/fixtures/runtime/src/fold.can [97480702a13c] edits=4 roles=map[emits-bound:4]
tools/performance/fixtures/runtime/src/frequency.can [97bbfd8ab458] edits=4 roles=map[emits-bound:4]
tools/performance/fixtures/runtime/src/main.can [05083e65470d] edits=2 roles=map[emits-bound:2]
tools/performance/fixtures/runtime/src/paths.can [f5948c260867] edits=26 roles=map[emits-bound:26]
tools/performance/fixtures/runtime/src/recovery.can [9fdfb0595791] edits=2 roles=map[emits-bound:2]
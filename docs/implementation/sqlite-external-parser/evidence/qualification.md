# SQLite external parser qualification

Base commit: `79830698227dd2706271ff1427c107ef2325f32a` (branch
`codex/sqlite-external-parser`, clean except untracked
`docs/implementation/sqlite-external-parser/`).
Worktree: `/Users/vince/.codex/worktrees/db71/can-lang`.
Toolchain: `go1.27.1 darwin/arm64`, `bun 1.4.2`, `CGO_ENABLED=1`,
`GOPROXY=https://proxy.golang.org,direct`,
`GOMODCACHE=/Users/vince/go/pkg/mod` (shared cache only).

## A01 baseline footprint

- `compiler/internal/sql/sqlite/`: 88 tracked files, 6,729,649 bytes
  (`git ls-files | xargs wc -c` total). `sqlite_parser.c` alone is
  5,809,039 bytes (86.3%).
- Authored files in that directory: `glue.c` (3,571 B), `glue.go`
  (4,414 B), `glue_test.go`, `PROVENANCE.md`. Everything else is
  vendored tree-sitter runtime v0.27.0 + grammar sources +
  `unicode/` + `portable/` + `wasm-stdlib/`.
- Current compiler/runtime contracts (from
  `compiler/internal/sql/parser.go`, `sqlite.go`, `descriptors.go`):
  `Analysis{Statements, Sites, Failure, Version}`; `Version` is the
  grammar `LanguageVersion` (15) recorded on checked descriptors;
  statement count/kind/byte spans, parameter sites in source order
  with mapped numbers, top-level SELECT LIMIT shape
  (`HasLimit`/`LimitParam`), RETURNING presence, verbatim grammar
  failure (message + rune cursor). No upstream AST/error types cross
  the boundary.
- Bounded test surface: `go test ./compiler/internal/sql/` compiles
  and runs (smoke: `-run TestNothingMatchesThis` → ok, 0.310s).
  Focused files: `corpus_test.go` (161 lines),
  `descriptors_test.go` (153), `parser_test.go` (267),
  `sqlite/glue_test.go` (75), plus `differential_test.go`,
  `f03_returning_test.go`, `locking_probe_test.go`.
- `go.mod` currently requires only `pg_query_go/v6 v6.2.2` (+indirect
  tidb parser etc.); no tree-sitter/Meyer modules yet.
- Known provenance defects to correct later: PROVENANCE.md says "No
  Go module ships this grammar" (false: upstream has go.mod +
  bindings/go at the pinned commit) and omits `glue_test.go` from the
  authored list.

Commands (2026-09-28, all in worktree):
- `git rev-parse HEAD` → 79830698227dd2706271ff1427c107ef2325f32a
- `git status --short` → only `?? docs/implementation/sqlite-external-parser/`
- `git branch --show-current` → codex/sqlite-external-parser
- `git ls-files compiler/internal/sql/sqlite/ | wc -l` → 88
- `go test ./compiler/internal/sql/ -run TestNothingMatchesThis -count=1` → ok 0.310s

## A02 official modules (shared cache, no copies into Can)

Resolved via `GOPROXY=https://proxy.golang.org,direct`,
`GOMODCACHE=/Users/vince/go/pkg/mod` (shared cache only):

- Grammar: `github.com/defin/tree-sitter-sqlite3`
  `v0.1.1-0.20260503135658-7f69bb66845b`
  - `.info`: `Hash 7f69bb66845beaac48d467f4f7d107ea2002865e`,
    `Time 2026-05-03T13:56:58Z` — matches design commit.
  - `go.sum`: `h1:x3pc549tfnMjYry83v8IoU3Ve6oOZaUd3d1ZVN01okA=`,
    `/go.mod h1:bE0lvVJIK1OJ9By3fOmtrnX+jzlQ/7bzvByT3RguSS8=`
    (sumdb-verified through proxy).
  - `go.mod`: `module .../tree-sitter-sqlite3`, `go 1.22`,
    requires `go-tree-sitter v0.24.0` (lower bound only).
  - Archive contains full grammar sources: `src/parser.c`
    (`LANGUAGE_VERSION 15`), `src/scanner.c`,
    `src/tree_sitter/parser.h`, `bindings/go/binding.go`
    (`#include "../../src/parser.c"` + `scanner.c`, exports
    `Language() unsafe.Pointer`).
  - Upstream `src/parser.c` sha256
    `99a9cf64e174704f6d250a43155e6425af040840aed3eb12441d0474fe28557b`,
    `scanner.c`
    `3b9cb55a3b20ac40ca065b5a16d0d3f2c7e7558f7d4d54f3a1094a7bbce30c6b`,
    `parser.h`
    `180b893c8734778fd32f372dfbc27bd6ad1cd2221f26150b31256ff6716320d2`
    — all three byte-identical to Can's vendored copies.
  - License: SQLite-style public-domain blessing + CC0-1.0 fallback
    (`LICENSE`).
  - Upstream `binding_test.go` only wraps the pointer
    (`NewLanguage(...) != nil`) — pointer-only, as design states.
- Runtime: `github.com/tree-sitter/go-tree-sitter v0.25.0`
  - `.info`: `Hash adc13ffd8b2c0b01b878fda9f7c422ce0df5fad3`,
    `Time 2025-02-02T21:10:42Z`, `Origin.Ref refs/tags/v0.25.0`.
  - `go.sum`: `h1:sx6kcg8raRFCvc9BnXglke6axya12krCJF5xJ2sftRU=`,
    `/go.mod h1:r77ig7BikoZhHrrsjAnv8RqGti5rtSyvDHPzgTPsUuU=`.
  - `go.mod`: `go 1.23`; runtime dep
    `github.com/mattn/go-pointer v0.0.1` (+ testify and 12
    grammar modules used by upstream's own tests; pruned from
    consumers' builds, `.mod`-only in graph).
  - ABI admission: `include/tree_sitter/api.h`
    `TREE_SITTER_LANGUAGE_VERSION 15`,
    `TREE_SITTER_MIN_COMPATIBLE_LANGUAGE_VERSION 13`;
    `Parser.SetLanguage` accepts `13..15`, else `LanguageError`.
    Grammar ABI 15 is admitted (v0.24.0 runtime supports only 14
    per design; not re-verified, not needed).
  - License: MIT (`LICENSE`, Amaan Qureshi).
  - API surface needed exists: `NewParser/SetLanguage/Parse`,
    `Parser.Close`, `Tree.Close/RootNode`, `Node.Kind/IsNamed/
    StartByte/EndByte/HasError/IsError/IsMissing/ChildCount/
    Child/NamedChildCount/NamedChild/ToSexp`, `Language.AbiVersion/
    Metadata/NodeKindCount/ParseStateCount`.
- Live-tag provenance anomaly (recorded, blocks nothing):
  `git ls-remote https://github.com/tree-sitter/go-tree-sitter
  refs/tags/v0.25.0` returns EMPTY (exit 0) — the live GitHub tag
  is absent, while the proxy `.info` records
  `Origin.Ref refs/tags/v0.25.0` at `adc13ffd`. The module
  resolves reproducibly from the proxy + sumdb; provenance doc
  must state "proxy-resolved, live tag absent" rather than
  pointing at a GitHub tag page.
- Real parse probe (NOT pointer-only): temp module builds both
  modules, `SetLanguage` succeeds, `Parse("SELECT name ... :id
  LIMIT :lim")` yields `source_file/select_statement`,
  `bind_parameter` spans `[34,37)` `:id` and `[44,48)` `:lim`
  (identical to vendored `glue_test.go` expectations),
  `HasError=false`; `... WHERE x = $1` yields `HasError=true`.
  Result: `A02-PROBE-PASS`. Grammar reports `AbiVersion=15`,
  425 symbols, 5066 states, metadata 0.1.0.
- `go mod tidy` behavior note (CORRECTED 2026-09-28 per Codex review;
  earlier wording claimed an unexplained downgrade without a
  reproducible case — withdrawn): during A02, two tidy runs that
  executed while the probe module had no importing package files
  for the new requirements appeared to rewrite the pins
  (grammar→`v0.1.0`, runtime→`v0.24.0`). This was not reproduced
  in follow-up runs, and B01 demonstrated the underlying
  mechanism directly: `go get` + `go mod tidy` with no importing
  package drops the unused requirement, and the pin sticks once
  an import exists. No `go.work`, no exotic `GOFLAGS`.
  Verified instead: with importing packages present, `go mod
  tidy` + `go list -m` + rebuild + rerun confirm
  `v0.1.1-0.20260503135658-7f69bb66845b` + `v0.25.0` selected
  and the probe passing (A02–A03). Separately verified fact
  (unchanged): `diff -rq` between module zips `v0.1.0`
  (commit `ce5901b`) and `7f69bb`: ONLY
  `.github/workflows/publish.yml` differs — grammar C, headers
  and binding byte-identical.

Commands (workspace `/tmp/can-sqlite-qual-DmApBm`, registered above):
- `go mod download repo@7f69bb66845beaac48d467f4f7d107ea2002865e`
  → pseudo `v0.1.1-0.20260503135658-7f69bb66845b`
- `go mod tidy && go run .` → `A02-PROBE-PASS` (final verified run)
- `go list -m ...` → pseudo + `v0.25.0` selected
- `git ls-remote ... refs/tags/v0.25.0` → empty output, exit 0

## A03 official-binding differential (63/63 match)

Method: same-grammar differential across two binaries (one binary is
impossible: vendored + official tree-sitter runtimes collide at link —
258 duplicate `ts_*` symbols — and the `/tmp` module cannot import
`compiler/internal/sql`). Vendored side: transient
`compiler/tmpqualprobe/main.go` (`sql.Analyze`, worktree `go.mod`
 untouched — temp requires added then reverted via `git checkout`).
Official side: `/tmp` probe `adapter.go` (B02-candidate shape:
per-call parser, iterative capped walks, extraction before Close).
Both read the corpus JSON + shared `cases.json` (42 extra cases) and
print identical summary lines; `diff` over the summaries.
Raw matched outcomes: [a03-official-outcomes.txt](a03-official-outcomes.txt).

- Corpus: all 21 `lite-*` sqlite cases MATCH (statements, spans,
  sites, numbers, LIMIT shape, RETURNING, failures, Version 15).
- Extra: all 42 match — UTF-8 identifiers/literals/decoys (u01–u05),
  parameter spellings/reuse (p01–p08), CTE/nested (c01–c03), LIMIT
  forms incl. comments (l01–l06), RETURNING (r01–r04), empty/comment/
  separator/multi/error edges (e01–e09), statement kinds (k01–k07).
- Notable agreed grammar facts (inputs for A05/B03): `:1`/`@1`
  (p05/p06) and `$1` (lite-20) are syntax errors; `?0` maps to
  number 0 (p07); `:a`/`@a`/`$a` collapse to one number (p02);
  bare `?` after `?5` takes 6 (p04); UTF-8 bare identifier parses
  (u03); empty/comment-only/`;` inputs yield zero statements with
  NO failure (e01–e05); `SELECT..RETURNING` is a syntax error (r03).
- Probe bug found and fixed by the differential: `Child()` returns
  a fresh Go wrapper per call, so node-identity maps must key by
  byte span, not pointer (first run: all LimitParams 0; after fix:
  63/63). B02 must key by span (or resolve LimitParam in one walk).
- `IsMissing()`-based failure detection agrees with the vendored
  ERROR/MISSING-string walk on all error cases (e06/e07/lite-16/
  lite-17/lite-20/p05/p06/r03).
- Resource checks (`go run . res`, exit 0, no timing):
  - Parser reuse == fresh-parser results on 3 inputs
    (`REUSE-DETERMINISTIC`; descendant counts 22/18/1).
  - Extraction-before-Close verified; new parser works after
    closes (`CLOSE-OWNERSHIP-OK`).
  - `Parser.Close` is NOT idempotent: explicit Close + deferred
    Close segfaulted in `ts_parser_delete` (probe bug, runtime
    behavior confirmed by trace). B02 must Close exactly once.
  - Node-cap guard triggers a clean Go error with a 10-node
    override cap (`CAP-GUARD-OK`); 1M-cap equivalent of
    `SQLITE_DUMP_NODE_CAP` retained in adapter.
  - 40,035-byte expression chain: 60,025 visits, correct
    analysis; 5,000-deep parens: 15,010 visits, parses fine
    (iterative walks, no recursion risk).
- Pins held across all `/tmp` tidies in A03 (pseudo + v0.25.0);
  Can's `go.mod` also held them during the transient require
  (then reverted; `git status` clean).

Commands:
- `go run ./compiler/tmpqualprobe > vend.txt` (worktree, exit 0)
- `go run . > off.txt; cut -f1,3 off.txt | diff vend.txt -`
  → `ALL-63-MATCH` (before AND after res.go/override edits)
- `go run . res` → `RESOURCE-CHECKS-PASS`, exit 0

## A04 Meyer v0.1.2 qualification

Resolution (normal, shared cache): `go get
github.com/sqlc-dev/meyer@v0.1.2` → `.info Hash
59b2dbf9b1872316bc6c6d3379b500df84db3e01, Time 2026-08-25T20:44:20Z,
Ref refs/tags/v0.1.2`; `go.sum
h1:40Ng9Glnx7CTf3yOYV7jfvDIN7voIFjrStkIULS+uws=`,
`/go.mod h1:pS4USCRf/SLjWtaMcnTo4YrEEFKBj8CyyqlxcVUJQH8=`.
`go.mod`: `go 1.24`, ZERO dependencies (verified). MIT (`The sqlc
Authors`). `git ls-remote HEAD` == v0.1.2 hash (v0.1.2 is HEAD; proxy
list shows v0.1.0/v0.1.1/v0.1.2 only). Non-test sources ≈6,200 lines;
`parser/testdata` holds 1,892 entries; 11 parser `*_test.go` files
(corpus/fuzz/roundtrip/span/snapshot/options/trivia/errors/grammar).
Maintenance: sqlc main imports meyer (`parser`, `ast`, `token` in
`internal/engine/sqlite/parse.go`, verified via raw.githubusercontent
fetch) with `Options{UpdateDeleteLimit: true}` ("sqlc has always
accepted them"); latest sqlc RELEASE v1.26.0's `.mod` does NOT yet
require meyer (adoption is main-only, young dependency confirmed).

API inspected (all in-module, no cgo):
- Entries: `Parse(ctx,r)`, `ParseString`, `ParseFile` (+trivia),
  `ParseStatement` (exactly-one, rejects trailing), `ParseExpr`;
  `Options{UpdateDeleteLimit}` (zero value = pinned build).
- `Error{Message, Offset, SQL}`: messages match SQLite byte-for-byte
  (`near "X": syntax error` / `unrecognized token` / `incomplete
  input`); Offset is a BYTE offset (-1 unknown).
- AST: every node embeds `Span{Start,Stop}` (BYTE offsets into the
  input, load-bearing per package doc); `Node{Pos,End,Children}`;
  `ast.Walk` in source order... EXCEPT comma-LIMIT children come out
  as (Count=y, Offset=x) — see lite-14 below. No Visitor interface;
  Walk+Children is the traversal API.
- `BindParam{Span, Kind, Number, Name(no sigil), Raw}`; Number
  assigned like `sqlite3ExprAssignVarNumber`; `varNums` keyed by RAW
  (sigil INCLUDED — `:a`/`@a`/`$a` are distinct, unlike vendored).
  `?NNN` Atoi failure → Number stays 0 (vendored: -1).
- `Limit{Span, Count, Offset, Comma}`; comma spelling swaps operands
  (Count=y, Offset=x). LIMIT attaches per `oneselect` core (even
  mid-compound — see m07); last core holds the top-level one.
- RETURNING: `[]*ResultColumn` on Insert/Update/Delete.
- Statements: `parseScript` skips bare `;`, errors on missing `;`
  between statements, WIDENS each span to include the terminating
  `;` (differs from tree-sitter spans).
- Resource: `maxDepth = 10000` recursion guard → clean
  `Recursion limit` error (SQLite itself gives up ~2,500 per
  comment); binary-expr chains are iterative (100k-term chain OK).

Differential vs vendored (same 74 cases: 21 corpus + 53 extra):
53 identical modulo version/kind-spelling; 21 substantive diffs.
Raw: [a04-meyer-outcomes.txt](a04-meyer-outcomes.txt).
Vendored-official re-verified 74/74 on the extended set; the refreshed
[a03-official-outcomes.txt](a03-official-outcomes.txt) now has 74 lines.

Accept/reject divergences (need A05 engine arbitration):
- lite-20 `$1`: tree-sitter REJECTS; Meyer accepts (`$1`→1).
- p05 `:1`, p06 `@1`: tree-sitter REJECTS; Meyer accepts (→1).
- m01 `UPDATE..LIMIT 1`, m02 `DELETE..ORDER BY..LIMIT`: tree-sitter
  ACCEPTS; Meyer default REJECTS (`near "LIMIT"/"ORDER"`, option-gated).
- m04 `$name(arg)` TCL form: tree-sitter REJECTS; Meyer accepts
  (span covers `$name(arg)`).

Numbering divergences (both accept):
- p02 sigils `:a @a $a` + bare: vendored 1,1,1,2 (L2); Meyer
  1,2,3,4 (L4). Smallest case: `SELECT * FROM t WHERE a = :a AND
  b = @a` → vendored 1,1 vs Meyer 1,2.
- m05 `?99999999999999999999999` overflow: vendored -1, Meyer 0.
- lite-14 `LIMIT ?, ?`: vendored sites in source order
  (1,2,3); Meyer Walk yields (1,3,2) — operand-swap artifact.
  Adapter MUST sort sites by span (modest fix, probe-verified
  shape otherwise).

Span divergences (CORRECTED 2026-09-28 per decision.md item 2: native
semicolon spans are INTENTIONAL — the adapter keeps them and
expectations update deliberately; the earlier "adapter strips it"
wording is withdrawn; raw outputs preserved unchanged):
- lite-18/lite-05/e08/e09/k07: Meyer statement spans include the
  trailing `;` (+1 length). Kept as the native contract.
- (lite-05/e09 `L2` on multi-stmt first stmt is a probe-format
  artifact: multi-stmt skips limit computation downstream.)

Failure presentation (both reject; wording/cursor differ — no
compat requirement, shape preserved):
- lite-16 (`FORM`): vendored `near "*"@7`; Meyer `near "FORM"@9`.
- lite-17 (unterminated string): vendored `@34`; Meyer
  `unrecognized token` `@36`.
- e06 `SELECT` / e07 `SELECT * FROM`: vendored `@0`/`@9`; Meyer
  `incomplete input` `@6`/`@13`.
- r03 SELECT..RETURNING: vendored `@14`; Meyer `near
  "RETURNING"@16`. m03 `#1`: vendored `near "#"@7`; Meyer `near
  "#1"@7`.
- Meyer offsets are BYTES; Can's Failure.Cursor is a CHARACTER
  offset (`descriptors.go` renders `at character %d`; postgres.go
  documents character offsets). Adapter must convert
  (rune-count prefix) — recorded for B02/B03.

LIMIT attribution:
- m07 mid-compound `LIMIT 1 UNION`: BOTH accept; vendored
  HasLimit=true/L0, Meyer(last-core rule) HasLimit=false. If the
  engine rejects m07, both over-accept (A05 checks).

Agreements worth noting: all mapped kinds match (no unknown
`M:*` types across pragma/explain/create/vacuum/begin/commit);
UTF-8 spans u01–u05 identical; `?0`→0 then bare→1 (p07) identical;
CTE/nested limits, expression/comma/OFFSET limits, RETURNING incl.
params-in-RETURNING (r05/r06), empty/comment/`;` inputs (zero stmts,
no failure), and compound-last LIMIT (m08, L1) all identical.

Resource probes (no timing): 40,035-byte chain OK; 5,000-deep
parens OK; 15,000-deep → clean `Recursion limit` error at byte
10,006; 100k-term chain OK; `UpdateDeleteLimit:true` accepts m01.
Open question for A05: does real SQLite accept 5,000-deep nesting
(then both parsers match the engine) or reject ~2,500 (then both
over-accept, Meyer less)?

CORRECTION 2026-09-28 (decision.md item 6; Codex source review):
the A04 claim that parser depth alone makes recursive `ast.Walk`
safe is WRONG — flat left-associative operators form deep
`BinaryExpr` chains without increasing parser recursion depth
(`parseExpr` loops; `enter()` guards only expression/SELECT/
FROM-item recursion cycles). The B02 adapter therefore uses an
iterative `Node.Children` traversal with its own 1M-node budget
and never calls recursive `ast.Walk` on unbounded input. The
A04 probe's own `ast.Walk` use was confined to small fixed
corpus/case inputs (bounded by construction), so probe results
stand; only the safety generalization is withdrawn.

Commands: `go get github.com/sqlc-dev/meyer@v0.1.2`;
`go run ./meyer` (in /tmp module) → 74 lines; normalizer →
53 identical + 21 listed; `go run ./meyerres` → limits above.

## A05 Bun SQLite oracle (engine 3.54.0, Bun 1.4.2, :memory:)

Method: in-memory `bun:sqlite` probe ([a05-probe.ts.txt](a05-probe.ts.txt))
with schema pre-created (`t`,`u`,`users`), so PREP-FAIL with a
syntax/validation message = engine rejects, PREP-OK (+ `LIMIT NULL`
datatype-mismatch at RUN = unbound limit param, syntax accepted).
Raw: [a05-oracle.txt](a05-oracle.txt) (30 lines).

Engine verdicts on the A04 divergences (engine = truth):
- `$1` (lite-20), `:1` (p05), `@1` (p06): PREP-OK and bindable
  (`{"$1":9}`→9 etc.). Tree-sitter WRONGLY rejects all three;
  Meyer CORRECT. (Old `sqlite.go` comment "the grammar rejects
  them as syntax errors" describes grammar behavior, not the
  engine — do not enshrine.)
- Sigils `:a`/`@a`/`$a`: DEFINITIVELY DISTINCT — independent
  binding yields 1,2,3; `:a IS @a` with only `:a`=5 bound is
  false(0). Vendored collapse (p02 → 1,1,1,2) is a BUG; Meyer
  (1,2,3,4) CORRECT. B03 must key by raw-with-sigil on either path.
- `UPDATE..LIMIT` (m01), `DELETE..ORDER BY..LIMIT` (m02): PREP-OK
  and RUN-OK — Bun's SQLite build accepts. Tree-sitter accepts
  ✓; Meyer DEFAULT rejects ✗. Meyer path MUST set
  `Options{UpdateDeleteLimit: true}` (same as sqlc) — B02 input.
- `$name(arg)` TCL form (m04): PREP-OK RUN-OK. Tree-sitter
  wrongly rejects; Meyer correct.
- `?0` (p07), `?99999…` overflow (m05): engine PREP-FAILs both
  (`variable number must be between ?1 and ?500000`). BOTH
  parsers over-accept (vendored -1, Meyer 0 — both meaningless).
  Open contract point: grammar-accept vs engine-reject (A06).
- Mid-compound `LIMIT 1 UNION` (m07): engine rejects (`LIMIT
  clause should come after UNION not before`). BOTH parsers
  over-accept (vendored HasLimit=true/L0; Meyer last-core rule
  HasLimit=false). Open contract point (A06).
- Nesting: engine accepts 2,000-deep, rejects 5,000-deep
  (`Recursion limit`). BOTH parsers accept 5,000-deep
  (over-accept; Meyer rejects ≥~10,000 cleanly). Minor
  hostile-input point (A06).
- Agreements confirmed: bare-after-`?2` is #3 (max+1 ✓ both);
  `SELECT..RETURNING`, `#1`, `FORM`, unterminated string,
  bare `SELECT` all engine syntax errors (both reject);
  UTF-8 bare identifier is syntax-OK (schema error only).
- Meyer failure messages match the engine BYTE-FOR-BYTE on all
  five shared rejections (`near "FORM"`, `unrecognized token…`,
  `incomplete input`, `near "RETURNING"`, `near "#1"`, incl.
  byte offsets 9/36/6/16/7). Tree-sitter-derived messages differ
  (e.g. `near "*"@7` vs engine `near "FORM"@9`).

Scorecard (accept/reject + numbering vs engine, Bun build):
- Tree-sitter grammar: 4 over-strict (`$1`,`:1`,`@1`,`$name(arg)`)
  + sigil-numbering bug; right on UPDATE..LIMIT.
- Meyer DEFAULT: 2 over-strict (UPDATE/DELETE..LIMIT) + shared
  over-acceptances; right on `$1`/sigils/TCL.
- Meyer + `UpdateDeleteLimit:true`: matches the engine on every
  probe EXCEPT the four shared over-acceptances (`?0`, overflow
  NNN, mid-compound LIMIT, >2k nesting) — all prepare-validation/
  nesting behavior beyond pure grammar, needing an explicit
  contract decision either way.

No blind agreement with old bugs: the oracle OVERTURNS the old
sigil-collapse and `$1`-rejection behaviors (both were
tree-sitter-grammar artifacts, contradicted by the engine).

Command: `bun probe.ts` in `/tmp/can-sqlite-qual-DmApBm/a05` → exit 0.

## A06 candidate comparison and recommendation (Codex decides)

Raw checks: §A02 (pins/provenance/parse), §A03 + 74-line
[a03-official-outcomes.txt](a03-official-outcomes.txt) (official
differential), §A04 + 74-line
[a04-meyer-outcomes.txt](a04-meyer-outcomes.txt) (Meyer differential),
§A05 + 30-line [a05-oracle.txt](a05-oracle.txt) (engine 3.54.0 oracle).

| Axis | Official pair (grammar 7f69bb + runtime v0.25.0) | Meyer v0.1.2 |
|---|---|---|
| Engine correctness (A05) | 4 over-strict (`$1`,`:1`,`@1`,`$name(arg)` wrongly rejected); sigil-collapse numbering bug (B03-fixable); right on UPDATE..LIMIT | Right on `$1`/sigils/TCL; default over-strict on UPDATE/DELETE..LIMIT (fixable via `UpdateDeleteLimit:true`, as sqlc does) |
| Shared over-accept vs engine | `?0`, `?NNN` overflow, mid-compound LIMIT, >2k nesting accepted though engine rejects | Same four (same contract question either way) |
| Failure messages | Tree-derived (`near "*"@7` etc.), differ from engine | Byte-for-byte engine wording + offsets on all 5 shared rejections |
| Differential vs current | 74/74 identical (zero behavior change) | 53/74 identical; 21 diffs: 6 accept/reject, 3 numbering (sigils, overflow, Walk-order), 5 `;`-spans (incl. lite-05/e09 probe-L artifact), 6 failure-message, 1 LIMIT attribution |
| Adaptation for Can contract | Span-keyed limit lookup (probe-found), single-Close discipline (non-idempotent), 1M node cap retained, sigil fix (B03); grammar rejections UNFIXABLE without fork | Sort sites by span (comma-LIMIT), KEEP native `;` stmt spans (intentional per decision; corrected), byte→rune failure cursor, kind mapping, last-core LimitParam rule, `UpdateDeleteLimit:true`, iterative traversal (recursive Walk UNSAFE for flat chains — corrected); all enumerated, shapes probe-verified |
| Resource behavior | Per-call parser; iterative C + capped Go walks; 40KB/5k-deep OK; double-Close segfaults (must Close once) | maxDepth=10000 → clean `Recursion limit` error; binary chains iterative (100k OK); Walk recursive but only runs on depth-capped parses; no cgo at all |
| Dependency identity | 2 modules + `mattn/go-pointer`; runtime test-only graph leaves; grammar go.mod lower-bounds runtime at v0.24.0 (ABI14) while v0.25.0 required; live v0.25.0 GitHub tag ABSENT (proxy-only provenance) | 1 module, ZERO deps, MIT; v0.1.2==HEAD (3 tags total); sqlc-main adoption (releases not yet) — young dependency |
| C build | Keeps SQLite cgo (grammar 5.8MB C + runtime C + bridge) | Removes ALL SQLite cgo (pure Go); PostgreSQL cgo remains either way |

Recommendation (Muse, advice only): **Meyer with
`Options{UpdateDeleteLimit: true}`**. It matches the engine on every
probed divergence once the option is set, removes the SQLite C
tool-chain entirely, has byte-exact engine failure messages, and its
required adaptations are modest, fully enumerated, and shape-verified
by the probes. The official pair is a credible fallback (bit-identical
behavior, mature grammar tables) but perpetuates four
engine-contradicted rejections that only a fork could fix, keeps the
cgo bridge with its single-Close hazard, and carries the
absent-live-tag provenance wrinkle.

Open limitations / questions for `decision.md`:
1. `?0` / `?NNN`-overflow: engine rejects at prepare; both parsers
   accept. Should Can reject (engine view, needs explicit check on
   either path) or accept (grammar view)? Smallest: `SELECT ?0`.
2. Mid-compound `LIMIT 1 UNION`: engine rejects; both accept
   (vendored HasLimit=true, Meyer last-core HasLimit=false). Reject
   explicitly, or accept with which LIMIT attribution?
3. Deep nesting (>2k): engine rejects; both accept (Meyer caps at
   10k). Accept over-acceptance as hostile-input tolerance?
4. Meyer pin: `v0.1.2` (recommended). Tree-sitter fallback pins:
   grammar pseudo `v0.1.1-0.20260503135658-7f69bb66845b` vs tag
   `v0.1.0` (identical grammar bytes; tag is cleaner).
5. Version identity (B05): what replaces `Version 15` on checked
   descriptors (content/version naming, not ABI-as-identity)?
6. `ParseStatement` vs `ParseString`: Can needs statement
   count+spans for multi-statement detection → `ParseString` (or
   `ParseFile`); confirm multi-statement stays a descriptor-level
   rejection, not a parse error.

## Temporary workspace registry (cleanup tracking)

- `/tmp/can-sqlite-qual-DmApBm` (created 2026-09-28 14:57, owner: this
  session). Purpose: throwaway Go probe module for A02–A05 (no repo
  writes, no dependency copies retained). Cleanup: `rm -rf` at D04 or
  on failure/interruption; binaries never kept. Status: RECLAIMED
  2026-09-28 (`rm -rf` verified; compact probe sources preserved as
  evidence/probe-*.txt + probe-cases.json, 24,253 bytes total).
- `compiler/tmpqualprobe/` in worktree (created 2026-09-28, owner:
  this session; CORRECTED — an earlier `~15:20` timestamp was a
  guess, removed). Purpose: TRANSIENT differential runner that
  must live inside the module to import `compiler/internal/sql`
  (Go internal rule); copies of `/tmp` probe sources. Cleanup:
  `rm -rf` + `git checkout -- go.mod go.sum` at D04 (or right after
  A04 if unneeded); probe sources preserved in `/tmp` workspace
  and compacted into evidence. Status: RECLAIMED 2026-09-28
  (`rm -rf` verified; main.go preserved as
  evidence/probe-vendored-run.go.txt; worktree go.mod/go.sum carry
  only the production Meyer pin).

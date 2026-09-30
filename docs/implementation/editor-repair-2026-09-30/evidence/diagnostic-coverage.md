# F01 diagnostic-producer inventory (baseline 9c283544)

Scope: every authored-source diagnostic family plus warning/note producers, bare-error and shared-validation
sites. Codes/severity/spans below are observed baseline behavior; F02+ defines the target contract.

Counts (non-test Go): `fmt.Errorf/errors.New` = 1057 sites across check/resolve/types/project;
`source.Locate/LocateCode/Relate/Suggest` = 51 sites. Gap = most semantic failures are bare or wrapped
without structured spans; driver then anchors them to line one.

## 1. Lexical — `compiler/internal/syntax/lexer.go`, `token.go`

- Codes: `CAN-LEX-BOM/TAB/CR/WHITESPACE/INDENT/CONTINUATION/DELIMITER/COMMENT/STRING/NUMBER/
  FLOAT-RANGE/NAME/PUNCTUATION` (13 families; `grep -rho` counts in evidence).
- Severity: error (syntax `Diagnostic`, no severity field; driver maps to error).
- Span: exact token/offending bytes via `fail(code,msg,start,end)`; delimiter fallback = last unclosed opener.
- Related/fix: none.
- Tests: `lexer_test.go`, `parser_test.go` (valid + first-failure cases).
- Recovery gap: `Lex` stops at first diagnostic (`len(Diagnostics)==0` loop guard); unterminated
  string/comment/nesting discards later tokens; `Parse` refuses any lexed result with diagnostics.
  No shadow lexer permitted; P10 must add bounded forward progress + multi-finding collection.

## 2. Parser — `syntax/{parser,declarations,expressions,native,matches}.go`

- Codes: single `Code:"syntax"` via `parseFailure{Diagnostic}`; message = `p.fail(message)` at `p.peek().Span`.
- Severity: error. Span: one-token peek span (often correct token, but single finding only).
- Related/fix: none.
- Tests: `parser_test.go`, `declarations_test.go`, `expressions_test.go`, `native_test.go`, etc.
- Recovery gap: `Parse` catches one panic and discards partial `File` (`result = ParseResult{Diagnostics:...}`).
  `ParseType`/`ParseExpression` same single-failure shape. No per-declaration/field/argument/statement
  boundaries. P11 must retain valid nodes + explicit invalid regions, synchronize on token/depth/indent.

## 3. Project/config — `project/{graph,overlay,manifest,manifest_sql,lock,json,assets}.go`

- Source errors: `SourceError{Path,File,Diagnostics,Message}`; `load()` returns first bad source only
  (`parsed.Diagnostics[0].Format(file)` at graph.go:278); driver `loadDiagnostics` renders only
  `Diagnostics[0]` and widens multiline spans to whole line (diagnostics.go:236-246).
- Manifest/lock/registry: `ParseManifest/ParseLock/ParseRegistry`, `readConfined`, lineage/edge/path/target/
  digest/registry-snapshot checks — all bare `fmt.Errorf` (graph.go:98-463, ~30 sites), no JSON key/value spans.
- Assets/fixtures: `assets.go`, `manifest_sql.go` descriptor checks — bare errors, no Can-token attribution.
- Severity: error (load failure). Related: none. Overlays: `LoadWithOverlay` substitutes bytes but still
  first-failure; overlay-only files discovered via walk substitution (P13 must preserve + aggregate).
- Tests: `graph_test.go`, `manifest_test.go`, `fixtures_test.go`, `overlay_test.go`, `assets_test.go`,
  `error_identity_test.go`.
- Recovery gap: truly locationless I/O (unreadable root, symlink escape, cycle, depth>256) needs explicit
  project diagnostic/status, never an invented Can line-one squiggle (D2/D3).

## 4. Resolver — `resolve/{symbols,native}.go`

- Producers: `Build(graph)` phases — define names (located duplicate), generated question records, exports
  (`provides not declared`/`duplicate provides`), imports (`resolveImport`), signatures. Located via
  `located(src, span, fmt.Errorf("%s: %w",...))` using declaration-name/export/import/signature token spans.
- Codes: none (empty `Code`); severity error. Related/fix: none.
- Tests: `symbols_test.go`, `native_test.go`, `instances_test.go`.
- Recovery gap: shared mutable `World` (Prelude/Packages/Files/Functions/NativeScopes); first `return nil, err`
  aborts all later symbols; failed commits not rolled back; imports/declarations missing must not cascade
  into invented unknown-name errors (R20 transactional + blocked identities).

## 5. Types — `types/{builder,annotations,compat,specialize,...}.go`

- Producer: `Builder{..., failure error}` — `Resolve` returns `b.failure` sticky (builder.go:51-52,59,300-305,347);
  `Finish/SeedDeclarations` same. Declaration/field/error-bound failures are bare errors with declaration context.
- Codes: none for types (check-side `CAN-CHECK-*` only). Severity error.
- Tests: `builder_test.go`, `infer_test.go`, `specialize_test.go`, `schema_test.go`, etc.
- Recovery gap: one bad type/field poisons all later `Resolve` calls; recursive/cyclic components need SCC-ordered
  validation with explicit valid/invalid/blocked per-unit state (R21). Invalid types must never become typed facts.

## 6. Checker contexts — `check/{program,specialize,templates,action_bindings,coordination}.go`

- Producer: `checkProgramForTarget` → `ErrorDeclarations`, per-declaration checks, specialization queues,
  registry/SQL-site/warning collectors sharing `bindings/callables/specializations/registry/warnings/sqlSites`.
- Severity error; spans where `source.Locate*` applied (51 sites total), else bare.
- Tests: `program_test.go`, `specialize_test.go`, `templates_test.go`, `checks_test.go`, acceptance A01/A02/A04.
- Recovery gap: shared mutable checker state; failure in one body/specialization aborts unrelated bodies.
  R22 must refactor into independently committed contexts with explicit blocked prerequisites, reusing canonical
  validators (no whole-project replay or source masking).

## 7. Expressions/bodies — `check/{expressions,calls→callables,region→patterns/locals/infer,completion*,match→patterns}.go`

- Top bare-error files: `expressions.go`(47), `actions.go`(39), `completions.go`(37), `completion_matches.go`(30),
  `patterns.go`(24), `callables.go`(20), `arguments.go`(17), `infer.go`(13), `locals.go`(9), `methods.go`(9).
- Current spans: mixed — some `Locate`d callee/name spans, but missing-callee case underlines whole call
  (`driver/diagnostics_test.go:148` expects 7..27 on `ok call missing_fn(row)`; target = callee-only per R02).
  Operator AST fields retain text only (design-established); qualifier/member token spans missing in places.
- Severity error. Related: sparse. Tests: `checks_test.go`, `native_test.go`, per-file `*_test.go`.
- Recovery gap: must migrate to structured codes + minimal callee/operator/member/argument/type spans,
  continue independent child/statement checks, suppress invalid-type cascades (R23).

## 8. Assertions/native contracts — `check/{assertions,native*,http,fetch,form,llm,questions→scenarios?,wrap}.go`

- Top files: `assertions.go`(56), `wrap.go`(31), `action_bindings.go`(31), `form.go`(26), `native_bodies.go`(19),
  `action_fetch.go`(18), `http.go`(13), `fetch.go`, `connections.go`(3), `native.go`(2).
- Known gaps: duplicate assertion labels can be unlocated (line-one fallback); native result/emits/connection/
  fetch/HTTP/form validation often bare; `raw_fixture.go`(57) fixture-definition errors need row/field spans.
- Severity error. Related: duplicate origins must become `relatedInformation` (R24), not second primaries.
- Tests: `assertions_test.go`, `native_test.go`, `http_test.go`, `fetch_test.go`, `form_test.go`, `wrap_test.go`,
  `raw_fixture_test.go`, `actions_*_test.go`.
- Recovery gap: carry declaration/row/contract-field locations; continue independent contexts (R24).

## 9. SQL/global — `check/{sql,sql_sites,sql_descriptors,sql_specialization,program,scenarios,assets}.go`

- Producers: `sql.go`(5), `sql_sites.go`(5), `sql_descriptors.go`(6), `sql_specialization.go`(7),
  `scenarios.go`(6), `assets.go`(4), `aggregate.go`(4), plus `errors.go`(18) error-registry joins.
- Current: cross-site records lack source file/span; global contract validation bare; F01 residual families
  (codec/crypto/transaction/stream/websocket/browser/judge/initialization) also bare.
- Severity error. Tests: `sql_test.go`, `sites_test.go`, `assets_test.go`, `server_test.go`, etc.
- Recovery gap: add file/span to cross-site records, validate only complete prerequisite sets, distinguish
  blocked work (R25). No authored-source bare error may remain unaccounted after R25.

## 10. Warnings/notes — `check/warnings.go`, `check/*` Warn callbacks, `driver/diagnostics.go:121`

- Type: `check.Warning{Severity,Code,File,Line,Column,Span,Message}`; severities `warning`/`note`.
- Codes observed: `CAN-CHECK-UNNECESSARY-LOCAL` (advisory alias), plus `CAN-CHECK-CAPTURE/EXPORTED-GENERIC/
  FIXTURE-USE/EXACT-SPECIALIZATION/OUTWARD-ERROR/MISSING-ARM/UNKNOWN-PATTERN/NOT-LOWERED/
  FIXTURE-DEFINITION/BOUND-CYCLE` (counts in F01 grep evidence; some are error codes reused in tests).
- Baseline bug: `warningDiagnostics` hardcodes `Severity:"warning"`, discarding `note` vs `warning`
  (design §D1/R04). Also `CheckSnapshot` returns warnings only on success — warnings vanish under any error.
- Tests: `warnings_test.go` (driver), check warning collection tests.
- Target (R26): preserve note-vs-warning, deterministic order/dedup, related locations; errors+warnings coexist;
  any cap explicit and never masquerading as complete.

## 11. Driver bridge — `driver/{diagnostics,fixes,hover}.go`, `compiler/lsp.go:378-433`

- `CheckSnapshot`: load→resolve→check, returns after first stage error with single diagnostic; warnings only
  when all stages succeed. Must become single canonical recoverable analysis consuming partial graph/world/index
  with all trustworthy issues (R26).
- `semanticDiagnostic`: converts one `LocatedError`; spanless → `anchored()` first-line fallback; invalid offsets
  → `CAN-SPAN-UNAVAILABLE` marker (correct explicitness, keep).
- `loadDiagnostics`: first-diagnostic-only + multiline whole-line widening (must go; R02 preserves end line and
  zero-width insertion ranges).
- `anchored`: longest-path-substring → open file → first loaded source, width = first-line UTF-16 width.
  Unexplained first-line fallbacks to be removed except explicit project-status diagnostics (D2).
- `Definition` (diagnostics.go:376+): file/package/prelude/import scopes; body-local names decline; catalogue/
  prelude sourceless decline. Editor lane (E30+) extends via shared index.
- Tests: `diagnostics_test.go`, `fixes_test.go`, `warnings_test.go`, `lsp_*_test.go` (G01-G05).

## 12. Fixes — `source.Fix`, `driver/fixes.go` (`SuggestedFixes`/`ValidateFix`)

- Shape: insert-only (`Start==End`) single-range repairs with title/file/text/version; validation rechecks
  isolated overlay snapshot, preserves contracts, resolves target without new errors.
- Baseline: compiler proposes/validates but LSP never exposes (design-established). E35 wires to code actions.
- Tests: `fixes_test.go`.

## Old tests needing replacement (not preservation)

- `driver/diagnostics_test.go:148` — whole-call span 7..27 for missing callee; replace with callee-only span.
- Any assertion of `len(Diagnostics)==1` under two independent faults (parse/resolve/check first-error goldens).
- Multiline-syntax whole-line widening expectations (driver loadDiagnostics branch).
- Notes-mapped-to-warnings expectations (warning severity hardcode).
- `anchored()` first-line fallback expectations for located-able failures (keep only explicit project-status cases).
- `Lex`/`Parse`/loader single-diagnostic goldens that forbid later independent findings.
- `types.Builder` sticky-failure propagation tests (replace with per-unit invalid/blocked behavior).
- Completion-declines-malformed-buffer tests (E32 requires useful recovered completion).

## Coverage claim

Syntax, imports/exports, type/error declarations, expressions, assertions, native/HTTP/fetch/SQL and config
families all mapped above to producers with code/severity/span/related/test. Internal/project-only failures
(unreadable roots, cycles, depth limits, lock/registry mismatches, symlink escapes) are called out in §3/§9 as
explicit project diagnostics, not Can squiggles. No other first-line fallback remains unexplained.

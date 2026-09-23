# Language-fixes implementation evidence

Implementation evidence for LF01–LF21 accumulates here. Each task adds its
`LFnn-*` result files; the per-task sections below record what was run,
with exact commands, tool identities, and actual outcomes.

Planning artifacts (order, dependencies, coverage) live in
`../language-fixes-plan/`; that validation is not implementation evidence.

## LF01 — revision inputs and acceptance baseline

- `LF01-baseline.json`: starting commit, working-tree state, drift analysis,
  canonical specification hashes, catalogue hash, pinned Bun identity, frozen
  input references, reproduction commands, and negative gates.
- `LF01-controls.json`: bounded G1–G4 control rerun results (15 CLI/LSP
  invocations with exits, timings, current-generation snapshots, diagnostics).
- `LF01-probes.py`: scratch runner; reads the frozen gap projects and writes
  only to scratch paths. Original evidence untouched.
- `LF01-AE-results.json`: result manifest for all 16 AE records. Every record
  is `evidence-required` with empty implementation-evidence slots.

Exit summary: starting commit `7b1542c`, clean tree; drift from the
verification baseline `760405f` is docs-only (126 files, no compiler/runtime
change). Eight-fetch baseline hash resolves. G1/G2/G3/G4 controls reproduce
the original observations on this checkout; G5 was intentionally not rerun
and stays the resource regression constraint.

## LF02 — precise semantic source spans

- `LF02-summary.json`: span-carrying diagnostics design and per-file changes.
- `LF02-lsp.json`: LSP diagnostic ranges incl. astral and unavailable spans.
- `LF02-lsp.py`: LSP probe runner reused by later G4 reruns.
- `LF02-gotest.log`: full suite; documents the pre-existing
  `TestCoreConsumerSpecification` docs-drift panic on an untouched file.

Exit summary: G4 semantic/resolve diagnostics highlight line 8; lexical
control unchanged; astral/overlay/imported/unavailable spans covered.

## LF03 — assertion staging vs production publication

- `LF03-summary.json`: Stage/SelectCurrent split with generation leases.
- `LF03-g2.json`: G2 reproducer results (assert never selects current).
- `LF03-g2.py`: G2 runner.

Exit summary: assert never selects `current` on success, failure, or kill;
driver suite green.

## LF04 — supervised assertion roots

- `LF04-summary.json`: per-root workers with 5000ms default budget.
- `LF04-assert.json`: CLI/LSP assert-run results incl. crash, protocol,
  and late-pass judgments.
- `LF04-assert.py`: assert runner.
- `LF04-gotest.log`: full suite.

Exit summary: G3 returns bounded failure reports; init-failure shape and
broken-pipe contracts preserved; assert/coordination/fetch/generation/judge
suites green.

## LF05 — exact generic-error heads and aliases

- `LF05-summary.json`: exact heads plus aliases across parse/check/race.
- `LF05-emit.txt`: emitted match lowering for exact specializations.
- `LF05-gotest.log`: full suite.
- `LF05-integration.log`: staged integration run.

Exit summary: 25/25 fixture roots incl. nested outer aggregate; 9 negative
subtests plus span and generic-spec tests; all 23 bundled suites plus
strict tsc green.

## LF06 — failure arms before final success

- `LF06-summary.json`: ok-final enforcement and arm-order diagnostics.
- `LF06-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF06-integration.log`: integration+driver+emit all ok.
- `LF06-runtime.log`: 321 pass.
- `LF06-emit-diff.txt`: branch-order-only diff; settle multiset identical.

Exit summary: success-first completion matches reject; reproduction commands
in the summary.

## LF07 — standard snapshots in ordinary catches

- `LF07-summary.json`: snapshot observation rules and binder rejections.
- `LF07-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF07-integration.log`: integration+driver+emit all ok.
- `LF07-runtime.log`: 321 pass.

Exit summary: stable same-run occurrence/kind/message; constructor, update,
emits, and string binders reject; selected-handler faults escape.

## LF08 — named runtime checks, arithmetic traps migrated

- `LF08-summary.json`: named checks plus trap migration.
- `LF08-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF08-integration.log`: integration+driver+emit all ok.
- `LF08-runtime.log`: 326 pass.
- `LF08-emit.txt`: single-evaluation lowering plus runtime branch.

Exit summary: checks carry names, spans, and invocation paths; no double
evaluation; no silent reason changes.

## LF09 — failure origin at native boundaries

- `LF09-summary.json`: origin capture at native operation boundaries.
- `LF09-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF09-integration.log`: integration+driver+emit all ok.
- `LF09-runtime.log`: 332 pass.
- `LF09-traces.json`: 9 private occurrence traces for LF10/LF12.

Exit summary: every native failure keeps occurrence identity, original
cause, and source/invocation path.

## LF10 — normalized fetch/judge infrastructure errors

- `LF10-summary.json`: public error normalization design.
- `LF10-gotest.log`: 15 ok; only the pre-existing docs-drift panic proven
  on a clean HEAD.
- `LF10-integration.log`: integration+driver+emit all ok.
- `LF10-runtime.log`: 335 pass.
- `LF10-normalize.log`: 17 pass.

Exit summary: one `request_failed` with unchanged typed detail, sanitized
fields, and new mapped/private original occurrence.

## LF11 — native assertion roots and raw exchange fixtures

- `LF11-summary.json`: attached roots plus raw-provider fixtures.
- `LF11-gotest.log`: 15 ok; only the pre-existing docs-drift panic proven
  on a clean HEAD.
- `LF11-integration.log`: integration+driver+emit all ok.
- `LF11-runtime.log`: 344 pass.
- `LF11-raw.log`: 20+6+3 focused pass.

Exit summary: native assertions attach to roots; raw exchanges replay
byte-identically with recorded evidence labels.

## LF12 — derived wrappers, calculated bounds, handler tests

- `LF12-summary.json`: wrapper derivation and calculated-bound design.
- `LF12-check.log`: 6 tests plus 13 reject subtests pass.
- `LF12-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF12-integration.log`: integration+driver+emit all ok.
- `LF12-runtime.log`: 349 pass.
- `LF12-policy.log`: 30 pass.
- `LF12-emit.txt`: dispatcher plus inherit-rule plus injection lowering.

Exit summary: 3-generation fetch chain plus judge wrapper plus callable
consumer in `compiler/testdata/current`.

## LF13 — typed fixture templates at lexical owners

- `LF13-summary.json`: template expansion with exact identity match.
- `LF13-check.log`: 10 tests plus 15 reject subtests pass.
- `LF13-syntax.log`: 4 pass.
- `LF13-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF13-integration.log`: integration+driver+emit all ok.
- `LF13-runtime.log`: 349 pass.

Exit summary: `compiler/testdata/current/templates` staged 13-row suite
(fn/generic/fetch templates, same-directory import, raw-in-template,
FIFO helpers, production-erased output).

## LF14 — complete verification inputs, locked raw fixtures

- `LF14-summary.json`: capture rules and the P15.1 framing oracle.
- `LF14-check.log`: project 11 plus driver 4 plus check 11 top-level with
  reject/source-tag subtests pass.
- `LF14-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF14-integration.log`: integration+driver+emit all ok.
- `LF14-runtime.log`: 349 pass.

Exit summary: attached/when/nested/template capture with dedup and
relocation-stable bytes; 5 capture rejects; captured bytes gate builds.

## LF15 — atomic production publication on all-root verification

- `LF15-summary.json`: P15.1 pipeline with every-root-required semantics.
- `LF15-check.log`: driver 43 plus check raw-fixture pass; 2 staged-only
  skips.
- `LF15-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF15-runtime.log`: 346 pass.
- `LF15-integration.log`: staged integration+driver+emit all ok.

Exit summary: G1/G2 frozen projects; locked-vendor 4-root graph build/run
plus wrong-vendor rejection; production purity (no assertions tree or
runner import); no live traffic during verification.

## LF16 — explicit native aggregate mapping

- `LF16-summary.json`: converter plus map-call lowering design.
- `LF16-check.log`: coordination/exact-head/new rejections/array suites pass.
- `LF16-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF16-runtime.log`: 347 pass.
- `LF16-integration.log`: staged integration+driver+emit all ok.
- `LF16-diff-main.can` + `LF16-diff-report.json`: 15-row recursive-vs-map
  differential incl. real snapshot occurrence compare, full pass plus
  verified build plus run.

Exit summary: `widen_one`/`widen_b_one` converters plus map call sites
plus explicit reconstruction replace 28 recursive lines; covariance,
omitted-leaf, and callback-contract negatives reject.

## LF17 — obligation diagnostics and checker-validated fixes

- `LF17-summary.json`: seven coded obligation classes and the fix-validation
  contract (insert-only, isolated snapshot, clean recheck).
- `LF17-check.log`: 15 obligation suites pass.
- `LF17-driver.log`: snapshot plus fix-validator suites pass.
- `LF17-runtime-focused.log`: assertion/barrier/execution files pass.
- `LF17-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF17-runtime.log`: full runtime suite pass.
- `LF17-g4.json`: G4 LSP probe rerun, identical highlights.
- `LF17-integration.log`: staged trio ok x3.

Exit summary: missing-arm, outward-error, fixture use/definition,
exact-specialization, capture, and bound/override-cycle diagnostics
carry primary/secondary spans; mismatch reports carry root/site/
invocation without value leaks (placeholder origins null); review
enforces the documented insert-only fix invariant with a test.

## LF18 — trivia-preserving safe formatting

- `LF18-summary.json`: single-renderer anchors, trivia attachment rules,
  and the `canlc format` validation contract.
- `LF18-syntax.log`: 7 trivia suites pass (46-file round-trip: 45 pass
  plus 1 intentional negative skip).
- `LF18-check.log`: structural equivalence plus arm-order negative pass.
- `LF18-cli.log`: 6 CLI tests plus retirement gate pass.
- `LF18-gotest.log`: 15 ok; only the pre-existing docs-drift panic.
- `LF18-runtime.log`: 353 pass, 0 fail.
- `LF18-integration-focused.log`: staged format integration passes.
- `LF18-integration.log`: staged trio ok x3.

Exit summary: 70 anchored emission sites; comments/blanks/strings survive
with exact attachment; output idempotent over CRLF/UTF-8; `--write`
validates through overlay snapshots and replaces atomically; misordered
arms preserved textually, never repaired.

## LF19 — capability and language guides

- `LF19-summary.json`: AE33/AE47 coverage, link tally, no-behavior-change
  gate.
- `LF19-corpus.log`: 20/20 guide examples behave as expected.
- `LF19-corpus-results.json`: per-case manifest with passed/total tally.

Exit summary: admission guide with worked pure and resource capabilities,
rejection boundary, and incomplete-proposal verdict; finite-rules doc
with verified examples, observed Bun-1.4.2 edges, and manifest/
build-assert/renderer corrections; F7/F8/F9 quotes labelled; corpus
under `docs/implementation/examples/2026-09-22/lf19/`.

## LF20 — equivalent programs and authoring workflows

- `LF20-summary.json`: comparison counts, gates, cost split, apps,
  scenarios, mutations, and workflow record.
- `LF20-fetch-gate.log`: `TestCurrentBundledFetch` passes.
- `LF20-apps-gate.log`: `TestApplicationsStaged` passes.

Exit summary: 63→9 bound entries, 56→8 arms, 8/8, 9 decls, 7/7 details
with gates green; E1–E4 green with traces; M1–M4 detected and repaired;
report plus rerunnable `measure_after.py` under
`docs/syntax-taste/evidence/2026-09-22/lf20-after-metrics/`.

## LF21 — integrated qualification and closed evidence

- `LF21-summary.json`: suite tallies, E2E flow, exceptions, release note.
- `LF21-qualification.json`: 16/16 AE audited to tasks, 24/24 BC audited
  to tests.
- `LF21-gotest.log`: 15 ok; only the pre-existing docs-drift panic proven
  identical on pristine HEAD `6b683ba`.
- `LF21-runtime.log`: 353 pass, 0 fail.
- `LF21-trio.log`: staged trio ok x3.
- `LF21-g5.log`: 8/8 resource-lifetime plus sql-escape pass.
- `README.md` (this file): the per-task evidence index.

Exit summary: every accepted AE record has a complete result manifest;
E2E normalized→wrapper→template→build→run green with old generation
preserved on failure; catalogue clean; no upload/signing performed.

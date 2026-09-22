# I43 acceptance — replaced stdlib and sample inventory

Closed 2026-09-21. Every tracked Can example is now labelled: eight
maintained manifest-backed projects (four `std/*/current` programs
plus the four admitted applications) that assert, build fresh from
staging, and run; fourteen std package READMEs carrying current
dispositions with predecessor text preserved as history; twenty
sketch programs labelled retired with a historical gallery note;
and the `catalogue/` mirror still generated, never handwritten. No
old kernel, extern, proof brand, or fuel algorithm was ported, and
no new native operation was invented.

Baseline `facee16` (I42) plus the I43 worktree. Apple M4 (Mac16,12)
darwin/arm64, macOS 27.0, Go 1.27.1, Bun 1.4.2 (`744846f84`, pinned
archive sha256 `90987a3a…6be1`, hash-verified), TypeScript 7.0.2 with
`@types/bun` 1.4.2 and `@types/node` 24.13.6, Playwright 1.55.1 with
Chromium 140.0.7339.186, disposable PostgreSQL 17.11 over loopback.

## Design consultations

[i43-jev](../i43-jev/decision.md): three fresh Jev consultations (all
prose rewritten, same facts/options). Unanimous at full confidence
for consolidate_domains (1.0 × 3): the five stale package READMEs
became disposition pointers and no new maintained project was
authored, since fixtures and applications already demonstrate every
domain. Unanimous at full confidence for catalogue_first (1.0, 0.98,
1.0): the inclusion gate walks catalogue operations to owning-task
evidence. Unanimous in direction but nearly tied in round 1 for
keep_historical_sketches (0.53/0.76/0.76 vs 0.45 retire in round 1);
investigated and overridden on the merits — git preserves the
history, while in-tree predecessor sources would cost permanent
exclusion rules and confusion, so all twenty sketch programs are
retired for I44 with history in `sketches/README.md` and
`CLEAN_ROOM_REVIEW.md`. Judgments are advice; the checks below are
the proof.

## Implementation

- `std/host`, `std/json`, `std/quota`, `std/schema`, `std/seq`
  READMEs rewritten as current dispositions (finite host
  catalogue, exact JSON codec, ordinary validation pattern,
  static asset approval, native arrays) with predecessor text
  preserved verbatim under `Historical implementation`, matching
  the nine already-current package READMEs.
- `std/README.md` rewritten as the inventory index: per-package
  dispositions, the eight-project maintained set, the
  coverage-row → catalogue-operation → source-assertion →
  integration-evidence mapping, the distinct P14 exclusion list,
  and the exact I44 deletion list (every `std/*/*.can/ts`
  outside `current/`/`catalogue/`, the twelve retired
  `errors.json` files, host externs/ambient types, all twenty
  sketches; `std/html/HISTORY.md` stays as history).
- `sketches/README.md` rewritten as a historical note: all twenty
  programs labelled retired, predecessor gallery text preserved
  below the line and marked no longer true.
- `std/text/README.md`: corrected the stale 31 assertion count to
  the measured 36.
- `compiler/internal/catalogue/integration_test.go`:
  `TestCatalogueInclusionInventory` requires every operation to
  name an owning task with a closed assertion kind
  (real/supplied/scoped), every package to carry an operation,
  type, or error unless reserved (`cli`/`json` are intentional
  bare namespaces — CLI surface in `io`/`env`, JSON in
  `codec`/`bytes`), and every owning task to hold
  implementation evidence outside `*-jev` directories.
- `tools/modcheck`: scope extended from the 42 fixtures to all 62
  maintained sources via manifest discovery, so retired shapes
  (`mod`, `extern`, pins, dec literals, braces) and unresolvable
  `uses` fail in maintained examples while manifest-less retired
  sources stay out of scope for I44.
- `tests/integration/stdlib_test.go`: `TestStdlibMaintained`
  discovers every maintained project by manifest walk (failing if
  any of the eight expected projects is missing), asserts real
  ordinary Can computation, builds twice with identical IDs,
  proves all emit stays inside owned `dist/`, and runs the four
  pure `std/current` projects to a clean exit.

## Verification

- `go test -count=1 ./...`: all 16 packages pass, including
  `tests/integration` (263s) with `CAN_BUN` 1.4.2,
  `CAN_BUN_ARCHIVE` (hash-verified), `CAN_TSC` (TS 7.0.2), and
  `CAN_TEST_POSTGRES_URL` on the disposable container.
- `TestCatalogueInclusionInventory`: 144 operations across 21
  packages resolve to 16 evidenced tasks. Scratch negative: a
  temporary `text::fake_op` owned by `I99` fails with `task I99
  holds no implementation evidence`; restored after.
- `modcheck`: 62 maintained sources pass; retired-shape unit
  checks still fail on every predecessor shape.
- `TestStdlibMaintained`: all 8 projects assert (23/16/14/20 for
  the applications, 19/23/40/36 for map/ratio/scalars/text, every
  one real-can), rebuild byte-identically, emit only into owned
  dist, and the std projects run clean. Scratch negative: hiding
  `std/text/current/can.project.json` fails with `maintained
  project std/text/current not discovered`; restored after.
- `bun test runtime/test/`: 850 pass, 0 fail, 63,285 expectations
  (142 files) — unchanged, proving no runtime regression.
- `cataloguegen --check`, `gramcheck`: pass; `std/catalogue/`
  untouched and fresh.
- `gofmt`/`go vet` clean on every touched file (4 remaining
  `gofmt` findings are pre-existing in untouched files).
- No committed `.ts`/`.js` beside any maintained `.can` source;
  the `current/` projects use only pure catalogue packages, and
  asset/SQL descriptor tracing runs through the same manifest
  loader in the admitted applications (I42 evidence).

## Limitations

- Retired sources still occupy the tree until I44 deletes them;
  gates exclude them by manifest absence, not by deletion.
- The inventory test proves evidence presence per owning task,
  not per-operation line coverage; corner coverage rests on the
  owning tasks' own suites linked from the mapping table.
- The `tscheck` project gate over `std/`/`sketches/` goldens still
  fails on predecessor rot; I44 owns that cleanup.

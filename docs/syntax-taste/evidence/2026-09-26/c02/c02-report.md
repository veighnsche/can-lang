# C02 evidence — required native browser controls

Task: `docs/syntax-taste/can-implementation-task-list-2026-09-26.md` §C02
(R02; F-R02-01/02/06; interface C-D).
Worktree base: `27338b09`; slices below. No task-list checkbox edits.

## Contract delivered

Extended `browser::event` snapshot (additive; `kind/target/value/key`
keep exact meaning) plus nine live calls:

| Addition | Projection / call | Justifying all-Can workflow |
| --- | --- | --- |
| `event.checked: bool` | checkbox/radio state at dispatch | controlled toggles; dirty checked reset |
| `event.selected: str[]` | selected option values, tree order | multiselect choose/read/reset |
| `event.files: browser::file[]` | `{name, size, mime}`, first 128 in selection order | upload selection read + pre-upload validation |
| `event.modifiers: browser::modifiers` | `{alt, ctrl, meta, shift}` | shift+click range select; keyboard shortcuts |
| `event.composing: bool` | native `isComposing` | defer normalization during IME |
| `event.selection: browser::selection` | `{start, end, direction}`, neutral `-1/-1/"none"` | normalization caret preservation; focus/caret repair |
| `set_value` / `read_value` | live string value IDL | normalization; dirty reset; autofill observation; save-time reads |
| `set_checked` / `read_checked` | live boolean checked IDL | controlled toggles; dirty reset |
| `set_selected` / `read_selected` | option selectedness by value | multiselect reset/programmatic set |
| `set_selection` | `setSelectionRange` + explicit direction | caret preservation/restore |
| `read_selection` | shared caret projection, total | caret re-reads (save-time, focus repair, async) |
| `read_files` | live file metadata, 128 cap | submit-time upload reads |

Rules: writes assign native IDLs and dispatch no events; reads deny
controls without the matching IDL (`rejected("property")`, text nodes
`rejected("text_node")`) except `read_selection`, whose neutral record
is unambiguous and therefore total like the snapshot. `set_selection`
rejects out-of-range bounds instead of clamping
(`0 <= start <= end <= value length`, `rejected("selection")`) and
admits directions case-insensitively (`rejected("direction")`
otherwise). Selection offsets are native UTF-16 code-unit offsets.
File bytes never cross; every collection escapes as a fresh frozen
copy. Snapshot evolution is additive-only per the C-D contract.

Design advice: three Jev consultations (`jev/`, model `jev-1.13.0`)
agreed on symmetric caret calls, file metadata records and nested
modifiers; composition signaling split three ways at low confidence
and was decided flag-only on minimal-sufficient grounds
(see `jev/findings.md`). Advice only, not qualification evidence.

## Slices

- `[C02]` design consultations (jev requests/responses/audit/findings).
- `[C02]` runtime: `browser.ts` adapter + `browser-controls.test.ts`
  (13 legs) + assert-surface extension.
- `[C02]` compiler: catalogue section + `cataloguegen` mirrors
  (types 102→105, operations 290→299), checker direction admission +
  C02 intrinsic task admission (one `program.go` condition), nine
  emit bindings + sealed nested-record identities, 43 positional
  `browser::event` constructions migrated with zero format drift.
- `[C02]` matrix: `browser-controls` fixture + `controls.mjs`
  (17 legs) + native page + `controls-native.mjs` (11 legs) + Go
  matrix across Chromium/WebKit/container-Firefox.
- `[C02]` this evidence report.

Catalogue note: the E-merge step (catalogue.json edit +
regeneration + mirror publication) was performed inline in this
worktree — no E worker is active, so serialization is moot. The E
owner should treat slice 3 as the merge candidate on rebase.

Handoff: the C-D operation/query contract (9 ops + 6 snapshot fields
+ 3 record types) is available for C06/W1 qualification; C04's
no-addition note stands (it completed before C02).

## Gates observed

- `bun run lint:fix:runtime`, `format:runtime` after runtime edits;
  `check:runtime` green (final re-run at close).
- `cataloguegen --check` clean; catalogue/check/emit/browser/driver/
  compiler Go suites green (check 696s single run; an earlier 600s
  default-timeout kill under concurrent load was re-run, not a code
  failure).
- `canlc format` drift-free on all touched `.can` files (one
  pre-existing 28-line drift in browser-conformance left untouched).
- `TestC02NativeMatrix`: 11/11 × chromium/webkit/firefox.
- `TestC02ControlsMatrix`: 17/17 × chromium/webkit/firefox; paired
  rebuild drift-free; database untouched; loopback-only ledgers.
- `TestGate5ServedMatrix`: 12/12 PASS (831s) — grid 32×3,
  conformance 17×3, empty 4×3, invoice 20×3 with the pinned
  L-redirect-webkit divergence only. Existing snapshots intact.

## Per-field per-browser evidence

Browsers: Chromium 140.0.7339.186, WebKit 26.0, Firefox 141.0
(container `can-ff`, pinned Playwright 1.55.1). N = native leg,
E = emitted-Can leg, U = runtime unit leg. All legs below passed on
all three browsers unless noted.

### Snapshot fields

| Field | Native | Emitted | Unit |
| --- | --- | --- | --- |
| kind/target/value/key (preserved) | checkbox-idl, dirty-divergence | text-echo, key-echo, check-echo (+ gate5 grid/conformance/invoice behavior) | snapshot-fields leg |
| checked | checkbox-idl (`checked` toggles, `value` stays `on`) | check-echo (`check=true\|value=on`), setchecked | checkbox snapshots |
| selected | multiselect (`a,c` then `b`), single-last-wins (`z`) | multi-echo, setselected (`setsel=b\|z`) | multiselect legs |
| files | files (2 files name/size/type) | files-echo (`files=2:a.csv:12:text/csv`) | metadata + 128-cap + malformed-skip |
| modifiers | modifiers (real shift+Enter, real s, synthetic ctrl+x) | key-echo (plain `a`, shift+Enter) | modifier record |
| composing | composing (real `false`, synthetic `true`) | text-echo (`false`), composing-synthetic (`true`) | composing flag |
| selection | selection (`1,3,backward`; checkbox `null` + `InvalidStateError`) | text-echo caret, setcaret (`1,3,forward`), reject neutral (`-1`) | caret + neutral + throw-path |

### Live calls

| Call | Emitted | Unit |
| --- | --- | --- |
| set_value / read_value | setvalue, reset, readall | dirty reset, IDL denials, mistype faults |
| set_checked / read_checked | setchecked, reset, readall | toggle, IDL denials |
| set_selected / read_selected | setselected, reset, readall | by-value set, unknown ignored, clear |
| set_selection / read_selection | setcaret, readall, reset, reject | bounds/direction/property denials, case-fold, throw mapping, totality |
| read_files | readall (count) | metadata + cap + property denial |

### Workflows

| Workflow | Native | Emitted | Unit |
| --- | --- | --- | --- |
| dirty value/checked reset | dirty-divergence, form-reset | dirty-reset (`reset=start\|false\|`) + DOM readback | set_value/set_checked reset |
| autofill observation | autofill-simulated | autofill-simulated (`text=ann@example.com\|…\|caret=15,15,…`) | external-change leg |
| no mutable escape | — (string verdicts) | — (string verdicts) | frozen/distinct projections |

## Engine divergences pinned (not hidden)

- Collapsed-caret direction: Firefox reports `forward` for implicit
  and explicit-`none` collapsed carets; Chromium and WebKit report
  `none` and honor an explicit `none` set. Pinned per engine in
  `text-echo` and `autofill-simulated`; the adapter projects each
  faithfully (no normalization).
- Checkbox `selectionStart` reads `null` in all three engines (the
  adapter's `try/catch` additionally guards throwing hosts; unit
  covers the throw path). `setSelectionRange` on a checkbox throws
  `InvalidStateError` in all three → `rejected("selection")`.

## Environment and commands

- macOS darwin-arm64; Bun 1.4.2; Go 1.27.1; Playwright 1.55.1
  (`tests/integration/browser`, `bun ci`); Firefox 141.0 via
  container `can-ff` (`CAN_FIREFOX_WS` from
  `distribution/provision-local.sh browser exports`); loopback
  forwarders 18651–18654 (firefox legs use 18651).
- `bun test runtime/test/browser-*.test.ts` → 45/45.
- `bun run check:runtime` → green.
- `go run ./compiler/internal/catalogue/cmd/cataloguegen --check` → clean.
- `go test ./compiler/internal/catalogue/ ./compiler/internal/emit/
  ./compiler/internal/browser/ ./compiler/internal/driver/ ./compiler/`
  → green; `go test -timeout 1800s ./compiler/internal/check/` → green.
- `go test ./tests/integration/ -run TestC02NativeMatrix` → 11/11 × 3.
- `CAN_BUN_ARCHIVE=/private/tmp/bun-darwin-aarch64.zip go test
  ./tests/integration/ -run TestC02ControlsMatrix` → 17/17 × 3.
- `CAN_BUN_ARCHIVE=/private/tmp/bun-darwin-aarch64.zip go test
  -timeout 3300s ./tests/integration/ -run TestGate5ServedMatrix`
  → 12/12 PASS (831s).
- No credentials in evidence; `TYPESAFE_API_KEY` env-provided for the
  three design consultations only (4812 in / 515 out tokens).

## Files changed (worktree slices 1–4)

- `runtime/platform/browser.ts`, `runtime/test/browser-controls.test.ts`,
  `runtime/test/browser-{dom,assert,cancel,query}.test.ts` (contracts).
- `compiler/internal/catalogue/{catalogue.json,generated.go,
  catalogue_test.go,browser_test.go}`, `runtime/catalogue.ts`,
  `std/catalogue/README.md` (generated mirrors).
- `compiler/internal/check/{browser.go,program.go,
  browser_controls_test.go}`, `compiler/internal/emit/
  {runtime_browser.go,browser_interface.go,browser_controls_test.go}`,
  positional-event migrations in check/emit/browser test fixtures.
- `examples/invoice-{grid,compare}/src/web/web.can`,
  `tests/integration/testdata/browser-conformance/src/main.can`
  (10-arg event samples).
- `tests/integration/testdata/browser-controls/`,
  `tests/integration/testdata/browser-controls-native/`,
  `tests/integration/browser/{controls,controls-native}.mjs`,
  `tests/integration/browser_controls_test.go`.


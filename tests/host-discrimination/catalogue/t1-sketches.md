# T1 generic catalogue sketches (D01, kept live)

For each X-R01-1 class, the catalogue-addition design that D02/H may
still select. Nothing here is implemented: E owns the
`catalogue.json` merge, C owns browser checker/runtime slices, and no
D01 leg writes either. Each sketch states the exact new checked
surface, the admission rules, the failure mapping (shared with the T2
prototypes), the merge/ownership cost, and the conditions under which
T1 beats the measured T2/T3 alternatives.

Conventions: identities use revision 1 (`can.std.<pkg>@1`); `emits`
bounds are finite and explicit per C-B; budgets mirror the prototype
validators byte-for-byte so a T1 implementation inherits tested
numbers.

## Op A — storage

New package `can.std.storage@1`:

- `local_get(key: str) -> str | none`
  emits `[storage::invalid_key, storage::unavailable]`
- `local_set(key: str, value: str) -> none`
  emits `[storage::invalid_key, storage::quota_exceeded, storage::unavailable]`
- `local_remove(key: str) -> none`
  emits `[storage::invalid_key, storage::unavailable]`

Types: none new (strings and none). Errors: `storage::invalid_key`
(key empty/overlong/malformed or value over budget),
`storage::quota_exceeded` (native quota), `storage::unavailable`
(blocked origin, security failure, unknown native).

Checker admission (C slice, alongside `check/browser.go`): keys and
values are runtime strings (no literal requirement — storage keys are
data, like state-cell identities); budgets enforced at runtime
(1..256-char keys, at most 1048576-char values). No new static
machinery: string-typed operands only.

Runtime (C slice, `runtime/platform/browser.ts` family): thin wrapper
over the WHATWG Storage calls with the prototype's failure map. No
handles, no callbacks, no disposal.

Merge/ownership cost: catalogue section append (E merge + regenerate);
checker + runtime slices (C); no LSP/formatter surface (no new
syntax). Stable native shape: the Storage interface is a decades-old
standard, so the checked surface will not churn.

T1 selection conditions for Op A: select T1 when more than one
shipped app needs origin-local persistence (generic, cross-app,
vendor-neutral, stable), and the per-app cost of a reviewed-adapter
dependency (versioning, audit per release) exceeds the one-time
catalogue addition. If only one app needs it, T2 wins on
catalogue-surface economy.

## Op B — clipboard

New package `can.std.clipboard@1`:

- `read_text() -> str`
  emits `[clipboard::denied, clipboard::empty, clipboard::unavailable]`
- `write_text(text: str) -> none`
  emits `[clipboard::invalid_text, clipboard::denied, clipboard::unavailable]`

Types: none new. Errors: `clipboard::invalid_text` (empty/overlong/
malformed write), `clipboard::denied` (permission denied, pre- or
mid-call), `clipboard::empty` (no text available), `clipboard::unavailable`
(insecure context, missing API, unknown native).

Checker admission (C slice): `write_text` takes a runtime string with
the prototype budget (non-empty, at most 1048576 chars); no literal
requirement. The permission/availability gates stay runtime (they
depend on origin, focus, and engagement — unknowable statically).

Runtime (C slice): async wrapper over `navigator.clipboard` with the
Permissions-API pre-gate and the prototype's failure map, including
the explicit no-ordering rule for concurrent calls. No handles, no
app callbacks, no disposal.

Merge/ownership cost: same shape as Op A (E append + C slices).
Native-shape risk is higher than storage: engagement gating and the
empty-read shape vary and must be qualified per browser in D02
conformance (the prototype records both as shared, non-discriminating
gaps).

T1 selection conditions for Op B: select T1 only after D02 live
qualification pins the permission/empty-read matrix on all three
browsers AND a second app demonstrates need. Until then T2 carries
the risk in a versioned package without growing the checked surface.
T3 is excluded by measurement (compare-target failure), not by this
sketch.

## Widget C — chart

T1 for the widget class is NOT a chart catalogue: charting is a
vendor-SDK domain (renderers, versions, theming, accessibility
trees), and a generic `can.std.chart` would either freeze one
vendor's semantics as the standard or grow a second chart grammar in
the checker. The live T1 alternative is narrower:

- Option T1a (generic primitive): no new catalogue at all — the
  widget builds from existing catalogue ops (the C04 library proof
  shows ordinary Can composes). Honest verdict: insufficient — a
  competitive chart needs canvas/SVG primitives the catalogue lacks,
  which returns to a C02-style native-addition proposal, not a D01
  decision.
- Option T1b (reviewed generic capability): admit a minimal
  vendor-neutral `can.std.chart@1` (`render/update/dispose` over the
  prototype's spec shape) with the renderer fixed by distribution
  review per release. Cost: new checked surface + C checker/runtime +
  E merge + per-release renderer re-review — the T2 audit burden
  without T2's versioning boundary.

T1 selection conditions for Widget C: select T1b only if D02/H
decide the invoice-analytics chart must work offline in-process AND
must avoid a per-app adapter dependency — i.e. charts become
platform surface. Otherwise the measured contest stands: T2
(in-process SDK under review) versus T3 (SDK behind the companion
boundary). T1a is excluded by the catalogue's missing canvas/SVG
primitives (a C-owned gap, not assumed); it reactivates only if C
ever adds those primitives.

## Cross-class T1 notes

- T1 never admits vendor-specific operations: any sketch that names
  a vendor SDK fails the W2 bar (P09-M1 permits generic reviewed-path
  additions, never vendor patches).
- T1 implementations inherit the prototype validators and failure
  leaves verbatim; D02 conformance re-runs the T2 legs against the
  T1 runtime slice where the surface matches (Op A fully, Op B after
  live qualification).
- Catalogue growth stays live by construction: this file plus the
  per-class conditions above are the kept-live artifact, and the Jev
  C3 near-tie (catalogue 0.33) is preserved as the standing
  counter-case rather than relitigated.

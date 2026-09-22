# a46 S2: Owner-authorized typed UTF-8 export (B2)

Second execution slice of the Bytes workstream
([bytes-plan.md](bytes-plan.md) B2,
as corrected by the B2 pre-implementation review).

## Scope (one capability)

- Grant syntax `exports_utf8 Brand via fn@rev`: new top-level
  declaration (`Utf8ExportDecl`), full-production parse (missing
  `via`/`@N`, surplus tokens, negative/overflowing revs are `CAN1000`,
  never partial grants). Grammar + gramcheck samples gain the keyword.
- `certifyExports`: whole-program pass after world build, before any
  linkage evaluation, in both `checkProgram` and `diagnose`. Validates
  same validated `Module.ID` (reject empty/duplicate identities and
  ambiguous targets — no first-wins), exact exporter shape
  (`CAN6011`), and annotates the exact certified call node
  (`Small.ExportBrand`). Uncertified export calls are `CAN6010`.
- `Bytes__Value` compiler-owned record with coherent lookup
  (`newTycker`, `recordShapes`, `recordDecl`, `externUnion`); named TS
  definition emitted exactly when referenced; source shadowing of
  `Bytes`/`Bytes__Value` and of the kernel name is `CAN6012`.
- Export kernel `bytes__utf8__export`: call-checking and `given`
  exemptions (deterministic), explicit `EmitsOf` entry with
  independent existence verification, evaluator dispatch (byte copy,
  NUL/BOM preserved), emitter lowering (`TextEncoder().encode` in an
  `Ok` record, `given`-free single-`Ok`-arm shape re-verified).
- Certificate = node annotation set only by the certifier after full
  validation; per-run freshness invalidates stale grants (no
  serialization, nothing cached across runs).

## Non-scope

No generic encoder (B3), no HTML consumer (B4), no wrappers (B5), no
decoders, no `Encoding__Text`, no catalog entries (no new language
error kind — `CAN6010`/`CAN6011` are diagnostics, not `emits`).

## Rollback

Revert the slice commit. Nothing outside `compiler/` + grammar files +
this doc + the `docs/README.md` row moves. `CAN6010`/`CAN6011` stay
registered (codes never change meaning).

## Acceptance (all in `compiler/bytes_b2_test.go`)

| # | Row | Gate |
| - | --- | ---- |
| E0 | Granted export clean, incl. byte-correctness rows (empty/ASCII/non-ASCII/supplementary/NUL/BOM) | clean |
| E1 | Client via `uses` + correct script clean, both module orders (CLI pipeline) | clean |
| E2 | Wrong scripted export bytes → `CAN3110`, both module orders | code |
| E3 | Grant removal → `CAN6010`, no stale certificate | code |
| E4 | Malformed grants → `CAN1000` | code |
| E5 | Cross-module / ambiguous / rev-mismatch / unknown-target grants → `CAN6010` | code |
| E6 | Shape violations (params, return, emits/effects/decreases, body, `given`, arms, RHS, arg spelling) → `CAN6011` | code |
| E7 | Export call without grant → `CAN6010` | code |
| E8 | Kernel-name and `Bytes__Value` shadowing → `CAN6012` | code |
| E9 | Two granted brands (both orders) + third ungranted rejected | clean/code |
| E10 | Sink controls: allowed route clean, denied route diagnostic | clean/code |
| E11 | Emit pins: `TextEncoder().encode` lowering, `Bytes__Value` def iff referenced | text pins |
| E12 | Same-basename two-directory attack → `CAN6010`, both orders | code |

Gates after the slice: `go test -count=1 ./...`, `go run ./tools/modcheck`,
`go run ./tools/gramcheck` — all green at implementation time, with the
grammar keyword sample added for `exports_utf8`. Node vectors executed
for the emitted `TextEncoder` lowering (empty/ASCII/é/😀/NUL/BOM/entity
byte-exact against the Go E0 outcomes) — all pass. Probe scripts live
in `/tmp/bytesvec2` (scratch, not committed); the committed evidence is
the static pins in `compiler/bytes_b2_test.go` E11.

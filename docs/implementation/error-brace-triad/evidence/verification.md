# V01–V03 verification record

Date: 2026-09-30. All Go commands `go test -p 1 -count=1`, serialized,
`CAN_BUN=$(which bun)` in env where noted. No timeouts; no `go test ./...`,
no benchmarks, no bundles.

## V01 — front-end and corpus gate: PASS

- Focused: `TestBracesLexAsBalancedOneLineDelimiters`,
  `TestErrorDeclarationBraces`, `TestFiniteErrorBoundBraces`,
  `TestBraceConstructorsRecordDelimiter`, `TestFormatTriviaBraceConstructors`
  — ok.
- Bounded: `./compiler/internal/syntax`, `./compiler/internal/resolve` — ok.
- 59-file `TestFormatTriviaTestdataRoundTrip`: 60/60 subtests pass/skip
  (1 intentional `lexer/core.can` negative skips), 0 fail.
- Remaining old-form hits: none unexplained (see `corpus.md`).

## V02 — semantics and product gate: PASS

- Focused: `TestConstructorDelimiterRequiresResolvedKind` — ok.
- Bounded with CAN_BUN: `check`, `emit`, `catalogue`, `compiler` — all ok.
- `make catalogue-check` — passes. `go run ./tools/gramcheck` — grammar OK.
- Incidents during the gate (both resolved, no run narrowed):
  - Foreign commit `bbd888b5` added `emits [` to
    `emit/runtime_image_test.go`; fixed in `8292f985` (2 bounds, record/op
    spellings kept). Earlier "transient" emit failures are now attributed:
    one was this file while dirty pre-commit.
  - `TestW4MeasuredLegs` flakes under load (timing-ratio assertion
    `aggregation growth ratio ... outside linear band [2,10]`, e.g. 1.68;
    caught with `-v` log). Pre-existing load-sensitive perf assertion,
    untouched by this task (zero production changes); full suite re-ran
    green. Owner lane should harden or quarantine it.

## V03 — integration and runtime gate: PASS with environment skips

- `tests/failure-conventions`: ok, 11/11 skip — no `CAN_BUN_ARCHIVE` or
  `CONV_BUNDLE` configured (actual environment skip, recorded here).
- `tests/integration`: 7 pass, 106 skip (no archive), 2 fail —
  `TestC02NativeMatrix`, `TestBrowserWireCodecParity`, both
  "required browser chromium did not launch" in untouched setup code
  (environment-only; no chromium in this env).
- `host/conformance`: ok (7.4s, real tests).
- Runtime: no authored `.ts` changed (only `runtime/modules.json` inventory
  sync), so no lint:fix/format run; `bun run check:runtime` passes
  (format clean on 295 files, tsc clean); `bun run lint:runtime --format=agent`
  clean; `bun test tools/runtime/module-inventory.test.ts` passes.
- No performance measurements, no distribution bundle (deferred).

## V04 — left for Codex

Independent diff review + owned-scratch cleanup verification.

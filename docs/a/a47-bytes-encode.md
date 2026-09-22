# a47 S3: Generic UTF-8 encode kernel (B3)

Third execution slice of the Bytes workstream
([bytes-plan.md](bytes-plan.md) B3).

## Scope (one operation)

- Public kernel `bytes__utf8__encode(value: str) -> Bytes__Value`:
  total, deterministic, `given`-free, explicit empty `EmitsOf`.
  Parameter name `value` is pinned API (positional and named call
  spellings both admitted via the normal binding rule).
- Kernel descriptor table (`bytesKernels`): signatures, result
  records, emits sets, and the restricted flag. B2's export entry
  moves into the table; shared dispatch (call rules, `given` rules,
  `EmitsOf` registration, exhaustiveness existence, evaluator,
  emitter) keys off it. Per-kernel differences stay explicit:
  certificate annotation (export only), static signature (public
  kernels only — the export input stays call-site-specific).
- Strict `str` admission through the existing checker: `callee()`
  serves the encode signature, so brand/int/wrong-type arguments
  fail `CAN6003` and arity failures fail binding. The nominal
  boundary is static by design (brands erase at runtime): a
  statically rejected brand call may still pass at runtime, and the
  committed row documents that the static rule is the whole boundary.
- Byte-correctness: same transformation as the exporter (total over
  CAN scalars; NUL and BOM preserved; no normalization).

## Non-scope

No decoder (B6 owns the fallible side and the full Secret
composition: B3 pins the encoder edge only), no wrappers (B5), no
Render (B4), no new error kinds, no grammar changes.

## Rollback

Revert the slice commit. Nothing outside `compiler/` + this doc +
the `docs/README.md` row moves.

## Acceptance (all in `compiler/bytes_b3_test.go`)

| # | Row | Gate |
| - | --- | ---- |
| N0 | Encode vectors incl. NUL/BOM (positional + named spellings) | clean |
| N1 | Brand argument → `CAN6003` (static boundary documented) | code |
| N2 | Non-`str` argument → `CAN6003` | code |
| N3 | Arity failures → binding errors | code |
| N4 | `given` on encode → deterministic-call rejection | code |
| N5 | `EmitsOf` entries exist explicitly for both kernels | unit |
| N6 | Emit pin: `TextEncoder().encode` lowering | text pin |
| N7 | Extra error arm on encode match → stale-arm rejection | code |

Gates after the slice: `go test -count=1 ./...`, `go run ./tools/modcheck`,
`go run ./tools/gramcheck` — all green at implementation time. The
lowering is the same code path B2's node vectors already executed
(shared `stmtBytesEncode`), so no new target vectors ship here; the
committed evidence is the static pin in `compiler/bytes_b3_test.go` N6.

# a71: explain + machine payloads (a61 item 1, first cut)

Structured diagnostics + `canlc explain`, no language
change. Payloads ride `Diag` into the existing JSON
lines (`omitempty`: payload-free lines byte-identical).
Docs versioned with the compiler.

1. `Diag` gains `Expected`, `Found`, `Hint` (all
   `omitempty` in `jsonDiag`). Set post-hoc at four
   emitters only:
   - CAN4001 (foreign raise): Found = raised kind,
     Expected = declared emits list, Hint = add-kind
     vs raise-declared choice.
   - CAN3204 (bare kind): Found = bare kind,
     Expected = complete construction with field
     names + types, Hint = write-the-value.
   - CAN3105 (dangling test): Found = test name,
     Expected = script-row shape, Hint = where the
     row goes.
   - CAN4101 (missing arm, in `proofDiag`): Expected
     = missing kind, Hint = add-arm shape.
2. `canlc explain CODE` (new `run()` branch):
   per-code {rule, minimal violation, legal fix} for
   the four payload codes; family-level text for all
   other registered codes (cut recorded: remaining
   61 per-code docs are a follow-up, not silence —
   the fallback says so). Unknown code errors
   non-zero. Lookup is a pure function for tests.
3. Golden: the existing `jsonGolden` CAN3105 line
   gains the three fields deliberately (the freeze
   working as intended).

Out of scope: remaining 61 per-code docs; span +
replacement-text suggested edits (fields are
strings); items 2–5 of a61.

## Rollback

`git checkout -- compiler/check.go compiler/types.go
compiler/lsp.go compiler/main.go
compiler/json_golden_test.go` plus delete
`compiler/explain.go compiler/explain_test.go`.

## Test plan

- Probe first (red): `compiler/explain_test.go` —
  known/unknown explain lookup, one payload
  assertion per code (exact Expected/Found/Hint),
  golden mismatch until the CAN3105 line is updated.
- `go test -count=1 ./...`, `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`.

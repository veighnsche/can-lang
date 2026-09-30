# F02 evidence — structured issues and complete source ranges

## Shared representation (`source/diagnostic.go`)

- `SeverityError/Warning/Note/Hint` constants; notes must never be reclassified as warnings.
- `UnitStatus`: `UnitValid/UnitInvalid/UnitBlocked` (blocked = named prerequisite unavailable, no cascades).
- `Issue{File,Span,Code,Severity,Message,Related,Fixes}` in original-byte coordinates; UTF-16 conversion
  happens exactly once at the editor boundary.

## Retained token spans (`syntax/ast.go`, `expressions.go`, `parser.go`)

- `UnaryExpr/BinaryExpr.OperatorSpan`, `ComparisonExpr.OperatorSpans` (including `is not` = `is`-start..`not`-end).
- `QualifiedName.QualifierSpan` (zero when unqualified) + `MemberSpan` (== Span when unqualified).
- All five non-test syntax constructors fill the new spans (grep-audited). Synthetic checker/resolve
  `QualifiedName{Name:...}` sites keep zero spans honestly and never enter diagnostics.
- `declarations_test.go` shape helper now ignores the new location fields (same as `Span`).

## Widening removed (`driver/diagnostics.go`)

- `loadDiagnostics` preserves multiline end lines and zero-width EOF insertions; line fallback remains only
  for genuinely unconvertible offsets.

## Tests (all passing 2026-09-30)

- `syntax/editor_spans_test.go`: operator/comparison/`is not`/qualifier/member/callee/argument/member spans.
- `source/eof_multiline_test.go`: EOF insertion (incl. empty file), mid-line insertion, multiline `😀\nsecond`
  end lines; existing `source_test.go` still covers BOM/CRLF/astral/combining.
- `driver/ranges_editor_test.go`: multiline + EOF-insertion bridge preservation.
- Suites: `source` ok, `syntax` ok, `driver` ok (43s), `go build ./...` + `go vet` clean.

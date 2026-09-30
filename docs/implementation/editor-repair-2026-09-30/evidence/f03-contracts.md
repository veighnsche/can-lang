# F03 frozen contracts — recoverable analysis + semantic index

Owner: coordinator. Dependent lanes (P10+, R20+, E30+, S40+) build on these exact types;
changes require coordinator approval until H75. Compiler mutable state stays private until Seal.

## Files

- `compiler/internal/driver/editor_analysis.go`: `DocInput`, `DiskInput`, `Fingerprint`
  (`NewFingerprint/Equal/Digest/Docs/Disk`), `UnitID`, `UnitReport{Status,BlockedBy}`,
  `Analysis{Root,Fingerprint,Sources,Graph,World,Issues,Units,Index}` (`Sealed/Errors/Unit`),
  `Builder` (`NewBuilder/AddSource/AddDisk/SetGraph/SetWorld/AddIssue/SetUnit/Index/Seal/Sealed`),
  `OrderIssues` (file, byte start/end, code, severity, message + dedup).
- `compiler/internal/driver/editor_index.go`: `OccurrenceKind` (16 kinds incl. unresolved),
  `Visibility` (local/package/exported/import/generated), `Occurrence{ID,Kind,Visibility,File,
  DeclSpan,NameSpan,Scope,Type,Signature,Arguments,Declaration,Valid}`,
  `OccurrenceIndex{Occurrences,Binding,At}` (valid-only, never guesses),
  `IndexBuilder{Add,Seal}` (rejects empty ID / empty name spans).
- Coordinates: all spans are original-byte `source.Span`; `source.Issue` (F02) carries
  severity `error/warning/note/hint`, related spans, fixes. UTF-16 conversion once at editor boundary.

## Contract guarantees (tested)

- Fingerprint: exact text + full disk vector; text edit, disk mutation, sibling add all invalidate;
  input order canonicalized; digest matches equality.
- Issues: deterministic order + exact-dup removal; input slice never reordered.
- Seal: deep-copies sources/units/issues/index; post-Seal builder mutation cannot affect sealed values;
  unknown units read `UnitBlocked` with `analysis:unknown-unit`.
- Index: invalid/unknown bindings and blocked/wrong-file offsets never resolve.

## Tests

`driver/editor_analysis_test.go`: `TestFingerprintRequiresCompleteInputs`,
`TestOrderIssuesDeterministicAndDeduped`, `TestBuilderSealIsImmutableAndBlockedExplicit`,
`TestOccurrenceIndexQueriesNeverGuess` — all pass.

# a79: LSP revision enforcement (`canlc lsp --baseline`)

Status: shipped. The a77 deferred item "LSP `--baseline` wiring"
is closed: the editor surfaces the same CAN6013 findings as the
CLI, per keystroke, with no change to unenforced behavior.

## Behavior

- `canlc lsp --baseline BASE.json` loads the baseline once at
  startup and runs `CheckRevisionIdentity` on every diagnosis
  (open + change), after all existing checks.
- Enforcement mirrors the CLI's `firstError` gate: identity
  findings append only when the world otherwise checks clean,
  so broken programs never gain drift noise atop real errors.
- Bare `canlc lsp` is byte-identical to before: `diagnose`
  delegates to `diagnoseWith(..., nil)`, and all existing
  call sites are untouched.
- Unknown flags are a usage error (exit 2). A missing or
  unreadable baseline warns on stderr and runs unenforced: a
  bad flag must not brick editing, and the CLI stays the
  authority (it fails hard on the same input).
- A generated-but-unaccepted baseline squiggles
  (`not an accepted baseline`), per generation-is-not-acceptance.
  A mid-session baseline change takes effect on restart.

## Deliberate limitations

- Findings anchor to the open document (`withFile`, same lossy
  stamping the CLI applies with `mods[0].File`): multi-file
  drift shows on the open file. Per-file publishing is a
  separate slice.
- No hot-reload of the baseline file; restart picks it up.

## Verification

- Probes first: `compiler/lsp_baseline_test.go` pins clean
  quiet, sibling drift → CAN6013 error, nil quiet,
  unaccepted squiggle, and flag parsing.
- Gates: `go test -count=1 ./...`, `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`, `tsc -p tscheck/tsconfig.json`.

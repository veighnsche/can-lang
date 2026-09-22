# canlc — the can-lang launcher (Go)

The launcher is five files: `main.go` (command dispatch),
`lsp.go` (stdio Language Server over the current bridge),
`current_parse.go`, `current_project.go`, and `current_types.go`
(inspect commands). The parse/resolve/check/emit pipeline lives in
`internal/`; the predecessor toolchain was deleted in I44.

Commands: `assert`, `build`, `run`, `parse`, `inspect-project`,
`inspect-types`, `runtime-check`, `catalogue-check`, `version`,
`lsp`, `clean`. The removed `explain`, `lint`, `baseline`, and
`normalize` modes exit 2 naming their retirement. `build` and `run`
refuse without a qualified sidecar; see the
[CLI guide](../docs/implementation/cli.md) and
[assertions guide](../docs/implementation/assertions.md).

From the repo root:

```
go build -o /tmp/canlc ./compiler
/tmp/canlc parse compiler/testdata/current/project/src/main/main.can
go test ./compiler/
```

`go test ./...` runs the compiler, mirror, integration, and
retirement gates (`retirement_test.go` pins the five-file launcher
and forbids retired symbols). Editor mode: `canlc lsp` speaks
minimal LSP over stdio; the client lives in `editors/vscode/`.

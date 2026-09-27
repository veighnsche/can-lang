# canlc — the can-lang launcher (Go)

The launcher owns command dispatch, project inspection, and the stdio
Language Server. The parse/resolve/check/emit pipeline lives in
`internal/`.

Commands: `assert`, `build`, `run`, `parse`, `inspect-project`,
`inspect-types`, `runtime-check`, `catalogue-check`, `version`,
`lsp`, `clean`. Unknown commands exit 2. `build` and `run` refuse
without a qualified sidecar; see the
[CLI guide](../docs/implementation/cli.md) and
[assertions guide](../docs/implementation/assertions.md).

From the repo root:

```
go build -o /tmp/canlc ./compiler
/tmp/canlc parse compiler/testdata/current/project/src/main/main.can
go test ./compiler/
```

`go test ./...` runs the compiler, mirror, and integration tests.
Editor mode: `canlc lsp` speaks
minimal LSP over stdio; the client lives in `editors/vscode/`.

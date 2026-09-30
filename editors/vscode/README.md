# Can language support for VS Code and Cursor

The extension colors current Can syntax with a TextMate grammar and starts the
`canlc lsp` server for diagnostics and editor features. It uses the bundled
host binary by default. Set `canlc.serverPath` to an absolute executable path
only when developing against a separately built server.

The status bar shows the server build reported during initialization. Select it
for the exact binary path and any project configuration errors. `Can: Restart
Language Server` restarts after a configuration change or on demand. Can files,
untitled Can buffers, the named project/lock/registry JSON files, and custom
registries declared by `can.project.json` are selected. File changes throughout
workspace folders reach the server for source, dependency, registry, and asset
invalidation. `canlc.inlayHints` controls proven type and parameter hints.

For a reviewed local host release, from `editors/vscode`:

```sh
bun ci
bun test
bun run build:server
bun run verify:server
bun run smoke:protocol
bun run package:host
```

`build:server` stamps the binary with the extension version, Git revision, and
a digest of compiler and extension inputs, then signs it locally on macOS.
`smoke:protocol` checks the bundled server initialize handshake; actual editor
activation is verified after installation in Cursor. `package:host` rejects a missing, changed, or stale binary before VSIX creation
and checks the resulting archive's version and binary hash. The build is for
the current host only. The output is `bin/can-lang.vsix`; remove the task-owned
archive after installation/verification. Running `package:host` after any source
change requires a new `build:server`.

The grammar and client tests use pinned `vscode-textmate` and
`vscode-oniguruma`, including compiler fixtures, unfinished input, nested
comments, and lifecycle failures. `go run ./tools/gramcheck` from the repo root
checks asset structure and fixture provenance; `bun run test:grammar` checks the
actual ordered TextMate scopes.

Installation into a running editor belongs after the reviewed build and
archive checks. Use the editor's extension install command on the verified VSIX
and reload the window without discarding unsaved work. The status bar build
must match the version printed by `verify:server`.

# can-lang editor support (Cursor / Antigravity / VSCode)

Two halves: `syntaxes/` colors (TextMate), `client/` + `bin/canlc` squiggles
(LSP client talking to `canlc lsp` over stdio).

Install (from the repo root; manual copy into the extensions dir does
NOT register — the editor only loads manifest-registered extensions):

```
go build -ldflags "-X main.version=$(git describe --tags --always --dirty)" -o editors/vscode/bin/canlc ./compiler
(cd editors/vscode && bun ci)
(cd editors/vscode && bunx -y @vscode/vsce package -o /tmp/can-lang.vsix)
cursor --install-extension /tmp/can-lang.vsix --force   # or: code --install-extension ...
antigravity-ide --install-extension /tmp/can-lang.vsix --force
codesign --force --sign - ~/.cursor/extensions/can-lang.can-lang-*/bin/canlc
codesign --force --sign - ~/.antigravity-ide/extensions/can-lang.can-lang-*/bin/canlc
```

(`codesign` re-signs the server so macOS runs it; without this the OS
kills it with "Code Signature Invalid".) Then Developer: Reload Window.
Open any `compiler/testdata/current/*/*.can` file. Files with errors
show squiggles; clean files show none. The server also answers
go-to-definition for nominal types, calls, module values, and
AI-generated record fields.

See a squiggle and disagree? The diagnosis comes from the inert bridge
(`compiler/internal/driver/diagnostics.go`, proven by
`compiler/internal/driver/diagnostics_test.go` and the protocol
exchanges in `compiler/lsp_server_test.go`) — fix it there, rebuild
`bin/canlc` with the stamped command above, reinstall.
The server parses, resolves, and checks an in-memory overlay snapshot on
every keystroke; it never builds, runs, asserts, dials out, or writes.
Unsaved sibling buffers diagnose as one coherent snapshot, and the
`--baseline` execution hook is retired: `canlc lsp --baseline` exits 2.

Known limitation: strictness (lint) squiggles are not published yet;
the editor shows parse, resolve, and check diagnostics only.

Settings: `canlc.serverPath` overrides the server binary (default: bundled
`bin/canlc`, else `canlc` on PATH). Token colors (e.g. forcing `error` red)
live in the user's `settings.json` via `editor.tokenColorCustomizations`:

```json
"editor.tokenColorCustomizations": {
  "textMateRules": [
    {
      "scope": "keyword.declaration.error.can",
      "settings": { "foreground": "#F14C4C", "fontStyle": "bold" }
    }
  ]
}
```

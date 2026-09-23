# can-lang

Can is a small contract-first language (`.can`) that compiles to
TypeScript. You write records, functions with mandatory assertions,
and native declarations for AI, HTTP, SQL, and HTML; the compiler
checks them and emits typed TS plus machine-readable reports. Every
function carries its assertion rows. `assert` executes those rows in
emitted code and never publishes; `build` executes the same rows as
gate verification and publishes one complete verified generation only
when every root passes (the [verified-build design](docs/implementation/language-behavior-contracts-2026-09-22.md#b5-verified-build-and-publication),
P15.1).

## Layout

- [`compiler/`](compiler/README.md) — `canlc`, the Go launcher plus
  the current parse/resolve/check/emit pipeline
- [`runtime/`](runtime/) — the private TypeScript runtime (partitioned
  by contract, never hand-edited per program)
- [`examples/`](examples/) — four admitted end-to-end applications
  (native-ai, account-search, form-validation, dashboard)
- [`std/`](std/README.md) — package dispositions and the four
  maintained `current/` example projects
- [`docs/archive/`](docs/archive/README.md) — retired predecessor gallery,
  historical design records, and the I36 binding comparison (history only)
- [`distribution/`](distribution/README.md) — pinned target,
  qualification, bundles, offline install/update, release notes
- [`tests/`](tests/integration/) — staged integration suites
  (incl. SQL, browser, applications, stdlib)
- [`docs/`](docs/README.md) — design records and the implementation
  ledger ([tasks](docs/implementation/tasks.md),
  [coverage](docs/implementation/coverage.md),
  [evidence](docs/implementation/evidence/2026-09-21/))
- [`editors/vscode/`](editors/vscode/README.md) — syntax highlighting
  plus an LSP client over `canlc lsp`
- [`tools/`](tools/) — distbuild, gramcheck, modcheck
- [`tscheck/`](tscheck/README.md) — strict TypeScript over fresh emit

## Install (end users)

End users install a qualified release into a user-owned root.
Releases are unsigned until signing credentials are authorized (see
[release notes](distribution/README.md#release-notes)):

```sh
# From the source tree (a standalone installer binary is future work).
go run ./tools/distbuild install --archive can-<version>-<target>.zip \
  --sha can-<version>-<target>.zip.sha256 --root ~/.can-root
~/.can-root/current/bin/canlc version
```

Running an installed release needs no Bun, Node, npm, C compiler,
or database: the bundle carries the exact pinned Bun sidecar and
every runtime asset. PostgreSQL is required only to run the
SQL-backed examples against a live database; everything else runs
offline. Only the installer itself runs from the Go source tree
today.

## Build from source (developers)

```sh
# Prereqs: Go 1.25+, plus the pinned Bun archive acquired once.
make build        # builds ./bin/canlc
make bundle BUN_ARCHIVE=/absolute/path/bun-darwin-aarch64.zip VERSION=dev-1
```

Source builds additionally need Node 24 + npm for the TypeScript
and browser legs, and PostgreSQL 17 for the live SQL legs; see the
[verifier workflow](.github/workflows/verifier.yml) for the exact
operated services. CGo (`pg_query_go`) needs a C compiler at Go
build time only.

## Quickstart

```sh
bin/canlc parse compiler/testdata/current/project/src/main/main.can
bin/canlc inspect-project compiler/testdata/current/project
CAN_BUN_ARCHIVE=/absolute/path/bun.zip go test ./tests/integration/ -run TestStdlibMaintained -count=1
```

From the repo root, `go test ./...` runs the compiler, mirror,
integration, and retirement gates; `bun test runtime/test/` runs
the 850-test runtime suite; `tscheck/` typechecks fresh emit.
CI runs all of it with zero skips on `macos-15`.

## Status

Implementation is complete: all fifty ledger tasks are checked
with evidence, the predecessor toolchain is deleted, and the
release candidate passes the mandatory macOS gates. Open gates
are publisher signature, notarization, and release upload, which
require authorized credentials. No Linux support is claimed, no
provider quality is claimed, and no proof/termination/effect
inference is offered — behavior is checked assertions plus
explicit contracts.

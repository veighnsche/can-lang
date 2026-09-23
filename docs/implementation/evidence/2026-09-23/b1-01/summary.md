# B1-01 evidence: files, directories, paths and globbing

Date: 2026-09-23. Pinned target: bun-1.4.2-darwin-arm64-v1 (revision
744846f844374847c902b5e7fd59b4342a51ef99, binary SHA-256
35d20dd0263e5c950194434b925454fdfa9ba6e4467da960410fa05b08a7a5b5).

## Contract

Catalogue packages `files` and `path` (16 operations, 9 errors 1300-1308,
records `files::file_info` and `files::entry`). Full operation/error table
and semantic contracts live in the implementation notes
(docs/implementation/design-revisions.md, B1-01 section) and the generated
catalogue mirror (std/catalogue/README.md).

## Native qualification (pinned binary)

- `Bun.file(path).stream()` reads bytes; directory input fails EISDIR and
  missing input ENOENT.
- `Bun.write` auto-creates missing parent directories, so Can writes use
  `node:fs/promises.writeFile` (`wx` for exclusive creation, `w` for
  overwrite) and report missing parents as `files::not_found`.
- `writeFile` with `wx` and `copyFile` with `COPYFILE_EXCL` fail EEXIST.
- `copyFile` on a directory fails ENOTSUP on macOS (EISDIR on Linux); the
  adapter stats the source first and reports `files::unexpected_kind`.
- `rename` has no copy fallback; EXDEV maps to `files::cross_device`.
- `rm` without recursion refuses every directory; non-recursive removal uses
  `rmdir` then `unlink`, reporting `files::not_empty` for trees.
- `Bun.Glob.scan` with `onlyFiles:false` lists files, directories and links;
  `followSymlinks` controls traversal through symlinked directories only.
  `*` excludes dotfiles; `.*` matches them; dangling links list as links.

## Verification

- `runtime/test/files.test.ts`: 15 tests against real temporary trees
  (round-trips, bounds, races, symlinks, permissions, glob corpus,
  assertion boundaries, error classification). Run with the pinned binary.
- `tests/integration/files_test.go`: staged-bundle execution of the runtime
  suite plus positive and rejected Can programs with diagnostic spans and
  generated-TypeScript native-lowering evidence.
- `examples/files`: maintained manifest-backed example covered by
  `TestStdlibMaintained`.
- `go test ./...`, `make catalogue-check`, `go run ./tools/modcheck`,
  `tests/conformance/native.test.ts` (new required APIs), and the ASAP
  layout size guard all pass; see the slice commit for identifiers.

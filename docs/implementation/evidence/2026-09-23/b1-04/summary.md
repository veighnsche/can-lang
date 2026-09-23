# B1-04 evidence: bounded child-process execution

Date: 2026-09-23. Pinned target: bun-1.4.2-darwin-arm64-v1 (revision
744846f844374847c902b5e7fd59b4342a51ef99).

## Contract

Catalogue package `process` (3 operations, 6 errors 1310-1315, records
`process::options` and `process::result`). Full operation/error table and
semantic contracts live in `contract.md` beside this file and the generated
catalogue mirror (std/catalogue/README.md).

## Native qualification (pinned binary)

- `Bun.spawn` with piped stdio and no shell: metacharacter arguments stay
  literal; missing executables throw ENOENT synchronously; denied working
  directories throw EACCES.
- `detached:true` makes the child a process-group leader (verified
  PGID == pid); `process.kill(-pgid, signal)` reaches grandchildren
  (verified: grandchild dead after group SIGKILL). Non-detached children
  share the parent group, so Can always spawns detached and terminates by
  group: SIGTERM, grace, then SIGKILL. Descendants that create their own
  session escape cleanup; that boundary is documented, not advertised away.
- `Bun.which` verifies existence (missing names and paths resolve to null).
- `Subprocess.exited` is stable across awaits; `signalCode` reports the
  terminating signal; `kill()` defaults to SIGTERM.
- Late stdin writes after early exit do not throw; unwritten input is
  silently dropped and the exit result is still reported.
- `env` replaces the environment wholesale; inheritance merges the live
  environment explicitly.

## Verification

- `runtime/test/process.test.ts`: owned-process cases (exit codes, signals,
  stdin, env/cwd, bounds, deadline, group cleanup, owner-drain kill).
- `tests/integration/process_test.go`: staged-bundle execution plus positive
  and rejected Can programs with diagnostic spans and lowering evidence.
- `examples/process`: maintained manifest-backed example covered by
  `TestStdlibMaintained`.
- `go test ./...`, `make catalogue-check`, `go run ./tools/modcheck`,
  `tests/conformance/native.test.ts`, and the ASAP layout size guard all
  pass; see the slice commit for identifiers.

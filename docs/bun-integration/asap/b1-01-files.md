# B1-01 — Files, directories, paths and globbing

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: none. Surface: Library.

Deliver application filesystem access, including paths and globbing, through typed library operations. Existing platform/io.ts is primarily standard-input handling; keep that purpose clear and add filesystem operations in their own module. Application paths are explicit authority, not compiler project confinement. An explicit base directory defines relative resolution; it is not a symlink-safe sandbox. Do not advertise race-free confinement without a native handle-relative proof.

## Where to implement

**Required layout:** follow [filetree and module boundaries](filetree.md). Its feature modules supersede the coarse existing-file locations below; do not append B1 implementation bodies to existing orchestration hubs.

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/platform/files.ts` | new | Bounded reads/writes, stat, directory operations and expected OS-error translation. |
| `runtime/platform/path.ts` | new | Native path operations, base-directory resolution and glob results. |
| `runtime/platform/io.ts` | existing | Reuse or extract compatible I/O error handling; do not broaden all catches indiscriminately. |
| `tests/integration/files_test.go` | new | Generated Can programs and temporary-tree cases. |
| `runtime/test/files.test.ts` | new | Limits, aliasing, failure and native filesystem behavior. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
files::read_bytes(path, max_bytes) -> bytes
files::read_text(path, max_bytes) -> str
files::write_bytes(path, bytes, overwrite) -> write_result
files::write_text(path, str, overwrite) -> write_result
files::stat(path, follow_symlinks) -> file_info
files::list(path, max_entries) -> [entry]
files::mkdir(path, recursive); copy(source, destination, overwrite)
files::move(source, destination, overwrite); remove(path, recursive)
path::resolve(base, parts); join(parts); basename(path); extension(path)
files::glob(base, pattern, follow_symlinks, max_entries) -> [path]
```

## Implementation sequence

1. Specify missing-vs-denied-vs-invalid-path failures, supported file kinds, overwrite/exclusive-create behavior, symlink following, recursive removal and ordering. Return absence only for actual missing paths, not permission failures.
2. Implement bounded chunk reads; check each chunk before accumulating. Decode text with fatal UTF-8. Copy native views before exposing immutable Can bytes. A stat size check is an optimization, not enforcement against a growing file.
3. Use Bun.file/Bun.write for applicable operations and node:fs/promises for directories, lstat, rename, copy and removal. Exclusive creation must use an atomic native flag, not exists-then-write. Do not call replacement writes atomic unless the implementation and tests support that promise.
4. Use node:path and Bun.Glob; normalize returned paths consistently, impose result limits during enumeration, and sort only if deterministic ordering is in the contract. Keep URL conversion separate from filesystem normalization.
5. Add catalogue signatures, errors, immutable metadata and emitter bindings. Fixture-capable effect calls must use the native-boundary provider before touching disk.
6. Add owned streaming readers/writers once B1-05 settles the lifecycle; bounded operations can ship earlier. Test short writes, failure cleanup and release of locks/handles.

## Acceptance evidence

- Success: binary including NUL and invalid UTF-8 through bytes; strict text round-trip; nested tree; empty directory; copy/move; reopen writes; glob dotfiles and symlinks.
- Rejected/failed: wrong types, unhandled declared errors, negative limits, text decoding failure, missing parent, existing exclusive destination, directory-as-file, permission denial and recursive-delete opt-in.
- Runtime: growing input cannot evade cap; cancelled reads close; failure cannot report complete write; symlink behavior agrees with declared policy; only a dedicated temporary tree is deleted.

## Fixtures and local assertions

Use a per-assertion temporary root for real filesystem tests. Fixture records bind operation, resolved logical path, mode, input digest and outcome; writes assert bytes and overwrite flags. Fixture reuse supplies data, never a global ambient mock. Add a strict no-live-filesystem assertion test. Paths in checked-in fixtures are symbolic test-root paths, not machine-specific absolute paths.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const input = Bun.file(resolvedPath).stream();
// Read through the shared byte-budget adapter; never await .bytes() first then check.
await fs.mkdir(resolvedDirectory, {recursive});
// Native operations implement the filesystem; adapters enforce Can values/failures.
```

## Gates and limitations

An application filesystem root is not a security sandbox. Do not apply compiler build-directory restrictions to arbitrary authorized application paths. Cross-device rename must either fail explicitly or use a separately documented copy/remove operation; silently losing atomicity is unacceptable.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/file-io)
- [Official Bun documentation](https://bun.sh/docs/runtime/glob)
- [Official Bun documentation](https://bun.sh/docs/runtime/nodejs-compat)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.

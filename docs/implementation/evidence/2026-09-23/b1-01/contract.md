# B1-01 operation/error table

As-built contract for the `files` and `path` catalogue packages. Native
cause (`code` plus the original error object) is retained privately on every
domain failure; public payloads carry only the fields below.

## Operations

| Operation | Native basis | Public errors |
|---|---|---|
| files::read_bytes(path, max_bytes) | Bun.file stream, counted before retaining | not_found, denied, invalid_path, unexpected_kind, limit_exceeded, io_error |
| files::read_text(path, max_bytes) | read_bytes + fatal UTF-8 decode | read_bytes errors + codec::invalid_data |
| files::write_bytes(path, value, overwrite) | writeFile `w`/`wx`, bytes copied out | not_found, already_exists, denied, invalid_path, io_error |
| files::write_text(path, value, overwrite) | TextEncoder + write_bytes contract | write_bytes errors |
| files::stat(path, follow_symlinks) | stat/lstat, kind + exact size | not_found, denied, invalid_path, io_error |
| files::exists(path) | lstat; false only when missing | denied, invalid_path, io_error |
| files::list(path, max_entries) | readdir typed entries, absolute sorted | not_found, denied, invalid_path, unexpected_kind, limit_exceeded, io_error |
| files::mkdir(path, recursive) | mkdir | not_found, already_exists, denied, invalid_path, io_error |
| files::copy(source, destination, overwrite) | stat pre-check + copyFile (+EXCL) | not_found, already_exists, denied, invalid_path, unexpected_kind, io_error |
| files::move(source, destination, overwrite) | rename, no copy fallback | copy errors + not_empty, cross_device |
| files::remove(path, recursive) | rmdir/unlink or recursive rm | not_found, denied, invalid_path, not_empty, io_error |
| files::glob(base, pattern, follow_symlinks, max_entries) | Bun.Glob scan, capped during iteration | not_found, denied, invalid_path, limit_exceeded, io_error |
| path::resolve/join/basename/extension | node:path, pure | none |

## Error mapping

| Public error | Fields | Native cause |
|---|---|---|
| files::not_found (1300) | path | ENOENT; copy/move attribute to the absent side best-effort |
| files::denied (1301) | path, operation | EACCES, EPERM, EROFS |
| files::already_exists (1302) | path | EEXIST (atomic `wx`/EXCL, mkdir, move pre-check) |
| files::invalid_path (1303) | path, reason | empty, nul_byte (static); not_directory (ENOTDIR), name_too_long |
| files::io_error (1304) | path, operation | EIO, EBUSY, ELOOP, ENOSPC, ENOTSUP and other listed I/O codes |
| files::limit_exceeded (1305) | limit | negative/huge-exceeded byte and entry caps, never silent truncation |
| files::not_empty (1306) | path | ENOTEMPTY |
| files::cross_device (1307) | source, destination | EXDEV; rename never falls back to copy |
| files::unexpected_kind (1308) | path, operation | EISDIR, list-on-file, copy-of-directory |

Non-listed codes, non-error throws, proxies and TypeErrors stay standard
failures with the adapter origin. Error outcomes never leak file contents;
payloads carry paths, operations, limits and reasons only.

## Deliberate deviations from the plan sketch

- Writes use `node:fs/promises.writeFile`, not `Bun.write`, because the
  pinned `Bun.write` silently creates missing parent directories while the
  contract requires `files::not_found` for missing parents.
- Writes return `void`, not a `write_result` record: `writeFile` is
  all-or-nothing, so a byte count would restate the input length.
- `files::glob` lists files, directories and links (`onlyFiles:false`):
  `follow_symlinks` controls traversal through symlinked directories and
  matching links always list as links, including dangling ones.
- `files::exists` reports `false` only for missing paths; permission and
  I/O failures are errors, never silent `false`.
- Streaming readers/writers (B1-01.06) are deferred to the B1-05 lifecycle;
  bounded operations ship first per the execution queue.

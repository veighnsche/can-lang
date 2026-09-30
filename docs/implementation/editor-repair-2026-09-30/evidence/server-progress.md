# Codex server lane progress

Ownership: root owns `compiler/lsp.go`, `lsp_scheduler.go`, `lsp_snapshot.go`, `lsp_scheduler_test.go`, and lifecycle/wire tests. Editor feature code was mechanically extracted to `lsp_features_base.go` and transferred to `/root/editor_implementation`. No concurrent writers to those files.

Implemented draft S40–S43 behavior:

- One serialized worker; intake owns live buffers, cancellation, queue limits and revision checks. Jobs use cloned overlays and documents. Rapid edits coalesce by root. Cancelled/stale responses receive explicit JSON-RPC errors. No compiler or semantic-index work occurs concurrently.
- Cached analyses include exact open-buffer text/version vectors, recursive manifest/dependency identities, source directory inventories, registry/lock/assets, compiler-discovered fixture inputs and canonical paths. Content hashes are streamed. Cache retains at most eight current roots, without histories; features share the cache lifetime.
- Input identities are checked again after analysis and before publication. Same-version/different-text input identity is distinct. URI paths decode once and canonicalize existing ancestors for new source files.
- Closing overlays rechecks the disk-backed root, including the last open buffer. Diagnostic publications have owning roots, preserving unrelated projects. Fileless project errors are sent as explicit project status; severities include error/warning/note/hint. Known multiline ranges retain both ends.
- Initialize reports `canlc` build version and UTF-16 coordinates. Full sync, save/watch/workspace/config handlers and no-disk-write untitled snapshots are connected. Inlay configuration passes through initialization and changes.
- Framing and request queues have explicit reported limits. Invalid framing cannot allocate arbitrary sizes; malformed JSON bodies do not stop valid following frames.

Verification at this milestone (not final independent acceptance):

```
go test -p 1 ./compiler -run '^Test(LSP|Server|Publish)' -count=1 -timeout=60s
ok github.com/veighnsche/can-lang/compiler 0.672s
```

The suite covers deterministic paused-worker input/cancellation, explicit request cancellation, latest-version publication, root isolation, config/dependency/asset/symlink-source invalidation, cache reuse, close-last-buffer reanalysis, exact UTF-16 wire ranges, build identity and malformed input. Old tests that required one diagnostic or a transient obsolete publication were updated to the new multiple-diagnostic/coalescing contract. The parser cascade exposed by an old test was repaired by the compiler lane, not hidden by changing a test.

Later changes after that gate: canonical scratch identity, unknown-method error precedence. Final combined tests, race check, independent review, installation/live verification and cleanup remain required. No performance measurement, application source edit, host binary build or installation has occurred in this lane.

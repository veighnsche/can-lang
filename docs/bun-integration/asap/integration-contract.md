# Shared implementation contract

This handoff prepares the thirteen ASAP capabilities. It does not amend the active LF01–LF21 implementation contract. Observe the finished language work before integration, preserve its native AI forms, grouped state, explicit contracts, attached assertions, normalized errors and native-request testing. There are no external users requiring legacy syntax or ABI aliases.

## Repository ownership

The [required filetree](filetree.md) refines this table and takes precedence for new code placement. Extract feature bindings from `emit/program.go`; keep it as orchestration, with per-feature bindings in `emit/runtime_<feature>.go`. Apply the layout size guard and responsibility review before every capability DONE step.

| Layer | Existing location | Required action |
|---|---|---|
| Public inventory | `compiler/internal/catalogue/catalogue.json` | Declare types, operations, fixed/callback error bounds, native lowering and assertion policy. Allocate identities against the current registry; never pre-reserve numeric IDs from this moving snapshot. |
| Generated inventory | `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md` | Regenerate with `make catalogue`; check using `make catalogue-check`. |
| Runtime binding | `compiler/internal/emit/program.go` | Add operation identity mappings, imports and adapter construction. Catalogue descriptors alone do not implement lowering. |
| Generic operation admission | `compiler/internal/check/arguments.go`, `callables.go`, `errors.go`, `codec.go` | Reuse type/error rules. Extend only for an actual new contract. |
| Resource evidence | `compiler/internal/ir/resources.go`, `runtime/owner.ts`, `runtime/callable.ts` | Preserve captured-resource ownership and callback scope. |
| Native forms | `compiler/internal/syntax/native.go`, `native_format.go`, `compiler/internal/check/native.go`, related IR/emitter files | Change only for the selected primitive. Format, check and lower its full behavior together. |
| Runtime adapters | `runtime/platform/`, `runtime/transport/`, `runtime/codec/` | Thin native adapters with Can values and completions. The capability plans assign individual files. |
| Runtime module graph | `runtime/modules.json` | Register exact import edges for every added/changed module. Reuse existing assembly and validation; no dynamic package installation. |
| Native qualification | `distribution/target.json`, `tests/conformance/native.ts`, `tests/conformance/native.test.ts` | Add supported required API probes; keep the runtime pinned unless a demonstrated gap needs a separately qualified upgrade. |
| Assertion boundary | `runtime/assert/provider.ts`, `context.ts`, `fixtures.ts`, `runner.ts`, compiler assertion/native-request checking | Extend completed LF behavior; no ambient mocks or provider access that bypasses local ownership. |
| Integration evidence | `tests/integration/`, `runtime/test/`, `compiler/internal/*/*_test.go` | Positive/negative programs, real native behavior, fixtures, diagnostics and generated TS. |
| Maintained applications | `examples/<capability>/` (new, name chosen after inventory) | Add valid manifest-backed Can examples to existing discovery rather than orphan snippets. |

Before editing, recheck these locations. [Source observation](evidence/source-observation.json) records per-file hashes while a different implementer was active; it is not an atomic clean-checkout snapshot.

## Source contract and authoritative documents

At implementation time reconcile each accepted API with `docs/syntax-taste/decisions.md`, `technical-spec.md`, `platform-testing-spec.md`, and, where relevant, `coordination-spec.md` and `ai-io-spec.md`. Update `docs/implementation/design-revisions.md` and the pertinent implementation notes when contracts change. This preparation deliberately leaves those shared documents untouched.

All proposed signatures in this directory are interface notation, not parser-approved Can syntax. The implementation agent must create real positive and negative `.can` programs before finalizing a surface. Library APIs use current declaration/call syntax. Primitive candidates must establish their grammar, handler contracts, test placement and semantics together. Do not copy illustrative pseudocode into the language as if already approved.

## Error and value rules

1. Enumerate expected environmental failures per operation: category, immutable public fields, retry implications and native evidence. Give them exact catalogue identities; do not invent a giant cross-platform error union to hide differences.
2. Preserve unknown adapter faults and violated internal invariants as standard failures. Avoid catch-all conversion to an ordinary I/O error or boolean false.
3. Preserve failure occurrence and private native cause through cleanup, wrappers and rethrows. Public diagnostics must not serialize passwords, keys, credentials, signed URLs or arbitrary native bodies.
4. Reuse the completed fetch/judge normalization only at its specified boundary. It is not permission to normalize every failure from files, SQL or process handlers into the fetch error family.
5. Copy native mutable byte views; construct immutable Can records/arrays. Opaque handles keep native objects private and retain the resource contents/captures needed by the owner.
6. Range-check exact Can integers before converting to native numbers. Time, buffer sizes, offsets and SQL values have distinct admissible ranges. Parsed safe integer values do not prove exact source-token semantics.

Each capability must add a small operation/error table to its implementation evidence before coding adapters. Fields include operation, expected cause/code, public error, public payload, retained private cause and non-domain exceptions. Use current ID allocation, not numbers guessed here.

## Ownership and publication

Use `registerResource`, `useResource`, `closeResource`, `withScope`, guarded callbacks and existing task supervision. Do not create another owner registry or background task supervisor. A scope's completion cannot silently abandon a stream, process, socket or transaction.

For long-lived operations define the exact start, one terminal publication, cancellation propagation, cleanup deadline and treatment of late callbacks. Exactly one terminal result is an adapter property; it does not imply exactly-once remote delivery. A deadline needs cancellation/reaping, not only Promise.race. Hard timing guarantees require native interruptibility or isolation; JavaScript timers cannot preempt synchronous native work.

Bound memory before allocation/accumulation where possible. Specify bytes, items, outstanding callbacks and shutdown duration separately. Native backpressure is preferred; event sources that cannot pause need a bounded queue and an explicit overflow behavior. Do not discard messages silently.

## Fixtures without weakening local tests

Pure parsing, hashing, formatting and mapping execute real native computation in attached assertions. Filesystem, process, network, randomness and time effects pass through locally supplied boundaries. Reusable fixtures are inert data/templates selected by each locally responsible assertion, not global exemptions from testing.

A request fixture records typed request identity and input, expected native response or event transcript, and consumed/remaining events. Handler code still runs. Add deliberate missing-fixture, extra-request, unused-event, mismatched-request and never-ending-transcript tests. Assertion timeout and standard-failure snapshots must retain the finished LF behavior.

Real native acceptance is separate: SQLite needs a real file, MySQL needs a real service, HTTP/WebSocket need loopback, and S3 needs a controlled compatible endpoint. Skipped service tests are visibly incomplete. Do not turn missing infrastructure into invented passing evidence.

## Per-capability completion evidence

Store evidence under a dated capability directory chosen by the implementer. Required artifacts:

- Before/after: one realistic program that lacked the capability and the complete new Can program, with its attached assertions. Avoid presenting a failing invented API as an old working program.
- Positive and rejected source programs, expected diagnostics and exact relevant source spans.
- Checked catalogue contract and operation/error table.
- Generated TypeScript excerpt showing the native API, binding and ownership; reject helper code that reimplements the runtime feature.
- Native tests, deterministic fixture tests, current target identity and relevant service/platform versions.
- Cancellation, overflow, cleanup, aliasing and provenance evidence for effectful capabilities.
- Updated maintained example and authoritative contract; status records every incomplete acceptance item explicitly.

## Commands and their limits

Run in the implementation checkout, after the active implementer's changes are integrated or safely isolated:

```sh
make catalogue
make catalogue-check
go test ./compiler/internal/catalogue ./compiler/internal/check ./compiler/internal/ir ./compiler/internal/emit
go run ./tools/modcheck
go test ./...
```

Choose focused runtime/integration tests while developing; use the repository's pinned `CAN_BUN` and `CAN_BUN_ARCHIVE` setup for runtime and bundled tests. For example, after adding the proposed runtime suite:

```sh
"$CAN_BUN" test runtime/test/files.test.ts
CAN_BUN_ARCHIVE="$CAN_BUN_ARCHIVE" go test ./tests/integration -run 'Files|SQLite|MySQL' -count=1
```

Those test names are intended matching groups, not claims the tests already exist. Verify that a test ran rather than matching zero tests or skipping because the archive/service is absent. Use the repository's existing bundle qualification procedure instead of downloading a random Bun binary. `tools/modcheck` checks maintained Can modules; it is not by itself a runtime import-graph or native compatibility test.

Preparation ran only the standalone evidence probe and documentation validation. It did not run these implementation acceptance commands against unfinished features.

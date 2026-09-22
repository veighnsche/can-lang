# B1-05 — Streams and reusable lifecycle contracts

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: B1-01, B1-04. Surface: Primitive candidate plus typed stream library.

Implement reusable lifecycle machinery and determine the Can surface from three complete programs: large-document ingestion, incremental process output, and a stateful WebSocket session. Native AI, grouped state and attached assertions remain the baseline. A match-like event form may fit Can well, but an ordinary one-shot match must not silently become a subscription.

## Where to implement

**Required layout:** follow [filetree and module boundaries](filetree.md). Its feature modules supersede the coarse existing-file locations below; do not append B1 implementation bodies to existing orchestration hubs.

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/transport/stream.ts` | new | Owned pull/consume/write/cancel adapters and byte budgets. |
| `runtime/owner.ts` | existing | Reuse resource registration, useResource, closeResource and guarded callback scopes. |
| `compiler/internal/ir/resources.go` | existing | Capture/ownership evidence for any new opaque stream types. |
| `compiler/internal/check/callables.go` | existing | Handler contract and error-bound checking. |
| `compiler/internal/syntax/native.go` | existing | Only if the comparison selects new native event syntax. |
| `compiler/internal/check/native.go` | existing | Only selected primitive admission, no unchecked callback escape. |
| `compiler/internal/emit/participant.go` | existing | Native participant lowering if required by the chosen form. |
| `runtime/test/streams.test.ts` | new | Backpressure, reentrancy, ownership and terminal races. |
| `tests/integration/streams_test.go` | new | Surface acceptance and negative compile programs. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
owned readable<chunk>, writable<chunk>  // conceptual types, not approved spelling
consume(source, handlers, limits) -> terminal_result
handlers = chunk, end, error; callback error bound is calculated
write(sink, chunk) -> accepted_or_backpressured
cancel(resource, reason); close(sink)
byte/text collection requires an explicit byte limit and deadline
optional primitive candidates must express repeated dispatch explicitly
```

## Implementation sequence

1. Write the three programs in library/callback, per-domain primitive and shared-event styles. Include grouped state, normal completion, handler failure, external failure, cancellation, early stop, backpressure and locally attached fixtures in each. Record diagnostic spans and generated-native sketches.
2. Choose the smallest surface justified by complete programs. Freeze its grammar and behavior in the authoritative specs at implementation time; do not wait to start independent library work. New parser/checker/IR files may be introduced only for an actually selected primitive.
3. Specify state transitions open -> closing -> closed, exactly one terminal publication, no new callbacks after closure, and observing late native rejections. Error events and handler-thrown failures remain distinguishable.
4. Prefer native pull/backpressure where available. Serialize handlers for one owned stream/session unless concurrency is explicitly requested. Await each handler before delivering its next item; bound any native event queue that cannot pause.
5. Register opaque resource contents with existing ownership/capture evidence. Reject use-after-close/foreign-owner consistently. Copy bytes at the boundary; no reusable native chunk alias crosses into immutable state.
6. Implement cancellation propagation to readers/writers/AbortController, release locks, bounded shutdown, and cleanup failure reporting without overwriting the primary completion.
7. Make fixtures drive the same dispatch/owner path as native events. Attach deadlines to assertions; a transcript that never ends cannot hang the build.
8. Wire file streams and process stdout/stderr first, then HTTP/S3/WebSocket. Do not invent a universal scheduler or resource system in parallel with owner.ts.

## Acceptance evidence

- Success: multiple chunks with exact order within one source; slow consumer propagates backpressure; grouped state updates once per item; UTF-8 spanning chunks.
- Rejected: mismatched handler type or error set, missing required terminal case under selected exhaustive syntax, escaping a disallowed borrowed resource, naked native JS stream data.
- Runtime: chunk handler fails, abort during pending read, simultaneous end/error, close twice, cancelled read rejects late, queue cap reached, alias mutation attempt.
- Generated TS uses ReadableStream/WritableStream and native hooks; no eager whole-stream buffering masquerading as streaming.

## Fixtures and local assertions

Transcript entries include item, normal end, native failure, cancellation, drain and logical timing. The test context owns each transcript and expected outgoing writes. Reusable transcript definitions remain inert until selected locally. Test extra/unconsumed events, missing terminal, handler failure and fixture exhaustion through real dispatch.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const reader = nativeReadable.getReader();
try {
  for (;;) {
    const next = await reader.read();
    if (next.done) break;
    await invokeCheckedHandler(copyIntoCanValue(next.value));
  }
} finally {
  // Cancel on early stop, observe rejection, then release the lock.
  reader.releaseLock();
}
```

## Gates and limitations

Grammar is a bounded early implementation gate, not an approval request or a reason to defer streams. Bun client/server event APIs differ; do not fabricate backpressure support or erase domain-specific events merely to fit one primitive.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/streams)
- [Official Bun documentation](https://bun.sh/docs/runtime/binary-data)
- [Official Bun documentation](https://bun.sh/docs/runtime/web-apis)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.

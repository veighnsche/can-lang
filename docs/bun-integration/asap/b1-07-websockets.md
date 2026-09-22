# B1-07 — WebSocket client and server integration

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: B1-05, B1-06. Surface: Strong primitive candidate plus connection operations.

Implement both client and server sessions, with a strong primitive candidate evaluated under B1-05. The user’s match-arm analogy is worth testing through a complete Can program with grouped session state. It does not settle the syntax, and WebSockets are only one of the thirteen capabilities.

## Where to implement

**Required layout:** follow [filetree and module boundaries](filetree.md). Its feature modules supersede the coarse existing-file locations below; do not append B1 implementation bodies to existing orchestration hubs.

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/platform/websocket.ts` | new | Client/server adapters, typed frames and session ownership. |
| `runtime/platform/server.ts` | existing | Upgrade integration and server-owned session cleanup. |
| `runtime/platform/http.ts` | existing | Typed upgrade result, request headers and protocol validation. |
| `compiler/internal/check/native.go` | existing | Selected primitive/session contract checks if needed. |
| `compiler/internal/ir/resources.go` | existing | Owned session capture evidence. |
| `runtime/test/websocket.test.ts` | new | Session ordering, queue caps and close races. |
| `tests/integration/websocket_test.go` | new | Real loopback client/server and surface tests. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
session events = open | text(str) | binary(bytes) | drain | close(code, reason) | failure
session operations = send_text, send_bytes, close
server = upgrade authenticated HTTP request with typed session data
client = connect URL with protocol/options and deadline
optional native event declaration = result of B1-05 comparison, not invented here
```

## Implementation sequence

1. Specify client/server event differences, frame types, connection handshake failure versus established-session failure, close code/reason validity and maximum message sizes.
2. Integrate Bun.serve upgrade and native WebSocket client. Authentication/authorization happens before server.upgrade; a rejected handshake remains ordinary HTTP.
3. Use per-session grouped state under serialized checked handlers. Native server hooks are installed once, with typed session data selecting the owner. Do not share mutable application state between sessions by accident.
4. Translate native send/backpressure results deliberately. Server drain can resume queued sends; the client API may need a different bounded strategy. Never claim the client exposes the same drain semantics as ServerWebSocket.
5. Handle cancellation, server stop, failed opening, terminal close and late callbacks through B1-05 ownership. Avoid automatic reconnect/replay; it changes delivery semantics.
6. Expose explicit protocol/subprotocol/TLS options supported by the target. Preserve close provenance; arbitrary event objects must not leak into Can.
7. If the primitive wins, add parser/format/check/IR/emitter support and attached event assertions together. Keep library connection operations available for straightforward operations.
8. Qualify server publication/subscription operations separately if included: distinguish socket publish excluding sender from server publish including all; do not conflate with a general message broker.

## Acceptance evidence

- Success: real client/server text and binary echo, two independent stateful sessions, explicit protocol, graceful close, upgrade denial.
- Rejected: wrong handler payload, invalid close code/reason length, unsupported protocol and invalid ws/wss address.
- Runtime: oversized message, saturated send queue, slow handler, callback failure, abrupt disconnect, close during pending send, server shutdown with active sessions.
- Acceptance includes selected primitive comparison evidence, not merely a working JavaScript echo server.

## Fixtures and local assertions

Script open/text/binary/drain/close/error events and verify sent frames through the actual adapter. Extra events after close must be rejected or ignored according to the terminal contract and must not invoke Can again. Real loopback tests qualify upgrade behavior; fixtures preserve local test ownership.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const client = new WebSocket(url, protocols);
// Server: Bun.serve({fetch(request, server) { ...server.upgrade(...) }, websocket: hooks})
// Hooks enter the Can owner and checked handler path; they do not call unchecked callbacks.
```

## Gates and limitations

WebSocket has no universal application delivery acknowledgment. Do not promise exactly-once network delivery from exactly-once terminal publication. Primitive comparison must also cover streams/processes, not extrapolate from this domain alone.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/http/websockets)
- [Official Bun documentation](https://bun.sh/docs/runtime/web-apis)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.

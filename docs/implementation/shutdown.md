# Bounded shutdown policy

T18 states the shutdown contract the runtime already enforces and pins it
with tests. There is no general `finally`, no cancellation of arbitrary
Can computation, and no consumed-fault channel: authors sequence explicit
closes, and each bound below covers only its own wait or operation.

## Bounds

- A caller close deadline bounds only the caller's wait
  (`closeResource` in `runtime/owner.ts`). Expiry reports the deadline
  failure and a cleanup diagnostic, but the native close keeps waiting
  for live leases and runs exactly once. Deadlines never force close,
  release leases, or cancel native work.
- A registered shutdown deadline (`shutdownMilliseconds`) marks a
  resource closing with a cleanup diagnostic when it expires, while
  owned participants keep their leases until they settle. Draining
  still waits for owned groups and callbacks.
- Server `stop` closes with the configured `shutdown_ms`: the caller
  observes a `shutdown_failed("deadline")` on expiry while handlers
  stay owned, and `wait` observes the eventual settlement.
- Stream `close_reader`/`close_writer`/`cancel_reader` wait under a
  fixed 5s caller bound (`STREAM_CLOSE_MS`); cancel is terminal and
  records its reason for the interrupted operation.

## Precedence and ownership

- The original outcome wins over close failure: a failing scope body
  keeps its exact completion while a failing automatic close sets
  `cleanupFailed` with a cleanup diagnostic. Cleanup never replaces an
  already selected domain outcome.
- Explicit close is exactly once per handle. A second close, like any
  use after close, is a standard `resource_state` rejection, never a
  domain failure. Only explicitly idempotent operations share repeats.
- Scope-owned handles die with their scope, including a handle
  returned as the scope result. Later use is `resource_state`.
- Races follow native selection: the winner returns, losers keep
  their leases until they settle, and the root waits for them. Losing
  domain failures stay silent; losing standard failures produce one
  sanitized late diagnostic each. An empty first-completion race stays
  pending; the harness owns its observation deadline.

## Operation-specific abort

- Abort exists only where an operation offers it: transport deadlines
  with native abort and owner-signal cancellation, stream cancel, and
  WebSocket cancel/stop prompts. Cancelling one operation never
  disturbs a sibling.
- Client disconnect does not abort an owned server handler: the
  handler runs to completion under its lease and `stop` drains it.
- A permanently pending owner can keep the process pending
  indefinitely; external supervisor termination makes no assertion
  that cleanup completed (see `runtime/test/owner-hung.test.ts`).

## Evidence

`runtime/test/shutdown.test.ts` covers precedence, exactly-once close,
scoped-result rejection, race winner with pending losers,
operation-specific cancel, and disconnect. Deadline, shutdown-timer,
and owned-work draining are covered in `runtime/test/owner.test.ts`,
`runtime/test/server.test.ts`, `runtime/test/transport-owned.test.ts`,
and `runtime/test/transport-late.test.ts`; empty-race behavior in
`runtime/test/coordination.test.ts`. The corrected cleanup funnel
lives in `examples/stream/src/main.can`.

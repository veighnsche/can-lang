# I33 acceptance — native server lifetime

Closed 2026-09-21. All 4 I33 catalogue operations are admitted, checked,
emitted, and verified end to end: `make_server_config` (real),
`server_start`, `server_wait`, `server_stop` (fixture-supplied).

## Design consultations

- `../i33-jev/decision.md`: config/body-limit/shutdown ranges, `bind_failed`
  address echo, `shutdown_failed` phases, signal ownership, lease span, and
  repeated-stop behavior. The 1-2 config split and the three-way numeric
  split were resolved by drift risk, sibling precedent, and timer overflow.

## What changed

Runtime: new `runtime/platform/server.ts` (`createServer`, `isServerValue`)
plus the `platform/server.ts` distribution inventory entry. Configuration
is a frozen private token validated without binding; the live server is an
owner resource in the caller's scope with `scopeManaged` close and the
configured `shutdown_ms` as both the registered drain bound and the stop
caller's wait bound. Fetch runs in a drainable child scope through
`guardCallback`, one `useResource` lease per request across snapshot,
dispatch, and conversion; rejections map to fixed 400/413 and escaped
failures to a fixed sanitized 500. `stop(false)` is the only native stop.
`server_wait` observes a server-owned settlement without initiating close
and owns per-wait SIGINT/SIGTERM listeners that re-enter through the server
scope, converging with explicit stop on one close.

Check: I33 admission in `check/program.go`; bare-`ok` fixture expectations
for opaque/callable results in `check/assertions.go` (previously rejected,
blocking every supplied boundary that returns a token); new
`check/server_test.go` for contracts, fixture attachment, and mismatches.
No `check/server.go` was needed: the four signatures need no static-input
specials beyond the gate.

Emit: `$canServer` factory, four function-table entries, `server_config`
and `server` kinds in the opaque predicate, and `$canServer` imports for
program and assertion modules.

Tests: `runtime/test/server.test.ts` (19 tests: config ranges, ephemeral
and conflicting binds, routed/keep-alive/413 traffic, sanitized 500s with
lease release, drain, caller deadline with continued ownership and waiter
observation, simultaneous and stale stops, coordination with a late waiter,
listener accounting, assertion boundaries, idle auto-close, child-process
SIGINT/SIGTERM, external-supervisor bound on nonsettling work);
`compiler/testdata/current/http/server.can` (7 assertions: real config
validation plus fixture-supplied lifecycle, no binds under assert);
`tests/integration/server_test.go` (staged assert/run/build, strict TSC,
compiled boot/traffic/stop/wait/serve over loopback).

## Specified behaviors pinned

- Port 0 binds ephemeral; two port-0 servers coexist. Double binds, bad
  ports, and bad hosts report `bind_failed` with the exact configured
  `host:port`, including huge-int ports rendered without float mangling.
- Caller-deadline expiry reports `shutdown_failed("deadline")` plus the
  specified cleanup failure while in-flight work continues; `server_wait`
  settles only on real settlement. Idle servers auto-close silently at
  drain; scope-managed close skips only the failure report.
- Repeated or stale stops fail standard `resource_state`; wait-after-close
  observes `ok`. A waiter retains no lease, so observation cannot deadlock
  the close it awaits. No `stop(true)` exists in runtime or tests.
- Unmapped `server_wait`/`server_stop` assertion use fails loudly as a
  missing fixture instead of binding; `make_server_config` runs for real.

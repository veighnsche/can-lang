# Request-token lifetime probe

24 September 2026. This experiment distinguishes two implementation
assumptions for U02: whether the current `abandonRequest` ends a Can request
capability, or whether a separate revocation step is required. Run from the
repository root with:

```sh
bun docs/syntax-taste/evidence/2026-09-24/selected-behavior/experiments/request-lifetime/probe.ts
```

The [script](probe.ts) snapshots one buffered and one lazy POST under Bun
1.4.2, calls `abandonRequest` on each, then checks `isRequest` and
`requestSnapshot` on the retained token. The exact [result](result.json)
shows both tokens still recognized and both snapshots still returning
`POST`. `abandonRequest` is a body cleanup operation, not a lifetime guard.
This confirms the selected packet must require a distinct revocation step
after the per-request owner scope drains, for buffered as well as live
requests. It does not prove the future revocation's ordering, WebSocket
behavior, or an HTTP response under load; those need implementation tests.

The request-response hedge/cancellation proposal P06 remains
[deferred](../../../../../post-upgrade-dispositions-2026-09-24.md#retain-or-defer-the-other-findings).
This probe does not claim that aborting a network request rolls back an
operation or releases losing owned work.

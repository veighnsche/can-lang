# Focused experiment: asynchronous browser ownership

The Gate 5 synchronous `AsyncLocalStorage` shim cannot support the selected
asynchronous browser owner contract. In Chromium 140.0.7339.186, two interleaved
branches both lose their owner context after suspension. Using the actual Can
owner runtime with that shim makes both resource operations return
`resource_state`; neither native resource operation runs. Bun 1.4.2 preserves both
contexts and both operations succeed.

This distinguishes retaining the fixture shim from implementing an owner context
that survives suspension. It does not establish a finished alternative runtime.

## Reproduction and reuse

Run from the repository root:

```sh
bun docs/syntax-taste/evidence/2026-09-24/selected-behavior/experiments/browser-owner/run.mjs
bun test ./runtime/test/owner.test.ts ./runtime/test/owner-roots.test.ts
```

The script requires the already installed Playwright/Chromium under
`tests/integration/browser/node_modules`; it downloads nothing. A local Chromium
launch needed sandbox escalation because macOS denied its Mach port registration.
The successful run used an ephemeral browser profile and injected a local bundle,
with no external page or server.

- [run.mjs](run.mjs) extracts the exact `asyncHooksShim` and `utilShim` text from
  [build-grid.mjs](../../../../../../../tests/integration/browser/build-grid.mjs),
  recording the async shim's SHA-256 in [results.json](results.json). It reuses
  the same SHA-256 crypto shim. A diagnostics-only `readFileSync` stub throws;
  this probe has no generated diagnostic index. Unknown engine imports fail the
  experiment build rather than receiving Bun browser stubs.
- [runtime-probe.ts](runtime-probe.ts) imports the actual
  [owner runtime](../../../../../../../runtime/owner.ts) and completion/failure
  machinery. It adapts the interleaved-root and external-callback scenarios from
  [owner.test.ts](../../../../../../../runtime/test/owner.test.ts).
- [context-probe.mjs](context-probe.mjs) makes the underlying context loss
  observable at two suspension points, using deterministic deferred gates.
  The tiny explicit-token control passes the token through ordinary function
  arguments and closures; it shares neither an ambient store nor a current-owner
  global.
- [owner-tests.txt](owner-tests.txt) records the existing owner/root regression
  tests against native Bun. No production source was changed.

## Observed results

| Observation | Native Bun owner context | Chromium with Gate 5 shim |
| --- | --- | --- |
| Synchronous entry in branches A/B | Correct A/B | Correct A/B |
| Context after gate and another microtask | Correct A/B | Missing in both |
| Actual owned resource use after await | Both `ok`; native operations run | Both `resource_state`; native operations do not run |
| Managed resources after roots drain | Both closed | Both closed |
| External guarded callback suspended with a registered resource | Root waits, one lease retained | Root waits, one lease retained |
| That callback's resource use after await | `ok` | `resource_state` |
| Calling the wrapper after scope closes | `resource_state` | `resource_state` |
| Explicit-token isolation control | Pass | Pass |

All eight machine assertions pass, including the expected negative observations.
No browser page errors were emitted. The guarded-callback experiment is useful
because retaining the callback task and its resource lease still works under the
shim: successfully draining that callback does **not** prove that the resumed
callback has its owner context. Its root completion remains `ok` even though its
separately observed callback completion is `resource_state`.

## Decision and limits

The supported browser build must replace the synchronous fixture shim. Binding a
callback only at entry is insufficient: the context must survive every emitted
`await`, calls made by resumed continuations, participant launch, and nested scope
transitions. The selected token-threaded profile remains unimplemented. This
experiment proves only that ordinary explicit arguments preserve identity across
these awaits; it does not prove compiler propagation, resource/capture leases,
exception paths, cancellation, nested coordination, or browser-wide lifecycle
correctness for that proposed profile.

The experiment runs two independent owner roots and an externally invoked
guarded callback; it is not a generated Can application, a DOM event integration
test, a multi-engine qualification, or a performance comparison. It does not
establish that every conceivable alternative to explicit token threading is
unsound. Implementation acceptance still needs generated Can fixtures covering
interleaving, nested calls/scopes and retained callbacks with the real browser
profile, plus the existing cross-root handle refusal obligations.

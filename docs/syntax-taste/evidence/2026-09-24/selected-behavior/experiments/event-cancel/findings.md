# Native event-cancellation timing

24 September 2026. The selected U05 behavior requires cancellation during
native dispatch, while the existing Can handler is asynchronous. This
experiment distinguishes an asynchronous handler decision from a
registration-time cancellation policy; it does not implement a Can catalogue
operation. Run from the repository root with:

```sh
node docs/syntax-taste/evidence/2026-09-24/selected-behavior/experiments/event-cancel/probe.mjs
```

The [Playwright script](probe.mjs) asserts each observation and ran in installed Chromium
140.0.7339.186. The initial sandboxed browser launch was denied by macOS;
the same script ran with approved unsandboxed execution. Its exact
[result](result.json) shows:

| Listener behavior | `defaultPrevented` before `dispatchEvent` returns | Later observation |
| --- | --- | --- |
| `preventDefault` after a Promise microtask | `false` | `true` after the microtask |
| Synchronous policy, matching cancelable Enter | `true` | Handler called once |
| Synchronous policy, unmatched Tab or noncancelable Enter | `false` | Handler called once per event |
| Same listener after `AbortController.abort()` | `false` | Handler count unchanged |

The delayed listener can change the Event object's flag later, but it cannot
meet the selected immediate-dispatch observation. A native listener policy
can prevent the matching default synchronously and still deliver one event
snapshot for later asynchronous work. The probe does not test Can's future
catalogue/checker, actual form navigation, browser-specific keyboard default
actions, or WebKit/Firefox. Those remain implementation acceptance tests.

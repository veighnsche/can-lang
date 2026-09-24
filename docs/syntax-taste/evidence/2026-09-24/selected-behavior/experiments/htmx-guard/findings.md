# Pinned HTMX guard experiment

24 September 2026. This is disposable browser evidence for the proposed S01 adapter, not production qualification. Run from the repository root with:

```sh
node docs/syntax-taste/evidence/2026-09-24/selected-behavior/experiments/htmx-guard/probe.mjs
```

The script uses Playwright 1.55.1 and its installed Chromium headless shell, intercepting loopback URLs rather than starting an application server. It loads this checkout's `distribution/assets/htmx-4.0.0.min.js` (SHA-256 `e484d9171a9db30a39c8f16e3d709d4137f3211c659f8e6125816635033d593f`). Its request, event, DOM and console observations are saved in [results.json](results.json). Browser launch required sandbox escalation on this host; the successful run exited 0.

| Case | Unguarded observation | Guarded observation |
| --- | --- | --- |
| `hx-target` absent before submission | HTMX warned but still sent `/reply`; its one `main` task targeted the source button. | `htmx:before:request` had `ctx.target` absent; canceling it prevented `/reply`. |
| 200 with `HX-Redirect: /hijack` | Browser navigated to `/hijack`. | Canceling `htmx:before:response` kept the original URL and DOM; `/hijack` was never requested. |
| 200 with `HX-Retarget: #other` | The one `main` task targeted `#other`, changing that unrelated node while the original target stayed old. | The same response event canceled processing; neither target changed. |
| Admitted 200 or 503 with `hx-swap-oob` | Both the main target and unrelated OOB node changed. | `htmx:before:swap` exposed `[main,oob]`; canceling it left both unchanged. |
| Admitted 200 or 503 with `<template hx type="partial" hx-target="#other">` | Both the main target and unrelated partial target changed. | The event exposed `[main,partial]`; canceling it left both unchanged. |
| Target removed while delayed request was in flight | Not separately probed unguarded. | The response event saw `ctx.target.isConnected=false`; the swap event still exposed a `main` task pointed at that detached node. Canceling the swap prevented other DOM changes. |

The probe's guard checks response-control headers at `before:response` and admits at `before:swap` only one `main` task for the original connected target. Those event positions occur early enough to stop every tested effect. Missing-target and rejected-protocol **reporting** was not implemented in this probe; the future adapter must produce the selected sanitized occurrences. Its target identity must come from the checked submission binding, not a fresh selector lookup: HTMX otherwise falls back to the source for an absent target, and a replacement element with the same id would not be the original target.

The experiment establishes the observed behavior for the pinned asset, these response shapes and this Chromium build only. It does not establish the compiler's emitted binding, real server response bytes, every HTMX response-control header, all possible task extensions, or cross-browser timing. The implementation's acceptance tests must cover those boundaries and the accepted 422/409/403 cases as well.

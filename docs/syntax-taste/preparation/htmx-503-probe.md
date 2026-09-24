# Current HTMX 503 visibility probe

24 September 2026. Bounded DI-09d/DI-13 observation on source commit `02a549d28fd5fc5c3996160e65a97de332390d30`. This tests the existing account-search action and pinned browser asset; it makes no syntax or product-policy choice.

## Setup and method

- macOS 27.0 arm64; Bun 1.4.2 from the checked-in pinned archive (archive SHA-256 `90987a3a16d7db556d886ac3edf0a1cf43acaed622e8676be1d12f`); Node 24.21.0; Playwright 1.55.1; installed Chromium 140.0.7339.186; disposable PostgreSQL 17.11 on loopback. HTMX is the repository's pinned 4.0.0 asset (`distribution/assets/htmx-4.0.0.min.js`, SHA-256 `e484d9171a9db30a39c8f16e3d709d4137f3211c659f8e6125816635033d593f`). No browser or asset was downloaded.
- [build-current.go](../../../probes/htmx-503/build-current.go) built the current distribution from this checkout with the verified local Bun archive. Its `canlc build` then compiled an unchanged copy of `examples/account-search` under `/private/tmp`. The build reported 23/23 assertions passing and build ID `e2d14ab3df2f7f7c75f17f35b3c54c04fe27df15ec24cad101a80abf7611d70a`.
- The existing `seed-driver.ts` setup created seven account rows in a disposable database. The generated account server used its normal fd-3 `ACCOUNTS_DB` credential and served `127.0.0.1:18563`. [probe.mjs](../../../probes/htmx-503/probe.mjs) opened `/accounts` in real Chromium, blocked non-loopback requests, and used the page's existing `#account_query` HTMX form. After the 200 and 422 controls, it renamed `can_i42_accounts` within that disposable database. The unchanged SQL loader's next query failed and the unchanged handler returned 503. The probe restored the table in `finally`; a subsequent count found all seven rows.
- The browser read each actual `GET /accounts/search` response status, body and content type, and compared `#account_results.innerHTML` immediately before versus 250 ms after the response. It checked that HTMX loaded and that the emitted `htmx-config` has `mode: "same-origin"`, includes 503 in `noSwap`, and excludes 422. The raw local observation is `/private/tmp/can-htmx-503-work/observations.json` (temporary, not a committed artifact).

| Browser action | HTTP status and body | `#account_results` before → after |
| --- | --- | --- |
| Search `Ann` | 200; `<ul><li>Ann</li></ul>` | `waiting` → `<ul><li>Ann</li></ul>` |
| Search empty input | 422; `<p>Enter a search term.</p>` | `<ul><li>Ann</li></ul>` → `<p>Enter a search term.</p>` |
| Search `Ann` with account table renamed | 503; `<p>Temporarily unavailable.</p>` | `<p>Enter a search term.</p>` → **unchanged** `<p>Enter a search term.</p>` |

All three responses had `Content-Type: text/html; charset=utf-8`. The 503 response fragment was delivered over HTTP but did not appear in the target. Thus the current page leaves stale validation feedback visible when the search fails. This is observed browser behavior, beyond the earlier source/policy inference in [server evidence](server-evidence.md). It does not determine whether a future product should swap 503 into this target, render an out-of-band notice, or use another feedback contract; those choices need the broader 409/403/503, focus, accessibility and out-of-order trials described in [product alternatives](product-alternatives.md).

The probe covers this one route, one target and a query failure caused by a missing disposable table. It does not test a network disconnect, removed target, focus, announcement semantics, retained form values after a failed write, or other failure statuses. No production/runtime source, vendor asset or canonical specification was edited. Runtime lint and checks were not applicable because this work changed only probe code and this note.

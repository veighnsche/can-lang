# dashboard

The admission example for polling and coordinated reads: a dashboard
page whose section refreshes every five seconds from two database
queries issued together through one captured pool.

## Layout

- `src/records/` — parameter, row, count, and view records.
- `src/model/` — `recent_count` and `account_total`, each mapping
  its descriptor's complete error bound to a known/unknown count.
  `recent_count` races two identical `read_recent` reads: the first
  success wins and only `all_failed` maps to unknown.
- `src/render/` — the dashboard page plus the summary fragment.
- `src/web/` — the two callbacks, route mounting, `boot`, and the
  serving `main`.
- `assets/site.css` — the only declared static asset.

## Run

```
canlc assert examples/dashboard
canlc build examples/dashboard
```

`main` takes one port argument and reads the `DASHBOARD_DB`
credential from its fd-3 environment snapshot. Schema and seed live
in test support at `tests/integration/testdata/applications/`; there
is no runtime migration API.

## Behavior

- `GET /dashboard` serves the page; the section polls
  `GET /dashboard/data` every five seconds and swaps the summary.
- Each poll issues the recent-accounts and account-total reads
  concurrently; known pairs answer 200 with the counts and clock
  stamp, while any unknown count answers 503 with a fixed fragment.
- Unknown routes answer compiler-owned 404/405 without callbacks.
- SIGINT/SIGTERM stops the server, closes the pool, and exits zero.

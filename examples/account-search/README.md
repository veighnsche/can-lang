# account-search

The P13 admission example: accounts over HTTP, SSR, HTMX, and SQL.
One page serves search, validation, and a polling dashboard section
from four named mounted callbacks; the search path queries PostgreSQL
through a static manifest descriptor and renders immutable rows as
escaped HTML.

## Layout

- `src/records/` — `search_parameters`, `account_row`, `search_view`,
  `account_form`.
- `src/model/` — the P13 consumer layer: `sample_loader`,
  `search_accounts_model` (complete six-error mapping), and the
  production `pool_loader`, which captures one open `sql::pool`
  through `near`.
- `src/render/` — safe page and fragment renderers; every display
  name passes through `html::text`.
- `src/web/` — the four callbacks, route mounting, `boot`, and the
  serving `main`.
- `assets/site.css` — the only declared static asset.

## Run

Build a development distribution, then assert, build, and serve:

```
canlc assert examples/account-search
canlc build examples/account-search
```

`main` takes one port argument and reads the `ACCOUNTS_DB` credential
from its fd-3 environment snapshot (never the environment). Schema and
seed live in test support at
`tests/integration/testdata/applications/seed.sql`; there is no
runtime migration API.

## Behavior

- `GET /accounts` serves the page; search submits
  `GET /accounts/search?query=…` and swaps a result fragment.
- Blank, missing, or repeated queries answer 422 with a validation
  fragment; SQL failures map to fixed 503/500 pages.
- `POST /accounts/validate` decodes `account_form` and answers 200
  or 422 fragments; `/dashboard/summary` polls every five seconds.
- Unknown routes answer compiler-owned 404/405 without callbacks.
- SIGINT/SIGTERM stops the server, closes the pool, and exits zero.

Callback error mapping is total, so mounted callbacks carry
`emits []`. Diagnostic logging is intentionally absent: every outcome
is a fixed safe response, and the sanitized boundary 500 remains the
fallback for standard failures.

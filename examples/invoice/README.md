# Invoice server example

Can-authored tenant invoice server: shared action contract, JSON API,
server-rendered HTMX form, SQLite storage with a same-transaction
replay ledger.

Follow the [clean-build recipe](../../docs/syntax-taste/post-upgrade-clean-build-2026-09-26.md):
install the release, apply `schema.sql` plus the documented seed rows,
build the grid (`examples/invoice-grid`) and then this server with
`--browser-manifest`, and serve the paired entry with the installed
Bun (port argument, JSON credential snapshot on fd 3). `canlc run`
rebuilds without the pairing, so it cannot serve the grid page.

Key sources: `src/web/web.can` (routes, handlers, renderers),
`src/model/` (authorized reads/writes), `can.project.json` (SQL
descriptors — SELECT and RETURNING-free mutations only; DDL stays
operator-owned). The shared contract lives in
`shared/invoice-contract` and is vendored here under `vendor/`; both
targets import the same locked instance.

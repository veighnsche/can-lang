# Invoice grid example

Can-authored browser grid editing the invoice server's lines: shared
action contract, typed fetch client, immutable notice/pending/focus
state.

Build with `canlc build --target browser examples/invoice-grid` from
the installed toolchain, then pair it into the server build with
`--browser-manifest` as shown in the [clean-build recipe](../../docs/syntax-taste/post-upgrade-clean-build-2026-09-26.md).
The grid boots from the report-selected paired asset only; query keys
are `tenant` and `invoice` (canonical decimal spelling).

Key sources: `src/web/` (state, rendering, focus coordination),
`src/model/` (client operations). Browser bootstrap input arrives
only through `browser::query_parameter`; there is no `main(args)` on
this target.

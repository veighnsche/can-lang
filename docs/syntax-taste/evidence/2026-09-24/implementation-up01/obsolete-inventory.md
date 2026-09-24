# UP01 obsolete-call inventory

Source, test, grammar, reference, and maintained-example spellings the
upgrade replaces. There are no external compatibility consumers: these are
migrated or removed in their owning task, never aliased. All paths verified
at baseline `13b6cdb`.

## Required `handles` action grammar (owning task: UP05)

| Location | Obsolete spelling |
| --- | --- |
| `compiler/internal/syntax/native.go:402-414` | `action requires a handles clause`; grammar parses `handles <name>` |
| `compiler/internal/syntax/format.go:335` | formatter emits `handles <name>` |
| `compiler/internal/syntax/token.go:80` | `handles` contextual word (action use; `wrap` use is separate) |
| `compiler/internal/syntax/action_test.go` | positive/negative fixtures around required `handles` |
| `compiler/internal/check/actions.go` | handler checking for the `handles`-bound model (no `mount` support) |
| `compiler/internal/emit/actions.go` | emission for the `handles`-bound model |

`wrap ... handles native/emitted` (`syntax/native.go:470-506`,
`syntax/wrap_test.go`, `compiler/testdata/current/wrap/main.can`) is a
different construct and is **not** obsolete.

## Mirrored actions and never-served stubs (owning tasks: UP16/19/20)

| Location | Obsolete spelling |
| --- | --- |
| `examples/invoice/src/web/web.can:5-36` | `save_invoice`, `save_invoice_form`, `load_invoice` with `handles`, query-era paths (`/invoices/save`, `/invoices/{invoice_id}`), string-ID wires |
| `examples/invoice/src/web/web.can:46-53` | per-request pool open, body session token (`save_json_validated` etc.) |
| `examples/invoice-grid/src/web/web.can:11-49` | mirrored `save_invoice`/`load_invoice` duplicating the server contract |
| `examples/invoice-grid/src/web/web.can:33-53` | `save_stub` / `load_stub` never-served anchors |
| `examples/invoice-grid/src/web/web.can` fetch client | ad-hoc `fetch_load`/save client pre-dating `action::request`/`browser::fetch` |

## Test boot contract and bundler shims (owning tasks: UP11/13/15/23)

| Location | Obsolete spelling |
| --- | --- |
| `examples/invoice-grid/src/web/web.can:1024-1033` | `fn void main` with `given str[] args`, `boot(args[0])` |
| `examples/invoice-grid/src/web/web.can:1004-1020` | `boot(invoice_id)` taking a host-supplied string |
| `tests/integration/browser/build-grid.mjs` | 3-line boot entry + 4 `node:` shims (`async_hooks`, `util`, `fs`, `crypto`) |
| `tests/integration/browser/sha256-shim.mjs` | engine shim consumed only by the test bundler path |
| `tests/integration/browser/build-vectors.mjs` | codec-vector bundler, migrated if its bundler supplies semantic aliases (UP23) |
| `tests/integration/gate5_frontend_test.go` origin rewrite (~L437) | GET captured-path → query-route rewrite |
| `tests/integration/gate5_frontend_test.go:869,976-993` | query-spelling load route, old-path expectations, `L-route` |

## Server route/query workarounds (owning tasks: UP09/19)

| Location | Obsolete spelling |
| --- | --- |
| served `/invoices/load?invoice_id=…` spelling | query workaround for declared captured GET (see `gate5_frontend_test.go:976`) |
| manual result/codec tables in server call sites | replaced by checked declaration metadata + native adapters |

## Browser entry and event gaps (owning tasks: UP11/13)

| Location | Obsolete spelling |
| --- | --- |
| `browser::mount("invoice-grid")` + `open_view` boot shape | superseded by compiler-owned DOM-ready entry invoking zero-arg `main` once |
| four-field event snapshot, listener API without cancel | superseded by `browser::on_cancel_key` / `browser::on_cancel_event` + immutable snapshots |
| callback settlement discarding completions | superseded by the sealed reporter (one sanitized occurrence) |

## References and generated layouts (owning tasks: UP12/18/24)

| Location | Obsolete spelling |
| --- | --- |
| `runtime/platform/html.ts` HTMX policy (~L402) | global noSwap with only a 422 exception; no checked per-action cases |
| `compiler/testdata/current/assets/page.can` | affected non-invoice page render sources (UP12 migrates to checked local policy) |
| old generated browser TS layout / test-bundled JS | replaced by compiler-owned entry + verified `Bun.build` output (UP11/15) |
| `build-grid.mjs`-style byte assertions in Gate 5 | replaced by UP21 contract-edit legs (UP23 removes rewrite/static-edit expectations) |

## Explicitly retained (not obsolete)

Package identity, owner records, variants, lexical fixtures, SQL adapters,
Bun coordination, the distribution catalogue, supervised assertions, verified
build publication, and the pinned HTMX vendor bytes. The plan consumes their
current interfaces; no worker hand-edits `runtime/catalogue.ts`, generated
Go/catalogue mirrors, or the pinned asset.

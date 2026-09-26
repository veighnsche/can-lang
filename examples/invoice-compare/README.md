# Invoice compare example

Can-authored browser board holding two independent invoice dock
panels side by side: the shared keyed-table, field and error
controls consumed unmodified, one state view and cell per
instance, and honest disposal — dropping the left panel leaves
the right panel working.

Build with `canlc build --target browser examples/invoice-compare`
from the installed toolchain. The board boots from seeds; there
are no query keys and no server calls.

Key sources: `src/web/` (two-instance render, folds, disposal),
`src/model/` (app-owned dock rows over the shared controls). The
shared controls live in `shared/grid-controls` and are vendored
here under `vendor/`; both browser apps import the same locked
instance.

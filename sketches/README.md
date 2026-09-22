# sketches — retired predecessor gallery (historical note)

All twenty programs below are retired artifacts in predecessor
syntax (`mod` blocks, `rev` markers, effects, fuel recursion, proof
brands, lint squiggles). They do not compile under the current
compiler, no gate or test may depend on them, and I44 deletes every
`sketches/*/` program. Their domains live on in current syntax in
the compiler fixtures and the admitted `examples/` applications.
`CLEAN_ROOM_REVIEW.md` stays as a labelled historical document.

The gallery once demonstrated, per program:

- `auth-login/` — password login with one retry.
- `retry-loop/` — retry with fuel; the termination demo.
- `counter/` — a bounded counter; the effects demo.
- `lint-errors/` — one fixture per `canlc lint` rule, each linting dirty.
- `broken-login/` — one broken file per editor squiggle.
- `bare-returns/`, `fn-callback/`, `fn-scalar/`, `fn-variant/`,
  `generic-option/`, `success-values/`, `variant-return/` —
  function-value, callback, and success-shape studies.
- `fibonacci/`, `fizzbuzz/`, `leap-year/`, `hello-world/` — small
  computation studies.
- `form-state/`, `notify/`, `reserve/`, `host-clock/` — form,
  notification, reservation, and clock studies.

Historical claims below the line (zero diagnostics, lint-clean
gallery, `given`-table checks) describe the predecessor toolchain
and are no longer true; they are preserved for the record.

## Historical gallery text

# sketches — can-lang use-case examples, one folder per example

- `auth-login/` — password login with one retry (`db.can` + `auth.can`).
  Clean: opening these shows zero diagnostics.
- `retry-loop/` — retry with fuel (`retry.can`): the termination +
  iteration demo. The loop is a proven self-call; minus `decreases`
  it is a cycle error. Clean: zero diagnostics.
- `counter/` — a bounded counter (`counter.can`): the effects
  demo. One private cell, declared capabilities, per-test
  isolation proved by the tables. Clean: zero diagnostics.
- `lint-errors/` — one proving fixture per `canlc lint` rule
  (`chain`, `merge`, `outcome`, `ranges`, `redundant`, `relay`,
  `table`): every file compiles and passes, and every file lints
  dirty. Open them with the editor extension installed to see each
  lint error live (red, error-severity — the linter has no warnings).
- `broken-login/` — same domain, one broken file per squiggle (`db.can`
  provider plus `missing-tests`, `bad-name`, `missing-rev`,
  `failing-test`, `missing-arm`, `dead-script`,
  `foreign-raise`, `unknown-call`, `unknown-emits`,
  `incomplete-expectation`, `missing-exchange`). Open them with the
  editor extension installed to see each diagnostic live.
  (`dangling-test` was retired by a91: omission is legal now,
  so "test with no script" is no longer a squiggle.)
  Each example folder is named by its contents.

Clean means compiler-clean AND lint-clean: gallery sketches carry
no findings of either kind (`canlc` passes, `canlc lint` exits 0).

Language regression fixture: `success-values/` demonstrates B11 whole generic
successes (`Ok<T>(value)` / `on Ok<T> value`), record-valued optional API
prototypes, callbacks, and trusted host execution. Its 54 rows and committed
TS/catalogue outputs are gated by `compiler/success_value_test.go`; see
`docs/b11-generic-success-values.md` for scope and reproduction.

Rules: `/REQUIREMENTS.md`. Design rationale: `CLEAN_ROOM_REVIEW.md`.

Checks: `go run ./tools/modcheck` (every `uses` resolves to
another file's `provides` + pins a rev; no `extern` re-declaration, no
`externals` sections, no curly braces, every `given` table total over
its tests).

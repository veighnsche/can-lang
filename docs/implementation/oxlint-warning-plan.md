# Oxlint warning cleanup plan

## Completion (2026-09-23)

The cleanup is implemented. `bun run lint:runtime` now uses
`--deny-warnings` and exits cleanly with zero diagnostics. The verifier
workflow installs pinned Oxlint and runs that command. A full local
`bun test runtime/ tools/runtime/` run passed (526 pass, 28 live-service
skips), and `go test ./...` passed. Intentional adversarial tests,
control-byte validators, the `JSON.parse` reviver receiver, and WebSocket
snapshot iteration have narrow explained suppressions; no authored
directory or whole rule was excluded.

## Scope and baseline

The pinned Oxlint 1.85.0 command `bun run lint:runtime` checks authored
`runtime/` and `tools/runtime/` TypeScript, excluding vendored code. A JSON run
on 2026-09-23 found 73 warnings in 178 files: 48 `eslint(no-unused-vars)`,
11 `unicorn(no-new-array)`, 6 `eslint(no-control-regex)`, 5
`unicorn(no-thenable)`, 2 `unicorn(no-useless-spread)`, and 1
`typescript(no-this-alias)`. No lint errors remain. Preserve Bun 1.4.2 runtime
behavior, Can's ownership and immutability contracts, and negative-test intent.
The user's unrelated untracked gap-fill plan is outside this cleanup.

## Work order

1. **Resolve the 48 unused names by role.** Remove imports, constants, helpers,
   and variables only after checking references and side effects. Keep setup
   calls such as `installFake()` even if their returned value is unused; remove
   the binding or assert on it if it is part of the test's purpose. For the
   eight `fake` bindings in `transaction.test.ts`, check each fixture's restore
   path before editing. Remove dead test-only scaffolding (`dockerized`,
   `sleepParams`, and similar names) only after checking that it does not mark
   missing coverage. Rename required positional parameters such as crypto
   `context` to `_context` rather than changing callable signatures. For unused
   catch values, use an optional catch binding only if no original cause must
   be preserved. Where an unused test result represents an observable action,
   assert its result or use `void` deliberately instead of silently dropping
   the action. Re-run lint after this phase so newly exposed warnings are seen.

2. **Review all 11 array constructions by their actual semantics.** `new
   Array(n).fill(value)` can become `Array.from({ length: n }, () => value)`
   when the output is meant to be dense. Preserve sparse arrays where holes
   are the point of a validation test (`domain.test.ts` and similar cases);
   use an explicit sparse-array construction or a narrowly documented lint
   exception. For `codec/json.ts` and `coordination.ts`, check whether the
   length-only allocation is later filled by indexed writes and whether holes,
   getters, or JSON serialization are observable before choosing a replacement.
   Verify million-element budget tests without weakening their size boundary.

3. **Keep adversarial and validation behavior explicit.** The five
   `no-thenable` findings are tests that intentionally install `then` members
   or accessors to prove the runtime never assimilates them. Retain those
   test inputs; use precise, explained line-level suppressions if Oxlint has
   no syntax that keeps the same test. The six `no-control-regex` findings
   reject control bytes in paths, URLs, headers, media types, and source map
   paths. Keep each rejected range unchanged and document a local suppression
   beside each regex rather than replacing the validation with a less clear
   character loop. Run focused tests for every affected boundary.

4. **Inspect the three remaining production warnings individually.** In
   `owner.ts`, determine whether copying `root.reports` before `Promise.all`
   is needed for a stable snapshot; remove the spread only if direct iterable
   consumption is equivalent. In `websocket.ts`, `unregister` can mutate the
   iterated session set, so preserve snapshot iteration unless evidence shows
   direct iteration is safe; `Array.from(owned)` may express the intent better.
   In `codec/document.ts`, the `JSON.parse` reviver deliberately uses its
   dynamic `this` as the holder for exact numeric tokens. Preserve that API
   behavior, with a narrow explained suppression if necessary.

5. **Prove and enforce zero warnings.** After each group, run Oxlint's JSON
   output and require zero diagnostics, including warnings. Run focused Bun
   tests for edited code, then `bun test runtime/ tools/runtime/`, the
   relevant Go output-validation tests, and `go test ./...` for the final
   cross-language gate. Keep native Bun version at 1.4.2. Once the baseline is
   clean, add `--deny-warnings` to the scoped Bun command and run it in CI
   after `bun ci`. Pinning and vendor exclusions stay as they are. Do not
   disable whole rules or exclude authored directories merely to make the
   count zero.

## Exit criteria

- The scoped Oxlint JSON report contains zero warnings and zero errors.
- Every intentional construct still has a focused assertion or an explained,
  local suppression; no broad category disable is added.
- Focused and full runtime tests, output-validation tests, and the repository
  test gate pass on the pinned toolchain.
- CI fails when a new warning appears in authored runtime TypeScript.

Three fresh Jev consultations about remediation policy and gate timing are
saved with their responses in
`evidence/2026-09-23/oxlint-warning-plan/`. All selected targeted fixes and
enforcement after cleanup; their agreement is advisory, not validation of
individual code changes.

# Main integration checks

The user authorized integration into main on 2026-09-28 after the implementation
handoff. The earlier checklist and verification documents record that uncommitted
handoff; this file records the subsequent merge preflight.

The SQLite change was committed and rebased without conflicts onto main
`721e1e0cc9251228b9093953b32bbb650d9ce25c`. A separate read-only reviewer found no
overlapping paths or semantic conflicts: main's new static-origin and callable
forwarding behavior does not alter SQL descriptor serialization or SQLite policy.

All combined checks passed; exact commands/results are in
[integration-checks.json](integration-checks.json):

- Full SQL package: 106 passing test/subtest events. The live MySQL differential
  test was skipped because CAN_TEST_MYSQL_URL was absent.
- Focused checker/emitter SQL, pool inputs, static-origin and callable-forwarding
  checks: 41 passing test/subtest events, no skips. CAN_BUN was explicitly set,
  and the shared dependency link enabled the strict TypeScript check.
- Runtime lint, formatting and type check passed.
- The five affected runtime suites: 42 passed, 14 live PostgreSQL/MySQL cases
  skipped, 0 failures.

The initial sandbox attempt stopped before any test body when it could not access
the normal shared Go build cache; the approved retry used that same shared cache.
The temporary dependency symlink was registered for cleanup before creation and
removed after checks; the shared target was preserved. No private cache, new
checkout, benchmark or distribution build was created.

While checks ran, main advanced to
`b4cc9ee83c6f64b2b156f117bc3bdddae8c73b5e` with seven documentation-only changes.
They are included in the final rebase. Integration requires a final check that
production sources are unchanged from the verified combined commit, followed by
a fast-forward of the main checkout. Git must refuse conflicting checkout work
or divergent main changes rather than overwrite them. No remote push is part of
this local merge.

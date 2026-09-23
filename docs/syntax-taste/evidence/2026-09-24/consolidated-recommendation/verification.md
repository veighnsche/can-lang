# Consolidation verification

The consolidation changes documentation only. The earlier reviews' executable probes, full source inventories, and test logs remain linked from this directory's `README.md`.

Checks run after writing the unified program:

- `bun run check:runtime` passed; output in `runtime-check.txt` (lint, formatting check, authored runtime/tooling TypeScript typecheck).
- `GOCACHE=/tmp/can-saas-review-go-cache go test ./compiler/internal/check ./compiler/internal/project` passed; output in `compiler-tests.txt`. Go reused the checker package cache and executed the project package tests.
- `bun test ./runtime/test/coordination.test.ts ./runtime/test/assertions.test.ts ./runtime/test/html.test.ts ./runtime/test/sql-descriptor.test.ts` passed: 38 tests, zero failures, 582 expectations across four files; output in `runtime-tests.txt`.
- Documentation file-link, trailing-whitespace and consultation-shape checks are recorded in `artifact-validation.json`.

These are consistency checks for a documentation-only consolidation. They do not qualify Linux, implement a proposed syntax, or execute the invoice, webhook and browser acceptance slices.

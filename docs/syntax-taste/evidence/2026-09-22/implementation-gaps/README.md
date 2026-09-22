# Targeted verification evidence

Read the [finding report](../../../../implementation/implementation-gap-verification-2026-09-22.md) for classifications and limits. This directory contains empirical probes, not approved new language syntax.

- `projects/`: 13 complete, minimal current-syntax Can projects (manifest, empty error registry, source). Builds/tests executed copies under the temporary workspace named in `workspace.txt`, not the saved evidence inputs.
- `cli-results.json`: 11 builds, 8 assertion runs and 4 real LSP sessions, including raw messages and current-generation comparisons. The recursive stress run fails on its own; only the pending-race run is externally stopped.
- `transaction-cli-results.json`: two additional transaction-signature projects, each built and asserted successfully. Their supplied fixtures are static/consumer evidence, not runtime transaction execution.
- `resource-lifetime.test.ts` and `resource-runtime.txt`: four lifetime-shape checks plus one existing transaction lifetime control.
- `sql-escape.test.ts` and `sql-runtime.txt`: four real transaction-adapter return-shape checks with fake native SQL, plus the existing in-scope query/commit control. The setup is copied from the existing runtime SQL test, with import paths adjusted; no runtime implementation was changed.
- `provenance.json`: checkout identity, absence of implementation changes, fresh bundle/launcher hashes, pinned Bun version, inspected implementation hashes and every saved project-file hash.
- `run-probes.py`: replay the CLI/LSP checks, now including both transaction-signature projects. It copies missing projects into the supplied workspace, saves results and externally terminates a test process group when its finite deadline expires. It does not claim the compiler provides that deadline.

## Reproduction

From the repository root, build a fresh sidecar with the pinned archive already on disk:

```sh
go run ./tools/distbuild --archive /absolute/path/to/pinned-bun.zip --out /private/tmp/new-can-probe-bundles --version gap-recheck
```

Use the resulting launcher's absolute path and a new canonical temporary workspace:

```sh
python3 docs/syntax-taste/evidence/2026-09-22/implementation-gaps/run-probes.py /absolute/bundle/bin/canlc /private/tmp/new-can-gap-projects
```

The runner writes new evidence results in this directory; copy it elsewhere first if preserving the original run verbatim. It uses only deterministic local projects. A replay contains 27 total CLI/LSP invocations; the original two-phase run saves 23 plus 4 in separate files.

Run the bounded runtime checks with the same bundled Bun:

```sh
/absolute/bundle/runtime/bun test docs/syntax-taste/evidence/2026-09-22/implementation-gaps/resource-lifetime.test.ts runtime/test/transaction.test.ts --test-name-pattern 'escaped |the handle dies with its scope'
/absolute/bundle/runtime/bun test docs/syntax-taste/evidence/2026-09-22/implementation-gaps/sql-escape.test.ts runtime/test/transaction.test.ts --test-name-pattern 'transaction commit returns|commit runs scoped queries'
```

No real database or network operation is used. Native SQL is replaced only within the isolated runtime test process and restored after each case. Ten selected tests passed; fourteen unrelated SQL tests were filtered out in each invocation. No whole-suite result or exhaustive lifetime proof is claimed.

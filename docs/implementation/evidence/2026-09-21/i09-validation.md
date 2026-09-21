# I09 validation

Completed 2026-09-21. The [output protocol](../../output.md) describes ownership,
content identity, publication, recovery and inherited run leases.

| Requirement | Evidence |
| --- | --- |
| Identical inputs produce identical content hashes | Artifact ordering/copy tests, option/input digest binding and generation reuse tests in `i09-output-tests.txt` |
| Exact relative ESM imports and one private runtime | Structured emitter tests cover value/type-only edges and missing targets; packaged execution shares opaque runtime identity across two generated modules |
| Safe refusal | Tests reject unowned/nonempty/symlink output, changed owners, unknown or modified files, path/case/file-directory collisions, duplicate metadata keys, stale source/assets and unvalidated native syntax |
| Atomic publication and recovery | Interruption tests cover reservation, file writes, completed staging, generation rename and pointer publication; recovery preserves unknown files |
| Concurrent build and source removal | Cross-process writer lock test; removed-source rebuild changes identity and prunes the obsolete generation |
| Clean/rebuild and active generations | Clean preserves unknown content and live leases; a Bun child retains its generation after the parent's descriptor closes, then pruning succeeds after child exit |
| Offline native execution | `i09-offline-integration.txt` records packaged generated execution and sidecar validation with network denied by macOS sandbox |

Validation commands passed:

```sh
CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache go test -count=1 ./...
/Users/vince/.bun/bin/bun test runtime tests/conformance
GOCACHE=/tmp/can-i02-go-cache go test -count=1 -v ./compiler/internal/driver ./compiler/internal/emit -run 'Artifact|Output|Publication|SourceRemoval|OwnerRevalidation|CurrentManifest|ProjectLock|ExactESM'
CAN_BUN_ARCHIVE=/tmp/can-i01-bun.zip GOCACHE=/tmp/can-i02-go-cache go test -count=1 -v ./compiler/internal/driver ./tests/integration -run '^(TestPackagedOutputValidationAndExecution|TestDevelopmentSidecar)$'
python3 distribution/qualify.py --archive /tmp/can-i01-bun.zip --report docs/implementation/evidence/2026-09-21/i09-native-capabilities-v1.json
GOCACHE=/tmp/can-i02-go-cache go run ./compiler/internal/catalogue/cmd/cataloguegen --check
git diff --check
```

Full Go results are in `i09-go-tests.txt`. Runtime/conformance results are in
`i09-runtime-tests.txt`: 29 passed, zero failed, 1,067 expectations. Native target
qualification is stored separately in `i09-native-capabilities-v1.json`; it also
requires the Bun transpiler APIs used before publication. No fallback is admitted.
The three publication-design consultations and their raw results are in `i09-jev/`.

The predecessor writer is test-only and the old `--out` CLI route is rejected.
This task supplies the tested emitter/driver interfaces and `clean`; completion
bodies and current-language build/run routing follow I10/I11. Type-only imports
come from structured compiler emission, not arbitrary authored TypeScript.
Detailed source-map production remains I40.

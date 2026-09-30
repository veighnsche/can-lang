# Server and configuration review

Independent review covered scheduler/lifecycle, cache identity, source URI mapping and JSON validation. Configuration parser review found no additional concrete offset-rebasing or validation defect; direct regressions exercise joined findings, nested lock/registry/SQL keys and values, duplicate origins, Unicode/CRLF and EOF insertion.

Material server findings and repairs:

- Protocol exit drained queued work. Exit now cancels and returns immediately; shutdown cancels work, clears queues and refuses subsequent changes. Canned request tests drain at EOF instead of treating exit as a request barrier. Deterministic paused-worker tests cover both commands.
- Watched changes after the last buffer closed did not refresh existing publications. Published project roots and current workspace manifests now schedule diagnosis, including closed-file findings.
- Real/symlink URI aliases shared an overlay entry with destructive close and nondeterministic URI selection. Preserve every buffer and reconcile the overlay deterministically; every alias version enters cache identity. Identical text receives findings under each original URI/version. Divergent text produces visible conflict status and clears old findings. Duplicate aliases explicitly refuse source edits. See `jev-alias/decision.md` for the three advisory consultations and rationale.
- Additional root audit: malformed header input is bounded before newline allocation; required coordinates cannot silently become zero; partial valid manifest fields remain cache inputs; escaped assets are not hashed; transient disk mutations have bounded retry and visible failure; untitled related locations retain original URI and unavailable locations never fabricate a URI.

Validation: `go test -p 1 ./compiler ./compiler/internal/project -run '^Test(LSP|Server|Publish|G0[1-5]|Config)' -count=1 -timeout=90s` passed (compiler 2.808s, project 0.193s). This includes all existing editor G01–G05 regressions plus the new lifecycle/configuration cases. Final race and whole affected-package checks remain pending after remaining compiler/feature review changes.

## Final compiler and package joins

The final compiler-specific correction gate passed full resolver/checker/driver suites (0.414s / 12.911s / 10.986s), including nested annotation children, multiple match scrutinees, duplicate metadata plus invalid value, and exact alias-token warning spans. Independent reviewer re-read those changes and closed the concrete findings.

Root release review found that the host fingerprint omitted the compiler's local `distribution` dependency and its embedded assets. The fingerprint now includes that tree, the pinned extension dependency lockfile, and `.vscodeignore`. Membership regressions cover embedded inputs and packaging controls. The client catches a rejected settings notification during a server transition rather than leaving an unhandled promise rejection.

`bun run test`: 24/24 client, real TextMate/Oniguruma and provenance tests passed. After the final lockfile/exclusion fingerprint addition, the three provenance tests passed again. `go test -p 1 ./tools/gramcheck -count=1 -timeout=30s` passed (0.266s); `go run -p 1 ./tools/gramcheck` passed the actual asset/corpus check.

The remaining editor integration now distinguishes sealed signature facts from executable body validity, so unfinished bodies can retain proven input-member assistance. Its focused final gate, final LSP race/integration gate, host build/install and actual Cursor verification are pending.

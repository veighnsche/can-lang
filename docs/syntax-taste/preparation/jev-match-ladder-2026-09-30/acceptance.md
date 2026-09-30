# String-match ladder warning acceptance

Implementation: `4f9c381a`; language-server regression and guide: `0e85e2fa`.

The checker emits one nonblocking `CAN-CHECK-STRING-MATCH-LADDER` at the first qualifying condition. It recognizes the same resolved immutable string binding, literal equality and OR groups, distinct decoded cases, either boolean arm order, and direct false-arm continuation. Terminal and ordinary value matches are covered. It leaves behavior and emitted code unchanged.

Validation passed with the existing shared Go cache and at most two workers:

- `GOMAXPROCS=2 go test -p=2 ./compiler/internal/check -count=1`: full checker suite, 14.155 seconds.
- `GOMAXPROCS=2 go test -p=2 ./compiler/internal/driver -run '^(TestStringMatchLadderDiagnostics|TestReportWarnings|TestCheckJSON|TestRecoveryReview|TestCheckSnapshot)' -count=1`: diagnostic, structured checking, recovery and snapshot regressions, 1.336 seconds.
- `GOMAXPROCS=2 go test -p=2 ./compiler -run '^(TestLSPStringMatchLadderWarning|TestG01|Test.*Warning)' -count=1`: wire publication, warning severity and formatting regressions, 0.599 seconds.
- Focused tests cover maximal-chain deduplication, grouped and reversed comparisons, value matches, nested successes, differing bindings, overlapping decoded strings, calls, projected subjects, intervening bindings, negative/relational/chained comparisons and already flat cases.
- The structured checker and editor preserve the warning after an unrelated failed statement in the same body. Warnings alone still accept the program.
- `git diff --check` passed.

The canonical host compiler was rebuilt with a single Go build worker and local macOS signing, without a runtime distribution or application build. The first package verification rejected a concurrent change to Git HEAD even though the source digest was identical. Rebuilding with the current revision passed provenance/signature verification and the 18-capability initialize smoke. The verified temporary VSIX installed successfully through Cursor CLI. Its installed binary hash/version matched the reviewed package. A bounded real-process LSP probe using that installed binary published exactly one warning for a ladder and zero diagnostics for its flat replacement. Exact evidence is in `delivery.json`.

The installed build is `0.2.0+baaead7af94e.50dced1a0dd6`. Restart **Can: Restart Language Server** in existing Cursor windows to activate it. Installation and protocol behavior were verified; no live editor window was restarted automatically.

All owned temporary projects and the VSIX were reclaimed. The existing canonical development binary/provenance and installed extension are retained. Unrelated native-testing edits and commits were preserved. No performance measurement, runtime assertion or broad application build was run. `inspect-types` remains a declaration-inspection command and does not publish advisory warnings; validation of the installed warning used LSP rather than interpreting its inspection output as lint diagnostics.

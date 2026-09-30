# Delivery

All implementation is committed on main: ab764c08 (editor/compiler repair), 611ca5e5 (VSCE README normalization), 7fa98c82 (actual client logging/start-failure cleanup APIs). Existing AGENTS.md edits are preserved.

The final host package build, signature/provenance verification and 18-capability protocol smoke passed for `0.2.0+7fa98c82a7ab.feea7763a822` on darwin/arm64. Cursor CLI confirmed installation of Can 0.2.0. The obsolete `keyword.declaration.error.ail` setting was narrowly corrected to `.can`.

The controlled source probe passed exact simultaneous diagnostic tokens (`missing_one`, `missing_two`, warning `alias`), local hover/definition, completion, constructor signature, versioned rename, references, symbols, semantic tokens, folds, formatting and inert real fixture/dependency inspection. Its initial fixture incorrectly exported a private record type; the fixture was corrected without changing application sources. Evidence: release-protocol.json.

Actual initial Cursor activation uncovered the LanguageClient 10 LogOutputChannel requirement and its non-running stop precondition. Those existing installation defects were fixed, checked against the installed library API, and covered by 11 passing client tests; rebuilt package and install succeeded. Live reload confirmation is pending because native UI reload attempts did not reliably apply. User was asked for one reload. No new investigation or artifact work is authorized by the user's latest cost constraint.

Temporary VSIX removed after confirmed install. Stopped owned tmux session had no viewers and was retired. Worktree archival follows; controlled probe remains only until live activation confirmation, then it is removed. The existing canonical host binary/provenance are retained for development; no private caches or application source changes were made.

Managed worktree archival confirmed by list_artifacts. Matching heartbeat was deleted to prevent further cost. Final installed code revision is 7fa98c82; later documentation commits do not change release inputs. Live activation is explicitly unverified pending the requested manual reload.

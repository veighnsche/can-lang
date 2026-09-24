# Can worktree committed-content audit — 2026-09-25

Read-only git inspection with `GIT_OPTIONAL_LOCKS=0`; the evidence file is the only write. Baseline `main` was `fbd2a5614dbba660b085b6fec8aef4f5d902c051` during this audit.

## Registry count

`git worktree list --porcelain` currently lists **36 registered worktrees**: main plus **35 extras**. My initial report of 35 total was a manual counting error; the listing itself has 36 `worktree` records. Of the extras, **7 HEADs are ancestors** of main (five `/private/tmp/lf12wt`–`lf16wt` worktrees and the two named Codex worktrees), leaving **28 nonancestor HEADs**: the named CI prefetch branch and 27 detached `.muse` task snapshots.

## Nonancestor HEAD coverage

The six commits on `codex/prefetch-go-modules` (HEAD `e3879429a5`) have the same *cumulative stable patch ID* as main's squash commit `c4a0386` (`90e1d53d723ce12e238e74f8a11a7a0b169797a7`). The main commit message names all six original changes. This branch has no missing cumulative patch despite `git cherry` marking its individual commits unmatched.

Of the 27 `.muse` task snapshots, 22 have a `git cherry main <HEAD>` result of `-` (patch equivalent in main). The other five have `+` because their patches were replayed amid other tasks. For each of these five, a main commit has the same title and the same changed-path set. I compared per-file multisets of added and removed lines from `git diff <original>^ <original>` against `git diff <integration>^ <integration>`:

| Task | Original → main integration | Difference and coverage |
| --- | --- | --- |
| T08 | `2eacc4c` → `cd2ec97` | All changed lines match except `compiler/internal/syntax/token.go`'s `contextualWords` line. The original adds `scenario link`; main adds the same words after the already-integrated `owner`. Current main retains `owner scenario link`. |
| T12 | `3230984` → `c8a2770` | Every added and removed line matches, per file. Stable patch IDs differ only due to patch context/order. |
| T13 | `0714ce3` → `b4051c4` | Every added and removed line matches, per file. Stable patch IDs differ only due to patch context/order. |
| T22 | `0581011` → `cea1500` | All browser-specific changed lines match. Differences are catalogue count/hash constants and one generic-operation condition in `compiler/internal/check/program.go`. Original catalogue increments are +1 package, +6 types, +4 errors, +18 operations; main's integration has the same increments from a larger baseline after T12/T13. Main adds `browserStateOperation` while retaining prior `formGenericOperation`; current main also retains it. Generated hashes differ because the integrated source catalogue has earlier tasks. |
| T24 | `952ac8c` → `ad1660d` | All changed lines match except the 13-line `platform/action-json.ts` dependency entry in `runtime/modules.json`. That exact entry already exists in `ad1660d^` and remains in current main; it was added by an earlier integration. |

Every file added by those five original commits exists at current main. No original behavior/hunk appears omitted from current main. The comparisons cover committed diffs, not uncommitted or ignored files in worktree directories.

**Recommendation:** no HEAD merge is needed based on committed content. In particular, blindly merging the stale CI branch or detached task snapshots would replay already integrated changes and may conflict with later main changes. Check worktree-local uncommitted and ignored contents separately before removal.

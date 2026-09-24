# Worktree consolidation evidence — 25 September 2026

The user requested that all worktree work be integrated and the extra checkouts
removed before starting the next implementation round. The initial registry
contained the main checkout and 35 extras.

- [Committed-content audit](committed-content-audit.md): all 35 extra HEADs are
  represented on main through ancestry, cherry-picks, or the CI prefetch squash.
  No new source merge is needed; replaying those old snapshots would duplicate
  already integrated work.
- [Dirty LF12–LF16 audit](dirty-lf-audit.md): all 137 modified/untracked paths
  exactly reproduce integration commits already on main, including modes.
- [Inventory](inventory.json): original paths, HEADs, dirty paths, ignored-file
  counts, and local archive refs retaining the original commits after removal.
- [Removal verification](verification.json): final registry and preservation checks.

The two files under `scratch/` are exact copies of untracked invoice diagnostic
experiments from the T14 worktree. They are retained as text for historical
reference, not installed as production tests: one requires a manually supplied
`SCRATCH_BUNDLE` and dumps a validation request to `/tmp`; the other is a scratch
invoice checker. Their original paths and SHA-256 hashes are in the inventory.
They provide no additional shipped language behavior.

Ignored files removed with the extra checkouts were generated `dist` output and
`node_modules` dependencies. Main-checkout build output, branches, and the
pre-existing stash are outside this worktree cleanup and remain intact.

The completed preparation and 27-task implementation plan were committed as
`65becc8` before cleanup. The plan validator passed (27 tasks, 11 fixes,
14 waves); `git diff --check` also passed. No runtime/compiler implementation
changed during consolidation, so runtime test suites were not rerun.

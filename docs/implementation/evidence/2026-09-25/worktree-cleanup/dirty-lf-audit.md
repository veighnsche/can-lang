# LF12–LF16 dirty worktree preservation audit

Date: 2026-09-25. Scope: `/private/tmp/lf12wt` through `/private/tmp/lf16wt`. This audit made no changes to the repository or those worktrees.

## Finding

The non-ignored contents of each dirty worktree reproduce its corresponding integration commit exactly. The five integration commits are already ancestors of `main`. There is no unique tracked or untracked work in these five worktrees to merge or preserve separately. Their dirty status results from keeping the completed LF changes on a checkout of the immediately preceding commit.

| Worktree | Worktree `HEAD` | Matching integration commit on `main` | Dirty paths | Exact at `main` now | Subsequently changed on `main` |
| --- | --- | --- | ---: | ---: | ---: |
| `lf12wt` | `7f9c066806f9` | LF12 `d1fd69bf455f` | 47 | 16 | 31 |
| `lf13wt` | `d1fd69bf455f` | LF13 `f04a6d0191bc` | 27 | 11 | 16 |
| `lf14wt` | `f04a6d0191bc` | LF14 `d8239cd4bf8f` | 25 | 8 | 17 |
| `lf15wt` | `d8239cd4bf8f` | LF15 `6898d11e7bc1` | 25 | 12 | 13 |
| `lf16wt` | `6898d11e7bc1` | LF16 `7c582004a81c` | 13 | 8 | 5 |
| **Total** | | | **137** | **55** | **82** |

The 82 files that differ from today's `main` still match their respective historical integration commit exactly. Their current differences reflect later changes along `main`, not missing work.

## Evidence and method

For each worktree, I freshly collected `git status --porcelain=v1 --untracked-files=all` and compared it with `/private/tmp/can-worktree-cleanup-inventory-2026-09-25.json`; the path lists and statuses are identical. Every status is either an unstaged modification or an untracked file. There are no staged changes. Fresh `git ls-files --others --ignored --exclude-standard` found zero ignored files in all five worktrees.

For every one of the 137 dirty paths, I read the worktree bytes and compared them to `git cat-file blob <integration-commit>:<path>`. All 137 matched exactly. I also compared each filesystem file type/executable bit to the integration commit's tree entry; all matched. For each pair, `git rev-parse <integration-commit>^` equals the worktree's `HEAD`, and `git diff --name-only <worktree-HEAD> <integration-commit>` yields exactly the worktree's dirty path set: 47/47, 27/27, 25/25, 25/25, and 13/13, with no paths present on only one side. Finally, `git merge-base --is-ancestor <integration-commit> main` succeeded for all five commits.

The comparison covers both source changes and untracked tests/evidence, including the LF12 wrap and assertion policy files, LF13 template files, LF14 fixture capture files, LF15 verified build files, and LF16 coordination evidence. It tests file content, path set, and mode rather than relying on similar filenames or commit messages.

## Removal implication

These five worktrees need no merge or file copy before removal. The corresponding complete snapshots are preserved in the LF12–LF16 commits on `main`. If cleanup occurs later, recheck `git status --porcelain=v1 --untracked-files=all` immediately beforehand so new work after this audit is not lost.

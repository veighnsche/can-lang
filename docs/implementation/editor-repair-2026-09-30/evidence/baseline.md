# F00 baseline — editor repair worktree

Date (UTC): 2026-09-30. Coordinator: Muse Spark 1.3 Contributor (this session).

## Repository state

- Workspace: `/Users/vince/.codex/worktrees/editor-repair/can-lang`
- Branch: `codex/editor-repair`
- HEAD: `9c283544466876700ef7a079f0bc3136451c0008` (matches checklist baseline)
- `git status --porcelain`: only `?? docs/implementation/editor-repair-2026-09-30/` (this task's own
  checklist/design/evidence, untracked by design). No modified tracked files, no foreign changes touched.
- Main checkout `/Users/vince/Projects/can-lang` and `/Users/vince/Projects/manolea-2`: not edited;
  main-checkout monitor read only for runtime state.
- AGENTS.md re-read 2026-09-30: zero-compat, native-compile, Jev 3-rewrite rule, runtime lint/format/check,
  disk/load bounds, temp-cleanup, worktree reuse, Muse-executor lane rules apply.

## Resources

- Disk (`df -h /`): 228Gi total, 7.1GiB avail (below 9.6GiB launch figure; tighter bound applies).
  Policy: one heavy compile/test/build at a time, `go test -p 1` selected packages, no benchmarks,
  no broad build matrix, shared caches only, owned temps with registered cleanup.
- Tools: `go1.27.1 darwin/arm64`, `bun 1.4.2`.
- No private full caches/bundles/source copies. No user Cursor install before Codex review.
- No execution of user Can programs/assertions/network during analysis.

## Native goal blocker

- Required: `create_goal` before delegation, then `get_goal`/`report_progress`/`update_goal complete` at H75.
- Observed: this Muse Code session exposes only `muse.*` tools
  (`workflow/read_file/search/write_file/edit_file/read_memory/add_memory/edit_memory/work_* /
  web_*/bash/bash_input/monitor/subagent_*/read_skill/work_status/snooze_reminder/write_todos`).
  No `create_goal`, `get_goal`, `report_progress`, `update_goal`, or token-budget goal schema is available.
- Disposition: blocker reported here and in checklist F00 evidence. No prose objective substituted.
  Coordination proceeds via this checklist + `write_todos` progress only. Delegation (if any) uses native
  `subagent_*` under the same checklist ownership/serialization rules, at most two implementation agents
  and one heavy command. `update_goal complete` cannot be issued; H75 handoff will state this explicitly.

## Initial targeted tests (pre-change reference)

- Design-established passing-but-insufficient suites: targeted source/driver/LSP G01–G05/grammar checks
  passed at baseline; they do not cover the new multi-diagnostic/range/lifecycle contracts.
- F00 runs no heavy suite; first bounded check is deferred to F01/F02 focused packages to preserve disk/load.
- Durable-test rule loaded: `bundled:durable-test-collateral` before first implementation edit.

## Lane ownership (exclusive writers)

| Lane | IDs | Files | Agent | State |
| ---- | --- | ----- | ----- | ----- |
| Coordinator | F00, F03 (contract), I70–I71, H75, checklist/evidence | `docs/implementation/editor-repair-2026-09-30/checklist.md`, `evidence/*` | this session | active, sole checklist writer |
| Compiler | F01–F02, P10–P13, R20–R27 | `compiler/internal/{source,syntax,project,resolve,types,check,driver}/**` (excl. frozen editor/server handoff files while owned) | coordinator inline (no separate writer yet) | starts after F00 |
| Grammar/client | G50–G53, C60–C62 | `editors/vscode/**`, `tools/gramcheck/**`, `.github/workflows/verifier.yml` (grammar parts) | unassigned, may start after F00 | blocked until compiler F00 done, then ready |
| Editor/server | E30–E39, S40–S43 | `compiler/lsp.go`, `compiler/internal/driver/editor*`, hover/definition/completion/rename/symbol/semantic/fold/inlay/format paths | unassigned | starts only after F03 contracts + explicit file transfer |

Serialization: `compiler/lsp.go`, driver files, manifests overlap — one writer at a time or interface-first
extraction. At most two coding agents at once; only one heavy `go test`/build at a time.

## Command cadence actually in force

- Requested native-goal cadence (2-min default reinspection, `yield_time_ms:120000`, handle/deadline state,
  runtime-overdue inspection only) cannot attach to a native goal because goal tools are absent.
- Applied substitute within available runtime: `muse.bash yield_time_ms:120000` (60000–300000 adjustable),
  await automatic completion, no empty `bash_input` polling, act immediately on completion/failure/input.
  Propagated to any subagent prompt verbatim if spawned.

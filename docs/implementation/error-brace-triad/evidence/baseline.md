# G00 baseline and leases

Date: 2026-09-29. Owner: Muse (implementation run `01a0ef1c-261c-7dd3-97a1-45794ce0451d`).

## HEAD and status

- HEAD: `4393a89ebf78ece033567db968265526802990c1` (`feat(html): allow validated email links in anchors`, 2026-09-29 23:30 +0200).
- Planning baseline `a129bf61` is now HEAD~1; the nine observed dirty HTML paths were committed as HEAD (10 files incl. new `compiler/internal/emit/email_href_test.go`).
- `git status --short` at G00:
  - `M docs/implementation/error-brace-triad/checklist.md` (Codex handoff status text)
  - `M docs/implementation/error-brace-triad/tasks.md` (Codex handoff status text)
  - `?? docs/implementation/error-brace-triad/evidence/` (Codex-owned `muse-monitor.md`)
- No implementation source path is dirty. The `html::email_href` feature is committed, not dirty.

## Tool versions

- `go version go1.27.1 darwin/arm64`
- `bun 1.4.2`, `node v24.21.0`
- `muse-bin-1.4.1-R4503.1`, model `muse-spark-1.3-contributor`, reasoning `max`, `--yolo`, `--worktree off`

## Worktrees

- Main: `/Users/vince/Projects/can-lang` on `main` at HEAD (implementation workspace).
- Foreign (do not touch): three `/Users/vince/.codex/worktrees/*` checkouts and six `/Users/vince/Projects/can-lang/.muse/worktrees/subagent-v2-*` detached checkouts from earlier sessions. None is owned by this run.

## Writer/owner check

- This Muse exec process (PID 73157 under owned tmux `muse-error-brace-20260929`) is the sole active implementation writer.
- No `go test`/`bun test` process running at G00. `lsof` on `compiler/internal/catalogue` shows no locks. `find -mmin -60` shows only HEAD-checkout mtimes plus task docs/monitor; no ongoing writes.
- Codex sandboxes with can-lang write scope exist (coordinator/monitor) but per `evidence/muse-monitor.md` Codex owns only session/heartbeat/process/temp state and V04 review; it holds no code-file lease.
- HTML paths (`catalogue.json`, `catalogue_test.go`, `generated.go`, `runtime_core.go`, `email_href_test.go`, `html/main.can`, `runtime/catalogue.ts`, `runtime/platform/html.ts`, `runtime/test/html.test.ts`, `std/catalogue/README.md`, `std/catalogue/errors.json`) have no live writer; committed at HEAD, free to read. Syntax-only edits to `html/main.can` remain authorized; generator-only rule still applies to mirrors.

## Lease decision

- All F/N/M/P lane files are free; Muse holds the exclusive implementation lease. No path requires waiting.
- Contested-path plan: none blocked. `html/main.can` syntax spans migrate under M01 with email-link lines preserved; catalogue mirrors regenerate only via the generator under P03 with diff inspection against this HEAD baseline.
- Addendum (2026-09-30, during G02): `docs/user-guide/README.md` gained a
  48+/4- hunk from a live foreign writer (verified platform notes: safe
  email actions, upload attributes/responses). Not Muse's work; P05 must
  preserve these lines and touch only old Can syntax spans in that file.
- Separation: syntax-change diff = everything after HEAD except the two task-doc status hunks and `evidence/` (Codex monitor + Muse evidence). `git diff HEAD --stat` plus this file distinguishes pre-existing HTML work.

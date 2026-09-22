# I50 acceptance — documentation and final traceability

Closed 2026-09-22. Every README now states the exact shipped
CLI, layout, and runtime; the docs index routes current readers
to the ledger, coverage, and dated evidence; all 50 tasks check
with passing evidence links; release notes name the candidate,
pins, gates, open gates, and limits. No release upload,
credentialed signing, or live provider call was performed.

Baseline `b5026cc` (I45) plus the I50 worktree. Apple M4
darwin/arm64, macOS 27.0, Go 1.27.1, Bun 1.4.2 (`744846f84`,
archive sha256 `90987a3a…6be1`, hash-verified), TypeScript 7.0.2,
Node v24.21.0, Playwright 1.55.1 with pinned Chromium,
PostgreSQL 17.11 (disposable `initdb` cluster on 127.0.0.1:5434;
port 5433 was held by an unrelated user Docker container and
left untouched).

## Design consultations

[i50-jev](../i50-jev/decision.md): three fresh Jev consultations
(all prose rewritten, same facts/options; 87/87 explanatory
strings pairwise distinct). Unanimous for relabel_index
(1.0, 1.0, 0.99): keep the index structure, repair false
claims, route readers to ledger/coverage/evidence. Unanimous for
banner_historical (0.67, 0.76, 0.85): keep REQUIREMENTS.md at
the root under a historical banner, frozen body untouched.
Unanimous in direction for distribution_section
(0.65, 0.46, 0.88) with a round-2 near-tie investigated on the
merits: release notes extend distribution/README.md where
shipping truth already lives. All three followed. Judgments are
advice; the checks below are the proof.

## Implementation

- `README.md`, `compiler/README.md`, `std/README.md` rewritten to
  the shipped CLI (`assert`, `build`, `run`, `parse`,
  `inspect-*`, `runtime-check`, `catalogue-check`, `version`,
  `lsp`, `clean`), the five-file launcher, the maintained
  example inventory, and exact prerequisites (Go 1.25+ with a
  filled module cache, C compiler at installer build time only,
  pinned Bun archive acquired separately).
- `docs/README.md` relabeled: current path
  (tasks/coverage/evidence/ledger) first, historical design
  records marked as such.
- `REQUIREMENTS.md` kept under a historical banner, body frozen.
- `docs/implementation/{tasks,coverage,plan}.md` reconciled to
  the shipped tree; every task row links its evidence.
- `distribution/README.md` extended with release notes:
  `can-<sha>-darwin-arm64-v1` candidate lineage, admitted
  target, shipped contents, gate verdicts, open gates (signature,
  notarization, upload, standalone installer binary), and limits
  (no Bun 1.4.4+ features, PostgreSQL-only SQL, upstream HTMX
  boundary, loopback AI evidence, no proof/termination/effects).
- `docs/a/*` archive links repaired: 20 root-relative and
  absolute-path prefixes corrected to same-directory targets,
  all verified to exist; 3 `sandbox:` review-harness URIs and 1
  never-committed prompt reference converted to labeled plain
  text so no dead link remains.

## Verification

Link audit over every non-vendored Markdown file: every
relative link (documents, sources, workflows, evidence) plus
GitHub-slug anchors, including `<a id>` targets:
604 links, 0 broken.

Clean-machine install→build→assert→run pass
(`/tmp/i50-clean-pass.sh`): fresh HOME, `GOPROXY=off`,
restricted PATH, documented prerequisites only. Release built,
installed into a fresh root, `runtime-check` identified the
pinned sidecar, `assert` on staged `std/scalars/current`
reported 40 passing assertions, `build` printed a `can.build`
report, `run` exited 0 with void completion. No hidden tool
prerequisite found.

Full gates with all services operated, zero skips:

- gofmt (empty), `go vet ./...`, cataloguegen `--check`,
  modcheck (62 maintained sources), gramcheck: all green.
- `qualify.py`: 14/14 native conformance checks pass.
- `go test -count=1 ./...`: all 16 packages ok, 0 failures;
  log contains 0 skip markers.
- `bun test runtime/test/`: 850 pass, 0 fail, 63285
  expect() calls across 142 files.
- tsc leg: `TestStdlibMaintained` emitted 504 fresh `.ts`
  files under `tscheck/.fresh-emit`; `tsc -p tsconfig.json`
  exited 0 with empty output.

Recount: 50/50 task checkboxes checked, 50/50 evidence links
present (`tasks.md`), every coverage row accounted for.

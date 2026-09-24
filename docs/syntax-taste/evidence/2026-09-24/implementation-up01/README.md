# UP01 — recorded inputs and execution boundaries

Coordinator integration record for the post-upgrade implementation round.
Task contract:
`docs/syntax-taste/post-upgrade-implementation-tasks-2026-09-24.md` (UP01);
correctness contracts: selected behavior + invoice gate. All 27 tasks were
**not started** at record time; this directory starts none of them.

## Contents

| File | Record |
| --- | --- |
| `baseline.json` | source commit, input/evidence digests, dirty state, toolchain identities |
| `prerequisites.md` | compiler/Bun/archive identities, browser/DB/Linux availability, UP24/25 operate-list |
| `acceptance-registry.md` | named +/− cases per Fix and invoice gate; current fail/skip vs pass |
| `obsolete-inventory.md` | obsolete action/browser calls in source, tests, grammar, references, examples |
| `interfaces.md` | internal handoffs I-1…I-7 (metadata, arities, context, imports, diagnostics, manifest, assets) |
| `ownership.md` | lane boundaries, per-task write sets, ordered handoffs, held outputs |

## Baseline

- Integration branch `integration/up01-inputs-boundaries` from `main` at
  `13b6cdb5afea513f06aa9a23ddede4dbe5f81eee` (clean tree; no dirty changes
  to preserve). Workers start from this commit.
- The plan's named source baseline `fbd2a56` omits the preparation inputs
  committed later; `13b6cdb` contains them. `baseline.json` pins both plus
  sha256 digests of the seven post-upgrade documents, four preparation
  contracts, five mechanism-experiment findings, and two native probes.

## Done criteria (UP01)

- Each worker can implement against named interfaces (I-1…I-7) and named
  evidence; no worker needs to rediscover the baseline state.
- Archives/engines that must be operated at UP24/25 are listed in
  `prerequisites.md` (pinned Bun archive, Chromium **and** WebKit,
  PostgreSQL service, pinned tsc, selected Linux installed root).
- No production claim relies on a missing fixture, placeholder helper, or
  skipped test; current failures/skips are recorded separately from passes.

## Out of scope

Everything in UP02–UP27. No production source, test, generated mirror, or
workflow was edited for this record; the branch adds only this directory.

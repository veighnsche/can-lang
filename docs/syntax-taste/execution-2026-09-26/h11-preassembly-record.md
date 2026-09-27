# H11 pre-assembly record — 2026-09-26

Status: **OPEN-behind-H12**. Aggregation of delivered owner seals complete; the
H12 W6-deploy native legs are the single pending evidence input
(BLOCKED-awaiting-UP25-grant). **No IC2 verdict is claimed.**

- Evidence: `docs/syntax-taste/evidence/2026-09-26/h11/h11-preassembly.md`
  (per-workload leg tables with owner-report citations + revision pins,
  version/env cross-check, double-count/unavailable-live audit, findings
  F0–F22, H12 pending-input section with the exact UP25 ask).
- Worktree base: `7480a791`. No live/heavy suite was run for this record; no
  file outside `docs/syntax-taste/` was touched; no network machine probed.

## Aggregation completeness

| Scope | Owner seal(s) | Completeness |
| --- | --- | --- |
| W1 (W1.1–W1.5) | C06 report + legs matrix + main re-qual | complete (W1.5 on substitute surface — F7) |
| W2 (W2.1–W2.3) | D03 record + conformance suites | complete as runnable+debt (F5, F6) |
| W3 (W3.1–W3.4) | B05 report (stale BLOCKED) + B01/B02 inputs + supporting H10/coordinator reruns | complete with staleness flagged (F4) |
| W4 (W4.1–W4.5) | A07 report + legs; F06 report + legs for W4.3 | complete (F9, F10 notes) |
| W5 (W5.1–W5.10) | E09 record + harnesses | complete incl. BLOCKED-evidence legs (F1, F2) |
| W6-data (W6.1–W6.2) | F04 report + live JSON + recipes; F02/F03 inputs | complete (F8, F14, F17, F18 notes) |
| W6-pair (W6.3) | F06 report + legs | complete (F13 note) |
| W6-deploy (W6.4) | H12 staging only | PENDING — §H12 below (F0) |
| W6-AI (W6.5) | H08 report + registration + X-R14-1 + recipe + plumbing run | complete as honest-BLOCKED (F3) |
| Authoring/tooling (AU-*, X-R08) | A01/A02/G01–G05 seals via H10 + reports; A08 comparison record | complete (F15, F16 notes) |
| Pre-pub examples (DOC-invoice/webhook/Linux) | C07 commit; F05 seal; H09 seal | complete as inputs (F12 note; H14 owns final closeout) |
| Gated syntax | H10 conditional table | all inactive-or-qualified (F8 execution gap flagged) |

## Findings routed

| Finding | Severity | Routed to |
| --- | --- | --- |
| F0 H12 pending (UP25 grant) | BLOCKING | user → H12 native run |
| F1 X-R15-3 scope return (W5 fails per contract) | HIGH | preparation + user scoping |
| F2 W5-L3 tx-ownership preparation return | HIGH | preparation (E owner) |
| F3 W6-AI grants + qualified bound U | HIGH | user + provider evidence |
| F4 B05 W3 re-seal post-emitfix | MEDIUM | B owner |
| F5 D live legs now runnable | MEDIUM | D owner |
| F6 W2-negative e2e gap | MEDIUM | H + D, or IC2 disposition |
| F7 W1.5 substitute-surface disposition | MEDIUM | preparation / IC2 disposition |
| F8 Can-path RETURNING execution gap | MEDIUM | E owner + coordinator |
| F9–F15, F17, F18 | LOW | owners / IC2 / H14 / H12-Leg-7 as listed |
| F16, F19–F22 | INFO | coordinator / record |

## What remains for an IC2 close (not done here)

1. UP25 grant → H12 native Legs 1–8 → W6-deploy evidence + lifecycle recipe.
2. Disposition of F1–F8 (scope returns, preparation returns, grants, re-seals,
   live runs, contract dispositions) — each owned as routed above.
3. Final IC2 verdict pass over the joined evidence (a later H11 turn, not this
   pre-assembly).

## Coordinator verification addendum (main, post-integration)

The coordinator independently verified the HIGH/MEDIUM findings against the
tree before accepting this pre-assembly:

- F1 CONFIRMED: task-list:933 (X-R15-3 negative row) requires returning
  storage scope/contract to preparation for explicit resolution — "no
  silent reduced scope or green IC2". A future IC2-green needs an
  explicit user scope decision alongside the UP25 grant.
- F2 CONFIRMED: `e09-w5.md:54-86` hands W5-L3 to IC2/preparation itself
  (drain-blocking overrun-tx; E04 semantic change, not a leg fix).
- F8 CONFIRMED: the only `runtime/` RETURNING hits are English comment
  words (`cli.ts:1`, `identity.ts:114`); checker-side
  `row_limit_parameter` plumbing exists (`manifest_sql.go`) but no SQL
  RETURNING execution path runs under `runtime/`. W6.1 generated
  identities rest on raw-SQL oracles + MySQL mapping + ledger
  identities, not end-to-end Can SQL.
- F11 RESOLVED by coordinator: `can-ff` is native aarch64 (`uname -m`
  inside the running container), pinned image
  `mcr.microsoft.com/playwright:v1.55.1-noble@sha256:2f293690…`,
  `--platform linux/arm64`, no qemu present. The `Linux x86_64` UA
  token is not an arch attestation (same class as Chrome's `Intel Mac
  OS X` UA on arm64 Macs). No action left for the C owner.
- F4/F5 acknowledged: F4 post-emitfix W3-green exists from coordinator
  (`failure-conventions` 11/11 on main) + H10 reruns; a B-owner re-seal
  at a post-emitfix revision is still the clean close. F5 D live legs
  are next in the coordinator queue (unblock commands runnable since
  C01-done).

IC2 stays OPEN behind H12 + F1/F2 dispositions at minimum. No verdict
claimed or waived by this addendum.

## F5 closure (coordinator-executed, post-pre-assembly)

F5 is CLOSED: the D02/D03 live unblock commands ran green on all three
pinned browsers — `TestLiveBrowserLegs` 8/8×3, `TestLiveVendorBLegs`
5/5×3 (Chromium 140.0.7339.186, WebKit 26.0, container Firefox 141.0).
Harness repairs (node-side asserts, Chromium-only clipboard grant,
container-FF connect + forwarded port 18651, explicit process exit)
are committed with the six `report.json` files under
`host/conformance/live/reports/2026-09-27/`; D02-LIVE-1..4 and
D03-LIVE-1..4 all close with per-engine pins in `host/d02-record.md`
and `host/d03-record.md` (""→null normalization confirmed, permission
matrix pinned, 250ms trip wire holds with >60× headroom,
secureContext true ×3). H11 §3.2/§7 "debt" language is superseded by
this closure; the §6 F5 finding stays as the audit trail.

## F4 closure (coordinator rerun, B-owner-noted)

F4 is CLOSED as far as evidence goes: the exact B05 rerun command
(`CAN_BUN_ARCHIVE=/tmp/bun-darwin-aarch64.zip go test
./tests/failure-conventions/ -v -count=1`) is green 11/11 at two
independent post-emitfix revisions — H10's rerun at `932924f7`
(`h10-battery.md:15`) and the coordinator's at post-H10 main (38s,
`/tmp/h10-cb.log` session log). No B lane is open to stamp an owner
re-seal, so this coordinator rerun pair stands as the W3-green
evidence with that provenance honestly noted; a future B-owner stamp
would be cosmetic only. (No third rerun was needed: the failure-conventions
bundle key covers `compiler/distribution/runtime/tools`, all
untouched since the coordinator's green run.)

## F8 scoping (coordinator investigation, for the user disposition)

F8's Can-path RETURNING gap is confirmed on all three handoff layers
(F03 `f03-report.md:51-57` specified but never implemented):

1. `compiler/internal/project/manifest_sql.go:91-103` REQUIRES
   `row_limit_parameter` for every row-returning cardinality —
   `one` cannot express RETURNING's absent-limit shape.
2. `runtime/platform/sql/descriptor.ts:100` allows `limit === 0`
   ONLY for `execute` — `(one, 0)` throws `invalid sql limit`.
3. `runtime/platform/sql/pool.ts` has no RETURNING execution path
   (bind-app-params-only + §3 shared-decoder enforcement unbuilt).

Closing F8 means E-lane implementation on all three layers + new
runtime/live legs across PG/MySQL/SQLite, intersecting open defect
F17 (cold-pool `withTransaction` burst, same E area). It is new work
beyond the 52 closed task records — not a hidden subtask of any of them —
so it needs the user's implement-vs-exclude call (F8 disposition).
W6.1 "generated identities" stands evidenced via raw-SQL oracles +
MySQL `LAST_INSERT_ID` mapping + ledger identities either way.

## F7 resolution (documented-limitation justification, evidence-backed)

F7's premise holds (no program declares a document-mode GET action:
invoice actions are GET-json, POST-json, POST-html-inner —
`shared/invoice-contract/.../invoice_contract.can:98-133`), but the
W1.5 functional content is proven across three covered layers, leaving
no document-mode-specific decision logic unproven:

1. Denial decisions on captured actions: `missing-id-denied`
   (`w1-grid.mjs:164-179`) — captured GET `/invoices/9` → 403
   `grid_load_forbidden`, no row leak, UI denial shown.
2. Document denial rendering over live HTTP through the real
   mount/dispatch/render stack with captures: `action-document.test.ts:211-277`
   — denied leaf serves a full `<!doctype html>` page under truthful
   404 (`<main>denied</main>`), canonical URL builder round-trips.
3. In-browser observation of truthful denial documents:
   `page-denied` (`w1-grid.mjs:181-191`) — navigated pages return 403
   with denied document text; `unknown-path-404` proves truthful 404.

The uncovered residue is only the composition (a real browser
observing a document-mode action denial through a paired build),
which adds no new decision logic beyond the three proven layers.
Accepted as a valid test-scope difference with this justification;
revisit if a document-mode GET action is ever declared (trip: any
`body html` + `get` action in a shipped contract).

## F9/F10/F13/F18 resolutions (individual, evidence-backed)

- F9 CLOSED (already documented): A07 handoff pins `step:<0-based
  iteration>` for loop iterations (`a07-report.md:124-128`); F06 pins
  the 1-based contiguous claim-order index for batch claims
  (`f06-report.md:74-77`). Different domains, both documented,
  occurrence linkage both sides. No owner action needed.
- F10 ACCEPTED (documented limitation): scan retained bytes are
  recorded-not-gated because JSC conservative-stack scanning is
  nondeterministic (probed: identical runs collect fully/partially/not
  at all); loop flatness is proven by scalar-machine legs sharing the
  exact loop shape (~0 heap at 100k/1M) (`a07-report.md:47-53`).
- F13 ACCEPTED (documented design): the webhook Can side is
  SQLite-only by F05 design, so the mkdb PG/MySQL path cannot apply;
  isolation holds via a fresh disposable SQLite file per run, never a
  shared table (`f06-report.md:92-98`). The isolation property — the
  actual requirement — is met.
- F18 ACCEPTED (scope confirmed): neither W6.1 nor W6.2 names
  ledger-under-budget evidence (task-list:888-889); F04's scope is the
  backend across dialects over native Bun.SQL, with no E/runtime files
  touched (`f04-report.md:58-60`). If a future leg needs
  ledger-under-budget, that is new scope, not a gap in W6.

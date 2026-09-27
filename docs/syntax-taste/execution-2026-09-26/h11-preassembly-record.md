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

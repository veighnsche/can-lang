# P09 readiness record — READY (after finding resolution)

Independent audit: [p09-audit-report.md](p09-audit-report.md) (verdict at
delivery: NOT READY, 9 blockers + 9 majors + 6 minors, all small recorded
decisions). Coordinator scenario walk:
[p09-scenario-walk.md](p09-scenario-walk.md). This file records every
resolution + recheck (P09.3) and the P09.4 owner/destination verification.

## P09.1 recheck — coverage, choices, evidence, contracts, boundaries

- Coverage R01–R16: PASS (audit) — unchanged.
- User choices: PASS after B1 — Q2's advisory vehicle is now specified
  (check-pipeline warnings + CLI/LSP surfacing, exit 0, builds unaffected;
  `canlc lint` stays retired — verified at `compiler/main.go:216`).
- Evidence quality: PASS (audit) — W2 conditioning (M1) and branch
  conditioning (B7/B8) close the partial-conditioning notes.
- Contract consistency: PASS after B2 (C-G → Lane F + F→H handoff), B5
  (bulk → Lane A), B9 (R16-03 rewritten to the authenticated envelope),
  M2 (row F inputs C-A/C-E).
- Lane boundaries: PASS after B6 (every multi-lane file owned with
  handoffs), B9 (R16-01→C, R16-03→F, R16-02/R16-04→H), m2 (single-lane
  owners listed).

## P09.2 recheck — scenario walks

| Scenario | Before | After |
|---|---|---|
| 1 second-app propagation | pass (m6 noted) | pass — re-migration duty stated in P08.3 |
| 2 S3 cancel-while-replacing | BREAKS under O2 | pass — W5 legs conditioned per branch; O2 removes `cancel_upload` for explicit `discard_upload` |
| 3 stalled SQL + SIGTERM | BREAKS | pass — shared-budget composition decided; W5 legs conditioned on X-R04-1/X-R04-3 with a specified cancel-absent story |
| 4 old-browser rollout | BREAKS | pass — fail-closed generation handshake decided in C-H; H qualifies against it |
| 5 companion crash recovery | BREAKS | pass — B3 claim/lease, at-least-once + idempotent ack, poison/dead-letter, restart responsibility all specified |
| 6 100k-step fault | pass w/ minors | pass — M8 step-index shape + M9 exact proof predicate + negative fixtures specified |

Disposal paths (coordinator walk): no post-disposal use path in the
selected contracts — unchanged, consistent with the audit.

## P09.3 resolution index

- B1 lint vehicle → contracts R08/Q2 + reconciliation C-I.
- B2 C-G owner → reconciliation C-G + lanes rows F/H + contracts R14 (M7).
- B3 companion claims/guarantees → contracts R11.
- B4 interop policy → reconciliation C-H + contracts R13 + acceptance W6.
- B5 bulk lane → lanes P08.1 row A + P08.3 (`runtime/collections/*`).
- B6 shared files → lanes P08.3 (full owner + handoff table).
- B7 budget composition → contracts R04 + acceptance W5.
- B8 S3 branches → contracts R15 + acceptance W5 (+ m4 orphans).
- B9 R16 contradiction → dispositions R16 + contracts R11/R16 + lanes C/F/H.
- M1 W2 conditioning → acceptance W2 + dispositions F-R01-03.
- M2 row-F inputs → lanes P08.1 row F.
- M3 Firefox provisioning → lanes row C + P08.5 register (verified: Firefox
  accepted by scripts, absent from the gate5 matrix).
- M4 provisioning register → lanes P08.5 (H-owned, gates W5/W6 legs).
- M5 rich-client boundary → contracts R16 + dispositions F-R02-06.
- M6 migration recipe → acceptance W6.
- M7 budget identity → reconciliation C-G + lanes F→H + contracts R14.
- M8 step index → contracts R05 + acceptance W4.
- M9 proof predicate → contracts R05.
- m1 split F-R13-01 → dispositions (F-R13-01a/b).
- m2 single-lane owners → lanes P08.3.
- m3 X-R11-1 superseded → alternatives README.
- m4 O2 orphans → acceptance W5.
- m5 batch mirror → acceptance W4.
- m6 re-migration duty → lanes P08.3.

## P09.4 owner/destination verification (final)

W1→C · W2→D · W3→B · W4→A (lowering; batch mirror→F) · W5→E ·
W6→F (PG/MySQL/companion) + H (UP25/AI/story) · Q1–Q3→A (Q2 warnings ride
the pipeline) · LSP→G · R16-01→C · R16-03→F · R16-02/R16-04→H ·
bulk→A · C-G→F · R03/R04 shared files per B6 table · Firefox→C ·
provisioning→H register. No missing destinations remain.
Qualification blockers (each with an explicit provisioning prerequisite):
S3 bucket + creds; live PG/MySQL + creds; x86 access + UP25 windows;
AI creds + spend caps; Firefox runners. Unavailable live evidence is not
a pass.

## Verdict

READY to write the implementation task list. Every choice that could
materially change architecture, syntax, public contracts, or task ordering
is settled or explicitly excluded with conditions. No design decision hides
inside a future implementation task; conditioned branches (X-R04-1, X-R04-3,
X-R15-1, X-R15-3, X-R10-1, X-R01-1, X-R06-1, X-R07-1, X-R02-1, Q4–Q6 gates)
carry both branches as first-class acceptance.

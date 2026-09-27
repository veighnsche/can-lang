# IC1 shared-interface integration record (H10) — 2026-09-26

Verdict: **IC1 GREEN** at join revision main `932924f7` (first validated at
`7c2dbc2a`, rebased-forward onto 9 C06/tempcache main commits, full battery
re-run green).
Owner slices integrated: A01–A05 (+A06 inactive), B01–B02 (+B03/B04 inactive),
C02–C04 (+C05 inactive), D02, E04–E06, E08, F01, F03, F05, G05 (via G01–G04),
H06–H07. Supporting (prereqs-of-prereqs): E01–E03, E07, F02.

## Slices joined (owner → consumer)

- C-A: A04 proof predicate + A05 bulk ops → A07 W4 legs (sealed), F06 batch
  mirror, G04 keyword reconciliation (no new surface; Q6 did not trigger).
- C-B: B01/B02 conventions + X-R06-1/X-R07-1 verdicts → all Can-authoring
  lanes; B05 W3 verdict consumed the joined conventions (green rerun).
- C-C: E-policy/vocabulary/hedge/redaction/S3 branches → F protocol
  (unknown-write + `lease_lost`/`replay`/`stale`), F01 ledger context, E09 W5
  legs. Authored caller-operand threading stays C06 downstream (explicitly out
  of E04 per its ledger); native meanings are qualified.
- C-D: C02 native slice (additive-only; C04 stands as recorded in
  `c02-report.md:64-66`) + C04 library/second app → D01/D02 tier work, G04
  queries, C06. X-R02-1 error-shape table handed to E09.
- C-E: A03 grammar/emission + C03 checker + E03 runtime → C04 captured reads,
  F05 endpoint status honesty, H06 deploy contract.
- C-F: F02 probe + F03 contracts → F04 backend qualification, F05 claim/lease
  transactions (lookup-first, no locking-read dependency).
- C-G: D01/D02 tiers + review manifest + conformance → D03 Vendor B (unchanged
  D02 client); F01 policy/ledger → H07 guard, F05 companion auth/destination
  policy; H07 guard → H08 (blocked on live access, IC2 matter).
- C-H: H06 paired-deploy contract + pins → C06 (E/C slices in progress),
  H10 (this record), H12.
- C-I: A01/A02 surfaces + G01→G05 sequence → B/C authoring (dogfood),
  rename/`with`-pin propagation proven by the G05 workflow leg.
- Supplementals: R09 local-bindings (1A) in G03/G05; R14 resolution in
  F01+H07+E (fail-closed, no live claim).

## Conditional gate: all triggered interfaces republished first

13/13 conditioned resolutions published before the join (11 experiment
branches + D01 tier + RETURNING mapping; 4 correctly inactive with trip
records). Full table: `docs/syntax-taste/evidence/2026-09-26/h10/h10-manifest.md`.
One non-blocking doc-sync note (H10-NOTE-01, E-owned): the C-C spec prose
still points S3/stream at "conditioned … not wired" while every normative
artifact carries the E07/E08 verdicts; the joined resolution is republished
in the manifest.

## Validation battery: 5/5 PASS

Evidence: `docs/syntax-taste/evidence/2026-09-26/h10/h10-battery.md`.

- Edit propagation: B-harness 11/11 (H10 rerun), grid-controls assert 128/128
  (H10 rerun; C06 +4 legs), G05 rename legs 4/4 (H10 rerun), C04 negatives +
  app asserts sealed.
- Status honesty: companion protocol 9/9 (H10 rerun), F05 183-assert seal,
  C-E exclusions both layers, E04 74/74 seal, host conformance 51/51 (H10 rerun).
- Failure vocabulary/provenance: CAN-CHECK codes, unknown-write chain
  (policy→budget-contract→carrier protocol), redaction 17/17 + bulk 8/8
  (H10 rerun), step-index emission, exact discard emits union.
- Wire/generation identity: C-H handshake 2/2 (H10 rerun), carrier v1 +
  chart-protocol pins consistent, catalogue SHA match, browser-closure
  witness green.
- Catalogue/runtime generated consistency: `catalogue-check` exit 0,
  catalogue suite ok, `check:runtime` fully green (lint+format+tsc).

## Follow-ups (none block IC1; routed, not silently dropped)

- H10-NOTE-01 → E (spec prose sync), H10-NOTE-02 → D (admission-test path
  list; contract assertion verified holding by H10 sweep), evidence-index
  staleness → coordinator. Details in `h10-battery.md`.
- Live-debt carried into IC2 (unchanged, not IC1 scope): D02-LIVE-1..4,
  D03-LIVE-1..4, E09 L3/S5b BLOCKED-by-design, H08 W6-AI blocked, C06 W1 legs.

## Handoff

IC1 release-candidate inputs to H12: the pinned interface versions in
`h10-manifest.md` (catalogue sha `87c05b44…`, carrier v1, chart `d02.chart/1`,
generation schema v1, policy/ledger/guard publications). Dependent lane
completion claims unblocked. No contract returned to preparation.

## Coordinator verification addendum (main, post-integration)

The coordinator independently re-ran the H10 battery on main after
integrating both H10 slices: C-B harness 11/11
(`CAN_BUN_ARCHIVE` bundle, 38s), C-D `canlc assert
shared/grid-controls` 128/128 + top-level pass via a fresh
`make bundle` dev build (bundle removed after), G05 rename 4/4,
C-H handshake 2/2, companion protocol 9/9, host conformance bun
51/51 + go 9 pass / 2 live-debt skips, redaction+bulk 25/25,
H07+F01 file set 75/75, `catalogue-check` exit 0, catalogue suite
ok, `check:runtime` green. All counts match the worker record.

H10-NOTE-02 is RESOLVED by the coordinator (no D worker active; the
stale path came from concurrent foreign-lane refactor `af0f42a2`,
which deleted `internal/scan/` outright with no replacement tree):
`TestDeliverablesUnreferenced` fatal reproduced on main, the
contract re-verified holding by an independent marker sweep (0 hits
in `runtime`, `tools/runtime`, `compiler`, `examples`,
`distribution`), the deleted path dropped from the walk list with an
explanatory comment, and the focused rerun green
(`TestDeliverablesUnreferenced` + `TestReviewManifestCoversDeliverables`
PASS). H10-NOTE-01 (E spec-prose sync) remains E-routed and
non-blocking; IC1 verdict unchanged: GREEN.

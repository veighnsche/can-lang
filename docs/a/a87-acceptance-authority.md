# a87 — Acceptance authority, A-light (draft, pre-decision)

Status: shipped (ratified). Answers the open decision from
`a85-agent-language-research.md` (removed) §4.1 at A-light strength: provenance
labels plus loud weakening reports. Rejection (A-strong) remains
deferred. Slices landed: parser marker, baseline recording, CAN6017
warning, committed tests, REQUIREMENTS amendment.

## Problem

The same agent authors the implementation and its decision table, so it
can change an expected result to match a wrong implementation and present
consistency as correctness. Measured direction: tests generated with
access to faulty implementations detected faults at 14%, versus 25% for
independent generation (a85 R08). The Goal's weak-table warning is prose;
nothing mechanical distinguishes acceptance criteria from proposed
evidence.

## Decision

Colocation stays; authority splits. One artifact, two row kinds:

- **pinned** — trusted acceptance. Weakening one is a reviewable act and
  is reported loudly.
- **proposed** — everything else (the default). Free churn, no drift
  tracking.

Pinning is a source marker plus a baseline record. The compiler never
rejects weakening under A-light; it reports it where it cannot be missed.

## Syntax (ratified, one canonical form)

Row suffix marker:

```
three(by = 3) => Ok(total = 3) pinned
```

Rationale: reads as assertion strength on the row; a prefix would collide
visually with the test name. `pinned` has no language-level collision
today (only Go locals of that name). New rows without the marker are
proposed. Removing the marker demotes the row to proposed — itself a
visible source diff, and the baseline record from the last accepted
baseline still reports the demotion (see below).

Table-level pinning (`tests pinned`) is refused: authority is per-row or
it is theater — a table with one strong row and five weak ones must not
borrow the strong row's status.

## Semantics (ratified)

- The accepted baseline records each pinned row as
  `test-name → canonical expectation rendering`, alongside the existing
  revision fingerprints (a77/a79). Rendering reuses the `normalize`
  canonical form, so formatting churn is not drift.
- After the ordinary gate passes — and only then (a79 ordering: no drift
  noise atop real errors) — the compiler compares pinned rows against
  the accepted baseline. Changed expectation, deleted row, or demoted
  row (marker removed while the baseline still pins it) each produce
  the weakening diagnostic.
- Regenerating a baseline never pins: a generated file has
  `Accepted false`, and rows without the marker are recorded as
  proposed regardless of history. Generation is not acceptance (a79);
  pinning requires the marker in source plus an accepted baseline that
  a human committed.

## Diagnostics (ratified)

- New code CAN6017, registered in `compiler/code.go` with an
  `explain.go` entry: weakened / removed / demoted rows, carrying test
  name, baseline rendering, and current rendering (Expected/Found/Hint
  payload shape, a71).
- Severity warning, always emitted, never silenced by flags. Exit code
  unchanged: A-light advises, it does not gate. (Gating is the A-strong
  decision, deferred.)

## Acceptance act (no new infrastructure)

Adding `pinned` is a source edit by the task owner; accepting is
committing a baseline with `Accepted true` through the existing review
flow (a79 baseline, a84 review base). No identity of the pinner is
recorded under A-light — the commit that accepts the baseline is the
attribution. Pinner identity belongs to A-strong.

## Interplay (non-goals held)

- Pinned rows still execute as tests; coverage (a03), exhaustiveness,
  and the contracts verifier treat pinned and proposed rows identically.
  Authority is about weakening, not about execution.
- Revision identity (CAN6013) is untouched: same-revision drift still
  rejects; pinned-row drift warns, including across revisions.
- Contracts, error declarations, and `given` scripts are out of scope:
  rows only. Pinning a contract is a later slice with its own draft.

## Verification plan (falsifiable)

1. Seed weakening of a pinned expectation → warning present naming the
   row with both renderings; exit code 0; emit still produced.
2. Same churn on an unpinned row → silent.
3. Delete a pinned row, remove a marker → warning in both cases.
4. Regenerate baseline without markers → rows recorded proposed; no
   silent pinning (negative control).
5. Full gates unchanged: `go test ./...`, modcheck, gramcheck, tsc.

## On ratification (amendment text, now live in REQUIREMENTS.md)

Goal gains one sentence (tagged a87): tests are evidence, pinned rows
are acceptance; the compiler reports weakening of the latter. R7/R8 gain
the row-kind definition and the warning behavior. Nothing else in
REQUIREMENTS.md moves.

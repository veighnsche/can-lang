# a91: `given` omission amendment — design (decided)

Status: shipped 2026-09-18. S0 rulings recorded below;
S1 (rule + tools) and S2 (migration) both landed same-day:
`CAN3105` retired, `CAN3111` bans `-`, 191 dash rows deleted,
full gates green.
Parent: `REQUIREMENTS.md` R8 (frozen: "Omission = error"),
a12 (producer-owned contracts), a89/S1a (the motivating
pain: `std/quota/quota.can` carries ~150 `=> -` entries
for 10 real exchanges).

## S0 rulings

- **Omission wins.** R8 replacement wording approved as
  drafted (see Amendment plan). Tables are partial; a
  missing entry claims non-reach; reaching it fails the
  test at execution.
- **`dangling-test.can` retired** (deleted), not
  repurposed. `failing-test.can` already covers
  execution failures; the static-error demo has no
  subject left.
- **`-` dies by hard cut.** Retired spelling is a new
  `CAN3111` error from day one; all rows migrate
  in-slice. No deprecation process invented.
- **Unknown keys stay errors** (status quo ante kept:
  modcheck errors, check.go `CAN3104` warns).
- **Editor grammar untouched**: the `unreachable` scope
  stays (gramcheck pins it); it simply never matches
  legal code anymore.

## Goal

Decide whether call-site `given` tables stay total or
allow omission, and if omission wins, shape the versioned
R8 amendment with its tool, fixture, and migration plan.
This doc authorizes no code; S1a/S1b ship under totality
regardless.

## Problem

R8 demands every test appear in every reachable table,
with unreachable tests written `-` explicitly. At
auth.can scale (11 rows, 2 sites) that is discipline. At
S1a scale (26 reaching rows × 6 sites) it is ~150 lines
of `-` burying 10 exchanges — and every future row in
the reaching set must be added to all 6 blocks. The
noise defeats the purpose: nobody reviews 150 `-` lines,
so the "explicit non-reach claim" is explicit to no one.

## Key fact: `-` and omission are already identical at runtime

Both mean "no entry." If the row reaches the call,
evaluation fails with "no script for call" either way
(`evScriptOutcome`, `compiler/eval.go`). Totality buys
only static forcing (a `CAN3105` cross-check error
instead of a test-execution failure) plus keystroke
feedback. And the gap is narrower than it looks:

- canlc has no test-skip flag (usage is
  `--format/--baseline/--out` only), so tables execute
  on every compile: a forgotten script fails the same
  command either way, differing only in diagnostic shape.
- R10 already has the editor running tables per
  keystroke, so the LSP latency argument for a static
  cross-check is thin even before any change.
- a18 honesty checks, arg matching, and typo rejection
  are unaffected by totality either way.

## Recommendation: omission means non-reach (RECOMMENDED)

- A test with no entry is implicitly `-`. Reaching it at
  runtime stays a hard failure ("no script for call").
- Unknown keys (no such test in the file) stay rejected:
  typo protection is the part worth keeping, and
  `dead-script.can` keeps its meaning.
- `-` is banned as redundant: two spellings for one
  thing is rejected, so omission wins completely and
  every `-` migrates away (counts below).

## Rejected middle grounds

- **Wildcard rows** (`other => -`): theater. Non-reach
  cannot be checked statically (path-sensitivity is
  undecidable), so a wildcard claims exactly what
  omission claims, plus one line.
- **Split by callee kind** (total for externs, omission
  for foreign-can): complexity for a distinction the
  runtime backstop does not respect. a18 already
  separates the trust levels (verified Ok claims vs
  trusted error stubs); totality adds nothing per kind.
- **Keep totality**: the honest status quo. Coherent,
  verdict-compatible, and increasingly hostile to every
  multi-site module after quota. Revisit trigger: if
  this proposal is declined, the boilerplate stays and
  future slices budget for it.

## Amendment plan (if approved)

- **R8 rewrite**: "Tables are partial: every reaching
  test SHOULD appear" becomes "a test with no entry
  claims non-reach; reaching it fails the test. Unknown
  keys are errors." Exact replacement wording closes at
  S0 review; the v0.1 freeze requires a tagged
  amendment, never a silent rewrite.
- **Retire `CAN3105`** (`CodeDanglingTest`): diagnostic,
  `explain.go` text, LSP tests. `checkGiven` keeps the
  missing-table, unknown-key, bad-stub, and
  deterministic-call checks; `reachingTests` is deleted
  or repurposed (no remaining caller).
- **Fixture dispositions**: `dangling-test.can` loses
  its subject — either retired or repurposed as an
  execution-failure demo (decide at S0; do not silently
  keep a demo whose rule is gone). `dead-script.can`
  stays green unchanged (unknown keys still rejected).
  `missing-exchange.can` and siblings: audit each for
  totality dependence.
- **modcheck**: the S1a reaching-scope check collapses
  to unknown-keys-only. Its three regression tests get
  rewritten for the new rule, not deleted.
- **Migration**: delete every `=> -` row (quota ~150,
  auth-login 16, retry-loop, fixtures) and re-green all
  gates. Mechanical; one commit.

## Slices

- **S0 (this doc): design review.** Acceptance: omission
  vs status quo ruled, R8 replacement wording fixed,
  fixture dispositions closed, `-` ban confirmed. No code.
- **S1: rule + tools.** R8 amendment text, `CAN3105`
  retirement, check.go + modcheck + explain + LSP
  updates with their tests. Acceptance: full gates
  green with totality suites rewritten, not skipped.
- **S2: migration.** All `-` rows deleted corpus-wide;
  demos re-pinned; `go test ./...`, modcheck, gramcheck
  green.

## Open questions — closed at S0

1. R8 wording: approved as drafted (see S0 rulings).
2. `dangling-test.can`: retire (deleted).
3. `-` ban: hard cut, `CAN3111`, in-slice migration.

## Non-goals

No grammar change (omission removes syntax, adds
none); no a18 change (honesty checks are orthogonal);
no S1a/S1b rework (they ship first under totality).

## Rollback

This doc authorizes no code, so there is nothing to roll
back. If S0 declines, R8 stands and quota.can's `-`
rows stay as written.

## Follow-up: stale-script lint (shipped same day)

`CAN3420` (`CodeLintUnreachedKey`): flags given keys naming
real tests with no static path to the call — the advisory
half omission enabled. Sound (no static path means truly
unreachable; every call is static) and silent on all blessed
code. Proving sketch: `docs/archive/sketches/lint-errors/stale.can`
(11th gallery file, findings + spans pinned).

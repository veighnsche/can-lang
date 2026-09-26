# X-R08-1/2/3: authoring-policy repair comparisons

Task A08. Source: R08; X-R08-1/2/3; P02.2. Prerequisites A01
(`19e45019`), A02 (`1365d0a6`) — both on main and rerun green at
the base below. Registration: `registry.json` (frozen 2026-09-26
before any attempt). This record compares repair cost under the
shipped Q1–Q3 surface against the pre-change idioms. It does not
reopen Q1–Q3: the user decisions stand; the measurements below are
cost information for the H14 supported story and the H11 IC2 gate.

Environment: darwin-arm64, go1.27.1, Bun 1.4.2 via an isolated
development bundle (distbuild version `a08-verify`, built from a
clean worktree at `e3a4148c`; the docs-only A08 delta does not
affect bundle inputs). Shipped surface executed at
`e3a4148cf12fb5a67a9a5ae6369ad6e5f0123cb2`; pre-change checker
behavior executed at `ba619613` (`19e45019^`) in a scratch
worktree (removed after; commands in the appendix).

## Method

Each case registers creation, refactor and repair prompts per the
[evaluation protocol](../../docs/syntax-taste/preparation/evaluation-protocol.md),
with sealed held-out variants (rename/data maps only — no
held-out program is authored anywhere). The repair leg is the
measured one; creation/refactor legs are registered for future
trials. Two kinds of evidence:

- Deterministic legs (executed here): `canlc assert` exit
  codes, verbatim stderr diagnostics, `canlc format` rewrites and
  fixpoints, and hand-edit counts between baseline and reference.
  These are authoring-mechanics facts, not agent outcomes.
- Agent legs (unrun): 0 attempts on every case — no model access
  in this environment. Whole-task tokens are unmeasured and no
  superiority is claimed in either direction. Candidate slots in
  `candidates/` stay empty under the frozen `frozen-x-r08`
  settings (primary gpt-6-sol medium, efficient gpt-6-luna
  medium, escalation gpt-6-astra high, five attempts).

Baselines are byte-frozen task inputs; references are owner
repairs proving the hidden checks pass. References are never
trial prompts.

## X-R08-1: Boolean arm repair

Question: what does an order-violation repair cost once both arm
orders check and the formatter canonicalizes, versus the old
false-first enforcement?

Old surface (executed at `ba619613`): the baseline fails the
check with `ordinary Boolean match requires false before true`.
Repair = one manual reorder edit by an author who knows the
canonical order; the pre-Q1 formatter offers no rewrite
(diff-verified in `19e45019`, not executed).

New surface (executed at `e3a4148c`): the baseline asserts green
(exit 0, silent stderr). Repair = zero hand edits plus one
`canlc format` run: the reference is byte-identical to formatter
output (`REF-EQ-FORMAT`) and fixpoint-stable (`REF-FIXPOINT`).
The duplicate-arm mutant still fails with `match arm is fully
covered by earlier arms` — coverage diagnostics are unchanged
(neutral leg, same under both surfaces).

Neutral/adverse: check-green no longer implies canonical order.
Under the old rule every green program was false-first by
construction; now uniformity depends on formatter discipline —
an agent that never runs `canlc format` leaves true-first arms
in the tree, and a reviewer cannot tell canonical order from
check output alone. The formatter rewrite is also diff the
author did not write (comment-preserving per
`TestBooleanArmCanonicalizationWithComments`, but still
unwritten diff).

## X-R08-2: local-warning repair

Question: what does the C8-shape repair cost as an advisory
warning versus the old hard error?

Old surface (executed at `ba619613`): the baseline fails the
check with `unnecessary local total at byte 160; replace with
ok left + right`. Repair is forced before the program builds —
and the error spells the exact replacement.

New surface (executed at `e3a4148c`): the baseline asserts
green (exit 0) with one stderr line:
`src/main.can:11:5: CAN-CHECK-UNNECESSARY-LOCAL: accidental
alias total — consider inlining`. Repair = one optional edit
(delete the binding, inline the expression); the reference
asserts green with silent stderr.

Neutral/adverse: two genuine costs. First, exit-0-plus-stderr
means exit-code-gated automation silently accumulates warnings
— nothing forces the repair, so repair rate may drop versus the
forcing error. Second, the warning drops the replacement text:
the old error told the agent exactly what to write (`replace
with ok left + right`); the new message says `consider
inlining` without the rewrite. The advisory gain (builds stay
green) trades against weaker repair guidance.

## X-R08-3: near-binding repair

Question: what does a silent-capture repair cost with explicit
`with` pinning versus pure name lookup?

Old surface (still live as the unlisted-input fallback,
executed at `e3a4148c`): the baseline asserts exit 1 with a
bare `outcome mismatch` on `compute:first` and empty stderr —
the 99/99 shadows redirect the capture with no diagnostic
pointing at it. Repair under the old rule = edit the shadowing
scope (here: two literal edits, 99→3 and 99→5, or a rename plus
rebind), with no capture-site fix available.

New surface (executed at `e3a4148c`): repair = one line edit at
the creation site (`callable combine with prefix = 3, suffix =
5`); the reference asserts green (exit 0) and is byte-stable
under `canlc format`. `combine` stays byte-identical either
way. The typo mutant fails with `expression at byte 394:
unknown with binding prefixx of can.project.root/app::combine`
(structured code CAN-CHECK-CAPTURE); unknown, duplicate,
non-near and mistyped bindings are all located refusals per the
rerun-green `TestAUQ3CoreBindingRefusals`.

Neutral/adverse: pinning is per-site authoring cost that never
goes away (here +1 line per creation site); unlisted inputs
still resolve by name lookup, so partial pinning leaves the
residual shadow hazard in place; direct-call argument supply is
unchanged (positional), so the pinning idiom does not transfer
there; and CAN-CHECK-CAPTURE is a new diagnostic class agents
must learn. Neutral: programs without `with` behave exactly as
before (fallback unchanged).

## Comparison table

Deterministic repair mechanics per registered repair task.
Hand edits count baseline→reference source edits; tool runs
count required CLI invocations. Agent attempts/retries/tokens
are unmeasured on all three (0 attempts).

| | Old idiom | Shipped surface |
|---|---|---|
| Boolean repair | 1 manual reorder edit; error names the rule; no formatter help | 0 hand edits + 1 `format` run; reference = formatter output, fixpoint-stable |
| Boolean failure signal | check error, exit ≠ 0 | none at check (green); canonical order needs a formatter run |
| Local repair | forced 1 edit; error spells the replacement | optional 1 edit; warning, exit 0 |
| Local failure signal | check error, exit ≠ 0 | stderr warning only; exit-code gates miss it |
| Near repair | 2 scope edits (revalue/rename shadows); silent `outcome mismatch` | 1 creation-site clause edit; typo-class located errors |
| Near failure signal | bare `outcome mismatch`, empty stderr | same when unpinned; located CAN-CHECK-CAPTURE for bad pins |

No row supports an agent-superiority claim: these are edit and
diagnostic facts. The measured agent repair cost per policy —
the actual X-R08 question — awaits trials under `frozen-x-r08`.

## Limitations

- Zero agent attempts ran (no model access); prompts,
  diagnostics and retries exist only as registered or
  deterministic artifacts. Tokens/model/tokenizer unmeasured.
- Old-surface evidence is checker-level execution at
  `ba619613` (scratch `programFixture` probes) plus one
  diff-verified claim (pre-Q1 formatter had no
  canonicalization); no old-toolchain `assert` bundle was built.
- Repair tasks are minimal single-shape programs; multi-shape
  files, warning-plus-error mixes and larger scopes are not
  covered. Held-out variants are registered, never executed.
- Reference repairs are owner-authored; they validate the
  hidden checks but predict no agent behavior.

## Handoff (H14 supported story, H11 IC2)

Handed to H14: this record + `registry.json` (prompts, hidden
checks, frozen settings) + baselines/references/candidate
slots. Attempts: 0 per case. Prompts: as registered (repair
prompts quote the baseline sources; held-out names/data never
appear). Diagnostics: verbatim above. Retries: none (no
trials). Tokens/model/tokenizer: unmeasured. Limitations: as
listed. Residual for H14: the supported story must state
formatter discipline for Boolean order (check no longer
enforces it), the advisory nature of the C8 warning (exit 0,
stderr-only, no replacement text), and the partial-pinning
residual for near captures — and must claim no measured agent
advantage from this comparison. For H11: the A08 authoring
evidence is the deterministic legs only; IC2 must not treat
unrun agent legs as coverage.

## Provenance appendix

- A01/A02 acceptance rerun at `e3a4148c`: `go test
  ./compiler/internal/check/ -run 'TestAUQ1|TestAUQ2|TestAUQ3'`
  ok (3.8s); `go test ./compiler/internal/syntax/ -run
  'TestBooleanArm|TestWith'` ok.
- Bundle: `go run ./tools/distbuild --archive
  /private/tmp/bun-darwin-aarch64.zip --out /tmp/a08dist
  --version a08-verify` → isolated `bin/canlc` (kept outside
  the tree; removed scratch dirs after).
- Fixture legs: each `tests/authoring-policies/<case>/
  <variant>/` copied to a scratch dir, then `canlc assert`
  (exits: boolean 0/0, locals 0/0 + warning on baseline,
  near 1/0) and `canlc format` (reference = format output,
  fixpoint, byte-stability).
- Old surface: `git worktree add --detach /tmp/a08old
  19e45019^`, scratch `programFixture` test on the exact
  baseline sources (boolean → `ordinary Boolean match
  requires false before true`; locals → `unnecessary local
  total at byte 160; replace with ok left + right`), worktree
  removed after. Mutants (`dup`, `typo`) executed through the
  bundle `canlc assert` for verbatim stderr.
- Registration correction (disclosed): the `near` hidden check
  `binding-typo-diagnosed` was reworded after freezing to match
  CLI-observable text (`unknown with binding <name> of
  <callee>`; CAN-CHECK-CAPTURE is the structured code, not
  printed by `assert`). Attempts were still 0 and no agent
  output had been observed; the check's intent is unchanged.

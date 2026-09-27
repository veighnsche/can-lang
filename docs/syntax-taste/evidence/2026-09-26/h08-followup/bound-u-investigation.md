# Bound-U investigation (H08 follow-up, task item 2)

27 September 2026. What a qualified complete-call bound requires,
what is measurable vs estimated today, and what engineering (not
credentials) unblocks qualification — with live evidence from
`probes/` and design advice from `jev/`.

## 1. What qualification requires

BLK-01: a qualified metering profile supplies a conservative upper
bound U for the complete encoded call's provider input plus output
consumption (instructions, schemas/questions, overhead, output
limits), pinned to one provider/model/version. Bytes and estimated
tokenizer counts are not proof. H08/X-R14-1 qualified nothing
because no provider-published enforced bound existed then.

This slice found the missing publication (TypeSafe Models page,
retrieved 2026-09-27): for `jev-1.13.0`, context 64k
tokens/request with 32k for state plus the longest question, and
price $42/Btok input-only with free output. Live probes (m1–m9)
then verified the enforcement half of derivation 1 for the first
time: over-budget requests fail pre-inference with HTTP 400
`max_tokens_exceeded`, typed, with zero usage and no model echo
(m5/m6/m7/m9). No truncation-and-charge, no silent overage.

## 2. Measurable vs estimated today

| Item | Status | Evidence |
| --- | --- | --- |
| Enforcement kind (reject, no consumption) | MEASURED | 4 rejects, all 400/typed/zero-usage |
| Cap location | BRACKETED (23546, ~42000] | m8 accept 23546 exact; m9 reject ~40k-est |
| Exact enforced figure (32000? 32768? other?) | ESTIMATED | rejects carry no usage; upper edge is rate-derived |
| Triage worst-case input (1446+54 = 1500) | MEASURED | m3 hostile max-caps state |
| Output per question shape (noul 21, choice-5 54) | MEASURED (n=2..3) | m1/m4, m2/m3/m8; stable across 50× input range |
| Answer-value determinism | REFUTED | m1/m4 same usage, noul 0.67 vs 0.69 |
| Output maximum (any ceiling) | UNPUBLISHED, unmeasured beyond n=3 | no provider text; sweep pending |
| Price ($0.042/MTok in, $0 out) | PUBLISHED (pinned this slice) | Models page + retrieval date |

Two limits are structural, not budgetary: (a) rejects report no
usage, so any cap bracket's upper edge is rate-derived and its
last ε is irreducible from outside — dense bracketing shrinks ε,
only provider publication of the exact enforced figure closes
it; (b) output has no published ceiling, so the output side can
only ever be measured-allowance plus breach backstop until the
provider publishes (Jev-advised `measured_allowance`, adopted as
advice: record the allowance, keep the verdict unqualified).

## 3. What engineering unblocks qualification

Done this slice: (i) price pin into the registration
(`conservative_single`: 4.2e-8 USD/token with source/date;
settled in+out billed at the input rate is a documented
safe-direction over-charge — the price leg now unblocks
mechanically); (ii) exact bound-evidence substrate in
`runtime/ai/bound.ts` with tests proving exact accounting
(per-measurement verify, sweep verify, U = in+out construction,
split-tariff quoting); (iii) the qualification protocol in the
model-change recipe (boundary-plus-sweep campaign, redacted
artifact rule, re-verification on provider change); (iv) the
candidate-U formulation below, recorded but explicitly
UNADOPTED.

Remaining for a qualified U (in order): (a) provider-published
exact enforced input figure (or program acceptance of a dense
bracket + stated rate assumption — a decision, not a
measurement); (b) provider-published output grounding, or
program acceptance of measured-allowance-with-margin under
quarantine backstop; (c) the gated boundary-plus-sweep campaign
itself (spend-cap env + price + candidate U through the live
gate). (a)–(b) need provider text or an explicit program
decision; (c) is mechanical once they land.

## 4. Candidate U (recorded, NOT qualified)

`U_candidate = 32768 (input) + 512 (output) = 33280` for
`typesafe/jev/1.13.0`: input takes the published 32k state
budget at its larger k-reading (covers 32000 and 32768);
output takes the measured choice-5 peak (54) with ~9× margin.
Triage worst-case measured total (1500) sits at 4.5% of this
candidate. Adoption awaits (a)–(c) above; `boundStatus` stays
`unqualified` and the guard keeps rejecting pre-send.

## Verdict

U is still UNQUALIFIED — precisely what remains is (a) the exact
enforced input figure and (b) output-side grounding, with the
input enforcement mechanism verified, the cap bracketed, the
price pinned, and every number above measured live. No blocker
is escalated: the next actions are provider text (or a recorded
program decision accepting bracket+margin) and then the
mechanical gated campaign.

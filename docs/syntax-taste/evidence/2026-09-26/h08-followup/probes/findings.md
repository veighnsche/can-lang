# Bound-U measurement probes: findings (m1–m9)

27 September 2026, pinned model `jev-1.13.0`, serial execution.
Builder: `probe.py` (deterministic; triage-shaped probes reuse the
registered instructions/categories verbatim; ticket text is
synthetic and unlabeled — no quality verdicts exist or are
claimed). Redacted per-call records: `records/m*.json`; response
bodies (answers+usage only): `responses/m*.json`. Usage ledger:
`../usage-ledger.md`.

## Outcomes

Accepted with exact usage: m1 275+21, m2 483+54, m3 1446+54,
m4 275+21 (byte-identical m1 repeat), m8 23546+54. Rejected with
HTTP 400 `{"detail": {"error_type": "max_tokens_exceeded"}}`,
zero usage, no model echo: m5 (~100k-token est), m6 (~58k est),
m7 (~64k est), m9 (~40k est).

## What this establishes

1. **Enforcement kind (verified).** Over-budget requests fail
   pre-inference with a typed rejection and no consumption — no
   truncation-and-charge, no silent overage. This is the fail-closed
   provider behavior derivation 1 of `h08-x-r14-1.md` needs; the
   remaining input-side gap is the exact enforced figure, not the
   mechanism.
2. **Cap location bracket (measured).** Accepts top out at exactly
   23546 input tokens (m8); rejects start at ~40k-est (m9; sizing
   from the measured 0.439 tok/char marginal rate at scale, ±~5%).
   The enforced cap sits in (23546, ~42000] — consistent with the
   published "32k tokens for state plus the longest question"
   binding before the 64k total for single-question calls. The 64k
   total is untested (state cap binds first with one question).
3. **Outside-verification limit (confirmed).** Rejects carry no
   usage, so a reject's true token count is unknowable from
   outside; only accepts yield exact points. The bracket's upper
   edge is therefore rate-derived and its last ε is irreducible
   without provider publication of the exact enforced figure (or
   the tokenizer). Dense bracketing shrinks ε; publication closes
   it. The recipe records this protocol with the stopping rule.
4. **Output stability (measured, not proven).** Choice-5 output is
   exactly 54 tokens across a 50× input range (m2/m3/m8: 483 →
   23546 in); noul output is exactly 21 twice (m1/m4). Output
   depends on question shape, not input size, for these shapes.
5. **Value nondeterminism (measured).** m1/m4 agree on usage
   (275/21) but differ on the noul value (0.67 vs 0.69). Fixed
   bytes fix token counts (n=2); they do not fix probabilities.
   Determinism claims cover counts only.
6. **Triage scale (measured).** Worst-case in-caps triage input
   (m3, hostile 200+2000 chars) settles at 1446+54 = 1500 total —
   under 5% of any candidate input cap. Triage can never approach
   the enforced boundary; any input cap at or above ~32k covers
   triage inputs with 20× headroom. (Headroom is comfort, not
   proof — the bound still needs the exact cap figure plus
   output grounding.)

## What this does not establish

No qualification claim: the exact enforced cap value and any
output-side grounding remain unpublished, so U stays unqualified
per the investigation (`../bound-u-investigation.md`). Answer
values on synthetic states (m2/m3/m8 all returned `billing` at
confidence 1.0) are anecdotes without labels, not accuracy data.

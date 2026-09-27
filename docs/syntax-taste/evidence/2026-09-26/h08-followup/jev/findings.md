# H08-followup Jev consultations: bound-U design decisions

27 September 2026. Advisory evidence for the H08 follow-up slice
(W6-AI bound-U investigation). Three fresh live requests to pinned
model `jev-1.13.0` cover the three hardest design decisions below.
Every explanatory state field, instruction, and option description
was rewritten in full across the three rounds (48 fields: 4 state
fields, 3 instructions, 9 option descriptions, each in 3 full
phrasings; zero shared 8-grams per `wording-audit.json`); model IDs,
prices, budgets, token sums, file paths, requirement IDs, timeouts,
caps, HTTP statuses, and code spellings stayed exact. No request
contains any prior Jev answer. Jev performed no research and
returned typed choices, distributions, and confidence values, not
prose reasons. Total consultation usage: **6208 input + 447 output
tokens** (~$0.00026 at the published input tariff; output free),
ledgered separately from measurement probes in
`../usage-ledger.md`.

## Why these three decisions

The BLK-01 proof standard plus the H08 seal fix *what is required*
(a conservative whole-call bound U; bytes and estimated tokenizer
counts excluded as proof) and the retrieved TypeSafe publications
fix *what is available* (64k-token/request input budget with
unpublished enforcement; no output maximum; input-only price
$0.042/MTok with free output). What remains genuinely open for
this slice is *how to act on that combination*: (1)
**output_treatment** — whether to measure an output allowance, and
what claim it may carry; (2) **verification_protocol** — which
live probes must gate any future U adoption; (3) **price_pin** —
how to carry the input-only tariff into the gate's single price
field. Each question carries at least one cheaper alternative
(deferral, edge-only or sweep-only probing, exact-split handoff,
no pin).

## Results

| Question | Round 1 | Round 2 | Round 3 |
| --- | --- | --- | --- |
| Output treatment | measured_allowance 0.64 | measured_allowance 0.96 | measured_allowance 0.43 |
| Verification protocol | boundary_plus_sweep 0.94 | boundary_plus_sweep 0.92 | boundary_only 0.40 |
| Price pin | conservative_single 0.92 | conservative_single 0.84 | conservative_single 0.99 |

Adopted (as advice, not proof): **measured_allowance**,
**boundary_plus_sweep**, **conservative_single**.

## Disagreement investigation

Round 3 is near-flat on two questions (output 0.43/0.41/0.16 at
confidence 0.14; protocol 0.40/0.36/0.24 at confidence 0.10),
while rounds 1–2 decide firmly. This pattern reads as genuine
option competition under the round-3 phrasing, not a confident
reversal: no losing option ever commands a confident round, and
the strongest round-3 challengers (defer_output 0.41,
boundary_only 0.40) argue cost minimization — do not spend probe
budget on a side that cannot qualify yet; settle enforcement with
the fewest calls. That objection is valid in general but dissolved
in particular here: the slice authorization explicitly budgets
roughly ten counted calls under $0.01, the parent requires the
residual gap quantified with measurements (which deferral cannot
supply), and the adopted protocol runs about six probes, so the
frugal alternatives save single-digit thousandths of a dollar
while leaving enforcement semantics assumed. The measurement
campaign therefore proceeds with the adopted options, the
verdict stays unqualified regardless of measured values, and the
deferral counter-case is retained visibly: if probe costs ever
stopped being negligible, defer_output would win on its own
terms.

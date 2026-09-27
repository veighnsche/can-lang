# H08-followup usage ledger — every live provider call

Endpoint for all calls: `POST https://api.typesafe.ai/v1/systemone`
(requested model `jev-1.13.0`; every 200 echoed `jev-1.13.0`).
Credential: `TYPESAFE_API_KEY` (presence only — the value appears
nowhere in this record). Tariff: published $42/Btok input
($0.042/MTok, 4.2e-8 USD/token), output free. No USD was metered
by any API response; the cost below is tariff × input tokens.

## Design consultations (Jev-the-classifier; advice, not measurement)

| Call | Timestamp (UTC) | Status | In | Out | Purpose |
| --- | --- | --- | --- | --- | --- |
| consult-1 | 2026-09-27T11:27:10 | 200 | 2202 | 150 | bound-U design round 1 (`jev/request-1.json`) |
| consult-2 | 2026-09-27T11:27:11 | 200 | 2037 | 150 | bound-U design round 2 (`jev/request-2.json`) |
| consult-3 | 2026-09-27T11:27:11 | 200 | 1969 | 147 | bound-U design round 3 (`jev/request-3.json`) |

Subtotal: 6208 in + 447 out (~$0.00026).

## Bound-U measurement probes (no design content; redacted records)

| Call | Timestamp (UTC) | Status | In | Out | Req bytes | Purpose |
| --- | --- | --- | --- | --- | --- | --- |
| m1 | 2026-09-27T11:31:46 | 200 | 275 | 21 | 143 | baseline: live path, model echo, usage decode |
| m2 | 2026-09-27T11:31:48 | 200 | 483 | 54 | 877 | triage-shaped small state: usage scale |
| m3 | 2026-09-27T11:31:50 | 200 | 1446 | 54 | 3411 | triage max-caps hostile state: worst in-caps input |
| m4 | 2026-09-27T11:31:53 | 200 | 275 | 21 | 143 | byte-identical m1 repeat: determinism |
| m5 | 2026-09-27T11:33:54 | 400 | 0 | 0 | 256066 | over-budget (~100k est): enforcement semantics |
| m6 | 2026-09-27T11:36:02 | 400 | 0 | 0 | 149060 | near-budget (~58k est): bracket from below |
| m7 | 2026-09-27T11:36:05 | 400 | 0 | 0 | 164884 | at-budget (~64k est): bracket from below |
| m8 | 2026-09-27T11:37:10 | 200 | 23546 | 54 | 63924 | mid-budget (~25k est): admission + rate calibration |
| m9 | 2026-09-27T11:37:36 | 400 | 0 | 0 | 110166 | mid-budget (~40k est): 32k sub-budget discriminator |

Subtotal settled: 26025 in + 204 out (~$0.00109). The four 400s
(`max_tokens_exceeded`, no usage, no model echo) consumed zero
tokens. Probe inputs were sized from measured token/char rates
(see `probes/findings.md`); sizing estimates are not proof inputs
— only observed usage and accept/reject outcomes are evidence.

## Totals

12 calls: **32233 input + 651 output tokens, ~$0.00135.**
Serial execution throughout (one call at a time). No other
providers, no open-ended runs, no secrets in any artifact.

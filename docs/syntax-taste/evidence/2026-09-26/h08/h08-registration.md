# H08 protocol registration — support-ticket-triage model-change eval (X-R14-1 / W6-AI)

Registered 2026-09-26 by H08 slice 1, **before any H08 live evaluation
result**. Frozen inputs live beside the harness at
`tools/runtime/ai-eval/protocol/`; this record pins their hashes. Any
edit to a frozen file after this point requires re-registration (new
hashes recorded here) and invalidates prior comparisons.

## Worked feature

Server-side support-ticket triage into five finite structured
categories via one SystemOne `choice` question per ticket. State is
`{subject, body}` text; caps (subject ≤ 200 chars, body ≤ 2000 chars)
are enforced by the harness, which refuses over-cap cases without
dispatch. No live provider result has been observed by H08.

## Pinned provider identity

- Provider `typesafe`, model `jev`, version `1.13.0`, protocol
  `typesafe_systemone_v1`, endpoint
  `https://api.typesafe.ai/v1/systemone` (H04 precedent; observed model
  `jev-1.13.0`).
- Timeout 30 000 ms, `maxBodyBytes` 65 536.
- Changing provider, model, or version invalidates bound qualification
  and starts a model-change comparison under the recipe in
  `h08-model-change-recipe.md`; it never silently reuses the old U.

## Frozen case sets

| Set | File | sha256 | n |
| --- | --- | --- | --- |
| Representative | `tools/runtime/ai-eval/protocol/representative-cases.json` | `e9cb714ee9210abce3c9f01864e353d8ef17958569b08cde33b71d1f895fcb4c` | 12 |
| Held-out | `tools/runtime/ai-eval/protocol/heldout-cases.json` | `a4e29ba1d4659878eabc3b5b205f8fd8164592f62ac55624673d4db8badc0b98` | 6 |

Labels (including the three documented edge calls rep-10, rep-11,
rep-12 and held-06) are part of the frozen files. The held-out set
carries no pass/fail threshold; it exists for model-change comparison.

## Measures and verdict rule

Per case: `correct | incorrect | abstain | invalid-output | rejected`.
Abstain iff answer confidence < 0.50 (registered floor, no tuning —
no results exist). Accuracy = correct / (correct + incorrect).

**Pass (representative only):** ≥10/12 correct, ≤1 abstain,
0 invalid-output, 0 rejected, every settled actual ≤ U, ledger
committed == Σ settled usage, every accounting report carries its
case correlation. Latency is reported per case plus run max; no SLO
is set or implied (P02.2). Cost is reported in tokens; USD is unknown
(no price table pinned).

## Execution rules

Strictly serial provider execution (one call at a time). The
`ai_budget::within` scope mints one correlation per case attempt
(each attempt also gets a unique invocation id); runs group by run
id in the report. Metered legs use the PG ledger (F04 default); file or
SQLite-file alternates where PG is unavailable; memory and
sqlite-memory backends are refused for metered runs. Live mode
additionally requires `TYPESAFE_API_KEY`, an explicit
`CAN_EVAL_SPEND_CAP_USD` cap, a pinned price table, and a qualified
bound U — all currently missing (see `h08-report.md`).

## Bound status at registration

UNQUALIFIED: no complete-call upper bound U exists for any profile.
The harness cannot dispatch a live budgeted call until one qualifies
(the H07 guard rejects unqualified profiles before send). The
qualification analysis is `h08-x-r14-1.md`.

## Amendments

- v2 (2026-09-26, still before any live result): correlation rule
  corrected — v1 assumed one run-wide correlation, but the H07
  `within` scope mints one correlation per attempt, which the
  selected contract permits ("correlation may group attempts"). Case
  files, labels, thresholds, and other measures unchanged.

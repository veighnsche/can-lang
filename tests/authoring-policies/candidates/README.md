# Candidate slots — X-R08-1/2/3 agent trials

One writable slot per registered case. No trial ran in this
environment (no model access): attempts = 0 on every case and
whole-task tokens are unmeasured.

- `boolean/` — repair output for `boolean-arm-repair` goes here.
- `locals/` — repair output for `local-warning-repair` goes here.
- `near/` — repair output for `near-binding-repair` goes here.

A future trial run must follow `registry.json` (`frozen-x-r08`:
primary gpt-6-sol medium, efficient gpt-6-luna medium, escalation
gpt-6-astra high, five attempts, fresh workspace each, pilot not
counted) and record per attempt: model identifier, date, effort,
prompt, tools, source revision, fixtures, diagnostics, retries,
and input/output/cached tokens with the tokenizer. Start each
attempt from a copy of the case `baseline/`; never show the
`reference/` repair or any held-out program in a prompt. Report
attempts, prompts, diagnostics, retries, tokens and limitations
to H14 per the A08 handoff.

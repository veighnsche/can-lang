# Jev advice on A02 probe durability

2 October 2026. Three new `jev-latest` requests were prepared before any
response was sent. Every state paragraph, the instruction, and each option
description was rewritten across all three versions while preserving facts,
constraints, and alternatives. The [wording audit](wording-audit.json) records
the 8 compared explanatory fields, pairwise wording differences, full-request
hashes, and 18 preserved technical identifiers (two identifier drifts found
during audit, `code-10` and `30-plus`, were standardized to `code 10` and
`30+` before sending). Requests [1](request-1.json), [2](request-2.json), and
[3](request-3.json), their exact [responses](response-1.json)
([2](response-2.json), [3](response-3.json)), transport metadata, and
[probability summary](summary.json) are saved. Each call returned HTTP 200
and model `jev-1.13.0`; total reported usage was 2,929 input and 132 output
tokens. No request contained another response. Jev did not inspect source or
run code; the supplied state summarized the worker's load measurements, the
coordinator's three failed reruns, the production-timing analysis, and the
F01 durability stakes.

The table shows selected options and probabilities, not measured design
correctness. Exact option meanings are in each request.

| Decision | Request 1 | Request 2 | Request 3 | Engineering selection |
| --- | --- | --- | --- | --- |
| A02 durability | patient .95 (accept .05) | patient .79 (accept .21) | patient .51 (accept .49) | **Patient timeouts.** See spread analysis below. |

Direction is unanimous (patient timeouts 3/3) but the margin collapses
across wordings (0.95 to 0.79 to 0.51), so the disagreement worth
investigating is the decay, not the winner. Comparing the texts: request 3
frames acceptance as "treat load-induced reds as environmental rather than
defects," which normalizes the quiet-window regime and buys nearly half the
mass; requests 1-2 frame the same regime as "fails under parallel-worker
load" and a "scheduling puzzle," which price the operating tax explicitly.
The stable signal across all framings is that a deadline which meters
toolchain boot (7.6 CPU-s per `canlc run` sender launch) rather than the
judged property (a real delivery in the mailbox) should not gate a durable
obligation — no wording disputes that the 5s clock measures the harness,
and the production property carries no timing term. The near-tie in
request 3 counsels humility about HOW MUCH patience: a minimal bump toward
the baseline's 30s phase budget, not a redesign.

Selection on repository facts: retime the mail phases toward 30s with
verdicts unchanged (real staged deliveries, mailbox-judged), then run one
fresh live verification. The retired .py files stay retired either way.
Because verification needs a quiet window and all worker slots are full,
the retime loop is queued after the running ports land and its live run
joins the serialized F01 rerun sequence; the A02 checklist row records
this. Agreement is treated as advice: the selection rests on the timing
analysis (the clock measures the harness, not production) and F01's
rerun requirement, not on the classifier vote. Per the user's standing
instruction, Jev's advice is followed without re-asking.

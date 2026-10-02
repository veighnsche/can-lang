# Jev advice on the blocked-leg retirement rule

2 October 2026. Three new `jev-latest` requests were prepared before any
response was sent. Every state paragraph, the instruction, and each option
description was rewritten across all three versions while preserving facts,
constraints, and alternatives. The [wording audit](wording-audit.json) records
the 9 compared explanatory fields, pairwise wording differences, full-request
hashes, and 22 preserved technical identifiers. Requests [1](request-1.json),
[2](request-2.json), and [3](request-3.json), their exact
[responses](response-1.json) ([2](response-2.json), [3](response-3.json)),
transport metadata, and [probability summary](summary.json) are saved. Each
call returned HTTP 200 and model `jev-1.13.0`; total reported usage was 3,595
input and 171 output tokens. No request contained another response. Jev did
not inspect source or run code; the supplied state summarized the
aspect-blocked versus check-blocked distinction, the blocked-leg inventory,
the checklist gate text, the goal-over-scaffolding mandate, and the
12-record clean-retirement precedent.

The table shows selected options and probabilities, not measured design
correctness. Exact option meanings are in each request.

| Decision | Request 1 | Request 2 | Request 3 | Engineering selection |
| --- | --- | --- | --- | --- |
| Blocked retirement | register .93 (substitute .05, stay .02) | register .58 (stay .35, substitute .07) | register .48 (stay .30, substitute .22) | **Retire with register.** See spread analysis below. |

Direction is unanimous (retire with register 3/3) with a decaying margin
(0.93 to 0.58 to 0.48) and a persistent runners-stay minority (0.30-0.35
in two wordings). Investigating the spread: request 3's register text
("closes all 28 records with openly declared holes") prices the coverage
loss bluntly, which moves mass to staying; request 1's ("completes with
stated exceptions") prices it gently. The runners-stay minority is the
voice of "never trade running coverage for prose," and it is right to
demand that the register be genuinely durable and visible rather than a
polite way to forget checks. The substitute-narrow-property minority
(0.22 max) is compatible with the winner rather than opposed to it: a
narrow live verdict where one is honest PLUS the register for the rest.

Selection on repository facts: retire each runner once its wire legs pass
live, and move every check-blocked leg into a permanent coverage register
(checklist entry, Can file header, F01 section) naming the check, the
missing primitive, and the parking date. Narrow-property bonus verdicts
are welcome where a worker finds an honest one, but they do not replace
register entries. Aspect-blocked precedent (A01/A02/B04 simultaneity with
properties proven sequentially) is grandfathered and unaffected.

Application note: the running C03 worker was dispatched under keep-runner
instructions and must not be messaged (a prior coordinator message killed
two workers via an infrastructure fault). It finishes as briefed; the
coordinator applies retire-with-register afterward (retire the runners
once wire legs verify, write the register). C02, not yet dispatched,
carries the new rule inline. Agreement is treated as advice: the
selection rests on the retirement contract and the goal's honesty
requirement, not on the classifier vote. Per the user's standing
instruction, Jev's advice is followed without re-asking.

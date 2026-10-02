# Jev advice on C02/C03/C04 browser-queue scope

2 October 2026. Three new `jev-latest` requests were prepared before any
response was sent. Every state paragraph, the instruction, and each option
description was rewritten across all three versions while preserving facts,
constraints, and alternatives. The [wording audit](wording-audit.json) records
the 10 compared explanatory fields, pairwise wording differences, full-request
hashes, and 26 preserved technical identifiers. Requests
[1](request-1.json), [2](request-2.json), and [3](request-3.json), their exact
[responses](response-1.json) ([2](response-2.json), [3](response-3.json)),
transport metadata, and [probability summary](summary.json) are saved. Each
call returned HTTP 200 and model `jev-1.13.0`; total reported usage was 4,688
input and 171 output tokens. No request contained another response. Jev did
not inspect source or run code; the supplied state summarized the coordinator
triage (baseline reads, catalogue grep for browser operations, live Bun
self-signed-TLS rejection probe), the 28-record retirement gates, and the
machine constraints.

The table shows selected options and probabilities, not measured design
correctness. Exact option meanings are in each request.

| Decision | Request 1 | Request 2 | Request 3 | Engineering selection |
| --- | --- | --- | --- | --- |
| Browser scope | port wire .97 (drive .02, defer .01) | port wire .99 | port wire .97 (defer .02, drive .01) | **Port wire legs with explicit blocked legs (option A).** See reasoning below. |

There is no cross-request disagreement to investigate: all three wordings
agree at confidence 0.96-0.98, and both rejected options score at most 0.02
in every run. The user leaned toward building browser-driving operations
(option C) before the consultation; Jev's near-zero mass on that option is
advice against it, not a veto — the user decides.

Why option A is the engineering selection, on repository facts rather than
the vote:

- The portable majority is large and already de-risked: S04 (all 3 groups)
  plus the S03 issuance/claim/exchange/cookie/SQL/file legs (lines 165-226)
  plus the grant/redirect/header legs of the delivery runners all fit the
  mechanism set behind 12 fulfilled records. Deferring (option B) strands
  that proven value; building drivers first (option C) gates it on the
  largest unestimated surface in the migration (supervised Chromium, a CDP
  client, cookie/CSP observation, a TLS-test client variant — each needing
  catalogue, check, emit, runtime, and qualification work comparable to or
  larger than the fetch_bytes increment).
- The blocked set is small, crisp, and honestly nameable: JS/DOM/CSS/font
  evaluation, real cookie-jar semantics, CSP enforcement signals,
  navigation/referrer defaults, worker gating, and TLS-listener serving.
  Blocked legs with named missing primitives keep F01 honest without
  pretending the runners are gone: the .mjs/.ts files stay until their
  blocked legs clear, which the checklist's per-file gates already allow.
- The standing goal-over-scaffolding rule and the 256 GB/load ceilings cut
  against a platform build whose only consumer is ~7 browser-enforcement
  legs: a CDP driver plus a pinned Chromium would add disk, flakiness, and
  maintenance surface disproportionate to the coverage it unlocks.
- Option C stays available later: nothing in option A burns the bridge.
  If the blocked legs ever justify the substrate, the wire ports already
  carry the server-side half of every scenario.

Agreement is treated as advice: the selection above rests on the triage
evidence and the migration's retirement contract, not on the classifier
vote. Awaiting the user's scope decision before dispatching C02/C03.

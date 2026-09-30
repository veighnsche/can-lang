# Authoring design consultations

Status: advisory design evidence, not implementation or qualification.
The repository requires three fresh Jev consultations for difficult choices.
Each triplet preserves the supplied facts and alternatives while rewriting all
explanatory context, instructions and option descriptions. Exact technical
identifiers remain stable. No previous answer was supplied to a later request.

The API was called with `model: jev-latest`; all six successful responses report
`jev-1.13.0`. Credentials were read from the environment and are absent from
these artifacts. An initial sandboxed attempt could not resolve the API host;
the successful requests used approved network access. No test subjects were run.

## Initial choices

| Question | Preferred alternative | Selected probability in requests 1 / 2 / 3 |
| --- | --- | --- |
| Registration | Ordinary Can library and explicit typed registry | 0.99 / 0.81 / 0.97 |
| Browser/native boundary | Typed generic mechanics returning facts | 0.95 / 0.99 / 0.79 |
| Caught expectation failure | Record mismatch before propagation; Can reducer retains it | 1.00 / 1.00 / 0.99 |
| Compiler rejection | Structured semantic check command | 0.98 / 0.64 / 1.00 |

These are the returned option probabilities, not measured correctness rates.
The responses contain classifications, not researched explanations.

Evidence:

- [Request 1](request-1.json), [response 1](response-1.json), [transport metadata](response-1.metadata.json).
- [Request 2](request-2.json), [response 2](response-2.json), [transport metadata](response-2.metadata.json).
- [Request 3](request-3.json), [response 3](response-3.json), [transport metadata](response-3.metadata.json).
- [Wording and semantic-equivalence audit](wording-audit.json); [reproduction script](../consult.py).

There was no winning-option disagreement. There was meaningful sensitivity:
structured checking received only 0.64 in request 2, with 0.36 for the existing
stderr route; generic mechanics received 0.79 in request 3, with 0.20 for native
scenario-family adapters. Registration alternatives also received some support.
We investigated these alternatives against source and the requirements:

- Existing stderr checks may serve a bounded intermediate migration. They do
  not automatically distinguish parsing, semantic rejection, assertion failure
  and crashes. The internal `CheckSnapshot` already exposes diagnostic facts,
  so a public structured command is a concrete missing general API. Its cost
  remains visible rather than being assumed implemented.
- A native function executing a fixed browser/native scenario would retain
  host test policy. Typed operations satisfy the required ownership split, but
  the five examples cannot establish a sufficient operation set for hostile
  values, interception, DB independence or the full ledger. Those remain
  design obligations.
- Callable records and explicit IDs avoid reflection and a discovery grammar.
  Subsequent source review found that the initial assertion-free function
  examples were invalid: every concrete Can function needs attached assertions.
  The original consultation's registry-function suggestion did not settle this.
  It was corrected to a package-level value and prompted the follow-up below.
- A caught failure and an early `ok` are different hazards. Sticky mismatch
  events address the first; a Can-owned required-check inventory and complete
  execution/cleanup evidence address the second. Native transport never decides
  test expectations or silently turns missing evidence into success.

## Follow-up after source corrections

The new facts were mandatory assertion rows, `when` restricted to a single
`match call`, exported scenario links, no arbitrary opaque-input elision,
constructible reporting/operation context, and the current shutdown order.
A held handler's lease can delay `server.stop(false)`, while the signal path
already closes WebSockets with code 1001 and reason `shutdown`.

| Question | Preferred alternative | Selected probability in requests 1 / 2 / 3 |
| --- | --- | --- |
| Mandatory case assertions | Existing rules plus explicit context and individual supplied boundary fixtures | 1.00 / 1.00 / 1.00 |
| Shutdown ordering witness | External WebSocket close while the held HTTP request remains pending | 1.00 / 1.00 / 1.00 |

Evidence:

- [Request 1](mandatory-followup/request-1.json), [response 1](mandatory-followup/response-1.json), [transport metadata](mandatory-followup/response-1.metadata.json).
- [Request 2](mandatory-followup/request-2.json), [response 2](mandatory-followup/response-2.json), [transport metadata](mandatory-followup/response-2.metadata.json).
- [Request 3](mandatory-followup/request-3.json), [response 3](mandatory-followup/response-3.json), [transport metadata](mandatory-followup/response-3.metadata.json).
- [Wording and semantic-equivalence audit](mandatory-followup/wording-audit.json); [reproduction script](../consult-followup.py).

No alternative won or received a nonzero returned probability in this triplet.
This does not prove either design works. Source review further constrained the
reporter to a constructible captured channel ID, with actual authority retained
by the worker scope. Offline context factories must reuse stable callable values
in actual/expected records; fresh nested callable wrappers are not assumed equal.
The live combined server fixture has not been executed, and request-scope expiry
may require a different retained-work fixture. Observing WebSocket closure proves
shutdown-path entry, not listener closure or the entire shutdown contract.

## Decision and limits

The [authoring contract](../../../native-can-test-authoring-2026-09-30.md) adopts
ordinary functions, static registration, meaningful supplied assertion roots,
typed observations, Can-owned check plans and reports, and the external shutdown
witness. It names the missing APIs and leaves backend/bootstrap selection open.
The [source review](../syntax-review.md), [observation review](../observation-review.md)
and [second review](../review.md) carry more evidential weight than agreement.

Full-field wording differences and a manual semantic audit reduce one source of
framing repetition; they cannot establish independence or remove bias. The six
responses are advice. No implementation, test compilation, live case, benchmark
or migration completion is established by this record.

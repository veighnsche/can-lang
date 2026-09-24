# DI-06 private-clock follow-up with Jev

24 September 2026. The three fresh [requests](request-1.json), [requests](request-2.json), [requests](request-3.json), [responses](response-1.json), [responses](response-2.json), [responses](response-3.json), [wording audit](wording-audit.json) and [reproduction script](consult.py) are saved here. Each request used `jev-latest`; all three replies report `jev-1.13.0`. The requests state the executed 58-root private-clock probe, the prior fixture evidence, identical requirements and the same four alternatives. The script checked pairwise difference across every explanatory state, question and option-description field; I checked semantic equivalence before sending. No earlier Jev answer was supplied as state.

| Request | Choice | Distribution | Confidence |
| --- | --- | --- | ---: |
| 1 | Bounded F2 prototype | prototype .85, scenario adoption .11, test entry .03, public parameter .01 | .79 |
| 2 | Exported test entry | test entry .42, prototype .36, scenario adoption .14, public parameter .08 | .22 |
| 3 | Bounded F2 prototype | prototype .73, scenario adoption .10, public parameter .10, test entry .07 | .64 |

Total reported usage was 3,558 input and 184 output tokens. Request 2 disagreed on the winning label with a low-confidence .42/.36 split. Its question asks for the “least speculative complete next disposition,” which may favor the already executed test entry; requests 1 and 3 emphasize the remaining integration and public-contract comparison. The same facts and options appear in all three, so this is a sensitivity in the judgment, not a detected factual contradiction. The test entry remains a credible current-Can integration path, while a scenario adds an unimplemented compiler/runtime surface. The hard-case observations therefore justify testing F2 in a small disposable prototype before adopting it. They do not establish that F2 beats the test entry, and no reported probability is proof of semantic or cost superiority.

The definite repair is to prevent foreign assertion labels from selecting helper lexical rows. The independent scenario comparison should execute an exact qualified link to helper-owned private clock behavior, reject vanished or changed links instead of using real time, preserve root/table/invocation FIFO and evidence labels, and measure the API/refactor burden beside the executed `stamp_with_clock` baseline. Only a subsequent complete agent-task trial can substantiate token claims.

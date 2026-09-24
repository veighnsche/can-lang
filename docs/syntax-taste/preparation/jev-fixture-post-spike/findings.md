# DI-06 choice after the exported-scenario spike

24 September 2026. Three newly worded `jev-latest` Choice requests considered
the [current-Can private-effect comparison](../fixture-hard-case.md) and its
subsequent disposable F2 generated-output spike. [The requests](request-1.json),
[responses](response-1.json), metadata and [wording audit](wording-audit.json)
are saved for all three rounds; [consult.py](consult.py) reproduces them.
Every explanatory state field, instruction and option description was rewritten
with the same facts, constraints and alternatives. Prior responses were not
included. All replies identify `jev-1.13.0`; total reported usage was 2,807
input and 162 output tokens. Rewording does not prove bias removal.

| Request | Exported scenario | Exported test entry | Public parameter | Confidence |
| --- | ---: | ---: | ---: | ---: |
| 1 | .78 | .21 | .01 | .67 |
| 2 | .95 | .05 | .00 | .92 |
| 3 | .62 | .38 | .00 | .42 |

The leading label agrees, but request 3's .38 for the already executable
`stamp_with_clock` is a material disagreement in strength. Jev supplies no
rationale. Engineering can identify the actual tradeoff: the ordinary test
entry needs no new fixture checker/runtime contract and executes the helper,
but publishes its private clock dependency as API; the scenario preserves
`stamp(int)` and normal callers, but the spike does not establish compiler
diagnostics or concurrent/recursive selection. A public parameter changes all
three app calls in the measured pilot and has little support in this question.
The classifier cannot weigh unmeasured agent creation/repair tokens.

Engineering selects the **bounded exported scenario** as the planned DI-06
contract because it directly satisfies explicit cross-package ownership while
preserving the helper's functional API and was shown feasible for this hard
case. This is not a claim that it is faster, smaller or more agent-efficient
than the test entry. The exact [planned source/selection contract](../fixture-scenario-design.md)
keeps `stamp_with_clock` as the strongest current-language evaluation control.
Production acceptance must cover static links, queue isolation, recursion,
concurrency and truthful evidence, beyond the two sequential calls in the
disposable prototype. The same-owner lexical selector repair is needed
regardless of this choice.

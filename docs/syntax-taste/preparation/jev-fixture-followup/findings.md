# DI-06 fixture follow-up after the current-Can comparison

24 September 2026. This consultation addresses the specific result in the
[fixture comparison](../fixture-comparison.md). It does not implement a new
fixture mechanism or revise the normative specification. The exact
[requests](request-1.json), [requests](request-2.json),
[requests](request-3.json), [responses](response-1.json),
[responses](response-2.json), [responses](response-3.json), and
[wording audit](wording-audit.json) are saved here. The
[reproduction script](consult.py) prepares all three payloads and sends them
only with `--send`. All requests used `jev-latest`; all three replies reported
`jev-1.13.0`. The live TypeSafe [API](https://docs.typesafe.ai/api.md) and
[Choice guidance](https://docs.typesafe.ai/primitives/choice.md) were read.

The three requests preserved the same evidence, constraints and four
alternatives while rewriting every explanatory state, instruction and option
description. Stable option keys and technical identifiers stayed fixed.
The script checked every explanatory field for exact pairwise difference; the
wording audit records the manual semantic check. No previous Jev response was
included in the state. Aggregate reported usage was 3,973 input and 201 output
tokens. These probabilities are classifier advice about a supplied design
question, not implementation evidence or agent-task measurement.

| Request | Selected option | Distribution | Confidence |
| --- | --- | --- | ---: |
| 1 | discriminating trial | trial .71, lexical plus injection .28, scenarios .01, seams .00 | .62 |
| 2 | discriminating trial | trial .70, lexical plus injection .30, scenarios .00, seams .00 | .59 |
| 3 | discriminating trial | trial .64, lexical plus injection .34, scenarios .01, seams .01 | .53 |

There is no winning-label disagreement. The meaningful division is between
making the proven lexical-plus-injection case the final rule and testing a
harder dependency before choosing. Jev consistently preferred the latter,
though the current-rule option retained 28–34% probability. The earlier
[core consultation](../jev-core-contracts/findings.md#di-06-the-three-rankings-optimize-different-coverage-contracts)
split among all three fixture contracts before the new probe. The changed
ranking is consistent with the new evidence making injection a credible
baseline and exposing the missing unsuitable-target case; it does not prove
why Jev changed its weights.

## Engineering disposition

The current typed callable example **does meet the observable
caller-through-helper coverage requirement for that dependency**. The
caller-authored converter runs inside the real helper and the helper's suffix
runs afterward. Changing either changes the result. It survives caller-label
rename. Its evidence is `real-can`, with no fixture-supplied completion.
Calling it an exported helper fixture would misstate the mechanism. It adds a
typed parameter and caller implementation to the helper's functional API.
The executed comparison does not establish that exposing every internal
dependency in that API is acceptable.

The existing ambient `customer` selector fails the rename requirement.
Making lexical rows select only from a test scope owned by that declaration
is a justified semantic repair: a foreign assertion's coincidentally equal
display label must not activate them. Templates can still reuse inert local
content. A whole-helper stub has a valid caller-only role, but cannot claim
helper-body integration; the adversarial helper change in the probe
demonstrated that blind spot. This repair and evidence-label discipline should
be scheduled regardless of which later cross-package mechanism wins.

**Do not yet schedule F2 exported scenarios or F3 exported seams as an
adopted language feature.** Neither has a prototype or a measured advantage
over normal typed injection. Equally, do not close the Gate 1
"deliberate cross-package scenario ownership" promise by citing the one
injectable example as a universal solution. That program explicitly asks for
an exported-link prototype, and a typed function argument is an application
contract rather than a scenario link. Schedule one bounded comparison before
the final feature disposition; keep F2 as the first link prototype because it
publishes a helper-owned behavior without exposing individual internal call
sites. Trial F3 only if F2 cannot express the controlled target at an
acceptable cost.

The discriminating case should use an internal helper dependency whose
configuration would distort or inappropriately expose its public functional
API—for example, a private operation bound to an owner resource—rather than
declare ordinary injection technically impossible. Author the best injection
version anyway, and record the added parameters/call sites. Compare it with a
whole-helper stub and the smallest helper-exported scenario implementation on
the same caller-through-helper assertion. A seam trial is conditional on a
specific scenario failure. This follows the existing evaluation rule requiring
a reproducible gap or measured benefit before more syntax.

## Acceptance contract for the comparison

1. Rename only a caller assertion's display label; the deliberately selected
   helper behavior and expected completion remain stable. A foreign lexical
   row cannot activate from a coincidental label. A link to a deleted or
   inaccessible scenario, or a changed target signature, must fail rather
   than silently run the real dependency.
2. Mutate the helper body and the controlled dependency separately. The
   integrated caller path must fail for both changes, while a whole-helper
   stub is labeled as lacking that coverage.
3. Repeat the helper twice, then nest calls, recurse and run concurrent
   participants. Preserve full root/table/invocation isolation, deterministic
   FIFO reservation, checked target and arguments, and errors for ambiguous,
   missing and unused required bindings. Duplicate display labels in
   independent roots cannot cross-select.
4. Keep `real-can`, `supplied-completion` and `raw-provider-fixture`
   distinct in reports. A passed-in ordinary Can callable is `real-can`.
   Supplied internal results never count as native target or provider
   conformance.
5. Measure creation, held-out caller/helper refactors, and diagnostic repair
   with equivalent agent models, tools and context; count prompts, code,
   diagnostics, retries and validation through success. Compare API
   parameters, published scenario contracts, changed files and diagnostic
   quality alongside tokens. Select the smallest contract satisfying these
   behaviors, and record the tested limits of each mechanism.

The current [fixture runner](../probes/fixture-comparison/run.py) is the
baseline. Its 28 root invocations and mutation controls support only the
specific measured behaviors above. No exported-link runtime or hard
dependency case was run in this follow-up.

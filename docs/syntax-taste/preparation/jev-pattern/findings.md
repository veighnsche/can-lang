# Pattern-intent technical consultation

24 September 2026. This is advice for P5, not a selected syntax or accepted
compiler rule. The live TypeSafe [Choice contract](https://docs.typesafe.ai/primitives/choice)
was used with `jev-latest`. All three [requests](request-1.json) and their
[responses](response-1.json) are saved individually as `request-1..3.json` and
`response-1..3.json`; [wording audit](wording-audit.json) records the semantic
equivalence check. All explanatory prose was rewritten for each request while
the facts, hard constraints and alternatives stayed fixed.

| Request | Highest-probability option | Probability | Next option | Confidence |
| --- | --- | ---: | --- | ---: |
| 1 | explicit binding everywhere | 0.44 | contextual variant rule 0.34 | 0.16 |
| 2 | contextual variant rule | 0.45 | explicit binding everywhere 0.39 | 0.18 |
| 3 | contextual variant rule | 0.55 | explicit both intents 0.27 | 0.33 |

The rankings **disagree** and each distribution is diffuse. The first two
options are close in requests 1 and 2. Jev's replies therefore give no strong
technical preference. It did not run compiler probes or agent tasks. The older
consolidated consultation's `constructors_only` candidate retained bare names
as binders and therefore failed the observed typo acceptance criterion; this
fresh consultation corrected that alternative by making bare names invalid in
variant contexts and requiring explicit capture intent.

The engineering question remains uniformity versus minimizing changed surface.
An everywhere-explicit binder would make identifier intent stable through a
refactor that changes the expected type of a nested element; it changes array
and record captures too. A contextual variant rule preserves current binders
outside nominal variant locations but makes a bare name's meaning dependent on
the expected type, including after a type edit. Requiring both leaf tests and
captures to be explicit has a uniform variant domain but adds syntax to every
case arm. All three reject `decliend`, admit `_` and an explicit whole-value
capture, and require positive/negative nested-pattern tests before presentation
as complete user syntax alternatives. Agent repair and total-task token costs
are not measured. No syntax preference has been inferred from Jev's plurality.

# Jev consultations for the Manolea experience review

30 September 2026. These are advisory classifications of supplied source evidence, not research, explanations, proof or authorization to implement. The [review](../../manolea-experience-review-2026-09-30.md) makes the recommendations and states their limits.

The TypeSafe [HTTP API](https://docs.typesafe.ai/api) and [Choice contract](https://docs.typesafe.ai/primitives/choice) were read live. Each request used `jev-latest`; each response identified `jev-1.13.0`. Requests were sent separately with no prior response in the next request. Raw credential-free request/response JSON and response metadata are saved alongside this file. Responses were validated against the requested question and option sets. No automatic retries or missing responses occurred.

Before sending, all state paragraphs, question instructions and option descriptions were rewritten for each variant. [wording-audit.json](wording-audit.json) records the manual fact/alternative equivalence review, distinctness of all 37 explanatory fields, hashes and pairwise textual similarity. Different wording does not eliminate shared framing or correlated model error.

## Initial three consultations

Each cell gives the selected option, its probability and reported confidence. Full distributions are in [response 1](response-1.json), [response 2](response-2.json) and [response 3](response-3.json).

| Question | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| Live application testing | live facility, .98 / .98 | live facility, .83 / .78 | live facility, .93 / .91 |
| Grouped error arms | group sugar, .91 / .88 | group sugar, .96 / .94 | group sugar, .87 / .83 |
| Repeated row labels | templates, .46 / .29 | templates, .53 / .38 | independent expansion, .54 / .38 |
| Domain-error modeling | layered trial, .99 / .98 | layered trial, .54 / .39 | layered trial, .98 / .98 |
| CRUD/auth reuse layer | packages, 1.00 / 1.00 | packages, 1.00 / 1.00 | packages, 1.00 / 1.00 |
| HTML authoring experiment | compare tree/components, 1.00 / 1.00 | compare tree/components, .99 / .98 | compare tree/components, .97 / .96 |

There was a label-choice disagreement. The domain-error selections agreed, but request 2 assigned .44 to retaining the current application structure, compared with .00 and .01 in requests 1 and 3. That is material wording sensitivity, not a strong uniform conclusion. Source inspection establishes that HTTP response construction and cleanup are real work and existing nominal errors work; it does not establish that every service benefits from a new domain error. The recommendation is therefore a bounded comparison on one operation where callers need distinct outcomes, not a universal conversion.

## Investigation and correction of the row-label disagreement

The initial state said parameterized assertion templates already exist, without supplying their important restrictions. Inspection found that [assertion roots reject template use](/Users/vince/Projects/can-lang/compiler/internal/check/assertions.go:40), while [template expansion assigns one use-site selector to every produced fixture row](/Users/vince/Projects/can-lang/compiler/internal/check/templates.go:246). Existing templates therefore cannot directly replace grouped assertion-root labels, and four fixture selectors still need four uses. The initial comparison gave the template alternative too much apparent applicability.

The actual [Manolea rows](/Users/vince/Projects/manolea-2/src/web/web.can:664) also exercise distinct fixture paths: `configured` receives an asset root and performs recovery, whereas `absent`, `sample` and `serve` receive `http::credentials_missing`. Compressing their written bodies must preserve these selectors.

The original evidence is preserved unchanged. Three fresh [follow-up requests](row-label-followup/) include the corrected template scope, concrete fixture paths and semantic constraints. Their alternatives are independent label expansion, explicitly extending templates to support roots and multiple selectors, retaining current rows, or gathering prototype evidence. The revised template alternative is a proposed extension, not a claim that existing templates already solve the problem. All nine explanatory fields were reworded and checked before sending; see the [follow-up wording audit](row-label-followup/wording-audit.json).

| Follow-up | Independent expansion | Extend templates | Retain rows | More evidence | Reported confidence |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1 | .97 | .01 | .02 | .00 | .95 |
| 2 | .87 | .00 | .04 | .09 | .84 |
| 3 | .65 | .02 | .06 | .27 | .54 |

All selected independent expansion after the correction. This does not identify a causal explanation for the original disagreement: both evidence and the applicable template alternative changed, and the classifier provides no rationale. The third follow-up still gives substantial probability to gathering more evidence. The source-based recommendation is label-list expansion with unchanged roots/fixtures, backed by a concrete equivalence check before implementation acceptance. Templates remain useful for a different kind of reuse.

## Interpretation

Advice agrees with investigating a bounded Can-facing live-test facility, preserving deterministic assertions, grouped completion arms, typed package-level CRUD/auth reuse, and comparing HTML components with a validated tree form. Those are investigation directions, not completed specifications. Exact binder behavior, row expansion, live resource ownership, HTML authoring benefit and package policies still require their stated design and acceptance work.

Generated-code optimization, indentation preference and shorter option names were evaluated from source and the user's stated goals; these consultations did not decide those topics. No performance, safety or correctness claim follows from the classifier probabilities.

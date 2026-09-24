# Jev advice on selected behavior

24 September 2026. Three new `jev-latest` requests were prepared before any
response was sent. Each context paragraph, question and option description was
rewritten in all three versions while preserving its facts, constraints and
alternatives. The [wording audit](wording-audit.json) records the 16 compared
explanatory fields and full-request hashes. Requests [1](request-1.json),
[2](request-2.json) and [3](request-3.json), their exact [responses](response-1.json)
([2](response-2.json), [3](response-3.json)), transport metadata and
[probability summary](summary.json) are saved. Each call returned HTTP 200 and
model `jev-1.13.0`; total reported usage was 3,617 input and 472 output
tokens. None of the requests contained another response. Jev did not inspect
source or run a browser; the supplied state summarized the source checks and
selected contract.

The table shows selected options and their probabilities, not measured design
correctness. Exact option meanings are in each request.

| Decision | Request 1 | Request 2 | Request 3 | Engineering selection |
| --- | --- | --- | --- | --- |
| Browser runtime | profile .89 | profile .96 | profile .91 | **Browser profile.** Existing shims lose async owner context and source maps; fully qualifying Node filesystem, proxy and async semantics adds more host emulation than a bounded profile. Native Promise/DOM/Fetch still own execution. |
| Browser bootstrap | root data .57 | root data .85 | root data .52 | **Disagree: bounded query read.** The current harness already proves native query extraction. A root record would also require admitting server-side `data-*` markup (currently rejected by `html::text_attribute`), a new root decoder, and an encoded boot-data contract. A literal-key, size-capped query read is smaller for the invoice workload; the server treats it solely as an identifier. No browser history/navigation API follows. |
| Public generic proof | SCC .81 | SCC .93 | SCC .63 | **Atomic SCC certification.** Validate every symbolic body/edge before publishing the component proof. Restrict cyclic opaque arguments to bare formals or closed types to bound concrete instance discovery. Separate tests must check this argument; classifier agreement alone does not establish soundness. |
| Event cancellation | synchronous Can predicate .71 | registration policy .62 | registration policy .55 | **Registration policy.** Current Can callback execution is asynchronous. A no-suspend/pure Can predicate would create a new synchronous effect lane; the selected event minimum needs only admitted key/submit cancellation. A checked listener policy calls native `preventDefault()` during dispatch, before the ordinary callback starts. |

The bootstrap and event results are wording-sensitive; the low confidence for
bootstrap requests 1/3 and event requests 2/3 reinforces that these are
engineering trade-offs, not stable empirical conclusions. The root-data
alternative remains a possible later design if a real app needs server-rendered
typed boot records. Its current missing HTML attribute admission is concrete
source evidence; the query choice still needs compiler and browser acceptance.
The later audit found that `URLSearchParams` alone tolerates malformed
percent/UTF-8 text, so the selected operation adds strict raw-pair validation
with native `decodeURIComponent` before using `URLSearchParams`; see the
[native query check](../query-native-probe.md). This tightens the same bounded
query alternative without changing Jev's choice set.
The synchronous-predicate alternative could support dynamic decisions later,
but would need a separately specified sync call graph and no-suspension proof.

The runtime-profile agreement is also conditional: a profile must pass the
empty-app/grid, native-failure, diagnostic, equality, codec, owner and
coordination matrix on its **final served bundle**, including the full runtime
closure, source maps and secret checks. The SCC result is conditional on a
checker proof that cannot borrow an unvalidated callee or emit an opaque
specialization. If either acceptance gate fails, return to mechanism design;
do not silently weaken the selected behavior.

The current API and Choice contract were checked against TypeSafe's live
[documentation index](https://docs.typesafe.ai/llms.txt),
[HTTP API](https://docs.typesafe.ai/api.md) and
[Choice guidance](https://docs.typesafe.ai/primitives/choice.md) before sending.
The last two Markdown pages required direct HTTP reads because the web reader
did not load them. Jev supplies advice; the decisions above use repository
evidence and the user's selected scope.

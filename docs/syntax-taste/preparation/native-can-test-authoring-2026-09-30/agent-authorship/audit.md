# Is the testing design optimized for AI agents?

**Not yet.** The initial proposal established a coverage and execution model
using current Can. Adding the intended-author principle did not establish that
its concrete APIs are best for agents. This audit changes the recommendation;
it is not an implementation, an agent experiment or a performance result.

Requirements remain arbitrary normal Can functions, Can-owned scenarios,
oracles and suite policy, elimination of authored host harnesses, preserved
coverage, independent observations, bounded resources and reliable cleanup.
The backend is open. Existing language implementation has no compatibility
entitlement.

## Keep useful explicitness

Explicit effects, stable case identities, individual fixture rows and ordinary
function composition give agents inspectable contracts. Typed observations,
named failures, exact selection and separate behavior/execution/cleanup outcomes
support diagnosis. These are reasons to retain the direction, not evidence that
every API choice is optimal.

Repetition is useful when it states an independent obligation or makes a local
operation explicit. It is harmful when the same convention must be updated in
several places without an enforced relationship. Line count does not distinguish
those cases.

## Priority changes

| Area | Baseline issue | Revised direction | Status |
| --- | --- | --- | --- |
| Check identity | String IDs repeat in registry and case body | Declare typed check references once, reuse them in an explicit plan and expectations, and reconcile against independent ledger obligations | Proposed library revision; explicit obligations remain required |
| Focused verification | Every selected live run waits for all offline roots | Check all source, execute a conservative selected case/helper/fixture closure for development, fall back to all roots on uncertainty; require full verification for qualification | Source/root separation exists; sound dependency selection is new work |
| Context and fixtures | Agent coordinates reporter identity, operation callables, exports, links and wrapper tables | One complete canonical pattern per boundary family, with small typed operation records and a fixed reporting contract | Current constraints do not prove this particular context design is optimal |
| Diagnostics | Required repair evidence is mostly prose | Machine-readable error categories, exact identities, source locations and reproduction selectors | Missing suite API contract |
| Language versus library | Fitting current syntax can prematurely settle language design | Compare canonical library patterns with narrow general Can improvements that prevent recurring errors | No requirement to freeze current fixture/assertion implementation |

## One source for identity, independent sources for obligations

Illustrative proposed library data, not a compiled file:

```can
str process_case_id = "process/nonzero-result"
spec::check_ref exit_code = spec::check_ref(process_case_id, "exit-code")
```

The declared plan contains `exit_code`; the body uses
`call expect::int(c, exit_code, result.code, 1)`. This `spec::check_ref` record
and expectation signature are proposed APIs. The externally serialized ID stays
`exit-code`. The plan states what must be observed; the body must fulfill it.
Do not derive required obligations solely from executed calls or passing events.

A constructible record gives shared identity and type distinction, not an
unforgeable capability or static ownership proof. Can preflight checks ownership,
uniqueness, obligation mappings and missing references; the reducer checks live
fulfillment. Native code can transport facts but cannot choose expected values.

An agent could delete both a check and its plan entry. Reconcile against a
separately retained, reviewed Can coverage manifest derived from the migration
ledger, not regenerated from current passing cases. Removing an obligation must
be an explicit, reviewable coverage change. Even this cannot prove oracle truth
if an agent weakens both the manifest and expectation: independent negative
controls and review remain necessary.

Co-locate the case descriptor, check definitions, function and fixture entry
points. The central registry references descriptors instead of restating fields.
Keep profile membership explicit and diagnose duplicate/dangling references.

## Focused feedback with honest scope

Current [`Runtime.Assert`](../../../../../compiler/internal/driver/assert.go#L34)
checks the full source graph, then selects assertion roots. This supports
separating source validity from executed verification scope; it does not already
provide the proposed case dependency planner.

Development runs should verify selected cases plus conservatively established
helper/fixture roots. Account for callable targets, generic specializations,
scenario providers and affected shared helpers. Uncertain dependency closure
means all roots. Report the exact selected root set and basis; do not silently
skip source errors. Full source checking can still be costly, so the expected
iteration benefit remains unmeasured.

Bind receipts to source/compiler/executor/options identities and root sets.
Reuse artifacts within an owned run, without adding persistent full bundles or
private caches. Mark partial verification and coverage explicitly. Full
qualification still requires every mandatory root, retained obligation and
required environment. Partial receipts cannot authorize verified production
publication; the executor needs an explicit nonpublishing development path.

## Canonical fixtures and repair diagnostics

The checker [requires assertions](../../../../../compiler/internal/check/assertions.go#L499)
for each concrete function and [limits `when`](../../../../../compiler/internal/check/completion_matches.go#L82)
to one resolved call. Opaque-input and nested callable-equality constraints are
also real current facts. They do not establish immutable language goals.

Complete a canonical package including context construction, stable callable
values, exact effects, fixture wrappers, scenario links, every helper's own
assertions and live invocation. Four body excerpts are insufficient as an
agent's implementation reference. Small per-family operation records should
avoid one universal context containing every platform operation. Templates may
render repeated Can, but scenarios and expected facts remain inspectable Can.

Keep actual comparisons active under supplied roots, with negative rows that
detect altered observations. A canned whole-case `ok` or assertion-mode branch
that bypasses behavior is unacceptable. Compare the library pattern with narrow
general improvements to fixture admission, propagation or scope construction if
they can prevent recurring agent mistakes. Alternatives must retain normal Can
functions, explicit effects, typed completions and separate live/supplied evidence.

Diagnostics should distinguish source/type/effect errors, plan/ownership errors,
fixture selection or consumption errors, subject mismatches, and execution or
cleanup failures. Include root/scenario/call-site identity, occurrence, expected
and observed facts, source spans, candidate/fixture digests and a structured
exact reproduction selector. Preserve the original mismatch if cleanup also
fails. Do not recommend changing an expected value to the observed value merely
to make a test pass; evidence must support identifying which component is wrong.

## Evidence before an optimization claim

Later bounded correctness trials should compare concrete patterns on adding a
compiler rejection, adding a browser check, renaming a check, extending effects,
repairing exhausted fixtures and diagnosing missing cleanup. Seed early returns,
missing obligations, wrong oracles and unavailable required engines.

Observe valid-case rate, repair attempts, missed obligations, false passes,
unrelated edits and required cross-file context. Do not reward fewer lines,
weakened expectations or deleted coverage. Keep compact diffs and diagnostics.
This is a future evaluation contract, not authorization to run trials or
measurements now or a promise of global optimality.

## Review and consultation

An independent source reviewer confirmed these distinctions between current
constraints and optional metadata/context/verification choices. No builds,
tests, live subjects, agent trials or measurements were run.

Three fresh Jev consultations reworded all context, questions and alternatives.
All selected shared typed Can descriptors (probabilities 0.81 / 0.64 / 0.93)
and conservative selected verification (1.00 / 1.00 / 1.00). These are model
outputs, not measured success rates.

- [Request 1](request-1.json), [response 1](response-1.json), [metadata](response-1.metadata.json).
- [Request 2](request-2.json), [response 2](response-2.json), [metadata](response-2.metadata.json).
- [Request 3](request-3.json), [response 3](response-3.json), [metadata](response-3.metadata.json).
- [Wording audit](wording-audit.json) and [consultation script](consult.py).

There was no winning-option disagreement. Request 2 assigned 0.28 to compiler
declarations, so this is a provisional library direction: compiler ownership
could validate identities earlier but adds language semantics. Validated
strings remain a simpler implementation baseline with synchronization costs.
Compare alternatives if typed descriptors leave important errors unchecked.
Jev cannot prove library superiority or dependency-closure soundness.

The current [TypeSafe API](https://docs.typesafe.ai/api) and
[Choice contract](https://docs.typesafe.ai/primitives/choice) were checked;
responses identify `jev-1.13.0`. Credentials were not saved. Source inspection
and later comparative correctness evidence remain authoritative.

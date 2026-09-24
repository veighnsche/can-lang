# Public generic operations and browser-grid offline minimum

24 September 2026. Three fresh `jev-latest` Choice requests considered two
independent planned contracts. The TypeSafe [API](https://docs.typesafe.ai/api.md)
and [Choice guidance](https://docs.typesafe.ai/primitives/choice.md) were read
live; the web reader failed but direct documentation HTTPS succeeded. All
[requests](request-1.json), [responses](response-1.json), [metadata](response-1.metadata.json)
and the [wording audit](wording-audit.json) are saved for each numbered round;
[consult.py](consult.py) reproduces them. Every explanatory objective, evidence
field, question and option description was freshly rewritten while facts,
constraints, alternatives and exact technical identifiers remained stable.
Previous Jev answers were not supplied. This wording review is not proof of
unbiased classification.

| Decision | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| DI-04 public generic operations | explicit callable/dictionary inputs .99 (confidence .98) | same .96 (.94) | same .78 (.68); generated requirement artifact .18 |
| Grid offline minimum | in-view draft and explicit retry .94 (.91) | same 1.00 (1.00) | same 1.00 (1.00) |

All three responses report `jev-1.13.0`; combined usage is 3,433 input and
300 output tokens. There is no winning-label disagreement. DI-04's third
request gave a meaningful .18 to a generated artifact, so it remains a
reasonable documented-template fallback; no response demonstrates an agent
advantage or implementation correctness. For offline, the strongly repeated
minimum follows a state that explicitly said reload survival is **not** in the
recommendation program. It cannot be generalized to a user requirement that
was never stated or to durable offline collaboration.

## Engineering choices

For **exported** generic functions, check the body once under symbolic type
parameters and its written public inputs. An operation on a symbolic type is
admitted only when its typing rule is valid for every type substitution allowed
by that signature. Passing, returning and storing a value without inspecting
it are examples; arithmetic, field access and equality require their own
proof or an explicit named callable/dictionary input. The public signature
cannot silently acquire a `+` requirement through a body edit. A diagnostic
at the generic declaration identifies the operation and missing requirement;
adding the callable changes the public contract, and concrete callers receive
an exact argument diagnostic. Retain the existing specialization-checked
policy for non-exported local templates, labeled as such. This is a semantic
rule, not a claim that a fixed list of primitive tokens is universally valid.
Generated TypeScript invokes ordinary supplied functions and native operations
where equivalent. No implicit instance search, user-defined trait system or
finite-error row is implied. The [current two-caller probe](../generic-helper-probe.md)
establishes a working explicit-input baseline; agent task-token gains remain
unmeasured.

For the **first supported Can browser grid**, promise offline editability
while the view remains alive: retain the in-memory draft, compute totals,
identify unsaved state, show network failure and require an explicit retry or
reconciliation after reconnect. A late response cannot overwrite newer local
edits. Identified application-owned saves remain accountable through view
disposal; a Fetch abort is not proof of server rollback. Reload/navigation
durability and automatic mutation replay are outside this first promise.
Those are separate increments requiring storage eviction, tenant/user
isolation, logout, migrations, multiple tabs, authorization expiry and
uncertain-commit rules. This resolves the unanswered offline question by
evidence and Jev-advised engineering judgment, without asking the user again.

These are planned design contracts. The future implementation must run a
held-out exported-generic body-change case and actual browser offline,
reconnect, late-response and disposal observations. The consultations
themselves ran no compiler, browser or product tests.

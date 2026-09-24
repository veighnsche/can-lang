# Remaining syntax placement: Jev advice and engineering disposition

24 September 2026. **Recommended planning decisions; no compiler/runtime implementation.** The user's latest direction supersedes the two packets' proposed user questions: resolve remaining choices with Jev and engineering evidence, without asking further questions. Earlier confirmed `bind` and dependency-qualified import choices remain unchanged.

## Evidence and consultation method

Read the [error consumer audit](../error-identity-audit.md), [error declaration alternatives](../error-declaration-syntax-question.md), [complete action comparison](../action-contract-design.md), and the current invoice-probe limitations documented there. Read the live [TypeSafe API](https://docs.typesafe.ai/api.md), [Choice guidance](https://docs.typesafe.ai/primitives/choice.md) and documentation index on this date; direct HTTPS succeeded after the web reader failed.

Three fresh requests to `POST /v1/systemone`, requesting `jev-latest`, each asked the same two independent Choice questions. Responses identify `jev-1.13.0`. Every explanatory state field, question and option description was rewritten for each request; technical identifiers/code remained exact. The requests contain the same facts, constraints and alternatives and exclude earlier Jev answers and preferred recommendations. [The audit](wording-audit.json) records the manual semantic review and string-uniqueness checks. Rewording and agreement do not prove freedom from framing effects or correctness.

| Choice | Request 1 probabilities | Request 2 probabilities | Request 3 probabilities |
| --- | --- | --- | --- |
| Application errors | qualified text **0.96**; owner number 0.04 | qualified text **0.91**; owner number 0.09 | qualified text **0.88**; owner number 0.12 |
| Action authoring | source **0.95**; manifest 0.02; helpers 0.03 | source **0.76**; manifest 0.22; helpers 0.02 | source **0.62**; manifest 0.06; helpers 0.32 |

Reported confidence for errors is 0.92 / 0.83 / 0.76; for actions 0.92 / 0.65 / 0.44. All winners agree, but action probability mass moves substantially among alternatives. Jev supplies no explanation; the discussion below is engineering analysis, not an attributed Jev rationale. Total reported usage is 4,561 input and 237 output tokens. All three calls succeeded without retries.

Raw evidence: [request 1](request-1.json), [response 1](response-1.json); [request 2](request-2.json), [response 2](response-2.json); [request 3](request-3.json), [response 3](response-3.json). [consult.py](consult.py) saves all requests, responses and HTTP metadata; running without `--send` only regenerates request/audit files.

## DI-05b — Choose unnumbered application declarations and qualified reporting

**Recommendation: adopt option A for the implementation specification:**

```can
error payment_declined(str reason)
```

Use the canonical owner/package/declaration plus concrete generic arguments as nominal identity. Emit an explicit qualified textual error key and full concrete type information in machine-readable reports. Remove authored application integers and duplicated active-number allocation. Catalogue-owned numeric metadata can remain internal where native adapters need it; it cannot become the application nominal key. Do not add optional compact application integers initially: they have no demonstrated consumer and would introduce another mapping contract.

The audited defect is global coordination between unrelated libraries, although exact matching already distinguishes qualified declarations and concrete arguments. Owner-scoped numbers could also repair composition, but retain two authored allocations and a reporting stability policy for which the audit found no requirement. Unnumbered source removes that obligation directly. The three consultations support this reasoning; they do not establish it by vote.

The final specification must include these concrete requirements before implementation tasks can be marked ready:

- Identical short names in different canonical owners remain distinct, and generic instantiations remain distinct in matching, bounds, assertions and reports.
- Specify canonical owner identity with DI-05a. Checkout relocation and unrelated graph growth cannot change an unchanged declaration's key. Reports also identify their locked build so historical artifacts can be interpreted.
- Rename creates a new declaration identity. Archive old lock/report metadata. Record retired qualified declaration keys in owner retirement metadata and reject reuse within that owner's promised identity scope; this is a tombstone list, not a duplicated active allocation registry. A foreign owner's retirement cannot reserve a local declaration key.
- Test two locked builds, independent libraries, foreign retirement, graph growth and generics using the [audit's cases](../error-identity-audit.md#exact-trial-cases-for-i1-i2-and-i3), adapted honestly to unnumbered source. Preserve payload redaction and coordinated loader, lock, checker, emitter, runtime, CLI and test changes.

This recommendation includes a deliberate source rewrite; it does not claim old numbered declarations continue to compile. No compatibility layer is required.

## DI-09 — Choose a source action declaration, with bounded semantics

**Recommendation: use option A's source declaration as the planned authoring surface**, beside its wire and outcome types. Use the [packet's complete declaration and consumers](../action-contract-design.md#a--source-declaration-with-checked-catalogue-operations) as the starting specification. Export a canonical static action symbol, consumed by checked `action::url`, `action::post` and `action::mount`. Keep the initial contract at exact paths, shallow wire forms and finite response mapping.

Both new candidates require almost all of the same semantic machinery. Manifest placement additionally needs synthetic member names, cross-file symbol strings and key-aware diagnostics; avoiding source grammar does not remove those costs. Source placement gives ordinary package export/resolution and adjacent type references one consistent authoring location. That is a reasoned implementation direction, not a measured claim about agent tokens. An action symbol must remain static checked metadata rather than unrestricted mutable runtime data.

The lower and variable action confidence warrants caution about **scope and adoption evidence**, not another user question. Request 3 assigned 0.32 to ordinary helpers, while request 2 assigned 0.22 to the manifest. All requests explicitly state that neither candidate has been implemented or agent-benchmarked; the existing 18 assertions do not observe HTTP, persistent writes or DOM. Therefore treat source placement as the specification direction while retaining the strongest current helpers as the evaluation control. Do not publish improved task efficiency or product correctness until the stated comparisons pass.

Implementation-plan boundaries:

- One canonical action identity joins route/method, URL construction, form ownership, mounting and exhaustive case mapping; static field references carry the owning wire record and exact field type.
- Preserve actual 200/422/409/403/503 statuses and verify the pinned client adapter's observable policy. A declaration alone cannot repair missing 503 behavior.
- Specify strict decoding and pre-handler 400/404/405/413/415/422 behavior; test missing/duplicate/unknown/malformed inputs, incomplete cases, stale names and wrong-owner references.
- Keep server mounting out of browser targets without dragging handlers/renderers into shared contract modules.
- Preserve independent authorization/CSRF, tenant/revision enforcement, uncertain-commit reconciliation, real database observation, target absence and stale-response tests. Static action typing proves none of these effects.
- Keep captures, nested rows, components and typed render scopes separate. Do not allow those unresolved mechanisms to block the exact-path contract or to enter through incidental syntax.
- Run creation, rename and diagnostic-repair comparisons against factored helpers, including held-out field/case names, under the existing evaluation protocol. Correctness and reliable edits precede secondary successful-task token measurements.

No central plan or status files were changed by this consultation task. No production code or runtime TypeScript was edited, and no new compiler/browser/database test result is claimed.

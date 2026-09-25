# Jev advice on UP19 invoice handlers and pages

25 September 2026. Three new `jev-latest` requests were prepared before any
response was read. Each state paragraph, instruction, and option description
was rewritten in all three versions while preserving facts, constraints, and
alternatives. The [wording audit](wording-audit.json) records the 24 compared
explanatory fields and full-request hashes. Requests [1](request-1.json),
[2](request-2.json), and [3](request-3.json), their exact
[responses](response-1.json) ([2](response-2.json), [3](response-3.json)),
transport metadata, and [probability summary](summary.json) are saved. Each
call returned HTTP 200 and model `jev-1.13.0`; total reported usage was 4,490
input and 522 output tokens. No request contained another response. Jev did
not inspect source or run code; the supplied state summarized the ownership
boundaries, the htmx noSwap/hx-status mechanism, the contract's swap cases,
the skip-gate evidence, the pairing/splice mechanism, and the candidate
designs.

Agreement is advice, not proof. All three consultations picked the same
option on every question, so there were no disagreements to investigate. The
selections match the engineering analysis recorded in the handoff: runtime
derivation keeps authors from forging policy, createHTML plumbing follows the
declared-URL precedent, and the captured page URL stays a plan-level gap
rather than a hardcoded or unchecked route.

| Decision | Request 1 | Request 2 | Request 3 | Engineering selection |
| --- | --- | --- | --- | --- |
| Swap producer | element 1.0 | element 1.0 | element .99 | **Element derivation.** The runtime element serializer appends exact hx-status attributes for declared non-2xx swap cases when an hx-post/hx-get URL matches an HTML POST action shape. |
| Table plumbing | createHTML 1.0 | createHTML 1.0 | createHTML .99 | **createHTML argument.** Optional fourth parameter, passed by the emitter only for programs declaring HTML POST actions. |
| Page scope | keep+gap 1.0 | keep+gap .87 | keep+gap .79 | **Keep query page, file gap.** Integer-key query form page stays, grid shell is added, captured /edit goes to planning as checker-owned follow-up. |
| Shell auth | public .95 | public .86 | public .66 | **Public shell.** Always 200; the browser app validates keys and renders denials. Lowest confidence of the set, still unanimous. |

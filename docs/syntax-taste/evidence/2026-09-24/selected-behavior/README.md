# Evidence for the selected behavior packet

24 September 2026. The [selected behavior packet](../../../post-upgrade-selected-behavior-2026-09-24.md)
is the authoritative design outcome for this step. The files here preserve
source observations, native probes, proposed mechanisms and decision advice;
they do not claim implementation or product qualification.

| Evidence | What it establishes and what remains |
| --- | --- |
| [Actions and server binding](actions-proposal.md) | Current handler coupling, explicit callable capture, selected request/Origin/ledger contracts. The subsequent audit identified and corrected the missing request-token revocation claim in the selected packet. |
| [Browser profile and delivery](browser-proposal.md) | Current four Node shims, inert entry and event settlement; a profile design. The selected packet adds strict query decoding, compiler-owned boot and an explicit server/browser manifest handshake. |
| [Public generic proof](generics-proposal.md) | Symbolic public-callee and atomic SCC rules with parse-only Can examples. Concrete checker/emitter acceptance remains. |
| [Pinned HTMX behavior](htmx-proposal.md) | Exact-status ordering and a disposable Chromium DOM check for real failure statuses and blocked unrelated 500 OOB content. The selected packet additionally confines all tasks of admitted action responses. |
| [Native regex](regex-native-probe.md) | Bun 1.4.2 `matchAll` positions for empty Unicode/non-Unicode patterns. |
| [Native query](query-native-probe.md) | Bun `URLSearchParams` replacement tolerance and encoded-key alias behavior. |
| [Three fresh Jev consultations](jev/findings.md) | Saved requests, raw responses, wording audit, disagreements and engineering selections; classifier advice only. |

## Focused mechanism experiments

| Question distinguished | Reproducible evidence | Result and limit |
| --- | --- | --- |
| Can the Gate 5 async-context shim preserve owner identity after `await`? | [Interleaved browser-owner probe](experiments/browser-owner/findings.md) | No: the exact shim loses context and real owner resource use fails in Chromium. Native Bun succeeds; a small explicit-token control is not a Can implementation. |
| Can an asynchronous event handler cancel the native default before dispatch returns? | [Event-cancellation timing](experiments/event-cancel/findings.md) | No; a synchronous listener policy can, while unmatched/noncancelable events still reach the handler and disposed listeners do not. |
| Can the current request cleanup also end a retained capability? | [Buffered and lazy token probe](experiments/request-lifetime/findings.md) | No; `abandonRequest` leaves both tokens usable, so the selected lifetime needs revocation. |
| Can the pinned HTMX asset's events confine a selected action response? | [Request/swap guard probe](experiments/htmx-guard/findings.md) | For the tested shapes, `before:request`, `before:response` and `before:swap` block missing-target submission, response redirects/retargeting and OOB/partial tasks. Reporting and production integration remain untested. |
| How does today's checker handle generic recursion and nested composition? | [Generic recursion probe](experiments/generic-recursion/findings.md) | Six fixtures distinguish finite mutual recursion, growing recursion and acyclic nesting. A minimal public identity checks but exposes a symbolic-type emission failure. The selected SCC proof still needs implementation and soundness tests. |

The selected B02 regex-iterator change already has a separate
[native `matchAll` probe](regex-native-probe.md) and reuses the review's
[Can regex output](../post-upgrade-review-961f921/core-probes/regex-output.txt).
General stack-safe iteration (P05) and HTTP hedge/request cancellation (P06)
are [deferred](../../../post-upgrade-dispositions-2026-09-24.md#retain-or-defer-the-other-findings).
The review's countdown and owner-latency probes document their current
limits; no new experiment here promotes either proposal into this upgrade.

The selected packet's five existing-syntax Can excerpts passed `canlc parse`
after extraction: shared records/variants, server call sites, browser main
with a minimal package header, and the two generic packages. Proposed action
declaration grammar and catalogue operations require new checker fixtures.
No server/browser build, live invoice transaction, supported asset publication
or cross-browser qualification was completed for this documentation task.

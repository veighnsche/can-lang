# DI-05b syntax question: application error declarations

24 September 2026 · comparison packet for three fresh Jev consultations and
engineering selection under the user's later no-questions instruction; no
grammar is adopted in this packet.

Two unrelated dependencies can each allocate application error `1000000`.
Current Can rejects that graph even though nominal matching already uses a
qualified declaration and concrete generic arguments, rather than the naked
integer ([consumer audit](error-identity-audit.md)). The current terminal
report includes both the number and `typeIdentity`; no repository evidence
requires a naked application number to identify an error across locked
builds. There are zero external users to migrate.

The hard contract under either option is that two independent libraries,
authored under the selected grammar and then left unchanged when linked,
compose; their error kinds stay distinct, a generic error's
specializations stay distinct in matching, reports are machine-decodable across
the promised stability scope, and redacted payloads remain redacted. Current
global numeric allocation cannot pass. Three fresh
[Jev judgments](jev-core-contracts/findings.md#di-05b-audit-requirements-before-treating-a-report-key-as-a-design)
split between audit-first and scoped numbers; the audit is now complete and
does not establish a naked cross-build numeric requirement. Jev did not choose
final syntax.

| Choice | Authored declaration and registry | Report identity and consequence |
| --- | --- | --- |
| **A. Qualified textual identity** | `error payment_declined(str reason)`; no authored numeric application ID or separate active-number registry. Canonical identity is the resolved owner/package/declaration plus concrete generic arguments. Renaming a declaration changes identity; old reports retain the old locked build identity. | Report an unambiguous qualified textual key and type arguments. An optional compact integer is build-local with an archived reverse map and build ID, never the canonical error identity. This removes two coordinated authored allocations and is the agent-first engineering recommendation; it may make report lines longer. |
| **B. Owner-scoped numeric identity** | `error 1000000 payment_declined(str reason)` and an owner-local active/retired registry. A second dependency may use or retire the same number without collision. | Report both canonical owner identity and local number, plus concrete type identity; never interpret `1000000` alone. Source and registry still duplicate the allocation, but numbers can remain stable inside one owner's reporting policy. |

Catalogue-owned built-in errors may retain internal numeric metadata for
adapters under either choice; application nominal matching never keys on that
metadata. A chosen contract must update compiler loading/checking, lock and
registry shape, emitted plan, runtime admission, terminal reports, diagnostics,
assertions and tests together. Option A requires a one-time source/registry
rewrite in this repository; it is not credited as accepting old numbered
source unchanged. No old source or report compatibility is required, but
two-build report interpretation and retirement/rename policy
must be explicit. The [full acceptance cases](error-identity-audit.md#exact-trial-cases-for-i1-i2-and-i3)
cover active/active, active/foreign-retired, generic and graph-growth cases.

The decision changes source syntax. The user can select either complete
contract or propose a different spelling that still passes those cases.

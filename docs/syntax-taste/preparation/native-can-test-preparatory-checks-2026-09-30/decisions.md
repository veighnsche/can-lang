# Preparatory scope decisions

Status: scope decisions only; all mechanics checks remain unrun.

Three independent `jev-latest` requests returned `jev-1.13.0`. All explanatory state, instructions and alternative descriptions were rewritten in each request, with equivalent facts/options checked before sending. The [wording audit](wording-audit.json), requests and complete responses are retained. Consultation used the current [TypeSafe API](https://docs.typesafe.ai/api) and [Choice contract](https://docs.typesafe.ai/primitives/choice). Jev is a classifier; it received the evidence and alternatives rather than being asked to research or invent the design.

| Decision | Response 1 | Response 2 | Response 3 |
| --- | --- | --- | --- |
| Temporary external instrument | `python_scoped`, probability 0.99 | `python_scoped`, 0.99 | `python_scoped`, 1.00 |
| Initial candidate ingress check | `source_partial`, 0.98 | `source_partial`, 0.98 | `source_partial`, 0.75; `defer`, 0.25 |

All selections agree, but the third ingress answer is less decisive (reported confidence 0.62). It gives no rationale. Our investigation of the tradeoff: importing an already exported codec does not settle the hard generated-Can ingress question, so deferring it avoids spending time on evidence with a narrow claim. Retaining the 30-second check gives a concrete runtime/import/hostile-access observation and validates that the proposed handle-based shim reaches actual candidate code without evaluating the accessor first. We retain that limited check, add the normal `[1n]` → `[1]` setup witness, and explicitly leave C-compiled ingress and reference qualification unproved. Do not interpret the classifier's probability as a measured chance of the mechanism working.

Python T avoids a new Go build while investigating POSIX-to-Bun fd mechanics and fixed native/browser fixtures. It is temporary reviewed scaffolding, with known graph, external deadline, explicit ownership and no parent-death claim. The selected final Go N architecture is unchanged. Building full N before these preparatory questions would reintroduce the sequencing mistake; compiling a small Go instrument remains a future alternative if a specifically Go-dependent question warrants it.

Browser scope uses installed Chrome explicitly through a parent-owned process and CDP, not a silent replacement for missing Playwright-managed Chromium. Its real click promise is measured; early settlement is inconclusive. A form navigation improves the chance that the intended dependency exists but is not assumed proof. No database/upstream/replay work is hidden in this first check.

Independent source review found no blocking issue in the main/descriptor scopes. It requested consistent per-leg budgets and propagation of the PM-N2 setup witness into the native card; both are included. Source evidence, classifier advice and later measured facts remain distinct. Agreement is neither feasibility proof nor guaranteed bias removal.

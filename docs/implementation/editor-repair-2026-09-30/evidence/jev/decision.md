# Architecture consultation disposition

Date: 2026-09-30. Facts inspected at `9c283544466876700ef7a079f0bc3136451c0008`.

Three final fresh requests to `jev-latest` resolved to `jev-1.13.0`. Every context, instruction/question and option description was independently reworded while preserving facts and alternatives. Independent Codex review corrected formatting/symbol scope parity, acknowledged closure-isolation entry-point cost and replaced ambiguous root-version shorthand with the complete document-version vector plus manifest/lock/registry/asset/dependency fingerprint. Earlier requests/responses are retained under `initial-wording/`; use final `request-*.json` and `response-*.json` as the decision evidence.

| Decision | Round 1 probability | Round 2 | Round 3 | Disposition |
| --- | --- | --- | --- | --- |
| Canonical compiler recovery | 0.90 | 0.78 | 0.73 | Select |
| One serialized analysis worker | 0.95 | 0.95 | 0.92 | Select |

No selection disagreements. Recovery confidence varies (0.85, 0.67, 0.59); round 3 assigns 0.17 to a focused prototype first. This variation reinforces the early recovery correctness gate, especially sticky type-builder failure and transactional dependency commits. Agreement is advice, not correctness proof or guaranteed bias removal. No performance benefit is claimed.

Canonical recovery is selected because lexical/parser, resolver, type and checker phases currently discard later findings and share mutable failed state. Repeated whole-project check-after-masking risks fabricated semantics and excess laptop work; real closure isolation still needs global-state partitioning and joins. Implement explicit invalid/blocked units and safe state commits in canonical routines. CLI can be refactored: its product contract is rejection of errors and no partial invalid emission, not preservation of historical first-error APIs/goldens. One worker keeps protocol input responsive while serializing compiler mutation; complete fingerprint checks govern publication and edits.

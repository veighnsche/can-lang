# I18 scheduler ordering consultation

Three fresh Jev requests compared retained priority-queue ordering with an
arrival-invalidated native sorted cache and the existing repeated scan/sort.
Every explanatory context, question, and alternative was rewritten while
retaining the same facts and constraints. `wording-audit.txt` records the
pre-dispatch semantic and wording check. Requests and raw responses are saved.

All three responses selected `priority_heap` with probability/confidence 1.0
(model `jev-1.13.0`). There was no disagreement to investigate. This is advisory;
phase-transition invariants, failure recovery, lexical order, comparison budgets,
and generated release execution require independent validation.

The TypeSafe HTTP API and Choice documentation read for the earlier consultation
were reused; no API or model interface was inferred from the classifier itself.

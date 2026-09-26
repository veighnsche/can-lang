# P05 decision packets — what Jev is and is not asked

Researched facts, constraints, exact examples, viable alternatives, acceptance
criteria, and evidence against the favored option live in
[research/](../research/README.md) and [alternatives/](../alternatives/README.md).
The Jev state fields below compress that record; the consultation tests
classifier judgment under wording variation, not new research.

## Consulted (one round, 7 questions, 3 fresh wordings)

| Question | Options | Why Jev |
|---|---|---|
| host_extension | catalogue / reviewed_adapters / companion | Genuine 3-way split in the review round; consequential architecture |
| request_lifetime | operation_deadlines / scope_policy / deployment_limits | 2–1 split in the review round; consequential contracts |
| iteration | self_tail / iteration_primitive / bounded_policy | Near-uniform third vote in review round; consequential lowering |
| failure_abstraction | error_rows / result_data / fixed_bounds | Central library decision; test whether result-data-first survives rewording |
| assertion_setup | setup_region / factory_pattern / scenario_coverage | Test whether factory-first survives rewording |
| ui_composition | library / syntax / interop_ui | Test whether library-first survives rewording |
| failure_hook | builtin_auto / author_callback / manual_only | New since the review round; small but consequential default |

## Deliberately not consulted

- **R08 authoring policies (Boolean order, final locals, near binding):** user
  syntax choices (Q1–Q3). Jev does not vote on user taste.
- **R15 S3 defect:** contract-led (data-loss implications dictate fixing code
  to the catalogue unless no no-complete release exists). Classification adds
  nothing; isolated storage qualification decides.
- **R03 route shape:** bounded two-shape engineering choice; decided in P07
  with rationale. Artificial consultation avoided.
- **R10/R11/R14 scope-heavy topics:** worker home, SQL breadth, and AI
  qualification turn on pending user scope answers (S1–S4). Consulting before
  scope is set would spend budget on possibly out-of-scope framings. A
  follow-up round can be run after P06 if a scoped technical question remains
  genuinely uncertain.
- **R09 editor order:** cost/dependency-led (format → hover → references →
  completion → rename); user prioritizes in Q8. No classifier judgment needed.
- **R13/R16:** qualification scheduling and doc corrections; mechanical.

## Procedure record

- `jev-request-1.json` … `jev-request-3.json`: prepared before sending the
  first; all explanatory prose rewritten; same facts, constraints, questions,
  alternatives; stable technical identifiers and option keys.
- `jev-wording-audit.json`: full corresponding-field difference check +
  semantic-equivalence review note.
- `jev-response-N.json` + `jev-response-N.metadata.json`: exact responses.
- `findings.md`: distributions, disagreement investigation, engineering use.

# P05 Jev findings — preparation round 2026-09-26

Model `jev-1.13.0` (requested `jev-latest`); 3/3 HTTP 200. Usage: 5,656 input
+ 918 output tokens total. All requests prepared before the first send; 37/37
explanatory fields differ pairwise (max similarity 0.655); semantic
equivalence manually reviewed — see `jev-wording-audit.json`.
Numbers are selected-option probabilities, not design-correctness probabilities.

| Decision | C1 | C2 | C3 |
|---|---|---|---|
| host_extension | reviewed_adapters (0.71) | reviewed_adapters (0.60) | reviewed_adapters (0.48) |
| failure_abstraction | result_data (0.58) | error_rows (0.50) | result_data (0.65) |
| assertion_setup | factory_pattern (0.92) | factory_pattern (0.96) | setup_region (0.53) |
| iteration | iteration_primitive (0.60) | iteration_primitive (0.76) | iteration_primitive (0.55) |
| request_lifetime | operation_deadlines (0.68) | operation_deadlines (0.66) | operation_deadlines (0.74) |
| ui_composition | library (0.97) | library (0.91) | library (0.89) |
| failure_hook | builtin_auto (0.96) | builtin_auto (0.93) | builtin_auto (0.80) |

## Agreement (advice, not proof)

- **ui_composition → library (0.97/0.91/0.89):** consistent with the review
  round (1.0/0.82/0.92). Supports the library-first extraction experiment
  (X-R02-1); component syntax stays deferred.
- **request_lifetime → operation_deadlines (0.68/0.66/0.74):** stronger than
  the review round (2–1). Supports O1-first layering under a written
  request-policy spec; scope_policy (~0.2–0.28 throughout) stays the
  evaluated second layer, not a rejected idea.
- **failure_hook → builtin_auto (0.96/0.93/0.80):** new question, strong
  agreement. Supports the automatic redacted reporter reusing existing
  machinery; manual-only is advised against (0.04–0.06).

## Disagreements and sensitivity (investigated, P05.4)

- **host_extension: unanimous yet weakening (0.71 → 0.60 → 0.48, catalogue
  0.33 in C3).** The review round split three ways; this round's state
  defined the adapter boundary more concretely, which plausibly pulled all
  three votes. The C3 near-tie preserves the uncertainty: catalogue growth
  for demonstrated cases remains a live alternative. Engineering use: keep
  the discriminating experiment (X-R01-1, third-party widget both ways);
  do not treat 3/3 as a decision. No majority-vote rule is applied.
- **failure_abstraction: C2 flips to error_rows (0.50, confidence 0.25,
  near-tie with result_data 0.36).** Wording 2 framed the gap as "failure
  abstraction lags" with "the prize is a single helper" — that framing
  plausibly pulled the vote toward the direct abstraction. The flip exposes
  genuine sensitivity, not noise to outvote. Engineering use: exactly the
  recorded decision rule — run the result-data experiment (X-R06-1) with a
  distinguishable oracle, and adopt error-set syntax only if adapters remain
  extensive. The C2 vote is preserved as the counter-case in P07.
- **assertion_setup: C3 flips to setup_region (0.53, confidence 0.30,
  factory 0.26, scenario 0.21 — nearly flat).** Wording 3 stressed shim
  burden ("adds declarations, rows, and reader burden"), plausibly pulling
  the vote. The flat distribution is itself the finding: all three wordings
  agree the cost question is open. Engineering use: factory-first with a
  worked extraction example (X-R07-1); setup syntax only via explicit user
  approval (Q4) since LD29 closed that gate.
- **iteration: primitive 3/3 (0.60/0.76/0.55) against the coordinator's lean.**
  Preparation's engineering lean favors attempting self-tail lowering first
  (no new syntax, identical observable behavior, native-loop lowering either
  way). Jev's repeated primitive preference is advice for an explicit
  contract over an inferred proof boundary. Investigation: both options share
  the same native-loop lowering work — the real decision is inferred surface
  (O2) vs explicit surface (O3, needing user approval Q6). Engineering use:
  build the lowering with the O2 proof rule first; the O2 experiment doubles
  as the primitive's lowering validation, and O3 is adopted only if the proof
  boundary excludes needed shapes. Evidence against the favored (O2) option:
  proof-boundary soundness risk (an over-broad proof silently changes
  semantics); Jev's 3/3 primitive preference; explicit step-indexed
  diagnostics come free with O3.

## Review-round context (preserved, not settling)

The review round's host 3-way split and lifetime 2–1 split are the reason
this round re-consulted with harder state. Shifts between rounds reflect
different state emphasis under rewording, not converging truth. All
classifications remain advisory to P07 engineering decisions.

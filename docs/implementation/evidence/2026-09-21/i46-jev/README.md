# I46 sealed-type architecture advice

Three fresh consultations compare isolated sealed extensions, a symbolic
discovery pass before one global seal, and restarting global graph construction
as requirements appear. Every explanatory context, instruction and option was
rewritten before submission; the saved wording audit records equivalent facts,
requirements and alternatives.

All three select sealed extensions, with probabilities 0.96, 1.00 and 1.00 and
confidence 0.93, 0.99 and 1.00. There is no selected-option disagreement. This is
advice, not proof or guaranteed bias removal.

The implementation direction is an immutable type inventory, fresh builders
which copy only required sealed argument graphs, and a cached concrete-body
worklist. Tests must establish graph isolation, preserved recursive identities,
complete reachable-body checking, exact inference and finite refusal of expanding
recursion. These consultations do not establish I46 completion.

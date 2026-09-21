# I04 integration consultation

Three independent System One requests evaluated the same implementation boundary:
an independent current parser and inert command with an explicitly legacy
predecessor; conversion into predecessor nodes; or pulling all later compiler and
editor work into I04. Each request contains the no-compatibility constraint, current
grammar requirements, existing consumers, task dependencies, costs, and required
verification. All context, instructions and option descriptions were rewritten.
The script checked every prose field for pairwise inequality; manual review checked
equivalent facts, constraints and alternatives before submission.

All three responses selected `independent`, with probability 1.0 and reported
confidence 1.0, 0.99 and 1.0, respectively. The returned model was `jev-1.13.0`.
There was no disagreement to investigate. These are design advice, not correctness
evidence or a guarantee of bias removal. Parser, rendering and offline integration
tests establish the implemented behavior. Requests and responses are retained
verbatim beside this file.

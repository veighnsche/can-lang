# I11 startup supervision consultations

Three fresh requests and their raw responses are retained here. All context,
question instructions and option descriptions were rewritten for each request;
the facts, constraints and alternatives were manually checked for equivalence.
The request generator also compared every explanatory field pairwise before
sending. No source credentials were included in any request.

All responses selected explicit supervision, with probabilities 0.99, 0.90 and
0.58. The third response assigned 0.41 to module-level initialization, so this
was not treated as strong consensus. Inspection of the actual contract resolves
that alternative: initialization faults must be caught before main, and authored
work cannot occur during module import. Generated modules therefore export an
initialization function invoked inside the root completion boundary. The entry
module awaits the supervisor before assigning `process.exitCode`.

Model agreement is advisory. Startup-fault, delayed-completion, safe-diagnostic,
and offline staged CLI tests provide the implementation evidence. Ownership
and draining remain I20's work; this slice cannot launch detached source work.

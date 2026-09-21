# I15 native operation lifetime

Three fresh requests preserve the same A4/Q10/P6 requirements, existing owner
mechanism, alternatives and ordinary-call constraints. Every explanatory context,
question instruction and option description differs; the script checked those
wording differences before submission. Requests and raw responses are saved here.

All three select ownership of actual native promises with independently timed
caller waits. Confidence is 1.0, 0.99 and 0.94; the third retains 0.04 probability
for waiting for all cleanup before timeout. There is no categorical disagreement.
The selected policy follows the explicit bounded caller wait and root retention
requirements; agreement is advice, not proof or guaranteed bias removal.

Tests show a deadline returning while a controlled native promise remains pending,
then root completion only after that promise settles. Late cleanup rejection is
observed without changing the selected result. Native loopback body stalls cause
abort and timeout with no decode entry. Scope integration additionally requires
native continuations to inherit an existing owner's key while that scope drains;
new coordination owners remain disallowed. This is the maintained native boundary,
not a whole-invocation resource lease on ordinary callables.

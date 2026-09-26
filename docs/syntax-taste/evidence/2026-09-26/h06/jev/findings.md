# H06 Jev consultations: generation-handshake design

26 September 2026. Advisory evidence for task H06 (generation handshake and
CAS-safe pairing, R13/R02/R03, interface C-H). Three fresh live requests to
`jev-latest` (all answered by `jev-1.13.0`) cover the three hardest
H-side design decisions below. Every explanatory state field, instruction,
and option description was rewritten in full across the three rounds
(45 fields: 3 state fields, 3 instructions, and 9 option descriptions,
each in 3 full phrasings); task IDs, requirement IDs, file
paths, kind strings, function names, `0400`/`0600` modes, the seven-day
bound, and code spellings stayed exact. No request contains any prior Jev
answer. The [TypeSafe skill](../../../../../.agents/skills/typesafe-ai/SKILL.md)
was followed. Jev performed no research and returned typed choices,
distributions, and confidence values, not prose reasons.

## Why these three decisions

The H06 scope sentence fixes *what* (browser sends paired server
generation; typed mismatch; blocking refresh) and the lane split fixes
*who* (H staging/pairing/pruning/distribution; E transport/server; C app
prompt; A emission). What remains genuinely open on the H side is *how
the identity flows given the build order*: (1) **acquisition** — the
browser bundle is built before the server generation exists, so the
paired server generation must reach it through a post-build channel;
(2) **comparison** — the server needs a stable expected value and a
fail-closed answer in both rollout and rollback directions;
(3) **rollback** — prune deletes non-current generations, so reverting
to a prior paired build needs a defined provision. Each question carries
at least one superficially cheaper alternative (metadata fetch,
selection-pointer comparison, retained generations, log-and-serve),
which is what makes the choice worth an outside judgment.

## Exact evidence

| Round | Submitted request | Saved response |
| --- | --- | --- |
| 1 | [request 1](request-1.json) | [response 1](response-1.json) |
| 2 | [request 2](request-2.json) | [response 2](response-2.json) |
| 3 | [request 3](request-3.json) | [response 3](response-3.json) |

[consult.py](consult.py) reproduces the requests byte-for-byte and sent
exactly three API calls with no retry; `--send` refuses to run when any
shared 8-gram violation is present (none were). The
[wording audit](wording-audit.json) lists the preserved fact sets, and
[summary.json](summary.json) records selections, probabilities, and
usage. Reported usage totals **5,272 input and 432 output tokens**.
Credentials were supplied through the authorization header and are absent
from saved evidence.

## Results

Selected probability is the model's probability for that candidate, not
the probability that the design is correct. Confidence is a separate
provider value. Order is round 1 / 2 / 3.

| Decision | Selected in every round | Probability | Confidence |
| --- | --- | --- | --- |
| Acquisition channel | `page_embedded` | 0.99 / 0.95 / 0.99 | 0.98 / 0.92 / 0.97 |
| Server comparison | `manifest_startup_typed` | 1.00 / 1.00 / 1.00 | 1.00 / 1.00 / 1.00 |
| Rollback provision | `rebuild_stable` | 1.00 / 0.96 / 1.00 | 0.99 / 0.95 / 0.99 |

H06 adopts all three selections: the served page embeds its serving
generation; the server pins its own manifest `buildID` at startup and
answers any missing/malformed/unequal generation with the typed
mismatch before running action logic; rollback rebuilds prior source
through stable build IDs and the ordinary atomic selection path.

## Disagreement investigation

There is no categorical disagreement: all nine judgments are unanimous.
That unanimity is investigated rather than celebrated. The supplied
constraints overdetermine two of the three outcomes: the fail-closed
interop policy (no compatibility across generations; old logic against
a new server is a defined error) is stated in the shared state, and the
`log_and_serve` / `assets_only_rollback` / `browser_build_echo`
alternatives explicitly concede the corresponding property, so the
consultation functions mainly as a coherence check on those legs — the
selected options are the only ones fully consistent with the stated
contracts — not as independent evidence that the contracts are right.
The `metadata_route` vs `page_embedded` and `retain_prior` vs
`rebuild_stable` contests are less determined (round 2 gives the
rejected options 0.01–0.04 combined), and there the judgment carries
real information: identity should travel with the artifact whose
behavior it labels, and stable rebuilds beat hoarded generations.
Agreement is therefore treated as advice: it confirms the H06 contract
direction but proves nothing about the E/C/A slices, which remain
pending handoffs with their own qualification burden.

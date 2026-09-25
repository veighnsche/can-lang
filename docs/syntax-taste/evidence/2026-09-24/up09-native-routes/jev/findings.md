# UP09 Jev findings: dispatch placement

Three fresh, fully rewritten consultations (jev-1.13.0) on where captured
action dispatch lives. All three select `router_carried_combined` with
probabilities 1.00/1.00/1.00 (confidence .99/1.00/1.00):

- Mounts return `http::route` tokens; `make_router` assembles one combined
  value (legacy exact table + compiled action table + callbacks).
- Dispatch serves exact legacy matches first, then strict captured matches,
  with target-level 400 first and a union 405 + Allow.
- Server start extracts native Bun route keys from the router value; every
  native callback re-enters the same canonical pipeline.
- Duplicate/ambiguous action shapes refuse at assembly with existing failures.

No disagreement to investigate. Agreement is advice, not proof: the status
vocabulary, lifecycle and probe coverage are verified by the UP09 tests,
not by this consultation.

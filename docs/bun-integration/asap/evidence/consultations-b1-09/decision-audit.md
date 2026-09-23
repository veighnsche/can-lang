# B1-09 Cookie/CSRF decision audit

Three independently rewritten packets asked four identical questions over
the B1-09 plan, the filetree bounds and the pinned Bun 1.4.2 probe ledger
(cookie_parse, cookie_serialize, cookie_headers, csrf_token, csrf_expiry).
All prose (state, instructions, criteria) differs across packets; question
keys, option keys, identifiers and measured facts are stable. Verified
mechanically before dispatch (0 identical prose fields of 51
comparisons; 25 fact tokens present in every state). Model:
`jev-1.13.0` via `jev-latest`, three HTTP 200 rounds, 3999 input /
516 output tokens total.

## Results

| Question | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| duplicates | first_wins 1.00 (conf 1.00) | first_wins 1.00 (conf 1.00) | first_wins 1.00 (conf 1.00) |
| policy | serialize_native 0.98 (conf 0.97) | serialize_native 0.88 (conf 0.82) | serialize_native 0.98 (conf 0.96) |
| csrf_failure | false_vs_config 0.93 (conf 0.88) | false_vs_config 0.93 (conf 0.89) | false_vs_config 0.94 (conf 0.91) |
| secret | default_secret 0.87 (conf 0.81) | default_secret 0.63 (conf 0.45) | default_secret 0.83 (conf 0.74) |

Requests, responses and SHA metadata: `request-N.json`,
`response-N.json`, `response-N.metadata.json` in this directory.

## Agreement (accepted as advice)

- duplicates `first_wins` 3/3 at certainty: lookup returns the first
  pair for a name, matching the native getter, while the full
  wire-ordered pair list stays exposed for inspection. `last_wins`
  and `all_pairs` score 0.00 in every round.
- policy `serialize_native` 3/3: the adapter claims no prefix or
  SameSite enforcement and serializes whatever the native constructor
  accepts; browser treatment is documented, not adapter-policed.
  `adapter_enforces` never exceeds 0.09.
- csrf_failure `false_vs_config` 3/3: malformed, expired and
  mismatched tokens answer false; only invalid configuration (empty
  secret, empty session, out-of-range durations) fails domain errors.
  `error_taxonomy` scores 0.00 in every round.

## Disagreement (investigated, plan prevails)

- secret `default_secret` 3/3 contradicts two explicit plan
  constraints: B1-09.03 requires a caller-supplied secret and forbids
  relying on a per-thread secret that changes at restart, and the
  capability summary bars the native thread-local default as the
  normal application contract. The packets carried default-secret
  self-consistency as a measured fact but restart volatility comes
  from the Bun documentation rather than a same-process probe, so the
  deployment constraint never entered the state; Jev optimized for
  call-site convenience without it. Round 2 is also weak (0.63 at
  confidence 0.45, `explicit_required` 0.36). Selected:
  `explicit_required` per plan; the native default is not exposed.
  `secret_handle` never exceeds 0.01, so secrets cross as strings
  with B1-09.04 keeping them out of diagnostics.

## Selected record

`cookie::parse` yields first-wins lookup over a retained ordered pair
list; `cookie::make`/`cookie::serialize` pass attributes to native
serialization with documented browser consequences and no
adapter-side prefix or SameSite policy; `csrf::generate` and
`csrf::verify` take explicit secret plus nonempty session with fixed
base64url/sha256, answering false on token faults and domain errors
on configuration faults.

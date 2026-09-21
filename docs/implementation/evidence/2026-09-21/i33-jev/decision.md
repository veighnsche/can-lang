# I33 server lifetime design decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round, same
question and option order) advised seven server contracts. Requests,
responses, and the equivalence audit live in this directory.

Unanimous: bind failures echo `host:port` from configuration with no native
detail (0.88, 0.82, 0.89); request leases span dispatch only while native
`stop(false)` owns socket drain (0.49, 0.89, 0.76); repeated or stale stops
report standard `resource_state` on a non-idempotent server (0.59, 0.73,
0.81); only awaited servers react to signals through waiter-owned listeners
(0.88, 0.99, 0.94); shutdown phases distinguish caller-wait expiry
(`deadline`) from native-stop failure (`stop`), 2-1 with the dissenting
round ranking it second and nobody supporting a single phase.

Config split went 1-2 for deferring all address validation to Bun while
`make_server_config` checks only adapter-consumed numbers. Investigation
confirmed the majority on the merits: Bun accepts empty hostnames
(measured), and any Can-side host grammar risks over-rejecting deployments
Bun would serve, with no Can contract requiring stricter rules. Port 70000
and unresolvable hosts therefore surface as `bind_failed`, never as config
errors.

Numeric bounds split three ways (sibling mirror, uncertain, unbounded).
The deciding facts are sibling precedent (`ai-io` `max_body_bytes`
1-67108864) and the `setTimeout` overflow above 2^31-1 ms, which would turn
an accepted bound into immediate expiry. So `body_limit` accepts 1-67108864
and `shutdown_ms` accepts 0-2147483647, with `invalid_server_config`
naming the field.

These judgments are design advice, not verification. Runtime, compiler and
staged integration checks remain required before I33 can be marked complete.

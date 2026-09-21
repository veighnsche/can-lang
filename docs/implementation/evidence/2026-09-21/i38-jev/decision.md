# I38 transaction decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round, same
questions, options, order, and measured facts) advised the begin-failure
error, the commit_unknown ID shape, and the pre-commit drain bound.
Requests, responses, and the equivalence audit live in this directory.

Split, decided for connection_begin: a refused or closed native begin on
an open pool reports connection_failed with phase begin (connection
0.44, review 0.60, connection 0.68). The evidence genuinely pulls both
ways: native codes frame a connection event while the failing operation
is transaction entry. The deciding merit is module consistency: I35 maps
the same ERR_POSTGRES_CONNECTION_REFUSED/CLOSED codes to
connection_failed on the query path, and a refused begin is the same
native event at a different site, which the begin phase already names.
transaction_failed stays reserved for failures inside a live transaction.

Unanimous abstention, decided on the merits for structured_sequence:
review_needed drew 0.81, 0.53, 0.85, with the two ID shapes splitting
the remainder. The request stated the tie honestly: any present, safe
value passes the fault-proxy check. The deciding merit is
correlatability: a pool-resource-derived identity plus a per-pool
attempt counter lets operators trace attempts in logs and lets tests
assert exact IDs, while still carrying no driver payload or credential.

Majority for unbounded_drain: 0.87, 0.66, then review_needed at 0.62.
The deciding merit beyond the majority is that with_transaction offers
no timeout input, so a bounded drain would invent a deadline from
nowhere, while owner scopes already drain to completion and the task
requires existing subleases to finish. Nonsettling owners stay
observable under an external supervisor instead of being masked by a
timer.

These judgments are design advice, not verification. Checking,
emission, runtime, fault-proxy, and live-database checks remain required
before I38 can be marked complete.

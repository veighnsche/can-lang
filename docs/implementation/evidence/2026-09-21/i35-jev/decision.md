# I35 pool and query decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round, same
questions, options, order, and measured facts) advised the pool
establishment proof, max_rows error placement, and unknown-shape handling.
Requests, responses, and the equivalence audit live in this directory.

Unanimous: pool_open awaits the native connect handshake before returning
the pool (0.78, 0.84, 0.90), so refusals and authentication failures
surface as connection_failed at open and a returned pool has demonstrated
reachability. Lazy construction alone cannot distinguish a live server
from a dead socket, and a probe query wastes a round trip for no extra
proof.

Unanimous and strong: defective max_rows arguments report
unsupported_value with path max_rows before anything launches (0.99,
0.99, 0.98), while row_limit fires only when a completed bounded fetch
observes more rows than allowed. Argument defects and observed overflow
are distinguishable before and after launch, so they stay distinct
errors.

Unanimous but weak: only PostgresError values classify into the query
errors; foreign throws propagate to the generic fault path (0.46, 0.74,
0.56, with the collapse-to-unknown option drawing up to 0.32). The split
is genuine: both readings fit part of the evidence. The deciding merit is
that a non-PostgresError throw is not a native failure at all but a
driver defect, and masking it as a query outcome would destroy exactly
the visibility tests need. Finite classification governs every
PostgresError; faults stay faults.

These judgments are design advice, not verification. Schema, checking,
emission, runtime, and live-database checks remain required before I35
can be marked complete.

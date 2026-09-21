# I37 descriptor strictness decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round, same
question and option order) advised limit exclusivity and trailing trivia.
Requests, responses, and the equivalence audit live in this directory.

Unanimous and strong: preserve trailing trivia (0.95, 0.95, 0.99). A
descriptor is valid with exactly one RawStmt; leading and trailing
semicolons, comments, and whitespace persist literally in the emitted
segments. The server tolerates all of it, independently rejects genuine
multi-statement input, and the single-RawStmt rule stays the one criterion.

Limit exclusivity split with no conviction: shared (0.61), once (0.61),
shared (0.50 against 0.48, a virtual tie). This weak 2-1 is overridden on
the merits. P12 calls the top-level LIMIT the distinct parameter after the
application parameters, and sharing lets a call-site LIMIT value (bound by
I35 from max_rows or 2, not from any record field) silently fill a filter
position: with parameters [term] and limit 2, a stray $2 in WHERE would
compare the filter against 2 or 26. That misbinding is silent under sharing
and a precise compile error under exclusivity, and no reasonable initial
descriptor wants a filter equal to its LIMIT. The row-limit number therefore
appears in exactly one PARAM token, the top-level LimitCount.

These judgments are design advice, not verification. Descriptor, checking,
emission, runtime, and live-database checks remain required before I37 can
be marked complete.

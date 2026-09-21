# I36 SQL binding decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round, same
question and option order) advised the parser binding, error vocabulary, and
adapter call shape. Requests, responses, and the equivalence audit live in
this directory.

Unanimous and strong: pin the official CGo binding pg_query_go/v6 v6.2.2
(0.97, 0.98, 0.91). The evidence agrees: identical PG 17.7 outputs on all
24 corpus cases, ~5 ms cold invocations against ~454 ms of one-time wazero
compile per SQL-touching process, ~12 MB against ~300 MB transient peak
RSS, and a tagged official release against an untagged third-party pin.
The price is clang plus CGO_ENABLED=1 wherever canlc itself is built,
which plan.md already allows as a source-build prerequisite.

Result packaging went 3/3 for separate Parse and Scan entry points, though
rounds 1 and 3 were soft (0.54, 0.59) against bundled_call. Separate calls
match the upstream API shape both bindings already expose, keep each pass
independently testable, and fit I37, whose statement-shape checks and token
segment splitting naturally consume one pass each.

Error vocabulary split 2-1 for verbatim upstream message plus cursor (r1
closed_reasons 0.71; r2 verbatim 0.68; r3 verbatim 0.91). The dissent was
investigated on the merits: several I37 classifications (notably
multi-statement) are structural facts, not upstream error shapes, so a
closed reason set cannot cover them without re-deriving structure anyway,
while every closed set still needs the upstream message as an "other"
detail. The adapter therefore surfaces message plus character cursor
verbatim, and I37 classifies from structure (statement counts, token
shapes, spans) rather than message substrings. Upstream rewording across
parser upgrades then changes diagnostic prose only.

These judgments are design advice, not verification. Adapter tests,
release-candidate builds, and staged checks remain required before I36
can be marked complete.

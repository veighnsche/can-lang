# I41 editor-surface decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round,
same questions, options, order, and measured facts) advised the
spanless-error anchor, the definition-resolution depth, and the
unsaved-file treatment. Requests, responses, and the equivalence audit
live in this directory.

Unanimous at full confidence for file_line_anchor (1.0, 1.0, 1.0):
resolve and check failures report the CLI-identical message on line 1
of the attributed file, falling back to the open file when none is
named. Token-exact ranges stay exclusive to the spanned lexer and
parser diagnostics; no heuristic underlines invented positions.
Followed.

Near-unanimous for file_scope_symbols (0.81, 0.97, 0.99): definition
jumps resolve the identifier at the offset through file, package,
prelude, and import scopes to nominal, function, constructor, and
generated declarations, and decline body-local names. Every offered
jump is a resolved declaration identity; scope-at-position machinery
for locals stays out. Followed.

Unanimous for ignore_until_saved (1.0, 0.96, 0.99): buffers whose
canonical path is absent from the loaded project publish empty
diagnostics until saved. Disk plus overlays always reproduces the
report; no virtual membership can contradict the CLI verdict.
Followed.

These judgments are design advice, not verification. Bridge, overlay,
transport, grammar, and parity checks remain required before I41 can
be marked complete.

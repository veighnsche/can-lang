# I32 opaque-assertion consultation decision

## Results

All three responses selected `bare_opaque_ok` with confidence 0.98, 0.94,
0.98 (probabilities 0.98, 0.96, 0.99). No disagreement: `exempt_handlers`
peaks at 0.02, `structural_response_compare` at 0.03, `vacuous_runner`
at 0.0.

## Decision (advice, not proof)

Bare `ok` is allowed in assertion expectations only, and only for
C-excluded opaque/callable results; the runner passes undefined-expected
against opaque actuals. Rows still verify fixture structure, boundary-call
arguments, and completion kinds; staged loopback verifies response content.
Valued expectations stay required for data results, and bare `ok` stays
rejected in bodies and match arms. The C exclusion is thus explicit in
source rather than a silent runner waiver, and no response comparison
reads the write-only sink.

Compiler, runner, and staged tests must still prove the gate (opaque-only,
expectations-only), the runner rule, and the content split before I32 can
close.

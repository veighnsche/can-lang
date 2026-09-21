# I18 empty aggregate selection correction

Independent review reproduced out-of-order fixture allocation when the first
outer participant awaits an empty `all`, `settled`, or `any` aggregate and the
second participant immediately requests the same FIFO fixture table. Empty
aggregates never call participant observation, so their native selection
continuation previously had no active scheduler frame.

The assertion context now reserves and starts the selection gate at admission
for these three empty aggregates. Selection and prelaunch abort both release
that gate. Empty `race` remains pending and does not acquire this gate.

Regression tests run the actual assertion runner, nested native coordination,
and shared fixture rows for all three settling modes. They require successful
assertion evidence and the exact event order: selection, row 0, row 1.

Validation: qualified Bun 1.4.2 ran 23 tests / 132 expectations across the new
empty-selection suite, barrier suite, and coordination suite. Strict TypeScript
checking passed for the changed context and new tests. This correction does not
close the separately confirmed scheduler scalability finding or complete I21.

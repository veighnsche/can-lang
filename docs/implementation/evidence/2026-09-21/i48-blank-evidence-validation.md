# I48 review correction: reject blank resolved identities

Independent review found two incomplete-evidence cases. An explicitly present but
empty terminal identity was treated as a different binding, while an empty string
inside a capture list could permit an unnecessary-local diagnostic. Both violate
the pass's complete-resolution prerequisite.

The terminal identity must now be nonempty, and every listed capture identity must
be nonempty. An explicitly empty capture slice still means a resolved reference
with no captures; existing positive and negative controls remain unchanged.

- [Failing regressions before correction](i48-blank-evidence-before.txt).
- [Finite-local and initialization regressions](i48-blank-evidence-tests.txt).
- [Isolated full Go suite](i48-blank-evidence-go-tests.txt), with qualified emitted
  Bun execution. This ran against an archive of committed I48 plus only the two
  corrected Go files, excluding the concurrent uncommitted I09 work.

Both regressions pass after correction. I48 remains complete; this does not mark
I09 or later body/capture implementation tasks complete.

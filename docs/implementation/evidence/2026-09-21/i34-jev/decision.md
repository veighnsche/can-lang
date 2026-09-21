# I34 asset pipeline design decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round, same
question and option order) advised four asset contracts. Requests,
responses, and the equivalence audit live in this directory.

Unanimous: `.json` assets must parse at build time (0.97, 1.00, 0.93);
names resolve in the caller's own project manifest only, with dependency
assets shared through callee functions returning `html::url` values
(0.74, 0.78, 0.89).

Binary signatures went 2-1 for validating magic plus fixed header
structure (lengths inside the file, nonzero dimensions and counts, legal
versions and codes; no checksums or decoding) over magic bytes alone.
The dissent favored the smallest adapter, but P11 requires rejecting
mismatched signatures, and truncated or garbled files deserve precise
build errors rather than silent service as corrupt bytes.

Reasons went 3/3 for a single `missing` reason, but at the consultation's
lowest confidences (0.14, 0.42, 0.28) with real mass on both alternatives.
This advice is overridden on the merits: the established `invalid_url`
vocabulary is per-cause specific (`syntax`, `scheme`, `authority`,
`same_origin`), the task names ownership and missingness as separate
cases, and the fixes differ (declare the name versus respect the
boundary). Unknown names report `missing`; names declared only in another
project report `unowned`.

These judgments are design advice, not verification. Runtime, compiler and
staged integration checks remain required before I34 can be marked complete.

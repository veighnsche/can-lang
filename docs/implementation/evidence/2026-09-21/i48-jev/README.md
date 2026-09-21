# I48 finite-local checker consultation

The design question was how to keep the four C8 predicates exact while the later
body/region and named-capture passes retain ownership of their resolution work.
The alternatives were complete resolved-use/capture evidence plus a typed
substitution check, spelling counts with absent captures guessed empty, or a
general purity/equivalence optimizer.

Each of three fresh requests rewrites all explanatory state, instructions and
option descriptions. A pairwise check verifies differing prose in every such
field; semantic review preserves the same four predicates, compiler capabilities,
future ownership, shadowing constraint, evidence-completeness requirement, lack
of compatibility obligations and absence of performance measurements.

Raw requests and responses are stored as `request-1.json` through `request-3.json`
and `response-1.json` through `response-3.json`. All three identify `jev-1.13.0` and
choose `evidence` (reported confidence 1.0, 0.99, 1.0). No disagreement needs
resolution. Agreement remains advisory, not proof or guaranteed bias removal.
Tests separately establish identity-based accounting, required capture records,
finite syntax exclusions, unchanged typing and retained inference-sensitive cases.

Live API/choice documentation was read through HTTPS after the web reader failed;
no API credentials are included in these artifacts.

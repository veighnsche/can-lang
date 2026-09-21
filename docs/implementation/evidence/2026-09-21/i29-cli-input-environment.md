# I29 bounded CLI I/O and environment

The compiler admits the remaining I29 intrinsics and emits native bounded stdin
and exact environment adapters. Stdin reads compare chunk lengths against the
remaining bigint budget before retaining bytes, preserve immutable ownership,
and cancel on overflow. Negative budgets fail before opening stdin; zero accepts
empty input. Text uses fatal UTF-8 decoding with BOM preservation. Native I/O
errors with known OS codes become declared failures; unrelated defects retain
the standard failure channel. Writes remain awaited Bun writes with exact counts.

Environment adapters validate names before lookup, expose no enumeration/mutation,
and read the original caller snapshot. Missing and empty values are distinct.
`optional` constructs the exact `option::some<str>` or `option::none` leaf from its
concrete result graph, even when unrelated `option::some<int>` also exists.
All four I/O calls and both environment calls require supplied completions during
ordinary assertions; no ambient input is used as a fallback.

P8 now documents negative/zero limits and empty-value presence explicitly. Three
fresh Jev consultations advised incremental caller-bounded input without an
invented fixed cap; their full requests/responses and wording/equivalence audit
are in `i29-jev/`. The implementation and tests, not agreement, establish behavior.

## Verification

- Five runtime tests (93 expectations) cover exact binary input, split multibyte
  UTF-8, BOMs, huge bigint limits, empty input, negative/zero limits, cancellation,
  malformed UTF-8, expected native read errors, unexpected defects, exact
  environment values, rejected names before lookup, and assertion isolation.
- `TestCurrentBundledInputEnvironment` runs the absolute packaged CLI with empty
  PATH and networking denied. Binary and text pipes round-trip exactly; overflow,
  malformed encoding, negative limits, missing environment and invalid names
  produce the declared status/error. A one-MiB payload drains to both stdout and
  stderr before success. Three source fixtures also pass deterministic assertions
  and strict generated TypeScript.
- The environment fixture proves empty entries, missing entries, Unicode values,
  and the original hostile `BUN_OPTIONS` value survive snapshot lookup without
  executing that option. An unrelated concrete option instance checks leaf IDs.
- Existing staged CLI coverage checks application-only argv and actual broken
  pipe error 1211. A pinned Bun probe using a directory as stdin reported native
  `{name:"Error",code:"EISDIR"}`, one of the explicit read-failure codes.

Final gate: `bun test runtime` passed 220 tests and 20920 expectations. Strict
TypeScript passed for the new adapters and tests. Full compiler and integration
checks passed with the pinned Bun archive and generated TypeScript checking:
`go test ./compiler/... ./tests/integration -count=1` (integration 98.017s).

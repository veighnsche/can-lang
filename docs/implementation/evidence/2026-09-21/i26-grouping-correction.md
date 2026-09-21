# I26 correction — grouped fetch entry context and literal Content-Type

Independent review found two admission inconsistencies. A direct empty query or
header array received its required `str[]` context, but the same expression inside
parentheses did not. Also, JSON-body Content-Type conflicts were diagnosed for
flat literals but not for grouped literals or fully literal array spreads.

The construction hint now looks through grouping while the checker still checks
the original expression. This preserves source locations and ordinary expression
semantics; existing arrays are not retyped or widened.

Static Content-Type inspection now gathers completely known string values through
grouping and literal array spreads. It preserves written value order, treats an
empty array as removal of the connection header, and rejects a known conflicting
effective value. Unknown calls/bindings defer to the existing runtime transport
check. No constant-function evaluation or alternate header transport is added.

Evidence:

- The first 17-case matrix failed 13 cases on `c162a3a`: grouped/deep-grouped empty
  query/header arrays failed inference, grouped empty Content-Type failed inference,
  and five known grouped/spread conflicts were accepted.
- The repaired checker passes that matrix: eight query/header empty-array forms,
  five invalid known Content-Type forms, and four valid grouped/empty forms.
- Five additional positive cases cover empty overrides of a conflicting connection
  default, a valid literal override and a dynamic expression. The same default
  without an override still rejects.
- Existing header-character and fetch-mode negative tests continue to pass.
- The maintained staged fetch fixture includes deeply grouped empty query/header
  entries and a grouped literal-spread JSON Content-Type. The server verifies the
  omitted entries are absent, alongside the existing exact body/header checks.

These are compiler admission fixes. Native header encoding and runtime dynamic
validation are unchanged.

Full compiler and staged integration validation passed with the pinned archive and
CAN_TSC: `go test ./compiler/... ./tests/integration -count=1` (integration:
89.624 s). This includes all eight staged fetch requests, four assertion roots,
strict generated TypeScript, mixed questions and existing release/runtime gates.

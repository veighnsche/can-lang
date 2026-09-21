# Noul judges

The current executable example is
[`compiler/testdata/current/native/noul.can`](../../compiler/testdata/current/native/noul.can).
Configure its connection endpoint and `CAN_I17_TOKEN`, then run it with a staged
`canlc run <project>`. It submits three independently named occurrences in one
TypeSafe SystemOne-compatible POST. Only declared state fields are disclosed.

Noul handlers receive `%` as the original true probability. The true handler runs
when that probability equals or exceeds `minimum` (default `0.5`); the false
handler receives the same probability. All answers are validated before any
handler starts, even when a result binding is unused. The judge continuation runs
after every registered handler succeeds.

Ordinary `canlc assert` uses the example's explicit `when` rows and reports supplied
completion evidence. Compiler conformance tests separately supply raw HTTP data
at the native transport boundary, execute request encoding and response validation,
and report raw-provider evidence. They never fall back to a live provider.

See [I17 validation](evidence/2026-09-21/i17-validation.md) for exact tests and limits.

# B1 — acceptance and canonicalization repair

Implementation commit: `d9037ca`.

Scope: F01 correctness repair in the existing language. No success/call syntax,
ABI, host trust policy, or higher-order admission redesign is selected here.

## Behavior

- Changing a pinned factory's expected callable and its implementation together
  now produces `CAN6017`, even when both candidate programs pass their own tests.
- Function references encode the target, specialization/type arguments, named
  captures, capture values, and capture order. Resolved row evidence also records
  the target's owner/revision and its transitive interface/type/error dependencies.
- Explicit call/constructor type arguments and typed-Ok pattern annotations are
  retained. Resolved generic target stamps remain distinct.
- Invocation structure includes the `with` argument. Argument encoding distinguishes
  a positional argument from a named argument literally called `pos`.
- Type whitespace, comments, and source locations do not change identity.
  Derived type/certificate annotations are not treated as authored structure.
- Unknown expression/pattern/match kinds and unelaborated chains are not printable
  hash inputs. They raise an internal typed canonicalization failure, converted
  to an error at generation/diagnostic boundaries; unrelated panics are not hidden.
  No partial candidate baseline is written. The compiler does not emit target
  artifacts following a canonicalization error.

## Identity and consumer audit

| Layer | Canonical input | Consumers and exclusions |
|---|---|---|
| Own interface | `revisionDeclEntries`: declared signatures, record/variant/error shapes, contracts, authority declarations, constant/state initializers | `RevisionEntry.Own` and structural drift diagnostics. Function bodies, test rows, and scripts are excluded. |
| Interface dependency closure | `FingerprintProgram`: own interface plus recursively resolved nominal/error/effect dependencies | `RevisionEntry.Fingerprint`, `Deps`, `DepPrints`; `CheckRevisionIdentity` in CLI/LSP. Fn input/result/error members are now traversed, including Fn-valued fields/results. Existing content-only re-key treatment remains unchanged. |
| Executable structure | `canonNode` / `canonSmall`: ordered checked body/match structure, invocation arguments, patterns, references and captures | Nested contract blocks are serialized this way; baseline generation/enforcement validates that bodies can be represented. Top-level bodies do **not** enter interface hashes. There is no persisted executable/proof cache added by B1. |
| Acceptance expectation | `canonPinnedRow` plus `canonRowDependencies`: row identity, inputs, type bindings, expected value, exact referenced declaration keys and dependency prints | `PinnedRows`, `CheckPinnedRows`, candidate baseline output, CLI/LSP acceptance warnings. A pure dependency revision re-key stays visible here even when its interface content is identical. Unmarked rows remain proposed; implementation bodies and given tables do not become pinned expectations. |
| Proof authority | Fresh admission and verification of the checked program | `VerifyContracts` re-proves; saved reports are not proof authority. Structural collision tests do **not** demonstrate a persisted proof-cache exploit. |

`canonNode` alone is not a complete executable-world cache key: in particular,
its raw reference spelling needs resolved world identity and executable dependency
closure before anyone could safely introduce such a cache. This repair audits
that distinction rather than creating an unused cache/proof protocol.

## Baseline format and authority

`RevisionFormat` is now **2**, and fingerprint hashes are version-domain-separated.
Format 1 omitted semantic inputs and is rejected, not upgraded in place. Format 2
requires a `pinned` section, including `{}` when there are no pins.

Generate a new **candidate** from checked sources, then independently review and
explicitly accept it through the existing trusted baseline-selection process.
`WriteBaseline` still writes `accepted: false`; generation never promotes an
expectation to accepted authority. Replacing the format integer in an old file
is not a migration procedure. Origin/accepted metadata is not a cryptographic
identity or an authenticated pinner; the selected baseline remains a trust input.

An implementation-only change does not change a promised callable's identity at
the same revision. It changes executable structure and must still pass current
checks/proofs. Changing a callable target/revision, capture, or referenced schema
changes the pinned reference evidence. These are deliberately different events.

## Regressions and gates

`compiler/canonical_test.go` covers:

- green pinned factory target/capture changes, including LSP warnings;
- target revision and transitive payload revision/schema changes;
- fully expanded int/bool generic callback stamps;
- explicit type arguments, typed-Ok constructors/patterns, invocation arguments;
- argument label discrimination/order and exclusion of derived annotations;
- comments/spacing, own-interface versus dependency versus body separation;
- unsupported/malformed canonical structures, no partial baseline file, and error diagnostics;
- old/missing evidence rejection, candidate round trip, and explicit acceptance.

Validation:

```sh
go test ./compiler -run TestCanonical -count=1
go test -count=1 ./...
go run ./tools/modcheck
go run ./tools/gramcheck
```

The B11 success-values bundle also generates a format-2 unaccepted baseline
(52 declaration entries). Fresh `success.ts`, `use.ts`, and `errors.json` remain
byte-identical to the committed goldens. Existing repository-wide strict-TS and
stdlib-composition defects remain B3/B4 work; B1 does not claim to repair them.

The old `evidence_test.go.txt` F01 collision probe is preserved as a historical
reproduction and now fails its old collision hypothesis. Use the normal
`TestCanonical*` regressions for current behavior, not the historical assertion.

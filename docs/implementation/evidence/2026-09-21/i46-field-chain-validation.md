# I46 finite nominal-field chain correction

Review of `ae406ba` found another false rejection: `walk<T>(value.next[0], count - 1)` visits `seed`, `step<seed>`, then `done<step<seed>>`, and stabilizes at the third instance. All transitions occur at one source call site. Preloading the terminal type through an assertion caused the same program to pass. The new regression reproduced exactly that failure before the correction.

The compiler no longer treats concrete containment or repeated call sites as proof of expansion. A direct self-call can prove growth through source substitutions that retain every corresponding type parameter and nest at least one under a constructor. Explicit type applications supply those substitutions directly. Inferred applications recover conservative evidence from original, lexically resolved input annotations and array construction. Each recovered substitution must resolve to the actual invariant inference result. Field projections and other unsupported expressions provide no proof. Excessive unknown or indirect specialization remains subject to separately labelled finite implementation limits.

The obsolete ancestor and instance-creation-site guard state has been removed. Diagnostic request sites and concrete instance caching remain intact. No language syntax or compatibility path is introduced.

Validation:

- `TestFiniteGenericFieldChainIgnoresAssertionOrder` requires exactly three `walk` instances with the terminal type unloaded, preloaded first, or preloaded last.
- `TestGenericWholeBodyAndRecursion` retains same-instance recursion and rejects explicit and inferred `item -> item[]` expansion with the expansion diagnostic.
- Existing explicit/inferred fixed-target and expected-element spread regressions pass.
- `TestCurrentBundledGenerics` executes all three field-chain variants, including two real runtime transitions, through a fresh absolute staged compiler with networking denied and `PATH=/nonexistent`.
- Checker, type and emitter suites pass in the working tree. `go test ./compiler/... ./tests/integration -count=1` also passes separately from unfinished I19 work in an export of HEAD containing only the correction's files, with the qualified Bun and archive configured. This includes the staged generic regressions and development-sidecar checks.

The three design consultations and their limits are recorded in [i46-field-chain-jev](i46-field-chain-jev/README.md).

# I46 review corrections

Independent review confirmed two false rejections in `47f7e38`. The first correction below was subsequently shown insufficient by a finite nominal-field chain and is superseded by the [source-evidence correction](i46-field-chain-validation.md).

1. A single structural enlargement was classified as expanding recursion even for `fixed<int>` calling the constant target `fixed<int[]>`, which stabilizes at that instance. Preloading the target through an assertion accidentally changed admission. The guard now requires a recurring source application site for the same declaration before diagnosing structural growth. The immutable creation site is separate from later diagnostic request sites. Tests cover explicit and inferred transitions in both cache-population orders and preserve rejection of genuinely growing recursion.
2. Inference represented literal-spread elements as indexing expressions over the complete spread array. That discarded known expected element context, so `int[] result = call pick(...[[], []])` failed while explicit `pick<int[]>` worked. Inference now constrains the original literal elements directly, including grouped and recursively flattened literal-spread syntax. Ordinary argument lowering still checks the complete array's homogeneous contract and evaluates it once; no general array covariance or error-bound inference is added.

Validation:

- Checker, type and emitter suites with the qualified Bun enabled.
- Fresh staged `TestCurrentBundledGenerics`: original cross-module program, six assertion rows, invalid unused branch rejection, both independent review reproducers, preloaded and inferred finite-transition variants; `assert` and `run` invoked through the absolute bundled compiler with networking denied and `PATH=/nonexistent`.
- `git diff --check`.

The three fresh guard-design consultations and equivalence audit are saved in [i46-review-jev](i46-review-jev/README.md). Their agreement supports investigating the approach but does not replace the executable regressions. I19 work in the worktree is separate from this correction.

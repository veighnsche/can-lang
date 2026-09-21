# I07 validation

Completed 2026-09-21. Current AST expressions now check into typed IR and lower to
ordered native TypeScript statements. The pass covers primitive operators,
comparison chains, array/record construction and updates, ordinary fields,
length, indexing, bracket slicing and array/string `.slice(int,int)`.

| Acceptance requirement | Evidence |
| --- | --- |
| Native bigint/float/bool/text operators | Generated programs execute arithmetic, shifts/bitwise operations, text concatenation and bool logic with native operations. C10 values above 2^53 remain exact. |
| Float and aggregate equality | Scalar float uses Object.is; aggregate data uses strict Bun.deepEquals. Generated tests cover NaN, signed zero, arrays and ordinary nominal records. Callable/opaque containers reject statically. |
| Left-to-right, once-only, short-circuit evaluation | Event logs prove reached operands execute once in written order. Logical branches and later comparison operands are skipped. Skipped zero-divisor/index faults do not run, but static errors in skipped branches still reject. |
| Exact static operand types | Mixed int/float, non-bool conditions, invalid ordering, wrong index/slice types, unequal nominal owners and implicit existing-array widening reject. Expected variant array construction is accepted. |
| Safe bigint index conversion | Bounds are checked against bigint native length before Number conversion. Negative, boundary, >2^53 and huge indices classify as bounds failures; no element is synthesized. |
| Native slicing and UTF-16 | Huge bounds normalize before narrowing; native slice owns copying and strings retain code-unit behavior. Astral pairs and individual surrogate halves agree with native access. A finite cross-product compares array/string slicing directly to native slice. |
| Arithmetic failure classification | Division/remainder by zero and negative integer exponents carry private thrown-object kind/message identity with the exact C9 fixed messages. Native engine exceptions are left for I49's native_exception boundary. |
| Later async integration | The emitter returns statements/result, not hidden async closures. An executed awaited boxed-call hook preserves comparison order and skipped operands inside an enclosing async region. |

Validation artifacts:

- [Source-check/IR/emitted-program tests](i07-expression-tests.txt): passed with
  an absolute Bun path whose binary hash, OS and architecture match the target.
- [Native primitive tests](i07-runtime-tests.txt): 4 tests and 916 expectations,
  including direct native slice comparisons and unforgeable fault classification.
- [Full Go suite](i07-go-tests.txt): passed with emitted execution enabled.
- [Staged offline integration](i07-offline-integration.txt): the packaged primitive
  tests pass using the absolute staged runtime with network denied and unavailable
  PATH. Earlier source/bundle hash guards and negative sidecar checks also pass.
- [Three Jev consultations](i07-jev/README.md) advised the statement representation;
  executable evidence independently verifies the choice.

A test fixture was corrected during development: Can retains unknown `\u` escape
text, so the UTF-16 ordering test now uses the actual Unicode scalar in its source.
No language escape rule was changed to accommodate the test.

The owning body pass supplies resolved values/functions/constructors. Full generic
call inference, captures, methods, spread/state argument handling and completion
regions remain their scheduled tasks; unsupported forms fail rather than falling
back. I49 owns full standard-failure occurrence/cause reporting, I09 publishes
whole generated modules, and I22 completes the numeric/conversion catalogue.

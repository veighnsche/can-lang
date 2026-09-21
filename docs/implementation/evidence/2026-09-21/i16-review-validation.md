# I16 independent review corrections

Five reproduced frontend defects are corrected before I17 lowering:

- Descriptor preparation checks metadata availability in the inherited checker
  context. Nested calls, criterion and threshold expressions, and implicit near
  captures cannot read answers before a response exists. Handler metadata stays
  available. Regressions require the phase diagnostic, not merely any rejection.
- Exported native signatures require an exported connection. Tests cover fetch,
  LLM, Noul and judge, with public-connection positive counterparts. The current
  exported Choice example now explicitly exports its classifier connection.
- Resolver bindings for variadic given inputs use the packed array type, matching
  the checked signature. A generated Choice spreading `choices[0]` from
  `arms ...choices` retains both arm fields in order.
- Probability expressions are leaves in source-use traversal. A handler can bind
  `float probability = %` and immediately return it. The contextual initializer
  remains excluded from the finite unnecessary-local rule.
- Every judge/LLM state field passes shared sealed codec schema admission.
  Regressions reject opaque bytes, callable and arm fields, including opaque data
  nested in records and arrays. Ordinary admissible state remains accepted.

[The full Go suite](i16-review-go-tests.txt) passed with the qualified Bun.
[The staged native frontend test](i16-review-offline-tests.txt) built a fresh
release, then invoked its absolute compiler with a nonexistent PATH and network
denied. The expanded positive fixture includes a probability local and variadic
generated spread; additional negative builds cover private connections, nested
metadata reads, and opaque state. No credentials or provider execution are needed.

These fixes preserve I16's frontend scope. I17 remains incomplete until the real
judge lowering and provider-boundary tests are implemented and verified.

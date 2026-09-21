# I21 compiler integration consultation

Three fresh requests compared explicit checked array invocation IR with reuse of
the synchronous Native-expression ABI and synthetic source wrappers containing
placeholder types. Every explanatory context, question and option was rewritten;
the pre-dispatch audit records their semantic equivalence and wording differences.
All requests and raw responses are retained here.

All three Jev responses selected `checked_array_ir` with probability/confidence
1.0 (model `jev-1.13.0`). There was no disagreement to investigate. Agreement was
used as advice, not proof; exact callback types/error bounds, capture timing,
fixture ordering, nominal option construction and generated execution were
validated separately in the compiler, runtime and staged integration tests.

The implementation reuses sealed types and finite generic specialization. It
adds concrete callback-input/result constraints and checked operation descriptors,
not a new inference type or a fabricated authored declaration. The HTTP API and
Choice documentation previously read for I18 were reused for these requests.

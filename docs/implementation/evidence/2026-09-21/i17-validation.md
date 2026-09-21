# I17 Noul vertical slice

The compiler now retains checked descriptor computations and judge phases, emits
Noul preparation and ordinary handler functions, and executes one batched request
through the maintained TypeSafe adapter. Connections use the launcher's original
environment snapshot; no credential is compiled into the program.

Preparation evaluates registration arguments, instructions, descriptions in
source order, and the threshold once. The complete batch and state pass admission
before authentication. Repeated question declarations remain distinct `q0`, `q1`,
etc. The shared codec formats the complete request, including exact bigint state
and float negative zero. Request encoding byte overflow maps to 1104; other codec
failures remain 1110. Native JSON parsing, UTF-8, duplicate detection and resource
preflight are shared with typed decoding.

Response admission checks the envelope and full ID set, then every answer in
registration order. No handler runs until validation succeeds. Generated code
uses `p >= minimum`, passes the original `p` to either handler, stops on the first
failed completion, and enters the continuation only after all handlers succeed.
Every asynchronous boundary returns a private Completion box.

Evidence:

- [Full runtime suite](i17-runtime-tests.txt): 118 tests, 1,844 expectations.
  Includes exact raw body tests, boundary/range/Unicode/duplicate refusals,
  ID-first validation, once-only credentials, no retry for 401/422/429/529,
  and ordinary assertion refusal of an unprovided live boundary.
- [Full Go suite](i17-go-tests.txt) with the qualified Bun passed.
- [Full staged integration, driver and emitter suites](i17-offline-tests.txt)
  passed, including sidecar integrity and source-map verification.
- [Final judge integration](i17-judge-final-tests.txt) reran after adding the
  example's supplied fixtures. A fresh staged release, absolute bundled runtime,
  nonexistent PATH and loopback-only network policy execute current Can code.
  Exact request bytes contain `9007199254740993` and `-0`. Output `TFT` proves
  threshold equality, unchanged false-handler probability, registration order
  and execution of an unused binding. A malformed final answer runs no handlers;
  an invalid descriptor sends no request. First-handler failure prints only `T`
  and neither starts later handlers nor redispatches.
- The same final integration runs a compiled main/real judge with a private raw
  HTTP fixture and **all networking denied**. Its report contains
  `raw-provider-fixture` and `real-can`, with no violations. The fixture compares
  method, URL, headers and exact encoded bytes, then feeds native Response bytes
  through the real adapter. This is separate from the example's ordinary
  `supplied-completion` assertions. Runtime regressions reject mismatched raw
  requests and leftover rows.
- [Strict TypeScript](i17-typescript-tests.txt) checks all modules of the emitted
  final example; strict checks also passed for the new runtime tests.
- [Three design consultations](i17-jev/README.md) advise explicit phase IR; tests,
  not that agreement, establish behavior.

Provider field spellings were checked against the live TypeSafe
[API](https://docs.typesafe.ai/api) and [Noul](https://docs.typesafe.ai/primitives/noul)
documentation. Local compatible responses establish protocol compatibility only.
No live-provider quality claim is made. Choice/Score lowering and general fixture
coordination remain their separately tracked tasks.

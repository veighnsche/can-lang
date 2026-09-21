# I08 validation

Named callable references now resolve eligible declarations, capture receivers
and exact-name near bindings once, derive sealed residual contracts, and emit
fresh native closures. Captures retain immutable identity; invocation forwards
the caller's assertion context and protected completion ABI.

- [Full Go suite](i08-go-tests.txt) passed.
- [Native runtime tests](i08-runtime-tests.txt): 43 passed, 1,250 expectations.
- [Strict TypeScript](i08-typescript-tests.txt): runtime/tests and the staged
  generated entry/assertion graph passed.
- [Offline staged suites](i08-offline-tests.txt): integration, driver and emitter
  passed, including absolute release invocation with network denied and no PATH
  runtime fallback.
- [Staged assertion report](i08-assertions.json): all 11 source assertions passed;
  the staged entry also completed with status zero and no output.
- Checker negatives cover missing/wrong captures, wrong input/result, widened
  actual errors, missing direct-call near inputs, and ineligible direct native
  references. Ordinary function/fetch reference bodies are admitted using checked
  symbols; question/judge/LLM reference bodies are refused.
- Native emitter tests prove different captures at one creation site produce
  distinct functions and stable later results, receiver evaluation occurs once,
  immutable receiver identity survives, and invocation uses the caller context.
  Dynamic callee evaluation precedes arguments, failure stops later work, and an
  explicit `then` callable field is invoked without promise assimilation.
- Packaged I/O wrapper assertions fail closed without a fixture. Replacing the
  invocation with ordinary completion passes, proving construction is inert.
- [Three fresh design consultations](i08-jev/README.md) support the closure choice.

The private callable receipt retains site/target/capture identity for later I18
fixture matching. Full fixture scheduling is not part of this task. Native
provider declarations/execution remain I16 and subsequent transport/AI tasks;
static eligibility evidence must not be read as provider execution evidence.

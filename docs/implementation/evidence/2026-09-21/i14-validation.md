# I14 validation

The typed codec uses native parse/source-context and stringify/rawJSON around
finite checked schema graphs, bounded numeric normalization and limited resource
and duplicate-key scanning. Explicit concrete codec calls and named references
execute in current Can source; ordinary generic inference remains I46.

- [Full Go suite](i14-go-tests.txt) passed after review corrections.
- [Native runtime tests](i14-runtime-tests.txt): 61 passed, 1,485 expectations,
  covering the complete runtime and
  the codec regressions, including exact public limits.
- [Strict TypeScript](i14-typescript-tests.txt) passed for runtime/tests and the
  freshly generated staged entry/assertion graph.
- [Offline integration, driver and emitter suites](i14-offline-tests.txt) passed.
  TestCurrentBundledCodec runs the absolute release with denied network,
  nonexistent PATH and an outside working directory.
- [Staged source assertions](i14-assertions.json): ten passed. They include a
  huge-int nominal record round-trip, fractional/quoted integer refusal, a
  specialized codec callable, canonical option tag encoding/decoding and private
  tag refusal. The entry completes with status zero and no output.
- Native tests preserve 9007199254740993, integral fraction/exponent spellings,
  float negative zero, and integers whose native Number value overflowed. They
  reject fractional underflow and huge exponents before expansion. Signed and
  cumulative integer byte budgets are exercised independently of input size.
- Exact 8,388,608-byte, depth-64 and 1,000,000-node values encode and decode;
  limit-plus-one fails. Tests also cover escaped-string/punctuation byte counts,
  preexisting huge bigints, malformed UTF-8, BOM, lone surrogates, duplicate escaped
  keys, missing/extra fields, canonical variant tags, shared subtrees and cycles.
- A shared schema/codec boundary test encodes AI state into a native Request,
  decodes a raw fetch Response, and decodes generated LLM text, preserving the
  same huge integer under a replaced private byte budget. It exercises raw
  boundary payloads, not the later provider envelope/transport policy tasks.
- Static tests refuse opaque/callable nested contracts, missing/excess codec
  type arguments and wrong input types. Recursive schema graphs remain finite.
- [Three fresh design consultations](i14-jev/README.md) support the architecture;
  the tests, not agreement, establish behavior.

Independent review found three defects before commit, all corrected:

1. Wire tags had used private declaration IDs. Type nodes now retain source
   package/declaration names separately; real-builder and staged generic-option
   regressions prove canonical tags and reject private IDs.
2. Duplicate diagnostics lost nesting. The scanner now retains the duplicate
   key's escaped path, including array indexes, while malformed native syntax
   still takes precedence.
3. A revoked array proxy escaped as TypeError. The array predicate now passes
   through the guarded data check and produces codec type failure without traps.

Unrelated native exceptions remain standard failures. Syntax-only catches defer
only SyntaxError; no fallback parser or alternate precision path is introduced.

The public-limit stress test uses a 30-second test allowance because one run of
its million-node formatting work exceeded Bun's default five-second test timeout.
All assertions completed; a rerun passed in the recorded full suite. This is a
test harness allowance, not a new codec timeout or hidden cancellation policy.

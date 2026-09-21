# I40 — source map encoding and runtime diagnostics

I40 is complete. Go supplies structured checked-span segments; bundled upstream
`gen-mapping` encodes adjacent maps and bundled `trace-mapping` consumes Bun's TS
stack coordinates. The implementation and source-content policy are described in
[source maps](../../source-maps.md).

- **P+:** the offline packaged test executes a nested generated Can call and maps
  its native bounds failure to `main.can:10:15`, byte interval `[177,186)`, with
  a second caller location. The expression follows an emoji, so byte offsets and
  UTF-16 columns differ. The [actual report](i40-runtime-report.json) contains
  neither native helper paths nor source/application values.
- **N−:** absent maps/index, corrupt mappings, unknown source IDs and helper
  execution failure cannot replace the current generation. Upstream canonical
  re-encoding also refuses malformed VLQ, unexpected map fields, source content,
  mismatched filenames/segments and duplicate or unordered generated positions.
  Stack accessors and proxy causes execute no getters or traps. The compiler's
  private mapping tokens cannot be injected by quoted source strings, and origin
  instrumentation cannot alter origin-like text in authored literals.
- **INT:** a staged absolute private Bun executes the packaged compiler/program
  with network access denied. Existing offline CLI, assertion and sidecar suites
  still pass. Separate direct-TypeScript native tests prove composition through
  Bun's transpilation for nested awaits and async callbacks, Unicode columns,
  runtime-helper throws and synthetic-origin fallback. These qualify the mapping
  consumer; they do not claim completion of I08's authored callback lowering.

Evidence:

1. [Full Go suite](i40-go-tests.txt), with `CAN_BUN` selecting the pinned executable.
2. [Native runtime/tool tests](i40-runtime-tests.txt): 40 tests, 1,217 expectations.
3. [Strict TypeScript results](i40-typescript-tests.txt): fresh emitted entry/import
   graph, maintained runtime and tests, and source-map tools; TypeScript 7.0.2.
   Broad checking exposed two existing test-only typing issues; explicit array
   typing and a callback presence assertion now make the whole selection pass.
4. [Offline staged suites](i40-offline-tests.txt): integration, driver and emitter.
5. [Versioned native mapping qualification](i40-qualification.json), separate from
   language/assertion evidence and provider-quality claims.
6. [Jev requests, responses and decision review](i40-jev/README.md).
7. Exact package archives, registry revisions, integrity/SHA-256, bundle hashes and
   license hashes: [upstream lock](../../../../distribution/notices/source-maps.lock.json).

Bun 1.4.2 does not automatically apply the external Can map during direct TS
execution. Explicit tracing is required. Its `.then(async …)` stacks can omit
outer callers; the implementation retains actual mapped frames and does not
invent missing ones. Source content is omitted from maps; generated code still
contains authored literals by necessity. Default diagnostics omit raw stacks,
messages, absolute paths and failure payloads.

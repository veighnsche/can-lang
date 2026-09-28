# Retain only consumed JSON numeric evidence

Production scope: runtime/codec/document.ts, the shared native JSON.parse path for
JSON/JSONL, raw assertion bodies and maintained AI protocol readers. Its reviver
currently allocates holder Maps and entries for strings, booleans and null whose
source spellings have no consumer. Keep native JSON parsing and duplicate scanning;
only gate source-token insertion on typeof value === "number". All numeric tokens
must remain verbatim, including overflow to Infinity (1e309), unsafe integer
rounding, exponent spellings and negative zero. Do not use Number.isFinite.

Complete caller audit: codec/json.ts and jsonl.ts pass tokens to exactInt in
codec/project.ts, which requires a number before token access; assert/provider.ts
jsonTokensEqual accesses source spelling only in its numeric comparison branch,
using Object.is for other scalars; ai/questions.ts, responses.ts and budget.ts
use parsed only. rootHolder assignment stays unconditional on key === "";
empty names in nested holders and roots must still end at the native root receiver.
Keep native receiver/key identity, reviver traversal, syntax-before-duplicate
refusal, duplicate paths, UTF-8/Unicode, schema projection, integer/byte/node/depth
budgets, failure occurrence/origin and frozen outputs. No caller mode/new API,
parser rewrite or compatibility scaffolding. Zero external users.

Three fresh fully reworded equivalent Jev requests/responses are saved under
.performance/performance-push-20260928/generated-numeric-retention-{request,response}-[1-3].json.
All choose numeric_only (.99/.97/.97), with separately saved equivalence and
reconciliation. Advice only; the actual consumer proof and correctness determine
acceptance. Removing unread retained entries is source-proven; latency and memory
bytes are unmeasured. No measurements, harness changes or auxiliary hardening.

One small functional before/after probe may use a fixed string-heavy object with
40 strings,10 booleans,10 nulls and4 original number spellings, recording parsed
values and holder token-entry counts. This is bookkeeping evidence, not a timing
or heap benchmark; do not add instrumentation/counters to production. Meaningful
regressions cover numeric roots, overflow, unsafe integers, negative zero, arrays,
empty/repeated names on separate holders, scalar parsing and rejection precedence.
Reuse existing typed codec, JSONL, raw-provider and AI contracts.

A separate narrow test-only correction closes the independently reproduced stale
browser owner-export inventory: both owner implementations already export
bindNativeCallback and match HEAD. Update the expected surface and verify its
browser ambient refusal using a real callback. Preserve the production owner API;
no backwards compatibility requirement exists for the obsolete eighteen-name list.

Muse alone owns implementation/tests/progress, sequentially. Required runtime
lint-fix/format/check and focused consumers must pass. Reuse successful generated
strictTS/24 oracle evidence from checkpoint52ae03f0 if compiler/module/driver and
all generated fixture-reachable runtime sources are unchanged: its emitted fixture
does not import codec/document.ts. Explain reachability before deciding; build only
one reused bounded actual graph if a new relevant contract needs it. Do not repeat
startup tool tests, timing or a full-system audit. Register/retire exact temporary
resources and keep compact source/command/result evidence. After ALL-writer
handoff Codex independently reviews and commits; invoke/startup/final twelve-slice
exhaustion review remain open.

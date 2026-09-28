# Audited text, byte and check context bypass

Collection context bypass is independently accepted9bdc7fe5. The next production
candidate removes the same unused assertion callback, private receipt lookup and
async callContext adoption at positively classified text/byte/check sites. Actual
paths.can/recovery.can invoke trim, to_lower_case, normalize_nfc, from_utf8,
encode_base64 and require. No latency/heap claim, timing or auxiliary capability.

Source audit: runtime/text.ts createText exposes16 own native async methods;
runtime/bytes.ts createBytes exposes9; runtime/checks.ts createChecks exposes1.
All three APIs are Object.freeze objects. Each audited method returns a genuine
native promise with a branded Completion box or rejects; hostile payloads stay
boxed. These are adapters over existing native algorithms and immutable storage,
which this patch does not change. program_state.go pairs the exact receivers with
these factories; factory initialization and Intl.Segmenter remain unchanged.

Exact finite canonical bindings are saved with current source hashes in
.performance/performance-push-20260928/generated-core-context-source-audit.json.
They are extracted from runtime_core.go and runtime_utilities.go:16 text,9 bytes,
1 checks (26 total). Text includes intrinsic string operations and canonical
Unicode/regex operations; bytes includes UTF8/integer and base64/hex adapters.
The proof covers only this explicit table. Other factories, numeric helpers,
specializations and all mixed-Functions targets are not inferred eligible.

Add a separate private coreAsyncProof to programAssembly, built after final
contribution merging from the finite exact identity/member table. Require the
final functions[canonicalIdentity] to equal the audited target. Missing, empty,
unknown, rebound or cross-receiver pairs cannot qualify. Share read-only with the
three assembly-created RegionEmitters as coreProof. Direct emitters default nil.
Do not add any entries to authoredProof/collectionAsyncProof or forwarding.
The finite table/helper belongs in runtime_core.go without changing operation
bindings or introducing a runtime classifier/cache/public surface.

Reuse existing plain Bun exclusions: Browser false; nonnil direct step; nil
Callee/Native/Array/Asset/Fixtures/SQL/FormAction/JSONFetch/Action. Identity and
resolved target must match the assembly proof. OR the positive predicate into
the established strict $canContext === undefined selection. Reuse the original
receiver.method expression inside unchanged outer invoke/thunk; defined context
keeps original child-frame arrow/private receipt path. No eager method lookup.
Checks require already appends its hidden exact authored origin before trailing
context; preserve that argument and all original lowering/fixture behavior.

Keep native algorithms/immutability, completion branding/admission/caught and
boxed hostile-payload safety, exact first call-site boundary/fresh occurrence,
argument evaluation once/in order, receiver and context/owner/lease/cancellation/
assertion contracts. No identical microtask-count promise is added; established
observable ordering remains required. Current proof is a compile-time audited
factory contract, not blanket trust in prefixes, Promise annotations or names.

New tests establish all26 canonical assembly pairings, meaningful missing/stale/
rebound/other-factory/indirect/special/browser negatives and actual installed Bun
frozen-own-async premise. Execute positively eligible emitted text/byte/check
callers through real factories under absent/defined context, once/ordered operands
and balanced frames. Use genuine text/bytes/check success/domain refusal and
native async standard rejection with exact authored call-site source/span and
fresh first-boundary occurrence; exercise hostile boxed payload without accessing
then/getters through an actual eligible native async method. Reuse unchanged
qualified generic invoke/owner/lease/authored/collection contracts; avoid duplicate
large suites, synthetic manual-invoke or function-entry boundary surrogates.

One bounded reused actual emitted validation graph qualifies changed modules:
13 gates/AST/mappings, strictTS,24 generated/native cases without sampling,14
actual binding entries retained before cleanup, full source/driver/module/runtime/
dependency identities and exact groups/graph retirement. Frozen accepted tooling
stays unchanged; no224 auxiliary suite, harness repair or measurement. Runtime/
factory/import/default/public/driver/vendor changes are outside this packet.

Three fresh fully reworded equivalent generated-core-context request/response
consultations select core_proof1/1/1, with equivalence and reconciliation saved.
Confidence is advice; source and executable acceptance decide. Current TypeSafe
[API](https://docs.typesafe.ai/api.md) and
[Choice](https://docs.typesafe.ai/primitives/choice.md) contracts were read.
Codex owns design and acceptance; sole SAME Muse Contributor/MAX/scopedYOLO owns
implementation and tests. Sequential coordinator; no extra agents are needed.

After acceptance continue startup/other native candidates and independent final
twelve-slice review. This packet is source-supported work removal; no speedup,
heap-byte or p95 conclusion follows merely from correctness or site counts.

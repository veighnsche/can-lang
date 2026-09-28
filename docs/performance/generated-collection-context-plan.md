# Checked collection async context bypass

Previous authored packet independently accepted5380d94a. Native collection calls
still build an assertion arrow, read a private callable receipt and enter async
callContext even when strict context is undefined. The frequency workload has
actual empty/get/insert/replace sites. This packet removes that unused machinery
only for positively classified checked map/set bindings, inside unchanged invoke.
No latency/heap/p95 claim or new measurement/harness capability.

Source evidence: runtime/collections/map.ts createMap and collections/set.ts
createSet each publish Object.freeze APIs with native async methods returning
genuine boxed Completion or rejecting on thrown failures. Native Map/Set
algorithms, immutable history and branded backing remain. Compiler
check/collections.go records concrete specialization key and canonical Operation,
Collection/Entry/Contract; emit/runtime_collections.go pairs it with deterministic
$canCollectionN.method and program_state.go initializes the same factory selected
by Entry!=nil (map) or nil (set). Methods have immutable own properties, and
calling the retained receiver.method expression preserves the method receiver.
Before initialization an undefined factory still fails through the retained
invoke boundary; do not introduce eager factory lookup or delayed initialization.

Finite canonical operation/factory/method proof:

| Factory | can.std.collections@1 operation suffix | Method |
|---|---|---|
| map | empty_map | empty |
| map | build_map | build_map |
| map | get | get |
| map | insert | insert |
| map | replace | replace |
| map | remove | remove |
| map | entries | entries |
| set | empty_set | empty |
| set | build_set | build_set |
| set | contains | contains |
| set | add | add |
| set | union | union |
| set | intersection | intersection |
| set | difference | difference |

Assembly builds a SEPARATE private collectionAsyncProof after final contribution
merge. Iterate actual checked program.Collections entries; require recognized
canonical operation and correct Entry-based factory kind, concrete collection
and factory binding, and exact final functions[specializationKey] equality to
collectionNames[Collection.Identity()]+audited method. Missing, empty, unknown,
mismatched kind/target/receiver cannot qualify. Share this finite map read-only
with assembly region emitters; direct emitters receive nil unless explicit tests
supply evidence. Do not populate authoredProof or change callable forwarding.
No name-prefix/Promise-type/whole-Functions inference or runtime mutable cache.

Reuse plain Bun direct-step exclusion: nil Callee/Native/Array/Asset/Fixtures/SQL/
FormAction/JSONFetch/Action, Browser false. Collection proof additionally requires
exact step.Identity+resolved target. Eligible undefined context selects original
direct receiver.method(args...,$canContext) inside the same outer invoke/thunk;
defined branch stays byte-identical original callContext+child arrow+receipt.
Unknown/indirect/special/fixture/browser and other native operations retain
existing behavior. Authored bypass stays unchanged. Configure ExpressionEmitter
Call and all runtime/factory/public/shared-driver implementations are outside
scope. Removing async adoption does not promise an identical microtask count;
actual ordering/owner/context/lease/cancellation contracts stay required.

Tests must establish actual assembly classification plus frozen native async
factory premise, actual emitted map/set calls and negative proofs, defined
context/frame balance, operands once/order, native immutable success/domain
refusal/standard first boundary/fresh occurrences and boxed hostile payloads.
Reuse qualified generic invoke/thenable/owner/origin and authored tests where
unchanged rather than duplicating suites. One bounded current emitted graph
strictTS/mappings/24 native oracles is needed because frequency body changes;
retain actual14 bindings/file/content/input/driver identities and eligible-site
counts. Runtime/source copies, cache installs, new harnesses and timing are
excluded; exact owned groups/scratch retire after checks.

Three fresh fully reworded equivalent generated-collection-context requests/
responses and equivalence/reconciliation select collection_proof .98/1/.93.
Confidence differs; advice is not proof. Source/code contracts and independent
verification decide acceptance. Live TypeSafe API/Choice docs were read via
urllib after web access failed. Codex owns design/review, Muse all changes/tests.

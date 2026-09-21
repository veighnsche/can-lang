# I18 callable creation identity consultation

Three fresh requests compared construction-time receipts, first-use reminting and rejection of initialization-created callables. Every explanatory context, question and option was rewritten and audited before sending while preserving the same facts and alternatives. All three responses selected `construction_receipt` at probability/confidence 1.0; no disagreement occurred. Agreement is advice, not proof.

`ownCallable` now records creation when the native closure is constructed. Inside an assertion, the identity includes the root and parent invocation path; using that receipt under another assertion rejects. Outside assertions, pure initialization has no root, so an immutable root-neutral receipt retains lexical site and creation occurrence and may be embedded in independent root-local invocation paths. Neither receipt owns queues or scheduling state. Frozen receiver/near capture references remain private; diagnostic hashes encode creation coordinates rather than serializing captures.

Runtime tests prove distinct creation occurrences, frozen capture aliases, distinct roots retaining the same initialization receipt, and rejection of dynamic cross-root reuse. Staged generated tests cover named callable invocation, bound near values, different receivers and spread callables. A negative staged test requires the callable creation fingerprint to appear in the full invocation path. These checks provide implementation evidence beyond the classifier result.

The HTTP API and Choice documentation read for the parent I18 consultation were reused. Requests specify `jev-latest`; responses identify `jev-1.13.0`. Raw requests, responses and the wording audit are retained here.

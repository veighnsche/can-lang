# Shared capability decision evidence

Status: source review and design only. No compiler/runtime implementation, test execution, browser/database session, benchmark or reference qualification was performed.

The [main contract](../../native-can-test-capabilities-2026-09-30.md) defines outside-worker resource ownership, factual typed operations, explicit partial/indeterminate results and Can-owned scenarios/oracles. Compiler diagnostic observation uses a proposed general product CLI.

## Source reviews and reconciliation

Three parallel reviews informed the contract:

- [Workspace/process](workspace-process-review.md): current native process spawning and file operations provide useful mechanics, but cleanup registered inside a killed worker cannot reclaim resources. Registration before exposure, private descriptors, bounded streaming, partial delivery and outside-worker recovery need explicit contracts.
- [Browser](browser-review.md): controls need independent IDL-property/attribute reads, physical versus synthetic actions, delayed interception and raw network facts. Shared remote Firefox ownership ends at the connection/context. Current routing/logging is not proof of pre-contact network isolation.
- [Native observations/diagnostics](observation-diagnostics-review.md): preserve raw identity, signed zero/source tokens, hostile callback counts and independent database/wire facts. Current inert `CheckSnapshot` is useful, but folded project-load errors cannot all be treated as expected source rejection.

The main contract resolves several ambiguities in these exploratory notes:

1. A quiet interval is only a prefix checkpoint. Final browser evidence requires stopped/closed activity, drained tracked callbacks and an explicit completeness scope. Missing callbacks, gaps or driver death cannot produce a clean empty error ledger.
2. Upstream fetch and response delivery are separate operations. Once upstream was fetched, continuing the original request is forbidden because it could repeat the write. Can chooses whether/how to fulfill from the recorded upstream response.
3. Synchronous getters/Proxy traps cannot wait for remote Can decisions while preserving synchronous semantics. Fixed immediate descriptors are allowed; asynchronous continuation points can use gates. Richer synchronous logic executes as ordinary Can compiled into a reviewed local probe artifact with a suitable binding.
4. Input API seals and read hashes are not filesystem isolation. Report protection strength; a stronger required guarantee needs host enforcement, not before/after-hash inference.
5. A pending process wait/read cannot block signaling, writing or cleanup through a coarse per-handle lock. Serialize state transitions, not the entire lifetime of every operation.
6. Completed rejection may legitimately stop at parse or resolve failure. It must establish a real attributed source error and known inputs; it need not claim later phases ran. Infrastructure cancellation, unavailable inputs, panic and report truncation remain failures.

The supporting notes preserve source findings and preliminary vocabulary. The main contract governs observable semantics; no proposed operation is an implemented API merely because it has a name.

## Jev consultations

The repository requires three fresh consultations on difficult design choices. All context prose, question instructions and option descriptions were rewritten for each request, holding facts/constraints/alternatives stable. Exact technical identifiers and option keys remain stable. Semantic equivalence and full-field wording differences were checked before sending. No previous response was provided to a later request. Requests used `jev-latest`; all responses identify `jev-1.13.0`.

| Question | Selected alternative | Option probability, requests 1 / 2 / 3 |
| --- | --- | --- |
| Resource authority | Outside-worker native owner with reservation, bounded facts and recovery | 1.00 / 1.00 / 1.00 |
| Observation boundary | Typed individual native operations/facts, composed and judged in Can | 1.00 / 1.00 / 1.00 |
| Compiler diagnostics | General structured product CLI, invoked and interpreted by Can | 0.98 / 0.99 / 1.00 |

These are returned option probabilities, not confidence intervals or measured correctness.

- [Request 1](request-1.json), [response 1](response-1.json), [transport metadata](response-1.metadata.json).
- [Request 2](request-2.json), [response 2](response-2.json), [transport metadata](response-2.metadata.json).
- [Request 3](request-3.json), [response 3](response-3.json), [transport metadata](response-3.metadata.json).
- [Pre-send wording audit](wording-audit.json), [consultation script](consult.py).

No selected alternative differed. The first two responses assigned 0.02 and 0.01 respectively to a compiler-specific native binding. That alternative still needs explicit candidate identity, isolated execution, a transport and source-versus-infrastructure classification. A public structured check command also serves editor/CI/product consumers and composes through the same process capability; no evidence here justifies adding a new language primitive first. Textual stderr remains useful context but does not supply stable producer phase/completion semantics.

Worker-local cleanup and later discovery scans cannot close the acquisition/death window on their own. Native ownership remains external, with unsupported host guarantees exposed honestly. JavaScript source evaluation and whole-scenario adapters would retain foreign-authored scenario policy, contrary to the agreed endpoint. Typed observation costs a larger reviewed binding catalogue; coverage mapping and ordinary Can probes must demonstrate sufficiency rather than asserting that five samples cover the ledger.

Jev classified the supplied facts/alternatives; it did not research the repository or validate the concrete protocol. Agreement and rewording do not prove bias removal, enforcement, correctness or optimal agent authorship. The [TypeSafe API](https://docs.typesafe.ai/api) and [Choice contract](https://docs.typesafe.ai/primitives/choice) were checked earlier in this ongoing design session. Credentials came from the environment and were not saved.

## Verification scope

The final [review record](review.md) distinguishes resolved contract findings from pending implementation proof. The [validation record](validation.json) records document links, consultation shape/wording checks and source fingerprints. It is not execution evidence. No temporary execution workspace, build, browser, database or worktree was allocated; retained material is compact design/consultation evidence.

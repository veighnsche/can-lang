# Source challenge review and reconciliation

This round used parallel source review for browser, unusual native values and SQL, with a separate descriptor/launcher review. The main design and experiment register then received independent reviews by those source reviewers. No runtime test or feasibility experiment ran.

Resolved findings:

1. **Browser phases:** separated target absent before submit (conditional no-request or request-phase `effect=none`) from removal after request (response-phase uncertain occurrence). Clone identity/remount is retained, not replaced with reattach.
2. **Browser ordering:** specified route progress during an unsettled input; same-task batches need event-realm delivery/microtask markers, not driver labels or request count alone. Cancellation observation needs a same-node bubble listener after the application handler with readiness/registration barriers.
3. **Browser effects:** remove or consume the corruption route before replay; distinguish upstream contact and delivered bytes; use an independent request trace because idempotency can hide duplicate dispatch. Page Fetch qualification includes an owned CSP-forbidden origin and no-contact witness.
4. **Shared Firefox:** the case cannot obtain server authority. Reject a server-close request before effect, preserve N-pinned service identity and verify a second owned context. Do not terminate the user's shared server as a negative control.
5. **Native identity:** compare references within the owning realm and preserve occurrence provenance. Copied IDs cannot prove identity. Keep raw-Bun, R-probe and C-adapter evidence separate.
6. **Linking controls:** R import-integrity rejection is a distinct control; it does not prove C ingress ran. Require actual C adapter/artifact witnesses. A forged completion is locally constructed in the subject realm, with no assumed cross-runtime completion ABI.
7. **Failure role:** judge-worker death leaves incomplete evidence; expected C-subject death can be judged by a surviving R worker. It cannot replace the normal late-event leg.
8. **Descriptor EOF:** snapshot-reading children need EOF before readiness. The unused large snapshot deliberately need not be drained; its writer must unblock and be reaped after the child exits.
9. **SQL cancellation:** current budget machinery does not cancel native SQL and rejects cancel signals. D4 deliberately observes continued work; unsupported cancellation is recorded, not used as a prerequisite that makes the question impossible to test.
10. **SQL quiescence:** driver promise settlement/connection closure may leave a server effect unknown. Require server acknowledgment or qualified fence/reconciliation before treating final rows or namespace cleanup as conclusive. Preserve unresolved state if that cannot be established.
11. **Probe bounds:** include tiny fixture preparation inside each total job deadline, make engine variants separately admitted, account all sublegs and cleanup, and use a two-connection actor-delay design rather than an unbudgeted third SQL lock holder. Resource ceilings remain unmeasured policy and require admission.

The original source notes remain evidence/exploration; the main challenge and central experiment register govern final proposed scope. Existing capability, architecture and lifecycle documents link to the challenge and carry its key corrections. Historical consultation requests remain immutable evidence of the facts available when sent; later review adds precision without claiming those responses proved it.

Validation checks local document/source links and source fingerprints, all 14 experiment entries/deadlines, three saved request/response pairs and distinct explanatory prose, JSON structure and edited-document whitespace. It cannot establish the feasibility of proposed APIs. See [validation.json](validation.json).

No runtime/compiler source was edited, no temporary execution workspace/worktree was allocated, no browser/DB service was started, and no measurements ran. There is no experiment scratch to clean. Existing unrelated `AGENTS.md` changes, older design artifacts and `output/` were left in place.

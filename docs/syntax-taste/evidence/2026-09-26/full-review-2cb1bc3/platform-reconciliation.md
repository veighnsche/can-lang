# Platform review reconciliation — current Linux target and action cancellation

This reconciles the independent first-pass report `/tmp/can-review-20260926-platform.md`. The first pass remains unchanged for auditability. This follow-up uses current implementation/source only; no prior review outcomes or execution evidence was used. No implementation changes or full-suite rerun.

## Linux correction

**Retract any inference that Can has no Linux/container path.** The root README and general distribution README carry stale “no Linux support” text; the target-specific implementation provides a real Debian/amd64 lane. I gave those broad README statements too much weight in the first-pass delivery section.

Current implementation evidence:

- `distribution/target-linux-amd64.json:3` pins the target identity `bun-1.4.2-linux-amd64-v1`; `:8` pins Linux/amd64, minimum OS version 13, executable and digest. The required native APIs and behavior probes are explicit (`:26`, `:44`).
- `distribution/manifest.go:196` returns the embedded Linux target, `:207` recognizes Linux/amd64 as a supported host, and `:213` selects the Linux target on that host. This is executable implementation, not merely a proposal.
- `distribution/os_linux.go:25` and `compiler/internal/driver/os_linux.go:27` require `ID=debian`; the target manifest carries the minimum version. `distribution/verify.go:112` verifies an x86-64 ELF sidecar. The admitted Linux contract is narrower than arbitrary Linux, Alpine/musl, Ubuntu, or ARM64.
- `distribution/linux/run.sh:22` verifies the archive against provenance, `:34` builds the pinned Docker image, `:36` releases the target bundle, `:42` runs installed-artifact smoke with networking disabled, and `:48` runs a digest-pinned PostgreSQL container leg. Artifact/report locations are `out/linux` and `out/linux-work` (`:18`).
- `distribution/linux/build.sh:17` checks archive size/digest against the actual target, `:35` builds/releases using distbuild and copies release files out, and `:39` records the build environment.
- `distribution/linux/smoke.sh:26` installs into a fresh root, `:32` checks sidecar runtime identity, `:45` runs native qualification, and `:81` onward checks process/files/crypto/SQLite applications plus repeated-build identity. These are concrete implemented smoke exercises, not a vague claim that Bun should run on Linux.

**Replacement delivery assessment:** Can has an implemented, pinned Debian 13-or-newer amd64/glibc packaging, install and smoke lane, including Docker orchestration. The review did not execute that lane, so it cannot independently certify its current success or broad SaaS production qualification. The target-specific README still describes publisher signature/upload and source-tree installer limits (`distribution/linux/README.md:83`) and says this lane proves packaging/installation while final product qualification depends on other gates (`:90`). Treat that last statement as the documentation's stated limit, not independent evidence those other gates remain unfinished.

The remaining acceptance exercise is therefore **run the existing Linux lane and extend/review production application qualification**, not “implement a Linux path”: operate the paired invoice/grid build on the intended admitted host, cover service lifecycle, credentials and schema migrations, observe failures, and verify coordinated frontend/backend rollouts/rollback and old asset retention. The need to build/pair the browser manifest (`examples/invoice/README.md`) remains a developer-workflow issue. Current stale README contradictions are themselves a documentation issue to fix, not a platform absence.

## Shared-action deadline/cancellation: narrowed static conclusion

The high-value finding survives, stated precisely: **`action::request` / `action::post` have no author-visible transport deadline or cancellation operand, and their adapter does not bind an abort signal to the native fetch.**

- `compiler/internal/check/action_bindings.go:374` derives the GET client's permitted inputs. `:386` requires exactly its captures record (or zero value inputs for no captures). There is no extra timeout or cancellation argument.
- `compiler/internal/check/action_bindings.go:411` derives POST inputs, with exact captures/body arity checked at `:423`; again no timeout or signal input.
- `runtime/platform/action-client.ts:22` forwards captures/body/site/context to the JSON fetch consumer.
- `runtime/platform/action-json.ts:246` does support an optional `AbortSignal` in the lower-level private TypeScript input, and `:324` passes it to native fetch. However the actual Can client projection at `:576` constructs that input with URL, method, request, body, response, cases and request byte limit only. It omits both a signal and any transport deadline.

An ordinary Can wrapper/helper around those calls cannot add an argument rejected by the checked API or inject an absent native signal. Existing native HTTP connection deadlines and failure policies elsewhere in the language must not be described as missing globally. Likewise, timers/coordination and app attempt IDs can implement some user-visible timeout or ignore-late-response behavior; that is distinct from aborting the underlying shared-action fetch or bounding response-body consumption. Native `Promise.race` selection exists (`runtime/browser/coordination.ts:84`), but selecting another participant does not insert a signal into this fetch adapter.

Do not overclaim that every Can HTTP request is uncancellable, that every app must remain visually stuck, or that wrappers cannot express any deadline-like policy. The supported shared-action projection lacks a transport cancellation/deadline contract. The current grid waits for the action and ignores new mid-flight presses (`examples/invoice-grid/src/web/web.can:973`, `:992`), so a never-settling action remains pending in this example unless the app adds a separate state transition. This is static source reasoning; no fresh stalled-server end-to-end probe was run.

Acceptance remains: stall headers and body separately; enforce a deadline/cancel action through the supported shared-action API; preserve unknown-commit semantics for POST; reject late stale outcomes; verify requests survive intended render replacements but stop when their actual owner is disposed.

## Host-extension recommendation: reconcile alternatives by evidence

The coordinator reported three fresh Jev consultations selecting different extension approaches (reviewed adapters, catalogue growth, and a companion frontend). Treat that disagreement as an instruction to expose the decision boundary, not as a vote to average or proof that any approach is correct. My first-pass “every product feature a compiler project” wording was too broad: the claim applies to *missing in-process browser capabilities*, not all integrations.

Existing extension paths already matter:

- Server Can can invoke bounded external processes (`examples/process/src/main.can:21`, particularly the executable/args/options call at `:30`), so some SDK/tool functionality can sit behind a CLI without a new compiler primitive.
- HTTP request/response/service boundaries and ordinary server routes let independently implemented services interoperate at a data boundary. The current webhook example returns typed JSON through normal HTTP (`examples/webhook/src/web/web.can:8`), and the shared action transport itself uses a regular JSON/HTTP protocol.
- A separately delivered companion frontend can call a Can backend. That architecture need not pretend the companion's JavaScript participates in Can's assertion, immutable-data or capability guarantees. Its contract parity, serving, authentication and release coordination would need an explicit supported recipe. The reviewed compiler-owned browser build does not currently accept arbitrary JS/npm imports, and the admitted asset table does not make arbitrary JS a supported Can asset.
- CSS and ordinary static assets already supply visual breadth without language changes. Existing Can functions and callables should cover considerable reusable UI code where all required host observations/operations are present.

These distinctions imply three different jobs:

1. **Catalogue additions fit small universal missing primitives.** A live value/checked setter and an event snapshot with the relevant actual control state are native DOM contracts. Adding them directly is a smaller, more auditable response to the demonstrated checkbox/dirty-input gap than designing a general FFI first. Preserve native semantics, target restrictions and lifecycle behavior; do not implement a second DOM in Can.
2. **Ordinary Can libraries fit reusable behavior over admitted primitives.** A keyed table, field/error presentation, pure draft reducers and focus policy should first be extracted as normal code. Such a library cannot manufacture checkbox checked state, caret selection or History operations absent from the boundary; evaluate primitives and library ergonomics separately.
3. **Reviewed adapters or companion code fit ecosystem breadth.** If the product actually requires a complex third-party browser library, the adapter must explicitly describe target availability, input/output types and immutability conversions, expected errors, callback/reentrancy rules, owner/disposal semantics and reproducible packaging. “Reviewed adapter” must mean an implemented, testable audit contract; it is not immediate permission to import unrestricted JS into a verified bundle. Conversely, a companion frontend makes the trust boundary visible but sacrifices a single-language frontend and needs cross-boundary contract tooling.

**Smallest discriminating experiment:** after adding only the small generic DOM contracts needed by the existing dirty-input/checkbox evidence, implement one two-instance form widget in an ordinary Can library, with state correction, focus retention, pending request and independent disposal. This determines whether the immediate friction is missing host observations or failed Can library composition. Keep source size/duplication and failure-handling burden as observations, not predetermined scores.

Then choose **one actually required** third-party widget from the intended SaaS feature list. Implement only that integration twice in isolated prototypes: (a) a minimal reviewed adapter with the explicit contract above, and (b) an independently served companion UI using the existing Can HTTP boundary. Do not generalize a plugin/FFI architecture beforehand. Compare build reproducibility, failure visibility, contract drift, callback/lifetime safety, duplicate application logic, and how much Can/compiler-specific code the second vendor would require. If the ordinary Can version handles the product requirement cleanly, catalogue plus library is sufficient for now. If the adapter preserves the checked boundary with modest reusable machinery, pursue adapters. If containing a large external runtime dominates, the companion boundary may be clearer.

This experiment deliberately does not make “adapters win” the acceptance condition. The existing code proves bounded native calls and shared HTTP contracts; it does not yet establish which extension strategy best serves the user's actual next browser requirement.

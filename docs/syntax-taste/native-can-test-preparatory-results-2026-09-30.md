# Native Can testing: preparatory execution results

**Three checks supported their scoped mechanics; browser acceptance failed on unexpected macOS credential UI.** All four scoped checks were run. A single bounded browser-only repair was also run before the user reported the Keychain dialog. No further browser execution followed that report.

These are preparatory observations, not integrated Can qualification. No reference/compiler/owner was promoted, no migration coverage was credited and no host harness was deleted. Product sources were unchanged.

| Check | Current result | Observations |
| --- | --- | --- |
| **PM-N1** | **Mechanism supported** | Inert handles preserved same-object identity with zero getter accesses before the explicit action. The explicit read accessed the getter once. Deliberate Promise assimilation changed the counter and failed the zero-access oracle. |
| **PM-N2** | **Mechanism supported** | The actual candidate `encodeJSON` export encoded `[1n]` as `[1]`, then rejected the getter-backed array with `CodecIssue(type, "")` without reading the getter. The deliberately contaminated shim was detected by its counter. This was a direct source-module call, not generated Can ingress. |
| **PM-F1** | **Mechanism supported** | Binary fd 0 and UTF-8 JSON fd 3 stayed separate; present-empty and absent remained distinct. Withheld EOF triggered the named timeout. A non-string snapshot was rejected. The non-reader accepted an offered 1 MiB stream only partially (65,536 bytes written), exited and left no writer waiting. This exercised Python/POSIX-to-Bun mechanics, not Go `ExtraFiles` or the product launcher. |
| **PM-B1** | **Browser isolation failed; route mechanism observed** | The independent parent received the route and ping while the actual click remained pending, resolved the route and observed the exact marker. The serialized control stayed blocked until cleanup. A later user screenshot revealed an unexpected macOS Keychain dialog. That host effect prevents overall acceptance despite the protocol observations. |

Canonical current outcomes and evidence references are in [execution-summary.json](preparation/native-can-test-preparatory-checks-2026-09-30/execution-summary.json). It supersedes the narrower browser protocol/process-cleanup acceptance recorded before the user reported the host UI.

## The browser findings

The first browser attempt passed its positive leg but failed its control-cleanup oracle. Aborting the held navigation made Playwright's actual click promise **fulfill**, while the instrument expected rejection. Its fulfillment handler then tried to read a result element during context closure. The original failure and full ordered facts are retained.

One reviewed repair preserved the actual settlement event, skipped the marker read only after cleanup began, and allowed either terminal click outcome after route abortion. The positive marker oracle was unchanged. The repaired control required `closing < route_aborted < input_settled < closed`, no pending route, no diagnostics and matching settlement facts. It passed. This establishes that input completion and response/delivery success need distinct observations; neither may stand in for the other.

Afterward, the user reported **“Keychain Not Found — A keychain cannot be found to store Chrome.”** The direct Chrome launch omitted `--password-store=basic` and `--use-mock-keychain`, both present in the installed Playwright defaults at `node_modules/playwright-core/lib/server/chromium/chromiumSwitches.js:77`. Combined with the temporary HOME, this is a plausible explanation for the prompt. The exact process that displayed that earlier dialog was not captured, so this attribution is not claimed as independently proved.

The isolation contract missed native credential UI outside the tracked process/profile graph. Therefore PM-B1 remains failed for overall preparatory acceptance, and this direct-launch configuration is **not admitted for more runs**. Future preparation must select explicit test-only credential behavior and verify absence of host UI. Do not reset or modify the user's Keychain, reuse personal credentials/profile data, or silently retry this launch configuration.

The user was advised to choose **Cancel**, not **Reset to Defaults**. No reset or Keychain setting change was performed. The user confirmed that the dialog had already been canceled. No further UI investigation or browser run is needed for this step. An unnecessary later system-dialog inspection timed out without performing an action. This does not change the observed host-UI isolation failure.

## Budgets and cleanup

The first executing batch took **10.828 seconds**, including 1.243 seconds of input preflight. The browser-only repair took **6.555 seconds**. Combined executing time was **17.383 seconds**; source authoring/review time is excluded. These are deadline-accounting facts, not performance measurements.

| Job / attempt | Job wall time | Sampled peak RSS | Peak observed process count¹ | Sampled scratch peak |
| --- | ---: | ---: | ---: | ---: |
| PM-N1 | 0.375 s | 28.3 MiB | 3 | 60.4 KiB |
| PM-N2 | 0.302 s | 30.3 MiB | 3 | 1.30 MiB |
| PM-F1 | 2.338 s | 42.4 MiB | 3 | 49.8 KiB |
| PM-B1 initial | 6.536 s | 1.72 GiB | 15 | 12.80 MiB |
| PM-B1 repair | 5.320 s | 1.66 GiB | 15 | 12.80 MiB |

¹ Includes the parent and process-inspection allowance. Sampled maxima are observations, not strict OS memory/storage/descendant containment guarantees. The unexpected dialog demonstrates that process/scratch release alone is insufficient to establish all host effects have ended.

Every recorded direct test child was reaped; owned browser descendants were checked; the browser main processes received TERM during cleanup. All three owned preparation/execution roots were removed, including the unused initial preparation root. Post-report inspection found none of the recorded test PIDs and no remaining `can-native-prep-*` roots. No user Chrome process was targeted by name, no shared cache was cleared and no worktree was created. Compact sources, hashes, ordered observations and receipts remain as evidence; execution workspaces, browser profiles and Bun-generated temporary files do not.

## Evidence and next step

- [First raw run](preparation/native-can-test-preparatory-checks-2026-09-30/run-20260930T162840-599df6-results.json), [cleanup finalization](preparation/native-can-test-preparatory-checks-2026-09-30/run-20260930T162840-599df6-finalization.json), [instrument source snapshot](preparation/native-can-test-preparatory-checks-2026-09-30/run-20260930T162840-599df6-instruments.json).
- [Bounded repair admission](preparation/native-can-test-preparatory-checks-2026-09-30/run-20260930T163537-e5ff9c-admission.json), [exact repair diff](preparation/native-can-test-preparatory-checks-2026-09-30/run-20260930T163537-e5ff9c-repair.diff), [repair raw run](preparation/native-can-test-preparatory-checks-2026-09-30/run-20260930T163537-e5ff9c-results.json), [cleanup finalization](preparation/native-can-test-preparatory-checks-2026-09-30/run-20260930T163537-e5ff9c-finalization.json).
- [Current result summary and host-UI amendment](preparation/native-can-test-preparatory-checks-2026-09-30/execution-summary.json).

The next design reconciliation must include inert handle transport, the remaining compiled-Can ingress seam, independent input/delivery completion, descriptor EOF ownership, and browser credential/UI isolation. Integrated N1/N2/B1/F1 remain unrun. The findings do not authorize promotion of R/N or retirement of existing test coverage.

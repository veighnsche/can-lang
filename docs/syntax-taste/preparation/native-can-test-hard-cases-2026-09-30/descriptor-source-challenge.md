# Descriptor credentials and launcher boundaries

Source review only; no child process, build, credential probe or experiment ran. The current `/tests` obligation is [CORE-028](../native-can-tests-migration-ledger-2026-09-30/core.md#core-028), with related distribution and application-launch obligations. Compiler-internal lease tests and installed Bun types are supporting mechanism evidence, not an expansion of the migration ledger.

## What the existing tests actually protect

[`TestCurrentBundledInputEnvironment`](../../../../tests/integration/io_env_test.go) launches the **candidate Go CLI**, with `PATH=/nonexistent`, owned `HOME`, `EMPTY=`, Unicode `VALUE=hé😀` and hostile `BUN_OPTIONS=--preload=/must-not-execute.ts`. The application must read the hostile option as environment **data** without executing it as a Bun startup option. Can must retain these scenarios and comparisons:

- Boundary assertions consume supplied completions rather than the real stdin/environment.
- Present empty, missing and invalid environment names remain distinct; Unicode is exact.
- Binary stdin preserves NUL and non-UTF8 bytes for byte input; text input preserves valid UTF8/BOM and rejects invalid text.
- Negative limits and overflow fail with the intended error. The one-MiB output case checks awaited stdout/stderr drain, not just an early prefix.
- The subsequent build is a candidate product action; optional strict TypeScript checking remains a separately declared coverage leg.

[`TestDevelopmentSidecar`](../../../../tests/integration/distribution_test.go) additionally supplies hostile `NODE_OPTIONS`, `BUN_OPTIONS`, configuration files and `.env` near the launch, then checks runtime identity, literal arguments, original environment access and secret non-disclosure. Its tiny throwing preload file is a static adversarial **input fixture**, not a retained TypeScript test harness. Can should create/select that payload, launch the product, and judge its effects. A native routine that performs this complete scenario and returns `passed` would violate the migration objective.

## Two launch contracts are necessary

1. **Product launcher subject.** N passes the exact explicit environment to C's Go CLI, including the hostile startup flags under test. C must sanitize the child Bun startup and preserve application data itself. Sanitizing these inputs before C sees them would mask the regression.
2. **Direct Can entry.** The native launch binding supplies a sanitized Bun startup environment, plus the separate application snapshot on fd 3, binary stdin on fd 0 and any declared generation lease on fd 4. R/N and their private reporting channels never inherit the hostile subject environment.

These are explicit executable/descriptor policies on generic `proc.spawn`; they do not require a language primitive per kind of test. Can selects the profile and data. N verifies the declared executable and allowed inherited descriptors and records launch facts. Snapshot payloads are never command arguments, ordinary logs or default retained files. Deliberate synthetic marker output is allowed as a test observation; actual service credentials must remain protected.

## EOF, backpressure and unused credentials are part of correctness

[`prepareEntry`, `begin` and `finishEntry`](../../../../compiler/internal/driver/runtime.go) create an OS pipe, put its read end first in `ExtraFiles` (fd 3), launch the child, write concurrently, then close the writer. Any output lease follows as fd 4. The actual Bun environment is a short allowlist and startup arguments disable installation, `.env` and macros with an explicit configuration.

[`runtime/environment.ts`](../../../../runtime/environment.ts) synchronously reads fd 3 to EOF during module initialization, closes it, requires a JSON object with string values and freezes a null-prototype copy. Consequently:

- Waiting for application readiness before writing/closing fd 3 can deadlock startup.
- Invalid, truncated or missing snapshot data is an import/startup failure, not `http::credentials_missing` from a successful environment lookup.
- Each launch needs a fresh descriptor/stream position and independent EOF. stdin and the snapshot must not consume one another.
- A non-reader can exit while a large snapshot writer is blocked. `finishEntry` closes the pipe and tolerates benign `EPIPE`/closed-pipe delivery after successful child exit, while preserving a genuine child failure.
- Cancellation must close and reap the writer as well as the child. A sent byte count proves pipe acceptance, not child consumption.

[`output_runtime_test.go`](../../../../compiler/internal/driver/output_runtime_test.go) explicitly uses a one-MiB unused snapshot. Therefore a proposed tiny fixed credential cap must not quietly remove that case if it becomes a retained qualification obligation. The focused delivery experiment below uses at most two MiB total input.

[`env.ts`](../../../../runtime/platform/env.ts) validates names before lookup, respects live-boundary denial and distinguishes absent from empty. [`io-env.test.ts`](../../../../runtime/test/io-env.test.ts) adds counters proving invalid names do not read, split UTF8/BOM, huge integer limits, cancellation failure and error classification. These checks belong in their individual delegated-ledger cases; successful descriptor delivery alone does not cover them.

## Lease and launcher death

[`RunOutput`](../../../../compiler/internal/driver/output_runtime.go) validates the output generation and runtime identity before passing its lease through fd 4. [`OutputLease`](../../../../compiler/internal/driver/output.go) is a kernel-held lifetime resource, not merely a path named in a report. The existing internal test closes the parent's copy and observes that the child keeps the generation alive, then kills/reaps the child and permits pruning.

The new owner must distinguish process exit, descriptor closure and generation reclamation. A candidate launcher disappearing does not authorize deleting a generation still leased by its child. Conversely, a report saying “lease released” cannot substitute for native release evidence. A future parent-death experiment should kill only an owned subordinate launcher, leaving N alive to reclaim its child; do not kill the machine's actual supervisor as an ordinary case.

## Binding evidence and remaining uncertainty

[`runtime/platform/process/spawn.ts`](../../../../runtime/platform/process/spawn.ts) exposes stdin/stdout/stderr and environment, but not the required extra-descriptor map, and its cleanup is worker-local after spawn. It is not already the proposed external launch capability.

The installed `bun-types` **1.4.2** declaration (`node_modules/bun-types/bun.d.ts`, `SpawnOptions.Options.stdio` and `Subprocess.stdio`) includes extra descriptors and POSIX `socket-fd`, with explicit caller ownership. [`package.json`](../../../../package.json) pins Bun and `@types/bun` 1.4.2. This is declaration evidence that a binding may be possible, not a successful runtime qualification, and it does not establish descriptor lease inheritance or cleanup. The existing Go `ExtraFiles` path is stronger implementation evidence and fits the already chosen external Go owner. No reason to move ownership into a Bun case worker merely because Bun declares an API.

## Future focused questions (unrun)

- **Delivery and launch isolation:** can the external launch binding provide fd 3 to EOF alongside binary stdin, while preserving hostile environment inputs to the candidate CLI and sanitizing only the intended Bun startup? Acceptance requires the Unicode/empty/missing distinctions, independent stdin, correct import failure for bad snapshots, no deadlock on a one-MiB non-reader and complete descriptor cleanup. Wrong-fd and withheld-EOF controls must fail the appropriate named checks; neither may become an unrelated credentials-missing result. Deadline 45 seconds including cleanup, one case, at most two simultaneous owned children, two MiB total input and 256 KiB retained facts; reuse existing qualified artifacts rather than build a distribution.
- **Lease inheritance:** does an owned child retain the correct generation after its subordinate launcher exits, and can it be reclaimed after child exit? Acceptance requires native lease identity, a clean attempt that retains the directory while held, later release/prune facts, and no descendant left alive. A deliberately omitted or prematurely closed lease must be detected. Deadline 30 seconds including cleanup, at most two children and one tiny owned generation; no full distribution copy or sleep-based liveness oracle.

These are mechanism gates only. They do not authorize running the experiments in this source/design round or deleting a host harness before the mapped Can cases exist and preserve coverage.

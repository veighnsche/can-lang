# B1-04 — Bounded child-process execution

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: none. Surface: Library; event primitive candidate for incremental output.

Expose bounded process execution first, then incremental output through B1-05. Input is an executable and argument array. A shell-language API is not needed for this capability. A nonzero exit is a completed process result; an explicit checked helper may convert it to a declared domain error.

## Where to implement

**Required layout:** follow [filetree and module boundaries](filetree.md). Its feature modules supersede the coarse existing-file locations below; do not append B1 implementation bodies to existing orchestration hubs.

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/platform/process.ts` | new | Spawn, concurrent pipe draining, deadlines, termination and reaping. |
| `runtime/owner.ts` | existing | Register the child and its pipes under the current owner. |
| `tests/integration/process_test.go` | new | Can API and owned subprocess cases. |
| `runtime/test/process.test.ts` | new | Controlled child scripts and race tests. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
process::run(executable, args, options) -> process_result
options = cwd, explicit_env, stdin_bytes, stdout_limit, stderr_limit,
          deadline_ms, termination_grace_ms
process_result = stdout_bytes, stderr_bytes, exit_code_or_signal
process::require_success(result) -> result emits [process::nonzero]
process::which(name, search_path) -> optional_path
```

## Implementation sequence

1. Define binary output, explicit UTF-8 conversion, exit-vs-signal representation, environment inheritance opt-in and path resolution. Validate durations and byte caps before spawning.
2. Start native Bun.spawn with arrays, explicit cwd/env and piped I/O. Register ownership immediately after spawn, before waiting for any output.
3. Drain stdout and stderr concurrently while writing stdin with backpressure. Independent per-pipe limits prevent one blocked pipe deadlocking the other. Increment counts before retaining bytes.
4. On timeout, overflow, input error or owner cancellation: stop producing stdin, signal child, wait grace, escalate if supported, await exited and finish/cancel pipe drains under a bounded shutdown deadline.
5. Specify direct-child versus process-tree guarantees. Add an owned-grandchild reproducer; do not promise no descendant survives merely because child.kill() completed. Qualify a platform-native group strategy or make descendants an explicit unsupported contract.
6. Map spawn/permission/I/O errors separately from a successful nonzero status. Preserve original timeout/overflow failure when cleanup also fails.
7. Add catalogue and fixture provider binding. Pass incremental reader ownership to B1-05 instead of exposing mutable native subprocess objects.

## Acceptance evidence

- Success: argument literally containing $(...) and semicolons; binary stdin/stdout; stderr concurrently larger than a pipe buffer; exit 0 and exit 7.
- Rejected: wrong args type, invalid limits, unsupported environment value; missing executable and denied cwd are declared failures.
- Runtime: endless writer bounded in memory; hanging child reaped; ignores graceful signal; cancellation while stdin blocked; exit simultaneous with deadline; descendant cleanup contract verified.

## Fixtures and local assertions

Use repository-owned tiny child scripts; never arbitrary installed tools for core tests. Fixture request includes executable identity, argv, selected env, cwd and stdin digest. Transcript records independent stdout/stderr chunks and terminal exit; do not invent a total ordering across the two OS pipes.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const child = Bun.spawn([executable, ...args], {cwd, env, stdin: "pipe", stdout: "pipe", stderr: "pipe"});
// Drain both pipes concurrently and await child.exited inside the resource owner.
// Do not use `new Response(pipe).bytes()` for unbounded adversarial output.
```

## Gates and limitations

A deadline promise race does not terminate a process. Bun.kill of one PID is not process-tree containment. Do not claim a hard shutdown bound for unsupported OS cases without a reproducer and implementation proof.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/child-process)
- [Official Bun documentation](https://bun.sh/docs/runtime/utils)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.

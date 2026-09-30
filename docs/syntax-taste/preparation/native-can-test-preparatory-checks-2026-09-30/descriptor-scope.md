# Preparatory descriptor delivery: PM-F1

Status: **scoped, unrun**. This investigates one subset of [F1](../native-can-test-hard-cases-2026-09-30/experiments.md). It does not execute the Go product launcher, qualify the eventual Go owner, or establish ordinary Can process bindings.

**Question:** Can the existing pinned Bun receive an EOF-terminated environment snapshot on inherited fd 3 independently of binary stdin on fd 0, without leaving a blocked writer when the child exits without consuming the snapshot?

**Selected mechanism:** the common Python parent T uses `os.pipe`, nonblocking parent writes and `os.posix_spawn` file actions. Duplicate each child-side descriptor to a distinct CLOEXEC descriptor numbered at least 10 before defining actions; map these to 0/1/2/3 with `POSIX_SPAWN_DUP2`, then close the high descriptors in the child. Close the parent's copies of child ends immediately after spawn. This avoids accidental fd 3 collisions and a threaded `preexec_fn`. T must inspect the exact mapping before launch and record only descriptor roles, never credentials. No Go compiler, dependency download or product bundle is needed.

The source precedent is `compiler/internal/driver/runtime.go::prepareEntry/begin/finishEntry`: Go `ExtraFiles` delivers the snapshot at fd 3, stdin is independent, writer EOF permits the Bun read, and closing the writer after child exit avoids non-reader deadlock. The probe instead tests the POSIX/Bun boundary. Do not report Go `ExtraFiles`, `finishEntry`, CLI sanitization or output-generation leases as tested.

## Minimal fixture and independent facts

T launches the absolute pinned Bun with `--no-install --no-env-file --no-macros`, the repository's explicit `tools/runtime/bunfig.toml`, and a temporary entry under owned scratch. Its environment is an explicit allowlist: owned HOME, XDG_CONFIG_HOME and TMPDIR, plus `PATH=/nonexistent`. Application values travel only through fd 3. Use only synthetic values; no real environment/credential snapshot.

The entry synchronously writes a small `before_import` fact to stdout, dynamically imports the absolute candidate `runtime/environment.ts`, reads fd 0 as bytes and calls `originalEnvironment` for three fixed names. Encode present/absent explicitly, since JSON omits `undefined`; emit only the byte hex, those tagged values and a terminal marker. Catch setup/import failures only to preserve a bounded diagnostic and nonzero exit, never convert them to success. Importing the actual module exercises its current synchronous `readFileSync(3, "utf8")` and validation. No copied parser or expected-value logic belongs in that module.

The external T constructs and independently compares:

- fd 0 bytes: hex `00ff0a0d41420043` (including NUL and non-UTF-8 bytes).
- fd 3 UTF-8 JSON: `{"PROBE_TEXT":"λ-é","PROBE_EMPTY":""}` followed by writer close.
- `PROBE_TEXT`: present with exactly `λ-é`; `PROBE_EMPTY`: present with an empty string; `PROBE_ABSENT`: absent.
- exact stdin hex, a complete bounded observation, successful child exit and verified pipe/process/scratch release. Neither an EOF-free prefix nor process exit by itself counts as success.

## Four finite legs, one question

1. **Ordinary delivery:** write the snapshot and stdin through separate pipes, close each writer, observe the fixed facts and reap the child.
2. **Negative control — withheld EOF:** write all valid snapshot bytes but deliberately retain the fd 3 writer. After `before_import` and completion of the parent's write, allow at most 2 seconds for the intentionally blocked import. T must record `fd3_eof_timeout`, close the writer and terminate/reap if needed. A later completion after cleanup releases EOF does not erase the control's failure. This tests that success depends on actual delivery completion, rather than accepting an early prefix or readiness event.
3. **Malformed snapshot:** send `{"PROBE_TEXT":42}` and close fd 3. Require no success facts, nonzero child exit and the bounded `invalid launcher environment snapshot` diagnostic. This is expected input rejection, separate from the deliberate delivery defect.
4. **Non-reader exit:** a separate tiny entry exits successfully without importing the environment module. Offer at most 1 MiB on fd 3 with nonblocking writes; on child exit T closes every writer and observes termination of its write loop within 2 seconds. An incomplete write caused by this expected exit is recorded, not converted into a lost child error. No need to assert the pipe has a particular capacity. This demonstrates the prototype's behavior only, not `finishEntry` correctness.

Run one child at a time, with a fresh operation ID and pipe set per leg. The normal and malformed legs each get at most 5 seconds; the withheld-EOF and non-reader legs use the 2-second local conditions above, within the common job deadline. A child that fails to launch, unexpected exit/diagnostic, mismatched stdin, missing observation or missed writer completion makes the job failed/inconclusive. Do not retry with a different launcher or silently omit a leg.

**Acceptance:** ordinary delivery passes all independent comparisons; the EOF defect is detected by its named failure; malformed input is rejected as specified; non-reader exit leaves no writer waiting; all four legs have matching cleanup facts. The overall record may say `mechanism_supported` only with all these outcomes. A negative leg's expected failure is recorded as a failure detection, never a passing product test.

## Budget, files and ownership

The single PM-F1 job has **45 seconds including at least 10 seconds for cleanup**, at most two sustained processes (T and one Bun), one transient process-inspection helper if needed, 12 open pipe endpoints, 1 MiB snapshot input, 64 KiB aggregate child output, 2 MiB total I/O, 8 MiB scratch and 64 KiB retained facts. Stop threshold: 512 MiB observed aggregate RSS. These small fixed fixtures allocate bounded data; RSS sampling is not an OS memory limit. Common ownership and failure rules in the [job plan](../../native-can-test-preparatory-checks-2026-09-30.md) apply.

Future writable files are only the registered scratch root's `owner.py`, `descriptor-subject.ts`, `nonreader-subject.ts`, owned HOME/config/tmp/cwd directories and short bounded logs. Candidate `runtime/environment.ts`, the explicit Bun config and executable remain read-only. T owns scratch, child start identity, all pipe ends and the write loop outside the child. It registers cleanup before pipe creation/spawn, closes every end on start failure, handles interruption through the same close/terminate/wait path, verifies the writer is stopped, then removes its exact owned directory. Failure to close/reap/remove is an unresolved cleanup error and prevents further jobs. No child-local `finally` is the cleanup authority.

Remaining integrated F1 obligations include the real Go launcher/CLI, hostile startup-control sanitization versus preserved application values, generated Can stdin/env/auth access, descriptor lease fd 4, negative exit preservation and the Can-native managed-process contract. Their existing registered gates stay unrun.

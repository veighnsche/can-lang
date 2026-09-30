# Native Can test authoring: source review

Status: **design evidence only**. These fragments show current Can spellings and
identify proposed surfaces. They were not compiled or executed. A test suite must
use ordinary Can functions for cases and helpers; these fragments do not choose an
execution backend or settle the final suite API.

## Syntax that already exists

- A package declares `provides` and `uses`; an ordinary function declares its
  result, `emits`, inputs under `given`, attached `asserts`, and a terminal body.
  `fn void main` takes `str[]` for a Bun CLI program. See
  [process example](../../../../examples/process/src/main.can#L1),
  [main](../../../../examples/process/src/main.can#L42), and the
  [compiler's main-shape checks](../../../../compiler/internal/check/browser_target_test.go#L73).
- `callable void (...) emits {...}` is a type usable in records, arrays and
  function inputs; `callable name` creates a named function reference and
  `call stored.action(...)` invokes it. The effect set is part of the callable
  type, so a registry needs one uniform case signature or adapters written in
  Can. See [record, local, field and array callables](../../../../compiler/testdata/current/callables/captures.can#L4),
  [callable argument](../../../../examples/gallery/src/26-callable-argument.can#L13),
  and [explicit capture](../../../../compiler/lsp_g03_test.go#L37).
- `relay call` forwards the success or named error completion within the
  declared result/effect contract. `match call` can consume a particular named
  error and distinguish it from `ok`; a successful `checks::require` is `void`,
  and failure is `checks::failed{reason}`. See
  [relay](../../../../examples/gallery/src/18-relay.can#L5),
  [recovery](../../../../examples/gallery/src/17-named-error-recovery.can#L5),
  and [checks](../../../../compiler/testdata/current/checks/main.can#L4).
- Attached `asserts` and lexical `when` rows can verify deterministic helper
  behavior with supplied process results. A real subprocess belongs in normal
  execution, because `process::run` refuses a live assertion context. See
  [process example](../../../../examples/process/src/main.can#L22),
  [research boundary](../../native-can-tests-investigation-2026-09-30.md#L134),
  and [assertion semantics](../../native-can-tests-investigation-2026-09-30.md#L114).

## Three authoring fragments

The first fragment shows static registration as **current language syntax**;
the choice to register cases this way is a **proposal**. This is a declaration
and body fragment, not a complete package. `check_label` is an ordinary Can
function with exactly the callable signature shown. Case selection, order and
coverage accounting would also be ordinary Can functions using these values,
not reflection or a manifest script. A proposed `spec::context` and
`spec::failed`/`spec::broken` can replace this minimal signature if the design
needs typed case failure and infrastructure evidence; those names do not exist
today.

```can
record test_case
    str id
    callable void () emits {checks::failed} run

test_case[] cases = [test_case("checks/label", callable check_label)]
```

The second fragment is a complete ordinary Can helper using an existing
standard check. Its attached rows validate the comparison rule before a live
case uses it. `relay` preserves the original `checks::failed` completion and
reason. A suite can add Can helpers for diagnostic codes, byte arrays, report
coverage, or process outcomes in the same form.

```can
fn void expect_exit_code
    emits {checks::failed}
    given
        int observed
        int expected
    asserts
        same: 3, 3 => ok
        different: 3, 4 => checks::failed{"unexpected exit code"}
    relay call checks::require(observed is expected, "unexpected exit code")
```

The third fragment follows the existing process example's exact call and
supplied-fixture pattern. The `when` row permits an offline assertion of this
function; normal execution actually spawns the process. It retains the raw
`process::result` so a negative compiler test can check a nonzero exit rather
than using `process::require_success`. All emitted operational errors must be
forwarded or handled explicitly; the uniform registry signature above would
therefore need a Can adapter that classifies them, or a wider exact callable
effect signature.

```can
fn void echo_exits_zero
    emits {files::not_found, files::denied, process::spawn_failed, process::timeout, process::output_limit, process::invalid_config, process::io_error, checks::failed}
    asserts
        sample: => ok
    bytes::buffer empty = call bytes::empty()
    process::result supplied = process::result(empty, empty, 0, "")
    match call process::run("/bin/echo", ["hello"], process::options("", true, [], empty, 100, 100, 5000, 100))
        when
            sample: "/bin/echo", ["hello"], process::options("", true, [], empty, 100, 100, 5000, 100) => ok supplied
        files::not_found
        files::denied
        process::spawn_failed
        process::timeout
        process::output_limit
        process::invalid_config
        process::io_error
        ok process::result done => relay call expect_exit_code(done.code, 0)
```

This fragment is based on [the live process example](../../../../examples/process/src/main.can#L21)
and the [catalogue's exact option, result and `run` signatures](../../../../runtime/catalogue.ts#L991).
The existing `process::options` contains cwd, inherited environment, explicit
environment, stdin bytes, two output limits, deadline and grace; `result`
contains separate stdout/stderr bytes, code and signal
([catalogue](../../../../runtime/catalogue.ts#L1034),
[run](../../../../runtime/catalogue.ts#L10998)). The operation uses native
`Bun.spawn`, owns and reaps its child group, and is `supplied` in assertions.

## Exact compiler rejection candidate

Use a **separate staged fixture project**, not malformed source in the suite's
own source root. Begin with the valid source at
[the explicit capture example](../../../../compiler/lsp_g03_test.go#L37):
`combine` declares `near int prefix`; `run` computes `int doubled` and binds
`callable combine with prefix = doubled`. Stage a control copy and one mutated
copy, changing exactly this source text in the latter:

```text
callable combine with prefix = doubled
callable combine with prefix = "bad"
```

Can should demand that the control reaches successful semantic checking and
that the mutated copy is rejected in semantic checking with
`CAN-CHECK-CAPTURE` at the binding expression's span. The checker stamps that
code for an explicit `with` binding whose value has the wrong exact type
([implementation](../../../../compiler/internal/check/callables.go#L124)); a
current compiler test asserts the same code and a located span for a mistyped
capture ([test](../../../../compiler/internal/check/callables_test.go#L152)).
Checking only nonzero status would let an unrelated parse or launch failure
pass. Can must own the expected code/phase/span, comparison and control; native
infrastructure can return compiler observations. A public structured semantic
`canlc check --json` remains **proposed**; today's `parse` is grammar-only and
`inspect-types` is declaration-level. The reusable compiler-side
`CheckSnapshot` API exists, but no public structured command is established
([research](../../native-can-tests-investigation-2026-09-30.md#L230)).

## Missing public mechanics and boundaries

| Required observation or lifetime | Current evidence and gap |
| --- | --- |
| Case registration, selection, report and status | Callable records and `main(str[] args)` can express static cases. There is no repository-wide live case registry, filter/list protocol, or report API; Can must own policy, while the launcher may transport and enforce the protocol. [Research](../../native-can-tests-investigation-2026-09-30.md#L169), [completion contract](../../native-can-tests-completion-contract-2026-09-30.md#L188). |
| Owned temporary fixture scope | `files::write_text` exists and is supplied in assertions, but there is no owned temporary workspace primitive. An appended `files::remove` cannot guarantee cleanup after failure, interruption or force-kill; parent supervision/recovery needs ownership and path checks. [Catalogue](../../../../runtime/catalogue.ts#L10503), [shutdown contract](../../../../docs/implementation/shutdown.md#L3), [contract](../../native-can-tests-completion-contract-2026-09-30.md#L231). A namespace such as `spec::workspace` would be **proposed**. |
| Managed long-lived child | `process::run` is one-shot. Server cases require start, readiness observation, fd 3 credentials, signals, bounded wait and restart. No current `process::start`/`process::child` API is established; those names are **proposed**. [Research](../../native-can-tests-investigation-2026-09-30.md#L214), [ledger LIFE-003](../native-can-tests-migration-ledger-2026-09-30/lifecycle.md#L48). |
| Browser and native observations | Browser-target `browser::*` is in-page behavior, not external automation; no complete browser driver, page-evaluate replacement, physical-input probe, or independent native-value/SQL observation surface has been demonstrated. Any `spec::browser` or `spec::native_probe` spelling is **proposed**. [Research](../../native-can-tests-investigation-2026-09-30.md#L246), [contract](../../native-can-tests-completion-contract-2026-09-30.md#L172). |
| CLI outcome | Existing `fn void main` accepts `str[]` and named errors; suite protocol still needs a defined process exit/report mapping, full versus selected coverage markers, and crash/malformed report behavior. Can owns the verdict and aggregation; supervisor enforces isolation and may classify absence of valid evidence as incomplete. [Main example](../../../../examples/process/src/main.can#L42), [contract](../../native-can-tests-completion-contract-2026-09-30.md#L188). |

The [migration ledger](../../native-can-tests-migration-ledger-2026-09-30.md)
requires declaration-level parity, including expected failures and delegated
Go/TypeScript oracles. The snippets show feasible authoring idioms, not
qualification of compiler, subprocess, server, browser or native coverage.

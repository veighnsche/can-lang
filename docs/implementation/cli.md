# Current build and run pipeline

This document describes the current implementation. The selected revised contract is [P15.1](../syntax-taste/platform-testing-spec.md#p151-verified-build-and-publication), with native assertions in [P4.1](../syntax-taste/platform-testing-spec.md#p41-attached-native-and-wrapper-assertions). The [targeted verification report](implementation-gap-verification-2026-09-22.md) records the confirmed gaps; the selected behavior is not yet an implementation claim.

Use a development bundle assembled with the pinned local archive as described
in [distribution instructions](../../distribution/README.md). A plain Go-built
launcher refuses `build` and `run` because it has no qualified sidecar.

```sh
/absolute/version/bin/canlc build /absolute/canonical/project
/absolute/version/bin/canlc build --target browser /absolute/canonical/project
/absolute/version/bin/canlc run /absolute/canonical/project -- 'application argument'
```

`build --target browser` verifies the same assertion roots under Bun, then
ships the distinct `browser.ts` main-thread root with its content-addressed
`browser/asset.json` manifest instead of the Bun entry. The checked program
must pass the transitive browser capability closure first: any reachable
path through authored calls, callable references or generic specializations
to SQL, process, filesystem, environment-secret, server-crypto or adjacent
server capabilities fails the build with a located diagnostic. Unknown
targets fail closed; there is no worker profile.

The project directory must use real, canonical directory components, as required
by manifest-owned output safety. `build` prints a versioned `can.build` JSON
report containing the generation ID, directory and entry. `run` always checks
and builds current sources, then executes the absolute bundled Bun. Its stdout
contains only application output. Arguments after `--` are application arguments,
including flag-shaped strings; the launcher, project selector, entry path and
runtime options are excluded. Omitting `--` is allowed when there are no args.

The root project must declare exactly one non-generic, non-method
`void main(str[] args)`; the input may have any valid name. Main may declare
and return domain failures. All current concrete function bodies and inert
initializers are checked before publication. File-local imports retain their
scopes. Initializers execute in dependency order inside the supervisor before
main, and every generated function uses the private completion ABI. The generated
entry uses top-level await. No Go interpreter executes the program.

Status 0 means successful void completion. Usage errors return 2. Static,
distribution and build failures return 1 with a stderr diagnostic. Domain and
standard runtime failures return 1 and a versioned JSON stderr diagnostic;
native process failure remains nonzero. Reports include the failure channel,
phase, occurrence identity, and either stable domain ID/type or standard category.
Domain payloads are explicitly redacted in default reports, and native causes,
messages, stacks and host paths are not serialized. Diagnostic writes are awaited;
a broken stderr pipe cannot change a failure into success.

One generation-private runtime and initialization state is shared by source
modules. `run` retains the active generation lease while releasing the project
writer lock, permitting later builds without pruning executing code. Static
failure does not replace `dist/current.json`.

The first stdout example is [echo.can](../../compiler/testdata/current/cli/echo.can).
It uses catalogue signatures for `bytes::from_utf8` and `io::stdout_write`.
`io::stderr_write` uses the same awaited adapter. Byte storage is opaque and copied,
UTF-8 conversion rejects unpaired surrogates, and writes return exact byte counts.
These are foundations for I13 and I29, not completion of their full inventories.
Unsupported intrinsics, specialization and capture lowering fail explicitly.
The [assert command](assertions.md) executes mandatory assertions in emitted code.
Generic specialization is I46; ownership/draining is I20. Both tasks are checked with evidence.

Runtime reports now include [mapped Can locations](source-maps.md), with exact
byte spans and displayed UTF-16 coordinates. Raw native stacks remain private.

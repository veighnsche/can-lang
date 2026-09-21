# Native target qualification (I01)

`target.json` freezes the initial upstream Bun archive, extracted executable,
revision, macOS arm64 target, minimum OS, native API inventory, and behavior
probes. Archive bytes were checked against the saved upstream release metadata;
the extracted executable digest was measured independently. Hashes bind bytes
and do not replace publisher-signature or release notarization checks (I39).
macOS 13.0 is the upstream minimum, not a claim that every supported OS release
has been exercised. Each report records the actual OS and runtime component
versions (including ICU and Unicode).

Acquire the exact `upstream.url` archive separately, then run:

```sh
python3 distribution/qualify.py --archive /absolute/path/bun-darwin-aarch64.zip --report /absolute/path/native-capabilities-v1.json
```

The harness never downloads anything or selects Bun from PATH. It checks the
archive and executable digests, creates a temporary versioned layout with
`runtime/bun`, and invokes that absolute executable from an unrelated directory
with an empty home, no ambient runtime configuration, PATH without Bun, and
network access denied by macOS `sandbox-exec`. Sandbox failure is a gate failure;
there is no online fallback. This is a qualification fixture, not the I02
launcher/sidecar installer. The temporary layout is removed after execution.

The report uses schema version 1 and kind `can.native-capability-report`, binds
the manifest and probe source hashes, records each capability and behavioral
result, and fails closed on a missing API or mismatched runtime. The accompanying
nine tests include removal of the actual `JSON.rawJSON` and `Array.fromAsync`
functions and rejection of runtime identity differences. The manifest's API
presence checks do not claim full HTTP, SQL, or language semantics coverage.

CI uploads `native-capabilities-v1` separately from any language conformance
results. The old Ubuntu/Z3 verifier workflow did not qualify the approved target
and has been replaced. This gate does not claim completion of C12/P15 language,
provider, resource, packaging, or application conformance.

Changing the pin requires updating provenance and both hashes, incrementing the
target ID, and rerunning qualification. Never follow `latest` or add API shims.

# Development sidecar (I02)

Build from a local archive with Go available and its module cache already filled:

```sh
make bundle BUN_ARCHIVE=/absolute/path/bun-darwin-aarch64.zip VERSION=dev-1
```

The result is `dist/development/can-dev-1-bun-1.4.2-darwin-arm64-v1/`:

```text
bin/canlc                  Go launcher, stamped with the manifest digest
runtime/bun                exact, unmodified upstream executable
runtime/environment.ts     private caller-environment accessor
tools/runtime/            fixed runtime check and compiler-owned config
distribution/target.json  pinned target
distribution/notices/     upstream licensing document and development caveats
manifest.json              versioned asset hashes
tsconfig.json             compiler-owned configuration at the bundle root
```

Existing version roots
are refused. Staging uses a sibling temporary directory; publication renames the
completed directory. The builder uses `GOPROXY=off`, `GOSUMDB=off`, and
`GOTOOLCHAIN=local` and never acquires Bun. Source builds require a local Go
installation; running the bundle does not.

Run `/absolute/path/to/version/bin/canlc runtime-check` from any directory,
including via a symlink and with PATH lacking Bun. This development command runs
only the fixed manifest-owned check; it does not expose arbitrary TS execution.
Current `build` and `run` commands use the checked compiler and bundled runtime; see
[CLI instructions](../docs/implementation/cli.md). The [assert command](../docs/implementation/assertions.md) runs mandatory assertions
through the same bundled runtime. Runtime
lookup follows the real launcher, verifies the launcher-bound manifest, target,
all owned asset hashes, Mach-O arm64 identity and executable permission, and
refuses missing, modified or symlinked assets with `CAN-DIST-*` diagnostics.

Bun starts in a fresh temporary cwd with private HOME/config directories and
only driver-selected startup variables. Caller environment data travels through
an inherited pipe, never a secret-bearing file or command argument. Approved
future env/auth operations must use `runtime/environment.ts`; process.env is the
isolated startup environment. The check exposes only the explicit
`CAN_DISTRIBUTION_PROBE` test variable, not arbitrary caller secrets. Private
working directories are removed after child exit.

`--config=path`, `--no-install`, `--no-env-file`, and `--no-macros` are exercised
on the exact pin. `--tsconfig-override` is intentionally absent: a direct probe
triggered an upstream directory-mismatch diagnostic, so native lookup finds the
hash-locked bundle-local config instead. This is config isolation, not a sandbox
for arbitrary JavaScript. Integration tests additionally deny all network using
macOS sandbox-exec.

```sh
CAN_BUN_ARCHIVE=/absolute/path/bun-darwin-aarch64.zip go test -count=1 -v ./tests/integration
```

CI executes this test with the already acquired archive and retains its report
separately. Without that explicit input, ordinary Go tests skip the packaged
integration suite; they do not claim offline-distribution coverage. The local
saved result is in `docs/implementation/evidence/2026-09-21/i02-sidecar-tests.txt`.
Publisher signatures, complete third-party source/relink material, notarization,
quarantine, end-user installation and updates remain I39 gates.

The I47 development command `canlc catalogue-check` accepts the fixed catalogue
conformance fixture on stdin and runs generated-registry checks through the
same isolated sidecar. It verifies identity/payload agreement and reports its
source digest; it does not accept arbitrary code or register host operations.

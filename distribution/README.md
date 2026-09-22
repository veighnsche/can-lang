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

# Offline release, install, and updates (I39)

A release packages one verified version directory into a single shippable
zip plus a detached SHA-256 record and a read-only inspection transcript.
Entry order and timestamps are fixed, so identical bundles release
identical bytes. Install stages the zip into a user-owned root, verifies
the staged tree against its manifest, then publishes it beside older
versions and swings the `current` symlink atomically:

```sh
# Build, then release (or pass --release-out to distbuild directly).
go run ./tools/distbuild --archive /absolute/path/bun-darwin-aarch64.zip \
  --out /tmp/bundles --version 0.1.0 --release-out /tmp/releases
# Install into a fresh root and run through the selected link.
go run ./tools/distbuild install --archive /tmp/releases/can-0.1.0-*.zip \
  --sha /tmp/releases/can-0.1.0-*.zip.sha256 --root /tmp/can-root
/tmp/can-root/current/bin/canlc runtime-check
# Update to a newer release; the old version keeps serving.
go run ./tools/distbuild install --archive /tmp/releases/can-0.2.0-*.zip \
  --sha /tmp/releases/can-0.2.0-*.zip.sha256 --root /tmp/can-root --update
```

Install-root layout:

```text
versions/can-<version>-<target>/   immutable installed bundle
current -> versions/...            atomically swapped selection link
.can-lock                          installer serialization lock
```

The installed launcher resolves through the real executable path exactly
like the development sidecar, so running processes keep their open file
handles while new launches enter the new tree. Updates never rewrite a
live version, never remove old versions, and never call `bun upgrade`;
any verification failure preserves the previous selection. Only
installer-owned `.can-stage-*`/`.can-current-*` staging names are ever
removed, via `PruneStaging`; foreign files are left alone and unowned
patterns refuse.

Each refusal names its cause: detached-record mismatch or malformed
record, archive escapes/absolute paths/symlinks/non-regular entries,
case-aliasing names, more than one top-level version, manifest or target
mismatch, modified or unknown files, non-arm64 or unpinned runtime,
missing executable bit, unstamped launcher, symlinked or foreign-owned
root, and selection escaping the install root. The inspection transcript
records read-only `codesign` display and entitlement output for the pinned
runtime as evidence; verification itself gates on hashes and manifest
completeness, never on signature content.

Upstream obligations ship inside every bundle under
`distribution/notices/`: the Bun license, HTMX 4.0.0 with its lock, the
acorn parser with its lock, the four Jridgewell source-map packages with
their lock, and the pinned `pg_query_go/v6` CGo binding with the
PostgreSQL/libpg-query/protobuf licenses recorded in
`sql-binding.lock.json`, consistent with `go.mod`/`go.sum`.

## Prepared signing commands (not executed)

Actual signing, notarization, and release upload require authorized
credentials and a separate user authorization; this task performs none of
them. When that authorization exists, the reviewed sequence is:

```sh
# 1. Sign the installed version tree (identifier/identity supplied then).
codesign --sign "$DEVELOPER_ID_APPLICATION" --timestamp --options runtime \
  --entitlements /path/to/can.entitlements \
  /path/to/install-root/versions/can-<version>-<target>/bin/canlc
# 2. Archive the signed tree for notarization.
ditto -c -k --keepParent \
  /path/to/install-root/versions/can-<version>-<target> /tmp/can-submit.zip
# 3. Submit and wait (credentials supplied then, never stored here).
xcrun notarytool submit /tmp/can-submit.zip --keychain-profile "$PROFILE" --wait
# 4. Staple the ticket to the shipped tree and verify the gate.
xcrun stapler staple /path/to/install-root/versions/can-<version>-<target>/bin/canlc
spctl -a -t exec -vvv /path/to/install-root/versions/can-<version>-<target>/bin/canlc
```

Until those steps run with real credentials on a clean machine, the
bundle is a qualified unsigned release: install/run/update behavior is
proven, and publisher signature plus notarization stay a reported open
gate. No release upload occurs merely because this documentation exists.

## Release notes

Candidate: `can-b5026cc-darwin-arm64-v1` lineage (versions derive
from `git describe`; no tags exist yet, so installer versions read
`can-<sha>-darwin-arm64-v1`). Admitted target only:
darwin-arm64. No Linux support is claimed.

Shipped: Go launcher (`canlc`) with parse/resolve/check/emit plus
stdio LSP; the exact pinned Bun 1.4.2 sidecar
(`744846f84`, archive sha256 `90987a3a…6be1`); the closed
native catalogue (21 packages, 144 operations); the private TS
runtime; packaged HTMX 4.0.0; eight maintained examples; offline
install/update with versioned roots and current-symlink swaps.

Gates: the [verifier](../.github/workflows/verifier.yml) runs
gofmt, vet, cataloguegen, modcheck, gramcheck, the full Go suite,
and the 850-test Bun suite with PostgreSQL 17, pinned Chromium,
and TypeScript 7.0.2 operated — zero skips. The
[tsc gate](../.github/workflows/tsc.yml) typechecks 504 freshly
emitted files. Conformance (14 native checks) runs inside
`qualify.py`. Full results: [I45 acceptance](../docs/implementation/evidence/2026-09-21/i45/README.md).

Open gates: publisher signature, notarization, and release upload
(prepared commands above; no credentials authorized). The
installer runs from the Go source tree; a standalone installer
binary is future work.

Limits: Bun 1.4.4+ features refused at qualification; SQL is
PostgreSQL 17 wire behavior through pooled `Bun.SQL` with
statement timeouts and no migrations; the browser boundary is
upstream HTMX only with no authored client JS; AI evidence is
loopback stub data only — no provider quality is claimed; proof,
termination, and effect inference are not offered.

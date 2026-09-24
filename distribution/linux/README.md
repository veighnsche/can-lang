# Linux amd64 target (T19)

Official Bun 1.4.2 for Linux x64, packaged on Debian 13 amd64/glibc.
[Provenance](provenance.json) records the archive, executable, base-image,
toolchain, and PostgreSQL digests; the compiler pins the runtime in
[../target-linux-amd64.json](../target-linux-amd64.json). Builds never
cross targets: macOS hosts build the darwin bundle, Debian 13 amd64
hosts build this one.

## Reproduce

From the repository root, with the pinned archive acquired separately:

```sh
BUN_ARCHIVE=/absolute/path/bun-linux-x64.zip ./distribution/linux/run.sh
```

`run.sh` verifies the archive against the provenance record, builds the
pinned image, releases one versioned bundle, installs it into a fresh
root, and smokes the installed artifact with networking denied. The
PostgreSQL leg then runs against a disposable `postgres:17` container.
Release artifacts land in `out/linux`, install state and reports in
`out/linux-work`.

The equivalent manual steps, from the repository root:

```sh
export BUN_ARCHIVE=/absolute/path/bun-linux-x64.zip VERSION=t19-1
docker build --platform linux/amd64 -f distribution/linux/Dockerfile -t can-linux/t19 .
mkdir -p out/linux out/linux-work
docker run --rm --platform linux/amd64 \
  -v "$BUN_ARCHIVE:/archive/bun.zip:ro" -v "$PWD/out/linux:/out" \
  -e BUN_ARCHIVE=/archive/bun.zip -e VERSION="$VERSION" \
  can-linux/t19 /src/distribution/linux/build.sh
docker run --rm --platform linux/amd64 --network none \
  -v "$BUN_ARCHIVE:/archive/bun.zip:ro" -v "$PWD/out/linux:/out:ro" -v "$PWD/out/linux-work:/work" \
  -e BUN_ARCHIVE=/archive/bun.zip -e VERSION="$VERSION" \
  can-linux/t19 /src/distribution/linux/smoke.sh
```

The committed Go suite mirrors the smoke matrix without `sandbox-exec`;
run it on the target (or in the image with the archive mounted and
networking denied):

```sh
docker run --rm --platform linux/amd64 --network none \
  -v "$BUN_ARCHIVE:/archive/bun.zip:ro" \
  -e CAN_BUN_ARCHIVE=/archive/bun.zip \
  can-linux/t19 sh -c 'cd /src && go test -count=1 -timeout 20m ./distribution/ && go test -count=1 -timeout 50m -run TestLinuxInstalledArtifactSmoke -v ./tests/integration/'
```

The `Bun.SQL` roundtrip is a separate test because it needs database
networking, which the smoke denies by construction. Point it at an
installed root and a disposable PostgreSQL 17:

```sh
docker run --rm --platform linux/amd64 --network can-t19-smoke \
  -v "$PWD/out/linux-work:/work" \
  -e CAN_LINUX_INSTALL_ROOT=/work/install-root \
  -e DATABASE_URL=postgres://postgres:smoke@can-t19-pg:5432/smoke \
  can-linux/t19 sh -c 'cd /src && go test -count=1 -run TestLinuxPostgresRoundtrip -v ./tests/integration/'
```

## What the smoke proves

- Install stages the release into a fresh root, verifies the staged
  tree against its manifest, and selects it through `current`.
- `runtime-check` reports Bun 1.4.2 at the pinned revision on
  linux/x64 from the installed sidecar.
- `qualify.py` checks the archive and executable digests, stages the
  sidecar with an empty home, no ambient config, `PATH` without Bun,
  and denied networking, then runs the native API and behavior probes
  plus the nine rejection tests.
- `canlc assert` and `canlc run` pass on the process, files, crypto,
  and sqlite examples: a real `/bin/echo` child, a real directory
  listing, argon2id match/mismatch, and file-persisted SQL rename and
  fetch. Rebuilding reports a stable build ID.
- The PostgreSQL leg roundtrips a row through `Bun.SQL` against a
  disposable server and records its version.

## Limits

- The admitted host is Debian 13 (`ID=debian`, `VERSION_ID>=13`) on
  amd64 with glibc; derivatives refuse at verify and launch time.
  Publisher signatures and release upload remain open gates, as on
  macOS. The installer runs from the Go source tree.
- Debian apt packages float with the pinned base digest; `build.sh`
  records the resolved versions in `run-report.txt` rather than
  claiming a fully locked userland.
- Final product qualification waits for the T17/T18 gates per the
  task list; this lane proves packaging and installation only.

# I39 acceptance — signed offline distribution, install and updates

Closed 2026-09-21 for the offline-qualifiable scope. A built bundle
releases to one deterministic zip plus a detached SHA-256 record and a
read-only inspection transcript; install stages the zip into a
user-owned root, verifies the staged tree against its manifest, and
publishes it beside older versions under an atomically swapped `current`
symlink selected through the proven I02 real-path resolution. Updates
add verified versions without touching live bytes and preserve the
previous selection on any failure. No step downloads, uploads, signs,
notarizes, auto-acquires, or bypasses Gatekeeper.

Baseline `dc0a8da` (I38) plus the I39 worktree. Apple M4 (Mac16,12)
darwin/arm64, macOS 27.0, Go 1.27.1, Bun 1.4.2 (pinned archive sha256
`90987a3a…6be1`, hash-verified by the distribution builder).

## Design consultations

[i39-jev](../i39-jev/decision.md): three fresh Jev consultations (all
prose rewritten, same facts/options). Unanimous and strong for
zip_archive (1.0, 0.99, 1.0): one shippable file with one hash.
Unanimous but uneven for pointer_file (0.48, 0.89, 0.59) over
symlink_swap; investigated and decided on the merits for symlink_swap:
the selection entry is installer-created rather than
attacker-controlled (archive symlinks are rejected unconditionally
either way), and the link reuses the proven I02 real-executable-path
resolution while a pointer file would add a new dispatcher binary to
the launch path. Unanimous and strong for record_only (0.94, 0.97,
0.84): inspection transcripts are evidence while verification gates on
hashes and manifest completeness. Judgments are advice; the checks
below are the proof.

## Implementation

- `distribution/verify.go`: strict bundle verification shared by
  release and install (platform/arch/OS gates, strict manifest shape
  and identity, required assets, exact per-file hashes with no unknown
  files, no symlinks, arm64 Mach-O runtime matching the pin with its
  executable bit, target record identical to the compiled pin, and a
  launcher stamped with the manifest digest).
- `distribution/release.go`: deterministic zip packaging (sorted
  entries, fixed timestamps) plus shasum-form detached record and
  inspection transcript; renamed, tampered, or already-published
  inputs refuse.
- `distribution/inspect.go`: credential-free read-only `codesign`
  display and entitlement transcript over the hash-pinned runtime.
- `distribution/install.go`: ownership/containment root validation,
  detached-record check, malicious-archive refusal (escapes, absolute
  paths, symlinks, non-regular entries, case aliasing, multiple
  tops), staged-tree verification, flock-serialized atomic publish
  and symlink swap, owned-staging-only `PruneStaging`, and
  containment-checked `Selection`.
- `distribution/update.go`: previous-selection capture with
  structural preservation plus a post-failure re-check; old versions
  are never modified or removed.
- `tools/distbuild`: `--release-out` after bundle builds and an
  `install` mode (`--archive/--sha/--root/--update`); default bundle
  behavior unchanged.
- `.github/workflows/release-qualify.yml`: build, release, install,
  offline smoke, and release test suites with the records uploaded as
  CI evidence; no signing or upload steps exist.
- `distribution/README.md`: install-root layout, commands, refusal
  catalog, upstream notice inventory, and the prepared (never
  executed) signing/notarization command sequence.

## Verification

- `go test -count=1 ./...`: all packages pass, including the new
  distribution unit tests (version comparison, 12 bundle-verification
  refusals, sha-record shape matrix, release round trip with
  byte-determinism and overwrite refusal, install round trip with
  reinstall refusal, 6 crafted-archive refusals, prune behavior, and
  update preservation across a held-open runtime) and the 6-version
  concurrent install/update race (every version verifies, selection
  stays valid).
- `TestReleaseInstallUpdate` / `TestInstallRootRefusals`: two real
  bundles release deterministically with transcripts binding the pin;
  the installed tree runs `runtime-check` offline through the
  `current` symlink under `sandbox-exec` with `PATH=/nonexistent`
  from an unrelated cwd; 8 tamper/crafted/edited inputs refuse with
  specific causes; update holds the old runtime open and the old
  bytes and old binary stay intact while selection moves; a tampered
  update preserves the previous selection; 8 racing updates land
  exactly once; prune leaves foreign files alone; symlink roots
  refuse before any archive is read; the source tree is byte-identical
  afterwards.
- `bun test runtime/test/`: 850 pass, 0 fail, 63,285 expectations
  (no runtime changes; regression only).
- `cataloguegen --check`, `modcheck`, `gramcheck`: all pass.
- `gofmt`/`go vet` clean on every touched file.

## Remaining gate (explicit)

Actual codesigning, notarization submission, release upload, and a
clean-machine quarantined first run were not performed: no signing
credentials were available or authorized, and the task forbids
credentialed signing and release uploads. The prepared command sequence
in `distribution/README.md` plus the `release-qualify.yml` evidence
artifacts (sha record, inspection transcript, test reports) are the
handoff for that gate. Until it runs, the output is a qualified
unsigned release — install/run/update behavior proven, publisher
signature outstanding.

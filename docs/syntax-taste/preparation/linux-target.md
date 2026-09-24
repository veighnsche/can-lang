# Linux qualification target for the planned SaaS gate

24 September 2026 · technical target selection for P8/P10, **not a qualified
release**.

Select **Debian 13 amd64 with glibc, running the official Bun 1.4.2 Linux x64
archive** as Can's first Linux service qualification target. The intended Can
distribution target identifier is `bun-1.4.2-linux-x64-glibc-debian13-v1`;
the exact release artifact checksum, Debian image digest and qualified patch
level must be frozen by the implementation/qualification task. At this review,
[Debian 13.7](https://www.debian.org/releases/trixie/) is the current Debian 13
point release, Debian lists amd64 as supported, and the release is supported
through its stated lifecycle. The [Bun installation documentation](https://bun.sh/docs/installation)
lists Linux x64 glibc binaries and their CPU/glibc requirements. A read-only
HTTP HEAD of the [official Bun 1.4.2 Linux x64 archive](https://github.com/oven-sh/bun/releases/download/bun-v1.4.2/bun-linux-x64.zip) returned
200 on 24 September; existence is not provenance or runtime qualification.

This chooses one concrete service environment rather than promising all Linux
distributions. It aligns with the current pinned Bun version while permitting
a separate Linux archive and build. The existing [distribution implementation
and evidence](lifetime-deployment-evidence.md#distribution-and-linux-evidence)
are Darwin ARM64-specific; no Linux binary, launcher, isolation, verifier,
installation or service scenario has passed yet. An implementation task must
pin official archive bytes and revision, build and verify the Linux launcher,
replace macOS `sandbox-exec` with an explicit Linux isolation policy, check
native APIs and behavior on installed bits, and run the invoice, webhook,
ownership and host-shutdown matrix under that environment. The host policy
must state grace period, close deadline and forced-exit behavior without
claiming arbitrary work was cancelled or a committed effect was rolled back.

Other architectures and distributions remain outside this first qualification
claim until separately selected and tested. The selected target is a planning
input, not a statement that Can currently supports Debian 13.

# I39 distribution decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round, same
questions, options, order, and measured facts) advised the release
container, the version-selection mechanism, and the inspection strictness.
Requests, responses, and the equivalence audit live in this directory.

Unanimous and strong for zip_archive (1.0, 0.99, 1.0): the release is one
zip archive plus a detached SHA-256 record. A single shippable file with
one hash beats a scattered directory; modes round-trip and the bundle
holds no symlinks, so zip carries everything. Followed.

Unanimous but uneven for pointer_file (0.48, 0.89, 0.59), with round 1
nearly tied against symlink_swap (0.30): investigated, decided on the
merits for symlink_swap. The selection entry is installer-created, never
attacker-controlled: archive symlinks are rejected unconditionally either
way, so the symlink-suspicion behind pointer_file does not apply to this
entry. The symlink reuses the proven I02 real-executable-path resolution
(the sidecar suite already invokes through a symlink), while a pointer
file would add a new dispatcher binary plus pointer-content validation to
the launch path. The installer still validates the link target for
containment after every swap.

Unanimous and strong for record_only (0.94, 0.97, 0.84): read-only
codesign/entitlement inspection is recorded as release evidence while
verification gates on hashes and manifest completeness alone. Byte
identity is what the hashes prove; a signature observation must not
override it, and Gatekeeper's own first-run verdict stays outside these
tests. Followed.

These judgments are design advice, not verification. Assembly,
verification, install, update, refusal, and concurrency checks remain
required before I39 can be marked complete.

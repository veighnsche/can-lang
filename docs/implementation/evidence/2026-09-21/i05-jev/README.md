# I05 identity consultation

Three fresh requests compared semantic root/dependency/package ownership,
machine-absolute paths, and loading-order enumeration. All context, instructions,
and option descriptions were rewritten, with pairwise inequality checked by the
submission script and equivalent facts/constraints/alternatives reviewed before
sending. Requests and responses are retained verbatim.

All three `jev-1.13.0` responses selected `semantic`, each with probability and
confidence 1.0. There was no disagreement. Agreement is advice, not proof or a
guarantee of bias removal.

The implementation follows semantic ownership with bounded, domain-separated
hashed output components and explicit collision refusal. Review refined file
identity from the proposed basename to the complete logical source-root-relative
path: a confined source symlink can give a canonical package multiple logical file
paths, and canonical target basenames are absent from P2's source digest. The
logical path is therefore the correct deterministic input. Canonical paths remain
the authority for confinement and internal-package access. Relocation, same-basename
addition, forced output-collision, and symlink-target-renaming tests establish these
properties independently of the consultations.

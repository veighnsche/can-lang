# Current inert project loader

`Load(directory)` reads only that directory's `can.project.json`, optional root
`can.lock.json`, explicitly mapped dependency manifests, registries, assets, and
source trees. It performs no parent search, package download, environment
expansion, script/hook execution, SQL execution, or lock/registry rewriting.

Manifest, registry and lock decoders reject duplicate/unknown fields, wrong JSON
shapes, invalid UTF-8, unpaired escaped surrogates, and non-integer ID tokens.
Paths must already be normalized relative paths without `..`. Reads use realpath
confinement to the owning manifest directory. Contained symlinks are allowed;
escapes, source aliases and directory cycles fail. SQL entries have P12's exact
data shape; actual PostgreSQL statement validation remains the native SQL task.

Dependency edge names are parent-local: the same edge name in different
manifests may reach different instances. Each manifest may declare a `project`
lineage ID. A declared lineage names exactly one real directory; a second
directory claiming the lineage fails even when byte-identical, as does a
root/dependency lineage collision. Instances without a lineage take a legacy
edge-path identity from first discovery in sorted edge order. Identical real
paths intern to one instance however many edges reach them, while manifest
cycles still fail.

The root lock pins the root's direct edges plus every transitively reachable
instance by canonical node identity: lineage, content digests, registry
snapshot, and each instance's own direct edges. Edited manifests, sources,
fixtures, registries, renamed lineages, retargeted or repathed edges, and
missing or unused entries all fail verification. Manifest bytes are hashed
exactly. Source hashes use P2's `can-source-tree-v1` NUL prefix and big-endian
length framing over byte-sorted logical `.can` paths and exact content.
Registry snapshots are compared structurally; active allocations must match each
project's source declarations and active/retired IDs must be globally disjoint.

Source folders form per-instance packages. Canonical source folders determine
ownership, including `internal` access boundaries for a symlinked source file.
Package names must agree within a folder, be unique within one instance, and
avoid reserved catalogue names; the same name composes across instances.

Nominal package identities are `can.project.root/<package>`,
`can.project.lineage/<lineage>/<package>`, or
`can.project.dependency/<edge-path>/<package>`. Declaration IDs append `::name`.
File IDs append the normalized source-root-relative logical path used in the
source digest. Absolute paths and load order never enter these identities.
Package output folders are `packages/p-<sha256>` and files are `s-<sha256>.ts`,
using separate `can-package-path-v1` and `can-source-path-v1` NUL-prefixed hash
domains. A reverse mapping refuses conflicting identities even under case folding.
The fixed ASCII components avoid reserved names and filename-length problems.

Changing a symlink target's basename without changing its logical source path or
bytes does not change source/output identity. Its canonical location still governs
security. This distinction is covered by a regression test, along with relocation
and unrelated same-basename additions.

The loader returns snapshots and planned relative output paths; it creates no
output files. Ownership, generation publication, cleanup and emitted imports belong
to I09. `canlc inspect-project DIRECTORY` exposes the loader plus declaration and
signature resolution as a versioned inert report, not a compile/run operation.

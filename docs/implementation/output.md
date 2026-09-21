# Owned ESM output generations

The I09 output driver requires Go 1.25 or newer for directory-relative `os.Root`
operations. The admitted native distribution remains the pinned macOS arm64 Bun
target. Unsupported locking platforms fail explicitly.

`BeginOutput` locks the real project directory with a nonblocking OS lock, then
loads a project snapshot. Concurrent compiler processes cannot both publish.
The lock disappears when its process exits; no PID file needs stale-lock guessing.
Canonical roots and source/configured input paths must not overlap `dist`.
Missing or empty real `dist` may be claimed; nonempty unowned output is refused.

The owner marker binds the canonical project directory. It is rechecked before
mutation and lease acquisition, along with directory identity and symlinks.
Regular outputs are opened without following a final symlink. Rooted operations
confine access even when a directory is renamed. Cleanup validates complete
inventories, rejects unknown/modified/nonregular entries, and unlinks only known
manifest paths. It never calls recursive `RemoveAll` on project output.

## Content and publication

A generation ID hashes source, dependency, catalogue, compiler, runtime and option
identities together with the exact artifact hashes, entry and import inventory.
Source/dependency identities also include registry and configured asset contents.
Absolute project paths, process IDs and random staging names are outside semantic
content identity. A source/manifest/registry/asset change during a build causes
publication to refuse the stale snapshot. The snapshot is checked again immediately
before replacing current, for both staged and reused generations.

Generated modules use structured imports rendered as exact relative `.ts` paths.
The complete graph includes type-only imports even though Bun erases them. Path
claims include directories, reserve `manifest.json` and its descendants under any
case spelling, and reject case aliases, device names, traversal and
file/directory conflicts. Assets bind their bytes through their digest path;
source maps must match their generated module and basic version-3 structure.
Detailed source-map generation belongs to I40.

Private runtime modules come only from the hash-verified distribution inventory
in `runtime/modules.json`. One namespace, derived from target metadata and module
hashes, is copied into each generation; all source modules share its registries.
Only these verified modules can import the finite admitted native module set.
Authored TypeScript, npm discovery and user host-module registration are absent.

Before publication, the pinned Bun transpiler parses every generated/runtime
module without executing it or resolving dependencies. Executable import edges
must match the emission inventory. A pinned, bundled Acorn parser additionally
checks Bun-transformed JavaScript ASTs, refusing computed imports (including
nested functions) that Bun’s import scan does not enumerate. Static re-export
edges are also checked. This pass performs no execution or dependency resolution.
Structured type-only edges are checked by the
Go graph even when erased by the native parser. This validates compiler-generated
fragments; it is not an API for admitting arbitrary authored TypeScript.

`pending.json` durably reserves a fresh `builds/.stage-<random>` directory and its
planned manifest before staging begins. Files are created exclusively and synced;
the complete tree is checked before rename to `builds/<build-id>`. `current.json`
is atomically replaced and synced last. Failed compilation/validation cannot make
partial output current. Recovery can reclaim only the reserved stage's known
paths, including interrupted partial writes; unknown files cause a refusal.
A generation already renamed before interruption remains an ordinary validated
orphan until publication or pruning.

The owner schema reserves `.can-owner.json`, `builds`, `current.json`, `pending.json`,
`.current.tmp` and `.pending.tmp`. The two fixed metadata temporaries are never
promoted after interruption. They are compiler-owned regular files; unexpected
paths elsewhere are preserved. Ownership markers are not authentication against
someone deliberately rewriting the entire owned tree with filesystem access.

## Runs and cleanup

Run acquires `current.json` by identity and manifest digest; it never selects the
newest directory. It checks the generation inventory and selected distribution's
runtime identity, then holds a shared OS lock on the immutable generation
manifest. The generated child inherits that descriptor on fd 4, so its lease
survives the launcher's closure or death. fd 3 retains the existing isolated
application-environment channel. An unused environment pipe does not turn a
successful program into a failure.

Pruning takes exclusive manifest locks and skips active runs. `clean PROJECT`
unpublishes current and deletes inactive owned generations while preserving live
ones. Cleanup preflights candidates before deleting files. A later build of the
same inputs produces the same semantic manifest and content. Removed source
modules cannot remain reachable through the new graph.

The old `--out` emitter is excluded from the shipped compiler. Historical emitter
fixtures live only in `_test.go` files. The current emitter/driver interfaces are
exercised by native integration tests; full current-language `build`/`run` command
routing follows I10/I11, rather than falling back to the predecessor compiler.

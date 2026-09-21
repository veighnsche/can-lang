# I05 validation

Completed 2026-09-21. The current project loader validates exact inert JSON,
confined local paths, dependency graph and lock digests, registry allocations,
flat packages, file-local imports, exports, eligible-kind names and receiver
ownership. `canlc inspect-project DIRECTORY` produces a versioned resolution
report without executing initializers or writing outputs.

Package/declaration identities are independent of checkout location. Source IDs
use the logical source-root-relative paths hashed by the lock; canonical paths
govern confinement and internal visibility. Planned output components are fixed
ASCII domain-separated hashes, with explicit collision refusal. Same basenames
and declaration names in distinct packages remain independent.

Validation:

- [Full Go suite](i05-go-tests.txt): passed.
- [Detailed project/resolver tests](i05-project-tests.txt): passed, including
  malformed JSON, duplicate names, symlink escapes/aliases, stale lock data,
  undeclared imports, private signatures and ineligible names.
- [Final targeted suite](i05-final-tests.txt): passed after deterministic
  diagnostic ordering was added.
- [JSON fuzz run](i05-json-fuzz.txt): 99,310 executions, no failures.
- [Staged offline integration](i05-offline-integration.txt): passed using the
  pinned archive, a relocated release, network denial, unrelated working
  directory, unavailable PATH and hostile ambient configuration. Two relocated
  projects produce identical reports; stale dependencies fail and inspection
  leaves source trees unchanged.
- [Versioned example report](i05-project-report-v1.json).
- [Three Jev consultations](i05-jev/README.md), including the later tested
  refinement of logical source identity.

This step supplies declaration/signature scopes. Concrete body type checking is
I06 onward; generated output publication is I09 and native SQL validation is
I35/I36. The predecessor loader is explicitly named legacy and is not used by
current project inspection.

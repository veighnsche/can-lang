# Concrete current-language types

`Builder` resolves explicit annotations through the I05 file scope and interns
concrete nodes. `Finish` validates the complete graph before any node becomes
compatibility evidence. A failed builder stays failed. `CheckDeclarations` seeds
ordinary declarations and validates declaration/signature/top-level annotations,
including parameter-independent errors in unused generic templates. It does not
execute expressions or claim to check function bodies.

Nominal identity includes the canonical declaration ID and invariant concrete
argument IDs. Structural array/callable/arm identities include their full checked
contracts; callable error sets are sorted and duplicate bounds fail. JSON array
framing plus domain-separated SHA-256 keeps nested identities bounded; the
original key is retained so a collision cannot silently merge nodes. Returned
field/argument/leaf slices are copies. Primitive identities have no machine paths.

Variants flatten to finite, disjoint nominal record/error leaves. Only the
catalogue-owned `standard_failure` opaque identity receives the specified leaf
exception. Variant-only cycles fail, even if another alternative is a base.
Finite inhabitation is a least fixed point: every record field must be inhabited,
any variant leaf suffices, and empty arrays provide a base. Callable/resource
values are admitted data. Equality separately walks every reachable data field;
a visited recursive node does not hide a later callable or opaque exclusion.

Compatibility allows leaf/narrower-variant inclusion and callable error-bound
widening. It never widens existing arrays, nominal generic arguments, or callable
input/results. Different phantom arguments of one generic variant remain
invariant even if their current flattened leaves coincide. Copy-update validation
requires at least one known unique field, compatible replacements, and the exact
ordinary record/error receiver specialization. Opaque values cannot be constructed
or updated through ordinary data emission.

Catalogue field/leaf descriptors use the catalogue's existing inventory parser,
then substitute resolved nodes. They never pass uppercase template placeholders
through the authored parser. Closed data/map-key/failure-variant constraints are
validated before publishing a graph. `MergeGeneratedFields` shares ordinary and
explicit metadata field collision checks with future native declaration checking;
it adds no implicit reserved field names.

Concrete expansion is bounded at 256 simultaneously active distinct instances
of one declaration and 16,384 graph nodes. The diagnostic explicitly says this is
an implementation limit, not proof of infinite expansion. Same-instance recursion,
finite parameter permutations, transient growth followed by stabilization, and
long acyclic chains are tested. I46 owns complete reachable generic body checking
and inference; these foundations do not guess arbitrary parameter substitutions.

`canlc inspect-types DIRECTORY` exposes a deterministic versioned declaration
report without evaluating initializers or writing generated code. I07+ adds
expression/completion passes, and I09 publishes complete generated modules.
The predecessor `legacy_types.go` and `legacy_result.go` remain isolated in the
old pipeline until its scheduled retirement; current types do not use their
string shapes or old success wrappers.

`ArrayOfChecked` derives a structural array type from sealed data for expression
inference. It cannot create new nominal declarations or recursive shape edges,
does not reopen the original graph, and uses the same framed identity as an
explicit array annotation.

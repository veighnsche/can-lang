# Package tables and eligible-kind lookup

`Build(projectGraph)` registers every package declaration before checking exports,
file-local imports, and signatures. Package import cycles are allowed. Dependency
manifest cycles are not. A package may import its own project's packages and the
closed catalogue unqualified, and a direct dependency's packages only through a
qualified `edge::package` entry with an optional `as` alias; transitive
dependencies do not become implicit direct imports, and unqualified names never
resolve into a dependency. Two instances may expose the same package name;
local aliases disambiguate them without entering canonical symbol identities.

Provides entries must name declarations in that exact file. Duplicate declarations
across kinds, duplicate exports, alias collisions, reserved prelude shadowing,
private exported signature types/errors, and canonical `internal` boundary escapes
fail. Generic parameters are installed before a return-first signature is resolved.
Receiver and input names share their collision domain with those parameters.

Each `File` retains its own alias table. Qualified lookup consults only that table;
local values cannot redirect it. Unqualified `Scope.Lookup` skips declarations that
are ineligible for a type, constructor, call, value or error position. This permits
a value named like its record type, and lets a callable-valued binding shadow an
outer function without treating an integer binding as callable. Opaque catalogue
types and variants do not become constructors. Prelude names remain reserved.

`LookupWhere` lets later type checking determine eligibility for concrete generic
specializations. The resolver does not guess that an unconstrained type parameter
is callable. `World.Functions` supplies the registered signature scope; later body
checking creates child scopes for locals, matches and handlers using the same
duplicate/prelude/eligible-kind rules. Pattern narrowing and full body references
require the I06+ type graph and are not claimed by project inspection.

Methods remain named package declarations. Only a project's nominal record owner
may declare its methods. Method lookup verifies receiver identity, export status,
and a file-local import of the receiver package. Methods cannot be called as static
functions with a hidden receiver. Catalogue methods remain closed operation
descriptors, not project extension points.

The old CLI loader and stem assignment are explicitly named `legacyParsePaths` and
`legacyAssignStems`. They serve the predecessor pipeline pending its scheduled
retirement. Current project resolution does not adapt to their module objects or
basename/counter output policy.

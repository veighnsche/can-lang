# Completion regions and protected async values

`check.CheckRegion` checks one named function or one selected handler against
sealed declaration types, file-scoped eligible-name resolution and an explicit
success/error contract. Functions have no parent; handlers name their enclosing
region and use their selected local result type. Every terminal IR node names
its region. The emitter refuses a terminal node owned by a different region.

Nested ordinary matches, call/chain matches, relay and `do` retain that identity.
Ordinary value matches produce synchronous value temporaries and cannot contain
completion arms. Terminal constructs cannot initialize locals; void success uses
bare `ok`, and a standalone call step requires void with `emits []`. The parser
rejects statements after completion; the checker also rejects missing terminals
and malformed region/context evidence.

A completion match has exactly one success arm, one arm for every concrete domain
error in the complete call or chain bound, and at most one optional standard arm.
Bare error names resolve against that exact bound, so multiple specializations of
one error declaration cannot be merged by a bare arm. Forwarding preserves the
original completion and occurrence. A constructed error creates a new occurrence;
`ok error(...)` remains successful error data when admitted by the success type.
Standard failures never enter a domain bound.

Locals become visible after initialization. Each block/arm receives a child scope;
chain success bindings are visible to subsequent calls and the success arm only.
The body pass supplies resolved use evidence to I48's finite local diagnostic.
The initial allowed domain bound and the actual escaping domain set are separate:
`Region.Errors` is permission, while `Region.Escapes` contains only errors emitted,
relayed or forwarded out by the checked body. Handled participant errors do not
appear merely because the invocation declared them.

## Native emission

Generated functions are async and return `Completion<T>`. Calls await a protected
carrier, inspect its category synchronously, and only then extract data. Native
operator evaluation remains ordinary JavaScript/Bun; there is no Can interpreter.
Argument preparation, receiver evaluation and reached method/chain steps are
inside the invocation catch. Selected arm bodies are outside it. A failure from
a selected arm therefore leaves the active region without entering another arm
of the same match. Standard occurrences retain their identity across catches.

Arguments and receivers evaluate once in source order. Literal spreads can cover
fixed positions; a runtime-length spread cannot supply a fixed input. Resolved
variadic declarations carry a final array parameter in the private ABI and a
separate declaration flag; their trailing values/spreads are packed immutably.
Splitting a literal spread at that boundary refers to one prepared array.
Source callables, generics and native state groups still require their scheduled
owning passes; the region checker does not guess capture/specialization/provider
contracts. Existing native array/string slice lowering remains available.

`RegionTypeDeclarations` collects signature and derived expression types for
strict TypeScript aliases. This includes arrays derived outside the initial
nominal declaration model. These annotations do not replace Can's nominal checks
or authorize structural source assignments.

## Pattern checking

Ordinary matching is ordered. The typed pattern matrix checks usefulness and
exhaustiveness over booleans, nominal leaves and constructor products, written
integer interval boundaries, array length partitions, and literal/other partitions
for strings/floats. Wildcards/binders cover the remaining infinite spaces. Recursive
fields are decomposed only as far as written patterns require. Excessive product
work produces a finite diagnostic rather than invoking theorem proving.

Alternatives must bind the same names with identical types. A bare admitted leaf
narrows a named scrutinee when every alternative agrees on that leaf; otherwise
its existing admitted type is retained. Error leaf names also bind payload data.
Array remainder bindings receive frozen native slices. The emitter uses native
comparisons, property/index reads and ordered branches, with no runtime pattern
interpreter.

## Carrier boundary

`runtime/completion.ts` creates null-prototype frozen boxes with fixed own data
properties and no `then` key. Private WeakSets reject copied, forged and proxied
carriers without reading their properties. Application keys never become carrier
keys. A separate element box protects data crossing native mapper/collection
boundaries. Extraction functions are synchronous; generated async returns and
callbacks must rebox before returning.

All generated function categories fulfill as completion boxes. The native Promise
bridge converts success to fulfillment and domain/standard failure to rejection
with the original private failure box. Recovery restores the category/occurrence.
Native `all`, `allSettled`, `any` and `race` therefore see the required settlement
semantics without assimilating a record's legal callable `then` field. The
one-shot handler dispatcher invokes the selected handler separately and returns
its outcome directly, including a failure of the same kind as the input.

I11 wires these interfaces into the current CLI; I08 supplies captured named
callables, I12/I18 supply assertion execution/fixtures, and I19 supplies complete
coordination preparation, native aggregation and handler ordering. Those pending
tasks do not select a predecessor compiler fallback.

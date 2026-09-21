# Current ordinary data lowering

`Record`, `Array`, and `Update` consume checked concrete type evidence and already
checked synchronous expressions. They emit native TypeScript calls to the private
`runtime/data.ts` module; they do not parse source strings or publish generations.
The generation driver supplies its exact relative runtime import path in I09.
Generated helper bindings use `$`, which cannot occur in authored identifiers.

Records are frozen null-prototype objects with own data properties. A private
module symbol carries the canonical nominal identity, so field names cannot
collide with representation metadata. Arrays are frozen fresh native literals.
Copy-update evaluates the receiver once, evaluates replacement expressions once
in their written order, and delegates shallow copying to native Object.assign.
Unchanged immutable subtrees retain their identities and original aliases remain
unchanged. Ordinary errors use the same nominal data layout; domain completion
and occurrence handling belong to the error/region passes.

The native integration compiles expressions from two real packages with the same
source basename and record name, invokes the exact qualified Bun binary, and
checks strict Bun.deepEquals, evaluation order, and immutability. The runtime
conformance is also executed from the staged release with network denied.
No custom recursive equality routine is introduced.

Expressions here are synchronous payload operations. An authored callable field
named `then` stays legal; the callable/region pass must keep every native promise
crossing in private non-thenable completion boxes before it calls these operations.
This data representation alone is not a promise-assimilation defense.

# Boundary and tooling design space for an agent-first Can

## Status and constraints

This is a divergent brainstorm, not a specification, selection, implementation
plan, or claim of measured usability. Every syntax block is **PROPOSED**. This
part explores how agents express operations that cross application boundaries,
manage work over time, and use tools to construct and change programs.

The current language has exact `int` and finite exact `dec`, nominal source
types, explicit declared errors, exhaustive decisions, tests and bounded proof
admission. The current generated TypeScript exposes private layouts directly:
records are mutable objects, sequences mutable arrays, Bytes mutable
`Uint8Array`, decimals and brands strings, and callables JavaScript functions.
There is no recursive hostile-ingress validation or callable provenance check.
Primitive faults and resource failures live outside `emits`. Editor support is
diagnostics-only LSP plus regex coloring; `normalize` prints test outcomes, not
canonical source. These limitations matter more than surface punctuation.

Human ergonomics has zero independent weight here. Reviewability still matters
because an agent's change must expose its meaning to another agent, the compiler,
and acceptance authority. Compatibility with source spelling, generated layouts,
or old goldens is not an obligation.

## Four representations should be allowed to differ

“The Can representation” currently hides four separate choices. They should be
compared independently.

| Layer | Alternative 1 | Alternative 2 | Alternative 3 |
| --- | --- | --- | --- |
| Source | Canonical indentation text with explicit names and rows | Lossless typed tree serialized as stable nodes, with text as a projection | Declaration graph plus local decision DAG, edited through semantic operations |
| Private backend | Direct calls returning outcome unions | Uniform completion-returning calls, as in B05 A | Typed frames, slots, receipts and worlds, as in B05 B |
| Public host/embedding facade | Validated data-only roots with an independently chosen sync or completion return | Capability session with callable tokens and owned values | No generic facade: generated per-root command endpoints |
| Wire/persistence | Explicit schema codecs with decimal/int text | Canonical binary/value tree with schema identity | Event/receipt log that stores requests and outcomes, never runtime handles |

The useful combinations are not diagonal. A typed-tree source can lower to
ordinary direct calls. A private frame machine can expose one JavaScript
`Promise` at the root. An internal Promise ABI can still reject arbitrary public
objects and accept only decoded owned snapshots. A wire codec must not serialize
TS mangled names, closures, state cells, task slots, resource handles, or brands
merely because their private representation is a string or object.

### Source representation alternatives

**Canonical text.** Keep files as the authority, but make the grammar fully
deterministic and expose a real parse/format round trip. Agent hypothesis: one
uniform textual form makes diffs and recovery easy to inspect. Failure case: a
rename still requires token/span reasoning and may accidentally edit comments,
shadowed binders, pins, or evidence rows.

**Stable-node documents.** Give every declaration, arm, test row and call a
document-local semantic ID. Store a lossless tree; render `.can` for review.
Whitespace, comments and other proven-inert trivia can remain projection data
rather than semantic identity. Argument, branch and arm order—and any effect or
evidence order whose meaning is not proved commutative—remain semantic fields.

```text
PROPOSED protocol
open(module = "account") -> snapshot S42
replace(node = arm:7f2, field = rhs,
        value = call(error = auth.denied, args = []), against = S42)
check(snapshot = S43, scope = affected)
render(snapshot = S43, style = canonical)
```

Agent hypothesis: edits target semantic roles and stale snapshots fail rather
than applying at the wrong repeated spelling. Failure case: regenerating node
IDs after parse damage turns every small edit into a large tree diff or makes a
previous patch address a different arm.

**Graph plus decision DAG.** Store declarations, ownership and call edges in one
graph, with exhaustive matches as explicit partitions rather than nested text.
Render a conventional file only at review/export time. Agent hypothesis: “add
the missing product case” becomes a schema operation over the uncovered space.
Failure case: the projection can hide meaningful ordering such as eager argument
evaluation or branch launch. The graph therefore needs explicit order edges,
not an unordered bag of children.

**Content-addressed semantic cells.** Treat each declaration body, interface,
evidence table and dependency set as separately addressed content. A module is a
manifest of identities. Agent hypothesis: parallel agents can replace disjoint
cells with exact preconditions and obtain small semantic diffs. Failure case:
content hashes are terrible navigation names and can encourage false confidence
that equal hashes imply accepted authority. Human-readable names and explicit
acceptance signatures remain separate.

**Typed holes and edit transactions.** Permit incomplete source only inside an
editor transaction, never an executable world. A hole carries the expected type,
outcome space, in-scope identities, effects it may consume, and obligations it
must discharge. The agent requests a minimal context slice rather than receiving
whole files: owner interface, relevant callees, error producers, authority path,
and affected evidence. Commit is compare-and-swap on the snapshot and succeeds
only after every hole closes. Agent hypothesis: compiler-selected context reduces
irrelevant text and preserves semantic intent through half-built edits. Failure
case: an underspecified slice hides a transitive effect or public codec. A
`why-included` edge and a query for omitted dependencies are therefore required.

The compiler source of truth could likewise be (a) hand-written grammar and
semantic code, (b) one declarative registry generating parser categories,
formatter cases, visitors, diagnostic schema and TS runtime imports, or (c) a
typed core schema from which text grammar is only one adapter. An intrinsic
registry is particularly testable: each primitive names operand/result types,
exact evaluation, possible faults, canonical encoding, evaluator implementation,
TS helper, ownership behavior and package version. Failure case: a registry says
`seq.at` can fault while one backend helper silently uses JavaScript indexing.
Cross-backend generated vectors and package-manifest checks must expose that
drift; registry presence is not proof of parity.

## Host trust, ownership, and callable provenance

There are at least four plausible ingress policies.

1. **Explicit trusted embedding.** Generated private functions remain callable
   by a host that promises shape, canonical numbers, ownership, and callable
   behavior. Cheap, but the trust must be named in the artifact and diagnostics.
2. **Validated data-only roots.** Public roots recursively reject getters,
   unexpected prototypes/tags, cycles, noncanonical decimals, wrong bigint
   ranges where a declared range exists, aliases that cannot be owned, and all
   functions. Copy or transfer Seq/Bytes on entry and snapshot on exit.
3. **Capability sessions.** The host opens a world and receives opaque handles.
   Values cross through decoders; callable use requires a minted callable token
   naming provider, signature, effects, revision, lifetime and invocation quota.
4. **Generated command endpoints.** Each approved root has a bespoke request and
   response codec. No arbitrary function export exists. This maximizes audit
   clarity at the cost of more generated boundary surface.

Callable provenance cannot be inferred from a TypeScript function type, closure
shape, frozen object, or successful examples. One possible explicit shape is:

```can
PROPOSED public callback declaration
callback Price__Lookup(input: Sku) -> Money rev 3
  authority [catalog.read]
  provenance provider catalog_adapter@8
  calls at_most 20
  owns input snapshot
  returns owned
```

The corresponding public token could be unforgeable and session-scoped. It may
refer only to a registered adapter whose revision and effect summary match. The
private backend is free to lower it to a direct closure, table index, or message.
Discriminating failure cases include a host closure that captures logging or
network authority absent from the declaration; a valid token replayed in another
world; a callback retaining mutable Bytes; and a callback whose provider changes
without changing the acceptance dependency.

Exact-number policy should also be layer-specific. Source `int` stays
mathematical and `dec` stays canonical finite decimal. Private TS can use
`bigint` and canonical strings. Public ingress could require tagged strings
(`{int:"…"}`, `{dec:"…"}`) even when direct trusted embedding uses bigint.
Wire schemas could choose canonical text or a canonical magnitude/scale form.
Tasks should include a 500-digit integer, `-0.0`, redundant decimal zeros,
malformed signs, hostile objects with numeric getters, and round trips through a
host that cannot represent all integers natively.

Ownership deserves verbs rather than `readonly` types: `copy`, `take`, `borrow
for call`, and `opaque`. A transferred buffer must become unusable to its old
owner; a copied sequence must recursively define whether nested Bytes copy; a
borrow cannot escape through a callback or completed outcome. Test alias mutation
after success and after error, not just during the call.

## Wire declarations and evolution

Three wire-authority models deserve separate trials. **Explicit owned codecs**
make encode/decode ordinary reviewed declarations; they can normalize fields or
reject old data, but their brand disclosure and minting authority must be
declared. **Generated structural schemas** derive codecs only for an admitted
data subset; generation is cheaper to request but cannot infer brand authority,
callback meaning, resource identity, or migration policy. **Declared wire
projections** let a nominal source type select a distinct stable wire record and
require explicit functions between them.

```can
PROPOSED
wire Account__V2 schema "account" version 2 (
  id: wire.int_text
  balance: wire.dec_text
)
project Account -> Account__V2 using account__to_wire
restore Account__V2 -> Account using account__from_wire
  authority [Account__Id.mint]
```

One identity policy uses owner, stable schema name, version, field/tag identities,
exact-number encodings and canonical ordering for the wire schema. Record exact
codec/projection interface and executable identities separately, so a body-only
codec repair can preserve an unchanged wire schema while invalidating the right
execution evidence. Compare this with whole-bundle identity and its wider churn.
An artifact manifest can list every schema it reads/writes and the codecs bound
to it. Public sessions, persisted stores and event peers can reject a
mixed-build interaction before decoding, unless a named migration edge is
present. “Same TypeScript shape” is never a migration edge.

Discriminating tasks include adding an optional-looking field without an
authorized default; swapping variant tags; decoding noncanonical decimal text;
attempting to reconstruct a brand through an unprivileged generated codec;
placing a callback or resource handle in a structural schema; changing a codec
body without updating its executable dependency; and replaying an event under a
new schema ID. The compiler should explain whether failure is shape, version,
authority, ownership or migration, rather than returning one generic decode
error.

## Async, joins, settlement, and resources

B05 A and B establish distinct candidate contracts. A uses `match join`, labeled
outcome slots, permits native singleton joins, and proposes a uniform
completion-capable callable ABI. Its host deadline is a dynamic argument;
rejections map to declared errors; it does not claim cleanup after timeout. B
uses a named local join site, positional real-outcome slots, rejects redundant
native singletons, and proposes private frames, fixed slots, receipts, serialized
world roots and a separate `T/S` settlement contract. Contract breach faults the
world. Neither is implemented, neither settles Resources, and B's obligations
must not silently be imposed on every alternative before a backend is chosen.

Additional source/control models worth testing:

**Outcome-product expression.** Make concurrency a completed-outcome control
construct whose declared effects aggregate every branch; it is not a pure
combinator. Its only result is a completed nominal product, and matching remains
ordinary.

```can
PROPOSED
match completed join (
  account = call account__read(id) using [],
  quota = call quota__read(id) using [Quota__total.read]
)
  on (Ok a, Ok q) => Ok(a, q)
  on (account.missing a, Ok _) => user.missing(a)
  on (Ok _, quota.denied q) => user.denied(q)
  on (account.missing a, quota.denied q) => user.lookup_both_failed(a, q)
```

Benefit hypothesis: completion is visibly not a pending value, while the product
has one reusable semantic form for checking and tooling. Failure: if `completed`
becomes storable or returnable, it quietly creates first-class task results and
lifetime rules.

**Parallel region with named policy.** Separate collection from disposition.

```can
PROPOSED
region lookup policy collect_all
  account <- account__read(id) owns []
  profile <- profile__read(id) owns []
settle lookup with
  account Ok a, profile Ok p => Ok(a, p)
  account missing a, profile Ok _ => user.missing(a)
  account Ok _, profile unavailable p => user.profile_unavailable(p)
  account missing a, profile unavailable p => user.lookup_both_failed(a, p)
```

Benefit hypothesis: policy is explicit and compiler-generated cases can be
queried before source is complete. Failure: two keywords may falsely suggest a
gap where work can escape. The region and settlement must be one indivisible
semantic node.

**Workflow graph.** Admit a finite dependency DAG rather than nesting joins.

```can
PROPOSED
workflow checkout
  inventory = reserve(cart)
  price = price(cart)
  charge = charge(price.total) after [price, inventory]
  finish all [inventory, price, charge] by checkout__decide
```

This could expose initiation, dependencies, authority and complete outcomes
directly to agents. It also invites partial rollback, cancellation and resource
lifetime questions immediately. A `charge` that succeeds after `inventory`
fails is a discriminating case; no implicit compensation may be invented.

**Actor/resource protocol.** Concurrency belongs to linear protocol handles,
not calls. An acquired `Transaction<Open>` permits operations and must end as
`Committed`, `RolledBack`, or `CommitUnknown`. This is more appropriate for SQL
than pretending all effects are disjoint. HTTP body streams and mounted UI
components similarly require ownership across time. It can coexist with lexical
joins: resource protocols define lawful sharing/lifetime; joins define child
accounting.

```can
PROPOSED
using tx <- sql__begin(pool, context)
  match call sql__commit(take tx)
    on Ok receipt => Ok(receipt)
    on sql.commit_unknown detail => sql.commit_unknown(detail)
    on sql.constraint_violation retryable =>
      // This error explicitly returns Transaction<Open>; `tx` was consumed.
      match call sql__rollback(take retryable.transaction)
      ...
```

`using` must not imply rollback on every error: an uncertain commit makes that
claim false. Only a failure kind whose payload returns an explicitly open handle
can authorize the rollback above; reusing `tx` after `take` is a static error. A
resource design needs protocol-specific terminal states, explicit disposition
evidence, cancellation ownership, and cleanup faults.

Discriminating concurrency tasks should cover both failures, both terminal
orders, same-cell conflicts through transitive callees, repeated shared helper
sites, a large bigint computation that delays host progress, timeout followed by
late mutation, duplicate receipt, host reentry, nested joins, and root disposal.
The concrete customer should include bounded HTTP body consumption or SQL commit
uncertainty, not only two pure reads. A UI unmount with outstanding commands is
another strong lifetime test.

## Primitive faults, evaluation, and resource handling

Three fault models deserve comparison.

**Loud runtime faults.** Preserve a strict distinction: declared domain errors
are values; division by zero, bounds violations, impossible emitter states,
runtime contract violations and resource exhaustion fault the root/world. The
public facade returns a separate fault envelope with redacted trace and semantic
site identity.

**Checked fault effects.** Give selected primitives explicit finite fault kinds,
but keep host-contract and resource failures outside ordinary recovery. This can
make `seq.at` total without pretending out-of-memory is a domain error.

**Total primitives plus budgeted execution.** Replace unsafe index/division with
total library operations and reserve faults for compiler/runtime defects and
budget exhaustion. This increases source handling but may simplify differential
parity.

Evaluation order must be a first-class semantic query. F06 shows why: written
named-argument order and parameter order can select different first primitive
faults. Candidate rules are written order, parameter order, or “prepare all and
report a deterministic fault product.” The last is radical but prevents
first-fault loss; it also changes cost and can evaluate expressions that another
rule would never reach. Tests need two independently faulting arguments, alias
transfers, calls with resource acquisition, and nonfaulting controls.

Numerical resource contracts may be source budgets, deployment policy, or both:

```can
PROPOSED
budget Checkout__Limits (
  steps <= 200000
  bytes <= 8000000
  children <= 8
)

root checkout(...) under Checkout__Limits
```

A proof of termination does not prove these bounds. A budget result should say
whether it is statically proved, evaluator-measured, runtime-enforced, or host
trusted. Avoid turning `budget.exceeded` into an ordinary business error unless
the program can safely recover from a partially used world.

Clocks, Unicode tables, entropy, network progress and memory allocation remain
host/runtime resources however elegant the grammar becomes. A deadline needs a
named clock, monotonicity and progress assumptions; Unicode normalization needs a
versioned data provider and allocation limits; a random result needs authority
and byte ownership. These belong in runtime packaging and boundary manifests,
with revisions and conformance evidence, rather than implicit “standard” source
semantics. A grammar-only prototype cannot unblock HTTP, SQL or UI lifecycles.

## Runtime packaging alternatives

Runtime helper ownership is independent of source grammar and backend control
flow. A **shared versioned runtime** imports exact-decimal, ownership, fault and
scheduler helpers from one pinned package. It reduces emitted duplication but
adds package resolution, runtime ABI and mixed-version obligations. **Standalone
emission** places every reachable helper into each artifact. It is easy to carry
but increases output, gives agents many generated copies when tracing a bug, and
can let separately emitted artifacts disagree silently. A **per-world linked
bundle** selects helpers and intrinsic implementations for the complete admitted
world, emits one content-addressed runtime bundle, and binds every module and
host adapter to its manifest. It prices whole-world linking and incremental
cache invalidation.

Each option needs a way to detect incompatible components. Compare a separate
machine-readable manifest, embedded artifact metadata, and a reproducible build
lock connecting compiler version, intrinsic-registry version, helper semantic
hashes, target assumptions, wire schemas and host contracts. An agent asking
“where does decimal division live?”
should receive the authoritative source and every generated/importing consumer,
not a search through copied helpers. Runtime changes should report affected
semantic identities and required re-emission before code is edited.

Two subcases of **E15** expose different failures. **Dependency closure:**
build two modules separately, omit or skew one runtime package/helper,
then compose them; the gate must reject the build before execution and identify
the missing or incompatible component. **Runtime parity:** run exact numerics,
bounds faults, nested owned Bytes, outcome products and host receipts through
evaluator plus each packaging mode; require equal values, faults, ownership and
causal traces. Also mutate one helper while retaining its old version label and
ensure identity/conformance gates detect it. These are agent-cost hypotheses and
backend parity tasks, not evidence that any packaging arm is already cheaper or
correct.

## Agent-first editing, diagnostics, navigation, and introspection

Text, tree, and patch protocols can coexist. Text is the portable review
projection. Tree operations provide precise mutation. Semantic patches express
intent and let the compiler calculate dependent edits.

```text
PROPOSED semantic patch
change-function account__load@4 {
  add-error account.suspended(reason: Suspension__Reason)
  add-arm after account.closed
  require-test suspended_account
}
precondition interface_hash = "…"
```

The response should include the new typed tree, canonical text diff, changed
interface/executable/evidence identities, new proof obligations, invalidated
callers, and unapplied conflicts. Reviewers must be able to reproduce the text
from the tree and reparse it to the same semantics.

A formatter should expose two products: canonical source and a semantic diff.
The latter says “arm moved, meaning unchanged,” “argument evaluation order
changed,” or “error domain widened.” It must never infer semantic equivalence
from formatting alone. Stable formatting should include declaration/arm order
where order affects evaluation, trace identity, or diagnostic precedence.

Diagnostics should be structured objects with stable rule code, primary node,
related nodes, expected/found semantic values, proof/evidence category, and
machine-applicable repairs with preconditions. Offer an error graph so one root
cause can own cascades. For an incomplete join, return the uncovered Cartesian
space as data, not only prose. Repairs should distinguish “insert arm skeleton”
from “declare this impossible,” since the latter requires authority.

Navigation should query resolved identities rather than spelling: definition,
all uses by role, callers, effect paths, error producers/consumers, brand minting
and disclosure, resource ownership, revision dependencies, evidence rows, and
generated-code origin. A useful compiler protocol might include:

```text
describe symbol account__load
explain type node:83
explain authority call:19
enumerate outcomes join:reads
trace error account.suspended
impact replace interface account__load@4
project generated ts location node:83
```

Compiler introspection must label facts as parsed, resolved, checked, proved,
executed, scripted, linked, trusted, accepted, or runtime-only. “Verified” is
too coarse. Queries should return source/private/public/wire views separately.
A source function can be proved exhaustive while its public decoder, TS bundle,
or real adapter remains unvalidated.

## Trials that discriminate instead of decorate

No alternative earns adoption from shorter examples. Run the same agent tasks
against text-only, stable-tree, and semantic-patch interfaces:

- add a new error payload and repair every producer, arm, test, public codec and
  wire schema without granting a catch-all;
- reorder two named arguments that can both fault and predict whether semantics
  changed before execution;
- rename a branch in a join while preserving outcome identity, event identity
  and script addressing;
- migrate `Seq<Bytes>` across copy, transfer and borrowed ingress, then detect a
  retained alias;
- add a SQL commit-unknown path and prove no automatic rollback claim appears;
- recover from a half-written match and request only authoritative repairs;
- locate why a source-clean JSON build fails strict target typing;
- explain which exact acceptance identities change when a callback target or
  host contract revision changes.

Measure successful semantic completion, invalid intermediate states, repair
turns, unintended authority changes, diff size, stale-patch rejection, and
cross-layer omissions. Until those trials exist, every agent-benefit statement
above remains a hypothesis. B05 should continue to settle contracts against a
real customer before choosing private execution; its current A/B obligations
are alternatives and evidence prompts, not universal law.

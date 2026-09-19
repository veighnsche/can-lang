# Brainstorm: whole-language source forms

**Status:** divergent design inventory, not a specification, recommendation, or
implementation plan. Every snippet below is **PROPOSED pseudo-Can** and is not
expected to compile. The point is to expose alternatives that deserve trials,
including alternatives outside the current audit TODO. Human familiarity has no
independent weight here; an AI agent's ability to retrieve, generate, revise, and
verify the program is the relevant usability question. Brevity, uniformity, and
deliberate unpleasantness are likewise not goals by themselves.

The comparison baseline comes from the [shared model](../workstream-a/shared-model.md)
and [language-core dossier](../workstream-a/language-core.md): exact integers and
decimals, no silent numeric conversion, explicit finite failures, exhaustive
decisions, explicit authority, deterministic evaluation, and honestly scoped
evidence. Existing sources are pressure cases, not syntax precedents. In
particular, generic nesting in [map](../../../std/map/map.can#L21), callback-heavy
traversal in [seq](../../../std/seq/seq.can#L158), exact decimal comparison in
[scalars](../../../std/scalars/scalars.can#L161), and explicit machines over
nested data in [JSON](../../../std/json/json.can#L63) should remain expressible.
Removing grammar-forced carrier records is one hypothesis to test; explicit
carriers might help agents inspect boundaries even when a more compositional
alternative can omit them. Meaningful nominal wrappers require separate treatment.

## Whole-source representation families

The familiar option is **canonical indented text**: declarations and expressions
remain readable tokens, but the grammar can be rebuilt around one expression
algebra. It may retain keywords or use punctuation more systematically.

```can
// PROPOSED pseudo-Can: canonical expression text
fn ratio.add(left: Ratio, right: Ratio) -> Ratio
  fails [ZeroDivisor]
  do
    den := left.denominator * right.denominator
    result <- ratio.make(
      numerator: left.numerator * right.denominator
               + right.numerator * left.denominator,
      denominator: den) relay [ZeroDivisor => same]
    yield result
```

This preserves useful locality for token completion and diffs. Its risks are
indentation recovery, keyword-specific subgrammars, precedence mistakes, and the
temptation to add convenient second spellings. A formatter can make it canonical,
but only if parsing damaged edits and attaching comments/source identities are
also deterministic.

Delimiter policy is open too. Offside indentation, explicit `end` tokens,
parenthesized prefix nodes, balanced braces, or length-prefixed tree records have
different recovery properties. The current brace ban is not evidence that braces
are poor for agents; balanced delimiters may bound a damaged edit better, while
`end` tokens expose nesting in a token stream and indentation reduces closure
noise. One serious challenge to maximal explicitness is that redundant indent,
open token, close token, and node keyword can create more inconsistent partial
states than one authoritative boundary. A serious challenge to uniformity is a
hybrid: expression text for arithmetic, tables for decisions, and typed nodes for
authority-bearing calls. Its extra modes may be justified if each makes its own
obligation substantially easier to retrieve.

A second family is a **prefix grammar** in which every semantic node names its
kind and owns a delimited child list. Surface order then mirrors the typed tree.

```can
; PROPOSED pseudo-Can: prefix form
(fn ratio.add
  (params (left Ratio) (right Ratio))
  (returns Ratio)
  (fails ZeroDivisor)
  (body
    (bind den (mul (field left denominator) (field right denominator)))
    (dispatch (apply static ratio.make
      (args
        (numerator (add (mul (field left numerator) (field right denominator))
                        (mul (field right numerator) (field left denominator))))
        (denominator den)))
      (ok result (yield result))
      (error ZeroDivisor e (relay e))))))
```

Prefix form makes precedence, call kind, target position, and source order
literal. Tree edits and structural validation may be easier for agents. It can
also bury the operation in parentheses, make malformed partial output cascade,
and reward token copying without understanding binder scope. Names such as
`static`, `relay`, and `yield` still need real semantics; explicit node tags alone
do not prove authority or exhaustiveness.

A third family makes the authored artifact a **typed node tree**, serialized as
canonical JSON, CBOR, or a schema-specific text format. Text is a projection,
not the authority. Stable node IDs can attach tests, revisions, and evidence to
semantic nodes rather than lines.

```text
# PROPOSED pseudo-Can: typed node/tree projection
Function id=f-ratio-add name=ratio.add
  signature: (left: Ratio, right: Ratio) -> Outcome<Ratio,{ZeroDivisor}>
  body id=n1: Sequence
    n2 Bind den:int = Mul(Field(left,denominator), Field(right,denominator))
    n3 Dispatch
      target: StaticRef(symbol=ratio.make, revision=1)
      args(source-order):
        numerator: n4 Add(Mul(Field(left,numerator), Field(right,denominator)),
                          Mul(Field(right,numerator), Field(left,denominator)))
        denominator: Ref(n2)
      Ok(result:Ratio) -> Yield(result)
      ZeroDivisor(e:ZeroDivisor) -> Relay(e)
```

This can eliminate ambiguous syntax and preserve semantic identities through
rewrites. It raises harder product questions: whether agents can author stable
IDs without collisions, how small diffs work, whether schema versions become a
second language, and whether a text projection can round-trip comments and
unknown future nodes. Generated IDs must not become proof authority merely
because they are stable.

A fourth, more radical family is **relational/tabular source**. Declarations,
flows, arms, and tests are rows keyed by node ID; the program is a normalized
database with a canonical rendered view. Calls could list ordered argument rows,
and a decision could be an explicit coverage table. This favors retrieval and
bulk consistency checks, while ordinary expression trees become verbose joins.
Dangling references, accidental duplicate keys, and reorder-only churn become
its primary failure modes. It might work best for tests and matches without being
the whole language, so mixed representations must also be tested rather than
assuming universal uniformity.

**Authority, proof, and evaluation obligations.** Every family must account for
resolved symbol identity, revision, target kind, captures, effect/error authority,
the chosen evaluation order, branch identity, and the boundary between authored
assertion and checked fact. The canonical representation must reject unknown
semantic nodes rather than hash or lower placeholders.

**Discrimination task.** Encode `std__map__get`, one JSON render-machine step,
and a partially typed broken edit in each family. Measure tokens, repair success,
semantic diff stability, earliest diagnostic locality, and whether an agent can
recover call order, authority, and uncovered branches without consulting prose.

## Declaration grammar and whole-module organization

Expression regularity is only part of the language. Agents also add owners,
schemas, capabilities and external dependencies. Three broad alternatives are a
keyword-led declaration grammar; a uniform declaration node with named slots; and
a module manifest containing owned declaration records plus body/evidence cells.
These could share one semantic model while presenting different editing units.

| Declaration | Keyword-led candidate | Uniform/manifest candidate | Obligation that survives the spelling |
| --- | --- | --- | --- |
| Module | `module account owns ... exports ...` | Module record containing owner, definitions and explicit grants | Unique owner; moving a file cannot silently transfer authority |
| Import/export | `import model.User at revision 2 as User`; `export load` | Resolved dependency/export entries referencing declaration IDs | No accidental shadowing, floating owner or unauthorized export |
| Record | `record Pair(left:int, right:bool)` | `type Pair = product(fields=[left:int,right:bool])` | Choose nominal versus structural identity; field order and construction rules explicit |
| Variant | `variant Option<T> cases [Some(item:T), None()]` | Type node containing owned constructor declarations | Cases retain parent identity; extension changes exhaustiveness obligations |
| Error | `error account.denied(reason:Reason)` | Producer-owned error-variant declaration or capability failure schema | Failure kind and payload ownership; occurrence is separate from ordinary payload data |
| Brand/opaque type | `brand Token is str`; `opaque Token representation Bytes` | Type node with private representation and explicit mint/disclose grants | Shape validation does not mint the value; representation and wire disclosure are independent |
| Constant | `const limit:int = 100` | Value declaration marked compile-time, with checked initializer | Exact value identity; allowed initializer expressions and evaluation/proof limits |
| State | `state Account.used:int = 0` | Owned region declaration with initial state and capability members | Instance lifetime, fresh test stores and read/write authority |
| Host binding | `extern host.clock(...) -> Instant ...` | Dependency node tagged host with owned signature and observation contract | Host observation remains distinct from source imports and pure intrinsics, even with no errors |
| Function | `fn load(...) -> User` plus contract/evidence/body clauses | Function node with signature, authority, contracts, evidence and body slots | One checked meaning for signature, implementation and result binders |
| Revision | `rev 2` beside the declaration | Interface revision/digest in a lock manifest, referenced by owner and declaration ID | Explicit policy for interface/body/evidence drift; generation is not acceptance |

Compare a fixed field order in every declaration with freely ordered labelled
clauses normalized before checking. Fixed order reduces permutations; reorderable
clauses can make inserts local but complicate partial parsing and duplicate fields.
Another choice is whether generic parameters live next to a name or in a separate
`parameters` slot with explicit kinds. A single declaration grammar can simplify
tools or force agents to repeatedly name fields the keyword already determines.

For example, these are two proposed presentations of the same *declaration
header*, with body/evidence intentionally omitted:

```text
fn account.load(id: AccountId) -> User rev 2
  emits [account.missing]
  effects [Accounts.read]

declaration function account.load
  parameters [id: AccountId]
  result User
  failures [account.missing]
  authority [Accounts.read]
  interface_revision 2
```

Declaration lookup can require declaration-before-use, allow file-local forward
references, or resolve a whole owned module before checking bodies. Forward
lookup need not permit recursion: cycle and termination admission still runs on
the resolved graph. Test reordering declarations separately from reordering
evaluated expressions. File-order-dependent ownership is never a valid way to
resolve duplicates.

**Discrimination task (E19).** Create and then split a small module containing
each declaration kind; add one host dependency and one public type, rename a
constructor, and change a constant and interface revision independently. Compare
incomplete-declaration recovery, redundant metadata edits, owner resolution,
import churn, authority changes and which evidence must be revisited. This task
needs whole-module examples, not just translated function bodies.

## Calls, invocation heads, and argument placement

Call spelling and legal target position are separate axes. Candidate spellings
include visible `call f(...)`, bare `f(...)`, target-last `apply(args) to f`, a
pipeline `value |> f(extra: x)`, and a fully typed operation node such as
`invoke[static=f]`. Candidate heads range from static names only; static names and
bare `Fn` parameters; any field/local/result with callable type; or capability
handles that must be opened explicitly. A language could intentionally keep
constructors visually distinct (`new Pair(...)`) or make call/construct/project
the same application form after resolution.

```can
// PROPOSED pseudo-Can alternatives
call transform(input: x)
apply(input: x) to callbacks.mapper
x |> callbacks.mapper()
invoke[authority = callbacks.mapper](input: x)
```

For agents, target-first form exposes the dependency early; target-last form can
make dataflow edits local; a typed operation tag makes static versus indirect
authority explicit. Bare application saves tokens but may confuse constructors,
intrinsics, and callable data in an incomplete edit. Requiring every label aids
reordering and semantic retrieval, whereas positional arguments reduce repeated
names. Hybrid rules could require labels for booleans, repeated types, or three
or more parameters, but such heuristics create classification work and unstable
refactors. Another alternative is declaration-owned argument records:
`ratio.make({numerator: n, denominator: d})`, so application always has one
nominal input value.

All forms must specify exactly-once evaluation and an order independent of
parameter slot placement. That could be written source order, declaration order,
or an explicit argument-order vector. Source order makes local reading enough;
declaration order makes callee signatures authoritative; an explicit vector is
maximally inspectable but costly. The present mismatch behind F06 demonstrates
that labels cannot be allowed to silently decide fault priority.

Wider `Fn` heads require more than syntax. A field or returned closure must carry
origin, captures, latent errors/effects, preconditions, and revision identity.
Cycle analysis must follow values through records and returns or conservatively
reject uncertain graphs. Capability-bearing captures need a rule for transfer and
escape; a closure cannot be declared pure solely because its application syntax
looks ordinary.

**Discrimination task.** Express a static call, reordered faulting arguments, a
callback parameter, a callback stored in nested data, and a factory-returned
callback in each family. Ask an agent to identify evaluation order, pinned target,
captures, latent failures, and whether an indirect cycle exists; then compare
false acceptance and unnecessary rejection rates.

### Creating callable values, captures and arity

Calling and constructing callable values are separate questions. Keep these
alternatives open:

- **Named targets with explicit captures:** `fnref adjust(offset = step)` records
  a checked declaration identity and the values filling selected parameters.
  This is easy to attach to revisions, but requires a named helper even for one
  local transformation and can make residual parameter order surprising.
- **Anonymous functions with explicit captures:**
  `function captures [offset = step] (x:int) -> int { ... }` permits local bodies.
  It removes helper navigation but needs stable body/evidence identity, rules for
  contracts and tests, and analysis of escapes and indirect cycles.
- **Explicit partial application:** `bind adjust with [offset = step]` treats
  captures as a checked input record. Named holes can specify the remaining
  parameters; positional holes risk incorrect argument association.
- **Lexical capture with a compiler-materialized capture list:** local code is
  concise, but an edit may acquire authority or retain data unexpectedly. A
  candidate must expose and check the changed capture list before acceptance;
  it cannot equate a generated list with permission to capture those values.
- **Checked symbol handles without anonymous closures:** pair an admitted target
  handle with owned data and a declared dispatch operation. This makes identity
  explicit while moving expressivity and boilerplate into the handle protocol.

Arity can remain unary, using one nominal argument record; become an ordered
parameter tuple; or be a labelled parameter row. Unary records make composition
uniform but introduce construction/projection and nominal wrappers. Multiple
arguments match ordinary declarations but expand partial-application, variance
and type-identity rules. Choose neither arity nor capture policy merely because
application has one spelling.

Capture initializers also need evaluation order, exactly-once behavior and an
ownership policy. Compare current data-only captures with richer checked captures;
transferring a capability into an escaping function is an authority/lifetime
decision. No spelling authorizes raw host callbacks. Explicit anonymous bodies
still need admissible termination and preconditions.

**Discrimination task (E09 extension).** Create the same two-input transformation
using every construction family, capture one parameter and invoke the other,
then rename parameters, change a captured value, return the function in a record,
and attempt a capability capture and self-reference. Check argument association,
observable capture order, callable identity changes, escape admission and cycles.

## Successes, failures, and error composition

There are several independent choices. A function may always produce
`Outcome<T,E>`; only fallible functions may do so while total functions produce
raw `T`; or every function may cross an implicit result boundary even when
`E = Never`. Success construction may be `Ok(value)`, `success value`, a dedicated
`yield`, or a declaration-controlled record-shaped form. `Unit` can be a real
one-value type (`Unit`, `()`), an empty nominal record, or absence represented by
an outcome tag with no payload. These differ when Unit is stored in nested data,
used as a generic argument, or distinguished from a domain-specific empty record.

```can
// PROPOSED pseudo-Can: a nested completed outcome is successful data
fn remember(attempt: Outcome<User, {Denied}>)
  -> Outcome<Record<Outcome<User,{Denied}>>, {StorageFault}>
  yield Ok(Record(attempt: attempt))
```

First-class completed outcomes enable queues, retries, and generic combinators,
but create two levels: `Ok(Err(Denied))` differs from the function emitting
`Denied`. The grammar must never flatten that boundary by convenience. An
alternative keeps outcomes non-storable and adds dedicated `map`, `zip`, and
`relay` operations. This reduces nesting but forces every composition shape into
language or stdlib primitives.

Failure declarations can remain concrete sets, become generic rows
`fails E`, use algebraic variants owned by the consumer, or attach errors to
capabilities rather than function signatures. Rows need kinds, closed/open status,
union and subtraction rules, duplicate identity, payload preservation, and a rule
for generalization. `E1 | E2` says either error may occur; it cannot record that
two parallel validations both failed. Simultaneous failures need a product or
collection such as `Many<ValidationError>` or
`Both<Outcome<A,E1>,Outcome<B,E2>>`, with deterministic ordering and duplicate
semantics. Conflating this with row union would lose information.

Handling forms include explicit exhaustive `match`, typed continuation arguments,
railway combinators, a postfix relay marker, and declaration-visible propagation:

```can
// PROPOSED pseudo-Can: propagation is listed at the binding site
user <- lookup(id) relay [Missing => account.not_found(id), Offline => same]
yield user
```

The failure mode is hidden discard or accidental widening: a catch-all, inferred
row, or `_ => relay` can let a newly declared error bypass review. Mapping errors
must be total and payload-aware. Runtime faults such as division by zero remain
distinct from declared domain failures unless deliberately converted by a checked
operation.

**Discrimination task.** Implement total `length`, fallible `get`, generic
`map`, recovery, sequential composition, parallel validation returning two
errors, `Unit`, and `Outcome` nested twice. Mutate a callee to add an error and
check which callers become invalid, which evidence is stale, and whether any
candidate silently flattens or discards a payload.

## Binding and sequencing

Alternatives include nested dispatch only; immutable `let` blocks with a final
expression; monadic `do` notation; dataflow graphs where bindings declare
dependencies and a separate order; continuation clauses; and explicit state
machines like JSON already writes by hand. Blocks need not imply early return.
They may permit irrefutable binding only for total computations and require a
visible dispatch node for fallible ones.

```can
// PROPOSED pseudo-Can: ordered immutable block
do
  left  := exact arithmetic expression
  right <- call checked_lookup(key) dispatch
    Missing(e) => yield fallback(e)
    Ok(v) => continue v
  yield Pair(left: left, right: right)
```

A graph representation might permit independent nodes to evaluate in a declared
canonical order or in parallel. If parallelism is admitted, first-fault behavior
must be replaced by a complete settlement rule; it cannot leak backend scheduling.
Pure-looking expressions can still fault, so reordering arithmetic, indexing, or
arguments changes observable behavior. Tail transfer should have its own form if
the whole outcome is relayed, preserving payload and evidence rather than acting
as hidden return.

**Discrimination task.** Translate the recursive sequence workers and a JSON
frame transition into every sequencing form. Inject two primitive faults, two
declared errors, and a state effect. Compare evaluator/target traces, proof-arm
identity, diff size for inserting a step, and whether dependencies permit an
agent to infer order without assuming purity.

## Patterns and decisions

One family unifies value, variant, record, and outcome matching under constructor
patterns. Another deliberately keeps outcome dispatch separate so authority and
error coverage remain visually privileged. A third uses decision tables with one
column per scrutinee and one explicit result column. A fourth replaces patterns
with typed predicates plus bindings, making overlap a solver obligation.

```can
// PROPOSED pseudo-Can: constructor patterns
match result
  Ok(User(id: id, roles: [first, ..rest])) => ...
  Err(auth.Denied(user: id)) => ...

// PROPOSED pseudo-Can: table projection
decision result
  | tag    | payload pattern              | action |
  | Ok     | User(id: id, roles: roles)   | ...    |
  | Denied | auth.Denied(user: id)        | ...    |
```

Nested patterns reduce projection noise but can duplicate expensive or
faulting observations if elaboration is careless. Or-pattern binders need equal
binding sets and types. Range, string, boolean, and sequence-rest patterns need
decidable coverage rules; otherwise exhaustiveness becomes an unverified claim.
Parent variant identity must qualify a case even when short names are rendered.
First-arm priority is simple operationally but allows shadowing; disjoint tables
avoid priority at the cost of rejecting useful ordered refinements.

**Discrimination task.** Encode scalar comparison's multi-scrutinee table, JSON's
nested variants, an outcome with a new error added, repeated case names from two
variants, and nested sequence/record destructuring. Test exhaustive coverage,
overlap explanations, exactly-once scrutinee evaluation, and stable source-arm
IDs after formatter rewrites.

## Types, generics, rows, and contextual information

Type applications could use `Map<K,V>`, `Map[K,V]`, prefix `(Map K V)`, or fully
named arguments `Map<Key = K, Value = V>`. Function types could stay positional,
use a signature record, or expose all latent authority:
`Fn<(item:T), R, errors:E, effects:F, requires:P>`. Generic declarations could
be angle parameters, explicit `forall`, or separate type-level input records.
Constraints may be nominal interfaces, structural operation sets, explicit
dictionaries, or per-specialization checking as today. Each choice changes what
counts as proof of a generic law: several successful stamps never establish the
universal case.

Context can be absent everywhere; admitted only inside constructors and patterns;
allowed wherever one expected type yields a unique elaboration; or replaced by
explicit typed holes that a checked elaboration operation fills into canonical
source. The last option gives agents convenience without storing inferred syntax:

```can
// PROPOSED pseudo-Can edit state; an explicit elaboration edit resolves the hole
items: Seq<Map<str,int>> = Seq<Map<str,int>>[hole<Map<str,int>>]
```

The context query can propose a uniquely typed construction or request missing
information. A hole has no executable value. The formatter only formats the
resolved edit; it does not invent a value or silently choose a type.

Empty collections, polymorphic function references, overloaded numeric syntax,
and open error rows are the ambiguity tests. Inference must never synthesize
public authority, contracts, error/effect rows, or universal claims from examples.
Nominal identity must not collapse to printed names. A radical alternative is to
store explicit types in the typed-tree authority while rendering omissions in a
view; then reviews must show the hidden elaboration diff whenever context changes.

**Discrimination task.** Type nested `Map<str,Seq<Option<int>>>`, an empty nested
collection, a polymorphic callback, a generic error-preserving compose, and two
same-named nominal types. Change only the expected outer type and record whether
the inner program silently changes, whether generated canonical source changes,
and which generic assertions are checked versus merely tested.

## Construction, data, collections, and primitive expressions

Records may be nominal constructors, structural literals, declaration-owned
argument records, or typed maps from field IDs. Field syntax can be positional,
required labels, or labels omitted only when names are punning (`Pair(:left,
:right)`). Variants may use qualified constructors `Json.Value.Num(text: "12")`,
short constructors resolved from expected type, or a uniform `case` node. Brands
need a visibly authoritative mint operation; making them look like ordinary
records would obscure provenance.

Nominality itself has alternatives. Records can be nominal by declaration,
structural by field set, or structural internally but wrapped by explicit opaque
boundaries. Equality might mean structural value equality, nominal identity plus
field equality, or only a type-authorized operation. Opaque brands may allow no
equality, same-brand equality, or an owner-supplied comparator. Ordering can be a
built-in closed set as today, a declared constraint, or an explicit comparator
value. A generic `compare<T>` must not acquire a universal law because three
stamps happened to work; dictionary or constraint authority must say which
operation and laws are available. Recursive data can be admitted directly
(`Json = Null | Arr(Seq<Json>)`), through indirection/reference nodes, or through
explicit folds. Direct recursion makes the domain model clear but requires
positivity, termination, traversal, serialization, and size rules. Rejecting it
forces wrappers and frame encodings like the current JSON customer. Those may be
useful execution machinery or useful explicit source structure. Compare their
agent editing and proof costs with direct recursive data before deciding removal.

Primitive partiality can also surface in several ways. Indexing and division may
remain loud runtime faults, return declared outcomes, require a proof-bearing
index/nonzero divisor, or split into checked and proven operators. A postfix
`xs[i]?` is compact but can hide which error it introduces; `checked_index(xs,i)`
is explicit but may drown algorithms in dispatch. Proof-only operations risk
turning solver incompleteness into expressivity loss. Whatever spelling wins,
unchecked failure, domain error, and host/resource fault must not collapse into
one catch-all channel.

Collection type spellings include `Seq<T>`, `[T]`, `T[]`, `(Seq T)`, and
`sequence of T`. Literal choices include `Seq<int>[1,2]`, `int[1,2]`,
`[1,2]:Seq<int>`, or `seq(type:int, items:[1,2])`. Empty literals expose whether
context is doing hidden work. Maps/sets could remain ordinary nominal data, gain
dedicated literals, or use comprehensions whose evaluation order and duplicate
policy are explicit. A builder form might solve repeated append cost while
remaining source-immutable, but alias and settlement semantics need proof.

Numeric spelling must not change numeric meaning silently. Plausible families
are current tagged exact decimals `d"1.50"`; ordinary `1.50` defined as exact
decimal; suffixes `1.50dec`; or a uniform constructor `dec("1.50")`. Introducing
binary floats is a semantic feature, not punctuation cleanup. Any family must
preserve arbitrary-precision integers, decimal canonicalization, signed-zero
rules, exact comparison, Euclidean integer division, and loud unsupported/zero
division. A type-directed bare literal is risky if moving it from `dec` to a
future float context changes value without changing source.

Strings can remain raw-by-default plus `e"..."`, become escaped-by-default plus
`raw"..."`, use length-delimited blocks, or use a structured scalar sequence.
Canonical source must distinguish value equality from spelling identity and
must not normalize Unicode or invisible control bytes unless the operation is
explicit. JSON's string-heavy source is the edit trial; schema's literal control
bytes are the adversarial one.

Boolean forms include words, symbols, prefix nodes, or a decision construct only.
Semantics are the real axis: eager `and/or`, short-circuit control flow, or two
explicit families such as `all_strict` and `and_then`. Changing `and/or` to
short-circuit silently would change fault order and evidence reachability. Two
families increase choice and near-duplicates; strict-only makes guards verbose;
short-circuit-only requires branches and skipped obligations to be part of the
proof model. `false and (1 / 0 == 0)` and `true or (1 / 0 == 0)` must be explicit
conformance cases for every proposal.

**Discrimination task.** Rewrite representative map, JSON, ratio, and schema
fixtures using every construction/literal family. Include empty and nested
collections, duplicated map keys, exact `0.10 + 0.20`, huge integers, signed
decimal zero, escapes, raw control bytes, both boolean fault probes, checked
and proof-bearing indexing, nominal/structural equality, recursive JSON, an
opaque brand comparator, and two labels deliberately reordered. Compare
semantic-token count, edit errors, canonical diff, exact evaluator result, and
evidence reachability.

## Abstraction, derivation and language extension

How much reusable structure should an agent be able to define? Compare a closed
core of functions, generics and ordinary combinators with mechanisms that generate
or abstract declarations and decisions. This question applies to building a
library from intent, before any defect or regression exists.

| Mechanism | Agent benefit hypothesis | Cost and failure case |
| --- | --- | --- |
| Ordinary functions, generics and data-driven combinators only | One vocabulary for code generation, navigation and checking | Related schema adapters or repetitive decision structures may need many declarations and synchronized edits |
| Compiler-owned derivation, such as equality or admitted codecs | A checked request replaces repetitive boilerplate; generated operations share known semantics | Supported derivations can be too restrictive, or hide important field/ownership policies behind a single keyword |
| Typed declaration templates with explicit parameters | Agents define a family of functions or schemas once and instantiate it predictably | Generated names, diagnostics, expansion size and per-instance obligations become part of the language model |
| Reusable pattern or decision abstractions | Repeated domain decisions have one explicit definition | Hidden case priority, variable binding and coverage can make a short use site harder to understand than an expanded match |
| Hygienic syntax extensions or staged computation | Agents can build domain forms and specialize repeated structure | Scope, phase ordering, termination, expansion inspection and dialect proliferation introduce substantial new obligations |

The last alternatives explicitly reopen the current no-macros policy for
discussion; they are not current features or automatically desirable extensions.
A closed grammar remains a serious candidate. Staging also needs to distinguish
pure compile-time computation from host observation; generation cannot silently
gain I/O authority. A derived codec cannot grant brand minting merely because
the generator can see the representation.

For every generating mechanism, compare inline expansion, a queryable generated
view and checked semantic templates. Ask whether an agent can explain a use site,
locate the source of a diagnostic, and modify one instance without accidentally
changing every instance. Keep behavior, scope, authority and evidence attributable
through expansion; constrain compile-time cost explicitly.

**Discrimination task (E20 extension).** Starting from a new multi-module program,
extract three repeated validation/serialization operations into an abstraction.
Then add one exception and one new data field. Compare ordinary functions with
derivation/templates, and only then any syntax extension that solves a demonstrated
gap. Measure correct completion, boilerplate, context needed and expansion-related
mistakes; short output alone does not establish a benefit.

## Lexical and grammar rules beyond literal spelling

| Question | Alternatives | Failure or cost to test |
| --- | --- | --- |
| Identifier identity | Restricted ASCII identifiers; Unicode identifiers with an explicit equality policy; opaque symbol IDs with display names | Confusable spellings, case-folding drift, tool-unfriendly IDs and accidental owner changes. Display text must not acquire minting authority. |
| Naming conventions | Compiler-enforced kind/domain prefixes; scope-qualified names with kind supplied by the declaration; unrestricted local names with resolved-role queries | Redundant renames, local-name ambiguity, same-name type/value collisions and guesses from capitalization. |
| Operators and precedence | Small fixed precedence table; parentheses required around mixed operators; all operators prefix; explicit typed operation nodes | Misgrouped exact arithmetic, unary-minus versus negative-literal ambiguity, and shifts in diagnostic location after inserting an operator. No alternative implies user-defined overloading. |
| Delimiters and separators | Offside rules; explicit end tokens; balanced delimiters; one-node-per-record schema | Newline-sensitive edits, missing terminators, generic-closer ambiguity, long literals and nested comments. Partial edit recovery must not become permissive executable parsing. |
| Comments and metadata | Lossless trivia; dedicated documentation nodes; separately attached annotations | Comment attachment after moves, braces or quotes inside comments, and accidentally interpreting prose as semantic directives. Whichever policy is chosen must distinguish data, comments and executable tokens. |
| Grammar extensibility | Closed versioned grammar; compiler-owned tagged node schema; declared versioned extension categories with complete checking rules | Unknown-node acceptance, formatter/parser disagreement and agent dependence on local dialects. Open-ended macros remain a separate semantic expansion, not a free consequence of a tagged syntax. |

**Discrimination task.** Rename two same-spelled items in different roles; add a
negative numeric expression inside nested generic construction; truncate and then
repair one block; move a documented arm; paste JSON/CSS inside strings and comments.
Compare bounded diagnostic cascades, exact literal preservation, resolved identity,
and parse/format/parse equivalence. This also tests whether one canonical source
form should reject alternate spellings or accept them only as explicit editor
input that is normalized before the executable artifact exists.

## Cross-axis trials

No axis should be selected from isolated pretty snippets. Build three small
equivalent corpora for each coherent combination: a scalar/collection kernel, a
nested JSON-like machine, and an authority-bearing callback pipeline. For each,
record parse recovery under truncation, type/authority errors introduced per
edit, repair attempts, semantic diff precision, evaluator/emitter parity, and
the questions an agent answers correctly from source alone. Include hostile
mutations: add an error, reorder labeled arguments, return a closure through a
record, create an indirect cycle, nest `Outcome`, introduce simultaneous errors,
change expected context, and replace eager boolean evaluation with a branch.

The comparison artifact should preserve every candidate's strongest form rather
than forcing a shared syntax template. It should also separate source usability
from proof power: a tree can be easy to validate yet hard to author; a terse
text can be easy to emit yet conceal authority; a table can make coverage obvious
while making arithmetic opaque. The result of this brainstorm is the space and
the experiments, not a preferred language form.

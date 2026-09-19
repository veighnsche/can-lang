# Authority and evidence: divergent design space

Status: brainstorm, not a selected design or specification. This part asks how
agents organize programs, express authority, specify intended behavior and obtain
useful feedback. Its baseline is the
[authority and evidence dossier](../workstream-a/authority-evidence.md).

The repository has zero external users, so no option earns points merely for
preserving existing spelling, ABI, generated TypeScript layout, or goldens. The
reader is an AI coding agent. Useful qualities are therefore explicit ownership,
few context-sensitive rules, mechanically checkable obligations, precise failure
locations, and edits whose semantic blast radius can be computed before compiling.

## Ground facts and separations

A language can represent the following concerns in different ways. Keep their
meanings distinct while exploring modules, contracts and evidence:

| Identity or authority | Question answered | Inputs that may matter |
|---|---|---|
| Interface | What does this declaration promise to callers? | owner, parameters, nominal types, errors, effects, contracts, dependency shapes |
| Executable | What computation will run? | resolved body, calls, callback targets/captures, specialization, termination annotations |
| Dependency | Which exact declarations does this artifact rely on? | resolved owner, declaration identity, revision, transitive interface closure |
| Acceptance | What reviewed expectation was approved? | row identity, canonical inputs/outcome, referenced declaration closure, accepted manifest |

These identities can share canonical infrastructure without becoming one hash. A
body edit can invalidate proof freshness and linked evidence while leaving both the
interface and a pinned expected value unchanged. Conversely, changing a callback
target in an expected value changes acceptance evidence even if both callbacks have
the same callable type. The current repair's details are in the
[supporting notes](support/implementation-notes.md); none of its serialization or
compiler structures is a requirement for the alternatives here.

That separation prevents two tempting mistakes. A test run is evidence for the
executed cases, not a universal proof. A cryptographic digest establishes content
equality relative to a canonicalizer; it does not identify a reviewer, grant a
capability, make a baseline accepted, or prove the canonicalizer complete.

## Declaration and module identity alternatives

### Fixed authored identities

Every module and public declaration could own an immutable, source-authored ID,
separate from display name and file path. Imports bind that ID and may assign a
local readable name. Renaming or moving is identity-inert; copying a declaration
without allocating a new ID is a duplicate-identity error. Private declarations
could use owner-local IDs allocated by the formatter or compiler.

This makes refactors and diagnostics predictable for agents and gives host grants,
brand minting, effects, and accepted rows a durable key. It also creates key
management work: clone-versus-move must be explicit, merge conflicts can duplicate
IDs, and humans or agents may cargo-cult identifiers. Audit must prove uniqueness,
owner binding, and that a source package cannot claim an already accepted ID.
Revocation and deletion need tombstones if resurrection matters.

### Semantic identities

Identity could be derived from declared meaning: owner identity, kind, local name,
type/effect/error/contract interface, and nominal dependency identities. A rename
or contract change then creates a new identity automatically; a body-only edit does
not. Imports pin semantic identities or a readable selector whose resolution is
recorded in a manifest.

This removes manually allocated IDs and makes identity drift self-describing. It
also turns changes that may be editorial, such as a parameter rename, into policy
questions about semantic equality. Recursive types require a graph canonicalizer,
not naive recursive hashes. Two independently authored declarations with identical
shapes must remain distinct when nominal ownership matters, so owner identity cannot
be erased. Audit burden moves heavily onto canonicalization completeness and cycle
handling.

### Content-addressed artifacts

A module artifact could be addressed by a digest of its resolved interface, or of
its complete executable bundle. Interface digests are useful dependency locks;
executable digests are useful build/cache and provenance keys. A manifest maps
source-level imports to exact digests. Mixed artifacts become easy to reject.

This gives excellent reproducibility and detects any canonicalized change, but it
does not itself decide which digest is authorized. An attacker can hash a malicious
artifact as easily as a reviewer can hash a good one. Signatures or a reviewed
accepted manifest still supply authority. Whole-executable addressing also makes
private body churn cascade through dependents unless interface and executable
addresses are distinct. Debug metadata, formatter changes, and backend versions
need an explicit inclusion policy.

### Layered identity

A fourth family combines a fixed nominal owner/declaration ID, a semantic interface
digest, and an executable artifact digest. Imports name the nominal ID plus an
accepted interface digest; builds record the executable digest actually linked.
This has the richest audit trail and the greatest machinery. Agents must understand
which component to bump or reaccept, and diagnostics must say exactly which layer
changed. It becomes harmful if every command exposes three opaque hashes instead of
human-readable structural diffs.

Across all four, visibility is independent. Options include public/private on the
owned declaration, explicit export blocks, capability-gated imports, or a sealed
module whose public interface is a separate declaration. Tests must cover both
source orders, same display names under distinct owners, ambiguous aliases, private
access, exact error/brand ownership, host binding by declaring owner, and duplicate
ID injection. F02 shows why load order and short-name registries cannot be authority.

### Namespace scope is a separate choice

The identity families above do not decide lookup. Keep at least these candidates:
strict globally unique qualified names with explicit imports; module-local names
with qualified references and optional explicit aliases; package-owned namespaces
with internal module visibility; or owner-ID lookup with names as local projections.
The current global model remains a legitimate candidate once duplicates are
rejected. Scoped lookup could reduce long prefixes or introduce shadowing and
more context retrieval. Packages could express a real authority boundary or add
an unnecessary intermediate owner.

Try the same record and error short names under two owners, move a private helper,
and split a module. Ask whether an agent can identify the provider using the local
source and compiler context, and whether brand minting and accepted expectations
retain the intended owner. A namespace choice does not itself require historical
multi-version linking or compatibility adapters.

### Authored authority versus generated metadata

`provides`, `uses` and module `emits` need their own brainstorm; their current
spelling does not prove that all three should be authored. Alternatives include:

- Author every declaration and summary, require exact checked agreement, and use
  redundant facts as explicit intent. This aids inspection but multiplies edits.
- Author ownership, public exports and dependency grants; generate used-symbol,
  error and effect summaries from checked bodies. This reduces drift but makes
  semantic diffs and queries necessary for local review.
- Author a module interface separately from implementation; derive private details
  and check implementations against the interface. This isolates public intent
  while creating another artifact that can fall out of date.

Declared limits and derived usage answer different questions. Inferring an actual
clock dependency does not grant permission to observe the clock. Removing a stale
metadata entry must not revoke or expand authority silently. Current function
`emits` is enforced; the frozen audit found module `emits` unenforced. Try adding
an unexpected host observation, deleting an export, and leaving a stale error
summary in each candidate. Measure missed drift, redundant edits and whether
diagnostics distinguish a derived fact from an authored grant.

## Four ways to author evidence

These are source-organization alternatives, not proposed grammar. The fragments
are intentionally schematic.

### Inline decision tables with explicit row modes

Keep evidence beside each function, but make each row state its execution mode and
authority intent:

```can
tests
  scripted denied(...) => payment.denied(...) using [...]
  linked roundtrip(...) => Ok(...)
  pure boundary(...) => Ok(...)
  accepted regression(...) => Ok(...) pinned
```

`scripted` consumes explicit exchanges at declared boundary sites. `linked`
executes a selected real dependency closure. `pure` asks the compiler to prove the
reachable graph has no observations and then executes real bodies. These modes may
collapse depending on the eventual default, but they should not be inferred from
file location. Inline rows are easy for agents to update with a body and keep the
current decision-table character. They can make large declarations noisy and tempt
an agent to change implementation and expected values together. A `pinned` token is
still only a request to compare with independently accepted authority.

### Separate owned evidence declarations

Evidence can be a first-class declaration owned independently of the function:

```can
evidence json_integer_contract rev 3
  targets [json.decode_int@1]
  mode linked
  vectors [...]
  accepts interface [...]
```

This separates implementer edits from reviewer-owned evidence, allows one suite to
cover several roots, and gives evidence its own visibility, revision, and accepted
manifest entry. It is well suited to generated adversarial corpora. The cost is
navigation and stale suites: the function no longer shows its complete examples.
The checker must prevent an implementation package from silently shadowing the
review-owned declaration and must define whether evidence revisions track vector
content, target identities, mode, or all three.

### Typed evidence graph

Represent evidence as a graph of typed facts and derivations: a vector executes an
arm; an admitted proof discharges a contract obligation; a linked run observes an
outcome and effect trace; an authority node accepts a precise expectation. Source
can display this graph through declarations, while tooling renders “why accepted?”
and “what invalidated?” paths.

This matches the real dependency structure and supports minimal reruns. It can say
that an interface change invalidated caller proofs, while a provider body change
invalidated linked vectors but not caller type checking. It is also the most complex
option: graph node kinds become a meta-language, accidental cycles need semantics,
and stored derivations start resembling a proof cache. If certificates are stored,
their checker, assumptions, solver/version, and dependency closure need independent
validation. The current compiler's reproof-on-compile model is simpler and must not
be described as cached proof reuse.

### Explicit dependency contracts

An importer can declare what it assumes from each dependency and attach evidence to
that edge rather than to either endpoint:

```can
dependency scalars.convert_int@1
  assumes interface [...]
  permits effects []
  tests scripted [...]
  verifies postconditions [...]
```

This makes foreign-call mocking honest: scripts assert an edge contract, while a
linked suite checks that the provider implements it. Agents can inspect one import
to learn exactly what a consumer relies on. Duplication is the main cost; many
consumers may restate the same contract, and a stronger provider contract does not
automatically prove each consumer's differently phrased assumption. The language
would need explicit implication rules or demand exact referenced contracts.

## Effects, capabilities, and observations

One family keeps current function effect rows but expands them beyond state cells:
`clock.observe`, `random.consume`, `secret.compare`, `log.disclose`, resource
acquisition, and mutation rights become typed capabilities. Callable types carry
latent effects, and imports may attenuate a capability. This makes transitive
admission direct but can produce large rows.

Another family passes capability values explicitly. A world or narrow handle is an
ordinary linear/affine parameter, so authority flow follows data flow and capture
rules. This works naturally for resource ownership but risks turning authority into
forgeable data unless construction is module-owned and opaque. A single `World`
handle is easy to thread and poor for audit because it hides which power is used.

A third family puts authority on module/dependency edges: a module may call a host
adapter only through a manifest grant. Function effects are inferred summaries
checked against that grant. This reduces annotations but makes local review depend
on tooling and can obscure higher-order capture. A hybrid can use explicit grants
for acquisition and effect rows or handles for propagation.

State itself has several plausible models. Module-owned cells preserve today's
simple fresh-store tests but make instance ownership and concurrent use implicit.
An explicit `World` value makes instances visible, yet a monolithic world hides
authority. Owned state regions can provide narrow read/write/transfer capabilities
and reject aliasing or use after retirement, at the price of affine or linear type
machinery. Protocol-state resources go further: acquisition returns a handle in an
`open` state, operations transition it, and release consumes it. This expresses
cleanup directly but can make ordinary counters unnecessarily ceremonial. A useful
trial should compare two independent quota instances, a read-only borrowed view,
and a handle captured by a callback that outlives its owner.

Higher-order code also raises effect polymorphism. A generic mapper might promise
exactly the latent effects of its callback rather than be permanently pure or list
every possible observation. Alternatives include explicit effect variables and
bounds, inferred effect parameters printed into canonical interfaces, or separate
pure/effectful combinators. Explicit variables expose the contract to agents but
add kinds and substitution rules. Inference shortens source while making interface
changes less obvious. Split combinators avoid polymorphism and multiply APIs.
Every option must preserve callback capture authority, instantiate effects in
dependency identity, and prevent an apparently pure function value from hiding a
host observation.

Whichever family is explored, errors stay separate. `emits []` cannot establish
purity for `host__wall_now`; the real current example is
[`std/host/host.can`](../../../std/host/host.can). Tests should include an
error-free observation, an unused impure branch, a callback capturing authority,
capability attenuation, declaring-owner host binding, and a scripted host call that
must never execute during compilation.

## Contracts, arm evidence, acceptance, and proof

Contract authoring has its own alternatives, independent of where proof results
are stored. The current inline family keeps `requires` and outcome-indexed
`ensures` on each function. All snippets in this section are schematic proposals;
the current verifier does not support arbitrary state/trace predicates or the
proposed named-contract forms. The inline presentation could look like:

```can
fn account__withdraw(account: Account, amount: int) -> Account rev 1
  emits [account.insufficient]
  requires amount >= 0
  ensures
    on Ok next => next.balance == account.balance - amount
    on account.insufficient _ => amount > account.balance
```

This gives an agent the promise beside the implementation and permits different
postconditions for success and each declared error. It duplicates common laws,
however, and contract edits become interface edits at every function. If state or
effects are in scope, the binders must say whether `account` is the old value,
whether `next` is returned data or new store state, and whether an error permits a
state change. Implicit snapshots are concise but easy to misread.

A second family gives contracts an owner and name, then lets declarations reference
or instantiate them:

```can
contract accounting.Withdrawal(account: Account, amount: int)
  pre amount >= 0
  post Ok(next) => next.balance == account.balance - amount
  post account.insufficient(_) => amount > account.balance

fn account__withdraw(...) -> Account rev 1
  emits [account.insufficient]
  satisfies accounting.Withdrawal(account, amount)
```

Named contracts support reuse, separate review ownership, and exact dependency
identity. They add navigation and parameter mapping; a reference must not silently
mean “similar enough.” Changing the named contract must invalidate every proof that
assumed it, while an implementation body change need only trigger fresh proof.

A third family authors an explicit pre/post relation over a transition record:

```can
predicate valid_withdraw(before: AccountState, input: Request,
                         outcome: WithdrawalOutcome,
                         after: AccountState, trace: EffectTrace) -> bool
```

The function promises that relation for every completion. Success and error are a
single typed outcome sum, and old/new state plus the effect trace are visible data
to the specification. This can express “error leaves balance unchanged” and “one
ledger write occurred,” and it composes naturally with state machines. It is more
verbose, may expose an overly concrete trace order, and risks turning specification
predicates into unrestricted executable helpers. An admitted pure predicate subset
or a separate specification language would need an explicit boundary.

All three need honest proof status per clause: verified in the supported solver
fragment, rejected, or inconclusive. Passing examples may refute a contract or
illustrate it; no quantity of examples makes the quantified contract universal.
Error binders, pre-state, post-state, and trace visibility must be consistent at
call sites and through named contract references.

Arm obligations could remain source-arm execution witnesses, with narrowly named
certificates for structural relays. This gives concrete examples but scales poorly
for large partitions. An alternative lets the verifier prove an arm reachable and
prove its result contract without executing a vector; a third allows proof of
unreachability to remove the witness obligation. Each fact must remain distinct:
reachable, executed, universally correct under path assumptions, and unreachable
are different claims. Optimization branches and lowered branches do not create or
erase source obligations.

Pinned acceptance has several authority models. A repository policy may turn
current advisory warnings into gate failures. A separate signed review manifest
may name reviewer identity and scope. A threshold manifest may require independent
acceptors for capability or public-interface changes. Or Can may deliberately keep
an unauthenticated local accepted file and promise only drift detection. In every
case candidate generation stays unaccepted, changed rows are compared with the old
authority, and structural diffs accompany hashes. Changing a factory body and its
expected callback together must not self-approve; changing only a callback's body
must not automatically claim that the expected callback identity changed.

Counterexamples for every model include a false contract with green examples, a
callee summary that failed verification, an unsupported clause partially ignored,
a row moved onto a different source arm, a callback with the same type but different
authority, and a candidate manifest supplied alongside malicious source. The active
proof fragment described in the frozen dossier is useful evidence, but passing
rows, strict TypeScript, linked execution, and solver proof each establish different
claims.

## Termination and resource obligation forms

The current `decreases` forms establish admitted mathematical descent; they do not
bound time, allocation, host calls, or target stack. Several authoring directions
deserve separate trials:

- Keep declaration-level `decreases`, add structured measures over data size and
  lexicographic tuples, and let each recursive call expose the decreasing argument.
- Attach an obligation to each recursive edge, useful when different arms decrease
  different measures or mutual recursion is later admitted.
- Replace common recursion with finite immutable traversals whose fixed input bounds
  justify termination. A growing or revisiting worklist still needs a decreasing
  measure, a finite-domain no-reinsertion invariant, or an explicit termination proof; ownership
  alone establishes none of these.
- Add explicit resource contracts such as maximum host observations, allocation
  growth, depth, or fuel consumption. These are cost claims, not termination proofs.
- Treat fuel as semantic input, returning a declared exhaustion outcome; or treat it
  as an implementation guard/fault. Mixing those interpretations makes tests lie.

Trials need a nondecreasing recursive counterexample, a mathematically terminating
function that overflows the emitted target stack, JSON's frame/fuel traversal, a
linear function with excessive allocation, and a resource-owning computation whose
cleanup obligation survives early failure. Async settlement and resource retirement
must say when capabilities cease to be usable; timeout alone is not cleanup proof.

## Discriminating agent tasks

| Task | What it distinguishes | Required observation |
|---|---|---|
| Move a pure helper to another module without changing behavior | location-based scripts versus semantic modes | exact edits, diagnostics, rows invalidated, linked result |
| Rename and split a public module while retaining one nominal type | fixed, semantic, content, and layered IDs | import churn, identity diff, host/brand authority preservation |
| Change a callback target, capture, type argument, then only its body | acceptance/dependency/executable separation | which warnings, proofs, and vectors rerun for each mutation |
| Add an error-free clock read behind an unexecuted branch | effect admission completeness | pure/linked refusal before execution |
| Introduce two modules with the same local `Record` name | namespace and owner rules | deterministic resolution in both file orders |
| Falsify a provider while leaving scripted consumer rows green | unit versus linked evidence | provider defect caught without granting arm credit to integration |
| Add a match arm and a false postcondition together | arm evidence versus universal proof | separate reachability, execution, and proof diagnostics |
| E12: extract two inline contracts into one named relation, then add an error-state mutation and effect | contract ownership and old/new/trace models | proof invalidation, caller obligations, success/error binder clarity, unsupported-clause status |
| Change formatting, parameter name, contract, body, and backend independently | identity policy | layer-specific structural diffs and no accidental reacceptance |
| Exhaust fuel after acquiring a resource | termination/resource semantics | declared outcome or fault plus mandatory cleanup evidence |
| Rebuild the same sources with a different compiler/canonical format | artifact authority | incompatible format refusal and explicit review bridge |

Each trial should record agent edit count, wrong-file edits, diagnostic recovery,
whether the agent could predict invalidation before compiling, and whether CLI,
LSP, evaluator, proof checker, emitter, and accepted manifest agree. The strongest
candidate is not necessarily the one with the shortest source; it is the one whose
authority claims survive these mutations without hidden inference or blanket churn.

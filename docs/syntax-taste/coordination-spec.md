# Coordination and callable-completion contract

Status: technical companion for implementation planning, 20 September 2026.

This document completes the technical contract around the four selected
coordination forms. It does not authorize compiler implementation. The surface
spellings in `decisions.md` remain authoritative. If this companion conflicts
with a selected surface decision, the selected decision wins and the conflict
must be resolved before implementation.

The selected surface facts are: all authored functions execute asynchronously
underneath ordinary synchronous-looking calls; the four headers are `match call
concurrent`, `match call concurrent with error`, `match call race`, and `match
call race with error`; they use native `Promise.all`, `Promise.allSettled`,
`Promise.any`, and `Promise.race`; coordination entries omit `call`; nonvoid
coordination results use a typed binding before the block; and an all-failed
first-success race selects one bare `all_failed` arm. See
[effects and asynchronous execution](decisions.md#effects-and-asynchronous-execution),
[coordination completion-arm layout](decisions.md#coordination-completion-arm-layout),
[binding collected handler results](decisions.md#binding-collected-handler-results),
and [runtime-sized collections](decisions.md#runtime-sized-collections-of-different-calls).

The rules below are technical completions of those choices. They use the
already selected generic, error, variant, callable, array, and typed-binding
syntax. They add no coordination keyword, anonymous tuple, anonymous union,
general dynamic JSON value, task value, or exposed promise type. Permitting
error types as named-variant alternatives, and supplying the prelude types
`all_failed` and `standard_failure`, complete the expressly open variant/error
contract. They introduce no new token or declaration form.

The native facts used here come from the ECMAScript algorithms for
[`Promise.all`](https://tc39.es/ecma262/multipage/control-abstraction-objects.html#sec-promise.all),
[`Promise.allSettled`](https://tc39.es/ecma262/multipage/control-abstraction-objects.html#sec-promise.allsettled),
[`Promise.any`](https://tc39.es/ecma262/multipage/control-abstraction-objects.html#sec-promise.any),
and
[`Promise.race`](https://tc39.es/ecma262/multipage/control-abstraction-objects.html#sec-promise.race).
The compiler must call those operations in generated TypeScript and add only
the completion, typing, immutability, and lifetime adapters specified here, in
accordance with [AGENTS.md](../../AGENTS.md) and the
[native-reuse decision](decisions.md#reuse-native-operations-do-not-reimplement-them).

## Q1. What is a participant, a participant order, and a handler region?

A coordination block contains one or more syntactic participant entries. An
entry is either a direct invocation or a spread of an immutable array of
captured, nullary callable values. A block with no participant entry is a
compile-time error. A spread is an entry even when its array is empty at
runtime.

Before launch, the runtime flattens entries into one participant sequence:

1. Direct entries contribute one participant in written order.
2. A spread contributes its snapshotted callable elements in array order.
3. Entries after a spread follow all elements of that spread.

Every participant receives its permanent zero-based position in that flattened
sequence before any participant starts. All input-order rules in this document
refer to that position. A separate indexed failure record is unnecessary: an
entry's position in an aggregate array is its participant identity.

A completion arm body in coordination is a **local handler region**. `ok` in that region
supplies a value to the coordination construct; it does not finish the enclosing
function. A `do` or nested ordinary-data `match` inherits the current handler
region. A domain or standard failure that leaves the region leaves the entire
coordination construct. It is never redispatched to another participant arm.

Only coordination has the selected typed completion-binding form. This contract
does not add `T value = match call ordinary_call(...)` or a value-binding form
for `match chain`; those remain terminal completion-handling forms. A
coordination binding used to move the construct's value into ordinary code is
semantically necessary and must not be rejected as a removable forwarding
local.

## Q2. When are callees, spreads, and nested arguments evaluated?

Coordination has a deterministic prepare phase followed by a launch phase.

During **prepare**, the compiler/runtime evaluates, in written order:

1. each direct callee and each of its argument expressions, with argument
   expressions evaluated left to right;
2. each spread expression once, then snapshots its immutable callable array;
3. every ordinary nested call to completion under the normal automatic-await
   rule.

No participant starts during prepare. If prepare produces a standard failure,
that failure leaves the surrounding region under ordinary call rules; no
coordination arm handles it, and no participant has started. A nested call with
a nonempty `emits` bound is not an admissible unchecked argument expression
under [C5](technical-spec.md#c5), so a declared domain completion cannot
originate in prepare. Use a named participant wrapper when such a call must be
handled or forwarded as part of one launched participant.

During **launch**, every prepared participant is invoked in flattened input
order without awaiting an earlier participant's completion. At the end of the
launch phase, including when expansion produced zero participants, the resulting
native-promise array is passed once to the selected native Promise operation.

Consequently, in this entry:

```text
save(call load_account(account_id))
```

Assuming `load_account` has `emits []`, it finishes during prepare. `save` does
not begin until every entry's arguments are prepared. This rule preserves
ordinary written argument order, prevents a later argument failure from leaving
an accidentally launched prefix, and needs no wrapper solely to make the launch
boundary legible. A named wrapper remains useful when the author wants the
dependent load itself to occur inside one launched participant.

Callable creation and `near` capture occur when `callable name` is evaluated,
before a later spread. A spread never recaptures surrounding values.

## Q3. How do Can completions drive native Promise settlement?

The compiler creates one private adapter promise per participant:

- a Can success fulfills the adapter with a private non-thenable success box
  containing the complete typed success payload;
- a declared domain error rejects it with a private tag containing the exact
  nominal error value, stable error ID, generic specialization, and payload;
- a standard failure rejects it with a distinct private tag containing a
  `standard_failure` snapshot.

These boxes and tags are compiler-private generated TypeScript values under
[C4's payload-shielding contract](technical-spec.md#c4). They are not Can
records, user-visible Promise rejection reasons, or authored completion types.
The aggregate operation is the native Promise operation over
those adapter promises. After native settlement, the compiler-owned adapter
recovers the Can category and invokes the selected arm.

This conversion is required because native promises distinguish fulfillment
from rejection, while Can distinguishes success, declared domain errors, and
standard failures. It also ensures that a Can domain error cannot accidentally
win `Promise.any` as a fulfillment. Native `AggregateError` is never exposed as
a Can error and never supplies Can's error identity.

## Q4. What result type does each coordination form produce?

Let direct participant `i` have success type `S_i` and declared domain-error
set `E_i`. Let `T` be the successful result type of a local handler. An
unbound coordination statement has expected type `void`; all per-participant
and shared handlers that recover successfully must therefore use bare `ok`.
The construct produces void, stores no `void[]`, and may be followed by later
steps. Values are not silently discarded. A function still needs an explicit
terminal completion after such a statement.

| Form | Participant success types | Local successful handler results | Bound result |
|---|---|---|---|
| `concurrent` | Direct calls may have different `S_i`; a spread is homogeneous | Each per-call success arm yields one `T`; a handled shared failure arm yields the whole `T[]` | `T[]` |
| `concurrent with error` | Direct calls may have different `S_i`; a spread is homogeneous | The selected success/error arm for each participant yields one common `T` | `T[]` |
| `race` | Every participant has exactly the same success type `S` | Shared success and `all_failed` arms each yield one common `T` | `T` |
| `race with error` | Every participant has exactly the same success type `S` | Every shared success/error arm yields one common `T` | `T` |

There is no implicit union result. Heterogeneous direct calls in either
`concurrent` mode use their own success handlers to construct a declared common
record or variant. Heterogeneous successes cannot use a shared race success arm;
named wrappers must first normalize their function success types to one declared
type. Callable results and inputs are invariant.

For plain `concurrent`, a shared error handler does not correspond to one array
slot. If it recovers successfully, it must construct the complete fallback
`T[]`. It cannot yield a single `T` for an unspecified slot. If it forwards or
produces an error, it produces no successful array.

For `concurrent with error`, exactly one locally successful value is appended
for every participant whose selected arm succeeds. Values are placed in input
order, independent of settlement order. If a handler fails, no result array is
produced.

A participant returning `void` can be used only in an unbound, void
coordination unless an arm explicitly constructs a non-void `T` without a
success payload. `void[]` is not introduced.

## Q5. Which completion arms are required, and what do they cover?

For a direct call, coverage uses the call's complete declared domain-error
upper bound, including any compiler-owned fixed domain errors in a native
declaration's exported signature. For a spread, coverage uses the callable
array element's declared error upper bound. Runtime membership cannot weaken
that static requirement.

Callable compatibility for coordination is:

- invocation inputs and success results are invariant;
- after `near` and receiver capture, a spread callable has no remaining ordinary
  inputs;
- the referenced callable's declared domain-error set may be a subset of the
  expected callable element bound;
- standard failures remain outside that bound.

The whole-callable array spelling is:

```text
callable receipt () emits [postgres_failed, redis_failed][] operations
```

The `[]` applies to the complete callable type after its `emits` list. An
explicit expected callable-array type is required when different references
need a widened common error bound; this rule does not introduce general inferred
error-set unions.

Coverage by mode is exact:

| Form | Domain arms | Standard arm |
|---|---|---|
| `concurrent` | One shared arm using the bare error-name pattern for every distinct error type in the union of participant bounds | One optional shared `[_]` arm |
| `concurrent with error` | Beneath each direct entry, one arm using the bare error-name pattern for every error that entry declares; beneath a spread, the corresponding arms for its element bound | One optional `[_]` beneath each direct entry or spread |
| `race` | No participant-error arms; one mandatory arm using the bare `all_failed` pattern replaces all of them | No `[_]` participant arm; standard failures are aggregate members |
| `race with error` | One shared arm using the bare error-name pattern for every distinct error type in the union of participant bounds | One optional shared `[_]` arm |

Omitting `[_]` never claims freedom from standard failures. It selects the
automatic propagation rule in Q7. A domain arm not present in the applicable
bound, a duplicate arm, a participant-error arm under unmodified `race`, or an
`all_failed` arm under another mode is a compile-time error.

Every generic error specialization is a distinct error type for bounds. A
single completion match cannot contain two specializations of the same error
kind when both use the same bare error-name pattern. For example, an outer block
whose participants expose both `all_failed<a_failure>` and
`all_failed<b_failure>` is ambiguous and must be rejected. The author must use
named wrappers that consume both specializations, explicitly widen their data
to one declared failure variant, and emit one common specialization. Generic
parameters remain invariant; the compiler must not silently choose a lossy
union.

## Q6. What exactly is `all_failed`?

The prelude reserves error ID `100` and the unqualified name `all_failed`:

```text
error 100 all_failed<failure>(failure[] failures)
```

Application code cannot redeclare or shadow this prelude name or allocate ID
100. All generic specializations are the same error kind and share ID 100, as
all specializations of an ordinary generic error share their declaration ID.
The arm pattern remains the selected bare `all_failed`; generic arguments do not
appear in an error pattern.

Every source-denotable specialization `all_failed<F>` requires `F` to be a
named finite variant. That variant may list record types, declared error types,
the prelude type `standard_failure`, and other named variants that recursively
flatten to disjoint nominal leaves. Error types are ordinary nominal data types
in such a variant. A captured domain failure value retains its error kind,
stable ID, generic arguments, and complete typed payload. It is not a message
string. A locally handled aggregate whose payload is never observed can remain
a compiler-private tag over its closed leaf set; it does not create an anonymous
source type.

`standard_failure` is a compiler-owned, constructor-restricted nominal snapshot
record. Authors can name it as a variant alternative and match it, but cannot
construct or copy-update it or access the raw native rejected/thrown value. It
has read-only projections:

- `int occurrence_id`, unique within one program run and stable for that captured
  occurrence;
- `str kind`, supplied by the shared standard-failure adapter;
- `str message`, the same canonical diagnostic description that an ordinary
  bound `[_]` handler would receive.

The runtime retains the exact original native value behind that snapshot until
the snapshot is unreachable. `kind` and `message` are projections; they do not
replace, serialize, or become the identity of the original failure. Their exact
cross-runtime classification and description use the single shared table in
[C9](technical-spec.md#c9), identically for ordinary `[_]` and coordination.

An ordinary data match on a failure variant narrows the original scrutinee
binding to the matched leaf. When that leaf is a declared error type, the bare
error pattern also introduces the usual error-name alias to the same value, so
both the narrowed scrutinee and error name can access its payload. A
`standard_failure` leaf is a record type, not an error declaration, so it has no
error-name alias; access it through the narrowed scrutinee.

```text
variant lookup_failure
    primary_unavailable
    backup_unavailable
    standard_failure

// `failure` is narrowed in every arm; error leaves also have error-name aliases.
match failure
    primary_unavailable => ok primary_unavailable.message
    backup_unavailable => ok failure.message
    standard_failure => ok failure.message
```

Error-constructor syntax is disambiguated by expected context. In a completion
position `primary_unavailable("offline")` produces that domain completion. In
an ordinary value position expecting `lookup_failure`, the same constructor
creates its nominal error data value. `ok primary_unavailable("offline")`
therefore succeeds with error data rather than producing an error completion.

### Finite inference and coverage

For one `race`, the checker computes a closed internal leaf set `U`:

1. flatten the declared domain-error bounds of every direct and spread entry;
2. retain complete generic specializations;
3. collapse identical error types but reject two specializations of the same
   bare-pattern error kind as ambiguous;
4. add `standard_failure`, because standard failure remains possible even for
   `emits []`.

The runtime array still contains one entry per failed participant. Type-level
deduplication never removes duplicate runtime occurrences.

When source observes, passes, or forwards `all_failed.failures`, expected-type
inference must identify exactly one named variant `F`. The checker flattens
`F` to disjoint nominal leaves and requires every leaf of `U` to inject into
exactly one leaf of `F`. Extra leaves in `F` are allowed so a named handler can
serve several races. The compiler does not search all declared variants for a
least common type and does not invent a source-visible anonymous union.

Two contexts provide the normal inference:

- a typed named call such as
  `call summarize_failures(all_failed.failures)`, whose parameter is
  `lookup_failure[]`;
- forwarding under an enclosing function declaration containing
  `emits [all_failed<lookup_failure>]`.

If the handler ignores `failures` and returns a fallback, the internal closed
set need not acquire a source name. If source observes the field without one
unique expected named variant, or inconsistent uses demand different variants,
the program is a compile-time error.

Arrays remain invariant. The elementwise injection from the internal failure
set into `F[]` occurs while the aggregate is created; it is not general array
covariance. Array literals may likewise inject their individual members into an
expected variant element type. Converting an existing `narrow_failure[]` to a
different `wide_failure[]` requires an explicit elementwise transformation.

This declaration fragment shows a typed top-level aggregate handler without new
race syntax:

```text
variant lookup_failure
    primary_unavailable
    backup_unavailable
    standard_failure

fn lookup_result summarize_failures
    emits []
    given
        lookup_failure[] failures
    // Mandatory assertions and body are omitted from this signature fragment.

lookup_result result = match call race
    primary::lookup(user_id)
    backup::lookup(user_id)
    ok profile found => ok lookup_result(true, found.name, 0)
    all_failed => ok call summarize_failures(all_failed.failures)
```

The call supplies the expected `lookup_failure[]` type. If every participant
fails, the adapter creates one `all_failed<lookup_failure>` and invokes the arm
once. No per-error handler runs.

## Q7. What is the priority of domain errors and standard failures?

The selected native operation determines participant priority. Handler failures
occur later and follow ordinary propagation.

### `concurrent` / `Promise.all`

The first adapter rejection to settle the native aggregate wins, whether it
contains a domain error or a standard failure. A domain error selects its shared
arm. A standard failure selects shared `[_]` when present and otherwise
propagates. No success handler runs, including handlers for participants that
already succeeded.

### `concurrent with error` / `Promise.allSettled`

The construct waits for every participant settlement, then dispatches outcomes
in input order. A domain error selects that participant's arm. A standard
failure selects that participant/spread `[_]` when present. Without `[_]`, it
propagates when its input slot is reached; earlier handlers may already have
run, and later handlers do not run.

### `race` / `Promise.any`

The first success wins. Earlier domain or standard failures remain failed
participants and do not propagate while another participant can succeed. If a
later success wins, those losing failures are observed and released by the
runtime owner; they never reappear in the completed caller.

If every participant fails, all domain and standard failures become the single
`all_failed.failures` array in input order, irrespective of settlement order.
No `[_]` runs. A standard failure in this array remains a `standard_failure`
value with its opaque original identity.

### `race with error` / `Promise.race`

The first participant completion wins. Success selects the shared success arm,
a domain error its shared domain arm, and a standard failure shared `[_]` or
automatic propagation. Later settlements do not change the result.

## Q8. What if a selected handler itself fails?

Handler execution is outside the participant aggregate. A handler-produced
domain error leaves the construct and must be permitted by the enclosing
function's `emits` bound or handled inside that handler with an ordinary nested
completion form. A handler-produced standard failure propagates under the
ordinary standard-failure rule. Neither is caught by participant arms in the
same block.

A bare participant-error arm forwards that exact original error and payload out
of the construct. A bare `all_failed` arm forwards its inferred
`all_failed<F>`. Forwarding does not count as a locally successful `T` result.

`relay call` inside a handler terminates that handler region. A callee success
becomes the handler's local success and must have the required local result
type; a callee error leaves the coordination construct. An ordinary `match call`
or `match chain` nested in a handler likewise resolves its arms in the handler
region. Their `ok` arms do not finish the enclosing named function, and their
errors are not caught by the surrounding coordination participant arms. This
applies the same region ownership to `do`, relay, call matching, and chains
without adding a general stored-completion value.

In either concurrent mode, post-settlement handler processing stops at the first
handler failure in input order. For `concurrent`, success handlers are reached
only when every participant succeeded. For `concurrent with error`, every
participant has already settled, but later local handlers still do not execute
after a handler failure. No partial `T[]` becomes observable.

The checker derives the coordination expression's escaping domain-error set
from bare forwarding plus every call or constructor that can fail inside its
handlers. Handled participant errors do not escape. Handler errors with the
same name as a participant error are still new completions and are not
redispatched.

## Q9. What happens when runtime expansion is empty?

The rules deliberately retain each native empty-input result:

| Form | Empty flattened sequence |
|---|---|
| `concurrent` | native `Promise.all([])` succeeds; the bound result is `[]`; no arm runs |
| `concurrent with error` | native `Promise.allSettled([])` succeeds; the bound result is `[]`; no arm runs |
| `race` | native `Promise.any([])` rejects with an empty aggregate; Can invokes `all_failed` once with `failures = []` |
| `race with error` | native `Promise.race([])` remains pending forever; no arm runs |

For an empty race spread, its callable element bound still supplies the finite
internal domain leaf set. If the payload is observed or forwarded, the
handler's expected context supplies named variant `F`, and the runtime value is
`all_failed<F>([])`. If the handler ignores the payload and maps to a fallback,
no source specialization is required. The possibility of an empty payload is
part of every aggregate handler's contract. A handler must not assume at least
one failure merely because the error is named `all_failed`.

The nontermination of empty `race with error` is an explicit consequence of the
selected exact native mapping, not an implicit timeout. Authors who need a
nonempty guarantee require a separately validated nonempty collection type or a
prior check; neither is introduced here.

## Q10. Who owns unfinished calls, resources, and late rejections?

Every coordination operation creates a runtime owner before launch. The owner
retains participant promises, completion tags, captured values, resource leases,
and rejection observers until every launched participant settles. Early native
settlement transfers only the aggregate result to the handler; it does not
destroy or reuse the owner.

There is no automatic loser cancellation. A late success or domain failure is
observed and released without running another arm. A late standard failure is
also observed, so it cannot become an unhandled JavaScript rejection. It emits
one sanitized runtime diagnostic containing its occurrence ID, kind, and C9
message, but it cannot alter, rethrow into, or retroactively invalidate the
already completed Can caller. The diagnostic alone does not change process
status: after a successful CLI `main`, complete owner drain still exits zero.
An independently detected resource leak, cleanup failure, or shutdown deadline
violation retains P's nonzero-exit rule.

An invocation may complete while one of its coordination owners remains live.
The owner is then retained by the program or service root supervisor rather than
by the completed lexical handler scope. The participant's assertion invocation
identity and captured data remain isolated; they are never reassigned to a later
call or test.

The generated module uses top-level await for the root supervisor, which awaits
`main` and then shutdown/drain; it must not fire-and-forget the root call.
A bounded Bun 1.4.2 probe of top-level `await Promise.race([])` remained pending
until explicitly interrupted, supporting the selected empty-race contract.

At successful CLI `main` completion, the runtime begins the root shutdown
contract in
[P6](platform-testing-spec.md#p6-opaque-values-callbacks-and-resource-enforcement).
It cannot report a clean zero exit
while a live coordination owner remains. An owner holding a catalogue resource
is subject to that resource's registered shutdown deadline and nonzero cleanup
rules. A pure participant with no applicable deadline that never settles can
therefore prevent clean shutdown indefinitely. This is the honest default in
the absence of a selected universal timeout or cancellation contract. An
explicit HTTP request deadline may abort that request under A4; server/pool
close deadlines follow P's lease-preserving failure rule. Neither retracts
effects that already occurred. Transactions have no implicit deadline. An empty
`race with error` owns a permanently pending native aggregate; a resource-close
deadline does not settle it or permit a clean root drain.

A resource deadline bounds its close caller's wait and reports failure; it does
not guarantee that the process exits afterward. A permanently live lease can
keep the root pending even after that failure. Deployments needing bounded host
termination must use an external process supervisor's kill deadline. Such a
host kill abandons the process; it is neither Can cancellation nor evidence of
clean resource drain or remote-effect rollback. Can does not install a hidden
hard-stop policy.

A resource captured by a participant remains leased to its owner until that
participant settles. A close, commit, rollback, or release operation must never
invalidate a live lease. Each catalogue resource contract must either wait for
leases or produce its declared busy/lifecycle failure. It must not silently
close underneath the participant. This is runtime lifecycle checking, not a new
general affine/effects type system. The need follows the resource-bearing
capability intent in ASTRA's [web backend](../ASTRA_STDLIB.md#3-web-backend-standard)
and [SQL connector](../ASTRA_STDLIB.md#4-standard-sql-connector), whose old
surface mechanisms are not authoritative.

An assertion runner must drain all owners before declaring the assertion passed,
so a losing participant cannot affect a later assertion. A harness deadline may
fail a test whose owner never settles; that harness deadline is not a language
timeout and must not be used to claim production cancellation behavior.

## Q11. How do async callables interact with native callback APIs?

All generated Can functions are async. A Can callable must therefore never be
passed unchanged where a native API consumes a synchronous return value.
`Array.prototype.map` would otherwise produce promises as elements;
`filter`, `every`, and `some` would treat a returned promise as truthy; and a
native sort comparator cannot await a promised number. These are callback ABI
facts, not reasons to reimplement an upstream collection or sorting algorithm.

The initial adapter rules are:

- selected `for_each` remains sequential and awaits each callable before the
  next element, as required by the
  [iteration decision](decisions.md#iteration-and-early-completion), and native
  `.reduce` builds its await chain;
- sequential `map` uses native `Array.fromAsync` over native source-array
  indices, with boxed Can callback results and native synchronous unboxing;
  user callbacks run one at a time in element order, and rejection prevents
  later callbacks. C4's non-thenable boxes protect both inputs and outputs;
- async `filter` first obtains sequential awaited boolean decisions, then uses
  native `.filter` with a compiler-owned synchronous predicate over that vector;
- `sort_by` extracts one int, finite-float, str, or bool key per element
  sequentially, then uses native `.toSorted` over key/index decorations with a
  compiler-owned synchronous comparator; equal keys retain input order and a
  nonfinite float key is a standard `arithmetic` failure;
- arbitrary authored async comparators are not admissible to native sort;
- async `find`, `some`, and `every` visit in ascending order and stop at the
  first true, true, or false result respectively, using the bounded await
  adapter selected in [C7](technical-spec.md#c7); empty results are `none`,
  false, and true;
- async `fold` uses native `.reduce` with an awaited accumulator and the
  explicitly supplied initial value, including on empty input.

Each operation exposes the callback's finite declared error bound and stops
under its stated rule on the first domain or standard failure. It exposes no
partial result. [C7](technical-spec.md#c7) is the complete collection catalogue
contract; this section states the callable/native ABI consequences shared with
coordination.

These adapters preserve Can completion/error contracts around native storage and
selection operations. They do not authorize a handwritten replacement for
native sort, filter, or Promise coordination. Reopening the all-functions-async
decision to create a synchronous callable subset is a separate language policy
choice and is not assumed here.

## Q12. Which execution traces and conformance tests are mandatory?

Implementation tests must use manually controlled deferred promises or an
equivalent deterministic scheduler hook. Wall-clock sleeps are not sufficient
evidence for winner, ordering, or late-rejection behavior. The generated code
must also be inspected to establish that the four native Promise operations are
actually used.

### Trace 1: `concurrent` rejects by settlement, not input order

Participants are `[A, B, C]`. `C` succeeds, then `B` produces a domain error,
then `A` produces a different domain error. Native `Promise.all` rejects when
`B` rejects. The shared `B` error arm runs once. No success arm and no `A` arm
runs. `A` remains owned until it settles, and its late domain error is observed
without redispatch.

Repeat with `B` producing a standard failure. Shared `[_]` receives it when
present; otherwise it propagates. In both cases the late `A` rejection must not
produce Bun's unhandled-rejection reporting.

### Trace 2: `concurrent with error` separates settlement from handling order

Participants `[A, B, C]` settle in order `C`, `A`, `B`. After all settle, arms
run `A`, `B`, `C`. Their successful mapped values appear in that order. If
`B`'s selected handler fails, `C`'s handler does not run and no partial array is
observable, even though `C` had already settled.

Repeat with an unhandled standard failure in slot `B`. `A`'s handler runs;
automatic propagation occurs at `B`; `C`'s handler does not run.

### Trace 3: `race` preserves original failures in input order

Participants `[A, B, C]` all fail in settlement order `B`, `C`, `A`. `A` is a
standard failure; `B` and `C` are distinct domain errors with nonempty payloads.
The single handler receives:

```text
[standard_failure_for_a, exact_b_error_value, exact_c_error_value]
```

The observed domain entries must expose their original nominal kinds, IDs,
generic specializations, and field values. The standard snapshot must retain the
original opaque native occurrence and expose its stable occurrence ID. No error
is converted into an aggregate string.

Repeat with `C` succeeding last. The shared success arm runs once; `all_failed`
does not run; the earlier standard failure from `A` never propagates into the
completed caller.

### Trace 4: `race with error` selects the first completion

`B` produces a domain error before `A` succeeds. The `B` arm runs, `A` does not
win later, and any later standard failure is observed only by the runtime owner.
Repeat with a standard failure first, both with and without `[_]`.

### Trace 5: prepare is all-or-nothing

Entry `A` has a nested `emits []` argument call that succeeds; entry `B` has a
nested `emits []` argument call that produces a standard failure. Neither `A`
nor `B` participant launches. With both arguments successful, the trace must
show both argument calls completing in written order, followed by launching
`A` then `B` without awaiting `A`.

### Trace 6: empty expansions

For an empty typed callable spread, verify `[]` from both concurrent modes, one
`all_failed` invocation with an empty payload from `race`, and continued pending
state from `race with error`. The final case must be observed with the harness
deadline, not converted into a Can timeout completion.

### Trace 7: heterogeneous direct calls and homogeneous spreads

Use two direct calls returning unrelated named records. Their per-call
concurrent handlers explicitly construct alternatives of one declared result
variant; the bound type is that variant array. The same calls must be rejected
from one race until wrappers normalize both function success types.

Construct an explicitly typed callable array whose members have the same success
type, no remaining inputs, and narrower declared errors than the array's common
bound. The spread must typecheck, and its arms must cover the full common bound.
Reject a member with a different success type, a remaining ordinary input, or
an error outside the bound.

### Trace 8: handler failures are not participant outcomes

Make every participant succeed, then make the first `concurrent` success handler
produce a domain error. That error must leave the construct; no shared
participant-error arm catches it, and later success handlers do not run. Repeat
with a standard failure and with the `all_failed` handler itself failing.

### Trace 9: generic aggregate composition

Create two functions emitting `all_failed<a_failure>` and
`all_failed<b_failure>`. Putting both directly in a completion form with one
bare `all_failed` pattern must fail static checking. Then add wrappers that
exhaustively transform their arrays into one declared `combined_failure[]` and
emit `all_failed<combined_failure>`. The normalized participants must typecheck
with one bare arm, preserving every original leaf payload.

### Source assertion integration

No `when` table beneath coordination is selected. Consumer assertions should
initially put substitutable ordinary calls inside named participant wrappers;
their existing call-site `when` tables remain inside those wrappers. The test
runtime assigns each dynamic invocation path its prelaunch participant position,
so concurrent scheduling cannot choose fixture rows. Controlling which
participant settles first requires the deterministic coordination harness; a
fixture's allocation order alone is not settlement order.

This test boundary distinguishes source consumer assertions from compiler/native
conformance. A passing substituted assertion does not prove that generated
TypeScript uses the required Promise operation, retains late rejection
observers, or preserves native failure objects.

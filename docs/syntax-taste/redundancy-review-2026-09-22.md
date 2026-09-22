# Can redundancy review — 2026-09-22

Status: design input, not decisions. Nothing here is approved until it goes
through the normal decision process. Per `AGENTS.md`, adopting any redesign
below requires three fresh Jev consultations (fully reworded packets, saved
requests/responses) before committing. Per the same file, there are zero
external users: no suggestion below preserves old spellings, ABIs, or goldens.

Method: read the four spec companions (`technical-spec.md`, `coordination-spec.md`,
`ai-io-spec.md`, `platform-testing-spec.md`, ~5,200 lines), `decisions.md`, and the
admitted examples under `examples/`, `std/`, and `compiler/testdata/current/`.
Each cluster shows a current `.can` example from the repo, then two alternative
sketches (A = structural unification, B = smaller compatible-with-thesis change).
Sketches are illustrative, not final grammar.

## R1 — The native declaration zoo

Eleven top-level declaration forms with bespoke headers and section orders:
`fn`, `record`, `variant`, `error`, `connection`, `noul`, `choice`, `score`,
`choice_arm`, `judge`, `llm`, `fetch`. Each has its own grammar production,
section-order rules (A3.1), and checker/emitter paths.

### Current

Four ways to ask one judgment question, each with different header shape and
handler rules (`compiler/testdata/current/native/questions.can`):

```text
noul float likelihood from classifier
    emits [ai::invalid_question, ai::invalid_answer, codec::invalid_data, io::write_failed]
    asks "Noul"
        true "Yes" => relay call report(%, "N")
        false "No" => relay call report(%, "n")

choice float choose from classifier
    emits [ai::invalid_question, ai::invalid_answer, codec::invalid_data, io::write_failed]
    confidence as certainty
    asks "Choice"
    minimum 0.5 => relay call report(certainty, "f")
        first "First" => relay call report(% + certainty, "A")
        second "Second" => relay call report(% + certainty, "B")

score float rate from classifier
    emits [ai::invalid_question, ai::invalid_answer, codec::invalid_data, io::write_failed]
    score as measured
    confidence as certainty
    minimum 0.5 => relay call report(certainty, "g")
    asks "Score"
        low "Low"
        medium "Medium"
        high "High"
        ok => relay call report(measured + certainty, "S")

record choice_weights choice float weights from classifier
    emits [ai::invalid_question, ai::invalid_answer, codec::invalid_data, io::write_failed]
    confidence as certainty
    asks "Weights"
        first "First" => relay call report(% + certainty, "C")
        second "Second" => relay call report(%, "D")
```

Note the section-order variance: Noul puts `minimum` after `asks`, Score puts
`minimum` before `asks`, Choice has *two admitted productions* for where
`confidence`/`minimum` sit (A3.1) solely to preserve two shown layouts.

### Suggestion A — one `question` declaration, kind + shape as parameters

```text
question float choose from classifier
    kind choice
    emits [ai::invalid_question, ai::invalid_answer, codec::invalid_data, io::write_failed]
    confidence as certainty
    minimum 0.5 => relay call report(certainty, "f")
    asks "Choice"
        first "First" => relay call report(% + certainty, "A")
        second "Second" => relay call report(% + certainty, "B")
```

`kind` is one of `noul`, `choice`, `score`; a `shape record choice_weights`
line selects the run-every-handler generated-record mode. One header grammar,
one section order, one checker; kind-specific rules (option counts, `%`
availability) become validation of section contents, not productions.

### Suggestion B — keep the kinds, unify the sections

Keep `noul`/`choice`/`score` keywords but require one section order for all
(`emits`, `given`, binders, `minimum`, `asks`), delete the second Choice
production, and spell record forms as a trailing `shape` clause instead of a
second header grammar:

```text
choice float weights from classifier
    shape record choice_weights
    ...
```

B keeps keyword-level readability for agents grepping question kinds while
deleting most of A3.1's ordering special cases.

## R2 — `choice_arm` vs `callable`: two function-value types

`callable` has `near` capture and `call` invocation; `choice_arm` is
capture-free with spread invocation — and they use different type syntax.

### Current

```text
choice_arm float left_arm
    emits [codec::invalid_data, io::write_failed]
    describes "Left."
    relay call report(%, "L")

record arms
    choice_arm<float> emits [codec::invalid_data, io::write_failed] left
    choice_arm<float> emits [codec::invalid_data, io::write_failed] right
```

(From `compiler/testdata/current/native/questions.can`; arms are then spread
as `...all_depts` beneath `asks`.) Compare the callable spelling for the same
payload type: `callable float (float) emits [...]`.

### Suggestion A — arms are callables with a `describes` line

```text
fn float left_arm
    emits [codec::invalid_data, io::write_failed]
    given
        float probability
    describes "Left."
    asserts
        sample: 0.5 => ok 0.5
    relay call report(probability, "L")
```

`describes` marks an ordinary named function as usable in option position;
`%` becomes an ordinary input. One function-value type, one capture rule, one
compatibility rule. Arm records become `callable float (float) emits [...]`
fields, and the arm-record spread is just the existing spread of a record
whose fields share one type.

### Suggestion B — keep the arm kind, unify the type syntax

Keep `choice_arm` declarations but spell the stored type exactly like a
callable (`callable float (float) emits [...]`, with `describes` carried as a
declaration attribute the checker verifies at spread sites). Removes the
`choice_arm<T>` generic-look type without touching declaration semantics.

## R3 — `connection` vs `record`: config that cannot be data

A connection is immutable configuration that cannot be constructed, passed, or
returned, which forces the `from`-resolution rules, connection-identity checks
(A4/A7: "independently declared connections with identical fields are not"
equal), and per-declaration `from` clauses.

### Current

```text
connection classifier
    endpoint "http://127.0.0.1:1/systemone"
    auth Bearer [REDACTED] "CAN_I28_TOKEN"
    timeout_ms 5000
    metadata
        protocol "typesafe_systemone_v1"
        model "jev-latest"

llm records::triage_plan draft from generator
    emits [...]
    state
        str account
    asks "Draft"
```

(From `examples/native-ai/src/oracles/oracles.can`.)

### Suggestion A — connections are ordinary validated records

```text
record connection
    str endpoint
    str auth_variable
    int timeout_ms
    str protocol
    str model

connection classifier = call connection::validated(
    "http://127.0.0.1:1/systemone", "CAN_I28_TOKEN", 5000,
    "typesafe_systemone_v1", "jev-latest")

llm records::triage_plan draft
    emits [...]
    given
        connection service
    state
        str account
    asks "Draft"
```

`from` disappears; the connection is an ordinary input (judge batching checks
it by value equality). Deletes a declaration kind, `from` grammar, and the
identity rules. Credential handling is unchanged: the record carries the
*variable name*, the runtime reads it once per request.

### Suggestion B — keep `connection`, drop `from`

Resolve the connection lexically: the nearest `connection` in scope (or the
file's single `uses`-visible connection) applies; a second connection in one
file requires an explicit `service` input. Keeps the config-declaration
readability while deleting `from` clauses and cross-declaration identity
checks.

## R4 — Completion forms: one `match`, five rulebooks

Value `match`, `match call`, `match chain`, four coordination headers, and
native handlers each have their own arm-coverage algorithm. The checker cost is
visible: `check/completions.go` (~660 lines) plus separate coordination and
question checkers.

### Current

Four headers, three arm layouts (`compiler/testdata/current/coordination/main.can`):

```text
int[] concurrent = match call concurrent
    one()
        ok int value => ok value
    text()
        ok str value => ok value.length
int[] settled = match call concurrent with error
    unavailable()
        ok int value => ok value
        codec::invalid_data => ok 2
    one()
        ok int value => ok value
int raced = match call race with error
    unavailable()
    one()
    ok int value => ok value
    codec::invalid_data => ok 9
int first_success = match call race
    unavailable()
    one()
    ok int value => ok value
    all_failed => ok 0
```

`concurrent` nests success arms per call but shares error arms; `concurrent
with error` nests both; `race` forms share both. Authors must memorize which.

### Suggestion A — one header, mode word, one arm layout

```text
int[] settled = match call each with error
    unavailable()
        ok int value => ok value
        codec::invalid_data => ok 2
    one()
        ok int value => ok value
```

One rule: arms nest beneath the entry they cover; a trailing shared arm
covers any error not already handled beneath an entry. `each` (= all),
`first` (= any), `first_settled` (= race) are mode words after `match call`,
not separate headers. `match chain` dissolves into the same form with
sequential mode, and `relay call f(x)` becomes `match call f(x)` with a single
shared forward arm — or stays as pure sugar.

### Suggestion B — keep the four headers, unify the arm layout

Keep `concurrent` / `concurrent with error` / `race` / `race with error` as
selected vocabulary, but require per-entry nested arms in all four modes
(shared trailing arms allowed as fallback). Deletes the per-mode arm-placement
table (Q5 rows) while keeping the searchable header names.

## R5 — `ok` / `as` / `...` overloading

- `ok` has ~7 roles: success constructor (`ok expr`), arm pattern
  (`ok int x =>`), bare forward (`ok` alone), void completion (bare `ok`),
  assertion expectation (`=> ok 3`), judge continuation (`ok =>`), chain arm
  (`ok =>`).
- `as` binds in chains (`call f() as user u`), judge registrations
  (`call q() as str name`), `[_] as str message`, and `confidence/score as name`.
- `...` has six meanings: call-arg spread, array-literal spread, array-pattern
  rest, variadic parameter, coordination entry spread, and choice-option spread
  (itself two modes: arm-record spread vs `choice_option[]` spread, plus inline
  options — three ways to feed one `asks`).

### Current

```text
match call load_account(found_user.account_id) as account found_account  // `as` binding
    ok => ok checked_account                                            // `ok` arm
    insufficient_balance                                                // bare forward
```

### Suggestion A — split construction from forwarding

Reserve `ok` for constructing success; spell forwarding explicitly:

```text
match call lookup(key)
    pass ok
    pass missing
```

`pass <pattern>` forwards the matched completion unchanged. Bare `ok` / bare
error arms disappear, and the void-completion `ok` vs success-forward `ok`
ambiguity goes with them. Similarly, replace every `as <type> <name>` with the
ordinary typed-binding shape already used by coordination results
(`account found_account = call ...`).

### Suggestion B — keep the tokens, delete the most confusing members

Keep `ok`/`as`/`...` but: (1) unify the two choice-option spreads into one
(spread always takes `choice_option[]`; arm records convert through an
ordinary function); (2) require `as` bindings to match typed-local order
everywhere (`type name`, which they already do — just delete the `as`
keyword); (3) forbid bare `ok` arms in favor of `relay`-style explicit
forwarding only at function tail.

## R6 — `given` vs `state`: two parameter lists, forced wrappers

Judges and LLMs take ordinary args plus a final grouped `(state)` argument.
Documented consequence (A3): judge/LLM callable references are rejected
because "ordinary callable types do not describe grouped state arguments" —
so every reusable native call needs a hand-written wrapper.

### Current

Pure-forwarding wrappers, ~20 lines each
(`examples/native-ai/src/oracles/oracles.can`):

```text
fn records::triage_plan draft_account
    emits [http::invalid_request, http::credentials_missing, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data, llm::refused, llm::truncated, llm::invalid_response]
    given
        str account
    asserts
        sample: "Ann" => ok records::triage_plan(["vip", "new"], true)
        garbage: "Ann" => llm::invalid_response("shape")
    match call draft((account))
        when
            sample: ("Ann") => ok records::triage_plan(["vip", "new"], true)
            garbage: ("Ann") => llm::invalid_response("shape")
        ok records::triage_plan plan => ok plan
        http::invalid_request
        http::credentials_missing
        http::transport_failed
        http::timeout
        http::body_limit
        http::status_error
        codec::invalid_data
        llm::refused
        llm::truncated
        llm::invalid_response
```

The wrapper exists only to adapt grouped-state calling to ordinary calling.

### Suggestion A — one parameter list, `state` as a per-parameter marker

```text
llm records::triage_plan draft from generator
    emits [...]
    given
        state str account
    asks "Draft"
```

Call sites pass arguments uniformly (`call draft(account)`); the `state`
marker controls only serialization grouping. Judge/LLM references become
ordinary callables, wrappers like `draft_account` disappear, and the
"grouped at every arity including `call name(())`" rule (C2) is deleted.

### Suggestion B — the state group is the last record-typed parameter

```text
record draft_state
    str account

llm records::triage_plan draft from generator
    emits [...]
    given
        draft_state state
    asks "Draft"
```

No new marker: state grouping is just "the last parameter, of record type, is
serialized as the state object." Slightly more verbose at declarations, zero
new syntax, and ordinary callable types describe judges/LLMs unchanged.

## R7 — Failure channels: four mechanisms for one idea

Domain errors (`emits` + arms), standard failures (`[_]` + string message),
the generic `all_failed<F>` aggregate with expected-named-variant inference
(Q6 — the most intricate typing rule in the language, serving exactly one
combinator), and the constructible-only-by-runtime `standard_failure`
snapshot.

### Current

Every `race` needs a bespoke variant plus a generic error in `emits`
(`compiler/testdata/current/coordination/main.can`):

```text
variant failure
    codec::invalid_data
    standard_failure

fn int fail_all
    emits [all_failed<failure>]
    asserts
        sample: => all_failed<failure>([codec::invalid_data("", "type")])
    int result = match call race
        unavailable()
        ok int value => ok value
        all_failed
    ok result
```

The author declares `failure`, the checker runs Q6 inference (flatten bounds,
inject leaves into exactly one named variant, reject ambiguity) — all to type
one array argument. Two `race` blocks with different participant errors in one
function need two variants or a shared widened one.

### Suggestion A — non-generic `all_failed` over a closed variant

```text
variant failure_data
    codec::invalid_data
    standard_failure
    // ... closed set, extended only by the distribution

error 100 all_failed(failure_data[] failures)
```

`all_failed` is an ordinary error carrying an ordinary array; no generic
parameter, no inference, no per-race variant declarations. Authors match
`failure_data` leaves with the existing variant rules. The distribution owns
the closed leaf set the same way it owns error IDs.

### Suggestion B — `race` returns an ordinary variant, no special error

```text
variant race_outcome<success>
    success
    failure_data[]
```

First-success race produces `success` or the collected failures array as a
plain value; callers `match` it with value rules. `all_failed`, `standard_failure`-as-leaf,
and Q6 all disappear. Cost: the failure array is no longer an `emits` member,
so functions must map it into a declared error explicitly — arguably more
honest, since "every replica failed" is a domain event the author should name.

## R8 — `emits` verbosity: full transitive lists, by hand

Every declaration repeats its complete error bound, including intrinsic
transport/codec errors. Wrappers that only forward (R6) restate 9–15 entries.

### Current

```text
fn void main
    emits [http::invalid_request, http::credentials_missing, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data, llm::refused, llm::truncated, llm::invalid_response, ai::invalid_question, ai::invalid_answer, sql::connection_failed, sql::close_failed, io::write_failed]
```

(From `examples/native-ai/src/app/main.can`, line 7 — one 15-entry line.)

### Suggestion A — named error sets

```text
errorset transport
    http::invalid_request
    http::credentials_missing
    http::transport_failed
    http::timeout
    http::body_limit
    http::status_error
    codec::invalid_data

fn void main
    emits [transport, llm::refused, llm::truncated, llm::invalid_response, ...]
```

An `errorset` is a named, immutable list; `emits` entries may name sets or
single errors, flattened by the checker. Callers still handle every leaf —
nothing about explicitness changes — but declarations state intent once.
Sets could alternatively be ordinary variants, reusing R7's closed-variant
idea instead of a new declaration kind.

### Suggestion B — forwarding shorthand for pure forwarders

A function whose body only forwards `draft`'s outcomes writes:

```text
fn records::triage_plan draft_account
    emits [=draft]
    ...
    relay call draft((account))
```

`emits [=name]` means "exactly the callee's bound, checked at compile time."
Narrower than A (only helps wrappers), but wrappers are where the pain
concentrates, and it keeps every other `emits` fully spelled out.

## R9 — Fixture repetition: `when` rows restate every call

`when` rows repeat full expected arguments at every lexical call site, once
per assertion name — including cross-assertion rows leaking into helpers.

### Current

(`examples/native-ai/src/model/model.can` — one of four near-identical blocks:)

```text
match call sql::query_rows<records::search_parameters, records::account_row>(pool, "recent_accounts", records::search_parameters("%"), 25)
    when
        sample: pool, "recent_accounts", records::search_parameters("%"), 25 => ok [records::account_row(1, "Ann"), records::account_row(2, "Bo")]
        down: pool, "recent_accounts", records::search_parameters("%"), 25 => sql::connection_failed("recent_accounts")
        triage: pool, "recent_accounts", records::search_parameters("%"), 25 => ok [records::account_row(1, "Ann"), records::account_row(2, "Bo")]
        unclosed: pool, "recent_accounts", records::search_parameters("%"), 25 => ok [records::account_row(1, "Ann"), records::account_row(2, "Bo")]
```

The `triage:`/`unclosed:` rows belong to *other functions'* assertions but must
be spelled at this call site because fixtures are lexical. Change the query
arguments and every row must be edited in lockstep.

### Suggestion A — assertion-level fixtures

Move substitution from the call site to the asserting function: each assertion
names the (function, call-ordinal) pairs it overrides, with arguments matched
by position:

```text
asserts
    sample: => ok records::count_value(2, true)
fixtures
    sample: model::recent_count#0 => ok [records::account_row(1, "Ann")]
```

Call sites stay clean; the P3 identity machinery (lexical ordinal + occurrence
+ participant path) is unchanged underneath, but authors write one fixture per
assertion instead of one row per call site per assertion. Argument checking
can stay (compare declared expected args) or be dropped in favor of
position-only matching — a smaller follow-up decision.

### Suggestion B — named fixture sets

Keep lexical `when` tables, but let a set of rows be declared once and
referenced by name:

```text
fixtures db_up
    sample: pool, "recent_accounts", records::search_parameters("%"), 25 => ok [...]

match call sql::query_rows<...>(...)
    when
        use db_up
        down: pool, "recent_accounts", records::search_parameters("%"), 25 => sql::connection_failed("recent_accounts")
```

Less restructuring than A; kills the copy-paste across the four call sites
while keeping fixtures next to the calls they substitute.

## R10 — Receiver vs `near` vs args: three capture mechanisms

`on` receivers (dot-calls + `callable panel.area` capture), `near` inputs
(exact-name capture at `callable` creation), and ordinary arguments overlap:
a receiver is a captured value with call-site sugar.

### Current

(`compiler/testdata/current/callables/captures.can` — a method whose receiver
and `near` input do the same job shape-wise:)

```text
fn int bump
    on box self
    emits []
    given
        near int increment
    asserts
        sample: box(7) => 3 => ok 10
    ok self.value + increment
```

Note the method-assertion double arrow (`box(7) => 3 => ok 10`) — a fourth
assertion layout alongside plain, zero-input, and native-`when` rows.

### Suggestion A — receivers are sugar for a first `near` input

```text
fn int bump
    emits []
    given
        near box self
        int increment
    asserts
        sample: box(7), 3 => ok 10
    ok self.value + increment
```

`call panel.bump(3)` desugars to `call bump(panel, 3)`; `callable panel.bump`
desugars to capturing `self`. One capture rule, one assertion layout, and the
`on` line plus method/assertion special cases disappear. Dot-call remains as
pure call-site sugar resolved by "first `near` input type."

### Suggestion B — keep `on`, delete `near`

Invert it: methods become the only capture mechanism, and today's `near`
functions are rewritten as single-receiver methods (possibly on a tiny
parameter record). Fewer concepts than today, but forces record-wrapping for
multi-value captures — likely worse than A; recorded for completeness.

## R11 — Smaller items (one sketch each)

**Error vs record layout.** Errors declare fields inline
(`error 1002 below_minimum(int actual, int minimum)`), records use indented
fields — for the same nominal-data concept (both are ordinary data per C4).
Suggestion: allow errors the indented record layout and delete the inline
parenthesized form (or vice versa); either way, one field layout.

**Bare-name patterns.** A bare record name narrows the scrutinee
(`circle => ok shape`), while a bare error name binds an alias *and* forwards
(`missing` vs `missing()` are context-sensitive per C3). Suggestion: all
bare-name patterns narrow the scrutinee; bind an alias only via an explicit
binder (`missing as m => ...`). Deletes the constructor-vs-reference
context sensitivity.

**Collection API homes.** Array ops are methods (`call plan.keywords.map(...)`
in `model.can`), map/set ops are namespaced functions
(`call collections::insert(empty, 3, "three")` in `std/map`), `append` is a
prelude function. Suggestion: pick one home — methods for all value-receiver
ops (`call empty.insert(3, "three")`), or namespaced functions for all —
and move the outliers.

**Top-level init subset.** C8 defines a bespoke inert-expression grammar plus
dependency ordering and cycle errors, yet every function may have effects
anyway. Suggestion: allow top-level bindings to run in source order with the
full expression grammar (calls included); a top-level failure reports before
`main`, exactly as C8 already specifies for primitive faults. One grammar
instead of two.

**String forms.** Four spellings (plain/triple/raw/raw-triple) with the
`r"before" + "\"" + r"after"` workaround for quotes in raw strings.
Suggestion: allow `\"` inside raw strings (the one exception), or always
prefer `r"""..."""` and drop single-line raw strings.

## What is not redundant

The `int`/`float` strictness, mandatory `emits` at API boundaries, exhaustive
matching, named-only functions, and the native-reuse rule are the thesis, not
duplication. The redundancy catalogued above is in *how many surface forms*
serve each thesis point. None of the suggestions remove explicitness from API
boundaries; they remove parallel machinery that states the same contract twice.

## Suggested sequence

1. R1 native-declaration unification (largest grammar + checker surface).
2. R4 completion-form unification + R8 `emits` shorthand (largest author
   verbosity win; R6's wrapper tax shrinks as a side effect).
3. R6 `given`/`state` unification (deletes the wrapper pattern entirely).
4. R9 fixture hoisting (largest test-authoring win).
5. R2, R7, R10, R11 as follow-ups, each independently shippable.

Each adopted step needs the AGENTS.md triple Jev consultation before the
decision is recorded.


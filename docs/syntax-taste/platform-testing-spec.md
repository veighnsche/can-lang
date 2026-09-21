# Platform and deterministic testing specification

Status: technical specification for the approved Can surface and Bun boundary.
This document does not approve additional syntax. It fixes compiler, runtime,
catalogue, and test-harness behavior needed to make the approved surface usable.

The normative words **must**, **must not**, **should**, and **may** have their
usual specification meanings. Contract signatures below use mathematical
notation to avoid inventing Can declaration syntax. `T`, `P`, `R`, and `E` are
type variables in that notation; they do not approve implicit generics or a new
source spelling.

## P1. Authority, scope, and classifications

### Policy

The approved decisions remain authoritative for source syntax. In particular:

- all authored and intrinsic functions have an explicit `emits` contract;
- all Can functions execute asynchronously underneath;
- every function may have effects, with no purity or effect checking;
- the compiler emits equivalent JavaScript or Bun operations and adds only the
  adapters needed for Can types, immutability, errors, and resources;
- only the maintained distribution extends the platform catalogue;
- applications cannot embed JavaScript or TypeScript, import arbitrary backend
  modules, register adapters, execute commands, or access the Bun namespace;
- the browser runs upstream HTMX while application Can runs on Bun; and
- LLM tools are outside this specification.

The approved `http::header`, `http::response<T>`, and `bytes::buffer` contracts
are shared with the native fetch and codec specification:

```text
http::header = { str name, str value }
http::response<T> = { int status, http::header[] headers, T body }
bytes::buffer = catalogue-owned opaque immutable bytes
```

Header names are lowercase and records are ordered by name lexicographically.
Values that native `Headers` combines remain one ordinary header record;
`set-cookie` values remain separate records in native `getSetCookie()` order.
This is a normalized snapshot, not physical wire-line preservation.
This specification uses the already allocated errors 1100--1105 and 1110 and
does not redeclare them.

### Technical contract

Every requirement is classified as one of:

- **surface**: would require user approval because it changes Can syntax;
- **technical**: compiler, generated-code, runtime, or catalogue behavior fixed
  here without changing syntax;
- **evidence**: a test or conformance obligation; or
- **deferred**: capability intentionally absent from the initial slice.

All numbered P-sections are stable references. New revisions append or replace
subsections without renumbering unrelated sections.

## P2. Distribution catalogue identity and linkage

### Policy

Platform access is available only through compiler-owned packages. A project
package can use these packages through ordinary `uses` and aliases. An alias
changes the local spelling, not the linked catalogue identity.

### Technical contract

The distribution manifest assigns each package a stable identity of the form
`can.std.<package>@<catalogue-revision>`. The initial reserved source package
names are:

```text
ai asset bytes cli clock codec collections crypto env html htmx http io json llm
log number option random sql text
```

Project packages must not claim a reserved source name. The compiler resolves a
source qualifier to the manifest identity before type checking. Generated
TypeScript imports compiler-owned modules by manifest identity, never by a
project-supplied path. A build records the compiler version, Bun version,
catalogue revision, and embedded HTMX asset digest.

Each catalogue operation has:

1. one fully qualified identity;
2. a Can input, success, and complete domain-error contract;
3. one Bun emission recipe;
4. a substitution recipe for assertions; and
5. target conformance cases against the supported Bun version.

The compiler rejects an unknown catalogue revision or an operation whose
emission recipe is unavailable for the selected target. Catalogue identity is
nominal: a project record with identical fields is not a catalogue-owned opaque
type.

No package above grants command execution, dynamic imports, source evaluation,
raw HTML construction, unsafe SQL, or arbitrary file-descriptor access.

The project root contains one inert `can.project.json` object with exactly these
top-level fields: `source_root` (required relative directory), `dependencies`
(optional object from dependency name to relative directory), `assets`
(optional object from asset name to relative file), `sql` (optional object
defined in P12), and `error_registry` (required relative C9 registry file).
Duplicate or unknown fields fail validation. Recursively, every directory at or
beneath `source_root` that contains Can source is one flat-named package;
filesystem nesting does not create a package hierarchy. Each dependency
directory contains its own validated manifest and source root. Package-name
collisions across the resolved graph are errors.

All manifest paths are UTF-8, relative to the manifest directory, normalized
without `..`, and checked after symlink resolution to remain beneath that
directory. Dependency resolution reads only these explicit local mappings; it
does not search parent directories, infer registry names, or access a network.
The manifest cannot declare scripts, hooks, generated adapters, backend imports,
commands, or environment expansion. Asset and SQL entries are data consumed by
the compiler-owned catalogues below.

Dependency-manifest paths form a directed graph. A dependency identity is its
key in `dependencies`, using the ordinary Can identifier grammar. Keys are
global across the resolved graph: repeated use of a key must resolve to the
same real directory, and two different keys cannot resolve to one directory.
A cycle, a missing lock entry, or an unused lock entry is an error. A repeated
dependency is checked once. This configuration identity does not rename any
source package.

The C9 error registry has exactly two fields, illustrated by this complete
registry object:

```json
{
  "active": [{ "id": 1000000, "kind": "billing::declined" }],
  "retired": [1000001]
}
```

Each active entry has exactly `id` and `kind`. `kind` is the fully qualified
source package and error declaration name, without type arguments: a generic
error contributes one entry. `retired` contains integer IDs only. Both arrays
are strictly increasing by ID; active kinds are unique; active and retired IDs
are disjoint. Project and dependency entries use C9's application range
1000000--2147483647. The distribution owns a separate registry in its reserved
range. IDs are JSON integer tokens, without fraction or exponent. Empty arrays
are valid. Registry JSON rejects duplicate keys and unknown fields.

`can.lock.json` has exactly one field, `dependencies`, whose value is an object
keyed by those graph identities. Each value has exactly `path`,
`manifest_sha256`, `source_sha256`, and `error_registry`. `path` is the
dependency's normalized directory relative to the root project manifest;
digests are lowercase 64-character hexadecimal strings; `error_registry` is
an embedded registry object of the exact shape above. Object field order and
insignificant JSON whitespace do not affect registry equality. Lock JSON
rejects duplicate keys and unknown fields. A dependency-free project may omit
the lock or use `{"dependencies": {}}`.

`manifest_sha256` hashes the dependency manifest's exact UTF-8 file bytes.
`source_sha256` hashes the bytes `can-source-tree-v1` followed by one zero byte,
then every `.can` file beneath the dependency's `source_root` in ascending
UTF-8 byte order of its normalized, slash-separated relative path. For each
file, append its path byte length as an unsigned 64-bit big-endian integer,
its UTF-8 path bytes, its content byte length in the same format, and its exact
content bytes. Symlink resolution must obey the same confinement rule as other
manifest paths. Referenced assets separately receive P11's build digests.

Before linking, each project's active registry must exactly equal its
source-declared ID-to-kind pairs, excluding dependency and catalogue
declarations; every retired ID must be absent from its source. Each dependency's
manifest-referenced registry must also equal the lock's embedded snapshot. The
compiler recomputes both digests and rejects any mismatch. It then checks the
root registry, verified dependency snapshots, and distribution registry as one
graph: no ID may be active or retired in more than one registry. These checks
make a changed standalone registry detectable even though it is not part of
the source digest. A build never allocates IDs or rewrites a registry or lock.

## P3. Assertion execution model

### Policy

Assertions execute ordinary Can code. A `when` row replaces only the matched
invocation. It does not turn the entire assertion into a scripted success.
Assertions never make live network, database, filesystem, clock, randomness,
or other platform calls.

### Technical contract

The test compiler creates an **assertion run** for one named assertion. An
assertion run has an immutable root identity:

```text
(package identity, declaration identity, assertion name)
```

An external test-plan or CLI selector supplies all three components. A
short assertion name is accepted only when it resolves to one root in the
selected package; collisions require the full package and declaration
identity. The short selector written in a source `when` row remains the local
assertion name and is resolved against that already selected root.

Every substitutable call receives an **invocation identity**. It is a path from
that root. Each path segment contains:

```text
(lexical call identity, occurrence, participant path, callable instance)
```

- The lexical call identity is the containing declaration identity plus the
  call node's preorder ordinal in the typed syntax tree. Formatting does not
  change it.
- The occurrence is a zero-based count scoped to the parent invocation and
  lexical call identity.
- The participant path is empty for sequential code. A coordination entry adds
  its written index; a spread adds its collection index after that entry index.
- A callable instance is its creation site's lexical identity and occurrence.
  Its receiver and `near` captures are frozen at creation and recorded in the
  invocation diagnostic fingerprint.

The compiler reserves identities before starting concurrent participants.
Direct entries reserve in written order and spread entries reserve in collection
order. Runtime completion order cannot change an identity.

The text before `:` in a `when` row is an assertion selector. Assertion names
are scoped by their declaring function; they are not globally unique. For one
compiled root assertion, the harness activates rows at dynamically entered call
sites whose selector text equals that root assertion's name. A helper row with
the same selector may intentionally serve multiple reachable roots; each root
gets an isolated queue, full root identity, and independent type checking.

A lexical table may repeat the same selector. The selected rows form one
ordered queue per `(root assertion identity, lexical when-table identity)`,
shared by recursive and concurrent visits to that lexical site. The canonical
scheduler assigns the next row to the next reserved full invocation identity;
it never searches later rows for matching arguments. It then validates that
row's expected explicit arguments against the actual typed arguments. Receiver
and `near` captures are compared with the frozen callable-instance fingerprint.
They do not require new row syntax.

```text
when
    eventually: "42" => unavailable()
    eventually: "42" => ok profile("Sam")
```

The two visits under root assertion `eventually` receive these outcomes in
reserved invocation order. A first-visit argument other than `"42"` is an
argument mismatch; the harness does not skip forward to the success row.

Expected arguments use the matched declaration's existing invocation grammar.
A judge or LLM row therefore writes ordinary `given` arguments followed by its
mandatory final state group, for example
`routed: question, options, (email) => ok suggestion(...)`. The parentheses
group only the state inputs; they do not enclose the whole row argument list or
create a tuple. Native method receivers and `near` captures remain implicit in
the row and are checked through the invocation fingerprint above.

Sequential calls reserve when reached. Recursive descendants append their full
parent path. After coordination prepare, the scheduler advances every launched
participant to its next harness await without releasing a participant fixture,
collects pending fixture requests, and assigns rows in full invocation-path
order. It repeats this barrier after each release. Host microtask timing cannot
allocate a shared helper queue.

This rule covers:

- repeated calls at one lexical site;
- recursive and mutually recursive transitive calls;
- a wrapper invoked through multiple callable values;
- method-like calls whose receiver differs;
- callable values whose `near` captures differ; and
- concurrent participants that reach the same helper and lexical call site.

An assertion run fails with the complete invocation path when any of the
following occurs:

- **missing fixture**: a substitutable boundary invocation has no matching row;
- **argument mismatch**: the next row assigned to the invocation identity has
  expected arguments different from the actual typed arguments;
- **ambiguous fixture**: more than one non-ordered fixture source claims the
  same invocation identity;
- **malformed fixture**: the supplied success or error is outside the call's
  declared completion contract, including an invalid opaque value;
- **unexpected live boundary**: generated test code attempts a real platform
  operation; or
- **unused fixture**: a row remains unused after the root completion settles.

Argument comparison uses the core equality contract. Opaque fixture handles
compare compiler-issued harness-token identity only; their native object and
contents remain unobservable. Missing, mismatched, ambiguous, unused, and
unexpected-live-call cases produce the core `assertion` standard failure.
A statically malformed `when` row is a compile error. A malformed external raw
fixture is reported by the adapter's declared validation error or as a harness
configuration failure before the run, according to which side is malformed.

Unused-row checking happens after all started coordination participants have
settled in the deterministic harness. It does not wait for live operations.

### Evidence

Compiler tests must cover two identical calls, unequal repeated arguments,
recursion, a transitive dependency, two callable instances from the same
creation site, different receivers, different `near` captures, a spread of
callables, and two concurrent participants reaching the same helper call site.
Each negative case must print the expected and actual invocation paths.

These FIFO cases exercise the one canonical assertion scheduler. They do not
establish production settlement behavior. Separate Bun-native conformance tests
must use controlled promises to force each two-participant settlement order and
representative larger and nested interleavings, then check the approved native
coordination result. That controlled suite is target evidence for the emitted
operations; it does not claim that an assertion fixture schedule reproduces
uncontrolled production timing.

## P4. Real execution, supplied completions, and provider fixtures

### Policy

Test evidence states which boundary was exercised. A supplied result is valid
consumer-unit evidence. It is not provider, serialization, transport, HTML,
SQL-binding, or target-runtime evidence.

### Technical contract

The test report assigns every assertion one or more evidence labels:

| Label | What executes | What it establishes |
| --- | --- | --- |
| `real-can` | Ordinary Can bodies and native-equivalent pure operations | Can logic and compiler lowering |
| `supplied-completion` | A `when` result replaces the target invocation | Consumer handling of that declared completion |
| `raw-provider-fixture` | Raw request/response or driver data crosses the real compiler-owned adapter | Encoding, decoding, validation, and error mapping |
| `bun-conformance` | Generated TypeScript runs against the pinned Bun API | Target integration and lifecycle behavior |
| `live-quality` | Explicit external quality/evaluation job | External service quality; outside ordinary assertions |

Ordinary local functions and deterministic catalogue operations execute their
real bodies unless their exact call has a `when` row. Side-effecting or
nondeterministic platform calls and native provider declarations require a
fixture in ordinary assertions; this includes HTTP, SQL, I/O, clock, random,
environment, and log calls, while HTML encoding, codecs, hashing, and collection
operations run for real. A fixture result is delivered through the
same completion adapter used after a real call, but it starts after native I/O:
it must not claim coverage of request construction or raw response parsing.

Raw provider fixtures are compiler conformance data, not Can syntax. A fixture
contains the raw boundary input and raw boundary output or driver failure. The
real adapter must serialize the input, consume the fixture, validate it, and
produce the Can completion. HTTP fixtures use raw method, URL, headers, and
bytes. SQL fixtures use statement text, ordered native parameter values,
driver rows, affected-row metadata, and phase-tagged failures.

An opaque catalogue value in a supplied completion must carry a valid
compiler-issued fixture representation. A record with the same visible data
does not forge `html::safe`, `sql::pool`, `sql::transaction`, `http::request`,
or another opaque type.

### Evidence

Release reports must keep the five labels separate. A capability is marked
supported only when its required labels are green; a consumer assertion alone
cannot graduate a provider or platform operation.

## P5. Deterministic coordination harness

### Policy

`concurrent`, `concurrent with error`, `race`, and `race with error` retain their
approved native meanings. The assertion harness controls settlement without
adding Can scheduling syntax and without making live calls.

### Technical contract

The coordination prepare phase first evaluates callable references, receivers,
arguments, and spread arrays left to right. Nested argument calls complete as
ordinary sequential calls, including sequential fixture delivery. A domain-
fallible nested call cannot appear as an unchecked argument under the core call
rules. Prepare failure launches no participant. After spread evaluation the
harness flattens and reserves all participant positions in direct-entry and
collection order.

All participant bodies then start before any participant-body fixture settles.
A supplied completion is represented internally by a deferred promise. The
harness releases ready fixture events according to their reserved invocation
identities and the barriers in P3.

The authored-assertion schedule is canonical:

1. direct coordination entries in written order;
2. spread entries in collection order;
3. nested ready calls by lexicographic invocation path; and
4. a participant's next event only after its preceding awaited event settles.

This produces deterministic `Promise.all`, `Promise.allSettled`, `Promise.any`,
and `Promise.race` observations while still starting every participant before
settlement. `race` selects the first scheduled success. If every participant
fails, its single `all_failed` value contains original failures in input order,
independent of settlement order. Remaining race participants settle under the
harness so missing and unused fixtures are still diagnosed.

Alternate schedules are an internal conformance-harness input. The compiler
suite must run every feasible ordering for two participants and representative
orderings for larger and nested sets. Ordinary authored assertions use the
canonical schedule in this initial slice. Authors can test another winner by
reordering participants in a separate function/assertion. This is an explicit
ergonomic limitation and does not justify timing or scheduling syntax.

Native conformance separately proves that production coordination emits
`Promise.all`, `Promise.allSettled`, `Promise.any`, or `Promise.race` as
approved. The deterministic harness does not replace those production calls.

## P6. Opaque values, callbacks, and resource enforcement

### Policy

Trusted values and host handles use catalogue-owned opaque types. This does not
add a general user-defined opaque-type facility, ownership syntax, affine types,
effect proofs, or purity requirements.

### Technical contract

Opaque values have no public positional constructor and no structural match.
Only catalogue constructors and compiler fixtures can create them. Getters
return immutable Can data. Generated TypeScript never exposes the native object
through a Can value.

Host callbacks are named Can callables with an exact input, success, and
`emits` contract. The adapter awaits the generated async function. A callback
cannot be retained after its owning operation closes. Invocation after closure
fails before user code runs.

The runtime keeps a per-process resource registry. Each resource has a nominal
kind, unique ID, owning scope, state `open`, `closing`, or `closed`, and a lease
count keyed by coordination owner. Catalogue operations validate kind, scope,
and state and acquire a lease before touching a native handle; settlement
releases it. Closing marks the handle `closing`; an owner that already holds a
lease may acquire the subleases needed to finish, while a new owner is denied.
Close waits for every existing owner lease. It never invalidates a live lease.
Closing is idempotent only where the signature explicitly says so; otherwise a
second close produces the core `resource_state` standard failure.

For scoped handles, including `sql::transaction`, settlement of the authored
callback starts closing: no new owner may enter, existing owners retain the
subleases needed to finish, and the enclosing native callback stays pending
until they drain. The handle becomes closed only after that drain and its
native commit, rollback, or release. Returning, recording, or capturing such a
handle is not given static ownership meaning in this slice. A use outside its
owning scope or after closure is rejected by the runtime. The
compiler should diagnose obvious direct escapes as a quality improvement, but
program correctness does not depend on that optional diagnostic.

A stale, wrong-kind, or out-of-scope handle produces the core `resource_state`
standard failure. Failure during automatic reverse-order cleanup produces the
core `cleanup` standard failure. These cases are outside `emits`; expected host
failures from an explicit stop or close use the declared domain errors below.

At `main` completion the root supervisor first drains every Q coordination
owner, including early-race losers, and only then closes unleased resources in
reverse creation order. A catalogue close deadline bounds only that close
caller's wait. Expiry returns its declared close error and leaves the resource
closing, owned, and observed; it does not permit closing beneath a live lease
or make the root drain complete. The root supervisor continues observing the
owner until settlement, so a nonsettling owner can keep the Can process pending
indefinitely. Bounded host termination is an external process-supervisor policy,
not a runtime hard kill. Such external termination makes no Can claim that
effects were rolled back, cleanup completed, or a runtime exit status was
reached. An open resource found because application code omitted its close is
still a cleanup failure when the runtime can close it after leases drain.

A Bun HTTP server remains a referenced native resource, so a service does not
exit merely because its startup function completed. `http::server_wait` is the
normal long-lived-main operation.

Bare `ok` from `main` exits zero after every coordination owner drains and every
resource closes cleanly. A losing or late coordination diagnostic alone does
not change that successful status. A terminal declared domain error from
`main` prints its stable ID and sanitized payload through the runtime
diagnostic path and exits nonzero. An uncaught standard failure follows the
core root-failure rule. Resource leak, cleanup, and shutdown-deadline failures
produce a nonzero exit when the process reaches a runtime-controlled exit, even
when the authored `main` result was `ok`.

At an HTTP callback boundary:

- declared domain errors have already been mapped by the mounted callback,
  whose type is `callable http::server_response (http::request) emits []`;
- an unhandled standard failure is logged with its stable diagnostic identity
  and becomes a fixed sanitized 500 response;
- the failure text and stack are never placed in the response; and
- failures in the boundary error renderer fall back to an embedded constant
  plain-text 500 response.

## P7. Platform domain errors

### Policy

IDs 1100--1199 remain owned by the shared AI, HTTP client, and codec contract.
This specification allocates platform additions from 1200--1299.

### Technical contract

```text
error 1210 io::read_failed(str operation)
error 1211 io::write_failed(str operation)
error 1212 io::limit_exceeded(int limit)

error 1220 html::invalid_structure(str reason)
error 1221 html::invalid_url(str reason)
error 1222 htmx::invalid_target(str reason)
error 1223 htmx::invalid_interval(int milliseconds)

error 1230 http::invalid_route(str reason)
error 1231 http::duplicate_route(str method, str path)
error 1232 http::ambiguous_route(str first, str second)
error 1233 http::invalid_server_config(str reason)
error 1234 http::bind_failed(str address)
error 1235 http::shutdown_failed(str phase)

error 1240 sql::connection_failed(str phase)
error 1241 sql::query_failed(str operation, str code)
error 1242 sql::row_missing(str query)
error 1243 sql::row_count(str query, int actual)
error 1244 sql::schema_mismatch(str path, str reason)
error 1245 sql::constraint_failed(str constraint)
error 1246 sql::transaction_failed(str phase)
error 1247 sql::commit_unknown(str transaction_id)
error 1248 sql::close_failed(str reason)
error 1249 sql::row_limit(int limit)
error 1250 sql::unsupported_value(str path, str reason)

error 1260 clock::invalid_duration(int milliseconds)
error 1261 random::invalid_length(int length)
error 1262 env::invalid_name(str name)
error 1263 log::write_failed(str level)
```

This block is registry notation: declarations live in their named packages and
use unqualified error names in source. These payloads are stable, sanitized Can
data. A SQL error payload's `query` is its stable compiler query identity, and a
transaction ID is a generated occurrence identity. Native messages, SQL text,
credentials, filesystem paths, and stacks stay in structured diagnostics and
logs. Intrinsic catalogue declarations list every applicable error explicitly;
the tables below abbreviate repeated lists but do not authorize hidden domain
errors.

P extends shared `http::invalid_request` reason vocabulary only with
`query_missing`, `query_repeated`, `invalid_status`, `invalid_header`,
`unsupported_media_type`, `form_missing`, `form_repeated`, and
`invalid_form_encoding`. It does not put names, values, bodies, or native error
messages in that payload. Codec paths and the P-specific errors carry the
remaining structured detail.

## P8. Minimal CLI and I/O catalogue

### Policy

The `main` function's existing `str[]` input is the command-line argument list.
No separate process or shell interface is provided.

### Technical contract

The input may use any valid source name and contains only application arguments.
It excludes the Bun executable, generated program path, and Bun runtime flags.

| Operation | Contract | Bun emission |
| --- | --- | --- |
| `io::stdin_bytes` | `(int max_bytes) -> bytes::buffer emits [io::limit_exceeded, io::read_failed]` | bounded read from `Bun.stdin` |
| `io::stdin_text` | `(int max_bytes) -> str emits [io::limit_exceeded, io::read_failed, codec::invalid_data]` | bounded bytes plus UTF-8 decoder |
| `io::stdout_write` | `(bytes::buffer) -> int emits [io::write_failed]` | `Bun.write(Bun.stdout, bytes)` |
| `io::stderr_write` | `(bytes::buffer) -> int emits [io::write_failed]` | `Bun.write(Bun.stderr, bytes)` |

Runtime file paths, directory traversal, deletion, watching, file descriptors,
subprocesses, and shell execution are deferred. Static web assets use the build
asset manifest in P11.

All I/O calls are asynchronous Can calls even where a particular Bun operation
can complete synchronously. `bytes::buffer` preserves immutability by copying a
native view before it crosses into Can or by proving exclusive ownership of a
fresh native buffer.

`max_bytes` is an exact nonnegative integer; a negative argument yields
`io::limit_exceeded(max_bytes)` before reading. Zero accepts only empty input.
Read incrementally and reject before retaining bytes beyond the caller's budget;
cancel the reader on overflow. No additional fixed byte ceiling is implied.
Unexpected native allocation defects remain standard failures. Text decoding is
fatal UTF-8 and preserves a leading BOM as text. Expected native I/O errors map
to the declared read/write failures; unrelated runtime defects do not.

### Finite host utilities

| Operation | Contract and native emission |
| --- | --- |
| `clock::wall_millis` | `() -> int emits []`; `BigInt(Date.now())` |
| `clock::monotonic_millis` | `() -> float emits []`; native `performance.now()` |
| `clock::sleep_millis` | `(int milliseconds) -> void emits [clock::invalid_duration]`; 0--2147483647 then `await Bun.sleep(Number(milliseconds))` |
| `random::secure_bytes` | `(int length) -> bytes::buffer emits [random::invalid_length]`; 0--65536 then native `crypto.getRandomValues` on a fresh `Uint8Array` |
| `random::uuid_v4` | `() -> str emits []`; native `crypto.randomUUID()` |
| `crypto::sha256` | `(bytes::buffer) -> bytes::buffer emits []`; fresh `Bun.CryptoHasher("sha256")`, `update`, `digest` |
| `env::required` | `(str name) -> str emits [env::invalid_name, http::credentials_missing]`; exact lookup in `Bun.env` |
| `env::optional` | `(str name) -> option::value<str> emits [env::invalid_name]`; exact lookup in `Bun.env` |
| `log::write_info` | `(str message) -> void emits [log::write_failed]`; one JSON line through `Bun.write(Bun.stderr, ...)` |
| `log::write_error` | `(str message) -> void emits [log::write_failed]`; same with fixed `error` level |

Environment names match `[A-Z_][A-Z0-9_]*`; the catalogue exposes no enumeration
or mutation. Optional lookup returns `option::none()` only for absence and
`option::some<str>(value)` for a present entry, including an empty string. Required
lookup likewise returns an empty present value; only absence yields
`http::credentials_missing(name)`. These operations read the launcher snapshot of
the caller environment, not the driver's rewritten runtime environment. Log JSON has exactly string fields `level` and `message`, uses
native `JSON.stringify`, and never invokes value inspection. Clock, random,
environment, sleep, and log operations require deterministic assertion
fixtures. SHA-256 is deterministic and executes in ordinary assertions.

## P9. Safe HTML and HTMX constructors

### Policy

HTML sinks accept only catalogue-owned trusted values. Ordinary `str` is always
text or validated attribute data. There is no raw-HTML escape, inline event
handler, inline script, or user JavaScript adapter.

### Technical contract

The initial opaque types are `html::node`, `html::safe`, `html::url`,
`html::attribute`, `html::tag`, `htmx::target`, and `htmx::swap`.

| Operation | Contract |
| --- | --- |
| `html::make_tag` | `(str) -> html::tag emits [html::invalid_structure]`; accepts only the closed author-tag inventory below |
| `html::text` | `(str) -> html::node emits []`; escapes for text context |
| `html::parse_url` | `(str) -> html::url emits [html::invalid_url]`; accepts same-origin relative HTTP paths and approved `https` URLs |
| `html::text_attribute` | `(str name, str value) -> html::attribute emits [html::invalid_structure]`; permits only the text-attribute inventory below |
| `html::url_attribute` | `(str name, html::url) -> html::attribute emits [html::invalid_structure]`; permits the tag-checked URL-attribute inventory |
| `html::element` | `(html::tag, html::attribute[], html::node[]) -> html::node emits [html::invalid_structure]` |
| `html::fragment` | `(html::node[]) -> html::safe emits [html::invalid_structure]` |
| `html::text_fragment` | `(str) -> html::safe emits []`; one escaped text node for fallback responses |
| `html::stylesheet` | `(html::url) -> html::node emits [html::invalid_url]`; one head-only stylesheet link |
| `html::meta_viewport` | `() -> html::node emits []`; one fixed head-only viewport element |
| `html::document` | `(str title, html::node[] head, html::node[] body) -> html::safe emits [html::invalid_structure]` |
| `htmx::get` | `(html::url) -> html::attribute emits [html::invalid_url]`; URL must be same-origin relative |
| `htmx::post` | `(html::url) -> html::attribute emits [html::invalid_url]`; URL must be same-origin relative |
| `htmx::target_id` | `(str id) -> htmx::target emits [htmx::invalid_target]` |
| `htmx::target_attribute` | `(htmx::target) -> html::attribute emits []` |
| `htmx::swap_inner` | `() -> html::attribute emits []` |
| `htmx::swap_outer` | `() -> html::attribute emits []` |
| `htmx::trigger_change` | `() -> html::attribute emits []` |
| `htmx::trigger_input_changed` | `(int delay_ms) -> html::attribute emits [htmx::invalid_interval]`; emits `input changed delay:<n>ms` |
| `htmx::trigger_every` | `(int interval_ms) -> html::attribute emits [htmx::invalid_interval]`; emits `every <n>ms` |
| `htmx::indicator_id` | `(str id) -> html::attribute emits [htmx::invalid_target]` |
| `htmx::disable_this` | `() -> html::attribute emits []`; emits `hx-disabled-elt="this"` |
| `htmx::runtime_head` | `() -> html::node emits []`; emits the pinned local script and response policy |

The closed author-tag inventory is `main`, `header`, `footer`, `nav`, `section`,
`article`, `aside`, `h1`--`h6`,
`p`, `div`, `span`, `ul`, `ol`, `li`, `a`, `form`, `label`, `input`, `textarea`,
`select`, `option`, `button`, `table`, `thead`, `tbody`, `tr`, `th`, `td`, `dl`,
`dt`, `dd`, `strong`, `em`, `small`, `br`, and `hr`. `input`, `br`, and `hr`
reject children. `ul`/`ol` admit only `li`; `select` only
`option`; `thead`/`tbody` only `tr`; `tr` only `th`/`td`; and `table` only an
optional `thead` followed by one `tbody`. `dl` admits alternating `dt`, `dd`
pairs. Nested `a` and nested `form` fail. Other author tags accept any author
node. `html::document` alone emits `html`, `head`, `title`, and `body`. Its head
input admits only `html::stylesheet`, `html::meta_viewport`, and the special
`htmx::runtime_head` node; its body input rejects head-only nodes.

The exact text-attribute inventory is global `id`, `class`, `title`, `lang`,
`dir`, `hidden`, `tabindex`, and `role`; `aria-*` names; and tag-checked `name`,
`value`, `type`, `placeholder`, `autocomplete`, `for`, `method`, `rel`, `checked`,
`selected`, `disabled`, `required`, `multiple`, `rows`, `cols`, `scope`,
`colspan`, and `rowspan`. Enumerated attributes accept only their HTML tokens.
The URL-attribute inventory is `href` on `a`, `action` on `form`, and
`formaction` on `button`. Duplicate names after ASCII-case normalization fail.

`html::text` and every attribute serializer pass the already typed `str` to
native `Bun.escapeHTML`; no adapter implements a second escaping algorithm.
`html::parse_url` uses the native `URL` parser, rejects controls, backslashes,
credentials, scheme-relative URLs, and schemes other than `https`, and records
whether the value was a single-slash same-origin path. `html::document` emits
one doctype, `html`, `head`, and `body` structure. Text and attribute encoding
remain distinct trusted construction contexts.

The text-attribute constructor excludes `href`, `src`, `action`, `formaction`,
`style`, `srcdoc`, event names, and every other URL, CSS, markup, or script
context. URL-bearing attributes require `html::url_attribute`. The initial tag
inventory excludes `script`, `style`, `iframe`, `object`, `embed`, and raw-text
elements; only `htmx::runtime_head` can produce its pinned script node.
`html::safe` is an immutable rendered document or fragment, not a mutable tree
or a claim that arbitrary bytes are safe.

The initial HTMX attribute inventory is `hx-get`, `hx-post`, `hx-target`,
`hx-swap` (`innerHTML` and `outerHTML`), typed change/input/poll triggers,
`hx-indicator`, and `hx-disabled-elt=this`. Input delays are 0--60000 ms and
poll intervals are 1000--3600000 ms. Catalogue constructors produce every
value; `html::text_attribute` rejects raw `hx-*` and `data-hx-*` names. `hx-on*`,
arbitrary selectors, arbitrary triggers, `hx-vals`, extensions, and script
attributes are deferred.

## P10. HTTP request, response, routing, and server catalogue

### Policy

The server boundary is an immutable request to one complete immutable response.
Routes store named typed callbacks. Application domain errors are exhaustively
mapped before a callback is mounted.

### Technical contract

Opaque server types are `http::request`, `http::server_response`, `http::status`,
`http::body_status`, `http::server_headers`, `http::route`, `http::router`,
`http::server`, and `http::server_config`. The exact mounted callback type is:

```text
callable http::server_response (http::request) emits []
```

| Operation | Contract |
| --- | --- |
| `http::request_method` | `(http::request) -> str emits []` |
| `http::request_path` | `(http::request) -> str emits []`; decoded normalized path |
| `http::query_one` | `(http::request, str name) -> str emits [http::invalid_request]`; missing or repeated is an error |
| `http::query_all` | `(http::request, str name) -> str[] emits [http::invalid_request]` |
| `http::request_headers` | `(http::request) -> http::header[] emits []` |
| `http::request_body` | `(http::request, int max_bytes) -> bytes::buffer emits [http::body_limit]` |
| `http::request_json<T>` | `(http::request, int max_bytes) -> T emits [http::body_limit, http::invalid_request, codec::invalid_data]` |
| `http::request_form<T>` | `(http::request, int max_bytes) -> T emits [http::body_limit, http::invalid_request, codec::invalid_data]` |
| `http::make_status` | `(int) -> http::status emits [http::invalid_request]` |
| `http::make_body_status` | `(int) -> http::body_status emits [http::invalid_request]`; excludes 204, 205, 304 |
| `http::status_ok`, `http::status_unprocessable`, `http::status_internal`, `http::status_unavailable` | `() -> http::body_status emits []` |
| `http::make_server_headers` | `(http::header[]) -> http::server_headers emits [http::invalid_request]` |
| `http::empty_server_headers` | `() -> http::server_headers emits []` |
| `http::response_empty` | `(http::status, http::server_headers) -> http::server_response emits []` |
| `http::response_bytes` | `(http::body_status, http::server_headers, bytes::buffer) -> http::server_response emits []` |
| `http::response_text` | `(http::body_status, http::server_headers, str) -> http::server_response emits []` |
| `http::response_html` | `(http::body_status, http::server_headers, html::safe) -> http::server_response emits []` |
| `http::response_json<T>` | `(http::body_status, http::server_headers, T) -> http::server_response emits [codec::invalid_data]`; T must satisfy the shared wire subset |
| `http::route_get` | `(static str path, mounted_callback) -> http::route emits [http::invalid_route]` |
| `http::route_post` | `(static str path, mounted_callback) -> http::route emits [http::invalid_route]` |
| `http::make_router` | `(http::route[]) -> http::router emits [http::duplicate_route, http::ambiguous_route]` |
| `http::make_server_config` | `(str host, int port, int body_limit, int shutdown_ms) -> http::server_config emits [http::invalid_server_config]` |
| `http::server_start` | `(http::server_config, http::router) -> http::server emits [http::bind_failed]` |
| `http::server_wait` | `(http::server) -> void emits [http::shutdown_failed]`; stale handle is standard `resource_state` |
| `http::server_stop` | `(http::server) -> void emits [http::shutdown_failed]`; stale handle is standard `resource_state` |

The initial router accepts normalized exact static paths. Capture segments,
wildcards, middleware, cookies, streaming, WebSockets, TLS configuration, and
HTTP/2 policy are deferred. A missing route returns a fixed 404 response. A
known path with another method returns 405 and an `Allow` header. `GET` does not
implicitly register `HEAD`.

The adapter snapshots method, normalized URL, headers, and a body bounded by the
server configuration before calling Can. Oversize and failed native body reads
receive compiler-owned 413 and 400 responses before user callback entry. The
native `Request` is never exposed. Body reads are cached in the snapshot, so a
smaller per-call limit can emit `http::body_limit` and repeated catalogue
decoders otherwise observe the same immutable bytes.

`request_form<T>` accepts only `application/x-www-form-urlencoded` with UTF-8,
uses native `URLSearchParams` after rejecting malformed percent escapes, and
derives its closed schema from `T`. The initial field types are `str`,
`option::value<str>`, and `str[]`. Missing/repeated scalar fields use the finite
`form_missing` and `form_repeated` reasons. A malformed percent escape uses
`http::invalid_request("invalid_form_encoding")`; invalid UTF-8 uses
`codec::invalid_data(path, "utf8")`; and a value that does not fit the derived
shape uses the shared codec `type` reason. Multipart upload is deferred.

`http::server_start` emits `Bun.serve({ fetch: async (...) => ... })`. The fetch
callback awaits the named Can callable and converts the complete response to a
native `Response`. `http::server_stop` marks the resource closing and initiates
`server.stop(false)`, which stops accepting new connections and permits
in-flight requests to finish. It awaits both that native stop and all registered
request/coordination leases; existing request owners can finish their work.
When the configured deadline
expires, it returns `http::shutdown_failed` and leaves the closing resource
owned and observed; it does not call `server.stop(true)` beneath live leases.
That deadline bounds only the `server_stop` caller. It does not settle
`server_wait`, release request leases, or authorize the runtime to terminate
the process; the root may remain pending until every owner and native stop
settles. A deployment's external supervisor may impose a host deadline under
P6, without a Can cleanup or rollback guarantee.
`server_wait` listens to compiler-installed
SIGINT/SIGTERM handling and to an explicit `server_stop` of the same handle;
otherwise it remains pending. A signal performs the same graceful stop before
`server_wait` can return.

Response status must be 200--599. Status 204, 205, and 304 can be used only with
`response_empty`. The pinned Bun version accepts a body for these statuses, but
hosts and HTTP transports can discard or reinterpret it; the catalogue rule
keeps Can behavior independent of that divergence. Other statuses may also use
`response_empty`. Response headers are checked for valid names and values.
`make_server_headers` rejects
`content-type`, `content-length`, `x-content-type-options`, and the hop-by-hop
fields `connection`, `keep-alive`, `proxy-authenticate`, `proxy-authorization`,
`te`, `trailer`, `transfer-encoding`, and `upgrade`. `response_bytes` always
sets `application/octet-stream`; `response_text` always sets UTF-8 `text/plain`;
`response_html` always sets UTF-8 `text/html`; and `response_json` calls shared
`codec::encode_json<T>` and sets UTF-8 `application/json`. Every response sets
`X-Content-Type-Options: nosniff`. The immutable Can response is converted
exactly once. These fixed sinks prevent bytes or text from bypassing HTML trust
or serving application-authored JavaScript.

For plain-text and safe-HTML serialization, native UTF-8 encoding replaces an
unpaired UTF-16 surrogate with U+FFFD; this replacement is the selected sink
contract and is covered by conformance tests. URL construction rejects unpaired
surrogates. JSON remains the shared codec's strict scalar-validating contract
and returns `codec::invalid_data` instead of applying this replacement.

### HTTP client status rule

For approved native fetch declarations, any final status 200--599 completes an
`http::response<T>` after `T` decodes successfully. Redirects are not followed
automatically. To inspect any status independent of a JSON shape, request
`http::response<bytes::buffer>` and explicitly call `codec::decode_json<T>`.
The body-only convenience succeeds only for 200--299 and maps another final
status to `http::status_error(status, headers)`. Transport, timeout, body-limit,
and decode errors remain distinct declared errors.

## P11. HTMX distribution and browser boundary

### Policy

The browser executes upstream HTMX and ordinary browser HTML behavior. It does
not execute Can, generated application JavaScript, authored JavaScript, or a
compiler reimplementation of HTMX.

### Technical contract

The distribution embeds one reviewed upstream HTMX minified asset, initially
the 2.0.10 release bytes with digest
`sha384-H5SrcfygHmAuTDZphMHqBJLc3FhssKjG7w/CeCpFReSfwBWDTKpkzPP8c+cLsK+V`.
The server reserves:

```text
/__can/assets/htmx-2.0.10.min.js
```

The route serves the embedded bytes with the JavaScript media type, immutable
cache headers, ETag, and the recorded digest. Project routes cannot shadow the
`/__can/` prefix. `htmx::runtime_head` renders a local `defer` script reference
with integrity metadata and a compiler-owned `htmx-config` meta element.
That meta configuration fixes `allowEval=false`, `allowScriptTags=false`, and
`selfRequestsOnly=true`. Application HTML cannot override these values because
raw meta/script construction and `hx-on*` are outside the catalogue.

The response policy is fixed initially:

| Status | Browser action |
| --- | --- |
| 204 | no swap, not an error |
| 200--399 except 204 | perform the declared swap |
| 422 | perform the declared swap and treat as an application response |
| other 400--599 | no swap; HTMX error behavior |

This policy lets a form or search handler return a safe 422 fragment containing
validation feedback without user script. Authentication, authorization,
transport, and internal failures can retain truthful non-2xx status without
replacing a page region. A later response-target extension needs a separate
catalogue and conformance decision.

Project static assets are build inputs listed in the compiler asset manifest.
The compiler hashes them, assigns `/__can/project/<digest>/<name>`, and emits
`Bun.file` response operations. `asset::url` has contract
`(static str name) -> html::url emits [html::invalid_url]` and resolves a
declared asset name at compile time. Runtime
path selection and arbitrary asset reads are absent.

The initial project-asset allowlist is `.css`, `.png`, `.jpg`, `.jpeg`, `.gif`,
`.webp`, `.avif`, `.ico`, `.woff`, `.woff2`, `.txt`, `.json`, and `.csv`, with
fixed media types and `nosniff`. The compiler rejects mismatched signatures and
all JavaScript, TypeScript, HTML, SVG/XML, WebAssembly, source maps, and unknown
extensions. The pinned HTMX route is the only executable browser asset.

## P12. Initial PostgreSQL catalogue

### Policy

The initial database target is PostgreSQL through `Bun.SQL`. Queries carry a
static statement, typed parameter record, and typed row record. Values are
bound as native parameters and never concatenated into SQL. The application has
no unsafe-query, raw-driver, migration, or adapter escape.

### Technical contract

Opaque types are `sql::pool`, `sql::transaction`, and compiler-internal query
descriptors. Ordinary records supply parameter and row values. Each `sql` entry
in `can.project.json` has exactly `dialect`, `statement`, `parameters`,
`parameter_type`, `row_type`, `cardinality`, and, for a row-returning descriptor,
`row_limit_parameter`. `dialect` is initially `postgresql`; `cardinality` is
`one`, `optional`, `many`, or `execute`. Type names are fully qualified. A call
passes the descriptor's static manifest name, not SQL text.

The compiler pins `libpg_query` for the selected PostgreSQL major and uses its
actual PostgreSQL parser and scanner. It requires one statement, validates the
cardinality's statement shape, and identifies real parameter tokens without
misreading quoted strings, dollar-quoted bodies, or comments. Application
parameters are contiguous `$1` through `$N` and correspond exactly to the
listed parameter-record fields in declaration order. The scanner locations
split the static source into Bun tagged-template segments. The compiler does not
implement a second SQL lexer, parser, or query algorithm.

Every row-returning descriptor is an initial `SELECT` whose top-level `LIMIT`
is the distinct parameter named by `row_limit_parameter`, immediately after
the application parameters. The adapter binds 2 for `one`/`optional` and
`max_rows + 1` for `many`; receiving the extra row produces `row_count` or
`row_limit`. A `row_count.actual` value is this bounded observed count, so the
overflow value for `one` and `optional` is 2; it never claims to be the total
number of matching database rows. Thus the database bounds
transmitted/materialized rows before Bun constructs its result array. `execute`
descriptors admit initial INSERT/UPDATE/DELETE without `RETURNING`. Richer
returning mutations wait for a separately bounded native contract.

The compiler materializes descriptors through C8's manifest configuration
case. There is no executable top-level query-constructor call. An ordinary
named Can function reuses a descriptor by passing its static name to the
intrinsic. Generated TypeScript emits a Bun tagged-template query with record
fields at the scanner-confirmed interpolation sites. It never emits
`sql.unsafe`, `.simple()`, or runtime-constructed SQL.

| Operation | Contract |
| --- | --- |
| `sql::pool_open` | `(str connection_variable, int max_connections) -> sql::pool emits [http::credentials_missing, sql::connection_failed]` |
| `sql::pool_close` | `(sql::pool, int timeout_ms) -> void emits [sql::close_failed]`; stale handle is standard `resource_state` |
| `sql::query_one<P,R>` | `(sql::pool, static str descriptor, P) -> R emits [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::row_missing, sql::row_count, sql::schema_mismatch, sql::constraint_failed]`; stale handle is standard `resource_state` |
| `sql::query_optional<P,R>` | `(sql::pool, static str descriptor, P) -> option::value<R> emits [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::row_count, sql::schema_mismatch, sql::constraint_failed]`; stale handle is standard `resource_state` |
| `sql::query_rows<P,R>` | `(sql::pool, static str descriptor, P, int max_rows) -> R[] emits [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_limit, sql::schema_mismatch]`; stale handle is standard `resource_state` |
| `sql::execute<P>` | `(sql::pool, static str descriptor, P) -> int emits [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed]`; result is affected rows; stale handle is standard `resource_state` |
| `sql::with_transaction<T>` | `(sql::pool, callable sql::decision<T> (sql::transaction) emits []) -> T emits [sql::connection_failed, sql::transaction_failed, sql::commit_unknown]`; stale handle is standard `resource_state` |

The transaction versions of `query_one`, `query_optional`, `query_rows`, and
`execute` have the same contracts with `sql::transaction` as their first input
and omit `sql::connection_failed` after the callback begins. Their source names
are distinct (`sql::transaction_query_one`, and so on), avoiding an implicit
subtyping or overload requirement.

`sql::decision<T>` is the compiler-owned ordinary named variant whose leaves are
ordinary records `sql::commit<T> { T value }` and
`sql::rollback<T> { T value }`. The callback handles query domain errors into
one of those values, which is why its declared error set is empty. Standard
failure still leaves the callback abruptly.

The adapter uses `pool.begin(async tx => ...)`. Every authored callback outcome
starts closing and drains every coordination owner created inside the callback
that holds a transaction lease. Existing owners may finish through subleases;
new owners cannot enter. The native callback remains pending throughout this
drain. A returned `commit` value then lets it finish normally. A `rollback`
value instead throws a private adapter sentinel, caught outside `begin`, so
Bun performs its native rollback; the payload is then returned as the
successful Can value. On a standard failure, the adapter retains the original
failure, performs the same drain, and then rethrows it through the native
callback so Bun rolls back. The sentinel handler does not catch that failure.
A rollback/cleanup failure accompanying a primary standard failure is a
secondary diagnostic under C9 and does not replace the primary failure. Can
does not implement transaction or rollback algorithms.

If `begin` rejects after the Can callback produced `commit`, the adapter cannot
prove whether the server committed. It returns
`sql::commit_unknown(transaction_id)`. Before callback entry, connection errors
map to `sql::connection_failed`; during the callback, query failures use their
query contract; rollback failure maps to `sql::transaction_failed("rollback")`.

The transaction handle is valid only within its owning callback scope and the
existing owners being drained for that scope. A transaction with a nonsettling
internal owner remains pending; there is no implicit timeout or cancellation.
`pool_close` marks the pool closing, rejects work from new owners, and first
waits for all leases to drain. Only then does it invoke Bun SQL `close`, using
the remaining timeout. The `pool_close` caller's deadline covers both phases.
Expiry returns `sql::close_failed` and leaves the closing resource owned and
observed; it never forces a native close beneath a live lease or completes the
root drain. The root may therefore remain pending. Only an external process
supervisor may bound host lifetime, and its termination carries no Can claim of
transaction rollback, cleanup completion, or runtime-controlled exit.
The manifest's driver-side limit prevents more than `max_rows + 1` rows from
materializing; observing the extra row returns `sql::row_limit`. Each native
row is copied and validated field by field before
it becomes `R`; missing, extra, null, unsafe integer, and wrong native values
produce `sql::schema_mismatch` with a field path.

`pool_open` emits `new SQL({ adapter: "postgres", url, max, bigint: true })`.
The row adapter accepts native `number` only when integral and safe, or native
`bigint`, and converts either exactly to Can `int`. Parameters outside signed
PostgreSQL `bigint` produce `sql::unsupported_value` before launch. A `str`
parameter containing an unpaired UTF-16 surrogate also produces
`sql::unsupported_value` before native UTF-8 encoding can replace it. A native
`null` maps to `option::none`; a non-null optional value validates and maps to
`option::some`. Null for a required field and a non-null value of the wrong type
produce `sql::schema_mismatch`.

Initial SQL values are `bool`, signed-64-bit Can `int` at this boundary, finite
`float`, scalar-valid `str`, and `bytes::buffer`. SQL NULL is admitted only as
`option::value<S>` where `S` is exactly one of those admitted scalar types; an
arbitrary variant or nested optional is not a SQL parameter or row type.
Decimal, timestamps, JSON columns, arrays, enums, streaming, batches,
savepoints, nested transactions, multiple dialects, migrations, and
schema-introspection proofs are deferred. Money in the initial slice uses an
integer minor-unit field.

## P13. Closed account-search trace

### Policy

This trace is the admission example for the initial HTTP, SSR, HTMX, and SQL
slice. It uses no browser Can, authored JavaScript, raw HTML, raw SQL execution,
or hidden application error mapping.

### Technical contract

The program defines ordinary records `search_parameters { str term }` and
`account_row { int id, str display_name }`, an ordinary HTML renderer, and four
named mounted callbacks:

```text
show_accounts  : callable http::server_response (http::request) emits []
search_accounts: callable http::server_response (http::request) emits []
validate_account: callable http::server_response (http::request) emits []
dashboard_summary: callable http::server_response (http::request) emits []
```

The mounted callbacks that query data capture one open `sql::pool` through
their named wrapper's `near` input. Every call below uses existing ordinary
call and completion matching; the signatures are contract notation, not new
source syntax.

The complete query descriptor in `can.project.json` is:

```json
{
  "source_root": "src",
  "dependencies": {},
  "assets": { "site_css": "assets/site.css" },
  "sql": {
    "search_accounts": {
      "dialect": "postgresql",
      "statement": "SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2",
      "parameters": ["term"],
      "parameter_type": "main::search_parameters",
      "row_type": "main::account_row",
      "cardinality": "many",
      "row_limit_parameter": 2
    }
  },
  "error_registry": "can.errors.json"
}
```

The ordinary consumer layer can be asserted without forging a pool. A
production loader captures `sql::pool pool` through `near`, accepts `str query`,
and has exactly the six-error bound shown on `sample_loader` below. Its body is
the following excerpt; this is not a complete declaration with its mandatory
documentation and assertions:

```text
match call sql::query_rows(pool, "search_accounts", search_parameters("%" + query + "%"), 25)
    ok
    sql::unsupported_value
    sql::connection_failed
    sql::query_failed
    sql::constraint_failed
    sql::row_limit
    sql::schema_mismatch
```

The consumer's assertion input uses a capture-free named callable whose body
is not executed during those consumer assertions because `when` supplies the
exact invocation. These are complete declarations in package `main` with
`uses [sql]`; only the package header is omitted:

```text
/// Parameters for the manifest search descriptor.
record search_parameters
    str term

/// One validated account row.
record account_row
    int id
    str display_name

/// Data passed to the safe fragment renderer.
record search_view
    int status
    account_row[] rows
    str message

/// Supply a named loader value for consumer assertions.
fn account_row[] sample_loader
    emits [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_limit, sql::schema_mismatch]
    given
        str query
    asserts
        empty: "x" => ok []
    ok []

/// Map the loader's complete domain bound to response data.
fn search_view search_accounts_model
    emits []
    given
        str query
        callable account_row[] (str) emits [sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_limit, sql::schema_mismatch] loader
    asserts
        found: "ann", callable sample_loader => ok search_view(200, [account_row(7, "Ann")], "")
        unavailable: "ann", callable sample_loader => ok search_view(503, [], "Temporarily unavailable.")
    match call loader(query)
        when
            found: "ann" => ok [account_row(7, "Ann")]
            unavailable: "ann" => sql::query_failed("search_accounts", "08006")
        ok account_row[] rows => ok search_view(200, rows, "")
        sql::connection_failed => ok search_view(503, [], "Temporarily unavailable.")
        sql::query_failed => ok search_view(503, [], "Temporarily unavailable.")
        sql::unsupported_value => ok search_view(500, [], "Internal error.")
        sql::constraint_failed => ok search_view(500, [], "Internal error.")
        sql::row_limit => ok search_view(500, [], "Internal error.")
        sql::schema_mismatch => ok search_view(500, [], "Internal error.")
```

The declarations and body excerpt use current Can syntax. The manifest and
catalogue signatures remain configuration/contract notation. The model leaves
diagnostic logging to the service wrapper, which uses the same mapping below.

The successful flow is:

1. `main` opens the PostgreSQL pool, constructs GET routes `/accounts`,
   `/accounts/search`, and `/dashboard/summary` plus POST `/accounts/validate`,
   starts the server, and awaits `server_wait`.
2. `GET /accounts` invokes `show_accounts`.
3. The callback builds a safe document whose head contains
   `htmx::runtime_head` and `html::stylesheet(asset::url("site_css"))`, and whose
   body contains a form with `hx-get=/accounts/search`,
   `hx-target=#account_results`,
   `hx-swap=innerHTML`, and `hx-trigger=change`, plus an empty result region.
4. Total status/header constructors provide `status_ok` and empty headers;
   `http::response_html(status_ok, empty_headers, document)` returns the page.
5. The browser loads the locally served pinned HTMX asset.
6. A search submits `GET /accounts/search?query=ann`.
7. Bun snapshots the request and invokes the exact named `search_accounts`
   callback.
8. `http::query_one(request, "query")` returns `ann`.
9. The callback constructs `search_parameters("%ann%")` and calls
   `sql::query_rows(pool, "search_accounts", parameters, 25)`.
10. The compiler emits the manifest descriptor as one Bun SQL tagged template,
    binding the term and compiler-owned row-limit value 26. PostgreSQL receives
    both as parameters, never as statement text.
11. The adapter validates each row as `account_row`, copies it into immutable
    Can records, and preserves database order.
12. The renderer calls `html::text` for every display name. A name such as
    `</li><script>bad()</script>` appears only as escaped text.
13. `html::fragment` produces the trusted result-list fragment and the total
    response constructor completes the callback with `status_ok` and empty
    headers.
14. HTMX swaps the fragment into `#account_results` with `innerHTML`.
15. On a shutdown signal, `server_wait` completes only after the server stops;
    `main` then closes the pool and reaches bare `ok` for exit code zero.

All alternate outcomes close:

| Trigger | Callback mapping | HTTP/HTMX observation |
| --- | --- | --- |
| query absent, repeated, or blank after application validation | safe validation fragment | 422; compiler-owned policy swaps it into the result region |
| SQL connection/query failure | fixed safe unavailable page; stable error logged | 503; HTMX does not swap |
| row schema mismatch | fixed safe unavailable page; field path logged | 500; HTMX does not swap |
| unsupported SQL value, constraint failure, or row limit | fixed safe internal-error page; stable error logged | 500; HTMX does not swap |
| HTML/URL/HTMX construction error | total escaped-text fallback response; reason logged | 500; HTMX does not swap |
| unknown route | compiler-owned response | 404; no callback and no swap |
| wrong method on known route | compiler-owned response with `Allow` | 405; no callback and no swap |
| standard failure in renderer or callback | sanitized boundary response | 500; HTMX does not swap |
| SIGINT/SIGTERM | stop accepting, await in-flight requests, close pool | clean zero exit if both closes succeed |
| shutdown or pool close failure | sanitized stderr diagnostic | nonzero process exit |

The mounted callback itself has `emits []` because it explicitly maps every
declared input, SQL, and HTML domain error to a complete response. Application
mapping can deliberately return 500 for a caught domain error; the boundary's
sanitized 500 remains the fallback for standard failures and adapter invariant
failures.

The same page closes the approved form and dashboard capabilities. A validation
form uses typed `htmx::post`, targets its own result region, marks its submit
button with `htmx::disable_this`, and names a progress element through
`htmx::indicator_id`. `validate_account` decodes `request_form<account_form>`;
invalid user data returns a safe 422 fragment and valid data returns a safe 200
fragment. A dashboard section uses `htmx::get` plus
`htmx::trigger_every(5000)` to replace its summary fragment every five seconds.
These behaviors use upstream HTMX request state and polling; no browser Can or
application script participates.

### Evidence

The admission suite needs:

- `real-can` renderer tests with adversarial text and duplicate attributes;
- `supplied-completion` callback assertions for every table row above;
- raw HTTP fixtures for query multiplicity, body limits, response headers, and
  non-2xx behavior;
- raw SQL fixtures for parameter order, zero/one/many rows, wrong row types,
  constraint failure, rollback, and unknown commit;
- Bun conformance for async fetch callbacks, graceful stop and deadline
  retention, SQL tagged parameters, async transactions, pool close, and
  referenced-resource lifetime;
- a browser integration test using the pinned HTMX asset for 200 and 422 swaps
  plus 404, 405, 500, and 503 no-swap behavior.

## P14. Capability reconciliation matrix

### Policy

`docs/ASTRA_STDLIB.md` supplies capability intent. Its syntax, type system,
effect, proof, ownership, frontend-runtime, and roadmap assumptions are not
authority. A missing library operation does not imply missing language syntax.

### Technical contract

The table records specification closure, not current implementation status.
Here, **supported** means this specification and the referenced core contract
define an admission target; P15 must still pass before a release claims that
the capability is implemented.

| Capability family | Initial status | Contract or disposition |
| --- | --- | --- |
| booleans and comparisons | supported | native JS operators with Can validation |
| unbounded integers | supported | native bigint arithmetic and the C6 standard-failure contract |
| binary64 floats | supported | native `Number`/`Math`, including C6 NaN, infinity, and signed-zero semantics |
| exact amounts | supported domain pattern | integer minor units plus C8 `divmod` and half-even ratio rounding; no decimal primitive |
| advanced numeric algorithms | deferred | admit individually when a native operation or explicit algorithm decision exists |
| explicit text formatting | supported | approved explicit calls; native string/locale operation fixed per function |
| Unicode and code-unit text operations | supported catalogue subset | exact C6/C7 indexing, scalar, grapheme, normalization, casing, search, and split contracts |
| arrays | supported core subset | immutable adapters over native arrays; append/map/for-each contracts remain authoritative |
| maps and sets | supported catalogue subset | C7 opaque `collections` values, scalar keys, insertion order, and immutable native copies |
| optional values | supported library data | ordinary compiler-owned variant; no null leakage |
| generic outcome storage | unnecessary initially | ordinary records and variants materialize application outcomes |
| bytes | supported | `bytes::buffer` over copied or exclusively owned `Uint8Array` |
| JSON codecs | supported typed subset | native JSON plus schema validation; no untyped JSON value |
| schemas | internal supported subset | compiler descriptors derived from Can types for JSON/fetch/SQL boundaries |
| safe HTML | supported initial subset | P9 trusted constructors and contextual encoding |
| HTTP client | supported initial subset | named fetch, immutable envelope, bounded bodies, no redirect following |
| HTTP server/router | supported initial subset | P10 exact static routes and complete responses |
| SSR and fragments | supported | server-side immutable HTML values; full documents and fragments |
| browser interaction | supported server-driven subset | pinned HTMX requests and swaps from P11 |
| browser Can target | out of scope | approved U7 boundary |
| authored browser JS/TS | out of scope | forbidden platform escape |
| client component mount/hydrate runtime | out of scope initially | interactive form/search/dashboard capability is delivered by SSR plus HTMX |
| immutable view/update functions | supported as ordinary Can | run on server; return new records and safe HTML; no special purity claim |
| SQL PostgreSQL | supported initial subset | P12 static parameterized statements and typed materialized rows |
| SQL other dialects/migrations/streaming | deferred | require dialect-specific contracts and native conformance |
| CLI args/stdin/stdout/stderr | supported | main arguments plus P8 bounded I/O |
| project data files | deferred | no runtime file-path catalogue in the initial slice |
| static web assets | supported build subset | P11 hashed manifest URLs |
| clock/random/crypto/env/log | supported initial subset | finite P8 operations mapped to native JS/Bun APIs with deterministic test substitution |
| resources | supported runtime subset | P6 registry, scoped handles, shutdown; no ownership syntax |
| concurrency | supported approved subset | native promises in production and P5 deterministic tests |
| effects/purity/affine/proofs | out of scope | expressly unnecessary for this slice |
| externs/backend imports/commands | out of scope | forbidden |
| LLM tools | out of scope | excluded from this specification |

No additional Can syntax is required by the initial supported rows. Exact
decimals, richer static SQL declarations, route captures, streaming, and an
alternate authored race schedule are candidate future ergonomics. They remain
library/compiler design questions first; syntax should be proposed only after a
closed program demonstrates that existing calls and declarations cannot carry
the contract.

## P15. Verification gates and release boundary

### Policy

A catalogue entry ships only with its contract, native emission, deterministic
substitution, raw adapter fixtures, and supported-target conformance. No old
adapter, extern, or scripted result counts as evidence for the new boundary.

### Technical contract

The initial platform slice is complete when all of these gates pass:

1. fixture identity tests in P3 and schedule tests in P5;
2. compile-time rejection of malformed, ambiguous, and opaque-forging fixtures;
3. no-live-call enforcement for every side-effecting or nondeterministic
   catalogue boundary in assertions;
4. HTML constructor adversarial cases and context matrix;
5. HTTP request snapshot, exact routing, response, non-2xx, and shutdown tests;
6. local HTMX asset digest verification and browser swap-policy tests;
7. PostgreSQL parameter, row, transaction, unknown-commit, and close tests;
8. resource leak and stale scoped-handle tests;
9. generated-code inspection proving native Bun operations are used; and
10. the complete account-search trace in P13 on the pinned Bun target.

The initial target is macOS on arm64. Linux requires the same conformance suite
before support is claimed. Windows is outside the approved distribution scope.

### Evidence basis and native reuse

The runtime decisions above are grounded in current primary documentation:

- [Bun HTTP server documentation](https://bun.sh/docs/runtime/http/server)
  documents async `fetch` results, referenced server lifetime, graceful
  `stop()`, and forced `stop(true)`.
- [Bun SQL documentation](https://bun.sh/docs/runtime/sql) documents its
  Promise-based API, tagged-template parameters, prepared statements, pooling,
  and async transaction callbacks with native commit/rollback.
- [Bun SQL options reference](https://bun.sh/reference/bun/SQL/Options)
  documents PostgreSQL adapter selection, pool sizing, and bigint output.
- [Bun TransactionSQL reference](https://bun.sh/reference/bun/TransactionSQL)
  documents reserved transaction connections and asynchronous close behavior.
- [Bun file I/O documentation](https://bun.sh/docs/runtime/file-io) documents
  `Bun.file`, `Bun.write`, and Bun stdin/stdout/stderr handles.
- [Bun HTML escaping guide](https://bun.sh/guides/util/escape-html),
  [hashing documentation](https://bun.sh/docs/runtime/hashing),
  [sleep guide](https://bun.sh/guides/util/sleep), and
  [environment documentation](https://bun.sh/docs/runtime/environment-variables)
  document the finite native operations reused by P8 and P9.
- [`libpg_query`](https://github.com/pganalyze/libpg_query) documents that it
  embeds PostgreSQL parser source and exposes scanner token locations, which
  P12 pins instead of defining another SQL parser.
- [HTMX documentation](https://htmx.org/docs/) documents declarative response
  handling, including configurable 422 swapping and default 4xx/5xx errors.
- [HTMX security guidance](https://htmx.org/docs/#security) documents the
  `allowEval`, `allowScriptTags`, and `selfRequestsOnly` configuration used by
  the pinned runtime head.
- [HTMX `hx-post`](https://htmx.org/attributes/hx-post/),
  [`hx-target`](https://htmx.org/attributes/hx-target/), and
  [`hx-swap`](https://htmx.org/attributes/hx-swap/) document the request,
  target, and swap behavior used by P13.
- [HTMX release quick start](https://htmx.org/) records the upstream 2.0.10
  asset and integrity digest used as the initial distribution input.

Compiler adapters may validate and copy values, map errors, manage the resource
registry, and bridge Can completions. HTTP serving, promise coordination, file
I/O, SQL parameterization, transaction control, and HTMX browser behavior stay
with the documented native implementations.

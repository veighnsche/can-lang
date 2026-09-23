# B1-05 comparison: nine programs, three surfaces (B1-05.01)

G-EVENT output: three workflows × three surfaces. Every program shows
acquisition, repeated dispatch, grouped state, explicit handler error sets,
normal end, external failure, backpressure, cancellation, cleanup and
locally attached fixtures. Recorded 2026-09-23 against bundle canlc
`can-214b1e3-bun-1.4.2-darwin-arm64-v1` and pinned Bun 1.4.2
(`35d20dd0…`, see [probe metadata](../probe-run-metadata.json)).

## Programs

Workflow W1 (large file, typed JSONL predicate stop):

- [library](w1-file-library.can) (83 lines)
- [per-domain](w1-file-domain.can) (80 lines)
- [shared-event pull](w1-file-event.can) (110 lines)

Workflow W2 (child stdout/stderr, grouped progress, deadline cancel, reap):

- [library](w2-process-library.can) (92 lines)
- [per-domain](w2-process-domain.can) (83 lines)
- [shared-event pull](w2-process-event.can) (127 lines)

Workflow W3 (socket session: text/binary/drain/close, handler failure):

- [library](w3-socket-library.can) (88 lines)
- [per-domain](w3-socket-domain.can) (81 lines)
- [shared-event pull](w3-socket-event.can) (152 lines, includes echo writes)

Line counts exclude this file. Size alone decided nothing; see the table.

## Surface definitions

Surface L (shared library + named callbacks): one generic
`stream::consume<T, S>(source, initial, limits, handler)` over opaque
`stream::source<T>`; handlers return `stream::decision<S>` built with
`stream::continue<S>` / `stream::stop<S>`; terminal errors
`stream::{read_failed, cancelled, limit_exceeded}`; domains supply
producers (`files::read_lines`, `process::spawn_text_streaming`,
`socket::accept_stream`). Mirrors `sql::with_transaction` /
`sql::decision` exactly.

Surface D (per-domain operations): each domain repeats the contract under
its own names (`files::lines` + `files::line_decision`,
`process::drain_output` + `process::drain_decision`, `socket::serve` +
`socket::session_decision`) with per-domain terminal errors. No shared
module is named.

Surface E (shared-event pull): opaque `stream::reader<T>` /
`stream::writer` with `stream::{read_many, write_some, close, cancel}`;
empty batch is the normal end; loops are ordinary self-recursion; cleanup
is explicit closes with primary-error re-raise by reconstruction.
Producers return readers (`files::read_lines_stream`,
`process::spawn_text_streaming`, `socket::accept_session`).

## Diagnostics

`canlc parse` accepts all nine files (no surface needs new grammar).
`canlc assert` rejects each at the check phase on the hypothetical
catalogue entry only:

| Program | First diagnostic |
|---|---|
| w1-file-library | unknown package "stream" |
| w1-file-domain | no eligible type files::line_decision |
| w1-file-event | unknown package "stream" |
| w2-process-library | unknown package "stream" |
| w2-process-domain | no eligible type process::drain_decision |
| w2-process-event | unknown package "stream" |
| w3-socket-library | unknown package "socket" |
| w3-socket-domain | unknown package "socket" |
| w3-socket-event | unknown package "socket" |

No diagnostic implicates syntax, exhaustivity, callable, recursion or
fixture machinery: every failure names a missing catalogue entry.

## Verified precedents

Each comparison claim below rests on an executed check or read body,
not on the hypothetical programs:

- Callback machinery exists: `array.fold` (`$callback`, `deriveErrors`,
  awaited accumulator) and `sql::with_transaction` with
  `sql::decision<T>` constructors; caller form proven by
  `compiler/testdata/current/sql/transactions.can`.
- Direct self-recursion executes: scratch probe `count_down` passes
  `canlc assert` with `real-can` evidence (2026-09-23).
- Fixture transcripts sequence across recursion: `withFixture` filters
  rows by assertion selector and consumes them FIFO
  (`runtime/assert/fixtures.ts`); per-visit identities come from the
  occurrence counter (`runtime/assert/identity.ts`); overrun is
  `missing fixture`, underrun is `unused fixture`
  (`runtime/assert/context.ts`, `runtime/assert/queue.ts`).
- Opaque-typed givens are engine-supplied in assertions and referenced
  from `when` rows: `decide_add(tx)` precedent in
  `compiler/testdata/current/sql/transactions.can`.
- Error arms bind (`http::request_failed as failed`) and arm bodies
  re-raise by reconstruction
  (`compiler/testdata/current/fetch/main.can`).
- Nested calls as call arguments, calls in assertion inputs, int match
  arms with `_`, `.length`, `+`, `and`, `is`, `do` blocks: scratch
  probe passes `canlc assert` (2026-09-23).
- Native pull is real: 0 pulls before `getReader`, 2 pulls after 2
  reads (`stream_pull` in [probe results](../native-probe-results.json)).
- Cancel settles a pending read as done with no value, and the lock
  releases cleanly (`stream_abort`).
- Chunk objects are distinct per read; `.slice()` detaches a boundary
  copy; streaming `TextDecoder` reassembles split UTF-8 (`stream_alias`).
- `FileSink.write` reports accepted bytes, but write-after-end is
  silent, so ended-state enforcement is adapter-side (`stream_filesink`).
- Child pipe reads are not message-aligned: two writes arrived as one
  four-byte chunk (`stream_process_reader`).

## Behavior table

| Dimension | L shared library | D per-domain | E shared-event pull |
|---|---|---|---|
| New grammar | None (parses) | None (parses) | None (parses) |
| New checker machinery | None (`$callback` + generic nominals exist) | None, but N copies of the same contract shape | None (ordinary ops + recursion + error arms) |
| Catalogue additions | 1 consume + 1 decision nominal + 2 constructors + limits + 3 errors + producers | Per domain: 1 drive op + 1 decision nominal + 2 constructors + options + terminal errors | 4 ops + 2 handle types + 3 errors + producers |
| Dispatch loop | Hidden in adapter | Hidden per adapter | Caller recursion, fully visible |
| Grouped state | Fold-style accumulator | Same, per-domain names | Threaded arguments |
| Handler error sets | Explicit (`emits` on handler, derived) | Explicit, per domain | Explicit (`emits` on pump) |
| Normal end | Decision value | Same | Empty batch |
| External vs handler failure | Derived errors vs `stream::read_failed` | Derived vs domain read error | Read error arms vs pump `emits` |
| Backpressure, read side | Implicit (adapter awaits handler) | Implicit per adapter | Explicit demand (`read_many(n)`) |
| Backpressure, write side | No write path in this surface | No write path in this surface | Explicit short-write retry (`pump_write`) |
| Cancellation | Deadline limit only; no mid-run handle | Same, per-domain options | `cancel` op + cooperative budget checks |
| Cleanup | Adapter-owned, invisible | Same per adapter | Explicit closes; failure arms re-raise primary |
| Terminal queries (exit) | Producer query on the source handle | Ambient `exit_of()` with no handle (smell) | Query on the reader handle |
| Drain (W3) | Control signal delivered as data item (conflation) | Same per domain | No drain case: demand replaces it |
| Fixtures | Handler asserts + whole-run `when` | Same per domain | Handler-equivalent asserts + per-read `when` transcripts with overrun/underrun detection |
| Stop granularity | One item | One item | Caller-chosen batch size |
| Caller boilerplate | Lowest | Low, × N names to learn | Highest (closes per terminal arm) |

## Native sketches

Surface L lowering (one sketch covers D with renamed imports):

```ts
const reader = source.native.getReader();
let state = initial;
try {
  for (;;) {
    const next = await reader.read();
    if (next.done) return ok(state);
    state = await invokeHandler(handler, [state, copyChunk(next.value)]);
    if (isStop(state)) { await reader.cancel(); return ok(unwrap(state)); }
  }
} finally { reader.releaseLock(); }
```

The loop, the stop/cancel coupling and lock discipline are all
adapter-owned and invisible to the caller and its fixtures.

Surface E lowering (one op per adapter; the loop stays in Can):

```ts
// read_many: exactly one native read, copied at the boundary.
export async function readMany(reader, max, context) {
  const target = checkOpen(reader);           // use-after-close / owner
  const next = await target.native.read();    // single pull
  if (next.done) return ok([]);
  return ok([copyChunk(next.value).slice(0, max)]);
}
// close: idempotent terminal publication, lock released once.
```

Each adapter performs one native step; ordering, retries, cleanup order
and terminal publication are caller-visible Can code.

## Decision (B1-05.02 exit)

Selected surface E. It is the only surface where the lifecycle contract
(open → closing → closed, exactly one terminal publication, no
callbacks after closure, cleanup that preserves the primary failure) is
caller code rather than adapter behavior, which makes B1-05.03–.07
directly testable through real dispatch and `when` transcripts. It
needs no new grammar and no new checker machinery, and its fixture
story uses engine semantics that already exist. Its cost is explicit
close boilerplate per terminal arm; a library `consume` in style L can
later be added as an op without changing the contract. Surface D is
rejected: it multiplies one contract into N near-copies and leaves
terminal queries without a handle. Three fresh Jev rounds and the
disagreement audit are saved in
[consultations-b1-05](../consultations-b1-05/decision-audit.md); the
exit record is the G-EVENT row in [decisions](../../decisions.md).

## Illustrative, not verified

- `stream::` / `socket::` error field shapes (`detail`, single-`str`
  construction) follow the `http::`/`codec::`/`text::` patterns but no
  stream error is registered yet.
- `codec::invalid_data("", "syntax")` as the JSON-decode failure value
  is a placeholder; the codec owner will fix exact reasons.
- `when` rows on zero-argument calls (`sample: => ok 1000`) and calls
  inside `when` argument lists follow the assertion-argument checker
  but have no executed precedent yet; B1-05.07 fixtures will prove them
  or restrict to bindings.
- `pending.slice(...)` receiver form for `array.slice` follows the
  `values.map(...)` method-call precedent.
- The pull programs abbreviate fallible recursion as `ok call pump(...)`;
  the implemented checker requires explicit `match` arms over the pump's
  error set there. Real programs (see `examples/stream`) spell those arms.

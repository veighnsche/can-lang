# A04 self-tail proof coverage (Q6 gate input)

## Predicate (exact)

A relay lowers iff it is a direct self call in a hazard-free function
frame:

- the relay is one plain call step targeting the enclosing function
  itself (`step.Identity == region.ID`), with no method chain, dynamic
  callee, native/array/asset/site step, fixture table, or arity
  mismatch (`directSelfCall` in `self_tail.go`);
- the enclosing frame holds no live lease, drain-owned value, pending
  timer, or deferred completion, and no fixture table (`scanTailFrame`).

The proof runs per region after checking (`proveSelfTail`); the emitter
lowers marked relays to `while (true)` iterations with argument
temps evaluated once in order before any parameter move. Non-relay
recursion (nested calls, post-processing) is never a candidate and keeps
nested calls silently.

## Hazard table

| Frame content | Exclusion reason |
|---|---|
| `sql::with_transaction`, `transaction_*` (any instantiation) | live lease (transaction) |
| `stream::*`, s3 stream/upload ops | drain-owned value |
| `browser::set_timeout`, `on_event`, `on_cancel_*` | pending timer / handler |
| coordination (`concurrent`/`race`) | deferred completion (coordination) |
| any `callable` value | deferred completion (callable value) |
| `action::mount` | deferred completion (action mount) |
| `when` fixture table | fixture table |

One-shot settled operations (plain SQL queries, one-shot S3 byte calls,
`sleep_millis`, `pool_open`) hold nothing across the relay and stay
lowerable. The first hazard in source order is recorded on
`ir.Region.TailExclusion`.

## Notes

Unlowered relays on a static recursion cycle get a `CAN-CHECK-NOT-LOWERED`
note: failed self proofs carry their exclusion; mutual relays name their
non-self target. Acyclic relays and non-relay recursion stay silent.

## Step diagnostics

Lowered-loop failures carry the 0-based iteration index as a `step:N`
element in the failure origin invocation path, next to the region
identity; the existing occurrence ID links to logs. Failure identity,
messages, and payloads are unchanged.

## Q6 coverage: required W4 shapes

| W4 shape | Verdict |
|---|---|
| 100k-step state machine (plain self relay) | COVERED: lowers; probe below |
| Growing aggregation via relay + bulk construction (no callables/fixtures) | COVERED by construction |
| Growing aggregation via fold with capturing callable | EXCLUDED (deferred completion); refinement available: allow callables whose inputs reference no region parameter |
| Bounded worker batch | OUT OF SCOPE: F06-owned companion shape mirroring C-A step/failure reporting |

No required W4 shape known today needs the Q6 gate: the state machine
lowers, relay-style aggregation lowers, and the worker batch belongs to
lane F. If A07's aggregation requires capturing callables, the recorded
refinement (capture analysis instead of the presence rule) is the first
step before proposing Q6-O3 syntax.

## Feasibility probe (2026-09-26, bun 1.4.2, MacBook Air)

Scratch probe `/tmp/tailprobe/loop.ts` (not committed): emitted
`countdown`/`faulty` regions executed against the worktree runtime.

- Lowered 100k countdown: `ok 0`, ~1ms, flat heap.
- Nested 100k countdown: standard failure
  `RangeError: Maximum call stack size exceeded` — lowering required.
- Fault after 3 relays: origin invocation
  `can.project.root/app::faulty,step:3` with occurrence ID and the
  preserved `arithmetic: integer division by zero` message.

Full W4 qualification (100/20k/100k state/aggregation runs, memory
growth) belongs to A07.

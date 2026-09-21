# I18 deterministic scheduler consultation

Three fresh requests rewrote all explanatory state, instructions and alternatives while preserving the full P3/P5 identity, queue and barrier obligations. All selected `explicit_frames` with probability and confidence 1.0. The raw requests, responses and equivalence audit are retained. No disagreement occurred. This is advice, not implementation evidence or proof of scheduler correctness.

The selected design carries immutable invocation-context views through generated call boundaries. Shared root state owns lexical FIFO queues; each invocation view owns its parent-relative occurrence counters and full path. Coordination reserves direct/spread positions before any participant starts. Explicit running, child-waiting, fixture-blocked and finished frame states establish barriers without guessing a microtask count. Callable receipts retain creation identity and frozen receiver/near fingerprints. Invalid tokens fail closed and roots remain isolated.

Integration must also account for native aggregate continuations: a participant frame cannot disappear before the native selection adapter has observed its completion. Once a completion can make the native aggregate ready, the barrier must retain an active continuation until the aggregate resumes its caller. This activity bookkeeping does not choose a winner; native Promise operations still do that. Otherwise a scheduler could release a later fixture in the gap between participant completion and the native aggregate's queued reaction.

Root completion does not finish the harness until all owned losers drain. Unused-row checks and final evidence reporting happen afterward. Alternate release orders belong only to compiler conformance tests; authored assertions keep the canonical full-path order. Fixed microtask delays and host-arrival FIFO allocation were rejected alternatives.

The live HTTP API and Choice documentation were read on 2026-09-21. Requests use `jev-latest`; responses identify `jev-1.13.0`. Implementation evidence is recorded in [I18 validation report v1](../i18-validation.md).

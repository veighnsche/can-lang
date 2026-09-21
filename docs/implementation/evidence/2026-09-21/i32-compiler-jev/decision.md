# I32 compiler-mount consultation decision

## Results

`mount`: all three responses selected `ordinary_callable` with confidence
0.96, 0.92, 0.98 (probabilities 0.98, 0.94, 0.99). No disagreement.

`paths`: request 1 selected `check_literal_runtime_recheck` (confidence
0.94, probability 0.97). Requests 2 and 3 selected `runtime_only` with
confidence 0.05 (0.52 vs 0.48) and 0.34 (0.67 vs 0.33).

## Disagreement investigation

The two `runtime_only` answers carry near-zero and low confidence; request
2 is an explicit toss-up (0.52/0.48). They express model uncertainty, not a
substantive counter-argument: no response disputes that the catalogue marks
path `static`, that P10 calls routes statically mounted, or that the
runtime re-validates at mount. A `runtime_only` checker would admit
dynamically computed paths into a `static str` position, contradicting the
catalogue contract the checker exists to enforce. The risk against
check-time validation is Go/TS normalization divergence; it is contained
because the runtime remains authoritative at mount (a check-time accept the
runtime rejects still fails safely) and because a shared normalization
corpus will pin both sides. The one-sided risk of `runtime_only` (dynamic
paths silently accepted as static) has no such backstop.

## Decision (advice, not proof)

Mount routes as ordinary checked calls: the callback operand lowers through
the standard callable path, its residual contract must read exactly
`(http::request) -> http::server_response` with empty emits, and the
resulting `$canOwnCallable` wrapper serves directly as the runtime
`MountedCallback`. No `ir/http.go`; no second invocation pipeline; no
synthetic per-site wrapper. Require a string literal path at check time,
validate its normalization there, pass it as an ordinary argument, and keep
runtime mount validation authoritative.

Compiler, runtime-conformance, and staged integration tests must still prove
exact callback types and error bounds, capture timing, normalization
agreement, duplicate/ambiguous collisions, and 404/405/Allow dispatch
before I32 can close.

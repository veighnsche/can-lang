// Package admission implements the host-wide live-run admission gate and
// atomic declared-peak reservation for native-test runs (task P12).
//
// Capacity is host-wide, not per repository root: one live case, one
// offline verification worker and one build producer. Arbitration is
// kernel flock on three well-known lock files inside one host lock
// directory (live-case.lock, offline-verify.lock, build-producer.lock),
// the same primitive as the P10/P11 leases — never an in-memory latch —
// so two repository roots (two processes, two checkouts) cannot bypass
// the gate. Tests isolate the host directory with t.TempDir.
//
// A process-wide registry keyed by absolute lock path shadows the kernel
// locks so same-process contenders (two Host handles over one directory,
// as in tests) arbitrate identically wherever the package builds;
// cross-process contention is decided by the kernel alone.
//
// Global lock order (never inverted): registry -> host -> kernel flock.
// Every flock attempt is non-blocking and every critical section is
// short, so a failed attempt releases everything already taken and
// returns a named reason instead of waiting: there is no lock cycle to
// deadlock. A demand issued while the calling Host already holds a
// reservation (for example a nested producer demand from inside a held
// live case) fails fast with ErrNestedDemand before touching the kernel.
//
// Declared peaks are atomic: Admit takes every demanded lane in lane
// order (live-case, offline-verify, build-producer) or none, releasing
// partial takes in reverse order. Every admission charges a cleanup
// reserve against an absolute monotonic deadline (start + budget from
// the Host clock, time.Now by default): a case whose body plus cleanup
// cannot fit strictly inside its budget never starts, and reaching the
// deadline is failure even if the body just succeeded — Complete at or
// past the deadline reports ErrDeadlineExceeded.
//
// A 10 GiB available-disk admission floor guards every admission: Admit
// refuses with ErrBelowFloor before touching any lane when the gate
// filesystem holds less, and every Allocate rechecks the floor, so disk
// that drops after Admit refuses the next allocation. The floor is a
// guard, not a reservation — it holds no bytes against other programs —
// and a failed probe refuses the same way: admission fails closed. The
// production probe reads Statfs on the host lock directory (production
// places that directory on the same filesystem as run scratch); tests
// inject a fake probe via OpenHostWithClockAndFreeDisk, beside the fake
// clock.
//
// Every request also declares finite capability ceilings (open handles,
// pending operations, scratch bytes). Omitted, zero, negative or
// otherwise unbounded limits are rejected before any effect; there is no
// unlimited spelling. Admission records the declared ceilings on the
// Grant. Policing live consumption against them — strict process, memory
// and disk enforcement with a demonstrated overshoot bound — is P13 host
// qualification, which this gate does not claim: the floor and the
// declared caps are necessary admission protection, not evidence of
// strict owned enforcement.
//
// Integration (explicit, no code coupling): Admit before the first live
// effect (journal Reserve / process Spawn), call Allocate before each
// allocation effect under the Grant, hold the Grant across the run, then
// Complete once the cleanup receipt exists (or Release on abort paths).
// The P11 stop-admission Gate sits above this gate for cooperative
// drain; this gate is the kernel-enforced capacity floor, and ProbeLane
// is the independent per-lane witness in the style of
// process.ProbeLease.
package admission

// Package host implements the strict host enforcement adapter and its
// qualification battery for native-test runs (task P13).
//
// Strict here means refuse-before-effect with finite stated bounds: every
// spawn authorization and every tmp byte is charged against the envelope
// BEFORE the effect exists, and an over-envelope charge fails synchronously
// with no effect. Nothing is admitted-then-killed and no bound rests on a
// sampled peak. Adapters whose enforcement is poll-and-kill or sampled
// peaks fail qualification behaviorally (the negative controls prove it),
// and an unavailable host blocks admission outright.
//
// The selected adapter is darwin-strict. Its mechanisms are macOS-real
// (no cgroups):
//
//   - processes: pre-fork ledger (overshoot 0, detached or not) with the
//     kernel RLIMIT_NPROC fork refusal demonstrated live and the envelope
//     sized within the probed soft limit;
//   - temporary data: pre-write ledger (overshoot 0 on the helper path)
//     with the kernel RLIMIT_FSIZE write refusal demonstrated live and a
//     decision-point rescan that catches writes outside workspace helpers
//     (bypass bytes bounded physically by probed free disk, fail closed
//     on overflow);
//   - memory: declared-peak ledger charged before spawn (overshoot 0 on
//     charged totals) with the envelope sized within probed host commit
//     (hw.memsize). There is no kernel RSS backstop on darwin:
//     RLIMIT_AS and RLIMIT_DATA are unsettable (EINVAL, shared-cache
//     mapping), which the battery probes and records instead of assuming.
//
// Every envelope holds a disposal reserve (one disposer proc slot, RSS
// headroom, receipt spill) back from the admission ceiling, so disposal
// succeeds at full charge. Admission builds on the P12 gate: Admit takes
// a valid Qualification, re-probes fresh host facts (fail closed), takes
// the admission.Host lane, then opens the scope. No lane is held unless
// the mechanism is demonstrated.
//
// Qualification or promotion scope: Qualify runs the full battery
// (behavioral checks on small control envelopes plus charged-scope
// arithmetic on the real envelope, never 64 real processes or GiB
// allocations) and returns the record the integrator commits separately
// as the host-profile acceptance. See ProfileRecord.
package host

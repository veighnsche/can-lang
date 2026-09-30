// Package acceptance independently accepts N, the native toolchain
// under test, as a separately built and versioned executable.
//
// N is never the test process itself: it is a distinct executable
// (in local controls a re-exec helper or fixture shell) whose version
// facts are attested against its executable bytes. T is the independent
// witness: test-side machinery with its own process owner, journal,
// lease, admission gate and recovery drain, holding a process/resource
// graph that is separate from N's.
//
// Accept runs N under T's witness and binds protocol, journal,
// enforcement, recovery and receipt behavior into one N acceptance
// manifest. Complete success needs every binding at once: N death, a
// forged or absent receipt, a lingering descendant or descriptor, a
// cleanup failure, T's own loss, host drift and foreign-host execution
// each force an incomplete verdict, never a green receipt.
//
// Acceptance is scoped to the exact qualified host profile: the
// selected host's facts are re-probed and compared, and another host
// requires explicit selection plus its own receipt. Execution never
// moves silently and unsupported enforcement never reads as a pass.
package acceptance

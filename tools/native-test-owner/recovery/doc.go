// Package recovery implements crash recovery for native-test owned
// resources (task P11): a stop-admission gate, a journal replay scanner
// that classifies every pending operation with a named allow/refuse
// decision, and a bounded drain of live owned children.
//
// The scanner replays journal intent after owner death without mutating
// it: no journal advances, no signals, no deletions. Each pending
// operation is either recoverable here (Allow) or refused with a named
// reason: owner-mismatch (worker/controller/owner death with an intact
// target), foreign-process-live (an active foreign PID), replaced-path (a
// directory or file that is no longer the reserved object),
// unknown-schema-version (an old run layout this scanner does not
// understand), or incomplete-liveness-proof (liveness that cannot be
// proven). A refusal authorizes no signal, no deletion and no old-run
// pass; the cleanup package consumes the decision and preserves the old
// incomplete outcome.
//
// Drain reuses process.Owner (never a fork): only retained children with
// verified spawn-start identity are ever signaled, so foreign PIDs are
// unreachable by construction.
package recovery

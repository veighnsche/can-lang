// Package f1 holds the K18 F1 descriptor fixtures: fixed inert byte
// vectors plus bounded local controls that drive descriptor-delivery
// mechanics through the P10 owner package
// (tools/native-test-owner/process).
//
// K17 (schemas/native-test/descriptors/) pins the wire contract this
// package exercises: child-side fds 0/1/2/3/4 with fixed directions
// (0/3 in, 1/2 out, 4 hold), byte/EOF/lifetime states, and fresh
// per-launch snapshots. QF1 (tests/native-can/qualification/qf1/) owns
// the scenario decisions, expectations and integrated credit; this
// package offers only stimuli (fixed bytes) and mechanism controls, and
// runs no live host-dependent checks.
//
// Subject of record: the existing Go CLI is the canlc compiler launcher
// (package main in compiler/main.go). It cannot speak the fd-3 status
// protocol: it never reads fd 3 and never emits the content-bound
// status ack the owner requires, so no canlc launch can be Accepted.
// canlc therefore stays the subject of record for startup-isolation
// fixtures only (identity pinned by TestSubjectOfRecord without building
// the compiler), while descriptor-delivery mechanics are driven through
// the P10 Owner with fixture children (/bin/sh, /bin/cat, /bin/sleep).
// No fixture CLI exists under tests/ or tools/; nothing here forks the
// owner package.
//
// Every control is bounded and local: exec-form sleepers where a live
// child is needed, t.TempDir for outputs and leases, deferred Abort of
// every spawned child, and a Live()==0 assertion so no test leaves
// strays.
package f1

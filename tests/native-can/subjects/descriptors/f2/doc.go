// Package f2 holds the K19 F2 inherited-lease fixtures: raw readiness
// and generation-lease facts plus bounded local controls that drive
// fd-4 lease mechanics through the P10 owner package
// (tools/native-test-owner/process).
//
// K17 (schemas/native-test/descriptors/) pins the descriptor contract
// this package exercises: fd 4 carries the generation lease with hold
// direction, and fresh per-launch snapshots keep each launch
// independent. K18 (f1) covers fd-3 delivery mechanics; this package
// covers the lease half: an actual inherited kernel lock (flock on the
// lease file) protects the generation while any descendant holds fd 4,
// and an early-closed or missing lease loses protection.
//
// Subject helpers expose raw facts only: readiness stimuli (shell
// scripts that drain fd 3, emit the content-bound ack, then hold or
// exit), the independent lease witness (process.ProbeLease recorded
// without interpretation), and the prune reporter (unlink attempt plus
// the post-prune witness). Readiness itself stays the owner's facts
// (Accepted/EOF/Reaped via CollectStatus/Facts); this package adds no
// verdict, sequence or policy. QF2
// (tests/native-can/qualification/qf2/) owns the Can sequence and
// verdict; this package runs no live host-dependent checks.
//
// Every control is bounded and local: exec-form sleepers where a live
// holder is needed, t.TempDir generations, deferred Abort of every
// spawned child, and a Live()==0 assertion so no test leaves strays.
package f2

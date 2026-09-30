// Package cleanup implements owner-only cleanup and compact completion
// receipts for native-test owned resources (task P11).
//
// Perform deletes at most the reserved target, and only when two gates
// both pass: the caller's spawn-start identity must match the journal
// reservation (a bare PID is never authority), and the recovery scan
// decision for the operation must allow it. Every refusal — wrong
// identity, replaced path, a negative scan decision, or a failed removal
// — deletes nothing and preserves the prior outcome; refusals never
// report a pass.
//
// Success reports Complete with the prior outcome preserved: a later safe
// cleanup of an incomplete run stays incomplete. History is never
// rewritten in either direction. Receipts are compact versioned JSON;
// unknown schemas fail closed and are never a pass.
//
// This package never signals a process: it imports no process owner, so a
// signal is unreachable here by construction. Draining live children is
// the recovery package's bounded drain.
package cleanup

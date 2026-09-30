// Package journal implements the durable outside-workspace reservation and
// ownership ledger for native-test resource ownership (task P07).
//
// The journal records intent before the first externally visible effect:
// owner identity (PID plus caller-supplied spawn-start token), path identity
// (absolute path plus device/inode file identity where the platform exposes
// it), lease, operation ID and argument digest, partial acquisitions, and a
// monotonic resource state (reserved → opening → live → closing → closed,
// with failed as a terminal side state).
//
// Records are appended as one JSON object per line to journal.jsonl inside a
// directory outside every managed workspace, fsynced before Reserve/Advance
// return, so a crash between reservation and marker creation stays
// discoverable: reopening the journal replays the log and Pending reports
// every non-terminal operation. Replaying the same operation ID with a
// different argument digest is rejected (ErrConflict); the same ID with the
// same digest rejoins the existing operation without a new effect.
// VerifyAuthority refuses wrong-PID, wrong-start-token and wrong/replaced-path
// callers: printed IDs and PIDs alone are never authority.
//
// OpenMemory provides the same state machine over a non-durable store for
// callers that do not need crash discoverability.
package journal

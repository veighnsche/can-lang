// Package dispatch implements run/case grants, typed handles, at-most-once
// operation dispatch and bounded framed supervisor channels for the
// native-test Go owner (task P08).
//
// Grants are unforgeable in-memory capabilities: a run grant authorizes one
// run, and a case grant derives from a live run grant and binds one case of
// the same run. Tokens are random 128-bit values rendered as short strings
// that fit the owner_grant envelope field; verification checks revocation,
// expiry, scope and run/case binding.
//
// Handles are typed references (run, case, operation) with generations.
// Every use re-checks kind, registration, generation and closed state:
// mixing kinds fails with ErrWrongKind, reuse after reopen fails with
// ErrStaleHandle, and use after close fails with ErrClosedHandle.
//
// The Dispatcher executes each operation effect at most once per operation
// ID. Repeating an ID with the same run, operation name and argument digest
// joins the recorded terminal result without a new effect; any difference
// is rejected with ErrChangedInput. A lost acknowledgment is reported with
// MarkAckLost (or adopted with AdoptIndeterminate after a restart) and all
// later joins report indeterminate without redispatching.
//
// Channels are kind-pinned bounded FIFOs of codec frames with per-channel
// sequence spaces: one each for requests, replies and events. Payload and
// queue-depth bounds are enforced on send; only bytes sent through Send are
// ever decoded, so subject stdout presented anywhere else can never forge a
// supervisor message.
package dispatch

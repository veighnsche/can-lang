// Package service implements the native-test Go owner service (task P08):
// run/case grant issuance, typed handles, journaled at-most-once operation
// dispatch and bounded framed request/reply/event transport.
//
// Every dispatch verifies its grant and handle, records journal intent
// before the first effect (P07), executes the registered handler at most
// once per operation ID, advances the journal to a terminal state and
// returns a typed operation handle plus the result. Repeats with identical
// arguments join the recorded outcome; changed arguments are rejected;
// lost acknowledgments and durable intents without a local terminal fact
// report indeterminate without redispatching. Pending exposes journaled
// intents left non-terminal by a crash for reconciliation.
//
// Framed requests arrive on the request channel and exactly one framed
// reply is produced per request by HandleRequestFrame; EmitEvent publishes
// supervisor events. Subject stdout is observed opaquely with
// ObserveSubjectStdout and is never decoded: forged bytes there cannot
// become supervisor messages, grants or dispatches.
package service

// Package process implements owned processes and descriptors for the
// native-test owner (task P10): an exact process tree with descriptor
// authority over the private environment channel (fd 3) and the
// generation lease (fd 4).
//
// Every spawn records the exact executable, environment snapshot and fd
// map before the child starts, and binds the child to a spawn-start
// identity (PID plus a minted start token): a bare PID is never
// authority, so PID reuse can never redirect an operation. Journal intent
// (P07) is reserved before fork and advanced along the lifecycle.
//
// The offered/accepted/EOF/writer-exit/child-exit facts stay separate:
// offered means the owner wrote the full environment frame to fd 3,
// accepted means the child returned a content-bound acknowledgment over
// the status channel, EOF means the channel then ended cleanly,
// writer-exit is the delivery goroutine outcome (EPIPE included), and
// child-exit is the reaped wait status. Release reports clean only when
// every fact holds together with the lease witness; each negative
// control (fd-3 non-reader, delayed EOF, malformed frame,
// orphan/detached child, PID reuse) blocks clean release on its own.
//
// The generation lease is a kernel-level shared flock inherited on fd 4.
// Lock state lives in the open file description, so it survives the
// launcher's exit while any descendant holds the descriptor. Release
// probes the lease with an independent fresh open (the release witness):
// held-then-released proves protection; never-held or still-held after
// reap proves loss or an orphaned holder and refuses clean release.
package process

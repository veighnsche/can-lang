// Raw readiness/lease helpers for the F2 inherited-lease fixture.
//
// Every helper here reports raw kernel/launch facts and renders fixed
// stimuli; none decides clean, sequences stages, or gates on another
// fact. Controls (controls_test.go) do the gating; QF2 owns the Can
// sequence and verdict.
package f2

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/veighnsche/can-lang/tools/native-test-owner/process"
)

// LeaseFileName is the fixed lease file name inside a generation
// directory. One generation, one lease path: the kernel lock on this
// file is the protection the controls witness.
const LeaseFileName = "generation.lease"

// LeasePath joins a generation directory with the fixed lease name.
func LeasePath(dir string) string {
	return filepath.Join(dir, LeaseFileName)
}

// ReadyHoldScript renders a shell script that drains fd 3 to /dev/null,
// emits ackFrame on the status channel (stdout), then replaces itself
// with a sleeper that holds inherited fd 4 for secs seconds. The ack
// is the child's readiness signal; the sleeper keeps the lease share
// open while ready. catBin and sleepBin are embedded verbatim and must
// be absolute trusted child paths; ackFrame is octal-escaped for
// printf %b.
func ReadyHoldScript(catBin, sleepBin string, ackFrame []byte, secs int) string {
	return fmt.Sprintf("%s <&3 >/dev/null; printf '%%b' '%s'; exec %s %d",
		catBin, process.OctalEscape(ackFrame), sleepBin, secs)
}

// ReadyExitScript renders a shell script that drains fd 3 to /dev/null,
// emits ackFrame on the status channel (stdout), then exits with code.
// Unlike ReadyHoldScript the holder is gone once ready, so a later
// witness sees only the launcher's own share.
func ReadyExitScript(catBin string, ackFrame []byte, code int) string {
	return fmt.Sprintf("%s <&3 >/dev/null; printf '%%b' '%s'; exit %d",
		catBin, process.OctalEscape(ackFrame), code)
}

// Witness is one independent lease observation: the ProbeLease outcome
// recorded without interpretation. Held is true only when an
// independent exclusive probe provably failed; Err carries any probe
// failure (a missing path included) verbatim.
type Witness struct {
	Path string
	Held bool
	Err  error
}

// WitnessLease probes path independently of every owner and child
// descriptor and records the raw outcome. It never decides.
func WitnessLease(path string) Witness {
	held, err := process.ProbeLease(path)
	return Witness{Path: path, Held: held, Err: err}
}

// PruneOutcome is one raw prune observation: the unlink attempt on the
// lease file plus the independent witness afterwards. Removed reports
// the unlink alone; After reports what the path witnesses once the
// unlink has (or has not) happened. No gating lives here: controls
// decide when pruning is allowed and assert what each order observes.
type PruneOutcome struct {
	Path      string
	Removed   bool
	RemoveErr error
	After     Witness
}

// PruneLeaseFile attempts to unlink the lease file at path, then
// witnesses the path. The outcome is raw facts only.
func PruneLeaseFile(path string) PruneOutcome {
	out := PruneOutcome{Path: path}
	if err := os.Remove(path); err != nil {
		out.RemoveErr = err
	} else {
		out.Removed = true
	}
	out.After = WitnessLease(path)
	return out
}

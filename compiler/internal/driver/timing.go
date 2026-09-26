package driver

import (
	"fmt"
	"os"
	"time"
)

// phaseTimer reports build-phase durations, one line per phase, when
// CANLC_TIMING=1: `canlc-timing <name> <ms>` on stderr. Off by default,
// so stdout/stderr contracts stay byte-identical. Phase timers nest; each
// line carries its own wall time including nested phases.
func phaseTimer(name string) func() {
	if os.Getenv("CANLC_TIMING") != "1" {
		return func() {}
	}
	start := time.Now()
	return func() {
		fmt.Fprintf(os.Stderr, "canlc-timing %s %dms\n", name, time.Since(start).Milliseconds())
	}
}

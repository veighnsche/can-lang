package driver

import (
	"fmt"
	"io"

	"github.com/veighnsche/can-lang/compiler/internal/check"
)

// reportWarnings prints advisory check findings to the CLI diagnostic
// stream. Warnings never fail the pipeline: callers print them after a
// successful check and continue with exit code 0.
func reportWarnings(stderr io.Writer, warnings []check.Warning) {
	for _, warning := range warnings {
		fmt.Fprintln(stderr, warning.Format())
	}
}

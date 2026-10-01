// Bounded-control process entrypoints.
//
// The observer and driver children are re-execed test binaries gated on
// environment, following the P12 cross-process pattern: no fixture
// binaries, no network, no live hosts. The observer child runs
// ObserverChildMain; the driver stub blocks on stdin until killed and
// exists only so the controls can SIGKILL a real driver process while
// the observer keeps observing.
package browser_observer

import (
	"io"
	"os"
	"testing"
)

// DriverEnvGate selects the driver stub child.
const DriverEnvGate = "CAN_BROWSER_DRIVER_CHILD"

func TestMain(m *testing.M) {
	if os.Getenv(ChildEnvGate) == "1" {
		os.Exit(ObserverChildMain())
	}
	if os.Getenv(DriverEnvGate) == "1" {
		// Block until stdin closes or the process is killed. The
		// stub performs no observation and holds no interval: it is
		// the killable driver subtree, in miniature.
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

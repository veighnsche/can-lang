package acceptance

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/tools/native-test-owner/protocol/codec"
)

// NHelperArgs are the argv entries that run this test binary as N, the
// subject under test, instead of as the test suite.
func NHelperArgs() []string {
	return []string{"-test.run=^TestNSubjectHelper$"}
}

// TestNSubjectHelper is N's entrypoint when this test binary is re-execed
// as the subject under test. In a normal suite run it skips: it acts
// only when explicitly selected on argv AND handed a valid fd-3 snapshot,
// so an accidental -run still skips instead of exiting the suite.
//
// As N it genuinely reads fd 3, requires the bound version facts,
// honors the die control mode, and emits exactly one content-bound
// codec ack frame before exiting. os.Exit skips the testing footer so
// the status channel carries only the frame.
func TestNSubjectHelper(t *testing.T) {
	invoked := false
	for _, a := range os.Args {
		if strings.Contains(a, "TestNSubjectHelper") {
			invoked = true
		}
	}
	if !invoked {
		t.Skip("N subject entrypoint: child-only")
	}
	snap, ok := readNEnv()
	if !ok {
		t.Skip("no fd-3 snapshot: not an N child")
	}
	var env map[string]string
	if err := json.Unmarshal(snap, &env); err != nil {
		os.Exit(5)
	}
	op, version, digest := env[EnvOp], env[EnvVersion], env[EnvDigest]
	if op == "" || version == "" || digest == "" {
		os.Exit(3) // version facts missing: N refuses to run versionless
	}
	if env[EnvMode] == ModeDie {
		os.Exit(1) // N death mid-run control: read fd 3, then die silent
	}
	sum := sha256.Sum256(snap)
	payload := "ACK:" + op + ":" + hex.EncodeToString(sum[:])
	var buf bytes.Buffer
	if err := codec.NewEncoder(&buf, 1<<20).Encode(codec.KindEvent, []byte(payload)); err != nil {
		os.Exit(4)
	}
	if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
		os.Exit(4)
	}
	// Exit at once with the ack in flight: the owner holds the status
	// pipe open until EOF is observed, however late collection comes,
	// so a fast N still collects clean.
	os.Exit(0)
}

func readNEnv() ([]byte, bool) {
	f := os.NewFile(3, "n-env")
	if f == nil {
		return nil, false
	}
	defer f.Close()
	snap, err := io.ReadAll(io.LimitReader(f, 1<<20+1))
	if err != nil || len(snap) == 0 || len(snap) > 1<<20 {
		return nil, false
	}
	return snap, true
}

func requireDarwin(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only host qualification (P13 OS-specific record)")
	}
}

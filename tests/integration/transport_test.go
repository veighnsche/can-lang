package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBundledTransportLoopback(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged transport qualification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	clean := t.TempDir()
	command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", `(version 1)(allow default)(deny network*)(allow network-bind (local ip "localhost:*"))(allow network-inbound (local ip "localhost:*"))(allow network-outbound (remote ip "localhost:*"))`, filepath.Join(bundle, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), "test", filepath.Join(bundle, "runtime/test/transport-body.test.ts"), filepath.Join(bundle, "runtime/test/transport-request.test.ts"), filepath.Join(bundle, "runtime/test/transport-owned.test.ts"), filepath.Join(bundle, "runtime/test/transport-fetch.test.ts"), filepath.Join(bundle, "runtime/test/transport-http.test.ts"), filepath.Join(bundle, "runtime/test/transport-named.test.ts"), filepath.Join(bundle, "runtime/test/transport-late.test.ts"))
	command.Dir = clean
	command.Env = []string{"HOME=" + clean, "XDG_CONFIG_HOME=" + clean, "PATH=/nonexistent"}
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "0 fail") {
		t.Fatalf("staged transport tests: %v\n%s", err, output)
	}
	for _, evidence := range []string{"named fetch loopback decodes exact JSON", "named fetch raw fixtures handle bodyless", "native fetch makes one attempt", "stalled native response", "response decode past", "transport failures construct exact", "native continuations finish", "late decoder defect", "native continuation retains captured resource"} {
		if !strings.Contains(string(output), evidence) {
			t.Fatalf("missing transport qualification %q: %s", evidence, output)
		}
	}
}

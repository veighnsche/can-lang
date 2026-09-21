package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func TestBundledOwnershipOffline(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline ownership qualification")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "owner-integration")
	if err != nil {
		t.Fatal(err)
	}
	clean := t.TempDir()
	command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), "test", filepath.Join(bundle, "runtime/test/owner.test.ts"), filepath.Join(bundle, "runtime/test/owner-roots.test.ts"), filepath.Join(bundle, "runtime/test/owner-hung.test.ts"))
	command.Dir = clean
	command.Env = []string{"HOME=" + clean, "XDG_CONFIG_HOME=" + clean, "PATH=/nonexistent"}
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "0 fail") {
		t.Fatalf("staged ownership tests: %v\n%s", err, output)
	}
	for _, evidence := range []string{"CLI waits for late losers", "assertion result waits", "external supervisor kills a nonsettling owner (leased=false)", "external supervisor kills a nonsettling owner (leased=true)", "idempotent reentrant close", "host callback wrappers preserve"} {
		if !strings.Contains(string(output), evidence) {
			t.Fatalf("missing ownership qualification %q: %s", evidence, output)
		}
	}
}

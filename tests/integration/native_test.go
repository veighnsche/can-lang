package integration

import (
	"bytes"
	"context"
	"errors"
	"github.com/veighnsche/can-lang/distribution"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCurrentBundledNativeDeclarations(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline native declaration checks")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "native-integration")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	run := func(command string, args ...string) (int, string, string) {
		t.Helper()
		argv := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root}
		argv = append(argv, args...)
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), out.String(), diag.String()
		}
		return 0, out.String(), diag.String()
	}
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/native/declarations.can"))
	if err != nil {
		t.Fatal(err)
	}
	write("src/main.can", string(data))
	status, out, diag := run("build")
	if status != 0 || diag != "" {
		t.Fatalf("native frontend: %d %s %s", status, out, diag)
	}
	for _, bad := range []string{
		strings.Replace(string(data), "question(\"Second\")", "question(first)", 1),
		strings.Replace(string(data), "model \"jev-latest\"", "unknown \"jev-latest\"", 1),
		strings.Replace(string(data), "ok false", "ok 1", 1),
	} {
		write("src/main.can", bad)
		status, _, diag = run("build")
		if status == 0 || diag == "" {
			t.Fatalf("invalid native declaration accepted: %d %s", status, diag)
		}
	}
}

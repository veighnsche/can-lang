package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/veighnsche/can-lang/distribution"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCurrentBundledUtilities(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "utilities-integration")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/cli/utilities.can"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	write("src/main.can", source)
	run := func(command string) (int, string, string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir}
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				return e.ExitCode(), out.String(), diag.String()
			}
			t.Fatal(err)
		}
		return 0, out.String(), diag.String()
	}
	if code, out, diag := run("assert"); code != 0 || diag != "" || !strings.Contains(out, "supplied-completion") || strings.Contains(out, "secret") {
		t.Fatalf("utility fixtures: %d %s %s", code, out, diag)
	}
	expected := "{\"level\":\"info\",\"message\":\"secret info\"}\n{\"level\":\"error\",\"message\":\"secret error\"}\n"
	if code, out, diag := run("run"); code != 0 || out != "" || diag != expected {
		t.Fatalf("native utilities: %d %q %q", code, out, diag)
	}
	for _, tc := range []struct{ from, to, id string }{{"sleep(0)", "sleep(-1)", "1260"}, {"sleep(0)", "sleep(2147483648)", "1260"}, {"secure_bytes(0)", "secure_bytes(-1)", "1261"}, {"secure_bytes(0)", "secure_bytes(65537)", "1261"}} {
		write("src/main.can", strings.Replace(source, tc.from, tc.to, 1))
		if code, out, diag := run("run"); code != 1 || out != "" || !strings.Contains(diag, `"id":`+tc.id) || strings.Contains(diag, "secret") {
			t.Fatalf("utility bound: %d %q %s", code, out, diag)
		}
	}
	write("src/main.can", source)
	code, out, diag := run("build")
	if code != 0 {
		t.Fatalf("build: %d %s %s", code, out, diag)
	}
	var build struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal([]byte(out), &build); err != nil {
		t.Fatal(err)
	}
	if tsc := os.Getenv("CAN_TSC"); tsc != "" {
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(build.Directory, "entry.ts"))
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("strict generated TypeScript: %v %s", err, result)
		}
	}
}

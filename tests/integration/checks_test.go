package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/veighnsche/can-lang/distribution"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCurrentBundledChecks(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline checks execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "checks-integration")
	if err != nil {
		t.Fatal(err)
	}
	qualified := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), "test", filepath.Join(bundle, "runtime/test/checks.test.ts"))
	qualified.Dir = t.TempDir()
	qualified.Env = []string{"PATH=/nonexistent", "HOME=" + qualified.Dir, "XDG_CONFIG_HOME=" + qualified.Dir}
	if output, err := qualified.CombinedOutput(); err != nil || !strings.Contains(string(output), "5 pass") || !strings.Contains(string(output), "0 fail") {
		t.Fatalf("offline checks traces: %v\n%s", err, output)
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

	for _, fixture := range []struct {
		name string
		rows int
	}{{"main", 7}} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/checks", fixture.name+".can"))
		if err != nil {
			t.Fatal(err)
		}
		write("src/main.can", string(data))
		status, out, diag := run("assert")
		if status != 0 || diag != "" {
			t.Fatalf("checks assertions: %d %s %s", status, out, diag)
		}
		var report map[string]any
		if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != fixture.rows {
			t.Fatalf("invalid checks report %v %s", err, out)
		}
		status, out, diag = run("run")
		if status != 0 || out != "" || diag != "" {
			t.Fatalf("checks execution: %d %s %s", status, out, diag)
		}
		if compiler := os.Getenv("CAN_TSC"); compiler != "" {
			nodeModules := filepath.Dir(filepath.Dir(filepath.Dir(compiler)))
			args := []string{compiler, "--noEmit", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(nodeModules, "@types"), "--types", "bun,node"}
			err := filepath.WalkDir(filepath.Join(root, "dist"), func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.IsDir() && strings.HasSuffix(path, ".ts") {
					args = append(args, path)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.CommandContext(ctx, os.Getenv("CAN_BUN"), args...)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("strict generated TypeScript: %v\n%s", err, output)
			}
		}

	}
}

// A require over a real decoded value reports its explicit reason call site:
// the outcome-mismatch frame covers the exact require expression, executed as
// real code with no supplied completions and no private detail disclosure.
func TestChecksMismatchSpan(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for checks diagnostic qualification")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "checks-mismatch")
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
	text := "package app\n    provides []\n    uses [bytes, codec, checks]\nfn void guarded\n    emits [checks::failed, codec::invalid_data]\n    asserts\n        sample: => ok\n    match call bytes::from_utf8(\"A\")\n        codec::invalid_data\n        ok bytes::buffer raw => do\n            int[] values = call bytes::to_ints(raw)\n            relay call checks::require(values is [66], \"explicit condition reason\")\nfn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"
	write("src/main.can", text)
	expression := `checks::require(values is [66], "explicit condition reason")`
	command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "assert", root)
	command.Dir = t.TempDir()
	command.Env = []string{"PATH=/nonexistent", "HOME=" + command.Dir}
	var out, diag bytes.Buffer
	command.Stdout = &out
	command.Stderr = &diag
	err = command.Run()
	var status *exec.ExitError
	if !errors.As(err, &status) || status.ExitCode() != 1 || diag.Len() != 0 {
		t.Fatalf("status/diagnostic: %v %s %s", err, out.String(), diag.String())
	}
	var report map[string]any
	if err = json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err, out.String())
	}
	if report["passed"] != false {
		t.Fatal(report)
	}
	var failed map[string]any
	for _, raw := range report["assertions"].([]any) {
		test := raw.(map[string]any)
		if test["root"].(map[string]any)["name"] == "sample" {
			failed = test
		}
	}
	if failed == nil || failed["reason"] != "outcome mismatch" {
		t.Fatal(report)
	}
	labels := []string{}
	for _, raw := range failed["evidence"].([]any) {
		labels = append(labels, raw.(string))
	}
	if len(labels) != 1 || labels[0] != "real-can" {
		t.Fatalf("checks evidence is not real execution: %v", labels)
	}
	start := strings.Index(text, expression)
	end := start + len(expression)
	line := strings.Count(text[:start], "\n") + 1
	column := start - strings.LastIndex(text[:start], "\n")
	found := false
	for _, raw := range failed["frames"].([]any) {
		frame := raw.(map[string]any)
		if frame["file"] == "main.can" && frame["start"] == float64(start) && frame["end"] == float64(end) && frame["line"] == float64(line) && frame["column"] == float64(column) {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing exact require span [%d,%d): %s", start, end, out.String())
	}
	for _, secret := range []string{root, bundle} {
		if strings.Contains(out.String(), secret) {
			t.Fatal("disclosed private detail", secret)
		}
	}
}

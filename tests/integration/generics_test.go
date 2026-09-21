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

func TestCurrentBundledGenerics(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline generic execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "generic-integration")
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

	for _, name := range []string{"definitions.can", "helper.can", "main.can"} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/generics", name))
		if err != nil {
			t.Fatal(err)
		}
		write("src/"+name, string(data))
	}
	status, out, diag := run("assert")
	if status != 0 || diag != "" {
		t.Fatalf("generic assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 6 {
		t.Fatalf("invalid generic report: %v %s", err, out)
	}
	status, out, diag = run("run")
	if status != 0 || out != "" || diag != "" {
		t.Fatalf("generic execution: %d %s %s", status, out, diag)
	}
	// A reachable instance must check a branch that this invocation never takes.
	data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/generics/definitions.can"))
	if err != nil {
		t.Fatal(err)
	}
	invalid := strings.Replace(string(data), "    ok value\n", "    match true\n        true => ok value\n        false => ok false\n", 1)
	write("src/definitions.can", invalid)
	status, out, diag = run("build")
	if status == 0 || !strings.Contains(diag, "concrete function") {
		t.Fatalf("invalid unused branch admitted: %d %s %s", status, out, diag)
	}
	for _, name := range []string{"definitions.can", "helper.can"} {
		if err := os.Remove(filepath.Join(root, "src", name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"finite-transition", "finite-field-chain", "spread-inferred"} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/generics", name+".can"))
		if err != nil {
			t.Fatal(err)
		}
		variants := []string{string(data)}
		if name == "finite-transition" {
			variants = append(variants, strings.Replace(string(data), "        sample: 1, 0 => ok 0", "        sample: 1, 0 => ok 0\n        array: [1], 0 => ok 0", 1))
			variants = append(variants, strings.Replace(string(data), "fixed<int[]>([1]", "fixed([1]", 1))
		}
		if name == "finite-field-chain" {
			row := "        sample: seed([]), 0 => ok 0\n"
			terminal := "        terminal: done<step<seed>>([]), 0 => ok 0\n"
			variants = append(variants, strings.Replace(string(data), row, terminal+row, 1), strings.Replace(string(data), row, row+terminal, 1))
		}
		for index, source := range variants {
			write("src/main.can", source)
			status, out, diag = run("assert")
			if status != 0 || diag != "" || !strings.Contains(out, `"passed":true`) {
				t.Fatalf("%s variant %d assertions: %d %s %s", name, index, status, out, diag)
			}
			status, out, diag = run("run")
			if status != 0 || out != "" || diag != "" {
				t.Fatalf("%s variant %d execution: %d %s %s", name, index, status, out, diag)
			}
		}
	}

}

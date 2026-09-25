package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const formatMessy = `// order totals with tip


package app
    provides [tally]
    uses []

// total with tip
fn   int   tally
    emits []
    given
        int seed  // per cover
    asserts
        sample: 1 => ok 2
        pair: 3 => ok 6
    ok seed + seed  // doubled
`

// Formatting preserves runtime behavior: the same roots pass before and
// after, the write is idempotent, and invalid source stays untouched.
func TestFormatPreservesExecution(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged format execution")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
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
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", formatMessy)

	run := func(args ...string) (int, string, string) {
		t.Helper()
		cmd := exec.Command(filepath.Join(bundle, "bin/canlc"), args...)
		cmd.Dir = root
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		return code, stdout.String(), stderr.String()
	}
	roots := func(t *testing.T, out string) []string {
		t.Helper()
		var report struct {
			Passed     bool `json:"passed"`
			Assertions []struct {
				Passed bool `json:"passed"`
				Root   struct {
					Package     string `json:"package"`
					Declaration string `json:"declaration"`
					Name        string `json:"name"`
				} `json:"root"`
			} `json:"assertions"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("decode %v: %s", err, out)
		}
		if !report.Passed {
			t.Fatalf("assert failed: %s", out)
		}
		var names []string
		for _, entry := range report.Assertions {
			if !entry.Passed {
				t.Fatalf("root failed: %+v", entry)
			}
			names = append(names, entry.Root.Package+"/"+entry.Root.Declaration+"#"+entry.Root.Name)
		}
		return names
	}

	code, out, diag := run("assert", root)
	if code != 0 || diag != "" {
		t.Fatalf("assert before: %d %s %s", code, out, diag)
	}
	beforeRoots := roots(t, out)
	if code, _, diag := run("format", "--write", filepath.Join(root, "src/main.can")); code != 0 {
		t.Fatalf("format: %d %s", code, diag)
	}
	after, err := os.ReadFile(filepath.Join(root, "src/main.can"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) == formatMessy {
		t.Fatal("format left messy source unchanged")
	}
	for _, want := range []string{"fn int tally\n", "// total with tip\n", "int seed  // per cover\n"} {
		if !strings.Contains(string(after), want) {
			t.Fatalf("formatted output omits %q:\n%s", want, after)
		}
	}
	if code, out, diag := run("assert", root); code != 0 || diag != "" {
		t.Fatalf("assert after: %d %s %s", code, out, diag)
	} else if afterRoots := roots(t, out); strings.Join(afterRoots, ",") != strings.Join(beforeRoots, ",") {
		t.Fatalf("roots changed: %v vs %v", beforeRoots, afterRoots)
	}
	if code, _, diag := run("format", "--write", filepath.Join(root, "src/main.can")); code != 0 {
		t.Fatalf("second format: %d %s", code, diag)
	}
	if second, err := os.ReadFile(filepath.Join(root, "src/main.can")); err != nil || string(second) != string(after) {
		t.Fatal("format is not idempotent through the CLI")
	}
	write("src/main.can", formatMessy+"fn int broken(\n")
	if code, _, _ := run("format", "--write", filepath.Join(root, "src/main.can")); code != 1 {
		t.Fatalf("invalid format exit %d, want 1", code)
	}
	if broken, err := os.ReadFile(filepath.Join(root, "src/main.can")); err != nil || !strings.HasSuffix(string(broken), "fn int broken(\n") {
		t.Fatal("invalid source was touched")
	}
}

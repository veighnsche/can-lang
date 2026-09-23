package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

func TestCurrentBundledCLI(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged offline CLI integration")
	}
	source, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, source, t.TempDir(), archive, "cli-integration")
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(bundle, "bin/canlc")
	outside := t.TempDir()
	link := filepath.Join(outside, "canlc")
	if err = os.Symlink(launcher, link); err != nil {
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
	echo, err := os.ReadFile(filepath.Join(source, "compiler/testdata/current/cli/echo.can"))
	if err != nil {
		t.Fatal(err)
	}
	write("src/main.can", string(echo))
	run := func(args ...string) (int, string, string) {
		t.Helper()
		command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", append([]string{"-p", "(version 1)(allow default)(deny network*)", link}, args...)...)
		command.Dir = outside
		command.Env = []string{"PATH=/nonexistent", "HOME=" + outside, "BUN_OPTIONS=--preload=/missing/injected.ts"}
		var stdout, stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		err := command.Run()
		if err == nil {
			return 0, stdout.String(), stderr.String()
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode(), stdout.String(), stderr.String()
		}
		t.Fatal(err)
		return 0, "", ""
	}
	for _, arg := range []string{"héllo 😀\n", "--inspect", "--", "-e", "--preload=/missing.ts", "", "\ufeffBOM", "space argument"} {
		code, out, diag := run("run", root, "--", arg)
		if code != 0 || out != arg || diag != "" {
			t.Fatalf("argv/stdout %q: status=%d out=%q diagnostic=%q", arg, code, out, diag)
		}
	}
	// The OS pipe has no reader before the child starts, so the awaited native
	// write must report the catalogue error rather than successful completion.
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()
	broken := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", link, "run", root, "--", "cannot write")
	broken.Dir = outside
	broken.Env = []string{"PATH=/nonexistent", "HOME=" + outside}
	broken.Stdout = writer
	var writeDiagnostic bytes.Buffer
	broken.Stderr = &writeDiagnostic
	writeErr := broken.Run()
	writer.Close()
	var writeExit *exec.ExitError
	if !errors.As(writeErr, &writeExit) || writeExit.ExitCode() != 1 || !strings.Contains(writeDiagnostic.String(), `"id":1211`) {
		t.Fatalf("native broken-pipe mapping: %v %q", writeErr, writeDiagnostic.String())
	}
	write("src/helper.can", "package app\n    provides []\n    uses []\nstr suffix = later\nstr later = \"!\"\nfn str decorate\n    emits []\n    given\n        str text\n    asserts\n        sample: \"hello\" => ok \"hello!\"\n    ok text + suffix\n")
	write("src/main.can", strings.Replace(string(echo), "args[0]", "call decorate(args[0])", 1))
	if status, out, diag := run("run", root, "--", "multi-file"); status != 0 || out != "multi-file!" || diag != "" {
		t.Fatalf("cross-module call and forward initializer: %d %q %q", status, out, diag)
	}
	write("src/main.can", string(echo))
	code, out, diag := run("build", root)
	if code != 0 || diag != "" {
		t.Fatalf("build: %d %s %s", code, out, diag)
	}
	var report struct {
		SchemaVersion int    `json:"schemaVersion"`
		Kind          string `json:"kind"`
		BuildID       string `json:"buildID"`
		Directory     string `json:"directory"`
		Entry         string `json:"entry"`
	}
	if err = json.Unmarshal([]byte(out), &report); err != nil || report.SchemaVersion != 1 || report.Kind != "can.build" || len(report.BuildID) != 64 || report.Entry != "entry.ts" {
		t.Fatalf("invalid build report: %s", out)
	}
	before, err := os.ReadFile(filepath.Join(root, "dist/current.json"))
	if err != nil {
		t.Fatal(err)
	}
	write("src/main.can", strings.Replace(string(echo), "str[] args", "int args", 1))
	code, out, diag = run("run", root)
	if code != 1 || out != "" || !strings.Contains(diag, "entry must") {
		t.Fatalf("bad entry: %d %q %q", code, out, diag)
	}
	after, _ := os.ReadFile(filepath.Join(root, "dist/current.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("static failure replaced current generation")
	}
	write("src/main.can", strings.Replace(string(echo), "args[0]", "true", 1))
	code, out, diag = run("build", root)
	if code != 1 || out != "" || diag == "" {
		t.Fatalf("static mismatch: %d %q %q", code, out, diag)
	}
	write("src/main.can", string(echo))
	code, out, diag = run("run", root)
	if code != 1 || out != "" || !strings.Contains(diag, `"category":"bounds"`) {
		t.Fatalf("runtime bounds: %d %q %q", code, out, diag)
	}
	if strings.Contains(diag, root) || strings.Contains(diag, "stack") {
		t.Fatal("root diagnostic disclosed host metadata")
	}
	// A failing initializer cannot verify: workers fail before any
	// execution, so the gate rejects with no output or publication.
	write("src/main.can", strings.Replace(string(echo), "fn void main", "int startup = 1 / 0\nfn void main", 1))
	code, out, diag = run("run", root, "--", "must not print")
	if code != 1 || out != "" || !strings.Contains(diag, "build verification failed") || !strings.Contains(diag, "initialization failed") {
		t.Fatalf("startup failure: %d %q %q", code, out, diag)
	}
	write("can.errors.json", `{"active":[{"id":1000000,"kind":"app::failed"}],"retired":[]}`)
	write("src/main.can", "package app\n    provides []\n    uses []\nerror 1000000 failed(str secret)\nfn void main\n    emits [failed]\n    given\n        str[] argv\n    asserts\n        sample: [] => failed(\"must-not-disclose\")\n    failed(\"must-not-disclose\")\n")
	code, out, diag = run("run", root)
	if code != 1 || out != "" || !strings.Contains(diag, `"id":1000000`) || strings.Contains(diag, "must-not-disclose") {
		t.Fatalf("domain failure: %d %q %q", code, out, diag)
	}
	for _, args := range [][]string{{"build"}, {"run", root, "--inspect"}, {"build", root, "extra"}} {
		code, _, diag = run(args...)
		if code != 2 || !strings.Contains(diag, "usage:") {
			t.Fatalf("usage: %v %d %q", args, code, diag)
		}
	}
}

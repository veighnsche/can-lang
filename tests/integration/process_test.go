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

const processLiveProgram = `package process_live
    provides [main]
    uses [bytes, codec, files, io, process]
fn void main
    emits [files::not_found, files::denied, process::spawn_failed, process::timeout, process::output_limit, process::invalid_config, process::io_error, codec::invalid_data, io::write_failed]
    given
        str[] args
    asserts
        sample: [] => ok
    bytes::buffer empty = call bytes::empty()
    process::result done = process::result(empty, empty, 0, "")
    match call process::run("/bin/echo", ["live-proc"], process::options("", true, [], empty, 100, 100, 5000, 100))
        when
            sample: "/bin/echo", ["live-proc"], process::options("", true, [], empty, 100, 100, 5000, 100) => ok done
        files::not_found
        files::denied
        process::spawn_failed
        process::timeout
        process::output_limit
        process::invalid_config
        process::io_error
        ok process::result got => match call bytes::to_utf8(got.stdout)
            codec::invalid_data
            ok str text => match call bytes::from_utf8(text)
                codec::invalid_data
                ok bytes::buffer message => match call io::stdout_write(message)
                    when
                        sample: message => ok 0
                    io::write_failed
                    ok int written => ok
`

func TestCurrentBundledProcess(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline process execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "process-integration")
	if err != nil {
		t.Fatal(err)
	}
	// Staged runtime suite against owned child processes.
	qualified := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), "test", filepath.Join(bundle, "runtime/test/process.test.ts"))
	qualified.Dir = t.TempDir()
	// Process tests resolve bare names through PATH; pin it to the system
	// directories instead of isolating it away.
	qualified.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + qualified.Dir, "XDG_CONFIG_HOME=" + qualified.Dir}
	if output, err := qualified.CombinedOutput(); err != nil || !strings.Contains(string(output), "13 pass") || !strings.Contains(string(output), "0 fail") {
		t.Fatalf("offline process traces: %v\n%s", err, output)
	}
	stage := func(program string) string {
		t.Helper()
		root, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		write := func(name, text string) {
			t.Helper()
			p := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
		}
		write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
		write("can.errors.json", `{"active":[],"retired":[]}`)
		write("src/main.can", program)
		return root
	}
	run := func(command, root, dir string, args ...string) (int, string, string) {
		t.Helper()
		argv := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root}
		argv = append(argv, args...)
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = dir
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + dir}
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
	// Live program: assertions use fixtures, the run spawns /bin/echo.
	live := stage(processLiveProgram)
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	status, out, diag := run("assert", live, home)
	if status != 0 || diag != "" {
		t.Fatalf("process assertions: %d %s %s", status, out, diag)
	}
	var report struct {
		Passed     bool  `json:"passed"`
		Assertions []any `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed || len(report.Assertions) != 1 {
		t.Fatalf("invalid process report %v %s", err, out)
	}
	// The staged run inherits a resolved PATH so the child resolves /bin/echo.
	status, out, diag = run("run", live, home)
	if status != 0 || out != "live-proc\n" || diag != "" {
		t.Fatalf("process execution: %d %q %s", status, out, diag)
	}
	// Rejected programs carry diagnostics with source spans.
	for _, fixture := range []struct {
		name     string
		program  string
		fragment []string
	}{
		{"wrong-type", "package neg\n    provides [main]\n    uses [process]\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        run: [] => ok\n    match call process::which(123)\n        ok str found => ok\n",
			[]string{"main.can", "expression type does not fit expected type"}},
		{"unhandled-error", "package neg\n    provides [main]\n    uses [bytes, process]\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        run: [] => ok\n    bytes::buffer empty = call bytes::empty()\n    match call process::run(\"/bin/echo\", [], process::options(\"\", true, [], empty, 10, 10, 0, 0))\n        ok process::result got => ok\n",
			[]string{"main.can", "missing completion arm", "can.std.process@1::run requires an arm for each bound member"}},
		{"unknown-operation", "package neg\n    provides [main]\n    uses [process]\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        run: [] => ok\n    match call process::dance(\"x\")\n        ok str done => ok\n",
			[]string{"main.can", "no eligible call process::dance"}},
	} {
		root := stage(fixture.program)
		status, out, diag := run("assert", root, home)
		if status == 0 {
			t.Fatalf("%s unexpectedly accepted: %s %s", fixture.name, out, diag)
		}
		for _, fragment := range fixture.fragment {
			if !strings.Contains(out, fragment) && !strings.Contains(diag, fragment) {
				t.Fatalf("%s diagnostic lacks %q: %s %s", fixture.name, fragment, out, diag)
			}
		}
	}
	// Generated TypeScript lowers to native process operations.
	_, dir := applicationBuild(t, ctx, bundle, home, live)
	lowered := map[string]bool{}
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"$canProcesses.run", "$canCreateProcesses", "Bun.spawn(", "Bun.which", "process.kill", "detached:true"} {
			if strings.Contains(string(data), want) {
				lowered[want] = true
			}
		}
		return nil
	})
	for _, want := range []string{"$canProcesses.run", "$canCreateProcesses", "Bun.spawn(", "Bun.which", "process.kill", "detached:true"} {
		if !lowered[want] {
			t.Fatalf("generated output lacks native lowering %q", want)
		}
	}
}

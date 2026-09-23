package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const filesLiveProgramTmpl = `package files_live
    provides [main]
    uses [bytes, codec, files, io, path, text]
fn void main
    emits [files::not_found, files::already_exists, files::denied, files::invalid_path, files::unexpected_kind, files::limit_exceeded, files::io_error, codec::invalid_data, io::write_failed]
    given
        str[] args
    asserts
        sample: [%s] => ok
    match call path::join([args[0], "work"])
        ok str dir => match call files::mkdir(dir, true)
            when
                sample: dir, true => ok
            files::not_found
            files::already_exists
            files::denied
            files::invalid_path
            files::io_error
            ok => match call path::join([dir, "note.txt"])
                ok str note => match call files::write_text(note, "live-data", true)
                    when
                        sample: note, "live-data", true => ok
                    files::not_found
                    files::already_exists
                    files::denied
                    files::invalid_path
                    files::io_error
                    ok => match call files::read_text(note, 100)
                        when
                            sample: note, 100 => ok "live-data"
                        files::not_found
                        files::denied
                        files::invalid_path
                        files::unexpected_kind
                        files::limit_exceeded
                        codec::invalid_data
                        files::io_error
                        ok str back => match call bytes::from_utf8(back)
                            codec::invalid_data
                            ok bytes::buffer message => match call io::stdout_write(message)
                                when
                                    sample: message => ok 9
                                io::write_failed
                                ok int written => ok
`

func TestCurrentBundledFiles(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline files execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "files-integration")
	if err != nil {
		t.Fatal(err)
	}
	// Staged runtime suite against real temporary trees.
	qualified := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), "test", filepath.Join(bundle, "runtime/test/files.test.ts"))
	qualified.Dir = t.TempDir()
	qualified.Env = []string{"PATH=/nonexistent", "HOME=" + qualified.Dir, "XDG_CONFIG_HOME=" + qualified.Dir}
	if output, err := qualified.CombinedOutput(); err != nil || !strings.Contains(string(output), "15 pass") || !strings.Contains(string(output), "0 fail") {
		t.Fatalf("offline files traces: %v\n%s", err, output)
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
	// Live program: assertions use fixtures; the run writes a real tree
	// under the directory passed as its argument.
	work, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	live := stage(fmt.Sprintf(filesLiveProgramTmpl, strconv.Quote(work)))
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	status, out, diag := run("assert", live, home)
	if status != 0 || diag != "" {
		t.Fatalf("files assertions: %d %s %s", status, out, diag)
	}
	var report struct {
		Passed     bool  `json:"passed"`
		Assertions []any `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed || len(report.Assertions) != 1 {
		t.Fatalf("invalid files report %v %s", err, out)
	}
	status, out, diag = run("run", live, home, "--", work)
	if status != 0 || out != "live-data" || diag != "" {
		t.Fatalf("files execution: %d %q %s", status, out, diag)
	}
	if data, err := os.ReadFile(filepath.Join(work, "work", "note.txt")); err != nil || string(data) != "live-data" {
		t.Fatalf("live write missing on disk: %v %q", err, data)
	}
	// Rejected programs carry diagnostics with source spans.
	for _, fixture := range []struct {
		name     string
		program  string
		fragment []string
	}{
		{"wrong-type", "package neg\n    provides [main]\n    uses [files, bytes]\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        run: [] => ok\n    match call files::read_bytes(123, 10)\n        ok bytes::buffer data => ok\n",
			[]string{"main.can", "expression type does not fit expected type"}},
		{"unhandled-error", "package neg\n    provides [main]\n    uses [files, bytes]\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        run: [] => ok\n    match call files::read_bytes(\"a.txt\", 10)\n        ok bytes::buffer data => ok\n",
			[]string{"main.can", "missing completion arm", "can.std.files@1::read_bytes requires an arm for each bound member"}},
		{"unknown-operation", "package neg\n    provides [main]\n    uses [files]\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        run: [] => ok\n    match call files::dance(\"a.txt\")\n        ok str done => ok\n",
			[]string{"main.can", "no eligible call files::dance"}},
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
	// Generated TypeScript lowers to native filesystem operations.
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
		for _, want := range []string{"$canFileWrites.writeText", "$canFileReads.readText", "$canPath.joinPath", "$canCreateFileReads", "Bun.file(", "new Bun.Glob", "node:fs/promises", "node:path"} {
			if strings.Contains(string(data), want) {
				lowered[want] = true
			}
		}
		return nil
	})
	for _, want := range []string{"$canFileWrites.writeText", "$canFileReads.readText", "$canPath.joinPath", "$canCreateFileReads", "Bun.file(", "new Bun.Glob", "node:fs/promises", "node:path"} {
		if !lowered[want] {
			t.Fatalf("generated output lacks native lowering %q", want)
		}
	}
}

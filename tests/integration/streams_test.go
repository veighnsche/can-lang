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
)

const streamsPumpProgram = `package streams_pump
    provides [pump, main]
    uses [stream, files]
fn int pump
    emits [stream::read_failed, stream::cancelled, files::limit_exceeded, stream::close_failed]
    given
        int count
        stream::reader<str> input
    asserts
        end: 5 => ok 5
        more: 0 => ok 2
    match call stream::read_many(input, 2)
        when
            end: input, 2 => ok []
            more: input, 2 => ok ["a", "b"]
            more: input, 2 => ok []
        stream::read_failed
        stream::cancelled
        files::limit_exceeded
        ok str[] batch => match batch.length is 0
            false => match call pump(count + batch.length, input)
                stream::read_failed
                stream::cancelled
                files::limit_exceeded
                stream::close_failed
                ok int total => ok total
            true => match call stream::close_reader(input)
                when
                    end: input => ok
                    more: input => ok
                stream::close_failed
                ok => ok count
fn void main
    emits []
    given
        str[] args
    asserts
        sample: [] => ok
    ok
`

const streamsLiveProgramTmpl = `package streams_live
    provides [main]
    uses [bytes, codec, files, io, path, stream, text]
fn void main
    emits [files::not_found, files::already_exists, files::denied, files::invalid_path, files::unexpected_kind, files::limit_exceeded, files::io_error, stream::read_failed, stream::cancelled, stream::close_failed, stream::write_failed, codec::invalid_data, io::write_failed]
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
                ok str note => match call files::write_stream(note)
                    when
                        sample: note => ok
                    files::not_found
                    files::denied
                    files::invalid_path
                    files::unexpected_kind
                    files::io_error
                    ok stream::writer out => match call bytes::from_utf8("l1\nl2\n")
                        codec::invalid_data
                        ok bytes::buffer payload => match call stream::write_some(out, payload)
                            when
                                sample: out, payload => ok 6
                            stream::write_failed
                            ok int accepted => match call stream::close_writer(out)
                                when
                                    sample: out => ok
                                stream::close_failed
                                ok => match call files::read_lines_stream(note, 64)
                                    when
                                        sample: note, 64 => ok
                                    files::not_found
                                    files::denied
                                    files::invalid_path
                                    files::unexpected_kind
                                    files::limit_exceeded
                                    files::io_error
                                    ok stream::reader<str> input => match call stream::read_many(input, 5)
                                        when
                                            sample: input, 5 => ok ["l1", "l2"]
                                        stream::read_failed
                                        stream::cancelled
                                        files::limit_exceeded
                                        ok str[] first => match call stream::read_many(input, 5)
                                            when
                                                sample: input, 5 => ok []
                                            stream::read_failed
                                            stream::cancelled
                                            files::limit_exceeded
                                            ok str[] rest => match call stream::close_reader(input)
                                                when
                                                    sample: input => ok
                                                stream::close_failed
                                                ok => match chain
                                                    call bytes::from_utf8(call text::from_int(first.length + rest.length)) as bytes::buffer message
                                                    codec::invalid_data
                                                    ok => match call io::stdout_write(message)
                                                        when
                                                            sample: message => ok 1
                                                        io::write_failed
                                                        ok int written => ok
`

const streamsOverrunProgram = `package streams_over
    provides [drain, main]
    uses [stream, files]
fn void drain
    emits [stream::read_failed, stream::cancelled, files::limit_exceeded, stream::close_failed]
    given
        stream::reader<str> input
    asserts
        sample: => ok
    match call stream::read_many(input, 2)
        when
            sample: input, 2 => ok ["a"]
            sample: input, 2 => ok ["b"]
        stream::read_failed
        stream::cancelled
        files::limit_exceeded
        ok str[] batch => match call stream::close_reader(input)
            when
                sample: input => ok
            stream::close_failed
            ok => ok
fn void main
    emits []
    given
        str[] args
    asserts
        sample: [] => ok
    ok
`

const streamsUnderrunProgram = `package streams_under
    provides [pump, main]
    uses [stream, files]
fn int pump
    emits [stream::read_failed, stream::cancelled, files::limit_exceeded, stream::close_failed]
    given
        int count
        stream::reader<str> input
    asserts
        more: 0 => ok 2
    match call stream::read_many(input, 2)
        when
            more: input, 2 => ok ["a", "b"]
        stream::read_failed
        stream::cancelled
        files::limit_exceeded
        ok str[] batch => match batch.length is 0
            false => match call pump(count + batch.length, input)
                stream::read_failed
                stream::cancelled
                files::limit_exceeded
                stream::close_failed
                ok int total => ok total
            true => match call stream::close_reader(input)
                stream::close_failed
                ok => ok count
fn void main
    emits []
    given
        str[] args
    asserts
        sample: [] => ok
    ok
`

func TestCurrentBundledStreams(t *testing.T) {
	t.Parallel()
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline streams execution")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	bundle, err := harnessBundle(t, ctx, archive)
	if err != nil {
		t.Fatal(err)
	}
	// Staged runtime suite against real temporary trees and synthetic sources.
	qualified := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), "test", filepath.Join(bundle, "runtime/test/streams.test.ts"))
	qualified.Dir = t.TempDir()
	qualified.Env = []string{"PATH=/nonexistent", "HOME=" + qualified.Dir, "XDG_CONFIG_HOME=" + qualified.Dir}
	if output, err := qualified.CombinedOutput(); err != nil || !strings.Contains(string(output), "11 pass") || !strings.Contains(string(output), "0 fail") {
		t.Fatalf("offline streams traces: %v\n%s", err, output)
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
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Pull-loop transcripts: repeated selectors replay FIFO across recursion.
	pump := stage(streamsPumpProgram)
	status, out, diag := run("assert", pump, home)
	if status != 0 || diag != "" {
		t.Fatalf("streams pump assertions: %d %s %s", status, out, diag)
	}
	var pumpReport struct {
		Passed     bool  `json:"passed"`
		Assertions []any `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &pumpReport); err != nil || !pumpReport.Passed || len(pumpReport.Assertions) != 3 {
		t.Fatalf("invalid pump report %v %s", err, out)
	}
	// Live program: assertions use fixtures; the run streams a real tree
	// under the directory passed as its argument.
	work, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	live := stage(fmt.Sprintf(streamsLiveProgramTmpl, strconv.Quote(work)))
	status, out, diag = run("assert", live, home)
	if status != 0 || diag != "" {
		t.Fatalf("streams assertions: %d %s %s", status, out, diag)
	}
	var report struct {
		Passed     bool  `json:"passed"`
		Assertions []any `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil || !report.Passed || len(report.Assertions) != 1 {
		t.Fatalf("invalid streams report %v %s", err, out)
	}
	status, out, diag = run("run", live, home, "--", work)
	if status != 0 || out != "2" || diag != "" {
		t.Fatalf("streams execution: %d %q %s", status, out, diag)
	}
	if data, err := os.ReadFile(filepath.Join(work, "work", "note.txt")); err != nil || string(data) != "l1\nl2\n" {
		t.Fatalf("live stream missing on disk: %v %q", err, data)
	}
	// Transcript bounds fail loudly instead of hanging or truncating.
	for _, fixture := range []struct {
		name     string
		program  string
		fragment string
	}{
		{"overrun", streamsOverrunProgram, "unused fixture"},
		{"underrun", streamsUnderrunProgram, "missing fixture"},
	} {
		root := stage(fixture.program)
		status, out, _ := run("assert", root, home)
		if status == 0 || !strings.Contains(out, fixture.fragment) {
			t.Fatalf("%s missing violation: %d %s", fixture.name, status, out)
		}
	}
	// Rejected programs carry diagnostics with source spans.
	for _, fixture := range []struct {
		name     string
		program  string
		fragment []string
	}{
		{"wrong-type", "package neg\n    provides [drain, main]\n    uses [stream, files]\nfn void drain\n    emits [stream::read_failed, stream::cancelled, files::limit_exceeded]\n    asserts\n        run: => ok\n    match call stream::read_many(\"nope\", 1)\n        stream::read_failed\n        stream::cancelled\n        files::limit_exceeded\n        ok str[] batch => ok\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n",
			[]string{"main.can", "inference shape mismatch"}},
		{"unhandled-error", "package neg\n    provides [drain, main]\n    uses [stream, files]\nfn void drain\n    emits [stream::read_failed, stream::cancelled, files::limit_exceeded]\n    given\n        stream::reader<str> input\n    asserts\n        run: => ok\n    match call stream::read_many(input, 1)\n        stream::read_failed\n        ok str[] batch => ok\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n",
			[]string{"main.can", "missing completion arm", "requires an arm for each bound member"}},
		{"result-type", "package neg\n    provides [drain, main]\n    uses [stream, files]\nfn void drain\n    emits [stream::read_failed, stream::cancelled, files::limit_exceeded]\n    given\n        stream::reader<str> input\n    asserts\n        run: => ok\n    match call stream::read_many(input, 1)\n        stream::read_failed\n        stream::cancelled\n        files::limit_exceeded\n        ok int total => ok\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        sample: [] => ok\n    ok\n",
			[]string{"main.can", "completion arm binding type mismatch"}},
		{"unknown-operation", "package neg\n    provides [main]\n    uses [stream]\nfn void main\n    emits []\n    given\n        str[] args\n    asserts\n        run: [] => ok\n    match call stream::dance(\"x\")\n        ok str done => ok\n",
			[]string{"main.can", "no eligible call stream::dance"}},
	} {
		root := stage(fixture.program)
		status, out, diag := run("assert", root, home)
		if status == 0 {
			t.Fatalf("%s unexpectedly accepted: %s %s", fixture.name, out, diag)
		}
		combined := out + diag
		for _, fragment := range fixture.fragment {
			if !strings.Contains(combined, fragment) {
				t.Fatalf("%s missing %q: %s %s", fixture.name, fragment, out, diag)
			}
		}
	}
}

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

func TestCurrentBundledAssertions(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline assertion execution")
	}
	source, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, source, t.TempDir(), archive, "assertion-integration")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
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
	basic, err := os.ReadFile(filepath.Join(source, "compiler/testdata/current/assertions/basic.can"))
	if err != nil {
		t.Fatal(err)
	}
	write("src/main.can", string(basic))
	run := func(selector ...string) (int, string, string) {
		t.Helper()
		args := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "assert", root}
		args = append(args, selector...)
		command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", args...)
		command.Dir = outside
		command.Env = []string{"PATH=/nonexistent", "HOME=" + outside}
		var out, diag bytes.Buffer
		command.Stdout = &out
		command.Stderr = &diag
		if err := command.Run(); err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				return exit.ExitCode(), out.String(), diag.String()
			}
			t.Fatal(err)
		}
		return 0, out.String(), diag.String()
	}
	report := func(text string) map[string]any {
		t.Helper()
		var value map[string]any
		if err := json.Unmarshal([]byte(text), &value); err != nil {
			t.Fatalf("invalid report or leaked application output: %q: %v", text, err)
		}
		if value["kind"] != "can.assertion-report" || value["schemaVersion"] != float64(1) {
			t.Fatal(value)
		}
		return value
	}
	code, out, diag := run()
	if code != 0 || diag != "" {
		t.Fatalf("native assertions: %d %s %s", code, out, diag)
	}
	good := report(out)
	if good["passed"] != true || len(good["assertions"].([]any)) != 3 || !strings.Contains(out, "supplied-completion") {
		t.Fatal(out)
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()
	broken := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "assert", root)
	broken.Dir = outside
	broken.Env = []string{"PATH=/nonexistent", "HOME=" + outside}
	broken.Stdout = writer
	var brokenDiagnostic bytes.Buffer
	broken.Stderr = &brokenDiagnostic
	brokenErr := broken.Run()
	writer.Close()
	var brokenExit *exec.ExitError
	if !errors.As(brokenErr, &brokenExit) || brokenExit.ExitCode() != 1 || brokenDiagnostic.Len() != 0 {
		t.Fatalf("report pipe failure: %v %q", brokenErr, brokenDiagnostic.String())
	}
	code, out, diag = run("can.project.root/arithmetic", "large")
	if code != 0 || diag != "" || len(report(out)["assertions"].([]any)) != 1 {
		t.Fatalf("short selection: %d %s %s", code, out, diag)
	}
	// The same typed body now disagrees with its expected numeric result. A Go
	// evaluator or an expected-result stub cannot supply a passing native report.
	write("src/main.can", strings.Replace(string(basic), "small: 3 => ok 9", "small: 3 => ok 10", 1))
	code, out, diag = run()
	if code != 1 || diag != "" || report(out)["passed"] != false || !strings.Contains(out, "outcome mismatch") {
		t.Fatalf("real result mismatch: %d %s %s", code, out, diag)
	}
	before, _ := os.ReadFile(filepath.Join(root, "dist/current.json"))
	write("src/main.can", strings.Replace(string(basic), "supplied: 3 => ok 7", "supplied: 3 => ok false", 1))
	code, out, diag = run()
	if code != 1 || out != "" || !strings.Contains(diag, "fixture") {
		t.Fatalf("static supplied mismatch: %d %q %q", code, out, diag)
	}
	after, _ := os.ReadFile(filepath.Join(root, "dist/current.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("static assertion error published output")
	}
	write("src/main.can", strings.Replace(string(basic), "large: 9007199254740993", "small: 9007199254740993", 1))
	code, out, diag = run()
	if code != 1 || out != "" || !strings.Contains(diag, "duplicate") {
		t.Fatalf("duplicate root: %d %s %s", code, out, diag)
	}
	write("src/main.can", strings.Replace(string(basic), "supplied: => ok 8", "small: => ok 10", 1))
	code, out, diag = run("can.project.root/arithmetic", "small")
	if code != 1 || out != "" || !strings.Contains(diag, "2 roots") {
		t.Fatalf("ambiguous short selector: %d %s %s", code, out, diag)
	}
	code, out, diag = run("can.project.root/arithmetic", "can.project.root/arithmetic::square", "small")
	if code != 0 || diag != "" || report(out)["passed"] != true {
		t.Fatalf("full root selector: %d %s %s", code, out, diag)
	}
	native, err := os.ReadFile(filepath.Join(source, "compiler/testdata/current/assertions/native-slice.can"))
	if err != nil {
		t.Fatal(err)
	}
	write("src/main.can", string(native))
	code, out, diag = run()
	if code != 0 || diag != "" || report(out)["passed"] != true || len(report(out)["assertions"].([]any)) != 3 || !strings.Contains(out, "supplied-completion") {
		t.Fatalf("native call fixture: %d %s %s", code, out, diag)
	}
	code, out, diag = run("can.project.root/native_fixture", "actual")
	if code != 0 || diag != "" || report(out)["passed"] != true || strings.Contains(out, "supplied-completion") {
		t.Fatalf("unselected native fixture replaced real operation: %d %s %s", code, out, diag)
	}
	write("src/main.can", strings.Replace(string(native), `selected: 1, 3 => ok "fake"`, `selected: 2, 3 => ok "fake"`, 1))
	code, out, diag = run("can.project.root/native_fixture", "selected")
	if code != 1 || diag != "" || report(out)["passed"] != false || !strings.Contains(out, "argument mismatch") {
		t.Fatalf("native fixture arguments were not validated: %d %s %s", code, out, diag)
	}
	queues, err := os.ReadFile(filepath.Join(source, "compiler/testdata/current/assertions/queues.can"))
	if err != nil {
		t.Fatal(err)
	}
	write("src/main.can", string(queues))
	code, out, diag = run()
	if code != 0 || diag != "" || report(out)["passed"] != true || len(report(out)["assertions"].([]any)) != 22 {
		t.Fatalf("shared fixture queues: %d %s %s", code, out, diag)
	}
	if compiler := os.Getenv("CAN_TSC"); compiler != "" {
		nodeModules := filepath.Dir(filepath.Dir(filepath.Dir(compiler)))
		args := []string{compiler, "--noEmit", "--ignoreConfig", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(nodeModules, "@types"), "--types", "bun,node"}
		if err := filepath.WalkDir(filepath.Join(root, "dist"), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.HasSuffix(path, ".ts") {
				args = append(args, path)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.CommandContext(ctx, os.Getenv("CAN_BUN"), args...).CombinedOutput(); err != nil {
			t.Fatalf("strict assertion TypeScript: %v\n%s", err, output)
		}
	}
	write("src/main.can", strings.Replace(string(queues), "sample: 0 => ok 10", "sample: 1 => ok 10", 1))
	code, out, diag = run("can.project.root/queues", "can.project.root/queues::captured_pair", "sample")
	if code != 1 || diag != "" || !strings.Contains(out, "argument mismatch") || !strings.Contains(out, `"callable":`) || !strings.Contains(out, `"instance":`) || !strings.Contains(out, `"fixturePaths":`) {
		t.Fatalf("callable fixture mismatch lacks creation path: %d %s %s", code, out, diag)
	}
	write("src/main.can", strings.Replace(string(queues), "            sample: 1 => ok 20\n", "", 1))
	code, out, diag = run("can.project.root/queues", "can.project.root/queues::repeated", "sample")
	if code != 1 || diag != "" || !strings.Contains(out, "missing fixture") || !strings.Contains(out, `"row":1`) {
		t.Fatalf("shared fixture exhaustion: %d %s %s", code, out, diag)
	}
	write("src/main.can", strings.Replace(string(queues), "            sample: 1 => ok 20\n", "            sample: 1 => ok 20\n            sample: 2 => ok 30\n", 1))
	code, out, diag = run("can.project.root/queues", "can.project.root/queues::race", "sample")
	if code != 1 || diag != "" || !strings.Contains(out, "unused fixture") || !strings.Contains(out, `"row":2`) || strings.Contains(out, "argument mismatch") {
		t.Fatalf("leftover check must follow losing participant drain: %d %s %s", code, out, diag)
	}
	const boundary = `package io_test
    provides []
    uses [bytes, codec, io]
fn void boundary
    emits [codec::invalid_data, io::write_failed]
    asserts
        guarded: => ok
    match call bytes::from_utf8("must-not-be-written")
        codec::invalid_data
        ok bytes::buffer payload => match call io::stdout_write(payload)
            io::write_failed
            [_] => ok
            ok int written => ok
`
	write("src/main.can", boundary)
	code, out, diag = run()
	if code != 1 || diag != "" || report(out)["passed"] != false || strings.Contains(out, "must-not-be-written") || !strings.Contains(out, "missing fixture") {
		t.Fatalf("caught external boundary: %d %q %q", code, out, diag)
	}
	supplied := strings.Replace(boundary, "            io::write_failed", "            when\n                guarded: payload => ok 19\n            io::write_failed", 1)
	write("src/main.can", supplied)
	code, out, diag = run()
	if code != 0 || diag != "" || report(out)["passed"] != true || !strings.Contains(out, "supplied-completion") {
		t.Fatalf("supplied I/O boundary: %d %s %s", code, out, diag)
	}
	write("can.errors.json", `{"active":["errors::failed"],"retired":[]}`)
	write("src/main.can", `package errors
    provides []
    uses []
error failed(int code)
fn int failure
    emits [failed]
    given
        int code
    asserts
        domain: 7 => failed(7)
    failed(code)
fn int consume
    emits []
    asserts
        supplied: => ok 7
    match call failure(1)
        when
            supplied: 1 => failed(7)
        failed => ok failed.code
        ok
`)
	code, out, diag = run()
	if code != 0 || diag != "" || report(out)["passed"] != true {
		t.Fatalf("expected domain completion: %d %s %s", code, out, diag)
	}
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", strings.Replace(string(basic), "fn int square", "int broken = 1 / 0\nfn int square", 1))
	code, out, diag = run()
	if code != 1 || diag != "" || !strings.Contains(out, "initialization failed") {
		t.Fatalf("initialization: %d %s %s", code, out, diag)
	}
}

// Exact runtime mapping must survive both suite setup failure and a source catch
// of a sticky harness violation. Neither path has an ordinary uncaught main.
func TestAssertionFailureLocations(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for assertion diagnostic qualification")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "assertion-locations")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(root, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`, "can.errors.json": `{"active":[],"retired":[]}`} {
		if err = os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(text, expression string, initialization bool) {
		t.Helper()
		if err = os.WriteFile(filepath.Join(root, "src/main.can"), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		command := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "assert", root)
		command.Env = []string{"PATH=/nonexistent", "HOME=" + t.TempDir(), "SECRET=must-not-disclose"}
		var out, diag bytes.Buffer
		command.Stdout = &out
		command.Stderr = &diag
		err := command.Run()
		var status *exec.ExitError
		if !errors.As(err, &status) || status.ExitCode() != 1 || diag.Len() != 0 {
			t.Fatalf("status/diagnostic: %v %s %s", err, out.String(), diag.String())
		}
		var report map[string]any
		if err = json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatal(err, out.String())
		}
		var frames []any
		if initialization {
			if report["reason"] != "initialization failed" {
				t.Fatal(report)
			}
			frames, _ = report["frames"].([]any)
		} else {
			test := report["assertions"].([]any)[0].(map[string]any)
			if test["reason"] != "harness violation" || !strings.Contains(out.String(), "missing fixture") {
				t.Fatal(report)
			}
			frames, _ = test["frames"].([]any)
		}
		start := strings.Index(text, expression)
		end := start + len(expression)
		line := strings.Count(text[:start], "\n") + 1
		column := start - strings.LastIndex(text[:start], "\n")
		found := false
		for _, raw := range frames {
			frame := raw.(map[string]any)
			if frame["file"] == "main.can" && frame["start"] == float64(start) && frame["end"] == float64(end) && frame["line"] == float64(line) && frame["column"] == float64(column) {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing exact span [%d,%d): %s", start, end, out.String())
		}
		for _, secret := range []string{root, bundle, "must-not-disclose", "can:cli", "Error:", "must-not-write"} {
			if strings.Contains(out.String(), secret) {
				t.Fatal("disclosed private detail", secret)
			}
		}
	}
	run(`package app
    provides []
    uses []
int broken = 1 / 0
fn int sample
    emits []
    asserts
        works: => ok 1
    ok 1
`, "1 / 0", true)
	run(`package app
    provides []
    uses [bytes, codec, io]
fn void sample
    emits [codec::invalid_data, io::write_failed]
    asserts
        guarded: => ok
    match call bytes::from_utf8("must-not-write")
        codec::invalid_data
        ok bytes::buffer payload => match call io::stdout_write(payload)
            io::write_failed
            [_] => ok
            ok int written => ok
`, "io::stdout_write(payload)", false)
}

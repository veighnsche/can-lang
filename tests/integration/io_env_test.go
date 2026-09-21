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

func TestCurrentBundledInputEnvironment(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "io-env-integration")
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
	// Empty PATH and a hostile startup option also verify original-environment lookup.
	run := func(command string, input []byte, args ...string) (int, []byte, string) {
		t.Helper()
		argv := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), command, root}
		if len(args) > 0 {
			argv = append(argv, "--")
			argv = append(argv, args...)
		}
		cmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", argv...)
		cmd.Dir = t.TempDir()
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + cmd.Dir, "EMPTY=", "VALUE=hé😀", "BUN_OPTIONS=--preload=/must-not-execute.ts"}
		cmd.Stdin = bytes.NewReader(input)
		var out, diag bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diag
		if err := cmd.Run(); err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				return e.ExitCode(), out.Bytes(), diag.String()
			}
			t.Fatal(err)
		}
		return 0, out.Bytes(), diag.String()
	}
	for _, name := range []string{"input", "input-text", "environment"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/cli", name+".can"))
			if err != nil {
				t.Fatal(err)
			}
			write("src/main.can", string(source))
			if code, out, diag := run("assert", []byte("must not be read")); code != 0 || diag != "" || !strings.Contains(string(out), "supplied-completion") {
				t.Fatalf("boundary fixtures: %d %s %s", code, out, diag)
			}
			if name == "environment" {
				for _, tc := range []struct{ name, want string }{{"VALUE", "hé😀"}, {"EMPTY", ""}, {"BUN_OPTIONS", "--preload=/must-not-execute.ts"}} {
					if code, out, diag := run("run", nil, tc.name); code != 0 || string(out) != tc.want || diag != "" {
						t.Fatalf("environment %s: %d %q %s", tc.name, code, out, diag)
					}
				}
				for _, tc := range []struct{ name, id string }{{"MISSING", "1101"}, {"lower", "1262"}, {"A=B", "1262"}, {"", "1262"}} {
					if code, out, diag := run("run", nil, tc.name); code != 1 || len(out) != 0 || !strings.Contains(diag, `"id":`+tc.id) {
						t.Fatalf("environment error %s: %d %q %s", tc.name, code, out, diag)
					}
				}
			} else {
				samples := [][]byte{{}, []byte("\ufeffhé😀"), bytes.Repeat([]byte{'x'}, 16)}
				if name == "input" {
					samples = append(samples, []byte{0, 255, 128, 10})
				}
				for _, sample := range samples {
					code, out, diag := run("run", sample)
					wantDiag := ""
					if name == "input" {
						wantDiag = string(sample)
					}
					if code != 0 || !bytes.Equal(out, sample) || diag != wantDiag {
						t.Fatalf("input roundtrip: %d %q %q", code, out, diag)
					}
				}
				if code, out, diag := run("run", bytes.Repeat([]byte{'x'}, 17)); code != 1 || len(out) != 0 || !strings.Contains(diag, `"id":1212`) {
					t.Fatalf("overflow: %d %q %s", code, out, diag)
				}
				if name == "input-text" {
					if code, out, diag := run("run", []byte{255}); code != 1 || len(out) != 0 || !strings.Contains(diag, `"id":1110`) {
						t.Fatalf("UTF-8: %d %q %s", code, out, diag)
					}
				}
				write("src/main.can", strings.ReplaceAll(string(source), "(16)", "(-1)"))
				if code, out, diag := run("run", nil); code != 1 || len(out) != 0 || !strings.Contains(diag, `"id":1212`) {
					t.Fatalf("negative limit: %d %q %s", code, out, diag)
				}
				if name == "input" {
					large := bytes.Repeat([]byte{'x'}, 1048576)
					write("src/main.can", strings.ReplaceAll(string(source), "(16)", "(1048576)"))
					if code, out, diag := run("run", large); code != 0 || !bytes.Equal(out, large) || diag != string(large) {
						t.Fatalf("awaited output drain: %d %d %d", code, len(out), len(diag))
					}
				}
				write("src/main.can", string(source))
			}
			code, out, diag := run("build", nil)
			if code != 0 {
				t.Fatalf("build: %d %s %s", code, out, diag)
			}
			var build struct {
				Directory string `json:"directory"`
			}
			if err := json.Unmarshal(out, &build); err != nil {
				t.Fatal(err)
			}
			if tsc := os.Getenv("CAN_TSC"); tsc != "" {
				cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), tsc, "--noEmit", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(tsc))), "@types"), "--types", "bun,node", filepath.Join(build.Directory, "entry.ts"))
				if result, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("strict generated TypeScript: %v %s", err, result)
				}
			}
		})
	}
}

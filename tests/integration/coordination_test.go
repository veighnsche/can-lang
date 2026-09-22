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

func TestCurrentBundledCoordination(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline coordination execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "coordination-integration")
	if err != nil {
		t.Fatal(err)
	}
	qualified := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "runtime/bun"), "--no-env-file", "--no-macros", "--no-install", "--config="+filepath.Join(bundle, "tools/runtime/bunfig.toml"), "test", filepath.Join(bundle, "runtime/test/coordination.test.ts"))
	qualified.Dir = t.TempDir()
	qualified.Env = []string{"PATH=/nonexistent", "HOME=" + qualified.Dir, "XDG_CONFIG_HOME=" + qualified.Dir}
	if output, err := qualified.CombinedOutput(); err != nil || !strings.Contains(string(output), "14 pass") || !strings.Contains(string(output), "0 fail") {
		t.Fatalf("offline coordination traces: %v\n%s", err, output)
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
	}{{"main", 8}, {"aggregate-composition", 25}, {"heterogeneous", 7}} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/coordination", fixture.name+".can"))
		if err != nil {
			t.Fatal(err)
		}
		write("src/main.can", string(data))
		status, out, diag := run("assert")
		if status != 0 || diag != "" {
			t.Fatalf("coordination assertions: %d %s %s", status, out, diag)
		}
		var report map[string]any
		if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != fixture.rows {
			t.Fatalf("invalid coordination report %v %s", err, out)
		}
		status, out, diag = run("run")
		if status != 0 || out != "" || diag != "" {
			t.Fatalf("coordination execution: %d %s %s", status, out, diag)
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

// C9.1: one standard occurrence observed through an ordinary catch and
// re-observed through an aggregate leaf retains identity and projections.
func TestStandardSnapshotIdentity(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for offline snapshot execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "snapshot-identity")
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
	write("src/main.can", `package app
    provides []
    uses [codec]
variant failure
    codec::invalid_data
    standard_failure
fn int boom
    emits []
    asserts
        sample: => ok 0
    ok 1 / 0
fn bool identical
    emits []
    asserts
        sample: => ok true
    match call boom()
        [_] as standard_failure caught => do
            all_failed<failure> wrapped = all_failed<failure>([caught])
            failure leaf = wrapped.failures[0]
            match leaf
                codec::invalid_data => ok false
                standard_failure => ok caught.occurrence_id is leaf.occurrence_id and caught.kind is leaf.kind and caught.message is leaf.message
        ok int value => ok false
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`)
	argv := []string{"-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "assert", root, "can.project.root/app", "can.project.root/app::identical", "sample"}
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
		t.Fatalf("snapshot identity assertions: %d %s %s", status.ExitCode(), out.String(), diag.String())
	}
	if diag.String() != "" {
		t.Fatalf("snapshot identity diagnostics: %s", diag.String())
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out.String()), &report); err != nil || report["passed"] != true {
		t.Fatalf("snapshot identity failed: %v %s", err, out.String())
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
			t.Fatalf("strict snapshot TypeScript: %v\n%s", err, output)
		}
	}
}

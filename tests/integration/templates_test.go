package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/veighnsche/can-lang/distribution"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCurrentBundledTemplates(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged template execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "template-integration")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	loads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path != "/load" {
			t.Error("unexpected template path")
		}
		loads++
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"count":7}`)
	}))
	defer server.Close()
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
		profile := `(version 1)(allow default)(deny network*)(allow network-outbound (remote ip "localhost:*"))`
		argv := []string{"-p", profile, filepath.Join(bundle, "bin/canlc"), command, root}
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

	stage := func(t *testing.T, mutate func(string) string) {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/templates", "main.can"))
		if err != nil {
			t.Fatal(err)
		}
		helpers, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/templates", "helpers.can"))
		if err != nil {
			t.Fatal(err)
		}
		source := strings.Replace(string(data), "http://127.0.0.1:1/", server.URL+"/", 1)
		if mutate != nil {
			source = mutate(source)
		}
		write("src/main.can", source)
		write("src/helpers.can", string(helpers))
		stageRawFixtures(t, write, sourceRoot, "templates", [2]string{"http://127.0.0.1:1", server.URL})
	}
	stage(t, nil)

	// 13 rows: target asserts, helper direct rows, consumer samples and
	// main. Expanded rows consume positionally: each consumer drives its
	// helper site twice through two template cases in order.
	status, out, diag := run("assert")
	if status != 0 || diag != "" {
		t.Fatalf("template assertions: %d %s %s", status, out, diag)
	}
	var report map[string]any
	if err = json.Unmarshal([]byte(out), &report); err != nil || report["passed"] != true || len(report["assertions"].([]any)) != 13 {
		t.Fatalf("invalid template report %v %s", err, out)
	}
	for _, label := range []string{"supplied-completion", "raw-provider-fixture", "real-can"} {
		if !strings.Contains(out, label) {
			t.Fatalf("template report omits %s evidence: %s", label, out)
		}
	}
	mu.Lock()
	if loads != 0 {
		t.Fatalf("assertions issued %d live requests", loads)
	}
	mu.Unlock()

	// Production output erases fixture declarations; the live run uses
	// real transport exactly once.
	status, out, diag = run("run")
	if status != 0 || out != "" || diag != "" {
		t.Fatalf("template execution: %d %s %s", status, out, diag)
	}
	mu.Lock()
	if loads != 1 {
		t.Fatalf("expected one live load, got %d", loads)
	}
	mu.Unlock()

	// A use naming the wrong exact target fails the staged build even
	// though the signatures match.
	stage(t, func(source string) string {
		return strings.Replace(source, "sample: use doubled(2)", "sample: use tripled(2)", 1)
	})
	status, _, diag = run("assert")
	if status == 0 || !strings.Contains(diag, "not the invoked") {
		t.Fatalf("wrong-target use passed: %d %s", status, diag)
	}
	if compiler := os.Getenv("CAN_TSC"); compiler != "" {
		nodeModules := filepath.Dir(filepath.Dir(filepath.Dir(compiler)))
		args := []string{compiler, "--noEmit", "--ignoreConfig", "--strict", "--skipLibCheck", "--target", "esnext", "--module", "esnext", "--moduleResolution", "bundler", "--allowImportingTsExtensions", "--typeRoots", filepath.Join(nodeModules, "@types"), "--types", "bun,node"}
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

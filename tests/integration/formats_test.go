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

const formatsManifest = `{"source_root":"src","error_registry":"can.errors.json"}`

var formatsDocuments = map[string]string{
	"svc.toml":        "title = \"edge\"\nports = [80, 443]\n[primary]\nhost = \"h\"\nport = 8080\n",
	"point.yaml":      "name: r\nn: 7\n",
	"point.json5":     "{/* c */name: 'r', // d\nn: 7,}",
	"points.jsonl":    "{\"name\":\"a\",\"n\":9007199254740993}\n\n{\"name\":\"b\",\"n\":-0}\r\n{\"name\":\"c\",\"n\":-12}",
	"bad.toml":        "title = \"a\"\ntitle = \"b\"\n",
	"truncated.jsonl": "{\"name\":\"a\",\"n\":1}\n{\"name\":\"b\",\"n\":",
}

func writeFormatsProject(t *testing.T, root string) {
	t.Helper()
	sourceRoot, _ := filepath.Abs("../..")
	main, err := os.ReadFile(filepath.Join(sourceRoot, "tests/integration/testdata/formats/main.can"))
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
	write("can.project.json", formatsManifest)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", string(main))
}

func TestCurrentFormatsDocuments(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged formats execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "formats-live")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFormatsProject(t, root)
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root)
	if status != 0 || diag != "" {
		t.Fatalf("formats assert: %d %s %s", status, out, diag)
	}
	buildCmd := exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1)(allow default)(deny network*)", filepath.Join(bundle, "bin/canlc"), "build", root)
	buildCmd.Dir = home
	buildCmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("live build: %v %s", err, string(buildOut))
	}
	var build struct {
		Directory string `json:"directory"`
	}
	if err := json.Unmarshal(buildOut, &build); err != nil || build.Directory == "" {
		t.Fatalf("invalid live build report %v %s", err, string(buildOut))
	}
	docs, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	doc := func(name string) string {
		t.Helper()
		p := filepath.Join(docs, name)
		if err := os.WriteFile(p, []byte(formatsDocuments[name]), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// The compiled CLI reads an fd-3 environment snapshot like every
	// production entry; this program takes no configuration.
	snapshot := filepath.Join(home, "snapshot.json")
	if err := os.WriteFile(snapshot, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (int, string, string) {
		t.Helper()
		file, err := os.Open(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{filepath.Join(build.Directory, "entry.ts")}, args...)...)
		cmd.Dir = home
		cmd.Env = []string{"PATH=/nonexistent", "HOME=" + home}
		cmd.ExtraFiles = []*os.File{file}
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			var status *exec.ExitError
			if !errors.As(err, &status) {
				t.Fatal(err)
			}
			return status.ExitCode(), stdout.String(), stderr.String()
		}
		return 0, stdout.String(), stderr.String()
	}
	checks := 0
	ok := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(args...)
		if status != 0 || out != want || diag != "" {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, out, diag)
		}
	}
	fault := func(args []string, want string) {
		t.Helper()
		checks++
		status, out, diag := run(args...)
		if status != 1 || out != "" || !strings.Contains(diag, `"error":"`+want+`"`) {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, out, diag)
		}
	}
	ok(nil, "usage: toml <path> | yaml <path> | json5 <path> | jsonl <path>")
	ok([]string{"bogus"}, "usage: toml <path> | yaml <path> | json5 <path> | jsonl <path>")
	ok([]string{"toml"}, "usage: toml <path>")
	ok([]string{"toml", doc("svc.toml")}, "8080")
	ok([]string{"yaml", doc("point.yaml")}, "7")
	ok([]string{"json5", doc("point.json5")}, "7")
	ok([]string{"jsonl", doc("points.jsonl")}, "3")
	fault([]string{"toml", filepath.Join(docs, "absent.toml")}, "files::not_found")
	fault([]string{"toml", doc("bad.toml")}, "codec::invalid_data")
	fault([]string{"jsonl", doc("truncated.jsonl")}, "codec::invalid_data")
	t.Logf("live formats: %d checks", checks)
}

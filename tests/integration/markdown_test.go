package integration

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/distribution"
)

const markdownManifest = `{"source_root":"src","error_registry":"can.errors.json"}`

func writeMarkdownProject(t *testing.T, root, main string) {
	t.Helper()
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
	write("can.project.json", markdownManifest)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", main)
}

func TestCurrentMarkdownRender(t *testing.T) {
	archive := os.Getenv("CAN_BUN_ARCHIVE")
	if archive == "" {
		t.Skip("set CAN_BUN_ARCHIVE for staged markdown execution")
	}
	sourceRoot, _ := filepath.Abs("../..")
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	bundle, err := distribution.Build(ctx, sourceRoot, t.TempDir(), archive, "markdown-live")
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join(sourceRoot, "compiler/testdata/current/markdown/main.can"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeMarkdownProject(t, root, string(fixture))
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	status, out, diag := canlcOffline(t, ctx, bundle, home, "assert", root)
	if status != 0 || diag != "" {
		t.Fatalf("markdown assert: %d %s %s", status, out, diag)
	}
	_, directory := applicationBuild(t, ctx, bundle, home, root)
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
		cmd := exec.CommandContext(ctx, filepath.Join(bundle, "runtime/bun"), append([]string{filepath.Join(directory, "entry.ts")}, args...)...)
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
	ok := func(args []string, want string) {
		t.Helper()
		status, out, diag := run(args...)
		if status != 0 || out != want || diag != "" {
			t.Fatalf("live %v: %d stdout=%q stderr=%s", args, status, out, diag)
		}
	}
	ok([]string{"html", "# Hi"}, "<h1>Hi</h1>\n")
	ok([]string{"html", "<script>alert(1)</script>"}, "<script>alert(1)</script>\n")
	ok(nil, "usage: md html <source>")
	ok([]string{"bogus"}, "usage: md html <source>")
	ok([]string{"html"}, "usage: md html <source>")
	negatives := []struct {
		name string
		text string
		want string
	}{
		{"raw string is not safe", strings.Replace(string(fixture), `call markdown::render_safe("# Hello") as html::safe sheet`, `call markdown::render_text_html("# Hello") as html::safe sheet`, 1), "chain binding type mismatch"},
		{"missing emits", strings.Replace(string(fixture), "fn html::safe page\n    emits [markdown::over_limit, html::invalid_url]", "fn html::safe page\n    emits [markdown::over_limit]", 1), "undeclared escaping domain error html::invalid_url"},
	}
	for _, negative := range negatives {
		if negative.text == string(fixture) {
			t.Fatalf("%s replacement missed", negative.name)
		}
		dir, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		writeMarkdownProject(t, dir, negative.text)
		status, out, diag := canlcOffline(t, ctx, bundle, home, "build", dir)
		if status == 0 || !strings.Contains(out+diag, negative.want) {
			t.Fatalf("%s admitted: %d %s %s", negative.name, status, out, diag)
		}
	}
}

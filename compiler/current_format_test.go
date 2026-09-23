package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const formatMain = `package app
    provides [tally]
    uses []

// total with tip
fn   int   tally
    emits []
    given
        int seed  // per cover
    asserts
        sample: 1 => ok 2
    ok seed + seed  // doubled
`

func writeFormatProject(t *testing.T, main string) (string, string) {
	t.Helper()
	root := t.TempDir()
	write := func(name, text string, mode os.FileMode) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), mode); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`, 0644)
	write("can.errors.json", `{"active":[],"retired":[]}`, 0644)
	write("src/main.can", main, 0640)
	real, err := filepath.EvalSymlinks(filepath.Join(root, "src/main.can"))
	if err != nil {
		t.Fatal(err)
	}
	return root, real
}

func TestFormatStdoutKeepsFileUntouched(t *testing.T) {
	_, file := writeFormatProject(t, formatMain)
	var stdout, stderr bytes.Buffer
	if code := runCurrentFormat(&stdout, &stderr, []string{file}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"fn int tally\n", "// total with tip\n", "int seed  // per cover\n", "ok seed + seed  // doubled\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout omits %q:\n%s", want, out)
		}
	}
	if data, err := os.ReadFile(file); err != nil || string(data) != formatMain {
		t.Fatalf("stdout mode touched the file: %v", err)
	}
}

func TestFormatWriteReplacesAtomically(t *testing.T) {
	_, file := writeFormatProject(t, formatMain)
	var stdout, stderr bytes.Buffer
	if code := runCurrentFormat(&stdout, &stderr, []string{"--write", file}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "fn int tally\n") || strings.Contains(string(data), "fn   int") {
		t.Fatalf("file not canonicalized:\n%s", data)
	}
	if info, err := os.Stat(file); err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("file mode not preserved: %v", info)
	}
	if leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(file), ".can-format-*")); err != nil || len(leftovers) != 0 {
		t.Fatalf("temp files left: %v %v", leftovers, err)
	}
	// A canonical file rewrites to identical bytes.
	again := bytes.Buffer{}
	if code := runCurrentFormat(&again, &stderr, []string{"--write", file}); code != 0 {
		t.Fatalf("second exit %d: %s", code, stderr.String())
	}
	if second, err := os.ReadFile(file); err != nil || string(second) != string(data) {
		t.Fatalf("second write moved bytes")
	}
}

func TestFormatWriteRefusesInvalidSource(t *testing.T) {
	_, file := writeFormatProject(t, formatMain+"fn int broken(\n")
	var stdout, stderr bytes.Buffer
	if code := runCurrentFormat(&stdout, &stderr, []string{"--write", file}); code != 1 {
		t.Fatalf("exit %d, want 1: %s", code, stderr.String())
	}
	if data, err := os.ReadFile(file); err != nil || !strings.HasSuffix(string(data), "fn int broken(\n") {
		t.Fatalf("invalid source touched")
	}

	semantic := strings.Replace(formatMain, "ok seed + seed", "ok missing", 1)
	_, bad := writeFormatProject(t, semantic)
	stderr.Reset()
	if code := runCurrentFormat(&stdout, &stderr, []string{"--write", bad}); code != 1 {
		t.Fatalf("semantic exit %d, want 1: %s", code, stderr.String())
	}
	if data, err := os.ReadFile(bad); err != nil || string(data) != semantic {
		t.Fatalf("semantic failure touched the file")
	}
	if !strings.Contains(stderr.String(), "cannot format") {
		t.Fatalf("refusal unexplained: %s", stderr.String())
	}
}

func TestFormatWriteRefusesStaleSource(t *testing.T) {
	_, file := writeFormatProject(t, formatMain)
	formatted, err := formatSource(file, formatMain)
	if err != nil {
		t.Fatal(err)
	}
	if formatted == formatMain {
		t.Fatal("fixture is already canonical; nothing is proven")
	}
	// A replaced file carries a foreign identity.
	other := file + ".other"
	if err := os.WriteFile(other, []byte(formatMain), 0644); err != nil {
		t.Fatal(err)
	}
	stale, err := os.Lstat(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeFormatted(file, stale, formatMain, formatted); err == nil || !strings.Contains(err.Error(), "replaced since read") {
		t.Fatalf("replaced source accepted: %v", err)
	}
	// Matching identity with moved bytes is equally stale.
	identity, err := os.Lstat(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeFormatted(file, identity, "package app\n", formatted); err == nil || !strings.Contains(err.Error(), "changed since read") {
		t.Fatalf("changed source accepted: %v", err)
	}
	if data, err := os.ReadFile(file); err != nil || string(data) != formatMain {
		t.Fatalf("stale refusal touched the file")
	}
	if leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(file), ".can-format-*")); err != nil || len(leftovers) != 0 {
		t.Fatalf("temp files left: %v %v", leftovers, err)
	}
}

func TestFormatWriteRequiresProject(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "lone.can")
	if err := os.WriteFile(file, []byte(formatMain), 0644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runCurrentFormat(&stdout, &stderr, []string{"--write", file}); code != 1 {
		t.Fatalf("exit %d, want 1: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "can.project.json") {
		t.Fatalf("project refusal unexplained: %s", stderr.String())
	}
	if data, err := os.ReadFile(file); err != nil || string(data) != formatMain {
		t.Fatalf("projectless write touched the file")
	}
	// Stdout mode needs no project.
	stdout.Reset()
	if code := runCurrentFormat(&stdout, &stderr, []string{file}); code != 0 {
		t.Fatalf("stdout exit %d: %s", code, stderr.String())
	}
}

func TestFormatUsage(t *testing.T) {
	for _, argv := range [][]string{{}, {"a.can", "b.can"}, {"--bogus", "a.can"}} {
		var stdout, stderr bytes.Buffer
		if code := runCurrentFormat(&stdout, &stderr, argv); code != 2 {
			t.Fatalf("runCurrentFormat(%q) = %d, want 2", argv, code)
		}
	}
	if code := run([]string{"format"}); code != 2 {
		t.Fatalf("run(format) = %d, want 2", code)
	}
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyOutputCLIRefusesWithoutWriting(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.can")
	if err := os.WriteFile(source, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"--out", root, source}); code != 2 {
		t.Fatalf("retired output command status %d", code)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "source.can" {
		t.Fatal("retired output command wrote files")
	}
}
func TestCurrentCleanRefusesUnknownContents(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`, "can.errors.json": `{"active":[],"retired":[]}`, "src/main.can": "package app\n    provides []\n    uses []\nint a = 1\n", "dist/keep.txt": "keep"} {
		p := filepath.Join(root, name)
		os.MkdirAll(filepath.Dir(p), 0700)
		if err = os.WriteFile(p, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if code := run([]string{"clean", root}); code != 1 {
		t.Fatal("clean claimed nonempty unowned dist")
	}
	if raw, err := os.ReadFile(filepath.Join(root, "dist", "keep.txt")); err != nil || string(raw) != "keep" {
		t.Fatal("clean damaged unknown file")
	}
}

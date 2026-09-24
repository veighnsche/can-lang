package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copyProjectFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := "testdata/current/project"
	if err := filepath.WalkDir(source, func(name string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, name)
		if err != nil {
			return err
		}
		target := filepath.Join(root, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0600)
	}); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCurrentProjectInspectionIsInertAndRelocatable(t *testing.T) {
	var first bytes.Buffer
	for i := 0; i < 2; i++ {
		root := copyProjectFixture(t)
		var out, diagnostics bytes.Buffer
		if code := runInspectProject(&out, &diagnostics, []string{root}); code != 0 {
			t.Fatalf("%d: %s", code, &diagnostics)
		}
		if strings.Contains(out.String(), root) {
			t.Fatal("report leaks machine path into identity")
		}
		var report projectReport
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if report.SchemaVersion != 2 || report.Kind != "can.package-resolution" || len(report.Packages) != 4 || len(report.Projects) != 2 {
			t.Fatal(report)
		}
		seen := map[string]bool{}
		shared := 0
		for _, pkg := range report.Packages {
			for _, source := range pkg.Sources {
				if seen[strings.ToLower(source.OutputPath)] {
					t.Fatal("output path collision")
				}
				seen[strings.ToLower(source.OutputPath)] = true
				if strings.HasSuffix(source.SourcePath, "shared.can") {
					shared++
				}
			}
		}
		if shared != 3 {
			t.Fatal("same-basename fixture not exercised")
		}
		if i == 0 {
			first.Write(out.Bytes())
		} else if !bytes.Equal(first.Bytes(), out.Bytes()) {
			t.Fatal("relocation changed report")
		}
		if _, err := os.Stat(filepath.Join(root, "dist")); !os.IsNotExist(err) {
			t.Fatal("inspection created output directory")
		}
	}
}

func TestCurrentProjectRefusesStaleDependencyWithoutOutput(t *testing.T) {
	root := copyProjectFixture(t)
	path := filepath.Join(root, "vendor/sample/src/shared.can")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := f.WriteString("\n// changed\n")
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatal(writeErr, closeErr)
	}
	var out, diagnostics bytes.Buffer
	if code := runInspectProject(&out, &diagnostics, []string{root}); code != 1 || out.Len() != 0 || !strings.Contains(diagnostics.String(), "stale dependency digest") {
		t.Fatalf("%d: %s %s", code, &out, &diagnostics)
	}
}

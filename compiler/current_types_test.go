package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeclarationTypeInspection(t *testing.T) {
	var previous string
	for i := 0; i < 2; i++ {
		root := copyProjectFixture(t)
		var out, diagnostics bytes.Buffer
		if code := runInspectTypes(&out, &diagnostics, []string{root}); code != 0 {
			t.Fatalf("%d: %s", code, diagnostics.String())
		}
		var report typeReport
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if report.SchemaVersion != 1 || report.Kind != "can.declaration-types" || len(report.Types) == 0 {
			t.Fatal(report)
		}
		if strings.Contains(out.String(), root) {
			t.Fatal("machine path in type identities")
		}
		if i != 0 && previous != out.String() {
			t.Fatal("type report changed across relocation")
		}
		previous = out.String()
		path := filepath.Join(root, "src/alpha/second.can")
		file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.WriteString("\nrecord infinite\n    infinite next\n"); err != nil {
			t.Fatal(err)
		}
		if err = file.Close(); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		diagnostics.Reset()
		if code := runInspectTypes(&out, &diagnostics, []string{root}); code == 0 || out.Len() != 0 || !strings.Contains(diagnostics.String(), "finite inhabitant") {
			t.Fatalf("invalid type report published: %s / %s", out.String(), diagnostics.String())
		}
	}
}

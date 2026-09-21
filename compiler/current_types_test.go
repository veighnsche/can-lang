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

func TestDeclarationInspectionRejectsUnusedGenericErrors(t *testing.T) {
	cases := []struct{ name, source, registry, diagnostic string }{
		{"variant", "record box<item>\n    item value\n\nvariant impossible<item>\n    box<item>\n    box<item>\n", `{"active":[],"retired":[]}`, "duplicate variant leaf"},
		{"constraint", "record holder<item>\n    collections::map<float,item> value\n", `{"active":[],"retired":[]}`, "catalogue constraint map_key"},
		{"bound", "error 1000000 failed<item>(item value)\nfn item work<item>\n    emits [failed<item>,failed<item>]\n    given\n        item value\n    asserts\n        sample: 1 => ok 1\n    ok value\n", `{"active":[{"id":1000000,"kind":"app::failed"}],"retired":[]}`, "duplicate error in bound"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for name, data := range map[string]string{"can.project.json": `{"source_root":".","error_registry":"can.errors.json"}`, "can.errors.json": tc.registry, "main.can": "package app\n    provides []\n    uses [collections]\n" + tc.source} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var out, diagnostics bytes.Buffer
			if code := runInspectTypes(&out, &diagnostics, []string{root}); code == 0 || out.Len() != 0 || !strings.Contains(diagnostics.String(), tc.diagnostic) {
				t.Fatalf("unused generic admitted: code=%d output=%s diagnostics=%s", code, out.String(), diagnostics.String())
			}
		})
	}
}

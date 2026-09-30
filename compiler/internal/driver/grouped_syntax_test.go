package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func TestGroupedCompletionDefinitionJumps(t *testing.T) {
	const prefix = `package app
    provides [read, handle]
    uses []
error denied{str key}
error missing{str key}
fn void read
    emits {denied, missing}
    given
        bool blocked
    asserts
        sample: false => missing{"item"}
    match blocked
        false => missing{"item"}
        true => denied{"item"}
fn void handle
    emits {}
    asserts
        sample: => ok
`
	for _, test := range []struct {
		name, body, group string
		heads             []string
	}{
		{"call", "    match call read(false)\n        denied | missing => ok\n        ok => ok\n", "denied | missing", []string{"denied", "missing"}},
		{"participant", "    match call concurrent with error\n        read(false)\n            denied | missing => ok\n            ok => ok\n    ok\n", "denied | missing", []string{"denied", "missing"}},
		{"shared", "    match call race with error\n        read(false)\n        denied | missing => ok\n        ok => ok\n    ok\n", "denied | missing", []string{"denied", "missing"}},
		{"single", "    match call read(false)\n        denied => ok\n        missing => ok\n        ok => ok\n", "denied => ok", []string{"denied"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			text := prefix + test.body
			root := writeBridgeProject(t, map[string]string{"src/main.can": text})
			if err := os.WriteFile(filepath.Join(root, "can.errors.json"), []byte(`{"active":["app::denied","app::missing"],"retired":[]}`), 0600); err != nil {
				t.Fatal(err)
			}
			file := canonical(t, filepath.Join(root, "src/main.can"))
			snapshot, err := CheckSnapshot(root, file, project.NewOverlay())
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Diagnostics) != 0 {
				t.Fatalf("grouped fixture does not check: %+v", snapshot.Diagnostics)
			}
			position := func(offset int) (int, int) {
				line := strings.Count(text[:offset], "\n")
				return line, offset - strings.LastIndex(text[:offset], "\n") - 1
			}
			start := strings.Index(text, test.group)
			if start < 0 {
				t.Fatalf("group %q missing from fixture", test.group)
			}
			for _, name := range test.heads {
				head := strings.Index(test.group, name)
				if head < 0 {
					t.Fatalf("head %q missing from group %q", name, test.group)
				}
				line, character := position(start + head)
				at, ok, err := Definition(snapshot, file, line, character)
				if err != nil || !ok {
					t.Fatalf("definition for %s: found=%v error=%v", name, ok, err)
				}
				declaration := strings.Index(text, "error "+name) + len("error ")
				wantLine := strings.Count(text[:declaration], "\n")
				if at.File != file || at.Line != wantLine || at.Start != len("error ") || at.End != len("error ")+len(name) {
					t.Fatalf("%s definition points at wrong source: %+v", name, at)
				}
			}
			if pipe := strings.Index(test.group, "|"); pipe >= 0 {
				line, character := position(start + pipe)
				at, ok, err := Definition(snapshot, file, line, character)
				if err != nil {
					t.Fatalf("definition at separator: error=%v", err)
				}
				if ok {
					t.Fatalf("separator resolves unexpectedly: %+v", at)
				}
			}
		})
	}
}

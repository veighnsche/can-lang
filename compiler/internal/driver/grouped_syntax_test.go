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
    provides []
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
fn int pick
    emits {denied, missing}
    asserts
        sample: => ok 1
    match call read(false)
        denied
        missing
        ok => ok 1
fn void handle
    emits {denied, missing}
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
		{"forward", "    match call read(false)\n        denied | missing\n        ok => ok\n", "denied | missing", []string{"denied", "missing"}},
		{"chain", "    match chain\n        call pick() as int value\n        denied | missing => ok\n        ok => ok\n", "denied | missing", []string{"denied", "missing"}},
		{"coord-single", "    match call concurrent with error\n        read(false)\n            denied => ok\n            missing => ok\n            ok => ok\n    ok\n", "denied => ok", []string{"denied"}},
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

func TestGroupedQualifiedAndSpecializedDefinitionJumps(t *testing.T) {
	helper := `package helper
    provides [boom, fused, other]
    uses []
error fused{str key}
error other{str key}
fn void boom
    emits {fused, other}
    asserts
        sample: => ok
    ok
`
	registry := `{"active":["helper::fused","helper::other"],"retired":[]}`
	for _, test := range []struct {
		name, main, group string
		heads             []struct{ cursor, declaration, file string }
	}{
		{"qualified", `package app
    provides []
    uses [helper]
fn void handle
    emits {}
    asserts
        sample: => ok
    match call helper::boom()
        helper::fused | helper::other => ok
        ok => ok
`, "helper::fused | helper::other", []struct{ cursor, declaration, file string }{
			{"fused", "error fused", "helper"},
			{"other", "error other", "helper"},
		}},
		{"specialized", `package app
    provides []
    uses [helper]
variant a_failure
    helper::fused
variant b_failure
    helper::other
fn void agg
    emits {all_failed<a_failure>, all_failed<b_failure>}
    asserts
        sample: => ok
    ok
fn void handle
    emits {}
    asserts
        sample: => ok
    match call agg()
        all_failed<a_failure> | all_failed<b_failure> => ok
        ok => ok
`, "all_failed<a_failure> | all_failed<b_failure>", []struct{ cursor, declaration, file string }{
			{"a_failure", "variant a_failure", "main"},
			{"b_failure", "variant b_failure", "main"},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := writeBridgeProject(t, map[string]string{"src/helper/helper.can": helper, "src/app/main.can": test.main})
			if err := os.WriteFile(filepath.Join(root, "can.errors.json"), []byte(registry), 0600); err != nil {
				t.Fatal(err)
			}
			main := canonical(t, filepath.Join(root, "src/app/main.can"))
			helperFile := canonical(t, filepath.Join(root, "src/helper/helper.can"))
			snapshot, err := CheckSnapshot(root, main, project.NewOverlay())
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Diagnostics) != 0 {
				t.Fatalf("grouped fixture does not check: %+v", snapshot.Diagnostics)
			}
			position := func(offset int) (int, int) {
				line := strings.Count(test.main[:offset], "\n")
				return line, offset - strings.LastIndex(test.main[:offset], "\n") - 1
			}
			start := strings.Index(test.main, test.group)
			if start < 0 {
				t.Fatalf("group %q missing from fixture", test.group)
			}
			for _, head := range test.heads {
				atCursor := strings.Index(test.group, head.cursor)
				if atCursor < 0 {
					t.Fatalf("head %q missing from group %q", head.cursor, test.group)
				}
				line, character := position(start + atCursor)
				at, ok, err := Definition(snapshot, main, line, character)
				if err != nil || !ok {
					t.Fatalf("definition for %s: found=%v error=%v", head.cursor, ok, err)
				}
				wantFile, wantText := main, test.main
				wantColumn := len("variant ")
				if head.file == "helper" {
					wantFile, wantText = helperFile, helper
					wantColumn = len("error ")
				}
				declaration := strings.Index(wantText, head.declaration)
				if declaration < 0 {
					t.Fatalf("declaration %q missing", head.declaration)
				}
				wantLine := strings.Count(wantText[:declaration], "\n")
				if at.File != wantFile || at.Line != wantLine || at.Start != wantColumn || at.End != wantColumn+len(head.cursor) {
					t.Fatalf("%s definition points at wrong source: %+v", head.cursor, at)
				}
			}
			pipe := strings.Index(test.group, "|")
			line, character := position(start + pipe)
			at, ok, err := Definition(snapshot, main, line, character)
			if err != nil {
				t.Fatalf("definition at separator: error=%v", err)
			}
			if ok {
				t.Fatalf("separator resolves unexpectedly: %+v", at)
			}
		})
	}
}

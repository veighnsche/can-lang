package driver

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const bridgeMain = `package app
    provides [tally, helper, account_row]
    uses []

record account_row
    int id
    str display_name

account_row config = account_row(1, "Ann")

fn int helper
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 1
    ok seed

fn int tally
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 2
    ok call helper(seed)
`

const bridgeSecond = `package app
    provides [describe]
    uses []

fn str describe
    emits []
    given
        account_row row
    asserts
        sample: account_row(1, "Ann") => ok "Ann"
    ok config.display_name
`

func writeBridgeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	for name, text := range files {
		write(name, text)
	}
	return root
}

func canonical(t *testing.T, path string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func TestCheckSnapshotHealthy(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	open := canonical(t, filepath.Join(root, "src/main.can"))
	snapshot, err := CheckSnapshot(root, open, project.NewOverlay())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Diagnostics) != 0 {
		t.Fatalf("healthy project diagnosed: %+v", snapshot.Diagnostics)
	}
	if snapshot.World == nil || snapshot.Graph == nil {
		t.Fatal("healthy snapshot lost graph or world")
	}
}

func TestCheckSnapshotParseError(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain})
	open := canonical(t, filepath.Join(root, "src/main.can"))
	broken := strings.Replace(bridgeMain, "fn int helper", "fn int helper(", 1)
	overlay := project.NewOverlay()
	if err := overlay.Set(open, 3, broken); err != nil {
		t.Fatal(err)
	}
	snapshot, err := CheckSnapshot(root, open, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %+v", snapshot.Diagnostics)
	}
	diagnostic := snapshot.Diagnostics[0]
	if diagnostic.File != open || diagnostic.Severity != "error" {
		t.Fatalf("diagnostic misattributed: %+v", diagnostic)
	}
	if diagnostic.Code == "" {
		t.Fatal("parse diagnostic lost its code")
	}
	// The editor span must point at the offending token: the broken
	// declaration sits on zero-based line 10.
	if diagnostic.Line != 10 {
		t.Fatalf("parse diagnostic on wrong line: %+v", diagnostic)
	}
	if diagnostic.Start == diagnostic.End {
		t.Fatalf("parse diagnostic lost its range: %+v", diagnostic)
	}
	// Saving the bytes makes the CLI fail identically.
	if err := os.WriteFile(open, []byte(broken), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := project.Load(root); err == nil {
		t.Fatal("saved breakage loads cleanly")
	} else {
		saved, saveErr := CheckSnapshot(root, open, project.NewOverlay())
		if saveErr != nil {
			t.Fatal(saveErr)
		}
		if len(saved.Diagnostics) != 1 || saved.Diagnostics[0].Message != diagnostic.Message || saved.Diagnostics[0].Code != diagnostic.Code || saved.Diagnostics[0].Line != diagnostic.Line {
			t.Fatalf("overlay diagnosis %+v differs from saved %+v", diagnostic, saved.Diagnostics)
		}
	}
}

func TestCheckSnapshotCheckErrorSpan(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	second := canonical(t, filepath.Join(root, "src/second.can"))
	broken := strings.Replace(bridgeSecond, "ok config.display_name", "ok call missing_fn(row)", 1)
	overlay := project.NewOverlay()
	if err := overlay.Set(second, 5, broken); err != nil {
		t.Fatal(err)
	}
	open := canonical(t, filepath.Join(root, "src/main.can"))
	snapshot, err := CheckSnapshot(root, open, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %+v", snapshot.Diagnostics)
	}
	diagnostic := snapshot.Diagnostics[0]
	// Located checker failures point at the offending expression in the
	// true file with the CLI-identical message; the world still resolves,
	// so navigation keeps working beneath the error.
	if diagnostic.File != second || diagnostic.Line != 10 || diagnostic.EndLine != 10 || diagnostic.Start != 7 || diagnostic.End != 27 {
		t.Fatalf("check diagnostic mislocated: %+v", diagnostic)
	}
	if !strings.Contains(diagnostic.Message, "missing_fn") {
		t.Fatalf("check diagnostic lost the CLI message: %+v", diagnostic)
	}
	if snapshot.World == nil {
		t.Fatal("snapshot lost the world beneath a check error")
	}
	if err := os.WriteFile(second, []byte(broken), 0644); err != nil {
		t.Fatal(err)
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	_, cliErr := check.CheckAssertionProgram(graph)
	if cliErr == nil || cliErr.Error() != diagnostic.Message {
		t.Fatalf("editor message %q differs from CLI %q", diagnostic.Message, cliErr)
	}
	saved, saveErr := CheckSnapshot(root, open, project.NewOverlay())
	if saveErr != nil {
		t.Fatal(saveErr)
	}
	if len(saved.Diagnostics) != 1 || !reflect.DeepEqual(saved.Diagnostics[0], diagnostic) {
		t.Fatalf("overlay diagnosis %+v differs from saved %+v", diagnostic, saved.Diagnostics)
	}
}

func TestCheckSnapshotResolveErrorSpan(t *testing.T) {
	duplicate := bridgeMain + "\nfn int helper\n    emits []\n    given\n        int seed\n    asserts\n        sample: 1 => ok 1\n    ok seed\n"
	root := writeBridgeProject(t, map[string]string{"src/main.can": duplicate})
	open := canonical(t, filepath.Join(root, "src/main.can"))
	snapshot, err := CheckSnapshot(root, open, project.NewOverlay())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %+v", snapshot.Diagnostics)
	}
	diagnostic := snapshot.Diagnostics[0]
	// The duplicate name token sits on its own declaration line.
	wantLine := strings.Count(duplicate[:strings.LastIndex(duplicate, "fn int helper")], "\n")
	if diagnostic.File != open || diagnostic.Line != wantLine || diagnostic.EndLine != wantLine || diagnostic.Start != 7 || diagnostic.End != 13 {
		t.Fatalf("resolve diagnostic mislocated: %+v", diagnostic)
	}
	if !strings.Contains(diagnostic.Message, `duplicate name "helper"`) {
		t.Fatalf("resolve diagnostic lost the CLI message: %+v", diagnostic)
	}
}

func TestCheckSnapshotAstralSpan(t *testing.T) {
	astral := "package app\n    provides [describe]\n    uses []\n\nfn str describe\n    emits []\n    given\n        str row\n    asserts\n        sample: \"x\" => ok \"x\"\n    ok \"\U0001D11E\" + missing\n"
	root := writeBridgeProject(t, map[string]string{"src/main.can": astral})
	open := canonical(t, filepath.Join(root, "src/main.can"))
	snapshot, err := CheckSnapshot(root, open, project.NewOverlay())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %+v", snapshot.Diagnostics)
	}
	diagnostic := snapshot.Diagnostics[0]
	// The astral clef counts two UTF-16 code units, so the failing name
	// starts at column 14 rather than byte column 16.
	if diagnostic.File != open || diagnostic.Line != 10 || diagnostic.EndLine != 10 || diagnostic.Start != 14 || diagnostic.End != 21 {
		t.Fatalf("astral diagnostic mislocated: %+v", diagnostic)
	}
}

func TestSemanticDiagnosticUnavailable(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain})
	open := canonical(t, filepath.Join(root, "src/main.can"))
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	absent := source.Locate(filepath.Join(root, "src", "ghost.can"), source.Span{Start: 0, End: 5}, errors.New("boom"))
	diagnostic := semanticDiagnostic(graph, open, absent)
	if diagnostic.File != filepath.Join(root, "src", "ghost.can") || diagnostic.Code != source.SpanUnavailable {
		t.Fatalf("absent file silently relocated: %+v", diagnostic)
	}
	stale := source.Locate(open, source.Span{Start: 1 << 30, End: (1 << 30) + 5}, errors.New("boom"))
	diagnostic = semanticDiagnostic(graph, open, stale)
	if diagnostic.File != open || diagnostic.Code != source.SpanUnavailable {
		t.Fatalf("out-of-range span silently anchored: %+v", diagnostic)
	}
	related := source.Relate(filepath.Join(root, "src", "ghost.can"), source.Span{Start: 0, End: 1}, "origin", source.Locate(open, source.Span{Start: 0, End: 7}, errors.New("boom")))
	diagnostic = semanticDiagnostic(graph, open, related)
	if len(diagnostic.Related) != 1 || !strings.Contains(diagnostic.Related[0].Message, "position unavailable") {
		t.Fatalf("unavailable related span silently dropped: %+v", diagnostic)
	}
	if diagnostic.Code != "" || diagnostic.Line != 0 || diagnostic.Start != 0 || diagnostic.End != 7 {
		t.Fatalf("primary span lost beside unavailable related: %+v", diagnostic)
	}
	spanless := semanticDiagnostic(graph, open, errors.New("boom"))
	if spanless.File != open || spanless.Line != 0 || spanless.Code != "" {
		t.Fatalf("spanless failure lost its anchor: %+v", spanless)
	}
}

const bridgeNative = `package app
    provides []
    uses [codec]

connection classifier
    endpoint "http://127.0.0.1:1/systemone"
    auth bearer env "CAN_I27_TOKEN"
    timeout_ms 1000
    metadata
        protocol "typesafe_systemone_v1"
        model "jev-latest"

fn float report
    emits [codec::invalid_data]
    given
        float probability
        str marker
    asserts
        sample: 0.5, "T" => ok 0.5
    ok probability

record choice_weights choice float weights from classifier
    emits [codec::invalid_data]
    confidence as certainty
    asks "Weights"
        first "First" => relay call report(% + certainty, "C")
        second "Second" => relay call report(%, "D")

choice_weights tally = choice_weights(0.1, 0.2, 0.3)

fn float show
    emits []
    given
        int seed
    asserts
        sample: 1 => ok 0.1
    ok tally.first
`

func TestDefinitionGeneratedFields(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/native.can": bridgeNative})
	native := canonical(t, filepath.Join(root, "src/native.can"))
	snapshot, err := CheckSnapshot(root, native, project.NewOverlay())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.World == nil {
		t.Fatalf("no world for generated navigation: %+v", snapshot.Diagnostics)
	}
	line := strings.Count(bridgeNative[:strings.Index(bridgeNative, "tally.first")], "\n")
	character := len("    ok tally.")
	location, ok, err := Definition(snapshot, native, line, character)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("no definition for a generated field")
	}
	landed := bridgeNative
	for i := 0; i < location.Line; i++ {
		landed = landed[strings.IndexByte(landed, '\n')+1:]
	}
	token := landed[location.Start:location.End]
	if location.File != native || token != "first" {
		t.Fatalf("generated field jump landed wrong: %+v token %q", location, token)
	}
	// The generated record name itself jumps to the record token.
	choiceLine := strings.Count(bridgeNative[:strings.Index(bridgeNative, "choice_weights tally")], "\n")
	location, ok, err = Definition(snapshot, native, choiceLine, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("no definition for a generated record")
	}
	landed = bridgeNative
	for i := 0; i < location.Line; i++ {
		landed = landed[strings.IndexByte(landed, '\n')+1:]
	}
	if token := landed[location.Start:location.End]; location.File != native || token != "choice_weights" {
		t.Fatalf("generated record jump landed wrong: %+v token %q", location, token)
	}
}

func TestDefinitionJumps(t *testing.T) {
	root := writeBridgeProject(t, map[string]string{"src/main.can": bridgeMain, "src/second.can": bridgeSecond})
	main := canonical(t, filepath.Join(root, "src/main.can"))
	second := canonical(t, filepath.Join(root, "src/second.can"))
	snapshot, err := CheckSnapshot(root, main, project.NewOverlay())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Diagnostics) != 0 {
		t.Fatalf("fixture does not check: %+v", snapshot.Diagnostics)
	}
	offsetOf := func(text, needle string) (int, int) {
		t.Helper()
		// The cursor sits where the | marker is; the marker is not source.
		plain := strings.Replace(needle, "|", "", 1)
		index := strings.Index(text, plain)
		if index < 0 {
			t.Fatalf("needle %q missing", needle)
		}
		cursor := index + strings.Index(needle, "|")
		line := strings.Count(text[:cursor], "\n")
		character := len(text[strings.LastIndex(text[:cursor], "\n")+1 : cursor])
		return line, character
	}
	jump := func(file, text, needle string) Location {
		t.Helper()
		line, character := offsetOf(text, needle)
		location, ok, err := Definition(snapshot, file, line, character)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("no definition for %q", needle)
		}
		return location
	}
	// A call callee jumps to the function name in its own file.
	at := jump(main, bridgeMain, "|helper(seed)")
	if at.File != main || at.Line != 10 || at.Start != 7 || at.End != 13 {
		t.Fatalf("call jump landed wrong: %+v", at)
	}
	// A type annotation jumps cross-file to the record name.
	at = jump(second, bridgeSecond, "|account_row row")
	if at.File != main || at.Line != 4 || at.Start != 7 || at.End != 18 {
		t.Fatalf("type jump landed wrong: %+v", at)
	}
	// A constructor jumps to the constructed record.
	at = jump(second, bridgeSecond, "acc|ount_row(1,")
	if at.File != main || at.Line != 4 || at.Start != 7 || at.End != 18 {
		t.Fatalf("constructor jump landed wrong: %+v", at)
	}
	// A field behind a module-value receiver jumps to the field name.
	at = jump(second, bridgeSecond, "|display_name\n")
	if at.File != main || at.Line != 6 || at.Start != 8 || at.End != 20 {
		t.Fatalf("field jump landed wrong: %+v", at)
	}
	// A declaration name jumps to itself.
	at = jump(main, bridgeMain, "fn int |helper\n")
	if at.File != main || at.Line != 10 || at.Start != 7 || at.End != 13 {
		t.Fatalf("self jump landed wrong: %+v", at)
	}
	decline := func(file, text, needle string) {
		t.Helper()
		line, character := offsetOf(text, needle)
		_, ok, err := Definition(snapshot, file, line, character)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatalf("offered a definition for %q", needle)
		}
	}
	// Body-local names decline: the parameter shadows any module value.
	decline(main, bridgeMain, "ok |seed\n")
	// Unknown names and primitive spellings decline.
	decline(second, bridgeSecond, "fn |str describe")
	// Positions outside any name decline.
	_, ok, err := Definition(snapshot, main, 0, 0)
	if err != nil || ok {
		t.Fatal("offered a definition for a keyword")
	}
	_, ok, err = Definition(snapshot, filepath.Join(root, "src", "absent.can"), 0, 0)
	if err != nil || ok {
		t.Fatal("offered a definition outside the project")
	}
	_, ok, err = Definition(nil, main, 0, 0)
	if err != nil || ok {
		t.Fatal("offered a definition without a snapshot")
	}
}

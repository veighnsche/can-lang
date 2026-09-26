package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

// G01: whole-document formatting and advisory diagnostics over the
// editor wire. Formatting renders through formatSource, validates the
// candidate like --write through a copied overlay, and replaces the
// whole document; check-pipeline warnings publish with warning
// severity, identical to the CLI stream.

// True-first ordinary Boolean match: checks clean (Q1) but is not
// canonical, so formatting must reorder the arms to false-first.
const g01TrueFirst = `package app
    provides [pick]
    uses []

fn int pick
    emits []
    given
        bool flag
    asserts
        sample: true => ok 1
    match flag
        true => ok 1
        false => ok 0
`

// C8 final-local shape: checks with one advisory warning (Q2); the
// binding stays as written.
const g01FinalLocal = `package app
    provides [forwarded]
    uses []

fn int forwarded
    emits []
    given
        int left
        int right
    asserts
        sample: 1, 2 => ok 3
    int total = left + right
    ok total
`

func g01Formatting(id int, uri string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"textDocument/formatting","params":{"textDocument":{"uri":%s},"options":{"tabSize":4,"insertSpaces":true}}}`,
		id, jsonQuote(uri))
}

func g01Response(t *testing.T, frames []map[string]any, id float64) any {
	t.Helper()
	for _, f := range frames {
		if f["id"] == id {
			if errObj, bad := f["error"]; bad {
				t.Fatalf("request %v errored: %v", id, errObj)
			}
			return f["result"]
		}
	}
	t.Fatalf("no response for id %v in %v", id, frames)
	return nil
}

// TestG01InitializeAdvertisesFormatting pins the capability: editors
// only offer formatting when the server declares it.
func TestG01InitializeAdvertisesFormatting(t *testing.T) {
	frames := runExchange(t, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	result, ok := g01Response(t, frames, 1).(map[string]any)
	if !ok {
		t.Fatalf("initialize result not an object: %v", frames)
	}
	capabilities, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("no capabilities in %v", result)
	}
	if capabilities["documentFormattingProvider"] != true {
		t.Fatalf("formatting not advertised: %v", capabilities)
	}
	if capabilities["textDocumentSync"] != 1.0 || capabilities["definitionProvider"] != true {
		t.Fatalf("existing capabilities regressed: %v", capabilities)
	}
}

// TestG01FormattingReordersBooleanArms pins AU-LSP-format: one
// whole-document edit rewrites true-first arms to false-first.
func TestG01FormattingReordersBooleanArms(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g01TrueFirst})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		didOpen(uri, g01TrueFirst, 1),
		g01Formatting(2, uri),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	edits, ok := g01Response(t, frames, 2).([]any)
	if !ok {
		t.Fatalf("formatting result not an edit list: %v", frames)
	}
	if len(edits) != 1 {
		t.Fatalf("expected one whole-document edit, got %v", edits)
	}
	edit, _ := edits[0].(map[string]any)
	rng, _ := edit["range"].(map[string]any)
	start, _ := rng["start"].(map[string]any)
	end, _ := rng["end"].(map[string]any)
	if start["line"] != 0.0 || start["character"] != 0.0 {
		t.Fatalf("edit does not start at document origin: %v", rng)
	}
	wantEndLine := float64(strings.Count(g01TrueFirst, "\n"))
	if end["line"] != wantEndLine || end["character"] != 0.0 {
		t.Fatalf("edit does not end at document end: %v", rng)
	}
	newText, _ := edit["newText"].(string)
	falseAt, trueAt := strings.Index(newText, "false => ok 0"), strings.LastIndex(newText, "true => ok 1")
	if falseAt < 0 || trueAt < 0 || falseAt > trueAt {
		t.Fatalf("edit did not reorder arms false-first: %q", newText)
	}
	// The edit is the formatter's own fixpoint: reformatting the
	// replacement is a no-op.
	again, err := formatSource(filepath.Join(root, "src/main.can"), newText)
	if err != nil {
		t.Fatalf("formatted edit fails to reformat: %v", err)
	}
	if again != newText {
		t.Fatalf("formatted edit not at fixpoint:\n%s\n--- again ---\n%s", newText, again)
	}
}

// TestG01FormattingFixpointOverWire pins end-to-end stability:
// applying the edit and reformatting yields no further edits.
func TestG01FormattingFixpointOverWire(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g01TrueFirst})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	first := runExchange(t, []string{
		didOpen(uri, g01TrueFirst, 1),
		g01Formatting(2, uri),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	edits, ok := g01Response(t, first, 2).([]any)
	if !ok || len(edits) != 1 {
		t.Fatalf("expected one edit, got %v", first)
	}
	applied, _ := edits[0].(map[string]any)["newText"].(string)
	second := runExchange(t, []string{
		didOpen(uri, g01TrueFirst, 1),
		didChange(uri, applied, 2),
		g01Formatting(3, uri),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	rest, ok := g01Response(t, second, 3).([]any)
	if !ok {
		t.Fatalf("second formatting not an edit list: %v", second)
	}
	if len(rest) != 0 {
		t.Fatalf("canonical buffer still formats: %v", rest)
	}
}

// TestG01FormattingInvalidBufferReturnsNull pins validation before
// edits: buffers that fail to parse or fail to check format to null,
// never to a partial or unchecked edit.
func TestG01FormattingInvalidBufferReturnsNull(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g01TrueFirst})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	unparseable := strings.Replace(g01TrueFirst, "fn int pick", "fn int pick(", 1)
	unchecked := strings.Replace(g01TrueFirst, "false => ok 0", "false => ok missing", 1)
	frames := runExchange(t, []string{
		didOpen(uri, g01TrueFirst, 1),
		didChange(uri, unparseable, 2),
		g01Formatting(2, uri),
		didChange(uri, unchecked, 3),
		g01Formatting(3, uri),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	if result := g01Response(t, frames, 2); result != nil {
		t.Fatalf("unparseable buffer formatted: %v", result)
	}
	if result := g01Response(t, frames, 3); result != nil {
		t.Fatalf("unchecked buffer formatted: %v", result)
	}
	if disk, err := os.ReadFile(filepath.Join(root, "src/main.can")); err != nil || string(disk) != g01TrueFirst {
		t.Fatalf("failed formatting reached disk: %v", err)
	}
}

// TestG01WarningPublishedWithWarningSeverity pins AU-Q2-LSP: the C8
// shape publishes one diagnostic with warning severity, positioned on
// the alias binding, and no error accompanies it.
func TestG01WarningPublishedWithWarningSeverity(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g01FinalLocal})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		didOpen(uri, g01FinalLocal, 1),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 {
		t.Fatalf("expected one advisory diagnostic, got %v", diags)
	}
	diag := diags[0]
	if diag["severity"] != 2.0 {
		t.Fatalf("warning published with severity %v, want 2: %v", diag["severity"], diag)
	}
	if diag["code"] != "CAN-CHECK-UNNECESSARY-LOCAL" {
		t.Fatalf("warning lost its code: %v", diag)
	}
	message, _ := diag["message"].(string)
	if !strings.Contains(message, "accidental alias") || !strings.Contains(message, "total") {
		t.Fatalf("warning lost its message: %v", diag)
	}
	rng, _ := diag["range"].(map[string]any)
	start, _ := rng["start"].(map[string]any)
	wantLine := float64(strings.Count(g01FinalLocal[:strings.Index(g01FinalLocal, "int total")], "\n"))
	if start["line"] != wantLine {
		t.Fatalf("warning anchored at %v, want alias line %v", rng, wantLine)
	}
}

// TestG01WarningMatchesCheckPipeline pins CLI diagnostic parity: the
// published warning is the same finding the check pipeline reports to
// the CLI stream, and the program still checks (warnings never fail a
// build).
func TestG01WarningMatchesCheckPipeline(t *testing.T) {
	root := writeServerProject(t, map[string]string{"src/main.can": g01FinalLocal})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		didOpen(uri, g01FinalLocal, 1),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 {
		t.Fatalf("expected one published diagnostic, got %v", diags)
	}
	graph, err := project.LoadWithOverlay(root, project.NewOverlay())
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckAssertionProgram(graph)
	if err != nil {
		t.Fatalf("warnings-only program failed to check: %v", err)
	}
	if len(program.Warnings) != 1 {
		t.Fatalf("pipeline reported %d warnings, wire published %d", len(program.Warnings), len(diags))
	}
	warning := program.Warnings[0]
	if diags[0]["code"] != warning.Code || diags[0]["message"] != warning.Message {
		t.Fatalf("wire %v != pipeline %s", diags[0], warning.Format())
	}
	if !strings.Contains(warning.Format(), "CAN-CHECK-UNNECESSARY-LOCAL") {
		t.Fatalf("CLI line lost the code: %q", warning.Format())
	}
}

// TestG01WarningsDoNotBlockFormatting pins the advisory contract on
// the authoring flow: a file carrying a warning still formats when
// its content is otherwise valid.
func TestG01WarningsDoNotBlockFormatting(t *testing.T) {
	combined := strings.Replace(g01TrueFirst, "provides [pick]", "provides [pick, forwarded]", 1) +
		"\n" + strings.TrimPrefix(g01FinalLocal, "package app\n    provides [forwarded]\n    uses []\n\n")
	root := writeServerProject(t, map[string]string{"src/main.can": combined})
	uri := uriFromPath(filepath.Join(root, "src/main.can"))
	frames := runExchange(t, []string{
		didOpen(uri, combined, 1),
		g01Formatting(2, uri),
		`{"jsonrpc":"2.0","method":"exit"}`,
	})
	edits, ok := g01Response(t, frames, 2).([]any)
	if !ok || len(edits) != 1 {
		t.Fatalf("warned buffer refused formatting: %v", frames)
	}
	last := lastPublishFor(frames, uri)
	if last == nil {
		t.Fatalf("no publish in %v", frames)
	}
	diags := diagnosticsOf(t, last)
	if len(diags) != 1 || diags[0]["severity"] != 2.0 {
		t.Fatalf("warned buffer misdiagnosed: %v", diags)
	}
}

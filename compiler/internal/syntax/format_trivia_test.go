package syntax

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func formatTrivia(t *testing.T, text string) string {
	t.Helper()
	src, err := source.New("main.can", text)
	if err != nil {
		t.Fatal(err)
	}
	parsed := Parse(src)
	if !parsed.OK() {
		t.Fatalf("parse: %+v", parsed.Diagnostics)
	}
	out, err := FormatTrivia(parsed.File)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func assertIdempotent(t *testing.T, out string) {
	t.Helper()
	again := formatTrivia(t, out)
	if again != out {
		t.Fatalf("not idempotent:\n%s\n---\n%s", out, again)
	}
}

// Trivia survives canonicalization: own-line comments keep attachment,
// trailing comments stay on their lines, block comments stay verbatim,
// and comment-like string contents pass through byte-for-byte.
func TestFormatTriviaPreservesComments(t *testing.T) {
	text := `// header
package   app
    provides [tally]
    uses [codec]  // trailing uses
/* block before */
/// doc tally
fn   int   tally
    emits [codec::invalid_data]
    given
        int seed
    asserts
        sample: 1 => ok 2
    match call number()
        // arm comment
        codec::invalid_data => ok 0  // trailing arm
        ok int v => ok v
/* multi
line
block */
fn str motto
    emits []
    asserts
        sample: => ok "// not a comment /* neither */"
    ok "// not a comment /* neither */"
`
	want := `// header
package app
    provides [tally]
    uses [codec]  // trailing uses
/* block before */
/// doc tally
fn int tally
    emits [codec::invalid_data]
    given
        int seed
    asserts
        sample: 1 => ok 2
    match call number()
        // arm comment
        codec::invalid_data => ok 0  // trailing arm
        ok int v => ok v
/* multi
line
block */
fn str motto
    emits []
    asserts
        sample:  => ok "// not a comment /* neither */"
    ok "// not a comment /* neither */"
`
	out := formatTrivia(t, text)
	if out != want {
		t.Fatalf("trivia misplaced:\n%s\n---\n%s", out, want)
	}
	assertIdempotent(t, out)
}

// Blank runs between rendered lines survive verbatim; leading file
// blanks drop while header comments stay, and EOF keeps one newline.
func TestFormatTriviaPreservesBlankGroups(t *testing.T) {
	text := `


package app
    provides []
    uses []


fn int one
    emits []
    asserts
        sample: => ok 1
    ok 1



// between
fn int two
    emits []
    asserts
        sample: => ok 2
    ok 2


`
	out := formatTrivia(t, text)
	want := `package app
    provides []
    uses []


fn int one
    emits []
    asserts
        sample:  => ok 1
    ok 1



// between
fn int two
    emits []
    asserts
        sample:  => ok 2
    ok 2
`
	if out != want {
		t.Fatalf("blank groups changed:\n%q\n---\n%q", out, want)
	}
	assertIdempotent(t, out)
}

// CRLF input formats to LF; UTF-8 content passes through untouched.
func TestFormatTriviaCRLFAndUTF8(t *testing.T) {
	text := "// caf\u00e9 \U0001D11E\r\npackage app\r\n    provides []\r\n    uses []\r\nfn str motto  // \U0001D11E trailing\r\n    emits []\r\n    asserts\r\n        sample: => ok \"\U0001D11E\"\r\n    ok \"\U0001D11E\"\r\n"
	out := formatTrivia(t, text)
	if strings.Contains(out, "\r") {
		t.Fatalf("CR survived: %q", out)
	}
	for _, want := range []string{"// café \U0001D11E", "// \U0001D11E trailing", `"𝄞"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("UTF-8 lost %q: %q", want, out)
		}
	}
	assertIdempotent(t, out)
}

// A trailing comment on a synthetic layout line stays on that line;
// own-line comments between the keyword and its items stay between.
func TestFormatTriviaSyntheticLines(t *testing.T) {
	text := `package app
    provides []
    uses []
fn int tally
    emits []
    given  // the inputs
        // seeded
        int seed
    asserts
        sample: 1 => ok 2
    match call number()
        when  // the rows
            sample: 1 => ok 1
        ok int v => ok v
`
	out := formatTrivia(t, text)
	for _, want := range []string{"    given  // the inputs\n        // seeded\n        int seed\n", "        when  // the rows\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("synthetic trivia misplaced:\n%s", out)
		}
	}
	assertIdempotent(t, out)
}

// Every shipped example round-trips: trivia formatting reparses cleanly
// and the second format is byte-identical to the first.
func TestFormatTriviaTestdataRoundTrip(t *testing.T) {
	root := "../../testdata/current"
	var entries []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".can") {
			entries = append(entries, path)
		}
		return nil
	})
	if err != nil || len(entries) == 0 {
		t.Fatalf("testdata walk: %v %d", err, len(entries))
	}
	sort.Strings(entries)
	if len(entries) != 55 {
		t.Fatalf("round-trip file inventory: found %d, expected 55", len(entries))
	}
	for _, path := range entries {
		t.Run(strings.TrimPrefix(path, "../../testdata/current/"), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			src, err := source.New(path, string(data))
			if err != nil {
				t.Fatal(err)
			}
			parsed := Parse(src)
			if !parsed.OK() {
				t.Skipf("negative fixture: %+v", parsed.Diagnostics)
			}
			first, err := FormatTrivia(parsed.File)
			if err != nil {
				t.Fatal(err)
			}
			reparsed := Parse(mustSource(t, first))
			if !reparsed.OK() {
				t.Fatalf("formatted output fails to parse: %+v\n%s", reparsed.Diagnostics, first)
			}
			second, err := FormatTrivia(reparsed.File)
			if err != nil {
				t.Fatal(err)
			}
			if second != first {
				t.Fatalf("not idempotent:\n%s\n---\n%s", first, second)
			}
		})
	}
}

func mustSource(t *testing.T, text string) *source.File {
	t.Helper()
	src, err := source.New("formatted.can", text)
	if err != nil {
		t.Fatal(err)
	}
	return src
}

// Triple-quoted strings keep indentation and comment-like bytes exactly;
// nested block comments stay verbatim.
func TestFormatTriviaStringsAndNesting(t *testing.T) {
	text := `package app
    provides [poem]
    uses []
str poem = """
    /* these are literal bytes */
    // still literal
    """
/* Outer comment
   /* a nested comment */
*/
fn str take
    emits []
    given
        str score
    asserts
        sample: "x" => ok "x"
    ok score
`
	out := formatTrivia(t, text)
	for _, want := range []string{"/* these are literal bytes */", "// still literal", "/* Outer comment\n   /* a nested comment */\n*/"} {
		if !strings.Contains(out, want) {
			t.Fatalf("string/comment bytes changed:\n%s", out)
		}
	}
	assertIdempotent(t, out)
}

// Comments inside native declarations, wrappers, judges, questions and
// fixtures keep their attachment; the reparse stays clean.
func TestFormatTriviaNativeForms(t *testing.T) {
	type injected struct {
		anchor string
		text   string
		marks  []string
	}
	cases := map[string][]injected{
		"../../testdata/current/wrap/main.can": {
			{"connection service\n", "connection service  // trailing conn\n", []string{"connection service  // trailing conn"}},
			{"    endpoint \"http://127.0.0.1:1/\"\n", "    // leading endpoint\n    endpoint \"http://127.0.0.1:1/\"\n", []string{"    // leading endpoint\n"}},
			{"        decoded: => ok receipt(7)\n", "        decoded: => ok receipt(7)  // trailing row\n", []string{"decoded:  => ok receipt(7)  // trailing row"}},
			{"    handles emitted\n        ai::invalid_answer as invalid => ok 0.0\n", "    handles emitted\n        // leading arm\n        ai::invalid_answer as invalid => ok 0.0  // trailing arm\n", []string{"        // leading arm\n", "ai::invalid_answer as invalid => ok 0.0  // trailing arm"}},
			{"        int amount\n", "        int amount  // trailing state\n", []string{"int amount  // trailing state"}},
			{"        true \"Yes\" => relay call report(%, \"T\")\n", "        true \"Yes\" => relay call report(%, \"T\")  // trailing option\n", []string{"// trailing option"}},
			{"    call likelihood(\"First\") as float first\n", "    // leading registration\n    call likelihood(\"First\") as float first\n", []string{"    // leading registration\n"}},
			{"                    ok => ok\n", "                    // deepest\n                    ok => ok\n", []string{"                    // deepest\n"}},
		},
		"../../testdata/current/templates/main.can": {
			{"fixture doubled for double\n    given\n", "fixture doubled for double\n    // leading given\n    given\n", []string{"    // leading given\n"}},
			{"        base => ok base + base\n", "        base => ok base + base  // trailing case\n", []string{"base => ok base + base  // trailing case"}},
		},
	}
	for path, edits := range cases {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			for _, edit := range edits {
				if strings.Count(text, edit.anchor) != 1 {
					t.Fatalf("anchor %q found %d times", edit.anchor, strings.Count(text, edit.anchor))
				}
				text = strings.Replace(text, edit.anchor, edit.text, 1)
			}
			out := formatTrivia(t, text)
			for _, edit := range edits {
				for _, mark := range edit.marks {
					if !strings.Contains(out, mark) {
						t.Fatalf("comment %q lost:\n%s", mark, out)
					}
				}
			}
			assertIdempotent(t, out)
		})
	}
}

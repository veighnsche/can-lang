package syntax

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const testHeader = "package demo\n    provides []\n    uses []\n\n"

func parseFile(t *testing.T, text string) *File {
	t.Helper()
	file, err := source.New("test.can", text)
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(file)
	if !result.OK() {
		t.Fatal(result.Diagnostics[0].Format(file))
	}
	formatted := Format(result.File)
	reprinted, err := source.New("rendered.can", formatted)
	if err != nil {
		t.Fatal(err)
	}
	second := Parse(reprinted)
	if !second.OK() {
		t.Fatalf("render failed: %s\n%s", second.Diagnostics[0].Format(reprinted), formatted)
	}
	if next := Format(second.File); next != formatted {
		t.Fatalf("unstable rendering:\n%s\n%s", formatted, next)
	}
	assertSyntaxShape(t, reflect.ValueOf(result.File), reflect.ValueOf(second.File))
	return result.File
}

// Compare node kinds and payloads across rendering, ignoring only source
// locations and retained comment trivia. Stable text alone cannot prove that
// grouping, completion ownership, or argument categories survived formatting.
func assertSyntaxShape(t *testing.T, a, b reflect.Value) {
	t.Helper()
	if a.Type() != b.Type() {
		t.Fatalf("node type changed: %s / %s", a.Type(), b.Type())
	}
	switch a.Kind() {
	case reflect.Pointer, reflect.Interface:
		if a.IsNil() != b.IsNil() {
			t.Fatalf("nil node changed: %s", a.Type())
		}
		if !a.IsNil() {
			assertSyntaxShape(t, a.Elem(), b.Elem())
		}
	case reflect.Struct:
		for i := 0; i < a.NumField(); i++ {
			name := a.Type().Field(i).Name
			if name == "Span" || name == "Source" || name == "Comments" {
				continue
			}
			assertSyntaxShape(t, a.Field(i), b.Field(i))
		}
	case reflect.Slice:
		if a.Len() != b.Len() {
			t.Fatalf("slice length changed: %s", a.Type())
		}
		for i := 0; i < a.Len(); i++ {
			assertSyntaxShape(t, a.Index(i), b.Index(i))
		}
	default:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			t.Fatalf("syntax value changed: %v / %v", a.Interface(), b.Interface())
		}
	}
}

func TestParserLocationsAndGroupingSurviveRendering(t *testing.T) {
	text := testHeader + "fn void main\n    emits []\n    asserts\n        smile: => ok\n    call f(a + b)\n    call g((a + b))\n    call h(())\n    call i((a, b))\n    str text = \"\"\"😀\nsecond line\"\"\" + suffix\n    ok\n"
	for _, program := range []string{text, "\ufeff" + strings.ReplaceAll(text, "\n", "\r\n")} {
		parseFile(t, program)
	}
	bad := testHeader + "fn void main\n    emits []\n    asserts\n        smile: => ok\n    call f(\"😀\",)\n    ok\n"
	file, err := source.New("bad.can", bad)
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(file)
	want := strings.Index(bad, ",)") + 1
	if result.OK() || result.Diagnostics[0].Span.Start != want {
		t.Fatalf("wrong original-byte diagnostic: %+v", result.Diagnostics)
	}
	human, _ := file.Position(want)
	lsp, _ := file.UTF16Position(want)
	if lsp.Character != human.Column {
		t.Fatalf("astral scalar/UTF16 distinction lost: %+v %+v", human, lsp)
	}
}

func specificationProgram(t *testing.T, path, heading string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, section, found := strings.Cut("\n"+string(data), "\n"+heading+"\n")
	if !found {
		t.Fatalf("%s: missing heading %q", path, heading)
	}
	// Keep extraction within this section so drift cannot select a later example.
	section, _, _ = strings.Cut(section, "\n#")
	_, block, found := strings.Cut(section, "\n```text\n")
	if !found {
		t.Fatalf("%s: heading %q has no text code block", path, heading)
	}
	program, _, found := strings.Cut(block, "\n```")
	if !found {
		t.Fatalf("%s: heading %q has an unclosed text code block", path, heading)
	}
	return program + "\n"
}

func TestCoreSpecificationPackage(t *testing.T) {
	program := specificationProgram(t, "../../../docs/syntax-taste/technical-spec.md", "## C10. Complete primitive and callback traces")
	file := parseFile(t, program)
	if len(file.Declarations) != 2 {
		t.Fatal(len(file.Declarations))
	}
	main := file.Declarations[1].(*FunctionDecl)
	match := main.Body.Terminal.(*MatchBody)
	if match.Match.Kind != ValueMatch || len(match.Match.Arms) != 2 {
		t.Fatal(match)
	}
	inner := match.Match.Arms[1].Body.(*MatchBody)
	if !inner.Match.Arms[1].Outcome.StandardFailure {
		t.Fatal(inner)
	}
}

func TestCoreConsumerSpecification(t *testing.T) {
	program := specificationProgram(t, "../../../docs/syntax-taste/technical-spec.md", "### Consumer of generation and judgment")
	parseFile(t, program)
}

func TestCompleteCoreDecisionExamples(t *testing.T) {
	data, err := os.ReadFile("../../../docs/syntax-taste/decisions.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	section := text[strings.Index(text, "## Packages"):strings.Index(text, "## Effects and asynchronous execution")]
	blocks := regexp.MustCompile("(?s)```text\\n(.*?)```").FindAllStringSubmatch(section, -1)
	count := 0
	for _, block := range blocks {
		program := block[1]
		// These are complete declaration examples. Other fences intentionally
		// illustrate header fragments, expressions, or incomplete sections.
		if !strings.Contains(program, "\n    asserts\n") {
			continue
		}
		count++
		t.Run(fmt.Sprintf("example_%02d", count), func(t *testing.T) { parseFile(t, testHeader+program) })
	}
	if count != 13 {
		t.Fatalf("review the complete-example inventory: found %d, expected 13", count)
	}
}

func TestCurrentDeclarationsAndBodies(t *testing.T) {
	program := testHeader + `record finished
record box<item>
    item value
error 1234 missing(str key)
variant result<item>
    box<item>
    finished
int max_retries = 3
fn box<item> wrap<item>
    emits []
    given
        item value
    asserts
        one: 1 => ok box<int>(1)
    ok box<item>(value)
fn int area
    on box<int> shape
    emits []
    asserts
        sample: box<int>(2) => => ok 2
    ok shape.value
fn int lookup
    emits [missing]
    given
        str key
    asserts
        known: "x" => ok 1
    match call external::lookup(key)
        when
            known: "x" => ok 1
        ok
        missing
fn int run
    emits [missing]
    asserts
        done: => ok 1
    match chain
        call lookup("x") as int value
        call log(value)
        ok => do
            int result = match value
                1..5 => 10
                _ => 20
            ok result
        missing => missing("x")
        [_] as str message => ok 0
`
	file := parseFile(t, program)
	if len(file.Declarations) != 9 {
		t.Fatal(len(file.Declarations))
	}
	wrap := file.Declarations[5].(*FunctionDecl)
	if len(wrap.Parameters) != 1 || wrap.Parameters[0].Text != "item" {
		t.Fatal(wrap)
	}
	area := file.Declarations[6].(*FunctionDecl)
	if area.Receiver == nil || area.Assertions[0].Receiver == nil {
		t.Fatal(area)
	}
	chain := file.Declarations[8].(*FunctionDecl).Body.Terminal.(*MatchBody).Match
	if chain.Kind != ChainMatch || len(chain.Chain) != 2 {
		t.Fatal(chain)
	}
	if len(chain.Arms[0].Body.(*DoBody).Block.Steps) != 1 {
		t.Fatal(chain)
	}
}

func TestPatternNodes(t *testing.T) {
	program := testHeader + `fn int inspect
    emits []
    given
        pair[] values
    asserts
        empty: [] => ok 0
    match values
        [] => ok 0
        [pair(0, _) | pair(_, 0), ...rest] => ok 1
        _ => ok 2
`
	file := parseFile(t, program)
	match := file.Declarations[0].(*FunctionDecl).Body.Terminal.(*MatchBody).Match
	array := match.Arms[1].Patterns[0].(*ArrayPattern)
	if array.Rest.Text != "rest" || len(array.Elements[0].(*AlternativePattern).Alternatives) != 2 {
		t.Fatal(array)
	}
}

func TestFileParserRejectsObsoleteAndMalformedGrammar(t *testing.T) {
	for _, fragment := range []string{
		"rev 1\n", "extern fn int thing\n", "dec value = d\"1.0\"\n",
		"record box\n    int value,\n", "error missing()\n", "error 1001 bad(int x,)\n", "variant empty\n",
		"fn int bad\n    emits []\n    ok 1\n",
		"fn int bad\n    asserts\n        a: => ok 1\n    emits []\n    ok 1\n",
		"fn int bad\n    emits []\n    asserts\n        a: => ok 1\n        ok 1\n",
		"fn int bad\n    emits []\n    asserts\n        a: => ok 1\n    ok 1\n    int x = 2\n",
		"fn int bad\n    emits []\n    given []\n    asserts\n        a: => ok 1\n    ok 1\n",
		"fn str bad\n    emits []\n    asserts\n        a: => ok \"\"\"a\nb\"\"\"\n    ok \"x\"\n",
	} {
		file, err := source.New("bad.can", testHeader+fragment)
		if err != nil {
			t.Fatal(err)
		}
		result := Parse(file)
		if result.OK() || result.File != nil {
			t.Fatalf("accepted %q", fragment)
		}
		if err := file.Validate(result.Diagnostics[0].Span); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCompletionRegionsRejectAmbiguousOrUnadmittedForms(t *testing.T) {
	function := testHeader + "fn int bad\n    emits []\n    asserts\n        sample: => ok 1\n"
	for _, body := range []string{
		"    match true\n        true => do\n            ok 1\n        false => ok 0\n",
		"    int value = match call f()\n        ok int x => ok x\n    ok value\n",
		"    int value = match chain\n        call f() as int x\n        ok => ok x\n    ok value\n",
		"    int value = match true\n        true => ok 1\n        false => ok 0\n    ok value\n",
		"    match call f()\n        _ => ok 0\n",
		"    match call f()\n        missing(_) => ok 0\n",
		"    match call f()\n        missing | other => ok 0\n",
		"    match call f()\n        ok int value\n",
		"    match call f()\n        [_]\n",
		"    match call f()\n        ok => ok 1\n        when\n            sample: => ok 1\n",
		"    match chain\n        ok => ok 1\n",
		"    match 1\n        1.0..2.0 => ok 1\n        _ => ok 0\n",
		"    match [1]\n        [int value] => ok value\n        _ => ok 0\n",
	} {
		file, err := source.New("bad.can", function+body)
		if err != nil {
			t.Fatal(err)
		}
		if result := Parse(file); result.OK() {
			t.Fatalf("accepted malformed region:\n%s", body)
		}
	}
}

func FuzzFileParser(f *testing.F) {
	f.Add(testHeader)
	f.Add(testHeader + "fn int one\n    emits []\n    asserts\n        one: => ok 1\n    ok 1\n")
	fixture, err := os.ReadFile("../../testdata/current/parser/offline.can")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(string(fixture))
	f.Fuzz(func(t *testing.T, text string) {
		file, err := source.New("fuzz.can", text)
		if err != nil {
			return
		}
		result := Parse(file)
		for _, d := range result.Diagnostics {
			if err := file.Validate(d.Span); err != nil {
				t.Fatal(err)
			}
		}
		if result.OK() && result.File == nil {
			t.Fatal("missing file")
		}
		if result.OK() {
			printed := Format(result.File)
			rendered, err := source.New("rendered.can", printed)
			if err != nil {
				t.Fatal(err)
			}
			again := Parse(rendered)
			if !again.OK() {
				t.Fatalf("rendered invalid syntax: %v\n%s", again.Diagnostics, printed)
			}
			if Format(again.File) != printed {
				t.Fatal("unstable syntax rendering")
			}
		}
	})
}

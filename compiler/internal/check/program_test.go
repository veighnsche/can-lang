package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func programFixture(t *testing.T, files map[string]string) (*Program, error) {
	t.Helper()
	root := t.TempDir()
	files["can.project.json"] = `{"source_root":"src","error_registry":"can.errors.json"}`
	files["can.errors.json"] = `{"active":[],"retired":[]}`
	for name, text := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		return nil, err
	}
	return CheckProgram(graph)
}

// nativeRawFixture is a minimal schema-valid response fixture for check-only
// inline tests. Request comparison never runs there; the response outcome
// satisfies the request/decoder coverage gate structurally.
func nativeRawFixture(target string) string {
	return `{"schema":"can.native-fixture.v1","target":"` + target + `","environment":{},"exchange":{"request":{"method":"POST","url":"http://localhost:1/","headers":[],"body":{"bytes_base64":""}},"outcome":{"response":{"status":200,"headers":[["content-type","application/json"]],"body_base64":"e30="}}}}`
}

// withNativeRaw stages one minimal fixture per named declaration for inline
// package-app sources.
func withNativeRaw(files map[string]string, names ...string) map[string]string {
	for _, name := range names {
		files["src/fixtures/"+name+".json"] = nativeRawFixture("can.project.root/app::" + name)
	}
	return files
}

// testdataFixtures stages the committed raw fixtures for an area next to the
// staged sources so using-raw rows resolve exactly as in real projects.
func testdataFixtures(t *testing.T, files map[string]string, area string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir("../../testdata/current/" + area + "/fixtures")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile("../../testdata/current/" + area + "/fixtures/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		files["src/fixtures/"+entry.Name()] = string(data)
	}
	return files
}

const programHeader = "package app\n    provides []\n    uses []\n"
const programMain = "fn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n"

func TestProgramConnectsFilesAndInitialization(t *testing.T) {
	p, err := programFixture(t, map[string]string{
		"src/main.can":   programHeader + "int answer = later + 1\n" + programMain + "    call check(answer, arguments.length)\n    ok\n",
		"src/helper.can": programHeader + "int later = 41\nfn void check\n    emits []\n    given\n        int answer\n        int count\n    asserts\n        sample: 42, 0 => ok\n    match answer is 42 and count >= 0\n        true => ok\n        false => ok\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Functions) != 2 || p.Entry == nil || len(p.Initializers) != 2 {
		t.Fatalf("incomplete program: %+v", p)
	}
	if !strings.HasSuffix(p.Initializers[0].Identity, "::later") || !strings.HasSuffix(p.Initializers[1].Identity, "::answer") {
		t.Fatal("initializers are not dependency ordered")
	}
	if got := p.Entry.Region.Inputs; len(got) != 1 || !strings.HasSuffix(got[0].Identity, "/input/arguments") {
		t.Fatal("main input identity lost")
	}
}
func TestProgramRefusesInvalidEntryAndBodies(t *testing.T) {
	for name, source := range map[string]string{
		"missing entry":        programHeader,
		"wrong result":         programHeader + strings.Replace(programMain, "fn void main", "fn int main", 1) + "    ok 1\n",
		"wrong input":          programHeader + strings.Replace(programMain, "str[] arguments", "int arguments", 1) + "    ok\n",
		"static mismatch":      programHeader + programMain + "    ok arguments\n",
		"unknown call":         programHeader + programMain + "    call missing()\n    ok\n",
		"bad unused body":      programHeader + programMain + "    ok\nfn int unused\n    emits []\n    asserts\n        sample: => ok 1\n    ok false\n",
		"initialization cycle": programHeader + "int first = second\nint second = first\n" + programMain + "    ok\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": source}); err == nil {
				t.Fatal("invalid program accepted")
			}
		})
	}
}

func TestProgramChecksMethodsAndNestedRegions(t *testing.T) {
	_, err := programFixture(t, map[string]string{"src/main.can": programHeader + `record item
    int number
item initial = item(7)
fn int read
    on item self
    emits []
    asserts
        sample: item(7) => => ok 7
    ok self.number
` + programMain + `    match call initial.read()
        [_] as standard_failure failure => match failure.message
            "" => ok
            _ => ok
        ok int number => match number
            7 => ok
            _ => ok
`})
	if err != nil {
		t.Fatal(err)
	}
}

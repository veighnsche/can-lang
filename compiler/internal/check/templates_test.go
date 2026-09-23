package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

const templateTarget = `fn int double
    emits []
    given
        int value
    asserts
        sample: 2 => ok 4
    ok value + value
fn int pick
    emits []
    given
        int first
        int second
    asserts
        sample: 1, 2 => ok 1
    ok first
`

func templateFixtureSteps(t *testing.T, program *Program, name string) ir.InvocationStep {
	t.Helper()
	for _, function := range program.Functions {
		if function.Symbol.Name == name {
			return function.Region.Body.Terminal.Match.Call.Steps[0]
		}
	}
	t.Fatalf("consumer %s missing", name)
	return ir.InvocationStep{}
}

func TestFixtureTemplateExpansion(t *testing.T) {
	text := programHeader + templateTarget + `fixture doubled for double
    given
        int base
    cases
        base => ok base + base
        3 => ok 6
fn int first_use
    emits []
    asserts
        sample: => ok 4
    match call double(2)
        when
            sample: use doubled(2)
        ok int got => ok got
fn int second_use
    emits []
    asserts
        sample: => ok 6
    match call double(9)
        when
            sample: use doubled(9)
            sample: 1 => ok 2
        ok int got => ok got
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	first := templateFixtureSteps(t, program, "first_use")
	if len(first.Fixtures.Rows) != 2 {
		t.Fatalf("first use expanded %d rows", len(first.Fixtures.Rows))
	}
	for _, row := range first.Fixtures.Rows {
		if row.Selector != "sample" {
			t.Fatalf("expanded row lost its selector: %+v", row)
		}
	}
	second := templateFixtureSteps(t, program, "second_use")
	if len(second.Fixtures.Rows) != 3 {
		t.Fatalf("second use expanded %d rows", len(second.Fixtures.Rows))
	}
	// Literal rows keep their relative position after expansion.
	last := second.Fixtures.Rows[2]
	if len(last.Arguments) != 1 {
		t.Fatalf("literal row moved: %+v", last)
	}
}

func TestFixtureTemplateRejects(t *testing.T) {
	base := programHeader + templateTarget + `fixture doubled for double
    given
        int base
    cases
        base => ok base + base
        3 => ok 6
`
	consumer := func(row string) string {
		return `fn int consumer
    emits []
    asserts
        sample: => ok 4
    match call double(2)
        when
            ` + row + `
        ok int got => ok got
` + programMain + "    ok\n"
	}
	cases := map[string]struct {
		text string
		want string
	}{
		"unknown template":    {base + consumer("sample: use missing(2)"), "no eligible declaration"},
		"arity":               {base + consumer("sample: use doubled(2, 3)"), "expects 1 template arguments"},
		"argument type":       {base + consumer(`sample: use doubled("x")`), "template argument base"},
		"executable argument": {base + consumer("sample: use doubled(call double(2))"), "is executable"},
		"captured local":      {base + strings.Replace(consumer("sample: use doubled(first)"), "match call double(2)", "int first = 2\n    match call double(first)", 1), `no eligible declaration for "first"`},
		"captured fn input": {base + `fn int consumer
    emits []
    given
        int second
    asserts
        sample: 5 => ok 4
    match call double(2)
        when
            sample: use doubled(second)
        ok int got => ok got
` + programMain + "    ok\n", `no eligible declaration for "second"`},
		"wrong exact target": {base + strings.Replace(consumer("sample: use doubled(2)"), "match call double(2)", "match call pick(1, 2)", 1), "not the invoked"},
		"reference call site": {base + `fn int consumer
    emits []
    asserts
        sample: => ok 4
    callable int (int) emits [] action = callable double
    match call action(2)
        when
            sample: use doubled(2)
        ok int got => ok got
` + programMain + "    ok\n", "not the invoked"},
		"use in attached row":   {base + "fn int bad\n    emits []\n    asserts\n        sample: use doubled(2)\n    ok 1\n" + programMain + "    ok\n", "lexical when tables only"},
		"question target":       {base + "fixture q for question\n    cases\n        1 => ok 1\n" + consumer("sample: 2 => ok 4"), "no invokable fixture target"},
		"callable value target": {base + "callable int () emits [] thunk = callable double\nfixture v for thunk\n    cases\n        1 => ok 1\n" + consumer("sample: 2 => ok 4"), "not a fixture target declaration"},
		"duplicate parameter":   {programHeader + templateTarget + "fixture doubled for double\n    given\n        int base\n        int base\n    cases\n        base => ok base\n" + consumer("sample: 2 => ok 4"), `duplicate field "base"`},
		"executable case":       {programHeader + templateTarget + "fixture doubled for double\n    given\n        int base\n    cases\n        call double(base) => ok 1\n" + consumer("sample: 2 => ok 4"), "case 1"},
		"executable outcome":    {programHeader + templateTarget + "fixture doubled for double\n    given\n        int base\n    cases\n        base => ok call double(base)\n" + consumer("sample: 2 => ok 4"), "case 1"},
		"using failure in case": {programHeader + templateTarget + "fixture doubled for double\n    cases\n        1 => ok 1\n            using failure native http::status_error(1, [])\n" + consumer("sample: 2 => ok 4"), "confined to attached wrapper assertions"},
	}
	for name, kase := range cases {
		t.Run(strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/main.can": kase.text})
			if err == nil || !strings.Contains(err.Error(), kase.want) {
				t.Fatalf("%s: got %v, want %q", name, err, kase.want)
			}
		})
	}
}

func TestFixtureShadowedStaticKeepsStaticMeaning(t *testing.T) {
	// A use argument naming a package value keeps its static meaning
	// even when a runtime input shadows the name: no capture, no
	// re-resolution.
	text := programHeader + "int limit = 9\n" + templateTarget + "fixture doubled for double\n    given\n        int base\n    cases\n        base => ok base\n" + `fn int consumer
    emits []
    given
        int limit
    asserts
        sample: 5 => ok 4
    match call double(2)
        when
            sample: use doubled(limit)
        ok int got => ok got
` + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	step := templateFixtureSteps(t, program, "consumer")
	if len(step.Fixtures.Rows) != 1 || len(step.Fixtures.Rows[0].Arguments) != 1 {
		t.Fatalf("shadowed use expanded %+v", step.Fixtures.Rows)
	}
	arg := step.Fixtures.Rows[0].Arguments[0]
	if arg.Kind != ir.Binding {
		t.Fatalf("shadowed argument lost its binding: %+v", arg)
	}
	var value *ir.Expression
	for _, prep := range step.Fixtures.Rows[0].Prepare {
		if prep.Local.Identity == arg.Text {
			value = prep.Value
		}
	}
	if value == nil {
		value = arg
	}
	if value.Kind != ir.Binding || value.Text != "can.project.root/app::limit" {
		t.Fatalf("shadowed argument rebound: %+v", value)
	}
}

func TestFixtureNativeTargetWithRaw(t *testing.T) {
	// The template lives beside its consumer; the raw case runs the
	// target plus policy shape through the ordinary raw path.
	lib := "package app\n    provides []\n    uses [http, codec]\nfixture fetched for load\n    given\n        int count\n    cases\n        => ok receipt(count)\n        => ok receipt(7)\n            using raw \"fixtures/fetched.json\"\n"
	main := wrapHeader + wrapLoad + `fn receipt consumer
    emits [http::request_failed]
    asserts
        sample: => ok receipt(7)
    match call load()
        when
            sample: use fetched(7)
        http::request_failed
        ok receipt got => ok got
` + programMain + "    ok\n"
	files := withNativeRaw(map[string]string{"src/main.can": main, "src/helpers.can": lib}, "load")
	files["src/fixtures/fetched.json"] = nativeRawFixture("can.project.root/app::load")
	program, err := programFixture(t, files)
	if err != nil {
		t.Fatal(err)
	}
	step := templateFixtureSteps(t, program, "consumer")
	if len(step.Fixtures.Rows) != 2 {
		t.Fatalf("native use expanded %d rows", len(step.Fixtures.Rows))
	}
	if step.Fixtures.Rows[0].Raw != nil {
		t.Fatal("plain case gained a raw exchange")
	}
	raw := step.Fixtures.Rows[1].Raw
	if raw == nil || raw.Operation != "can.project.root/app::load" {
		t.Fatalf("raw case lost its root operation: %+v", raw)
	}
	if _, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": main}, "load")); err == nil {
		t.Fatal("use of an undeclared template passed")
	}
}

func TestFixtureImportedTemplateResolvesRawAtDefinition(t *testing.T) {
	// The imported template's raw path resolves relative to its own
	// defining directory; no copy exists beside the use site.
	vendor := "package helpers\n    provides [fetched, receipt, load, service]\n    uses [http, codec]\nrecord receipt\n    int count\nconnection service\n    endpoint \"http://localhost:1\"\n    timeout_ms 1000\nfetch receipt load from service\n    emits [http::request_failed]\n    asserts\n        decoded: => ok receipt(7)\n            using raw \"fixtures/load.json\"\n    get \"/load\"\nfixture fetched for load\n    given\n        int count\n    cases\n        => ok receipt(count)\n        => ok receipt(7)\n            using raw \"fixtures/fetched.json\"\n"
	main := "package app\n    provides []\n    uses [http, codec, helpers]\nfn helpers::receipt consumer\n    emits [http::request_failed]\n    asserts\n        sample: => ok helpers::receipt(7)\n    match call helpers::load()\n        when\n            sample: use helpers::fetched(7)\n        http::request_failed\n        ok helpers::receipt got => ok got\n" + programMain + "    ok\n"
	program, err := programFixtureWithVendor(t, map[string]string{"src/main.can": main}, map[string]string{
		"src/lib.can":               vendor,
		"src/fixtures/load.json":    nativeRawFixture("can.project.dependency/vendor/helpers::load"),
		"src/fixtures/fetched.json": nativeRawFixture("can.project.dependency/vendor/helpers::load"),
	})
	if err != nil {
		t.Fatal(err)
	}
	step := templateFixtureSteps(t, program, "consumer")
	if len(step.Fixtures.Rows) != 2 || step.Fixtures.Rows[1].Raw == nil {
		t.Fatalf("imported use expanded %+v", step.Fixtures.Rows)
	}
}

// programFixtureWithVendor stages a main project with one locked vendor
// dependency. Vendor sources live in their own directory, so raw paths
// prove definition-relative resolution.
func programFixtureWithVendor(t *testing.T, main, vendor map[string]string) (*Program, error) {
	t.Helper()
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", `{"source_root":"src","dependencies":{"vendor":"vendor"},"error_registry":"can.errors.json"}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("vendor/can.project.json", `{"source_root":"src","error_registry":"can.errors.json"}`)
	write("vendor/can.errors.json", `{"active":[],"retired":[]}`)
	for name, text := range main {
		write(name, text)
	}
	var sources []project.SourceBytes
	for name, text := range vendor {
		write("vendor/"+name, text)
		if strings.HasSuffix(name, ".can") {
			sources = append(sources, project.SourceBytes{Path: strings.TrimPrefix(name, "src/"), Bytes: []byte(text)})
		}
	}
	manifestData, err := os.ReadFile(filepath.Join(root, "vendor/can.project.json"))
	if err != nil {
		t.Fatal(err)
	}
	registryData, err := os.ReadFile(filepath.Join(root, "vendor/can.errors.json"))
	if err != nil {
		t.Fatal(err)
	}
	registry, err := project.ParseRegistry(registryData)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := project.SourceDigest(sources)
	if err != nil {
		t.Fatal(err)
	}
	lock, err := json.Marshal(map[string]any{"dependencies": map[string]any{"vendor": map[string]any{"path": "vendor", "manifest_sha256": project.Digest(manifestData), "source_sha256": digest, "error_registry": registry}}})
	if err != nil {
		t.Fatal(err)
	}
	write("can.lock.json", string(lock))
	graph, err := project.Load(root)
	if err != nil {
		return nil, err
	}
	return CheckProgram(graph)
}

func TestFixtureJudgeGroupedTarget(t *testing.T) {
	text := nativeHeader + nativeClassifier + nativeQuestion + nativeJudge + `fixture judged for assess
    cases
        ("x") => ok true
fn bool consumer
    emits [http::request_failed, ai::invalid_question, ai::invalid_answer]
    asserts
        sample: => ok true
    match call assess(("x"))
        when
            sample: use judged()
        http::request_failed
        ai::invalid_question
        ai::invalid_answer
        ok bool got => ok got
` + programMain + "    ok\n"
	program, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": text}, "assess"))
	if err != nil {
		t.Fatal(err)
	}
	step := templateFixtureSteps(t, program, "consumer")
	if len(step.Fixtures.Rows) != 1 || len(step.Fixtures.Rows[0].Arguments) != 1 {
		t.Fatalf("grouped use expanded %+v", step.Fixtures.Rows)
	}
}

func TestFixtureGenericTarget(t *testing.T) {
	text := "package app\n    provides []\n    uses []\nfn item identity<item>\n    emits []\n    given\n        item value\n    asserts\n        integer: 3 => ok 3\n    ok value\nfixture ints for identity<int>\n    cases\n        3 => ok 3\nfn int consumer\n    emits []\n    asserts\n        sample: => ok 3\n    match call identity<int>(3)\n        when\n            sample: use ints()\n        ok int got => ok got\n" + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if step := templateFixtureSteps(t, program, "consumer"); len(step.Fixtures.Rows) != 1 {
		t.Fatalf("generic use expanded %d rows", len(step.Fixtures.Rows))
	}
	untyped := strings.Replace(text, "fixture ints for identity<int>", "fixture ints for identity", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": untyped}); err == nil || !strings.Contains(err.Error(), "concrete type arguments") {
		t.Fatalf("untyped generic target admitted: %v", err)
	}
	textured := strings.Replace(text, "fixture ints for identity<int>", "fixture ints for identity<int, str>", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": textured}); err == nil || !strings.Contains(err.Error(), "concrete type arguments") {
		t.Fatalf("over-applied generic target admitted: %v", err)
	}
	strung := strings.Replace(text, "call identity<int>(3)", "call identity<str>(\"x\")", 1)
	strung = strings.Replace(strung, "sample: => ok 3", "sample: => ok 3", 1)
	if _, err := programFixture(t, map[string]string{"src/main.can": strung}); err == nil || !strings.Contains(err.Error(), "not the invoked") {
		t.Fatalf("cross-specialization use admitted: %v", err)
	}
}

func TestFixtureWrapperTarget(t *testing.T) {
	text := wrapHeader + wrapLoad + `wrap cached from load
    emits calculated
    asserts
        sample: => ok receipt(0)
            using failure native http::status_error(404, [])
    handles native
        http::status_error => ok receipt(0)
fixture recent for cached
    cases
        => ok receipt(0)
fixture older for load
    cases
        => ok receipt(7)
fn receipt consumer
    emits [http::request_failed]
    asserts
        sample: => ok receipt(0)
    match call cached()
        when
            sample: use recent()
        http::request_failed
        ok receipt got => ok got
` + programMain + "    ok\n"
	program, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": text}, "load"))
	if err != nil {
		t.Fatal(err)
	}
	if step := templateFixtureSteps(t, program, "consumer"); len(step.Fixtures.Rows) != 1 {
		t.Fatalf("wrapper use expanded %d rows", len(step.Fixtures.Rows))
	}
	// A base-operation template never matches a derived wrapper site even
	// though the signatures are identical.
	based := strings.Replace(text, "sample: use recent()", "sample: use older()", 1)
	if _, err := programFixture(t, withNativeRaw(map[string]string{"src/main.can": based}, "load")); err == nil || !strings.Contains(err.Error(), "not the invoked") {
		t.Fatalf("base template at wrapper site admitted: %v", err)
	}
}

func TestFixtureCatalogueTarget(t *testing.T) {
	text := "package app\n    provides []\n    uses [bytes, codec]\nfixture decoded for bytes::from_utf8\n    cases\n        \"T\" => codec::invalid_data(\"text\", \"unit\")\nfn bytes::buffer consumer\n    emits [codec::invalid_data]\n    asserts\n        sample: => codec::invalid_data(\"text\", \"unit\")\n    match call bytes::from_utf8(\"T\")\n        when\n            sample: use decoded()\n        codec::invalid_data\n        ok bytes::buffer got => ok got\n" + programMain + "    ok\n"
	program, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if step := templateFixtureSteps(t, program, "consumer"); len(step.Fixtures.Rows) != 1 {
		t.Fatalf("catalogue use expanded %d rows", len(step.Fixtures.Rows))
	}
}

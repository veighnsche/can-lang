package syntax

import (
	"testing"
)

const wrapSource = `wrap cached_load from load_json
    emits calculated
    asserts
        absent: => ok receipt(0)
            using failure native http::status_error(404, [])
        stored: receipt(7) => ok receipt(7)
            using raw "fixtures/cached.json"
    handles native
        http::status_error as failed => match failed.status
            404 => ok receipt(0)
            _ => inherit
    handles emitted
        cache_failed => http::request_failed(http::status_error(429, []))
`

func TestWrapDeclarationParsing(t *testing.T) {
	result := nativeParse(t, wrapSource)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	decl, ok := result.File.Declarations[0].(*WrapDecl)
	if !ok {
		t.Fatalf("wrap parsed as %T", result.File.Declarations[0])
	}
	if decl.Name.Text != "cached_load" || decl.Base.Name != "load_json" || decl.Calculated.Text != "calculated" {
		t.Fatal("wrap header lost its name, base or marker")
	}
	if len(decl.Assertions) != 2 || !decl.HasNative || !decl.HasEmitted {
		t.Fatal("wrap sections missing")
	}
	injected := decl.Assertions[0].Mode
	if injected == nil || injected.Failure == nil || injected.Failure.Origin.Text != "native" {
		t.Fatal("using failure native mode missing")
	}
	if _, ok := injected.Failure.Value.(*ConstructorExpr); !ok {
		t.Fatalf("injected value parsed as %T", injected.Failure.Value)
	}
	if decl.Assertions[1].Mode == nil || decl.Assertions[1].Mode.Failure != nil || decl.Assertions[1].Mode.Raw.Value != "fixtures/cached.json" {
		t.Fatal("wrapper using raw mode missing")
	}
	if len(decl.Native) != 1 || decl.Native[0].Pattern.Alias == nil || decl.Native[0].Pattern.Alias.Text != "failed" {
		t.Fatal("native arm lost its error alias")
	}
	arms := decl.Native[0].Body.(*MatchBody).Match.Arms
	if _, ok := arms[1].Body.(*InheritBody); !ok {
		t.Fatalf("fallback arm parsed as %T", arms[1].Body)
	}
	if len(decl.Emitted) != 1 {
		t.Fatal("emitted arm missing")
	}
	formatted := Format(result.File)
	for _, want := range []string{
		"wrap cached_load from load_json",
		"emits calculated",
		"using failure native http::status_error(404, [])",
		"handles native",
		"http::status_error as failed => match",
		"_ => inherit",
		"handles emitted",
	} {
		if !contains(formatted, want) {
			t.Fatalf("formatted output omits %q:\n%s", want, formatted)
		}
	}
}

func TestWrapDeclarationRejects(t *testing.T) {
	cases := map[string]string{
		"explicit list":      "    emits [http::request_failed]\n",
		"missing handles":    "",
		"duplicate native":   "    handles native\n        http::status_error => ok receipt(0)\n    handles native\n        http::timeout => ok receipt(0)\n",
		"emitted first":      "    handles emitted\n        cache_failed => ok receipt(0)\n    handles native\n        http::timeout => ok receipt(0)\n",
		"ok arm":             "    handles native\n        ok => ok receipt(0)\n",
		"standard arm":       "    handles native\n        [_] => ok receipt(0)\n",
		"missing origin":     "    handles native\n        http::timeout => ok receipt(0)\n",
		"empty native table": "    handles native\n",
	}
	for name, tables := range cases {
		t.Run(name, func(t *testing.T) {
			text := "wrap cached_load from load_json\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure native http::status_error(404, [])\n" + tables
			if name == "missing origin" {
				text = "wrap cached_load from load_json\n    emits calculated\n    asserts\n        sample: => ok receipt(0)\n            using failure http::status_error(404, [])\n" + tables
			}
			if nativeParse(t, text).OK() {
				t.Fatalf("invalid wrap admitted: %s", name)
			}
		})
	}
}

func TestEmitsCalculatedRejectedElsewhere(t *testing.T) {
	for _, text := range []string{
		"fn int double\n    emits calculated\n    given\n        int value\n    asserts\n        sample: 1 => ok 2\n    ok value\n",
		"fetch str load from service\n    emits calculated\n    asserts\n        sample: => ok \"x\"\n            using raw \"fixtures/load.json\"\n    get \"/\"\n",
	} {
		if nativeParse(t, text).OK() {
			t.Fatalf("emits calculated admitted outside wrap:\n%s", text)
		}
	}
}

func TestInheritOutsideCompletionRejects(t *testing.T) {
	text := "fn int pick\n    emits []\n    given\n        int value\n    asserts\n        sample: 1 => ok 1\n    int chosen = match value\n        1 => inherit\n        _ => value\n    ok chosen\n"
	if nativeParse(t, text).OK() {
		t.Fatal("inherit admitted in a value position")
	}
}

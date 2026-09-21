package resolve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if _, ok := files["can.project.json"]; !ok {
		files["can.project.json"] = `{"source_root":"src","error_registry":"can.errors.json"}`
	}
	if _, ok := files["can.errors.json"]; !ok {
		files["can.errors.json"] = `{"active":[],"retired":[]}`
	}
	for name, text := range files {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
func buildFiles(t *testing.T, files map[string]string) (*World, error) {
	t.Helper()
	graph, err := project.Load(writeFiles(t, files))
	if err != nil {
		return nil, err
	}
	return Build(graph)
}
func header(pkg, provides, uses string) string {
	return "package " + pkg + "\n    provides [" + provides + "]\n    uses [" + uses + "]\n"
}
func function(name, typ, inputs, body string) string {
	text := "fn " + typ + " " + name + "\n    emits []\n"
	if inputs != "" {
		text += "    given\n" + inputs
	}
	return text + "    asserts\n        example: => ok\n" + body
}
func fileNamed(w *World, pkg, name string) *File {
	for source, file := range w.Files {
		if source.Package.Name == pkg && source.Name == name {
			return file
		}
	}
	return nil
}

func TestFlatPackagesFileImportsAndIndependentIdentities(t *testing.T) {
	w, err := buildFiles(t, map[string]string{
		"src/alpha/shared.can": header("alpha", "item", "beta as other") + "record item\n    other::item value\n",
		"src/alpha/second.can": header("alpha", "holder", "") + "record holder\n    item value\n",
		"src/beta/shared.can":  header("beta", "item", "alpha") + "record item\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	alpha := w.Packages["alpha"].Scope.Symbols["item"]
	beta := w.Packages["beta"].Scope.Symbols["item"]
	if alpha == beta || alpha.ID == beta.ID || alpha.Source.OutputPath == beta.Source.OutputPath {
		t.Fatal("same basename/name merged identities")
	}
	first := fileNamed(w, "alpha", "shared.can")
	second := fileNamed(w, "alpha", "second.can")
	if got, err := first.Lookup(nil, syntax.QualifiedName{Package: "other", Name: "item"}, TypeUse); err != nil || got != beta {
		t.Fatalf("file alias failed: %v", err)
	}
	if _, err := second.Lookup(nil, syntax.QualifiedName{Package: "other", Name: "item"}, TypeUse); err == nil {
		t.Fatal("file alias leaked")
	}
	if got, err := second.Lookup(nil, syntax.QualifiedName{Name: "item"}, TypeUse); err != nil || got != alpha {
		t.Fatal("multi-file package table missing")
	}
}

func TestEligibleKindLookupSkipsIneligibleShadowing(t *testing.T) {
	w, err := buildFiles(t, map[string]string{"src/main.can": header("app", "item, transform", "") + "record item\n" + function("transform", "int", "", "    ok 1\n")})
	if err != nil {
		t.Fatal(err)
	}
	file := fileNamed(w, "app", "main.can")
	local := NewScope(file.Scope)
	value := &Symbol{Name: "item", Kind: Value}
	if err := local.Define(value); err != nil {
		t.Fatal(err)
	}
	for _, usage := range []Usage{TypeUse, ConstructorUse} {
		got, err := local.Lookup("item", usage)
		if err != nil || got.Kind != Record {
			t.Fatalf("eligible lookup: %+v %v", got, err)
		}
	}
	if got, err := local.Lookup("item", ValueUse); err != nil || got != value {
		t.Fatal("value lookup did not use local")
	}
	if err := local.Define(&Symbol{Name: "transform", Kind: Value}); err != nil {
		t.Fatal(err)
	}
	if got, err := local.Lookup("transform", CallUse); err != nil || got.Kind != Function {
		t.Fatal("noncallable local displaced callable declaration")
	}
	nested := NewScope(local)
	callable := &Symbol{Name: "transform", Kind: Value, Callable: true}
	if err := nested.Define(callable); err != nil {
		t.Fatal(err)
	}
	if got, err := nested.Lookup("transform", CallUse); err != nil || got != callable {
		t.Fatal("callable local did not shadow function")
	}
	if _, err := nested.Lookup("item", CallUse); err == nil {
		t.Fatal("ineligible constructor accepted as call")
	}
	if err := nested.Define(&Symbol{Name: "append", Kind: Value}); err == nil {
		t.Fatal("prelude shadowing")
	}
	if err := nested.Define(&Symbol{Name: "transform", Kind: Value}); err == nil {
		t.Fatal("same-scope duplicate")
	}
}

func TestQualifiedLookupBypassesLocalValuesAndKeepsOpacity(t *testing.T) {
	w, err := buildFiles(t, map[string]string{"src/main.can": header("app", "", "http as transport, bytes, option") + "record holder\n    transport::header value\n"})
	if err != nil {
		t.Fatal(err)
	}
	f := fileNamed(w, "app", "main.can")
	scope := NewScope(f.Scope)
	if err := scope.Define(&Symbol{Name: "transport", Kind: Value}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Lookup(scope, syntax.QualifiedName{Package: "transport", Name: "header"}, ConstructorUse); err != nil {
		t.Fatal(err)
	}
	for _, q := range []syntax.QualifiedName{{Package: "bytes", Name: "buffer"}, {Package: "option", Name: "value"}} {
		if _, err := f.Lookup(scope, q, ConstructorUse); err == nil {
			t.Fatal("opaque/variant constructor admitted")
		}
		if _, err := f.Lookup(scope, q, TypeUse); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReceiverPackageMethodOwnership(t *testing.T) {
	method := "fn int area\n    on panel shape\n    emits []\n    asserts\n        sample: panel() => => ok 1\n    ok 1\n"
	w, err := buildFiles(t, map[string]string{
		"src/panels/main.can": header("panels", "panel, area", "") + "record panel\n" + method,
		"src/app/main.can":    header("app", "", "panels") + "record holder\n    panels::panel value\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	receiver := w.Packages["panels"].Scope.Symbols["panel"]
	file := fileNamed(w, "app", "main.can")
	if got, err := file.Method(receiver, "area"); err != nil || got.Receiver != receiver {
		t.Fatalf("method lookup: %v", err)
	}
	if _, err := file.Lookup(nil, syntax.QualifiedName{Package: "panels", Name: "area"}, CallUse); err == nil {
		t.Fatal("method callable statically without receiver")
	}
	delete(file.Imports, "panels")
	if _, err := file.Method(receiver, "area"); err == nil {
		t.Fatal("unimported method admitted")
	}
}

func TestResolutionRejectsScopeAndSignatureViolations(t *testing.T) {
	base := func(text string) map[string]string { return map[string]string{"src/main.can": text} }
	cases := []map[string]string{
		base(header("app", "", "") + "record item\nrecord item\n"),
		base(header("app", "", "") + "record append\n"),
		base(header("app", "missing", "") + "record item\n"),
		base(header("app", "item, item", "") + "record item\n"),
		base(header("app", "item", "") + "record private\nrecord item\n    private value\n"),
		base(header("app", "", "") + "record item\n    int value\n    str value\n"),
		base(header("app", "", "http as transport, http as transport")),
		base(header("app", "", "http as bytes")),
		base(header("app", "", "http as item") + "record item\n"),
		base(header("app", "", "missing")),
		base(header("app", "", "") + "record item\n" + function("bad", "item", "        int value\n        str value\n", "    ok item()\n")),
		base(header("app", "", "") + function("bad<item>", "item", "        item item\n", "    ok item\n")),
		base(header("app", "", "") + function("bad<item, item>", "item", "", "    ok x\n")),
		base(header("app", "", "") + function("transform", "int", "", "    ok 1\n") + "record bad\n    transform value\n"),
		base(header("app", "", "") + "fn int bad\n    on int value\n    emits []\n    asserts\n        x: 1 => => ok 1\n    ok 1\n"),
		{"src/a/one.can": header("app", "from_other_file", "") + "record item\n", "src/a/two.can": header("app", "", "") + "record from_other_file\n"},
		{"src/a/main.can": header("a", "item", "") + "record item\n", "src/b/main.can": header("b", "", "a") + "fn int bad\n    on a::item value\n    emits []\n    asserts\n        x: a::item() => => ok 1\n    ok 1\n"},
		{"src/a/main.can": header("a", "", "") + "record private\n", "src/b/main.can": header("b", "", "a") + "record bad\n    a::private value\n"},
	}
	for i, files := range cases {
		if _, err := buildFiles(t, files); err == nil {
			t.Errorf("invalid scope/signature case %d accepted", i)
		}
	}
}

func TestGenericParametersPrecedeReturnTypeResolution(t *testing.T) {
	w, err := buildFiles(t, map[string]string{"src/main.can": header("app", "identity", "") + function("identity<item>", "item", "        item value\n", "    ok value\n")})
	if err != nil {
		t.Fatal(err)
	}
	fn := w.Packages["app"].Scope.Symbols["identity"].Declaration.(*syntax.FunctionDecl)
	if got, err := w.Functions[fn].Lookup("item", TypeUse); err != nil || got.Kind != TypeParameter {
		t.Fatal("return-first parameter not registered")
	}
}

func TestSignatureBoundsRequireEligibleVisibleErrors(t *testing.T) {
	base := header("app", "run", "") + "error 1000000 failed()\nfn void run\n    emits [failed]\n    asserts\n        sample: => ok\n    ok\n"
	files := map[string]string{"src/main.can": base, "can.errors.json": `{"active":[{"id":1000000,"kind":"app::failed"}],"retired":[]}`}
	if _, err := buildFiles(t, files); err == nil || !strings.Contains(err.Error(), "private type") {
		t.Fatalf("exported bound hid private error: %v", err)
	}
	files["src/main.can"] = strings.Replace(base, "provides [run]", "provides [run, failed]", 1)
	if _, err := buildFiles(t, files); err != nil {
		t.Fatal(err)
	}
	files["src/main.can"] = strings.Replace(files["src/main.can"], "emits [failed]", "emits [int]", 1)
	if _, err := buildFiles(t, files); err == nil {
		t.Fatal("non-error declaration admitted in emits")
	}
}

func TestInternalVisibilityUsesCanonicalSourceLocations(t *testing.T) {
	secret := header("secret", "item", "") + "record item\n"
	caller := header("caller", "", "secret") + "record holder\n    secret::item value\n"
	if _, err := buildFiles(t, map[string]string{"src/store/internal/secret/main.can": secret, "src/store/client/main.can": caller}); err != nil {
		t.Fatal(err)
	}
	if _, err := buildFiles(t, map[string]string{"src/store/internal/secret/main.can": secret, "src/client/main.can": caller}); err == nil {
		t.Fatal("internal visibility bypassed")
	}
	root := writeFiles(t, map[string]string{"hidden/internal/secret/main.can": secret, "src/client/main.can": caller})
	if err := os.Symlink(filepath.Join(root, "hidden/internal/secret/main.can"), filepath.Join(root, "src/alias.can")); err != nil {
		t.Fatal(err)
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(graph); err == nil || !strings.Contains(err.Error(), "internal package") {
		t.Fatalf("source symlink bypassed internal policy: %v", err)
	}
}

func TestTransitiveDependencyDoesNotGrantDirectImport(t *testing.T) {
	files := map[string]string{
		"can.project.json":              `{"source_root":"src","dependencies":{"vendor":"vendor"},"error_registry":"can.errors.json"}`,
		"can.errors.json":               `{"active":[],"retired":[]}`,
		"src/main.can":                  header("app", "", "child_pkg") + "record holder\n    child_pkg::item value\n",
		"vendor/can.project.json":       `{"source_root":"src","dependencies":{"child":"child"},"error_registry":"can.errors.json"}`,
		"vendor/can.errors.json":        `{"active":[],"retired":[]}`,
		"vendor/src/main.can":           header("vendor_pkg", "item", "") + "record item\n",
		"vendor/child/can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"vendor/child/can.errors.json":  `{"active":[],"retired":[]}`,
		"vendor/child/src/main.can":     header("child_pkg", "item", "") + "record item\n",
	}
	entries := map[string]any{}
	for key, dir := range map[string]string{"vendor": "vendor", "child": "vendor/child"} {
		manifest := files[dir+"/can.project.json"]
		data := files[dir+"/src/main.can"]
		digest, err := project.SourceDigest([]project.SourceBytes{{Path: "main.can", Bytes: []byte(data)}})
		if err != nil {
			t.Fatal(err)
		}
		entries[key] = map[string]any{"path": dir, "manifest_sha256": project.Digest([]byte(manifest)), "source_sha256": digest, "error_registry": map[string]any{"active": []any{}, "retired": []any{}}}
	}
	lock, err := json.Marshal(map[string]any{"dependencies": entries})
	if err != nil {
		t.Fatal(err)
	}
	files["can.lock.json"] = string(lock)
	if _, err := buildFiles(t, files); err == nil || !strings.Contains(err.Error(), "undeclared direct dependency") {
		t.Fatalf("transitive dependency became visible: %v", err)
	}
	files["can.project.json"] = `{"source_root":"src","dependencies":{"vendor":"vendor","child":"vendor/child"},"error_registry":"can.errors.json"}`
	if _, err := buildFiles(t, files); err != nil {
		t.Fatalf("explicit repeated dependency failed: %v", err)
	}
}

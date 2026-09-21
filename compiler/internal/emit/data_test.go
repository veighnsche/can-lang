package emit

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"github.com/veighnsche/can-lang/distribution"
)

func fixtureTypes(t *testing.T) map[string]*types.Type {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"can.project.json":     `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":      `{"active":[],"retired":[]}`,
		"src/alpha/shared.can": "package alpha\n    provides [item]\n    uses []\nrecord item\n    int value\n    int[] shared\n",
		"src/beta/shared.can":  "package beta\n    provides [item]\n    uses []\nrecord item\n    int value\n    int[] shared\n",
		"src/app/main.can":     "package app\n    provides []\n    uses [alpha, beta, bytes, number]\nrecord box<item>\n    item value\n\nvariant both\n    alpha::item\n    beta::item\n",
	}
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
		t.Fatal(err)
	}
	world, err := resolve.Build(graph)
	if err != nil {
		t.Fatal(err)
	}
	builder := types.NewBuilder(world)
	if err = builder.SeedDeclarations(); err != nil {
		t.Fatal(err)
	}
	var file *resolve.File
	for _, f := range world.Files {
		if f.Package.Name == "app" {
			file = f
		}
	}
	out := map[string]*types.Type{}
	for _, name := range []string{"alpha::item", "beta::item", "box<int>", "box<str>", "int", "float", "bool", "str", "int[]", "str[]", "both", "both[]", "alpha::item[]", "bytes::buffer", "callable int () emits []", "callable bool () emits []", "callable float () emits []", "callable str () emits []", "callable int () emits [number::inexact]"} {
		src, _ := source.New("type.can", name)
		node, ds := syntax.ParseType(src)
		if len(ds) != 0 {
			t.Fatal(ds)
		}
		out[name], err = builder.Resolve(file, node, nil, false)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = builder.Finish(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDataEmissionRejectsInvalidContracts(t *testing.T) {
	ts := fixtureTypes(t)
	integer := DataExpression{"1n", ts["int"]}
	text := DataExpression{`"x"`, ts["str"]}
	if _, err := Record(ts["bytes::buffer"], nil); err == nil {
		t.Fatal("opaque constructor")
	}
	if _, err := Record(ts["box<int>"], []DataExpression{text}); err == nil {
		t.Fatal("wrong field type")
	}
	if _, err := Record(ts["box<int>"], nil); err == nil {
		t.Fatal("wrong constructor arity")
	}
	if _, err := Array(ts["int[]"], []DataExpression{text}); err == nil {
		t.Fatal("wrong array element")
	}
	if _, err := Array(nil, nil); err == nil {
		t.Fatal("untyped empty array")
	}
	if _, err := Update(DataExpression{"resource", ts["bytes::buffer"]}, []Replacement{{"value", integer}}); err == nil {
		t.Fatal("opaque update")
	}
	if _, err := Update(DataExpression{"box", ts["box<int>"]}, []Replacement{{"value", integer}, {"value", integer}}); err == nil {
		t.Fatal("duplicate update")
	}
}

func TestEmittedNativeNominalData(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to the absolute qualified Bun runtime for emitted execution")
	}
	if !filepath.IsAbs(bun) {
		t.Fatal("CAN_BUN must be absolute")
	}
	target := distribution.PinnedTarget()
	binary, err := os.ReadFile(bun)
	if err != nil || distribution.Hash(binary) != target.Runtime.SHA256 || runtime.GOOS != target.Runtime.Platform || runtime.GOARCH != target.Runtime.Architecture {
		t.Fatal("CAN_BUN does not match the exact qualified binary and target")
	}
	version, err := exec.Command(bun, "--version").Output()
	if err != nil || strings.TrimSpace(string(version)) != "1.4.2" {
		t.Fatalf("wrong qualified runtime: %s %v", version, err)
	}
	ts := fixtureTypes(t)
	arrayExpr, err := Array(ts["int[]"], []DataExpression{{"1n", ts["int"]}})
	if err != nil {
		t.Fatal(err)
	}
	makeRecord := func(name, value string) string {
		r, err := Record(ts[name], []DataExpression{{value, ts["int"]}, {"shared", ts["int[]"]}})
		if err != nil {
			t.Fatal(err)
		}
		return r.Code
	}
	initial := makeRecord("alpha::item", "1n")
	same := makeRecord("alpha::item", "1n")
	other := makeRecord("beta::item", "1n")
	changed, err := Update(DataExpression{"receiver()", ts["alpha::item"]}, []Replacement{{"value", DataExpression{"replacement()", ts["int"]}}, {"shared", DataExpression{"replacementArray()", ts["int[]"]}}})
	if err != nil {
		t.Fatal(err)
	}
	runtimePath, err := filepath.Abs("../../../runtime/data.ts")
	if err != nil {
		t.Fatal(err)
	}
	program := DataImports(runtimePath) + `
const assert = (value: boolean) => { if (!value) throw new Error("data invariant failed"); };
const shared = ` + arrayExpr.Code + `;
const original = ` + initial + `;
const same = ` + same + `;
const other = ` + other + `;
assert(Bun.deepEquals(original, same, true));
assert(!Bun.deepEquals(original, other, true));
const events: string[] = [];
function receiver() { events.push("receiver"); return original; }
function replacement() { events.push("value"); return original.value as bigint + 1n; }
function replacementArray() { events.push("shared"); assert(original.value === 1n); return shared; }
const changed = ` + changed.Code + `;
assert(events.join(",") === "receiver,value,shared");
assert(original.value === 1n && changed.value === 2n);
assert(changed.shared === original.shared);
assert(Object.isFrozen(changed) && Object.isFrozen(original) && Object.isFrozen(shared));
assert(Bun.deepEquals(changed, ` + makeRecord("alpha::item", "2n") + `, true));
console.log("emitted nominal identity, immutable update and evaluation order passed");
`
	path := filepath.Join(t.TempDir(), "data.ts")
	if err = os.WriteFile(path, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(bun, "--no-env-file", "--no-macros", "--no-install", path)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s\n%s", err, output, program)
	}
	t.Log(strings.TrimSpace(string(output)))
}

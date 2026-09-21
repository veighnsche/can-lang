package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const genericIdentity = `fn item identity<item>
    emits []
    given
        item value
    asserts
        integer: 3 => ok 3
        text: "x" => ok "x"
    ok value
`

func TestExplicitGenericFunctionInstances(t *testing.T) {
	text := programHeader + genericIdentity + programMain + `    int first = call identity<int>(3)
    str second = call identity<str>("x")
    int repeated = call identity<int>(4)
    match first + repeated is 7 and second is "x"
        true => ok
        false => do
            int invalid = 1 / 0
            ok
`
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, fn := range p.Functions {
		if fn.Instance == "" {
			continue
		}
		count++
		if fn.Region == nil || fn.Region.ID != fn.Identity() || len(fn.TypeArguments) != 1 || !types.Equal(fn.Region.Result, fn.TypeArguments[0]) {
			t.Fatal("missing concrete instance body")
		}
		if fn.Region.Inputs[0].Identity != fn.Identity()+"/input/value" {
			t.Fatal("instance input identity shared")
		}
	}
	if count != 2 {
		t.Fatalf("want two cached instances, got %d", count)
	}
}

func TestGenericWholeBodyAndRecursion(t *testing.T) {
	for name, body := range map[string]string{
		"same instance recursion": `    match count is 0
        true => ok value
        false => relay call repeat<item>(value, count - 1)
`,
		"invalid unused branch": `    match count is 0
        true => ok value
        false => ok "wrong"
`,
		"expanding recursion": `    match call repeat<item[]>([value], count)
        ok item[] ignored => ok value
`,
		"inferred expanding recursion": `    match call repeat([value], count)
        ok item[] ignored => ok value
`,
	} {
		t.Run(name, func(t *testing.T) {
			declaration := `fn item repeat<item>
    emits []
    given
        item value
        int count
    asserts
        sample: 3, 0 => ok 3
` + body
			text := programHeader + declaration + programMain + "    int result = call repeat<int>(3, 0)\n    ok\n"
			p, err := programFixture(t, map[string]string{"src/main.can": text})
			if name == "same instance recursion" {
				if err != nil {
					t.Fatal(err)
				}
				if len(p.Functions) != 2 {
					t.Fatal("recursive instance duplicated")
				}
			} else if err == nil {
				t.Fatal("invalid generic body admitted")
			} else if strings.Contains(name, "expanding recursion") && !strings.Contains(err.Error(), "expanding polymorphic recursion") {
				t.Fatalf("wrong expanding recursion diagnostic: %v", err)
			}
		})
	}
}

func TestExplicitGenericCallableReference(t *testing.T) {
	text := programHeader + genericIdentity + programMain + `    callable int (int) emits [] action = callable identity<int>
    int result = call action(7)
    ok
`
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Functions) != 3 || p.Functions[1].Region == nil {
		t.Fatal("reference did not schedule concrete body")
	}
}

func TestGenericCallInference(t *testing.T) {
	declarations := genericIdentity + `fn item first<item>
    emits []
    given
        item[] items
        item fallback
    asserts
        sample: [], 3 => ok 3
    match items.length is 0
        true => ok fallback
        false => ok items[0]

fn item[] empty<item>
    emits []
    asserts
        sample: => ok call integer_empty()
    ok []

fn int[] integer_empty
    emits []
    asserts
        sample: => ok []
    ok []
`
	text := programHeader + declarations + programMain + `    int value = call identity(3)
    int selected = call first([], value)
    str[] strings = call empty()
    int repeated = call identity<int>(selected)
    ok
`
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Functions) != 7 {
		t.Fatalf("expected five shared generic instances plus two ordinary functions, got %d", len(p.Functions))
	}
	for name, bad := range map[string]string{
		"conflicting expected result": strings.Replace(text, "int value = call identity(3)", "str value = call identity(3)", 1),
		"conflicting fixed arguments": strings.Replace(text, "first([], value)", "first([\"text\"], value)", 1),
		"arity":                       strings.Replace(text, "first([], value)", "first([])", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
				t.Fatal("invalid inference accepted")
			}
		})
	}
}

func TestGenericAssertionsOwnInstances(t *testing.T) {
	text := programHeader + genericIdentity + programMain + "    ok\n"
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Functions) != 3 || len(p.Assertions) != 3 {
		t.Fatalf("assertion-only instances missing: %d functions, %d assertions", len(p.Functions), len(p.Assertions))
	}
	for _, fn := range p.Functions {
		if fn.Region == nil {
			t.Fatal("assertion instance body unchecked")
		}
	}
	bad := strings.Replace(text, "    ok value\n", "    ok value + 1\n", 1)
	if _, err = programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
		t.Fatal("invalid assertion-only string instance accepted")
	}
	bad = strings.Replace(text, `text: "x" => ok "x"`, `text: "x" => ok 3`, 1)
	if _, err = programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
		t.Fatal("conflicting generic assertion types accepted")
	}
}

func TestGenericReferenceInference(t *testing.T) {
	declarations := genericIdentity + `fn item captured<item>
    emits []
    given
        near item value
    asserts
        sample: 3 => ok 3
    ok value
`
	text := programHeader + declarations + programMain + `    int value = 7
    callable int (int) emits [] identity_action = callable identity
    callable int () emits [] captured_action = callable captured
    int result = call identity_action(call captured_action())
    ok
`
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Functions) != 4 {
		t.Fatalf("reference duplicated cached instances: %d", len(p.Functions))
	}
	for name, bad := range map[string]string{
		"capture conflicts with expected": strings.Replace(text, "callable int () emits [] captured_action", "callable str () emits [] captured_action", 1),
		"nonidentical callable inputs":    strings.Replace(text, "callable int (int) emits [] identity_action", "callable int (str) emits [] identity_action", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
				t.Fatal("invalid generic reference accepted")
			}
		})
	}
}

func TestGenericConstructorInference(t *testing.T) {
	declarations := `record box<item>
    item value
record collection<item>
    item[] values
    item fallback
record tree<item>
    item value
    tree<item>[] children
variant selection
    box<int>
    box<str>
fn int unbox
    emits []
    given
        box<int> value
    asserts
        sample: box(3) => ok 3
    ok value.value
`
	text := programHeader + declarations + programMain + `    int local = 3
    box<int> wrapped = box(local)
    collection<int> empty = collection([], local)
    tree<int> recursive = tree(local, [])
    selection selected = box("text")
    int result = call unbox(box(local))
    ok
`
	if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"conflicting expected": strings.Replace(text, "box<int> wrapped = box(local)", "box<str> wrapped = box(local)", 1),
		"conflicting fields":   strings.Replace(text, "collection([], local)", `collection(["text"], local)`, 1),
		"wrong arity":          strings.Replace(text, "box(local)", "box()", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
				t.Fatal("invalid constructor inference accepted")
			}
		})
	}
}

func TestGenericMethods(t *testing.T) {
	declarations := `record box<item>
    item value
fn item read<item>
    on box<item> self
    emits []
    asserts
        integer: box(3) => => ok 3
        text: box("x") => => ok "x"
    ok self.value
fn item replace<item>
    on box<item> self
    emits []
    given
        near item replacement
    asserts
        integer: box(3) => 7 => ok 7
    ok replacement
`
	text := programHeader + declarations + programMain + `    box<int> value = box(3)
    int replacement = 7
    int direct = call value.read()
    int explicit = call value.read<int>()
    callable int () emits [] read_action = callable value.read
    callable int () emits [] replace_action = callable value.replace
    int result = call replace_action()
    ok
`
	p, err := programFixture(t, map[string]string{"src/main.can": text})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Functions) != 4 {
		t.Fatalf("method instances not shared: %d", len(p.Functions))
	}
	bad := strings.Replace(text, "int replacement = 7", `str replacement = "wrong"`, 1)
	if _, err = programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
		t.Fatal("mismatched receiver and near type accepted")
	}
}

func TestGenericCrossModuleIdentity(t *testing.T) {
	files := map[string]string{}
	for _, name := range []string{"definitions.can", "helper.can", "main.can"} {
		data, err := os.ReadFile(filepath.Join("../../testdata/current/generics", name))
		if err != nil {
			t.Fatal(err)
		}
		files["src/"+name] = string(data)
	}
	p, err := programFixture(t, files)
	if err != nil {
		t.Fatal(err)
	}
	instances := map[string]bool{}
	shared := 0
	for _, fn := range p.Functions {
		if fn.Instance == "" {
			continue
		}
		if instances[fn.Instance] {
			t.Fatal("duplicate concrete instance")
		}
		instances[fn.Instance] = true
		if fn.Symbol.Name == "identity" && fn.TypeArguments[0].Kind() == types.Record {
			shared++
			sites := strings.Join(fn.Requests, ";")
			if !strings.Contains(sites, "helper.can") || !strings.Contains(sites, "main.can") {
				t.Fatalf("lost shared applications: %s", sites)
			}
			if !types.Equal(fn.Region.Result, fn.TypeArguments[0]) {
				t.Fatal("nominal result identity changed")
			}
		}
	}
	if shared != 1 {
		t.Fatalf("want one cross-module box identity instance, got %d", shared)
	}
	files["src/definitions.can"] = strings.Replace(files["src/definitions.can"], "    ok value\n", "    ok false\n", 1)
	if _, err = programFixture(t, files); err == nil || !strings.Contains(err.Error(), "generic source") || !strings.Contains(err.Error(), "requested at") {
		t.Fatalf("missing generic application diagnostic: %v", err)
	}
}

func TestFiniteGenericTransitionsIgnoreCacheOrder(t *testing.T) {
	declaration := `fn int fixed<item>
    emits []
    given
        item value
        int count
    asserts
        sample: 1, 0 => ok 0
    match count is 0
        true => ok 0
        false => relay call fixed<int[]>([1], count - 1)
`
	for _, inferred := range []bool{false, true} {
		for _, preloaded := range []bool{false, true} {
			source := declaration
			if inferred {
				source = strings.Replace(source, "fixed<int[]>([1]", "fixed([1]", 1)
			}
			if preloaded {
				source = strings.Replace(source, "        sample: 1, 0 => ok 0", "        sample: 1, 0 => ok 0\n        array: [1], 0 => ok 0", 1)
			}
			p, err := programFixture(t, map[string]string{"src/main.can": programHeader + source + programMain + "    int result = call fixed<int>(1, 2)\n    ok\n"})
			if err != nil {
				t.Fatalf("inferred=%v preloaded=%v: %v", inferred, preloaded, err)
			}
			if len(p.Functions) != 3 {
				t.Fatalf("finite transition did not stabilize: %d", len(p.Functions))
			}
		}
	}
}
func TestGenericLiteralSpreadPreservesExpectedElements(t *testing.T) {
	declaration := `fn item pick<item>
    emits []
    given
        item first
        item second
    asserts
        sample: 1, 2 => ok 1
    ok first
`
	for _, spread := range []string{"...[[], []]", "...([[], []])"} {
		text := programHeader + declaration + programMain + "    int[] result = call pick(" + spread + ")\n    ok\n"
		if _, err := programFixture(t, map[string]string{"src/main.can": text}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFiniteGenericFieldChainIgnoresAssertionOrder(t *testing.T) {
	data, err := os.ReadFile("../../testdata/current/generics/finite-field-chain.can")
	if err != nil {
		t.Fatal(err)
	}
	row := "        sample: seed([]), 0 => ok 0\n"
	terminal := "        terminal: done<step<seed>>([]), 0 => ok 0\n"
	for name, rows := range map[string]string{"unloaded": row, "first": terminal + row, "last": row + terminal} {
		t.Run(name, func(t *testing.T) {
			p, err := programFixture(t, map[string]string{"src/main.can": strings.Replace(string(data), row, rows, 1)})
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Functions) != 4 {
				t.Fatalf("want three walk instances and main, got %d", len(p.Functions))
			}
		})
	}
}

package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const exportedAppHeader = "package app\n    provides []\n    uses [lib]\n"
const exportedAppMain = "fn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"

func exportedInstance(t *testing.T, program *Program, name string) *ProgramFunction {
	t.Helper()
	for _, fn := range program.Functions {
		if strings.HasSuffix(fn.Symbol.ID, "::"+name) && fn.Instance != "" {
			return fn
		}
		if strings.Contains(fn.Instance, "/symbolic") {
			t.Fatalf("symbolic declaration %s leaked into emitted functions", fn.Instance)
		}
	}
	t.Fatalf("no checked instance for %s", name)
	return nil
}

// Six-fixture probe from the generic-recursion experiment (UP02, enabled by
// UP14): the five finite cases must check with zero declaration-only
// Parameter nodes in the runtime model, and only the expanding cycle keeps
// its hard rejection, now diagnosed as a cycle rather than a broad opaque
// restriction.
func TestExportedGenericSymbolicProofIsolation(t *testing.T) {
	const main = "fn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"
	const identity = "fn item identity<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 3\n    ok value\n"
	cases := map[string]struct {
		source string
		accept bool
	}{
		"public-identity-control": {
			source: "package app\n    provides [identity]\n    uses []\n" + identity + main,
			accept: true,
		},
		"stationary-self": {
			source: "package app\n    provides [first]\n    uses []\nfn item first<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call first<item>(value, true)\n        true => ok value\n" + main,
			accept: true,
		},
		"acyclic-nested-private": {
			source: "package app\n    provides [box]\n    uses []\nrecord box<item>\n    item value\n" + identity + "fn box<item> nested<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok box(3)\n    ok call identity<box<item>>(box(value))\n" + main,
			accept: true,
		},
		"stationary-mutual": {
			source: "package app\n    provides [first, second]\n    uses []\nfn item first<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call second<item>(value, true)\n        true => ok value\nfn item second<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call first<item>(value, true)\n        true => ok value\n" + main,
			accept: true,
		},
		"expanding-mutual": {
			source: "package app\n    provides [a, b]\n    uses []\nfn void a<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok\n    match stop\n        false => relay call b<item[]>([value], true)\n        true => ok\nfn void b<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok\n    match stop\n        false => relay call a<item>(value, true)\n        true => ok\n" + main,
		},
		"acyclic-nested-public": {
			source: "package app\n    provides [box, identity, nested]\n    uses []\nrecord box<item>\n    item value\n" + identity + "fn box<item> nested<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok box(3)\n    ok call identity<box<item>>(box(value))\n" + main,
			accept: true,
		},
	}
	for name, kase := range cases {
		t.Run(name, func(t *testing.T) {
			program, err := programFixture(t, map[string]string{"src/app.can": kase.source})
			if !kase.accept {
				if err == nil {
					t.Fatalf("expanding symbolic cycle admitted: %s", name)
				}
				if !strings.Contains(err.Error(), "exported generic function") || !strings.Contains(err.Error(), "expanding symbolic cycle") {
					t.Fatalf("missing cycle rejection for %s: %v", name, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, typ := range program.Model.Types() {
				if typ.Kind() == types.Parameter {
					t.Fatalf("symbolic proof graph leaked into the runtime model: %s", typ.Declaration())
				}
			}
			for _, fn := range program.Functions {
				if strings.Contains(fn.Instance, "/symbolic") {
					t.Fatalf("symbolic declaration %s leaked into emitted functions", fn.Instance)
				}
			}
		})
	}
}

func TestExportedGenericPassThrough(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [identity]\n    uses []\nfn value identity<value>\n    emits []\n    given\n        value input\n    asserts\n        one: 1 => ok 1\n    ok input\n",
		"src/app/main.can": exportedAppHeader + "fn int use_identity\n    emits []\n    asserts\n        sample: => ok 3\n    ok call lib::identity(3)\n" + exportedAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	instance := exportedInstance(t, program, "identity")
	if len(instance.TypeArguments) != 1 || instance.TypeArguments[0].Declaration() != "int" {
		t.Fatalf("identity instance has wrong arguments: %+v", instance.TypeArguments)
	}
}

func TestExportedGenericRejectsPlus(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [doubled]\n    uses []\nfn item doubled<item>\n    emits []\n    given\n        item value\n    asserts\n        triple: 3 => ok 6\n    ok value + value\n",
		"src/app/main.can": exportedAppHeader + "fn int use_doubled\n    emits []\n    asserts\n        sample: => ok 6\n    ok call lib::doubled(3)\n" + exportedAppMain,
	})
	if err == nil {
		t.Fatal("exported generic with unsupported + admitted")
	}
	if !strings.Contains(err.Error(), "exported generic function") || !strings.Contains(err.Error(), "operator + is not defined") {
		t.Fatalf("missing declaration diagnosis: %v", err)
	}
	located, ok := source.AsLocated(err)
	if !ok || !strings.Contains(located.File, "lib.can") {
		t.Fatalf("failure is not located at the exporting declaration: %v", err)
	}
	related := false
	for _, span := range located.Related {
		if span.Note == "exported generic declared here" && strings.Contains(span.File, "lib.can") {
			related = true
		}
	}
	if !related {
		t.Fatalf("failure does not relate the exported declaration: %v", err)
	}
}

func TestExportedGenericCallableRestoresBehavior(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [doubled]\n    uses []\nfn item doubled<item>\n    emits []\n    given\n        item value\n        callable item (item, item) emits [] plus\n    asserts\n        triple: 3, callable int_plus => ok 6\n    ok call plus(value, value)\nfn int int_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 3, 3 => ok 6\n    ok first + second\n",
		"src/app/main.can": exportedAppHeader + "fn int app_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 1, 2 => ok 3\n    ok first + second\nfn int use_doubled\n    emits []\n    asserts\n        sample: => ok 8\n    ok call lib::doubled(4, callable app_plus)\n" + exportedAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	instance := exportedInstance(t, program, "doubled")
	contract := instance.Region.Inputs
	if len(contract) != 2 || contract[1].Type.Kind() != types.Callable {
		t.Fatalf("doubled instance lost its callable input: %+v", contract)
	}
	seen := map[string]bool{}
	for _, assertion := range program.Assertions {
		seen[assertion.Root.Name] = true
	}
	for _, name := range []string{"triple", "sample", "empty"} {
		if !seen[name] {
			t.Fatalf("assertion row %q missing after callable repair", name)
		}
	}
}

func TestExportedGenericDiagnosesStaleCaller(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [doubled]\n    uses []\nfn item doubled<item>\n    emits []\n    given\n        item value\n        callable item (item, item) emits [] plus\n    asserts\n        triple: 3, callable int_plus => ok 6\n    ok call plus(value, value)\nfn int int_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 3, 3 => ok 6\n    ok first + second\n",
		"src/app/main.can": exportedAppHeader + "fn int use_doubled\n    emits []\n    asserts\n        sample: => ok 6\n    ok call lib::doubled(3)\n" + exportedAppMain,
	})
	if err == nil {
		t.Fatal("caller missing the written callable admitted")
	}
	if !strings.Contains(err.Error(), "arity") {
		t.Fatalf("stale caller misdiagnosed: %v", err)
	}
}

func TestPrivateGenericTemplatePreserved(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/main.can": "package app\n    provides []\n    uses []\nfn item doubled<item>\n    emits []\n    given\n        item value\n    asserts\n        triple: 3 => ok 6\n    ok value + value\nfn int use_doubled\n    emits []\n    asserts\n        sample: => ok 6\n    ok call doubled(3)\n" + programMain + "    ok\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	exportedInstance(t, program, "doubled")
}

func TestExportedGenericDictionaryRecord(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [combined, ops]\n    uses []\nrecord ops<item>\n    callable item (item, item) emits [] plus\nfn item combined<item>\n    emits []\n    given\n        item value\n        ops<item> dictionary\n    asserts\n        sample: 3, ops(callable int_plus) => ok 6\n    ok call dictionary.plus(value, value)\nfn int int_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 3, 3 => ok 6\n    ok first + second\n",
		"src/app/main.can": exportedAppHeader + "fn int app_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 1, 2 => ok 3\n    ok first + second\nfn int use_combined\n    emits []\n    asserts\n        sample: => ok 8\n    ok call lib::combined(4, lib::ops(callable app_plus))\n" + exportedAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	exportedInstance(t, program, "combined")
}

func TestExportedGenericRejectsRepresentationOperations(t *testing.T) {
	shell := func(body string) string {
		return "package lib\n    provides [doubled]\n    uses []\nfn item doubled<item>\n    emits []\n    given\n        item value\n    asserts\n        triple: 3 => ok 3\n    " + body + "\n"
	}
	cases := map[string]string{
		"ordering":   "ok value < value",
		"equality":   "ok value is value",
		"negation":   "ok -value",
		"not":        "ok not value",
		"field":      "ok value.field",
		"index":      "ok value[0]",
		"method":     "ok call value.helper()",
		"match":      "match value\n        1 => ok value\n        _ => ok value",
		"other call": "ok call other(value)",
	}
	other := "fn item other<item>\n    emits []\n    given\n        item input\n    asserts\n        sample: 1 => ok 1\n    ok input\n"
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			text := shell(body)
			if name == "other call" {
				text += other
			}
			_, err := programFixture(t, map[string]string{
				"src/lib/lib.can":  text,
				"src/app/main.can": exportedAppHeader + exportedAppMain,
			})
			if err == nil {
				t.Fatalf("exported operation admitted: %s", name)
			}
			if !strings.Contains(err.Error(), "exported generic function") {
				t.Fatalf("missing declaration diagnosis for %s: %v", name, err)
			}
		})
	}
}

func TestExportedGenericStructuralComposites(t *testing.T) {
	// Known composite structure stays usable: matching, projecting,
	// updating and constructing composites never inspects the opaque
	// payload, and array iteration routes elements through the explicit
	// callable input.
	cases := map[string]struct {
		uses     string
		provides string
		source   string
		instance string
	}{
		"option match": {
			uses:     "option",
			provides: "unwrap",
			source:   "fn option::value<int> wrap\n    emits []\n    given\n        int value\n    asserts\n        sample: 3 => ok option::some(3)\n    ok option::some(value)\nfn item unwrap<item>\n    emits []\n    given\n        option::value<item> maybe\n        item fallback\n    asserts\n        sample: call wrap(3), 0 => ok 3\n    match maybe\n        option::none => ok fallback\n        option::some => ok maybe.value\n",
			instance: "unwrap",
		},
		"box match": {
			provides: "box, get",
			source:   "record box<item>\n    item value\nfn item get<item>\n    emits []\n    given\n        box<item> cell\n    asserts\n        sample: box(3) => ok 3\n    match cell\n        bind held => ok held.value\n",
			instance: "get",
		},
		"copy update": {
			provides: "box, replace",
			source:   "record box<item>\n    item value\nfn box<item> replace<item>\n    emits []\n    given\n        box<item> cell\n        item next\n    asserts\n        sample: box(1), 2 => ok box(2)\n    ok cell with value = next\n",
			instance: "replace",
		},
		"construct": {
			provides: "box, wrap",
			source:   "record box<item>\n    item value\n    box<item>[] children\nfn box<item> wrap<item>\n    emits []\n    given\n        item value\n    asserts\n        sample: 1 => ok box(1, [])\n    ok box(value, [])\n",
			instance: "wrap",
		},
		"map callable": {
			provides: "mapped",
			source:   "fn item[] mapped<item>\n    emits []\n    given\n        item[] items\n        callable item (item) emits [] each\n    asserts\n        sample: [1], callable identity_int => ok [1]\n    ok call items.map(each)\nfn int identity_int\n    emits []\n    given\n        int value\n    asserts\n        sample: 1 => ok 1\n    ok value\n",
			instance: "mapped",
		},
	}
	for name, kase := range cases {
		t.Run(name, func(t *testing.T) {
			program, err := programFixture(t, map[string]string{
				"src/lib/lib.can":  "package lib\n    provides [" + kase.provides + "]\n    uses [" + kase.uses + "]\n" + kase.source,
				"src/app/main.can": exportedAppHeader + exportedAppMain,
			})
			if err != nil {
				t.Fatal(err)
			}
			exportedInstance(t, program, kase.instance)
		})
	}
}

func TestExportedGenericRejectsCodec(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [encoded]\n    uses [codec, bytes]\nfn bytes::buffer encoded<item>\n    emits [codec::invalid_data]\n    given\n        item value\n    asserts\n        sample: 1 => ok\n    ok call codec::encode_json<item>(value)\n",
		"src/app/main.can": exportedAppHeader + exportedAppMain,
	})
	if err == nil {
		t.Fatal("codec over an opaque parameter admitted")
	}
	if !strings.Contains(err.Error(), "exported generic function") || !strings.Contains(err.Error(), "opaque type parameter") {
		t.Fatalf("missing declaration diagnosis: %v", err)
	}
}

func TestExportedGenericRejectsBareParameterBound(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [work]\n    uses []\nfn item work<item>\n    emits [item]\n    given\n        item value\n    asserts\n        sample: 1 => ok 1\n    ok value\n",
		"src/app/main.can": exportedAppHeader + exportedAppMain,
	})
	// A bare variable is not an eligible error bound (resolve rejects it
	// before checking), and T07 adds no error-set parameter kind, so an
	// exported generic cannot smuggle one through `emits`.
	if err == nil {
		t.Fatal("exported generic with bare parameter bound admitted")
	}
	if !strings.Contains(err.Error(), "item") {
		t.Fatalf("bare parameter bound misdiagnosed: %v", err)
	}
}

func TestExportedGenericSelfRecursion(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [first]\n    uses []\nfn item first<item>\n    emits []\n    given\n        item[] items\n        item fallback\n    asserts\n        sample: [1, 2], 0 => ok 1\n        empty: [], 0 => ok 0\n    item[] rest = call items.slice(1, items.length)\n    match items.length is 0\n        false => ok call first(rest, fallback)\n        true => ok fallback\n",
		"src/app/main.can": exportedAppHeader + "fn int use_first\n    emits []\n    asserts\n        sample: => ok 7\n    ok call lib::first([7, 8], 0)\n" + exportedAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	exportedInstance(t, program, "first")
}

func TestExportedGenericMethod(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/lib/lib.can":  "package lib\n    provides [box, read]\n    uses []\nrecord box<item>\n    item value\nfn item read<item>\n    on box<item> self\n    emits []\n    asserts\n        integer: box(3) => => ok 3\n    ok self.value\n",
		"src/app/main.can": exportedAppHeader + "fn int use_read\n    emits []\n    asserts\n        sample: => ok 9\n    lib::box<int> cell = lib::box(9)\n    ok call cell.read()\n" + exportedAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	exportedInstance(t, program, "read")
	bad := strings.Replace("package lib\n    provides [box, read]\n    uses []\nrecord box<item>\n    item value\nfn item read<item>\n    on box<item> self\n    emits []\n    asserts\n        integer: box(3) => => ok 6\n    ok self.value + self.value\n", "box(3)", "box(3)", 1)
	_, err = programFixture(t, map[string]string{
		"src/lib/lib.can":  bad,
		"src/app/main.can": exportedAppHeader + exportedAppMain,
	})
	if err == nil {
		t.Fatal("exported generic method with unsupported + admitted")
	}
	if !strings.Contains(err.Error(), "exported generic function") || !strings.Contains(err.Error(), "operator + is not defined") {
		t.Fatalf("missing declaration diagnosis: %v", err)
	}
}

func TestExportedGenericDependencyEdit(t *testing.T) {
	broken := "package helpers\n    provides [doubled]\n    uses []\nfn item doubled<item>\n    emits []\n    given\n        item value\n    asserts\n        triple: 3 => ok 6\n    ok value + value\n"
	main := "package app\n    provides []\n    uses [vendor::helpers]\nfn int use_doubled\n    emits []\n    asserts\n        sample: => ok 6\n    ok call helpers::doubled(3)\n" + programMain + "    ok\n"
	_, err := programFixtureWithVendor(t, map[string]string{"src/main.can": main}, map[string]string{"src/lib/lib.can": broken})
	if err == nil {
		t.Fatal("dependency body edit adding + admitted")
	}
	if !strings.Contains(err.Error(), "exported generic function") {
		t.Fatalf("missing declaration diagnosis: %v", err)
	}
	located, ok := source.AsLocated(err)
	if !ok || !strings.Contains(located.File, "lib.can") {
		t.Fatalf("failure is not located at the dependency declaration: %v", err)
	}
	fixed := "package helpers\n    provides [doubled]\n    uses []\nfn item doubled<item>\n    emits []\n    given\n        item value\n        callable item (item, item) emits [] plus\n    asserts\n        triple: 3, callable int_plus => ok 6\n    ok call plus(value, value)\nfn int int_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 3, 3 => ok 6\n    ok first + second\n"
	repaired := "package app\n    provides []\n    uses [vendor::helpers]\nfn int app_plus\n    emits []\n    given\n        int first\n        int second\n    asserts\n        sample: 1, 2 => ok 3\n    ok first + second\nfn int use_doubled\n    emits []\n    asserts\n        sample: => ok 8\n    ok call helpers::doubled(4, callable app_plus)\n" + programMain + "    ok\n"
	program, err := programFixtureWithVendor(t, map[string]string{"src/main.can": repaired}, map[string]string{"src/lib/lib.can": fixed})
	if err != nil {
		t.Fatal(err)
	}
	exportedInstance(t, program, "doubled")
}

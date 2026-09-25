package check

import (
	"reflect"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const symbolicCorePackage = `package core
    provides [identity]
    uses []
fn item identity<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok value
`

const symbolicHelpersPackage = `package helpers
    provides [box, pass, wrap, nested]
    uses [core]
record box<item>
    item value
fn item pass<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok call core::identity<item>(value)
fn box<item> wrap<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok box(3)
    ok box(call pass<item>(value))
fn box<item> nested<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok box(3)
    ok call core::identity<box<item>>(box(value))
`

const symbolicAppMain = `fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

// symbolicInstances indexes checked concrete instances by generic source name.
func symbolicInstances(t *testing.T, program *Program) map[string][]*ProgramFunction {
	t.Helper()
	byName := map[string][]*ProgramFunction{}
	for _, fn := range program.Functions {
		if strings.Contains(fn.Instance, "/symbolic") {
			t.Fatalf("symbolic proof %s leaked into emitted functions", fn.Instance)
		}
		for _, argument := range fn.TypeArguments {
			if name := types.OpaqueParameterName(argument); name != "" {
				t.Fatalf("instance %s carries opaque parameter %s", fn.Instance, name)
			}
		}
		byName[fn.Symbol.Name] = append(byName[fn.Symbol.Name], fn)
	}
	for _, typ := range program.Model.Types() {
		if typ.Kind() == types.Parameter {
			t.Fatalf("symbolic proof graph leaked into the runtime model: %s", typ.Declaration())
		}
	}
	return byName
}

// symbolicCallTargets collects every invocation identity a checked region calls.
func symbolicCallTargets(region *ir.Region) []string {
	var targets []string
	stepType := reflect.TypeOf(ir.InvocationStep{})
	var walk func(value reflect.Value)
	walk = func(value reflect.Value) {
		if !value.IsValid() || !value.CanInterface() {
			return
		}
		if value.Type() == stepType {
			targets = append(targets, value.FieldByName("Identity").String())
		}
		if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			node := value.Interface()
			if _, ok := node.(*types.Type); ok {
				return
			}
			walk(value.Elem())
			return
		}
		switch value.Kind() {
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				walk(value.Field(i))
			}
		case reflect.Slice:
			for i := 0; i < value.Len(); i++ {
				walk(value.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(region))
	return targets
}

func TestSymbolicCrossPackageChain(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/core/core.can":       symbolicCorePackage,
		"src/helpers/helpers.can": symbolicHelpersPackage,
		"src/app/main.can":        "package app\n    provides [top]\n    uses [helpers]\nfn helpers::box<item> top<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok helpers::box(3)\n    ok call helpers::wrap<item>(value)\n" + symbolicAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	byName := symbolicInstances(t, program)
	for _, name := range []string{"identity", "pass", "wrap", "nested", "top"} {
		if len(byName[name]) == 0 {
			t.Fatalf("no checked instance for %s", name)
		}
	}
	// A reached nested<int> directly calls concrete identity<box<int>>: the
	// box-carrying instance exists with a closed nominal argument.
	var nested, boxed *ProgramFunction
	for _, fn := range byName["nested"] {
		if len(fn.TypeArguments) == 1 && fn.TypeArguments[0].Declaration() == "int" {
			nested = fn
		}
	}
	for _, fn := range byName["identity"] {
		if len(fn.TypeArguments) == 1 && strings.HasSuffix(fn.TypeArguments[0].Declaration(), "::box") {
			boxed = fn
		}
	}
	if nested == nil {
		t.Fatal("no checked nested<int> instance")
	}
	if boxed == nil {
		t.Fatal("no checked identity<box<int>> instance")
	}
	if len(boxed.TypeArguments[0].Arguments()) != 1 || boxed.TypeArguments[0].Arguments()[0].Declaration() != "int" {
		t.Fatalf("identity box argument is not box<int>: %s", boxed.TypeArguments[0].Declaration())
	}
	found := false
	for _, target := range symbolicCallTargets(nested.Region) {
		if target == boxed.Instance {
			found = true
		}
		if strings.Contains(target, "/symbolic") {
			t.Fatalf("nested<int> calls symbolic proof %s", target)
		}
	}
	if !found {
		t.Fatalf("nested<int> never calls %s", boxed.Instance)
	}
}

func TestSymbolicMutualPermutation(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [first, second]\n    uses []\nfn a first<a, b>\n    emits []\n    given\n        a x\n        b y\n        bool stop\n    asserts\n        base: 1, \"s\", true => ok 1\n    match stop\n        false => ok call second<b, a>(y, x, true)\n        true => ok x\nfn b second<a, b>\n    emits []\n    given\n        a x\n        b y\n        bool stop\n    asserts\n        base: 1, \"s\", true => ok \"s\"\n    match stop\n        false => ok call first<b, a>(y, x, true)\n        true => ok y\n" + symbolicAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	byName := symbolicInstances(t, program)
	if len(byName["first"]) == 0 || len(byName["second"]) == 0 {
		t.Fatal("permuted mutual instances missing")
	}
}

func TestSymbolicDuplicationDrop(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [dup, pair]\n    uses []\nfn a dup<a>\n    emits []\n    given\n        a value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call pair<a, a>(value, value, true)\n        true => ok value\nfn a pair<a, b>\n    emits []\n    given\n        a x\n        b y\n        bool stop\n    asserts\n        base: 3, 4, true => ok 3\n    match stop\n        false => ok call dup<a>(x, true)\n        true => ok x\n" + symbolicAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	byName := symbolicInstances(t, program)
	if len(byName["dup"]) == 0 || len(byName["pair"]) == 0 {
		t.Fatal("duplication/drop mutual instances missing")
	}
}

func TestSymbolicClosedComponent(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [a, b]\n    uses []\nfn void a<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok\n    match stop\n        false => relay call b<int>(3, true)\n        true => ok\nfn void b<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok\n    match stop\n        false => relay call a<int>(3, true)\n        true => ok\n" + symbolicAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	byName := symbolicInstances(t, program)
	if len(byName["a"]) == 0 || len(byName["b"]) == 0 {
		t.Fatal("closed-component instances missing")
	}
}

func TestSymbolicInferredMutual(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [first, second]\n    uses []\nfn item first<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call second(value, true)\n        true => ok value\nfn item second<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call first(value, true)\n        true => ok value\n" + symbolicAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	symbolicInstances(t, program)
}

func TestSymbolicCallableThroughChain(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [apply, middle]\n    uses []\nfn item apply<item>\n    emits []\n    given\n        item value\n        callable item (item) emits [] each\n    asserts\n        number: 3, callable identity_int => ok 3\n    ok call each(value)\nfn int identity_int\n    emits []\n    given\n        int value\n    asserts\n        sample: 1 => ok 1\n    ok value\nfn item middle<item>\n    emits []\n    given\n        item value\n        callable item (item) emits [] each\n    asserts\n        number: 3, callable identity_int => ok 3\n    ok call apply<item>(value, each)\n" + symbolicAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	byName := symbolicInstances(t, program)
	if len(byName["apply"]) == 0 || len(byName["middle"]) == 0 {
		t.Fatal("callable-chain instances missing")
	}
}

func TestSymbolicExactEmitsThroughChain(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [checked, caller]\n    uses [codec]\nfn int checked<item>\n    emits [codec::invalid_data]\n    given\n        item value\n    asserts\n        number: 3 => ok 3\n    ok 3\nfn int caller<item>\n    emits [codec::invalid_data]\n    given\n        item value\n    asserts\n        number: 3 => ok 3\n    relay call checked<item>(value)\n" + symbolicAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	symbolicInstances(t, program)
	_, err = programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [checked, caller]\n    uses [codec]\nfn int checked<item>\n    emits [codec::invalid_data]\n    given\n        item value\n    asserts\n        number: 3 => ok 3\n    ok 3\nfn int caller<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 3\n    relay call checked<item>(value)\n" + symbolicAppMain,
	})
	if err == nil {
		t.Fatal("narrowed emits bound through a symbolic call admitted")
	}
	if !strings.Contains(err.Error(), "exported generic function") {
		t.Fatalf("missing declaration diagnosis: %v", err)
	}
}

func TestSymbolicRejectsPrivateCallee(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/app.can": "package local\n    provides [pass]\n    uses []\nfn item double<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 6\n    ok value + value\nfn item pass<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 6\n    ok call double<item>(value)\n" + symbolicAppMain,
	})
	if err == nil {
		t.Fatal("private template under opaque argument admitted")
	}
	if !strings.Contains(err.Error(), "exported generic function") || !strings.Contains(err.Error(), "opaque type parameter") || !strings.Contains(err.Error(), "private") {
		t.Fatalf("missing private-callee diagnosis: %v", err)
	}
	located, ok := source.AsLocated(err)
	if !ok || !strings.Contains(located.File, "app.can") {
		t.Fatalf("failure is not located at the symbolic call: %v", err)
	}
	notes := map[string]bool{}
	for _, span := range located.Related {
		notes[span.Note] = true
	}
	for _, want := range []string{"symbolic callee declared here", "exported generic declared here"} {
		if !notes[want] {
			t.Fatalf("failure lacks %q evidence: %v", want, err)
		}
	}
}

func TestSymbolicCalleeArithmeticFailsAtCallee(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/core/core.can": "package core\n    provides [identity]\n    uses []\nfn item identity<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 6\n    ok value + value\n",
		"src/app/main.can":  "package app\n    provides [pass]\n    uses [core]\nfn item pass<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 6\n    ok call core::identity<item>(value)\n" + symbolicAppMain,
	})
	if err == nil {
		t.Fatal("illegal callee arithmetic admitted")
	}
	if !strings.Contains(err.Error(), "exported generic function") || !strings.Contains(err.Error(), "operator + is not defined") {
		t.Fatalf("missing callee diagnosis: %v", err)
	}
	located, ok := source.AsLocated(err)
	if !ok || !strings.Contains(located.File, "core.can") {
		t.Fatalf("failure is not located at the callee declaration: %v", err)
	}
}

func TestSymbolicRejectsNestedGrowth(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [box, a, b]\n    uses []\nrecord box<item>\n    item value\nfn void a<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok\n    match stop\n        false => relay call b<box<item>>(box(value), true)\n        true => ok\nfn void b<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok\n    match stop\n        false => relay call a<item>(value, true)\n        true => ok\n" + symbolicAppMain,
	})
	if err == nil {
		t.Fatal("nested growing cycle admitted")
	}
	if !strings.Contains(err.Error(), "expanding symbolic cycle") || !strings.Contains(err.Error(), "::a") || !strings.Contains(err.Error(), "::b") {
		t.Fatalf("missing nested-growth chain: %v", err)
	}
	located, ok := source.AsLocated(err)
	if !ok || !strings.Contains(located.File, "app.can") {
		t.Fatalf("failure is not located at the growing call: %v", err)
	}
}

func TestSymbolicRejectsSelfGrowth(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [grow]\n    uses []\nfn void grow<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok\n    match stop\n        false => relay call grow<item[]>([value], true)\n        true => ok\n" + symbolicAppMain,
	})
	if err == nil {
		t.Fatal("growing self call admitted")
	}
	if !strings.Contains(err.Error(), "expanding symbolic cycle") || !strings.Contains(err.Error(), "calls itself") {
		t.Fatalf("missing self-growth diagnosis: %v", err)
	}
}

func TestSymbolicRejectsFailedComponentReuse(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/app.can": "package app\n    provides [a, b, c]\n    uses []\nfn item a<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call b<item>(value, true)\n        true => ok value\nfn item b<item>\n    emits []\n    given\n        item value\n        bool stop\n    asserts\n        base: 3, true => ok 3\n    match stop\n        false => ok call a<item>(value, true)\n        true => ok value + value\nfn item c<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 3\n    ok call a<item>(value, true)\n" + symbolicAppMain,
	})
	if err == nil {
		t.Fatal("proof reuse after failed component admitted")
	}
	// The provisional pass of `a` publishes nothing: the diagnosis is the
	// root failure at `b`, never a consumer-side false proof.
	if !strings.Contains(err.Error(), "exported generic function") || !strings.Contains(err.Error(), "operator + is not defined") || !strings.Contains(err.Error(), "::b") {
		t.Fatalf("missing failed-component root diagnosis: %v", err)
	}
}

func TestSymbolicOwnerPassthrough(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/mail/box.can":     ownerMailPackage,
		"src/helpers/help.can": "package helpers\n    provides [pass, wrap]\n    uses []\nfn item pass<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 3\n    ok value\nfn item wrap<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok 3\n    ok call pass<item>(value)\n",
		"src/app/main.can":     "package app\n    provides []\n    uses [mail, codec, bytes, helpers]\n" + "fn mail::email use_token\n    emits []\n    given\n        str address\n    asserts\n        sample: \"a@b\" => ok call mail::make_email(\"a@b\")\n    mail::email held = call mail::make_email(address)\n    ok call helpers::wrap(held)\n" + ownerAppMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	byName := symbolicInstances(t, program)
	for _, name := range []string{"pass", "wrap"} {
		seen := false
		for _, fn := range byName[name] {
			if len(fn.TypeArguments) == 1 && strings.HasSuffix(fn.TypeArguments[0].Declaration(), "::email") {
				seen = true
			}
		}
		if !seen {
			t.Fatalf("no owner-typed instance for %s", name)
		}
	}
}

func TestSymbolicRejectsOwnerConstruction(t *testing.T) {
	_, err := programFixture(t, map[string]string{
		"src/mail/box.can": ownerMailPackage,
		"src/app/main.can": "package app\n    provides [forge]\n    uses [mail, codec, bytes]\n" + "fn mail::email forge<item>\n    emits []\n    given\n        item value\n    asserts\n        number: 3 => ok call mail::make_email(\"a@b\")\n    ok mail::email(\"forged\")\n" + ownerAppMain,
	})
	if err == nil {
		t.Fatal("bogus owner construction from a generic admitted")
	}
	if !strings.Contains(err.Error(), "exported generic function") || !strings.Contains(err.Error(), "only be constructed in its declaring package") {
		t.Fatalf("missing owner-construction diagnosis: %v", err)
	}
	located, ok := source.AsLocated(err)
	if !ok || !strings.Contains(located.File, "main.can") {
		t.Fatalf("failure is not located at the offending declaration: %v", err)
	}
}

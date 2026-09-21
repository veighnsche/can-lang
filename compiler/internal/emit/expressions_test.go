package emit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"github.com/veighnsche/can-lang/distribution"
)

func expressionContext(ts map[string]*types.Type) *check.Expressions {
	values := map[string]string{"ints": "int[]", "words": "str[]", "text": "str", "item": "alpha::item", "other": "beta::item", "either": "both", "items": "alpha::item[]", "resource": "bytes::buffer", "callback": "callable int () emits []", "yes": "bool", "nan": "float", "negative_zero": "float"}
	functions := map[string]string{"first": "callable int () emits []", "middle": "callable int () emits []", "last": "callable int () emits []", "truth": "callable bool () emits []", "falsehood": "callable bool () emits []", "fallible": "callable int () emits [number::inexact]"}
	resolver := func(m map[string]string) func(syntax.QualifiedName) (check.ValueBinding, error) {
		return func(name syntax.QualifiedName) (check.ValueBinding, error) {
			typ := ts[m[name.Name]]
			if name.Package != "" || typ == nil {
				return check.ValueBinding{}, fmt.Errorf("unknown binding %s", name.Name)
			}
			return check.ValueBinding{Identity: name.Name, Type: typ}, nil
		}
	}
	return &check.Expressions{Scalars: ts, Value: resolver(values), Function: resolver(functions), Constructor: func(n *syntax.ConstructorExpr, expected *types.Type) (*types.Type, error) {
		name := n.Name.Name
		if n.Name.Package != "" {
			name = n.Name.Package + "::" + name
		}
		if len(n.Types) != 0 {
			args := make([]string, len(n.Types))
			for i, t := range n.Types {
				args[i] = syntax.FormatType(t)
			}
			name += "<" + strings.Join(args, ",") + ">"
		}
		typ := ts[name]
		if typ == nil {
			return nil, fmt.Errorf("unknown concrete constructor %s", name)
		}
		return typ, nil
	}}
}
func checkedExpression(t *testing.T, c *check.Expressions, text string, expected *types.Type) (*ir.Expression, error) {
	t.Helper()
	file, err := source.New("expression.can", text)
	if err != nil {
		t.Fatal(err)
	}
	node, ds := syntax.ParseExpression(file)
	if len(ds) != 0 {
		t.Fatal(ds)
	}
	return c.Check(node, expected)
}
func TestPrimitiveExpressionTypeRefusals(t *testing.T) {
	ts := fixtureTypes(t)
	checker := expressionContext(ts)
	for _, text := range []string{"alpha::item(1, []) is beta::item(1, [])", "bytes::buffer()", "alpha::item(1.0, [])", "(alpha::item(1, []) with value=1.0).value", "item is other", "false and (1 + 1.0 is 0)", "1 + 1.0", "1 ** 2.0", "1 and true", "not 1", "~1.0", "true < false", "1 < 2.0", "callback is callback", "resource is resource", "[callback] is [callback]", "ints[1.0]", "text[false]", "ints[0:1.0]", "1[0]", "ints.missing", "1 .length", "[1, 2.0]", "[]", "call fallible() + 1", "call text.slice(0)", "call text.slice(0, 1.0)"} {
		t.Run(text, func(t *testing.T) {
			if _, err := checkedExpression(t, checker, text, nil); err == nil {
				t.Fatalf("accepted %s", text)
			}
		})
	}
	for _, text := range []string{"0xAb + 0b10", "1 << -2", "0.0 / 0.0 is nan", "-0.0 is negative_zero", "ints[0]", "text[-1:100]", "call text.slice(0, 1)", "item.value + 1", "[1, ...ints, 2]", "[1,2] is [1,2]"} {
		if _, err := checkedExpression(t, checker, text, nil); err != nil {
			t.Fatalf("%s: %v", text, err)
		}
	}
	if _, err := checkedExpression(t, checker, "item is either", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := checkedExpression(t, checker, "[item, other]", ts["both[]"]); err != nil {
		t.Fatal(err)
	}
	if _, err := checkedExpression(t, checker, "items", ts["both[]"]); err == nil {
		t.Fatal("existing array widened")
	}
	if _, err := checkedExpression(t, checker, "[]", ts["int[]"]); err != nil {
		t.Fatal(err)
	}
}

func TestEmittedPrimitiveExpressions(t *testing.T) {
	bun := os.Getenv("CAN_BUN")
	if bun == "" {
		t.Skip("set CAN_BUN to the exact qualified absolute runtime")
	}
	target := distribution.PinnedTarget()
	binary, err := os.ReadFile(bun)
	if err != nil || !filepath.IsAbs(bun) || distribution.Hash(binary) != target.Runtime.SHA256 || runtime.GOOS != target.Runtime.Platform || runtime.GOARCH != target.Runtime.Architecture {
		t.Fatal("wrong qualified runtime")
	}
	ts := fixtureTypes(t)
	checker := expressionContext(ts)
	primitivePath, _ := filepath.Abs("../../../runtime/primitive.ts")
	var program strings.Builder
	dataPath, _ := filepath.Abs("../../../runtime/data.ts")
	program.WriteString(DataImports(dataPath))
	program.WriteString(PrimitiveImports(primitivePath))
	fmt.Fprintf(&program, "import { primitiveFailureKind, primitiveFailureMessage } from %s;\n", quote(primitivePath))
	program.WriteString(`
const assert=(ok:boolean,label:string)=>{if(!ok)throw new Error(label)};
const events:string[]=[];
const ints=Object.freeze([10n,20n,30n]);
const words=Object.freeze(["a","b"]);
const text="A😀B";
const nan=NaN;
const negative_zero=-0;
const yes=true;
function first(){events.push("first");return 1n;}
function middle(){events.push("middle");return 2n;}
function last(){events.push("last");return 3n;}
function truth(){events.push("truth");return true;}
function falsehood(){events.push("falsehood");return false;}
`)
	cases := []struct{ source, want, events string }{
		{"alpha::item(1, [2]) is alpha::item(1, [2])", "true", ""},
		{"(alpha::item(call first(), [call middle()]) with value=call last()).value", "3n", "first,middle,last"},
		{"999999999999999999999999999999 + 1", "1000000000000000000000000000000n", ""},
		{"9007199254740993 + 2", "9007199254740995n", ""},
		{"[9007199254740993, 9007199254740993] is [9007199254740993, 9007199254740993]", "true", ""},
		{"(1 | 2) ^ 1", "2n", ""}, {"16 >> 2", "4n", ""}, {"-7.0 % 3.0", "-1", ""}, {"not false", "true", ""}, {"1 is not 2", "true", ""},
		{"-7 / 3", "-2n", ""}, {"-7 % 3", "-1n", ""}, {"0 ** 0", "1n", ""},
		{"0xff & 0b1010", "10n", ""}, {"8 << -1", "4n", ""}, {"1 >> -3", "8n", ""},
		{"2 ** 3 ** 2", "512n", ""}, {"-2 ** 2", "-4n", ""}, {"~0", "-1n", ""},
		{"0.0 / 0.0 is nan", "true", ""}, {"0.0 is -0.0", "false", ""}, {"-0.0 is negative_zero", "true", ""},
		{"1.0 / 0.0", "Infinity", ""}, {"(-1.0) ** 0.5", "NaN", ""}, {"nan < 1.0", "false", ""},
		{"false and (1 / 0 is 0)", "false", ""}, {"true or (ints[100] is 0)", "true", ""},
		{"call falsehood() and call truth()", "false", "falsehood"}, {"call truth() or call falsehood()", "true", "truth"},
		{"call truth() and call falsehood()", "false", "truth,falsehood"},
		{"call first() < call middle() < call last()", "true", "first,middle,last"},
		{"call first() > call middle() < call last()", "false", "first,middle"},
		{"call first() < call middle() > call last()", "false", "first,middle,last"},
		{"ints[1]", "20n", ""}, {"text.length", "4n", ""}, {"text[1]", `"\ud83d"`, ""},
		{"text[1:3]", `"😀"`, ""}, {"call text.slice(1, 3)", `"😀"`, ""},
		{"text[-1:]", `"B"`, ""}, {"text[:0]", `""`, ""}, {"ints[-2:100]", "[20n,30n]", ""},
		{"ints[999999999999999999999999999999:]", "[]", ""},
		{"ints[-999999999999999999999999999999:999999999999999999999999999999]", "[10n,20n,30n]", ""},
		{`"x" + "😀"`, `"x😀"`, ""}, {`"😀" < ""`, "true", ""},
		{"[1, ...ints, 2]", "[1n,10n,20n,30n,2n]", ""}, {"[0.0 / 0.0] is [nan]", "true", ""}, {"[0.0] is [-0.0]", "false", ""},
	}
	emitter := &ExpressionEmitter{Bindings: map[string]string{"ints": "ints", "words": "words", "text": "text", "nan": "nan", "negative_zero": "negative_zero", "yes": "yes"}, Call: func(identity string, args []string) (LoweredExpression, error) {
		return LoweredExpression{Value: identity + "(" + strings.Join(args, ",") + ")"}, nil
	}}
	for _, tc := range cases {
		node, err := checkedExpression(t, checker, tc.source, nil)
		if err != nil {
			t.Fatalf("%s: %v", tc.source, err)
		}
		lowered, err := emitter.Lower(node)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&program, "events.length=0;\n%sassert(Bun.deepEquals(%s, %s, true), %s);\nassert(events.join(',')===%s, %s);\n", lowered.Statements, lowered.Value, tc.want, quote(tc.source), quote(tc.events), quote("order: "+tc.source))
	}
	for _, tc := range []struct{ source, kind, message string }{{"1 / 0", "arithmetic", "arithmetic: integer division by zero"}, {"1 % 0", "arithmetic", "arithmetic: integer remainder by zero"}, {"2 ** -1", "arithmetic", "arithmetic: negative integer exponent"}, {"ints[999999999999999999999999]", "bounds", "bounds: index out of range"}, {"text[-1]", "bounds", "bounds: index out of range"}} {
		node, err := checkedExpression(t, checker, tc.source, nil)
		if err != nil {
			t.Fatal(err)
		}
		lowered, err := emitter.Lower(node)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&program, "{let caught=false;try{\n%s}catch(error){caught=true;assert(primitiveFailureKind(error)===%s,%s);assert(primitiveFailureMessage(error)===%s,'failure message');}assert(caught,'missing fault');}\n", lowered.Statements, quote(tc.kind), quote(tc.source), quote(tc.message))
	}

	asyncEmitter := &ExpressionEmitter{Call: func(identity string, args []string) (LoweredExpression, error) {
		return LoweredExpression{Value: "(await laterCall(" + quote(identity) + ")).value"}, nil
	}}
	asyncNode, err := checkedExpression(t, checker, "call first() > call middle() < call last()", nil)
	if err != nil {
		t.Fatal(err)
	}
	asyncLowered, err := asyncEmitter.Lower(asyncNode)
	if err != nil {
		t.Fatal(err)
	}
	program.WriteString(`
async function laterCall(name:string){
 await Promise.resolve();
 const value = ({first,middle,last} as const)[name as "first"]();
 return Object.freeze(Object.assign(Object.create(null),{value}));
}
events.length=0;
async function ownerRegion(){
` + asyncLowered.Statements + "return " + asyncLowered.Value + `;}
assert(await ownerRegion()===false,"awaited comparison result");
assert(events.join(",")==="first,middle","awaited skipped operand");
`)
	program.WriteString("console.log('primitive expressions, skipped effects, comparison order, UTF-16 bounds and failures passed');\n")
	path := filepath.Join(t.TempDir(), "primitive.ts")
	if err = os.WriteFile(path, []byte(program.String()), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(bun, "--no-env-file", "--no-macros", "--no-install", path).CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	t.Log(strings.TrimSpace(string(output)))
}

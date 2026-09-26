package emit

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

const selfTailCountdownSource = "package app\n" +
	"    provides []\n" +
	"    uses []\n" +
	"fn int countdown\n" +
	"    emits []\n" +
	"    given\n" +
	"        int n\n" +
	"    asserts\n" +
	"        sample: 3 => ok 0\n" +
	"    match n\n" +
	"        0 => ok 0\n" +
	"        _ => relay call countdown(n - 1)\n" +
	"fn void main\n" +
	"    emits []\n" +
	"    given\n" +
	"        str[] arguments\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    ok\n"

const selfTailSwapSource = "package app\n" +
	"    provides []\n" +
	"    uses []\n" +
	"fn int swap\n" +
	"    emits []\n" +
	"    given\n" +
	"        int a\n" +
	"        int b\n" +
	"    asserts\n" +
	"        sample: 2, 5 => ok 5\n" +
	"    match a\n" +
	"        0 => ok b\n" +
	"        _ => relay call swap(b, a - 1)\n" +
	"fn void main\n" +
	"    emits []\n" +
	"    given\n" +
	"        str[] arguments\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    ok\n"

func selfTailRegion(t *testing.T, source, name string) *ir.Region {
	t.Helper()
	program := actionEmitProgram(t, map[string]string{"src/main.can": source})
	for _, fn := range program.Functions {
		if fn.Symbol.Name == name {
			return fn.Region
		}
	}
	t.Fatalf("missing concrete %s region", name)
	return nil
}

func emitSelfTailRegion(t *testing.T, region *ir.Region) string {
	t.Helper()
	emitter := RegionEmitter{Bindings: map[string]string{}, Functions: map[string]string{}}
	code, err := emitter.Function("test_loop", region)
	if err != nil {
		t.Fatal(err)
	}
	return code
}

// A04: a proven self relay lowers to a native loop with a step counter;
// failures name the iteration in private occurrence metadata.
func TestSelfTailLowersToWhileLoop(t *testing.T) {
	code := emitSelfTailRegion(t, selfTailRegion(t, selfTailCountdownSource, "countdown"))
	for _, want := range []string{
		"while (true) {",
		"continue;",
		"= 0;",
		"++;",
		`"step:"+$canRegion`,
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("lowered loop omits %q:\n%s", want, code)
		}
	}
	if strings.Count(code, "test_loop") != 1 {
		t.Fatalf("lowered loop still calls itself:\n%s", code)
	}
	if !strings.Contains(code, "} catch ($canCause) { return $canCaught($canCause, {source:") || !strings.Contains(code, `"step:"+$canRegion`) {
		t.Fatalf("loop catch lost its step origin:\n%s", code)
	}
}

// A04: relay arguments evaluate exactly once, in order, before any
// parameter moves, so swaps observe iteration values.
func TestSelfTailArgumentsEvaluateBeforeMove(t *testing.T) {
	code := emitSelfTailRegion(t, selfTailRegion(t, selfTailSwapSource, "swap"))
	relay := code[strings.Index(code, "else if ((true))"):]
	evalB := strings.Index(relay, "($canArg1)")
	evalA := strings.Index(relay, "($canArg0)")
	moveA := strings.Index(relay, "$canArg0 = ")
	moveB := strings.Index(relay, "$canArg1 = ")
	if evalB < 0 || evalA < 0 || moveA < 0 || moveB < 0 {
		t.Fatalf("relay arguments lost:\n%s", code)
	}
	if !(evalB < evalA && evalA < moveA && moveA < moveB) {
		t.Fatalf("arguments not evaluated once in order before moves:\n%s", code)
	}
	if !strings.Contains(code, "continue;") {
		t.Fatalf("swap relay did not lower:\n%s", code)
	}
}

// A04: unproven relays keep nested-call emission without a loop.
func TestSelfTailExcludedKeepsNestedCall(t *testing.T) {
	text := "package app\n" +
		"    provides []\n" +
		"    uses []\n" +
		"fn int ping\n" +
		"    emits []\n" +
		"    given\n" +
		"        int n\n" +
		"    asserts\n" +
		"        sample: 0 => ok 0\n" +
		"    match n\n" +
		"        0 => ok 0\n" +
		"        _ => relay call pong(n)\n" +
		"fn int pong\n" +
		"    emits []\n" +
		"    given\n" +
		"        int n\n" +
		"    asserts\n" +
		"        sample: 0 => ok 0\n" +
		"    match n\n" +
		"        0 => ok 0\n" +
		"        _ => relay call ping(n)\n" +
		"fn void main\n" +
		"    emits []\n" +
		"    given\n" +
		"        str[] arguments\n" +
		"    asserts\n" +
		"        empty: [] => ok\n" +
		"    ok\n"
	program := actionEmitProgram(t, map[string]string{"src/main.can": text})
	for _, fn := range program.Functions {
		if fn.Symbol.Name != "ping" {
			continue
		}
		emitter := RegionEmitter{Bindings: map[string]string{}, Functions: map[string]string{"can.project.root/app::pong": "test_pong"}}
		code, err := emitter.Function("test_ping", fn.Region)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(code, "while (true)") || strings.Contains(code, "continue;") {
			t.Fatalf("mutual relay lowered:\n%s", code)
		}
		if !strings.Contains(code, "test_pong(") {
			t.Fatalf("mutual relay lost its nested call:\n%s", code)
		}
	}
}

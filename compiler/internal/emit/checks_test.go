package emit

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// C9.2 lowers require to one native call: both authored arguments evaluate
// once each, left to right, before the call, and only the hidden origin plus
// the assertion context travel with them.
func TestChecksRequireSingleEvaluation(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/current/checks/main.can")
	if err != nil {
		t.Fatal(err)
	}
	program := sourceProgram(t, "../../testdata/current/checks/main.can")
	emitter := RegionEmitter{Functions: map[string]string{"can.std.checks@1::require": "$canChecks.require"}}
	var body string
	for _, fn := range program.Functions {
		if fn.Symbol.Name == "require_positive" {
			body, err = emitter.Function("$requirePositive", fn.Region)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if body == "" {
		t.Fatal("require_positive region missing")
	}
	if got := strings.Count(body, "$canChecks.require("); got != 1 {
		t.Fatalf("require calls emitted %d times", got)
	}
	if got := strings.Count(body, `"expected a positive value"`); got != 1 {
		t.Fatalf("reason lowered %d times", got)
	}
	if got := strings.Count(body, "0n"); got != 1 {
		t.Fatalf("condition bound lowered %d times", got)
	}
	if strings.Index(body, "0n") > strings.Index(body, `"expected a positive value"`) {
		t.Fatal("reason lowered before condition")
	}
	call := body[strings.Index(body, "$canChecks.require("):]
	depth := 0
	end := -1
	for i, r := range call {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		t.Fatal("unterminated require call")
	}
	args := call[:end]
	for _, want := range []string{"{source:", ",start:", ",end:", ",invocation:["} {
		if !strings.Contains(args, want) {
			t.Fatalf("hidden origin argument misses %s: %s", want, args)
		}
	}
	for _, dup := range []string{`"expected a positive value"`, "0n", ">"} {
		if strings.Contains(args, dup) {
			t.Fatalf("authored argument re-evaluated inside call: %s", args)
		}
	}
	var source string
	var start, stop int
	at := strings.Index(args, "{source:")
	if at < 0 {
		t.Fatalf("origin literal missing: %s", args)
	}
	if _, err := fmt.Sscanf(args[at:], "{source:%q,start:%d,end:%d", &source, &start, &stop); err != nil {
		t.Fatalf("origin literal unreadable: %v", err)
	}
	if stop > len(raw) || string(raw[start:stop]) != `checks::require(value > 0, "expected a positive value")` {
		t.Fatalf("origin span [%d,%d) misses the require call site", start, stop)
	}
}

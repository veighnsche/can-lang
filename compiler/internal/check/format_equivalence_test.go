package check

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const formatEquivalenceSource = `// leading header comment


package app
    provides []
    uses []

// doubler docs
fn   int   double
    emits []
    given
        int value  // trailing
    asserts
        sample: 2 => ok 4
    ok value + value

fn int consumer  // trailing consumer
    emits []
    asserts
        sample: => ok 18
    // first site below
    int first = call double(3)
    match call double(first)
        when
            low: 1 => ok 2  // first row
            high: 9 => ok 18

        ok int got => ok got + first  // trailing arm
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

const formatMisorderedSource = `package app
    provides []
    uses [codec]
fn int number
    emits [codec::invalid_data]
    asserts
        sample: => ok 1
    ok 1
fn int bad  // trailing
    emits [codec::invalid_data]
    asserts
        sample: => ok 1
    match call number()
        ok int got => ok got
        codec::invalid_data => ok 0
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`

// Formatting moves only trivia: the checked program before and after must
// agree structurally with source positions ignored. Lexical site order,
// fixture FIFO rows and assertion roots all participate.
func TestFormatPreservesCheckedSemantics(t *testing.T) {
	original, err := programFixture(t, map[string]string{"src/main.can": formatEquivalenceSource})
	if err != nil {
		t.Fatal(err)
	}
	src, err := source.New("main.can", formatEquivalenceSource)
	if err != nil {
		t.Fatal(err)
	}
	parsed := syntax.Parse(src)
	if !parsed.OK() {
		t.Fatalf("parse: %+v", parsed.Diagnostics)
	}
	formatted, err := syntax.FormatTrivia(parsed.File)
	if err != nil {
		t.Fatal(err)
	}
	if formatted == formatEquivalenceSource {
		t.Fatal("fixture is already canonical; nothing is proven")
	}
	after, err := programFixture(t, map[string]string{"src/main.can": formatted})
	if err != nil {
		t.Fatalf("formatted program rejected: %v\n%s", err, formatted)
	}
	compare := &irComparer{t: t}
	compare.programs(original, after)
}

// A misordered match still fails after formatting: the renderer never
// repairs arm order, bounds or ownership to obtain a passing check.
func TestFormatNeverRepairsArms(t *testing.T) {
	bad := formatMisorderedSource
	if _, err := programFixture(t, map[string]string{"src/main.can": bad}); err == nil {
		t.Fatal("misordered arms admitted")
	} else if !strings.Contains(err.Error(), "precede the final ok") {
		t.Fatalf("misordered arms misdiagnosed: %v", err)
	}
	src, err := source.New("main.can", bad)
	if err != nil {
		t.Fatal(err)
	}
	parsed := syntax.Parse(src)
	if !parsed.OK() {
		t.Fatalf("parse: %+v", parsed.Diagnostics)
	}
	formatted, err := syntax.FormatTrivia(parsed.File)
	if err != nil {
		t.Fatal(err)
	}
	okAt := strings.Index(formatted, "ok int got => ok got")
	errAt := strings.Index(formatted, "codec::invalid_data => ok 0")
	if okAt < 0 || errAt < 0 || okAt > errAt {
		t.Fatalf("formatter reordered arms:\n%s", formatted)
	}
	if _, err := programFixture(t, map[string]string{"src/main.can": formatted}); err == nil {
		t.Fatal("formatted misordered arms admitted")
	} else if !strings.Contains(err.Error(), "precede the final ok") {
		t.Fatalf("formatted misorder misdiagnosed: %v", err)
	}
}

type irComparer struct {
	t *testing.T
}

func (c *irComparer) fail(format string, args ...any) {
	c.t.Helper()
	c.t.Fatalf(format, args...)
}

func typeID(t *types.Type) string {
	if t == nil {
		return "<nil>"
	}
	return t.Identity()
}

func typeIDs(list []*types.Type) []string {
	out := make([]string, 0, len(list))
	for _, t := range list {
		out = append(out, typeID(t))
	}
	return out
}

func (c *irComparer) strings(what string, a, b []string) {
	c.t.Helper()
	if len(a) != len(b) {
		c.fail("%s length %d != %d (%v vs %v)", what, len(a), len(b), a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			c.fail("%s [%d] %q != %q", what, i, a[i], b[i])
		}
	}
}

func (c *irComparer) programs(a, b *Program) {
	c.t.Helper()
	if len(a.Functions) != len(b.Functions) {
		c.fail("function count %d != %d", len(a.Functions), len(b.Functions))
	}
	for i := range a.Functions {
		if a.Functions[i].Symbol.ID != b.Functions[i].Symbol.ID {
			c.fail("function [%d] %s != %s", i, a.Functions[i].Symbol.ID, b.Functions[i].Symbol.ID)
		}
		c.region("function "+a.Functions[i].Symbol.ID, a.Functions[i].Region, b.Functions[i].Region)
	}
	if len(a.Assertions) != len(b.Assertions) {
		c.fail("assertion count %d != %d", len(a.Assertions), len(b.Assertions))
	}
	for i := range a.Assertions {
		ra, rb := a.Assertions[i].Root, b.Assertions[i].Root
		if ra != rb {
			c.fail("assertion root [%d] %+v != %+v", i, ra, rb)
		}
		c.region("assertion actual", a.Assertions[i].Actual, b.Assertions[i].Actual)
		c.region("assertion expected", a.Assertions[i].Expected, b.Assertions[i].Expected)
		if (a.Assertions[i].Raw == nil) != (b.Assertions[i].Raw == nil) {
			c.fail("assertion [%d] raw presence differs", i)
		}
	}
}

func (c *irComparer) local(what string, i int, a, b *ir.Local) {
	c.t.Helper()
	if a.Identity != b.Identity || typeID(a.Type) != typeID(b.Type) || a.ErrorAlias != b.ErrorAlias {
		c.fail("%s [%d] %s/%s != %s/%s", what, i, a.Identity, typeID(a.Type), b.Identity, typeID(b.Type))
	}
}

func (c *irComparer) region(what string, a, b *ir.Region) {
	c.t.Helper()
	if a.ID != b.ID || a.Parent != b.Parent || filepath.Base(a.Source) != filepath.Base(b.Source) || a.Kind != b.Kind {
		c.fail("%s identity %s/%s/%s != %s/%s/%s", what, a.ID, a.Parent, a.Source, b.ID, b.Parent, b.Source)
	}
	if typeID(a.Result) != typeID(b.Result) {
		c.fail("%s result %s != %s", what, typeID(a.Result), typeID(b.Result))
	}
	c.strings(what+" errors", typeIDs(a.Errors), typeIDs(b.Errors))
	c.strings(what+" escapes", typeIDs(a.Escapes), typeIDs(b.Escapes))
	if len(a.Inputs) != len(b.Inputs) {
		c.fail("%s inputs %d != %d", what, len(a.Inputs), len(b.Inputs))
	}
	for i := range a.Inputs {
		c.local(what+" input", i, &a.Inputs[i], &b.Inputs[i])
	}
	c.block(what+" body", a.Body, b.Body)
}

func (c *irComparer) block(what string, a, b *ir.Block) {
	c.t.Helper()
	if len(a.Steps) != len(b.Steps) {
		c.fail("%s steps %d != %d", what, len(a.Steps), len(b.Steps))
	}
	for i := range a.Steps {
		c.statement(what, i, &a.Steps[i], &b.Steps[i])
	}
	c.completion(what+" terminal", a.Terminal, b.Terminal)
}

func (c *irComparer) statement(what string, i int, a, b *ir.Statement) {
	c.t.Helper()
	if (a.Local == nil) != (b.Local == nil) || (a.Value == nil) != (b.Value == nil) || (a.Call == nil) != (b.Call == nil) {
		c.fail("%s step [%d] shape differs", what, i)
	}
	if a.Coordination != nil || b.Coordination != nil {
		c.fail("%s step [%d] uses coordination outside this comparison", what, i)
	}
	if a.Local != nil {
		c.local(what+" local", i, a.Local, b.Local)
	}
	if a.Value != nil {
		c.expression(what+" value", a.Value, b.Value)
	}
	if a.Call != nil {
		c.invocation(what+" call", a.Call, b.Call)
	}
}

func (c *irComparer) completion(what string, a, b *ir.Completion) {
	c.t.Helper()
	if a.Kind != b.Kind || a.RegionID != b.RegionID || a.Inherit != b.Inherit {
		c.fail("%s kind/region %+v != %+v", what, a, b)
	}
	if (a.Value == nil) != (b.Value == nil) || (a.Call == nil) != (b.Call == nil) || (a.Block == nil) != (b.Block == nil) || (a.Match == nil) != (b.Match == nil) {
		c.fail("%s shape differs", what)
	}
	if a.Value != nil {
		c.expression(what+" value", a.Value, b.Value)
	}
	if a.Call != nil {
		c.invocation(what+" call", a.Call, b.Call)
	}
	if a.Block != nil {
		c.block(what+" block", a.Block, b.Block)
	}
	if a.Match != nil {
		c.match(what+" match", a.Match, b.Match)
	}
}

func (c *irComparer) match(what string, a, b *ir.Match) {
	c.t.Helper()
	if len(a.Values) != len(b.Values) || (a.Call == nil) != (b.Call == nil) || len(a.Arms) != len(b.Arms) || typeID(a.ValueResult) != typeID(b.ValueResult) {
		c.fail("%s shape differs", what)
	}
	for i := range a.Values {
		c.expression(what+" value", a.Values[i], b.Values[i])
	}
	if a.Call != nil {
		c.invocation(what+" call", a.Call, b.Call)
	}
	for i := range a.Arms {
		aa, bb := &a.Arms[i], &b.Arms[i]
		if len(aa.Patterns) != 0 || len(bb.Patterns) != 0 {
			c.fail("%s arm [%d] uses data patterns outside this comparison", what, i)
		}
		if aa.Outcome != bb.Outcome || typeID(aa.Error) != typeID(bb.Error) || aa.Forward != bb.Forward {
			c.fail("%s arm [%d] differs", what, i)
		}
		if (aa.Binding == nil) != (bb.Binding == nil) {
			c.fail("%s arm [%d] binding presence differs", what, i)
		}
		if aa.Binding != nil {
			c.local(what+" arm binding", i, aa.Binding, bb.Binding)
		}
		if (aa.Body == nil) != (bb.Body == nil) || (aa.Value == nil) != (bb.Value == nil) {
			c.fail("%s arm [%d] body presence differs", what, i)
		}
		if aa.Body != nil {
			c.completion(what+" arm", aa.Body, bb.Body)
		}
		if aa.Value != nil {
			c.expression(what+" arm value", aa.Value, bb.Value)
		}
	}
}

func (c *irComparer) invocation(what string, a, b *ir.Invocation) {
	c.t.Helper()
	if typeID(a.Result) != typeID(b.Result) {
		c.fail("%s result differs", what)
	}
	c.strings(what+" errors", typeIDs(a.Errors), typeIDs(b.Errors))
	if len(a.Steps) != len(b.Steps) {
		c.fail("%s steps %d != %d", what, len(a.Steps), len(b.Steps))
	}
	for i := range a.Steps {
		sa, sb := &a.Steps[i], &b.Steps[i]
		if sa.Site != sb.Site || sa.Identity != sb.Identity || sa.Receiver != sb.Receiver || sa.SuccessBinding != sb.SuccessBinding {
			c.fail("%s step [%d] site/identity %+v != %+v", what, i, sa, sb)
		}
		if typeID(sa.Contract) != typeID(sb.Contract) || typeID(sa.Result) != typeID(sb.Result) {
			c.fail("%s step [%d] contract differs", what, i)
		}
		c.strings(what+" step errors", typeIDs(sa.Errors), typeIDs(sb.Errors))
		if (sa.Callee == nil) != (sb.Callee == nil) || len(sa.Arguments) != len(sb.Arguments) || len(sa.Prepare) != len(sb.Prepare) {
			c.fail("%s step [%d] shape differs", what, i)
		}
		if sa.Array != nil || sb.Array != nil || sa.Native != nil || sb.Native != nil || sa.Asset != nil || sb.Asset != nil || sa.SQL != nil || sb.SQL != nil {
			c.fail("%s step [%d] uses native forms outside this comparison", what, i)
		}
		if sa.Callee != nil {
			c.expression(what+" callee", sa.Callee, sb.Callee)
		}
		for j := range sa.Arguments {
			c.expression(what+" argument", sa.Arguments[j], sb.Arguments[j])
		}
		for j := range sa.Prepare {
			c.local(what+" preparation", j, &sa.Prepare[j].Local, &sb.Prepare[j].Local)
			c.expression(what+" preparation", sa.Prepare[j].Value, sb.Prepare[j].Value)
		}
		if (sa.Fixtures == nil) != (sb.Fixtures == nil) {
			c.fail("%s step [%d] fixture presence differs", what, i)
		}
		if sa.Fixtures != nil {
			if sa.Fixtures.Identity != sb.Fixtures.Identity || len(sa.Fixtures.Rows) != len(sb.Fixtures.Rows) {
				c.fail("%s step [%d] fixture table differs", what, i)
			}
			for j := range sa.Fixtures.Rows {
				ra, rb := &sa.Fixtures.Rows[j], &sb.Fixtures.Rows[j]
				if ra.Selector != rb.Selector || len(ra.Arguments) != len(rb.Arguments) || len(ra.Prepare) != len(rb.Prepare) || (ra.Raw == nil) != (rb.Raw == nil) {
					c.fail("%s step [%d] fixture row [%d] differs", what, i, j)
				}
				for k := range ra.Arguments {
					c.expression(what+" fixture argument", ra.Arguments[k], rb.Arguments[k])
				}
				for k := range ra.Prepare {
					c.local(what+" fixture preparation", k, &ra.Prepare[k].Local, &rb.Prepare[k].Local)
					c.expression(what+" fixture preparation", ra.Prepare[k].Value, rb.Prepare[k].Value)
				}
				if (ra.Expected == nil) != (rb.Expected == nil) {
					c.fail("%s step [%d] fixture completion presence differs", what, i)
				}
				if ra.Expected != nil {
					c.completion(what+" fixture completion", ra.Expected, rb.Expected)
				}
			}
		}
	}
}

func (c *irComparer) expression(what string, a, b *ir.Expression) {
	c.t.Helper()
	if a.Kind != b.Kind || a.Source != b.Source || typeID(a.Type) != typeID(b.Type) || a.Text != b.Text {
		c.fail("%s %+v != %+v", what, a, b)
	}
	c.strings(what+" operators", a.Operators, b.Operators)
	if len(a.Inputs) != len(b.Inputs) || len(a.Fields) != len(b.Fields) || len(a.Spread) != len(b.Spread) || len(a.Equality) != len(b.Equality) {
		c.fail("%s shape differs", what)
	}
	c.strings(what+" fields", a.Fields, b.Fields)
	for i := range a.Inputs {
		c.expression(what+" input", a.Inputs[i], b.Inputs[i])
	}
	if (a.Coordination == nil) != (b.Coordination == nil) || (a.Callable == nil) != (b.Callable == nil) || (a.Invocation == nil) != (b.Invocation == nil) || (a.Match == nil) != (b.Match == nil) {
		c.fail("%s nested shape differs", what)
	}
	if a.Coordination != nil || a.Callable != nil {
		c.fail("%s uses coordination/callable outside this comparison", what)
	}
	if a.Invocation != nil {
		c.invocation(what+" invocation", a.Invocation, b.Invocation)
	}
	if a.Match != nil {
		c.match(what+" match", a.Match, b.Match)
	}
}

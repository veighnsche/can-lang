package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

// C5.1: every error arm precedes exactly one final ok in each completion
// region handling both; the offending head carries the diagnostic span.
func TestCompletionArmOrder(t *testing.T) {
	callProgram := func(t *testing.T, arms string) error {
		t.Helper()
		text := strings.Replace(programHeader, "uses []", "uses [codec]", 1) +
			coordinationDeclarations + programMain +
			"    match call number()\n" + arms
		_, err := programFixture(t, map[string]string{"src/main.can": text})
		return err
	}
	chainProgram := func(t *testing.T, arms string) error {
		t.Helper()
		text := strings.Replace(programHeader, "uses []", "uses [codec]", 1) +
			coordinationDeclarations + programMain +
			"    match chain\n        call number() as int entry\n" + arms
		_, err := programFixture(t, map[string]string{"src/main.can": text})
		return err
	}
	expectOrder := func(t *testing.T, err error) {
		t.Helper()
		if err == nil {
			t.Fatal("misordered arms admitted")
		}
		if !strings.Contains(err.Error(), "failure arms precede the final ok") &&
			!strings.Contains(err.Error(), "duplicate completion arm for ok") {
			t.Fatalf("wrong order diagnostic: %v", err)
		}
		located, ok := source.AsLocated(err)
		if !ok || located.Span.Start == 0 || located.Span.End <= located.Span.Start {
			t.Fatalf("order diagnostic lost its head span: %v", err)
		}
	}
	t.Run("error after success", func(t *testing.T) {
		expectOrder(t, callProgram(t,
			"        ok int value => ok\n"+
				"        codec::invalid_data => ok\n",
		))
	})
	t.Run("standard after success", func(t *testing.T) {
		expectOrder(t, callProgram(t,
			"        ok int value => ok\n"+
				"        [_] as standard_failure failure => ok\n",
		))
	})
	t.Run("string binder rejected", func(t *testing.T) {
		err := callProgram(t,
			"        [_] as str message => ok\n"+
				"        ok int value => ok\n",
		)
		if err == nil || !strings.Contains(err.Error(), "standard_failure snapshot") {
			t.Fatalf("string binder admitted: %v", err)
		}
	})
	t.Run("duplicate success", func(t *testing.T) {
		expectOrder(t, callProgram(t,
			"        codec::invalid_data => ok\n"+
				"        ok int value => ok\n"+
				"        ok int again => ok\n",
		))
	})
	t.Run("chain error after success", func(t *testing.T) {
		expectOrder(t, chainProgram(t,
			"        ok => ok\n"+
				"        codec::invalid_data => ok\n",
		))
	})
	t.Run("race shared error after success", func(t *testing.T) {
		body := "    callable int () emits [codec::invalid_data][] operations = [callable number]\n" +
			"    match call race with error\n" +
			"        ...operations\n" +
			"        number()\n" +
			"        ok int value => ok\n" +
			"        codec::invalid_data => ok\n"
		_, err := coordinationProgram(t, body)
		expectOrder(t, err)
	})
	t.Run("missing success", func(t *testing.T) {
		err := callProgram(t, "        codec::invalid_data => ok\n")
		if err == nil || !strings.Contains(err.Error(), "requires exactly one success arm") {
			t.Fatalf("missing success admitted: %v", err)
		}
	})
	t.Run("failure first accepts", func(t *testing.T) {
		if err := callProgram(t,
			"        codec::invalid_data => ok\n"+
				"        [_] as standard_failure failure => ok\n"+
				"        ok int value => ok\n",
		); err != nil {
			t.Fatalf("failure-first call rejected: %v", err)
		}
	})
}

// Missing arms name the bound obligation with the match span, the
// invocation link and one inserted forward arm when forwarding preserves
// the region contract. Anything else is omitted rather than guessed.
func TestMissingArmObligation(t *testing.T) {
	forwarding := strings.Replace(programHeader, "uses []", "uses [codec]", 1) +
		coordinationDeclarations +
		`fn int handle
    emits [codec::invalid_data]
    asserts
        sample: => ok 1
    match call number()
        ok int value => ok value
` + programMain + "    ok\n"
	_, err := programFixture(t, map[string]string{"src/main.can": forwarding})
	if err == nil {
		t.Fatal("missing error arm admitted")
	}
	for _, want := range []string{"missing completion arm for codec::invalid_data", "requires an arm for each bound member", "can.project.root/app::number"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("obligation diagnostic omits %q: %v", want, err)
		}
	}
	located, ok := source.AsLocated(err)
	if !ok {
		t.Fatalf("missing-arm diagnostic lost its span: %v", err)
	}
	if located.Code != "CAN-CHECK-MISSING-ARM" {
		t.Fatalf("missing-arm diagnostic code is %q", located.Code)
	}
	if located.Span.Start == 0 || located.Span.End <= located.Span.Start {
		t.Fatalf("missing-arm diagnostic lost its match span: %v", err)
	}
	if len(located.Related) != 1 || !strings.Contains(located.Related[0].Note, "invocation requires an arm") {
		t.Fatalf("missing-arm diagnostic lost its invocation link: %+v", located.Related)
	}
	if len(located.Fixes) != 1 {
		t.Fatalf("forwardable arm produced %d fixes", len(located.Fixes))
	}
	fix := located.Fixes[0]
	if fix.Start != fix.End || !strings.HasSuffix(fix.Text, "codec::invalid_data\n") || !strings.Contains(fix.Title, "codec::invalid_data") {
		t.Fatalf("forward fix is not an arm insertion: %+v", fix)
	}

	strict := strings.Replace(programHeader, "uses []", "uses [codec]", 1) +
		coordinationDeclarations +
		`fn int strict
    emits []
    asserts
        sample: => ok 1
    match call number()
        ok int value => ok value
` + programMain + "    ok\n"
	_, err = programFixture(t, map[string]string{"src/main.can": strict})
	if err == nil {
		t.Fatal("missing error arm admitted under empty emits")
	}
	located, ok = source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-MISSING-ARM" {
		t.Fatalf("strict missing-arm diagnostic lost code or span: %v", err)
	}
	if len(located.Fixes) != 0 {
		t.Fatalf("unforwardable arm produced fixes: %+v", located.Fixes)
	}

	voidSuccess := strings.Replace(programHeader, "uses []", "uses [codec]", 1) +
		coordinationDeclarations +
		`fn void sink
    emits [codec::invalid_data]
    asserts
        sample: => ok
    match call number()
        codec::invalid_data => ok
` + programMain + "    ok\n"
	_, err = programFixture(t, map[string]string{"src/main.can": voidSuccess})
	if err == nil || !strings.Contains(err.Error(), "requires exactly one success arm") {
		t.Fatalf("missing success admitted: %v", err)
	}
	located, ok = source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-MISSING-ARM" || len(located.Fixes) != 1 || !strings.HasSuffix(located.Fixes[0].Text, "ok => ok\n") {
		t.Fatalf("void success fix missing: %+v", located)
	}

	valuedSuccess := strings.Replace(programHeader, "uses []", "uses [codec]", 1) +
		coordinationDeclarations +
		`fn int valued
    emits [codec::invalid_data]
    asserts
        sample: => ok 1
    match call number()
        codec::invalid_data => ok 0
` + programMain + "    ok\n"
	_, err = programFixture(t, map[string]string{"src/main.can": valuedSuccess})
	if err == nil {
		t.Fatal("missing valued success admitted")
	}
	located, ok = source.AsLocated(err)
	if !ok || len(located.Fixes) != 0 {
		t.Fatalf("valued success produced fixes: %+v", located)
	}
}

// Outward errors name the escaping member at the escaping expression and
// link the region bound both must agree on. No fix is proposed: widening
// emits or dropping the escape would change the region contract.
func TestOutwardErrorObligation(t *testing.T) {
	relay := strings.Replace(programHeader, "uses []", "uses [codec]", 1) +
		coordinationDeclarations +
		`fn int pass
    emits []
    asserts
        sample: => ok 1
    relay call number()
` + programMain + "    ok\n"
	_, err := programFixture(t, map[string]string{"src/main.can": relay})
	if err == nil {
		t.Fatal("relay escape admitted")
	}
	for _, want := range []string{"undeclared escaping domain error", "region declares emits []"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("outward diagnostic omits %q: %v", want, err)
		}
	}
	located, ok := source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-OUTWARD-ERROR" {
		t.Fatalf("relay escape lost code or span: %v", err)
	}
	if located.Span.Start == 0 || located.Span.End <= located.Span.Start {
		t.Fatalf("relay escape lost its expression span: %v", err)
	}
	if len(located.Related) != 1 || !strings.Contains(located.Related[0].Note, "extend the region bound") {
		t.Fatalf("relay escape lost its region link: %+v", located.Related)
	}
	if len(located.Fixes) != 0 {
		t.Fatalf("outward error proposed fixes: %+v", located.Fixes)
	}

	construct := strings.Replace(programHeader, "uses []", "uses [codec]", 1) +
		coordinationDeclarations +
		`fn int raise
    emits []
    asserts
        sample: => ok 1
    codec::invalid_data("b", "type")
` + programMain + "    ok\n"
	_, err = programFixture(t, map[string]string{"src/main.can": construct})
	if err == nil {
		t.Fatal("constructed escape admitted")
	}
	located, ok = source.AsLocated(err)
	if !ok || located.Code != "CAN-CHECK-OUTWARD-ERROR" || len(located.Related) != 0 || len(located.Fixes) != 0 {
		t.Fatalf("constructed escape misdiagnosed: %+v", located)
	}
}

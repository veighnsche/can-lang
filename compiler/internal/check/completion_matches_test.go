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

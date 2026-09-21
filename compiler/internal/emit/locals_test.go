package emit

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func localContext(t *testing.T, ts map[string]*types.Type, typ, initializer string) check.LocalForwarding {
	t.Helper()
	text := "package app\n    provides []\n    uses [alpha]\nfn " + typ + " run\n    emits []\n    asserts\n        example: => ok 1\n    " + typ + " result = " + initializer + "\n    ok result\n"
	file, _ := source.New("locals.can", text)
	parsed := syntax.Parse(file)
	if !parsed.OK() {
		t.Fatal(parsed.Diagnostics)
	}
	block := parsed.File.Declarations[0].(*syntax.FunctionDecl).Body
	uses := check.LocalUses{Names: map[*syntax.NameExpr]string{}, Captures: map[*syntax.ReferenceExpr][]string{}}
	// The fixture supplies resolved identities to the body-independent I48 pass.
	// Tests below mutate these records explicitly to exercise shadowing/captures.
	var visit func(reflect.Value)
	visit = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Interface {
			if !v.IsNil() {
				visit(v.Elem())
			}
			return
		}
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return
			}
			if v.CanInterface() {
				switch n := v.Interface().(type) {
				case *syntax.NameExpr:
					uses.Names[n] = "value:" + n.Name.Name
				case *syntax.ReferenceExpr:
					uses.Captures[n] = []string{}
				}
			}
			visit(v.Elem())
			return
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				visit(v.Field(i))
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i))
			}
		}
	}
	visit(reflect.ValueOf(block))
	terminal := block.Terminal.(*syntax.SuccessBody).Value.(*syntax.NameExpr)
	uses.Names[terminal] = "local:result"
	checker := expressionContext(ts)
	prior := checker.Value
	checker.Value = func(name syntax.QualifiedName) (check.ValueBinding, error) {
		if name.Package == "" && (name.Name == "left" || name.Name == "right" || name.Name == "result") {
			return check.ValueBinding{Identity: "value:" + name.Name, Type: ts["int"]}, nil
		}
		return prior(name)
	}
	return check.LocalForwarding{File: file, Block: block, Binding: check.ValueBinding{Identity: "local:result", Type: ts[typ]}, Uses: uses, Checker: checker, Expected: ts[typ]}
}

func TestFiniteLocalRule(t *testing.T) {
	ts := fixtureTypes(t)
	for _, example := range []struct{ typ, expr string }{
		{"int", "left + right"}, {"int", "(left * 2) - right"}, {"int", "~left | right & 3"}, {"int", "item.value"}, {"int", "ints.length"}, {"bool", "left < right and not false"}, {"str", "text + \"!\""}, {"int", "result + 1"},
	} {
		t.Run(example.expr, func(t *testing.T) {
			c := localContext(t, ts, example.typ, example.expr)
			err := check.CheckLocalForwarding(c)
			var diagnostic *check.UnnecessaryLocal
			if !errors.As(err, &diagnostic) || diagnostic.Replacement != "ok "+example.expr {
				t.Fatalf("expected finite replacement, got %v", err)
			}
			if diagnostic.Span.Start <= 0 {
				t.Fatal("missing source span")
			}
		})
	}
	for _, example := range []struct{ typ, expr string }{
		{"int", "left / right"}, {"int", "left % right"}, {"int", "left ** right"}, {"int", "left << right"}, {"int", "left >> right"},
		{"float", "1.0 / 2.0"}, {"int", "ints[0]"}, {"int[]", "ints[0:1]"}, {"int[]", "[]"}, {"int[]", "[1]"},
		{"alpha::item", "alpha::item(1, [])"}, {"alpha::item", "item with value=1"}, {"int", "call first()"},
		{"callable int () emits []", "callable first"}, {"both", "item"},
	} {
		t.Run("retain/"+example.expr, func(t *testing.T) {
			c := localContext(t, ts, example.typ, example.expr)
			if err := check.CheckLocalForwarding(c); err != nil {
				t.Fatal(err)
			}
		})
	}
	c := localContext(t, ts, "int", "left + right")
	// The same candidate in a value-producing arm reports an expression, not ok.
	terminal := c.Block.Terminal.(*syntax.SuccessBody)
	c.Block.Terminal = &syntax.ValueBody{BodyLocation: terminal.BodyLocation, Value: terminal.Value}
	var diagnostic *check.UnnecessaryLocal
	if err := check.CheckLocalForwarding(c); !errors.As(err, &diagnostic) || diagnostic.Replacement != "left + right" {
		t.Fatal("wrong value-arm replacement", err)
	}
}

func TestLocalUseIdentityAndCaptureEvidence(t *testing.T) {
	ts := fixtureTypes(t)
	t.Run("same spelling is not same binding", func(t *testing.T) {
		c := localContext(t, ts, "int", "left + right")
		name := c.Block.Terminal.(*syntax.SuccessBody).Value.(*syntax.NameExpr)
		c.Uses.Names[name] = "outer:result"
		if err := check.CheckLocalForwarding(c); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("two resolved uses retain the local", func(t *testing.T) {
		c := localContext(t, ts, "int", "result + 1")
		for name := range c.Uses.Names {
			if name.Name.Name == "result" {
				c.Uses.Names[name] = "local:result"
			}
		}
		if err := check.CheckLocalForwarding(c); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("incomplete names refuse rather than guess", func(t *testing.T) {
		c := localContext(t, ts, "int", "left + right")
		for name := range c.Uses.Names {
			if name.Name.Name == "left" {
				delete(c.Uses.Names, name)
			}
		}
		if err := check.CheckLocalForwarding(c); err == nil || !strings.Contains(err.Error(), "missing resolved") {
			t.Fatal(err)
		}
	})
	for _, mode := range []string{"captured", "not captured", "missing evidence"} {
		t.Run(mode, func(t *testing.T) {
			c := localContext(t, ts, "int", "left + right")
			// A resolved reference elsewhere in the owning scope can require an implicit
			// exact-name capture without spelling the captured name in its source AST.
			ref := &syntax.ReferenceExpr{Callee: &syntax.NameExpr{Name: syntax.QualifiedName{Name: "worker"}}}
			c.Uses.Names[ref.Callee.(*syntax.NameExpr)] = "function:worker"
			c.Block.Steps = append([]syntax.Step{&syntax.BindingStep{Binding: syntax.Binding{Value: ref}}}, c.Block.Steps...)
			switch mode {
			case "captured":
				c.Uses.Captures[ref] = []string{"local:result"}
			case "not captured":
				c.Uses.Captures[ref] = []string{}
			}
			err := check.CheckLocalForwarding(c)
			switch mode {
			case "captured":
				if err != nil {
					t.Fatal(err)
				}
			case "not captured":
				var diagnostic *check.UnnecessaryLocal
				if !errors.As(err, &diagnostic) {
					t.Fatal(err)
				}
			default:
				if err == nil || !strings.Contains(err.Error(), "capture evidence") {
					t.Fatal(err)
				}
			}
		})
	}
	t.Run("intervening work retains timing", func(t *testing.T) {
		c := localContext(t, ts, "int", "left + right")
		c.Block.Steps = append(c.Block.Steps, &syntax.CallStep{})
		if err := check.CheckLocalForwarding(c); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("terminal calculation is not immediate name forwarding", func(t *testing.T) {
		c := localContext(t, ts, "int", "left + right")
		c.Block.Terminal = &syntax.SuccessBody{Value: &syntax.BinaryExpr{Operator: "+"}}
		if err := check.CheckLocalForwarding(c); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("substitution rechecks expected typing", func(t *testing.T) {
		c := localContext(t, ts, "both", "item")
		if err := check.CheckLocalForwarding(c); err != nil {
			t.Fatal(fmt.Errorf("variant conversion must retain its local: %w", err))
		}
	})
}

package main

import (
	"fmt"
)

// B11's typed Ok is an outcome boundary, never a data constructor. The
// annotation is exact; it does not change the existing declared success ABI.
func (c *tycker) checkWholeOk(s *Small, want string, line int, env map[string]string) {
	fail := func(msg string) {
		c.out = append(c.out, spanDiag(c.text, line, "error", msg, "Ok", CodeTypeMismatch))
	}
	if len(s.TypeArgs) != 1 || !c.knownType(s.TypeArgs[0]) {
		fail("typed Ok needs exactly one known success type")
		return
	}
	typ := s.TypeArgs[0]
	if want == "" || !sameType(typ, want) {
		fail(fmt.Sprintf("typed Ok<%s> requires a matching success boundary, got %s", typ, want))
	}
	if len(s.Args) != 1 || (s.Args[0].HasName && s.Args[0].Name != "value") {
		fail("typed Ok takes exactly one whole value (positional or value =)")
		for _, a := range s.Args {
			c.value(a.V, "", line, env, "typed Ok value")
		}
		return
	}
	a := &s.Args[0]
	c.value(a.V, "", line, env, "typed Ok value")
	if got, ok := c.typeOf(a.V, env); !ok {
		fail(fmt.Sprintf("typed Ok needs a data value of type %s, not an outcome", typ))
	} else if !sameType(got, typ) {
		c.mismatch(line, "typed Ok value", got, typ, tokenOf(a.V))
	}
	a.Name, a.HasName = "value", true
}

func (c *tycker) patternSuccessType(p Pattern, ret string, line int) string {
	if len(p.TypeArgs) == 0 {
		return c.successBinderType(ret)
	}
	if len(p.TypeArgs) != 1 || !c.knownType(p.TypeArgs[0]) || !sameType(p.TypeArgs[0], ret) {
		c.out = append(c.out, spanDiag(c.text, line, "error",
			fmt.Sprintf("typed Ok pattern must name the exact success type %s", ret), "Ok", CodeTypeMismatch))
	}
	return ret
}

// Kernel-only synthetic payloads have no source-namable success type.
func (c *tycker) rejectKernelWholeOk(p Pattern, line int) {
	if len(p.TypeArgs) != 0 {
		c.out = append(c.out, spanDiag(c.text, line, "error",
			"typed Ok pattern requires a declared success type, not a synthetic kernel payload", "Ok", CodeTypeMismatch))
	}
}

// Proof lowering alone expands a whole record into projections. Runtime
// lowering evaluates the operand once; proof expressions are pure terms.
func wholeOkArgs(s *Small, recs map[string][][2]string) []Arg {
	if s.Ctor != "Ok" || len(s.TypeArgs) != 1 || len(s.Args) != 1 {
		return s.Args
	}
	v := s.Args[0].V
	if fields, ok := recs[s.TypeArgs[0]]; ok {
		args := make([]Arg, 0, len(fields))
		for _, f := range fields {
			args = append(args, Arg{Name: f[0], HasName: true, V: &Small{Kind: "proj", L: v, Field: f[0]}})
		}
		return args
	}
	return []Arg{{Name: "value", HasName: true, V: v}}
}

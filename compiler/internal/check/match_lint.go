package check

import (
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const codeStringMatchLadder = "CAN-CHECK-STRING-MATCH-LADDER"

type matchLint struct {
	match *ir.Match
	span  source.Span
}

// warnStringMatchLadders uses checked identities, not repeated source spelling.
// Only literal string tests on an immutable binding can become direct cases
// without changing evaluation or case priority. Retaining successful children
// lets recovery still diagnose them if an unrelated statement or arm fails.
func (c *regionChecker) warnStringMatchLadders() {
	if c.context.Warn == nil {
		return
	}
	sort.SliceStable(c.matchLints, func(i, j int) bool {
		return c.matchLints[i].span.Start < c.matchLints[j].span.Start
	})
	covered := map[*ir.Match]bool{}
	for _, candidate := range c.matchLints {
		if covered[candidate.match] {
			continue
		}
		var chain []*ir.Match
		var subject *ir.Expression
		literals := map[string]bool{}
		for current := candidate.match; current != nil; {
			binding, cases, next, ok := stringMatchDecision(current)
			if !ok || subject != nil && subject.Text != binding.Text {
				break
			}
			if subject == nil {
				subject = binding
			}
			unique := true
			for _, value := range cases {
				if literals[value] {
					unique = false
				}
				literals[value] = true
			}
			if !unique {
				// A flat match would reject overlapping cases. Never suggest
				// flattening the preceding chain across that overlap.
				chain = nil
				break
			}
			chain = append(chain, current)
			current = next
		}
		if len(chain) < 2 {
			continue
		}
		for _, match := range chain {
			covered[match] = true
		}
		file := c.context.File
		position, err := file.Position(candidate.span.Start)
		if err != nil || subject.Span.Start < 0 || subject.Span.End > file.Len() || subject.Span.Start >= subject.Span.End {
			continue
		}
		name := file.Text()[subject.Span.Start:subject.Span.End]
		c.context.Warn(Warning{
			Severity: SeverityWarning, Code: codeStringMatchLadder,
			File: file.Name(), Line: position.Line, Column: position.Column, Span: candidate.span,
			Message: fmt.Sprintf("nested string comparisons on %s; use match %s with literal arms, | for shared cases, and _ for the fallback", name, name),
		})
	}
}

// The branch that continues the search must directly contain another match:
// steps, calls or bindings between tests must remain in their original scope.
func stringMatchDecision(match *ir.Match) (*ir.Expression, []string, *ir.Match, bool) {
	if match.Call != nil || len(match.Values) != 1 || len(match.Arms) != 2 {
		return nil, nil, nil, false
	}
	var fallback *ir.Arm
	seen := map[string]bool{}
	for i := range match.Arms {
		arm := &match.Arms[i]
		if len(arm.Patterns) != 1 {
			return nil, nil, nil, false
		}
		pattern := arm.Patterns[0]
		if pattern.Kind != "literal" || !scalar(pattern.Type, "bool") || seen[pattern.Text] || pattern.Text != "true" && pattern.Text != "false" {
			return nil, nil, nil, false
		}
		seen[pattern.Text] = true
		if pattern.Text == "false" {
			fallback = arm
		}
	}
	var subject *ir.Expression
	var cases []string
	if !stringLiteralTests(match.Values[0], &subject, &cases) {
		return nil, nil, nil, false
	}
	var next *ir.Match
	if fallback.Body != nil && fallback.Body.Kind == ir.MatchCompletion {
		next = fallback.Body.Match
	} else if fallback.Value != nil && fallback.Value.Kind == ir.MatchValue {
		next = fallback.Value.Match
	}
	return subject, cases, next, true
}

func stringLiteralTests(value *ir.Expression, subject **ir.Expression, cases *[]string) bool {
	if value.Kind == ir.Binary && value.Text == "or" && len(value.Inputs) == 2 {
		return stringLiteralTests(value.Inputs[0], subject, cases) && stringLiteralTests(value.Inputs[1], subject, cases)
	}
	if value.Kind != ir.Comparison || len(value.Operators) != 1 || value.Operators[0] != "is" || len(value.Inputs) != 2 {
		return false
	}
	binding, literal := value.Inputs[0], value.Inputs[1]
	if binding.Kind == ir.Literal {
		binding, literal = literal, binding
	}
	if binding.Kind != ir.Binding || binding.Text == "" || !scalar(binding.Type, "str") || literal.Kind != ir.Literal || !scalar(literal.Type, "str") || binding.Source != "" {
		return false
	}
	if *subject != nil && (*subject).Text != binding.Text {
		return false
	}
	*subject = binding
	*cases = append(*cases, literal.Text)
	return true
}

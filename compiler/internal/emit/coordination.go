package emit

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"strings"
)

func (e *RegionEmitter) coordinationValue(node *ir.Coordination) (LoweredExpression, error) {
	lowered, err := e.coordination(node)
	if err != nil {
		return LoweredExpression{}, err
	}
	lowered.Value = "$canValue(" + lowered.Value + ") as " + TypeName(node.Result)
	return lowered, nil
}
func (e *RegionEmitter) outcomeHandler(handler *ir.OutcomeHandler) (string, error) {
	if handler == nil {
		return "async ($canOutcome: $canCompletion<unknown>) => $canOutcome", nil
	}
	saved := e.region
	e.region = handler.Region
	defer func() { e.region = saved }()
	body, err := e.match(&ir.Match{Arms: handler.Arms}, "", "$canOutcome")
	if err != nil {
		return "", err
	}
	origin := e.origin(handler.Region.Span)
	prefix := ""
	if e.SourceID != "" {
		prefix = "let $canOrigin = " + origin + ";\n"
		origin = "$canOrigin"
	}
	return "async ($canOutcome: $canCompletion<unknown>): Promise<$canCompletion<unknown>> => {\n" + prefix + "try {\n" + body + "} catch ($canCause) { return $canCaught($canCause," + origin + "); }\n}", nil
}
func (e *RegionEmitter) coordination(node *ir.Coordination) (LoweredExpression, error) {
	if node == nil || !types.Equal(node.Result, node.Result) || len(node.Entries) == 0 {
		return LoweredExpression{}, fmt.Errorf("invalid checked coordination")
	}
	var out strings.Builder
	participants, handlers := e.temp(), e.temp()
	positions := e.temp()
	fmt.Fprintf(&out, "const %s: number[][] = [];\n", positions)
	fmt.Fprintf(&out, "const %s: $canParticipant[] = [];\nconst %s: ((outcome: $canCompletion<unknown>) => Promise<$canCompletion<unknown>>)[] = [];\n", participants, handlers)
	for writtenIndex, entry := range node.Entries {
		handler, err := e.outcomeHandler(entry.Handler)
		if err != nil {
			return LoweredExpression{}, err
		}
		callback := e.temp()
		fmt.Fprintf(&out, "const %s = %s;\n", callback, handler)
		if entry.Spread != nil {
			spread, err := e.expression.Lower(entry.Spread)
			if err != nil {
				return LoweredExpression{}, err
			}
			out.WriteString(spread.Statements)
			snapshot, action := e.temp(), e.temp()
			fmt.Fprintf(&out, "const %s = Object.freeze([...%s]);\nfor (const %s of %s) {\n%s.push({captures:[%s],run:($canContext?: $canAssertionContext)=> $canInvoke(()=> $canCallContext($canContext,%s,($canContext)=>%s($canContext),$canCallableInstance(%s)),%s)});\n%s.push(%s);\n}\n", snapshot, spread.Value, action, snapshot, participants, action, quote(node.Site), action, action, e.origin(entry.Span), handlers, callback)
			fmt.Fprintf(&out, "for(let i=0;i<%s.length;i++) %s.push([%d,i]);\n", snapshot, positions, writtenIndex)
			continue
		}
		prepared, captures, err := e.prepareParticipant(entry.Call)
		if err != nil {
			return LoweredExpression{}, err
		}
		out.WriteString(prepared.Statements)
		fmt.Fprintf(&out, "%s.push({captures:[%s],run:($canContext?: $canAssertionContext)=> $canInvoke(%s,%s)});\n%s.push(%s);\n", participants, strings.Join(captures, ","), prepared.Value, e.origin(entry.Span), handlers, callback)
		fmt.Fprintf(&out, "%s.push([%d]);\n", positions, writtenIndex)
	}
	aggregate, err := e.outcomeHandler(node.Aggregate)
	if err != nil {
		return LoweredExpression{}, err
	}
	aggregateName := e.temp()
	fmt.Fprintf(&out, "const %s = %s;\n", aggregateName, aggregate)
	shared, err := e.outcomeHandler(node.Shared)
	if err != nil {
		return LoweredExpression{}, err
	}
	sharedName, selection, result := e.temp(), e.temp(), e.temp()
	fmt.Fprintf(&out, "const %s = %s;\n", sharedName, shared)
	out.WriteString(e.mark(node.Span, "coordination"))
	fmt.Fprintf(&out, "const %s = await $canCoordinateSettle(%s,%s,$canContext,%s,%s);\n", selection, quote(node.Mode), participants, quote(node.Site), positions)
	aggregateCompletion := "$canSuccess(undefined)"
	if node.AggregateType != nil {
		if e.DomainRuntime == "" {
			return LoweredExpression{}, fmt.Errorf("typed aggregate requires domain runtime")
		}
		aggregateCompletion = "$canCoordinateAggregate(outcomes," + e.DomainRuntime + "," + quote(node.AggregateType.Identity()) + "," + e.origin(node.Span) + ")"
	}
	fmt.Fprintf(&out, "const %s = await $canCoordinateHandle(%s,{each:(index,outcome)=>%s[index](outcome),shared:(_index,outcome)=>%s(outcome),allFailed:(outcomes)=>%s(%s)},%t,%s);\n", result, selection, handlers, sharedName, aggregateName, aggregateCompletion, node.Result.Kind() != types.Void, e.origin(node.Span))
	return LoweredExpression{Statements: out.String(), Value: result}, nil
}

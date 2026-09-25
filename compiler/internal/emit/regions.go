package emit

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func CompletionImports(path string) string {
	return "import {callableInstance as $canCallableInstance} from " + quote(filepath.ToSlash(filepath.Join(filepath.Dir(path), "callable.ts"))) + ";\n" + "import {callContext as $canCallContext} from " + quote(filepath.ToSlash(filepath.Join(filepath.Dir(path), "assert/context.ts"))) + ";\n" + "import { success as $canSuccess, failure as $canFailure, value as $canValue, invoke as $canInvoke, caught as $canCaught, errorType as $canErrorType, errorPayload as $canErrorPayload, type Completion as $canCompletion, type AssertionContext as $canAssertionContext } from " + quote(path) + ";\n"
}
func PatternImports(dataPath, failurePath string) string {
	return "import { recordIdentity as $canRecordIdentity } from " + quote(dataPath) + ";\nimport { isStandardFailure as $canIsStandardFailure } from " + quote(failurePath) + ";\n"
}
func TypeName(t *types.Type) string { return "$canType" + t.Identity() }

// NativeTypeDeclarations supplies strict TS annotations for the sealed graph.
// Can nominal admission is already checked in Go and branded at runtime; emitted
// structural aliases do not authorize new source assignments.
func NativeTypeDeclarations(graph []*types.Type) (string, error) {
	nodes := map[string]*types.Type{}
	var add func(*types.Type) error
	add = func(t *types.Type) error {
		if !types.Equal(t, t) {
			return fmt.Errorf("unsealed type in emission")
		}
		if nodes[t.Identity()] != nil {
			return nil
		}
		nodes[t.Identity()] = t
		children := append(t.Arguments(), t.Inputs()...)
		children = append(children, t.Errors()...)
		if t.Result() != nil {
			children = append(children, t.Result())
		}
		if t.Element() != nil {
			children = append(children, t.Element())
		}
		for _, f := range t.Fields() {
			children = append(children, f.Type)
		}
		if t.Kind() == types.Variant {
			children = append(children, t.Leaves()...)
		}
		for _, child := range children {
			if err := add(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, t := range graph {
		if err := add(t); err != nil {
			return "", err
		}
	}
	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, key := range keys {
		t := nodes[key]
		var text string
		switch t.Kind() {
		case types.Primitive:
			text = map[string]string{"int": "bigint", "float": "number", "str": "string", "bool": "boolean"}[t.Declaration()]
		case types.Void:
			text = "undefined"
		case types.Opaque:
			text = "unknown"
		case types.Array:
			text = "ReadonlyArray<" + TypeName(t.Element()) + ">"
		case types.ChoiceArm:
			text = "Readonly<{description:string;run:(probability:number,$canContext?:$canAssertionContext)=>Promise<$canCompletion<" + TypeName(t.Result()) + ">>}>"
		case types.Callable:
			var args []string
			for i, arg := range t.Inputs() {
				args = append(args, fmt.Sprintf("arg%d: %s", i, TypeName(arg)))
			}
			args = append(args, "$canContext?: $canAssertionContext")
			text = "(" + strings.Join(args, ", ") + ") => Promise<$canCompletion<" + TypeName(t.Result()) + ">>"
		case types.Variant:
			var parts []string
			for _, leaf := range t.Leaves() {
				parts = append(parts, TypeName(leaf))
			}
			text = strings.Join(parts, " | ")
		case types.Record, types.Error:
			var fields []string
			for _, f := range t.Fields() {
				fields = append(fields, "readonly "+quote(f.Name)+": "+TypeName(f.Type))
			}
			text = "{ " + strings.Join(fields, "; ") + " }"
		case types.Parameter:
			return "", fmt.Errorf("cannot emit opaque type parameter %s from an exported generic declaration: declaration-only proofs never reach runtime output", types.OpaqueParameterName(t))
		default:
			return "", fmt.Errorf("unknown emitted type")
		}
		fmt.Fprintf(&out, "type %s = %s;\n", TypeName(t), text)
	}
	return out.String(), nil
}

type RegionEmitter struct {
	Bindings  map[string]string
	Functions map[string]string
	// RuleNames maps wrapper rule region IDs to their emitted function
	// names so inherit delegates to the predecessor rule.
	RuleNames map[string]string
	// DomainRuntime is the private instance created from the checked error plan.
	DomainRuntime string
	SourceID      string
	expression    ExpressionEmitter
	region        *ir.Region
	serial        int
}

func (e *RegionEmitter) temp() string { e.serial++; return fmt.Sprintf("$canRegion%d", e.serial) }
func (e *RegionEmitter) origin(span source.Span) string {
	return fmt.Sprintf("{source:%s,start:%d,end:%d,invocation:[%s]}", quote(e.sourceID()), span.Start, span.End, quote(e.region.ID))
}
func (e *RegionEmitter) sourceID() string {
	if e.SourceID != "" {
		return e.SourceID
	}
	return e.region.Source
}
func (e *RegionEmitter) mark(span source.Span, operation string) string {
	if e.SourceID == "" {
		return ""
	}
	return mappingMark(e.SourceID, span, operation) + "$canOrigin = " + e.origin(span) + ";\n" + mappingMark(e.SourceID, span, operation)
}

// markNode attributes one lowered expression to its own defining source.
// Template substitution moves definition nodes into use regions, so their
// spans address the definition file; every other node belongs to the
// region's module. The runtime origin travels with the same source, so a
// position never pairs one file's offsets with another file.
func (e *RegionEmitter) markNode(node *ir.Expression, operation string) string {
	if e.SourceID == "" {
		return ""
	}
	source := e.SourceID
	if node.Source != "" {
		source = node.Source
	}
	origin := fmt.Sprintf("{source:%s,start:%d,end:%d,invocation:[%s]}", quote(source), node.Span.Start, node.Span.End, quote(e.region.ID))
	return mappingMark(source, node.Span, operation) + "$canOrigin = " + origin + ";\n" + mappingMark(source, node.Span, operation)
}
func (e *RegionEmitter) Function(name string, region *ir.Region) (string, error) {
	if region == nil || region.ID == "" || region.Body == nil || !types.Equal(region.Result, region.Result) || !jsBinding.MatchString(name) {
		return "", fmt.Errorf("invalid checked region")
	}
	args, err := e.configure(region)
	if err != nil {
		return "", err
	}
	body, err := e.block(region.Body)
	if err != nil {
		return "", err
	}
	origin := e.origin(region.Span)
	prefix := ""
	if e.SourceID != "" {
		prefix = mappingMark(e.SourceID, region.Span, "function") + "let $canOrigin = " + origin + ";\n"
		origin = "$canOrigin"
	}
	return fmt.Sprintf("async function %s(%s): Promise<$canCompletion<%s>> {\n%stry {\n%s} catch ($canCause) { return $canCaught($canCause, %s); }\n}\n", name, strings.Join(args, ", "), TypeName(region.Result), prefix, body, origin), nil
}
func (e *RegionEmitter) configure(region *ir.Region) ([]string, error) {
	e.region = region
	bindings := map[string]string{}
	for id, value := range e.Bindings {
		bindings[id] = value
	}
	e.expression = ExpressionEmitter{Bindings: bindings, TypeName: TypeName}
	if e.SourceID != "" {
		e.expression.Mark = func(node *ir.Expression) string { return e.markNode(node, string(node.Kind)) }
	}
	e.expression.Callable = e.callable
	e.expression.Invocation = e.invocationValue
	e.expression.Match = e.valueMatch
	e.expression.Coordination = e.coordinationValue
	e.expression.Call = func(id string, args []string) (LoweredExpression, error) {
		target, err := e.target(id)
		if err != nil {
			return LoweredExpression{}, err
		}
		boxed := e.temp()
		return LoweredExpression{Statements: fmt.Sprintf("const %s = await $canInvoke(() => %s(%s), %s);\n", boxed, target, strings.Join(append(args, "$canContext"), ", "), e.origin(region.Span)), Value: "$canValue(" + boxed + ")"}, nil
	}
	var args []string
	for i, input := range region.Inputs {
		if input.Identity == "" || !types.Equal(input.Type, input.Type) {
			return nil, fmt.Errorf("invalid region input")
		}
		param := fmt.Sprintf("$canArg%d", i)
		bindings[input.Identity] = param
		args = append(args, param+": "+TypeName(input.Type))
	}
	args = append(args, "$canContext?: $canAssertionContext")
	return args, nil
}
func (e *RegionEmitter) target(id string) (string, error) {
	if name := e.Functions[id]; name != "" {
		return name, nil
	}
	if name := e.expression.Bindings[id]; name != "" {
		return name, nil
	}
	return "", fmt.Errorf("missing emitted callable %s", id)
}
func (e *RegionEmitter) invocation(call *ir.Invocation) (LoweredExpression, error) {
	if call == nil || len(call.Steps) == 0 || !types.Equal(call.Result, call.Result) {
		return LoweredExpression{}, fmt.Errorf("invalid checked invocation")
	}
	var out strings.Builder
	result := e.temp()
	label := e.temp()
	fmt.Fprintf(&out, "let %s: $canCompletion<unknown>;\n", result)
	for _, step := range call.Steps {
		if step.SuccessBinding == "" || !types.Equal(step.Result, step.Result) {
			return LoweredExpression{}, fmt.Errorf("missing invocation binding")
		}
		local := e.temp()
		e.expression.Bindings[step.SuccessBinding] = local
		fmt.Fprintf(&out, "let %s!: %s;\n", local, TypeName(step.Result))
	}
	fmt.Fprintf(&out, "%s: { try {\n", label)
	for _, step := range call.Steps {
		var target string
		if step.Callee != nil {
			callee, err := e.expression.Lower(step.Callee)
			if err != nil {
				return LoweredExpression{}, err
			}
			out.WriteString(callee.Statements)
			target = callee.Value
		}
		for _, prepared := range step.Prepare {
			value, err := e.expression.Lower(prepared.Value)
			if err != nil {
				return LoweredExpression{}, err
			}
			out.WriteString(value.Statements)
			name := e.temp()
			e.expression.Bindings[prepared.Local.Identity] = name
			fmt.Fprintf(&out, "const %s = %s;\n", name, value.Value)
		}
		if step.Native != nil && step.Fixtures == nil {
			value, err := e.expression.Lower(step.Native)
			if err != nil {
				return LoweredExpression{}, err
			}
			out.WriteString(value.Statements)
			fmt.Fprintf(&out, "%s = %s;\n%s = $canSuccess(%s);\n", e.expression.Bindings[step.SuccessBinding], value.Value, result, e.expression.Bindings[step.SuccessBinding])
			continue
		}
		if target == "" && step.Native == nil && step.Array == nil && step.Asset == nil {
			var err error
			target, err = e.target(step.Identity)
			if err != nil {
				return LoweredExpression{}, err
			}
		}
		var args []string
		for _, argument := range step.Arguments {
			lowered, err := e.expression.Lower(argument)
			if err != nil {
				return LoweredExpression{}, err
			}
			out.WriteString(lowered.Statements)
			args = append(args, lowered.Value)
		}
		callArgs := args
		if step.SQL != nil {
			// The checked descriptor value is bound per call site. It
			// joins the invocation only: fixture matching still compares
			// the authored arguments, including the static name literal.
			callArgs = append(append([]string{}, args...), "$canSQL.declareDescriptor("+quote(step.SQL.Owner)+","+quote(step.SQL.Name)+")")
		}
		if step.FormAction != nil {
			// The frozen adapter contract and the declared handler join
			// the invocation only: fixture matching still compares the
			// authored action name and renderer callables.
			metadata, err := formActionMetadata(step.FormAction)
			if err != nil {
				return LoweredExpression{}, err
			}
			handler, err := e.target(step.FormAction.Handler)
			if err != nil {
				return LoweredExpression{}, err
			}
			callArgs = append(append([]string{}, args...), metadata, handler)
		}
		if step.JSONFetch != nil {
			// The frozen fetch contract joins the invocation only:
			// fixture matching still compares the authored action
			// name, captures and POST body.
			metadata, err := fetchActionMetadata(step.JSONFetch)
			if err != nil {
				return LoweredExpression{}, err
			}
			callArgs = append(append([]string{}, args...), metadata)
		}
		if step.Action != nil {
			// The frozen consumer contract joins the invocation only:
			// fixture matching still compares the authored value
			// operands, and the static symbol never lowers to a value.
			// HTML mounts select the three-callable form entry.
			metadata, err := actionMetadata(step.Action)
			if err != nil {
				return LoweredExpression{}, err
			}
			if step.Action.Operation == "can.std.action@1::mount" {
				target = actionMountTarget(step.Action)
			}
			callArgs = append(append([]string{}, args...), metadata)
		}
		if step.Identity == "can.std.checks@1::require" {
			// C9.2: the call-site span and invocation path travel as a
			// hidden argument into private occurrence metadata. Fixture
			// matching still compares the two authored arguments only.
			callArgs = append(append([]string{}, args...), e.origin(step.Span))
		}
		var invocation string
		if step.Array != nil {
			var err error
			invocation, err = e.arrayInvocation(step, args)
			if err != nil {
				return LoweredExpression{}, err
			}
		} else if step.Native != nil {
			value, err := e.expression.Lower(step.Native)
			if err != nil {
				return LoweredExpression{}, err
			}
			invocation = "(() => {\n" + value.Statements + "return $canSuccess(" + value.Value + ");\n})()"
		} else if step.Asset != nil {
			var err error
			invocation, err = assetInvocation(step.Asset)
			if err != nil {
				return LoweredExpression{}, err
			}
		} else {
			invocation = target + "(" + strings.Join(append(callArgs, "$canContext"), ", ") + ")"
		}
		if step.Fixtures != nil {
			var rows []string
			for _, row := range step.Fixtures.Rows {
				var prepare strings.Builder
				for _, binding := range row.Prepare {
					value, err := e.expression.Lower(binding.Value)
					if err != nil {
						return LoweredExpression{}, err
					}
					prepare.WriteString(value.Statements)
					name := e.temp()
					e.expression.Bindings[binding.Local.Identity] = name
					fmt.Fprintf(&prepare, "const %s = %s;\n", name, value.Value)
				}
				var expectedArgs []string
				for _, arg := range row.Arguments {
					value, err := e.expression.Lower(arg)
					if err != nil {
						return LoweredExpression{}, err
					}
					prepare.WriteString(value.Statements)
					expectedArgs = append(expectedArgs, value.Value)
				}
				expected, err := e.completion(row.Expected)
				if err != nil {
					return LoweredExpression{}, err
				}
				head := "{selector:" + quote(row.Selector) + ", owner:" + quote(row.Owner)
				if row.Scenario != "" {
					head += ", scenario:" + quote(row.Scenario)
				}
				arguments := ", arguments: async () => {" + prepare.String() + "return $canSuccess([" + strings.Join(expectedArgs, ",") + "]);}, expected: async () => {" + expected + "}"
				literal := head + arguments + "}"
				if row.Raw != nil {
					spec, err := RawSpec(row.Raw)
					if err != nil {
						return LoweredExpression{}, err
					}
					literal = head + arguments + ", raw:{operation:" + quote(row.Raw.Operation) + ",spec:" + spec + "}}"
				}
				rows = append(rows, literal)
			}
			invocation = "$canWithFixture($canContext," + quote(step.Fixtures.Identity) + ",[" + strings.Join(rows, ",") + "],[" + strings.Join(args, ",") + "],()=>" + invocation + "," + e.origin(step.Span) + ")"
		}
		instance := "undefined"
		if step.Native == nil && step.Array == nil && step.Asset == nil {
			instance = "$canCallableInstance(" + target + ")"
		}
		invocation = "$canCallContext($canContext," + quote(step.Site) + ",($canContext) => " + invocation + "," + instance + ")"
		out.WriteString(e.mark(step.Span, "call"))
		fmt.Fprintf(&out, "%s = await $canInvoke(() => %s, %s);\nif (%s.kind !== 'ok') break %s;\n%s = $canValue(%s) as %s;\n", result, invocation, e.origin(step.Span), result, label, e.expression.Bindings[step.SuccessBinding], result, TypeName(step.Result))
	}
	if call.Result.Kind() == types.Void {
		fmt.Fprintf(&out, "%s = $canSuccess(undefined);\n", result)
	}
	fmt.Fprintf(&out, "} catch ($canCause) { %s = $canCaught($canCause, %s); } }\n", result, func() string {
		if e.SourceID != "" {
			return "$canOrigin"
		}
		return e.origin(call.Span)
	}())
	return LoweredExpression{out.String(), result}, nil
}
func (e *RegionEmitter) invocationValue(call *ir.Invocation) (LoweredExpression, error) {
	if call == nil || len(call.Errors) != 0 || call.Result.Kind() == types.Void {
		return LoweredExpression{}, fmt.Errorf("unchecked completion used as a value")
	}
	lowered, err := e.invocation(call)
	if err != nil {
		return LoweredExpression{}, err
	}
	// A failed operand must escape before another operand is prepared, even
	// when the consumer (for example a wildcard pattern) never reads its value.
	value := e.temp()
	lowered.Statements += fmt.Sprintf("const %s = $canValue(%s) as %s;\n", value, lowered.Value, TypeName(call.Result))
	lowered.Value = value
	return lowered, nil
}
func (e *RegionEmitter) block(block *ir.Block) (string, error) {
	if block == nil {
		return "", fmt.Errorf("missing checked block")
	}
	var out strings.Builder
	for _, step := range block.Steps {
		if step.Local != nil {
			value, err := e.expression.Lower(step.Value)
			if err != nil {
				return "", err
			}
			out.WriteString(value.Statements)
			name := e.temp()
			e.expression.Bindings[step.Local.Identity] = name
			fmt.Fprintf(&out, "const %s: %s = %s;\n", name, TypeName(step.Local.Type), value.Value)
		} else if step.Coordination != nil {
			lowered, err := e.coordination(step.Coordination)
			if err != nil {
				return "", err
			}
			out.WriteString(lowered.Statements)
			fmt.Fprintf(&out, "$canValue(%s);\n", lowered.Value)
		} else if step.Call != nil {
			call, err := e.invocation(step.Call)
			if err != nil {
				return "", err
			}
			out.WriteString(call.Statements)
			fmt.Fprintf(&out, "$canValue(%s);\n", call.Value)
		} else {
			return "", fmt.Errorf("invalid checked statement")
		}
	}
	terminal, err := e.completion(block.Terminal)
	if err != nil {
		return "", err
	}
	out.WriteString(terminal)
	return out.String(), nil
}
func (e *RegionEmitter) completion(node *ir.Completion) (string, error) {
	if node == nil || node.RegionID != e.region.ID {
		return "", fmt.Errorf("completion belongs to a different region")
	}
	switch node.Kind {
	case ir.SuccessCompletion:
		if node.Value == nil {
			return "return $canSuccess(undefined);\n", nil
		}
		value, err := e.expression.Lower(node.Value)
		if err != nil {
			return "", err
		}
		return value.Statements + "return $canSuccess(" + value.Value + ");\n", nil
	case ir.DomainCompletion:
		if e.DomainRuntime == "" {
			return "", fmt.Errorf("missing checked domain runtime")
		}
		value, err := e.expression.Lower(node.Value)
		if err != nil {
			return "", err
		}
		return value.Statements + e.mark(node.Span, "domain") + "return $canFailure(" + e.DomainRuntime + ".create(" + quote(node.Value.Type.Identity()) + ", " + value.Value + ", " + e.origin(node.Span) + "));\n", nil
	case ir.RelayCompletion:
		call, err := e.invocation(node.Call)
		if err != nil {
			return "", err
		}
		return call.Statements + "return " + call.Value + " as $canCompletion<" + TypeName(e.region.Result) + ">;\n", nil
	case ir.DoCompletion:
		return e.block(node.Block)
	case ir.MatchCompletion:
		return e.match(node.Match, "")
	case ir.InheritCompletion:
		if node.Inherit == "" {
			return "return $canOriginal;\n", nil
		}
		target := e.RuleNames[node.Inherit]
		if target == "" {
			return "", fmt.Errorf("missing inherit target %s", node.Inherit)
		}
		var args []string
		for _, input := range e.region.Inputs {
			name := e.expression.Bindings[input.Identity]
			if name == "" {
				return "", fmt.Errorf("missing inherit argument %s", input.Identity)
			}
			args = append(args, name)
		}
		args = append(args, "$canOriginal", "$canContext")
		return "return await " + target + "(" + strings.Join(args, ", ") + ");\n", nil
	default:
		return "", fmt.Errorf("unknown completion kind")
	}
}
func (e *RegionEmitter) valueMatch(match *ir.Match) (LoweredExpression, error) {
	if match == nil || match.ValueResult == nil || match.Call != nil {
		return LoweredExpression{}, fmt.Errorf("not an ordinary value match")
	}
	result := e.temp()
	code, err := e.match(match, result)
	if err != nil {
		return LoweredExpression{}, err
	}
	return LoweredExpression{fmt.Sprintf("let %s!: %s;\n", result, TypeName(match.ValueResult)) + code, result}, nil
}
func (e *RegionEmitter) match(match *ir.Match, result string, supplied ...string) (string, error) {
	if match == nil {
		return "", fmt.Errorf("missing match")
	}
	outcomeMatch := match.Call != nil || len(supplied) > 0
	var out strings.Builder
	var values []string
	var completion string
	var patternDeclarations strings.Builder
	if len(supplied) > 0 {
		completion = supplied[0]
	} else if match.Call != nil {
		call, err := e.invocation(match.Call)
		if err != nil {
			return "", err
		}
		out.WriteString(call.Statements)
		completion = call.Value
	} else {
		for _, v := range match.Values {
			value, err := e.expression.Lower(v)
			if err != nil {
				return "", err
			}
			out.WriteString(value.Statements)
			values = append(values, value.Value)
		}
	}
	if outcomeMatch && len(match.Arms) == 0 {
		return "return " + completion + " as $canCompletion<" + TypeName(e.region.Result) + ">;\n", nil
	}
	for i, arm := range match.Arms {
		var condition, setup string
		if outcomeMatch {
			switch arm.Outcome {
			case "ok":
				condition = completion + ".kind === 'ok'"
			case "standard":
				condition = completion + ".kind === 'standard'"
			case "domain":
				condition = completion + ".kind === 'domain' && $canErrorType(" + completion + ") === " + quote(arm.Error.Identity())
			default:
				return "", fmt.Errorf("invalid outcome arm")
			}
			if arm.Binding != nil {
				local := e.temp()
				e.expression.Bindings[arm.Binding.Identity] = local
				value := "$canValue(" + completion + ")"
				if arm.Outcome == "domain" {
					value = "$canErrorPayload(" + completion + ")"
				}
				if arm.Outcome == "standard" {
					// C9.1: bind the opaque snapshot itself. Projections
					// read kind/message/occurrence_id through adapters.
					value = completion + ".value"
				}
				setup = fmt.Sprintf("const %s = %s as %s;\n", local, value, TypeName(arm.Binding.Type))
			}
		} else {
			var conditions []string
			for j, p := range arm.Patterns {
				cond, err := e.pattern(p, values[j], &patternDeclarations)
				if err != nil {
					return "", err
				}
				conditions = append(conditions, "("+cond+")")
			}
			condition = strings.Join(conditions, " && ")
			if i > 0 {
				out.WriteString("else ")
			}
		}
		if outcomeMatch && i > 0 {
			out.WriteString("else ")
		}
		fmt.Fprintf(&out, "if (%s) {\n%s", condition, setup)
		if arm.Forward {
			fmt.Fprintf(&out, "return %s as $canCompletion<%s>;\n", completion, TypeName(e.region.Result))
		} else if result != "" {
			value, err := e.expression.Lower(arm.Value)
			if err != nil {
				return "", err
			}
			out.WriteString(value.Statements)
			fmt.Fprintf(&out, "%s = %s;\n", result, value.Value)
		} else {
			body, err := e.completion(arm.Body)
			if err != nil {
				return "", err
			}
			out.WriteString(body)
		}
		out.WriteString("}\n")
	}
	if outcomeMatch {
		fmt.Fprintf(&out, "else { return %s as $canCompletion<%s>; }\n", completion, TypeName(e.region.Result))
	} else {
		out.WriteString("else { throw new Error('checked match was not exhaustive'); }\n")
	}
	return patternDeclarations.String() + out.String(), nil
}
func (e *RegionEmitter) pattern(p *ir.Pattern, value string, declarations *strings.Builder) (string, error) {
	var conditions []string
	assign := func(local *ir.Local, expression string) {
		if local == nil {
			return
		}
		name := e.expression.Bindings[local.Identity]
		if name == "" {
			name = e.temp()
			e.expression.Bindings[local.Identity] = name
			fmt.Fprintf(declarations, "let %s!: %s;\n", name, TypeName(local.Type))
		}
		conditions = append(conditions, "("+name+" = "+expression+" as "+TypeName(local.Type)+", true)")
	}
	switch p.Kind {
	case "any":
		conditions = append(conditions, "true")
	case "literal":
		literal := p.Text
		switch p.Type.Declaration() {
		case "int":
			literal += "n"
		case "str":
			literal = quote(literal)
		case "float":
			literal = "Number(" + quote(literal) + ")"
		}
		if p.Type.Declaration() == "float" {
			conditions = append(conditions, "Object.is("+value+", "+literal+")")
		} else {
			conditions = append(conditions, value+" === "+literal)
		}
	case "range":
		conditions = append(conditions, "("+value+" >= "+p.Text+"n && "+value+" <= "+p.Upper+"n)")
	case "nominal":
		if p.Type.Kind() == types.Opaque {
			conditions = append(conditions, "$canIsStandardFailure("+value+")")
		} else {
			conditions = append(conditions, "$canRecordIdentity("+value+") === "+quote(p.Text))
		}
		for i, child := range p.Children {
			part, err := e.pattern(child, "("+value+" as "+TypeName(p.Type)+")["+quote(p.Fields[i])+"]", declarations)
			if err != nil {
				return "", err
			}
			conditions = append(conditions, "("+part+")")
		}
	case "array":
		operator := " === "
		if p.Rest != nil {
			operator = " >= "
		}
		conditions = append(conditions, fmt.Sprintf("%s.length%s%d", value, operator, len(p.Children)))
		for i, child := range p.Children {
			part, err := e.pattern(child, fmt.Sprintf("%s[%d]!", value, i), declarations)
			if err != nil {
				return "", err
			}
			conditions = append(conditions, "("+part+")")
		}
		if p.Rest != nil {
			assign(p.Rest, fmt.Sprintf("Object.freeze(%s.slice(%d))", value, len(p.Children)))
		}
	case "or":
		var choices []string
		for _, child := range p.Children {
			part, err := e.pattern(child, value, declarations)
			if err != nil {
				return "", err
			}
			choices = append(choices, "("+part+")")
		}
		conditions = append(conditions, "("+strings.Join(choices, " || ")+")")
	default:
		return "", fmt.Errorf("unknown pattern kind")
	}
	assign(p.Binding, value)
	assign(p.Narrow, value)
	return strings.Join(conditions, " && "), nil
}

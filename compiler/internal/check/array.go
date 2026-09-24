package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"regexp"
	"strings"
)

type callbackHint struct {
	inputs []*types.Type
	result *types.Type
}

func callbackConstraints(d *syntax.FunctionDecl, inputs []*types.Type, result *types.Type) ([]typeConstraint, error) {
	var seeds []typeConstraint
	index := 0
	for _, input := range d.Inputs {
		if input.Near {
			continue
		}
		if index >= len(inputs) {
			return nil, fmt.Errorf("array callback input arity mismatch")
		}
		node := input.Type
		if input.Variadic {
			node = &syntax.ArrayType{Element: node}
		}
		// A nil entry means no equality is known for this argument yet. It is
		// never installed as an inference type or a checked callable input.
		if inputs[index] != nil {
			seeds = append(seeds, typeConstraint{node, inputs[index]})
		}
		index++
	}
	if index != len(inputs) {
		return nil, fmt.Errorf("array callback input arity mismatch")
	}
	if result != nil {
		seeds = append(seeds, typeConstraint{d.Result, result})
	}
	return seeds, nil
}
func (c *programChecker) inferCallback(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, inputs []*types.Type, result *types.Type, e *Expressions) (ValueBinding, bool, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, false, nil
	}
	if op := collectionOperation(symbol.ID); op != nil {
		return c.referenceCollection(op, nil, inputs, result)
	}
	if op := browserStateOperation(symbol.ID); op != nil {
		return c.referenceBrowserState(op, nil, inputs, result)
	}
	d, ok := symbol.Declaration.(*syntax.FunctionDecl)
	if !ok || len(symbol.Parameters) == 0 {
		return ValueBinding{}, false, nil
	}
	if d.Receiver != nil {
		return ValueBinding{}, true, fmt.Errorf("method callback requires receiver")
	}
	seeds, err := callbackConstraints(d, inputs, result)
	if err != nil {
		return ValueBinding{}, true, err
	}
	return c.inferDeclaredReference(symbol, d, nil, e, applicationSite(file, name.Span.Start), seeds...)
}

var catalogueIdentifier = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9]*`)

func catalogueTypeNode(text string, parameters []string) (syntax.TypeNode, error) {
	aliases := map[string]string{}
	for _, name := range parameters {
		aliases[name] = "catalogue_" + strings.ToLower(name)
	}
	text = catalogueIdentifier.ReplaceAllStringFunc(text, func(name string) string {
		if alias, ok := aliases[name]; ok {
			return alias
		}
		return name
	})
	src, err := source.New("can:catalogue", text)
	if err != nil {
		return nil, err
	}
	node, ds := syntax.ParseType(src)
	if len(ds) > 0 {
		return nil, fmt.Errorf("invalid catalogue type: %v", ds)
	}
	return node, nil
}
func (c *programChecker) catalogueType(text string, parameters map[string]*types.Type) (*types.Type, error) {
	bound := map[string]*types.Type{}
	var names []string
	for name, typ := range parameters {
		names = append(names, name)
		bound["catalogue_"+strings.ToLower(name)] = typ
	}
	node, err := catalogueTypeNode(text, names)
	if err != nil {
		return nil, err
	}
	return c.specializer.Resolve(&resolve.File{Scope: c.world.Prelude, Imports: c.world.Packages}, node, bound, true)
}
func arrayMethod(name string) bool {
	switch name {
	case "map", "filter", "for_each", "fold", "find", "some", "every", "sort_by", "concat", "to_reversed":
		return true
	}
	return false
}
func (c *regionChecker) arrayStep(receiver *ir.Expression, name, site string, args []syntax.Argument, span source.Span, scope bodyScope, expected *types.Type) (ir.InvocationStep, error) {
	fail := func(message string) (ir.InvocationStep, error) {
		return ir.InvocationStep{}, fmt.Errorf("array.%s: %s", name, message)
	}
	inventory := catalogue.Builtin().Inventory()
	operationName := "array." + name
	if name == "append" {
		operationName = name
	}
	originalArgs := args
	args, err := fixedArgumentElements(args)
	if err != nil {
		return ir.InvocationStep{}, err
	}
	op, err := catalogue.Builtin().Operation(operationName, inventory.TargetID, inventory.Revision)
	if err != nil {
		return ir.InvocationStep{}, err
	}
	if receiver.Type.Kind() != types.Array || (op.Receiver != "T[]" && name != "append") || op.Lowering.Task != "I21" {
		return fail("invalid catalogue receiver")
	}
	arity := len(op.Inputs)
	if name == "append" {
		arity--
	}
	if len(args) != arity {
		return fail("argument arity mismatch")
	}
	for _, arg := range args {
		if arg.Spread || arg.Group != nil {
			return fail("fixed ordinary arguments required")
		}
	}
	if c.context.CatalogueType == nil {
		return fail("missing catalogue type resolver")
	}
	e := c.expressions(scope)
	parameters := map[string]*types.Type{"T": receiver.Type.Element()}
	inputs := []*ir.Expression{receiver}
	resolveType := func(text string) (*types.Type, error) { return c.context.CatalogueType(text, parameters) }
	if name == "concat" || name == "append" {
		argumentType := receiver.Type
		if name == "append" {
			argumentType = receiver.Type.Element()
		}
		value, err := e.Check(args[0].Value, argumentType)
		if err != nil {
			return ir.InvocationStep{}, err
		}
		if !types.Equal(value.Type, argumentType) {
			return fail("array element types must be equal")
		}
		inputs = append(inputs, value)
	} else if len(op.Callbacks) > 0 {
		callbackIndex := 0
		callbackInputs := []*types.Type{receiver.Type.Element()}
		var callbackResult *types.Type
		if name == "fold" {
			initial, err := e.Check(args[0].Value, expected)
			if err != nil {
				return ir.InvocationStep{}, err
			}
			parameters["U"] = initial.Type
			inputs = append(inputs, initial)
			callbackInputs = []*types.Type{initial.Type, receiver.Type.Element()}
			callbackResult = initial.Type
			callbackIndex = 1
		} else if name == "map" {
			if expected != nil && expected.Kind() == types.Array {
				callbackResult = expected.Element()
			}
		} else if name != "sort_by" {
			callbackResult, err = resolveType(op.Callbacks[0].Result)
			if err != nil {
				return ir.InvocationStep{}, err
			}
		}
		node := args[callbackIndex].Value
		for {
			group, ok := node.(*syntax.GroupExpr)
			if !ok {
				break
			}
			node = group.Value
		}
		var action *ir.Expression
		if reference, ok := node.(*syntax.ReferenceExpr); ok {
			action, err = c.reference(reference, scope, nil, callbackHint{callbackInputs, callbackResult})
		} else {
			action, err = e.Check(node, nil)
		}
		if err != nil {
			return ir.InvocationStep{}, err
		}
		inputs = append(inputs, action)
	}
	step, err := c.arrayContract(name, site, receiver.Type, inputs[1:], span)
	if err != nil {
		return ir.InvocationStep{}, err
	}
	step.Prepare, step.Arguments, err = c.arguments(e, ValueBinding{Identity: step.Identity, Type: step.Contract}, originalArgs, receiver)
	if err != nil {
		return ir.InvocationStep{}, err
	}
	return step, nil
}

// arrayContract validates concrete operands independently of source inference so
// direct calls and first-class catalogue references share the same exact ABI.
func (c *regionChecker) arrayContract(name, site string, receiver *types.Type, arguments []*ir.Expression, span source.Span) (ir.InvocationStep, error) {
	fail := func(message string) (ir.InvocationStep, error) {
		return ir.InvocationStep{}, fmt.Errorf("array.%s: %s", name, message)
	}
	inventory := catalogue.Builtin().Inventory()
	operationName := "array." + name
	if name == "append" {
		operationName = name
	}
	op, err := catalogue.Builtin().Operation(operationName, inventory.TargetID, inventory.Revision)
	if err != nil {
		return ir.InvocationStep{}, err
	}
	arity := len(op.Inputs)
	if name == "append" {
		arity--
	}
	if receiver == nil || receiver.Kind() != types.Array || len(arguments) != arity || op.Lowering.Task != "I21" {
		return fail("invalid receiver or argument arity")
	}
	for _, argument := range arguments {
		if argument == nil || !types.Equal(argument.Type, argument.Type) {
			return fail("missing concrete argument type")
		}
	}
	parameters := map[string]*types.Type{"T": receiver.Element()}
	var errors []*types.Type
	if name == "slice" {
		for _, argument := range arguments {
			if !scalar(argument.Type, "int") {
				return fail("slice bounds must be int")
			}
		}
	} else if name == "append" || name == "concat" {
		want := receiver
		if name == "append" {
			want = receiver.Element()
		}
		if !types.Equal(arguments[0].Type, want) {
			return fail("array element types must be equal")
		}
	} else if len(op.Callbacks) > 0 {
		callbackIndex := 0
		inputs := []*types.Type{receiver.Element()}
		if name == "fold" {
			parameters["U"] = arguments[0].Type
			inputs = []*types.Type{arguments[0].Type, receiver.Element()}
			callbackIndex = 1
		}
		action := arguments[callbackIndex].Type
		if action.Kind() != types.Callable || len(action.Inputs()) != len(inputs) {
			return fail("callback input arity mismatch")
		}
		for i, input := range inputs {
			if !types.Equal(input, action.Inputs()[i]) {
				return fail("callback inputs must equal element and accumulator types")
			}
		}
		switch name {
		case "map":
			if action.Result().Kind() == types.Void {
				return fail("map callback must return data")
			}
			parameters["U"] = action.Result()
		case "sort_by":
			key := action.Result()
			if !(scalar(key, "int") || scalar(key, "float") || scalar(key, "str") || scalar(key, "bool")) {
				return fail("sort key must be int, float, str or bool")
			}
			parameters["K"] = key
		default:
			result, err := c.context.CatalogueType(op.Callbacks[0].Result, parameters)
			if err != nil {
				return ir.InvocationStep{}, err
			}
			if !types.Equal(result, action.Result()) {
				return fail("callback result must equal required type")
			}
		}
		errors = action.Errors()
	}
	result, err := c.context.CatalogueType(op.Result, parameters)
	if err != nil {
		return ir.InvocationStep{}, err
	}
	step := ir.InvocationStep{Site: site, Array: &ir.ArrayOperation{Name: name}, Receiver: name != "append", Identity: op.Identity, Span: span, Result: result, Errors: errors, SuccessBinding: c.identity("array")}
	if name == "find" {
		for _, leaf := range result.Leaves() {
			switch leaf.Declaration() {
			case "can.std.option@1::none":
				step.Array.None = leaf
			case "can.std.option@1::some":
				step.Array.Some = leaf
			}
		}
		if step.Array.None == nil || step.Array.Some == nil {
			return fail("missing sealed option identities")
		}
	}
	inputTypes := []*types.Type{receiver}
	for _, argument := range arguments {
		inputTypes = append(inputTypes, argument.Type)
	}
	step.Contract, err = types.CallableOfChecked(result, inputTypes, errors)
	return step, err
}

func (c *regionChecker) arrayReference(n *syntax.ReferenceExpr, name string, receiver *ir.Expression, expected *types.Type, hints []callbackHint) (*ir.Expression, error) {
	var inputs []*types.Type
	if expected != nil && expected.Kind() == types.Callable {
		inputs = expected.Inputs()
	}
	if len(hints) > 0 {
		inputs = hints[0].inputs
	}
	if name == "append" {
		var receiverType *types.Type
		if len(n.Types) > 0 {
			if len(n.Types) != 1 {
				return nil, fmt.Errorf("append reference requires one type argument")
			}
			element, err := c.context.Type(n.Types[0], false)
			if err != nil {
				return nil, err
			}
			receiverType, err = types.ArrayOfChecked(element)
			if err != nil {
				return nil, err
			}
			if inputs == nil {
				inputs = []*types.Type{receiverType, element}
			}
		} else if len(inputs) == 2 {
			receiverType = inputs[0]
		}
		if receiverType != nil && receiverType.Kind() == types.Array && len(inputs) == 2 {
			inputs = append([]*types.Type(nil), inputs...)
			if inputs[0] == nil {
				inputs[0] = receiverType
			}
			if inputs[1] == nil {
				inputs[1] = receiverType.Element()
			}
		}
		if receiverType == nil || receiverType.Kind() != types.Array || len(inputs) != 2 || !types.Equal(inputs[0], receiverType) {
			return nil, fmt.Errorf("append reference requires a concrete array and element contract")
		}
		receiver = &ir.Expression{Type: receiverType}
		inputs = inputs[1:]
	} else {
		if len(n.Types) != 0 {
			return nil, fmt.Errorf("array reference does not take authored type arguments")
		}
		if inputs == nil {
			switch name {
			case "slice":
				integer, err := c.context.CatalogueType("int", nil)
				if err != nil {
					return nil, err
				}
				inputs = []*types.Type{integer, integer}
			case "concat":
				inputs = []*types.Type{receiver.Type}
			case "to_reversed":
				inputs = []*types.Type{}
			default:
				return nil, fmt.Errorf("array callback reference requires a concrete callable input contract")
			}
		}
	}
	// Partial callback hints carry absent equalities, not concrete types. A
	// bound concat already determines its sole input from the captured receiver.
	if name == "concat" && len(inputs) == 1 && inputs[0] == nil {
		inputs = []*types.Type{receiver.Type}
	}
	var arguments []*ir.Expression
	for _, typ := range inputs {
		if !types.Equal(typ, typ) {
			return nil, fmt.Errorf("array callback reference requires a concrete callable input contract")
		}

		arguments = append(arguments, &ir.Expression{Type: typ})
	}
	site, err := c.lexicalSite("callable", n.Span)
	if err != nil {
		return nil, err
	}
	step, err := c.arrayContract(name, site, receiver.Type, arguments, n.Span)
	if err != nil {
		return nil, err
	}
	declaration := &ir.Callable{Site: site, Target: step.Identity, Contract: step.Contract, Array: step.Array}
	out := &ir.Expression{Kind: ir.CallableValue, Span: n.Span, Callable: declaration, Type: step.Contract}
	if name != "append" {
		declaration.Positions = []int{0}
		out.Inputs = []*ir.Expression{receiver}
		out.Type, err = types.CallableOfChecked(step.Result, inputs, step.Errors)
		if err != nil {
			return nil, err
		}
	}
	declaration.ResourceCaptures = ir.ResourceCaptureIndices(out.Inputs)
	c.uses.Captures[n] = []string{}
	return out, nil
}

// Empty array literals contribute no element evidence. A concrete result or
// callback input may supply it without evaluating any authored expression.
func (c *regionChecker) arrayReceiver(node syntax.Expr, name string, args []syntax.Argument, scope bodyScope, expected *types.Type) (*ir.Expression, error) {
	e := c.expressions(scope)
	receiver, original := e.Check(node, nil)
	if original == nil {
		return receiver, nil
	}
	if size, known := literalSpreadLength(node); !known || size != 0 {
		return nil, original
	}
	args, err := fixedArgumentElements(args)
	if err != nil {
		return nil, err
	}
	var element *types.Type
	if expected != nil && expected.Kind() == types.Array && name != "map" && name != "fold" {
		element = expected.Element()
	}
	if element == nil && name == "concat" && len(args) == 1 {
		right, err := e.Check(args[0].Value, nil)
		if err == nil && right.Type.Kind() == types.Array {
			element = right.Type.Element()
		}
	}
	callbackIndex := 0
	if name == "fold" {
		callbackIndex = 1
	}
	if element == nil && name != "concat" && name != "to_reversed" && len(args) > callbackIndex && args[callbackIndex].Value != nil {
		node := args[callbackIndex].Value
		for {
			group, ok := node.(*syntax.GroupExpr)
			if !ok {
				break
			}
			node = group.Value
		}
		var hint *callbackHint
		if name == "map" && expected != nil && expected.Kind() == types.Array {
			hint = &callbackHint{inputs: []*types.Type{nil}, result: expected.Element()}
		} else if name == "fold" {
			initial, err := e.Check(args[0].Value, expected)
			if err == nil {
				hint = &callbackHint{inputs: []*types.Type{initial.Type, nil}, result: initial.Type}
			}
		}
		var action *ir.Expression
		var err error
		if reference, ok := node.(*syntax.ReferenceExpr); ok && hint != nil {
			action, err = c.reference(reference, scope, nil, *hint)
		} else {
			action, err = e.Check(node, nil)
		}
		if err == nil && action.Type.Kind() == types.Callable {
			inputs := action.Type.Inputs()
			if len(inputs) == callbackIndex+1 {
				element = inputs[len(inputs)-1]
			}
		}
	}
	if element == nil {
		return nil, original
	}
	wanted, err := types.ArrayOfChecked(element)
	if err != nil {
		return nil, err
	}
	return e.Check(node, wanted)
}

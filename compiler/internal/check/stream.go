package check

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type StreamSpecialization struct {
	Operation string
	Contract  *types.Type
}

// isStreamScopeRequest reports whether the type is an ingress-only stream
// handle. No Can expression constructs a reader or writer, so assertion
// rows omit handle inputs while the harness splices its scope token; any
// handle operation the row does not when-supply fails at the denied live
// boundary, exactly like pool and transaction handles.
func isStreamScopeRequest(typ *types.Type) bool {
	if typ == nil || typ.Kind() != types.Opaque {
		return false
	}
	switch typ.Declaration() {
	case "can.std.stream@1::reader", "can.std.stream@1::writer":
		return true
	}
	return false
}

// streamOperation covers the generic pull-stream operations so they resolve
// per call site: T binds from the reader argument or explicit type
// arguments. Concrete stream operations pre-bind like any other call.
func streamOperation(identity string) *catalogue.Operation {
	switch identity {
	case "can.std.stream@1::read_many", "can.std.stream@1::close_reader", "can.std.stream@1::cancel_reader":
	default:
		return nil
	}
	for _, op := range catalogue.Builtin().Inventory().Operations {
		if op.Identity == identity {
			return &op
		}
	}
	return nil
}

func streamGenericOperation(identity string) bool {
	return streamOperation(identity) != nil
}

func (c *programChecker) streamSignature(op *catalogue.Operation) (*resolve.File, []string, *syntax.CallableType, error) {
	file := &resolve.File{Scope: c.world.Prelude, Imports: c.world.Packages}
	var original, names []string
	for _, p := range op.Parameters {
		original = append(original, p.Name)
		names = append(names, "catalogue_"+strings.ToLower(p.Name))
	}
	signature := &syntax.CallableType{}
	var err error
	signature.Result, err = catalogueTypeNode(op.Result, original)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, input := range op.Inputs {
		node, e := catalogueTypeNode(input.Type, original)
		if e != nil {
			return nil, nil, nil, e
		}
		signature.Inputs = append(signature.Inputs, node)
	}
	for _, failure := range op.Emits {
		node, e := catalogueTypeNode(failure, original)
		if e != nil {
			return nil, nil, nil, e
		}
		signature.Errors.Types = append(signature.Errors.Types, node)
	}
	return file, names, signature, nil
}

func (c *programChecker) instantiateStream(op *catalogue.Operation, args []*types.Type) (ValueBinding, error) {
	if len(args) != len(op.Parameters) {
		return ValueBinding{}, fmt.Errorf("stream type argument count mismatch")
	}
	parameters := map[string]*types.Type{}
	for i, p := range op.Parameters {
		parameters[p.Name] = args[i]
	}
	key, err := types.SpecializationKey(op.Identity, args)
	if err != nil {
		return ValueBinding{}, err
	}
	if old := c.program.Streams[key]; old != nil {
		return ValueBinding{Identity: key, Type: old.Contract}, nil
	}
	result, err := c.catalogueType(op.Result, parameters)
	if err != nil {
		return ValueBinding{}, err
	}
	var inputs, errors []*types.Type
	for _, input := range op.Inputs {
		typ, e := c.catalogueType(input.Type, parameters)
		if e != nil {
			return ValueBinding{}, e
		}
		inputs = append(inputs, typ)
	}
	for _, failure := range op.Emits {
		typ, e := c.catalogueType(failure, parameters)
		if e != nil {
			return ValueBinding{}, e
		}
		errors = append(errors, typ)
	}
	contract, err := types.CallableOfChecked(result, inputs, errors)
	if err != nil {
		return ValueBinding{}, err
	}
	if c.program.Streams == nil {
		c.program.Streams = map[string]*StreamSpecialization{}
	}
	c.program.Streams[key] = &StreamSpecialization{Operation: op.Identity, Contract: contract}
	c.program.Intrinsics[key] = contract
	c.bindings[key] = contract
	descriptor := CallableDeclaration{Kind: resolve.Function, Contract: contract}
	for _, input := range op.Inputs {
		descriptor.Names = append(descriptor.Names, input.Name)
		descriptor.Near = append(descriptor.Near, false)
	}
	c.callables[key] = descriptor
	return ValueBinding{Identity: key, Type: contract}, nil
}

func (c *programChecker) inferStream(op *catalogue.Operation, args []syntax.Argument, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
	file, names, signature, err := c.streamSignature(op)
	if err != nil {
		return ValueBinding{}, true, err
	}
	var inputs []syntax.Input
	for _, node := range signature.Inputs {
		inputs = append(inputs, syntax.Input{Field: syntax.Field{Type: node}})
	}
	constraints, err := genericArguments(inputs, args)
	if err != nil {
		return ValueBinding{}, true, err
	}
	arguments, err := c.inferArguments(file, names, signature.Result, expected, constraints, e)
	if err != nil {
		return ValueBinding{}, true, err
	}
	binding, err := c.instantiateStream(op, arguments)
	return binding, true, err
}

func (c *programChecker) referenceStream(op *catalogue.Operation, expected *types.Type, inputs []*types.Type, result *types.Type) (ValueBinding, bool, error) {
	file, names, signature, err := c.streamSignature(op)
	if err != nil {
		return ValueBinding{}, true, err
	}
	solver, err := types.NewInference(names)
	if err != nil {
		return ValueBinding{}, true, err
	}
	constrain := func(node syntax.TypeNode, actual *types.Type) error {
		if actual == nil {
			return nil
		}
		pattern, err := types.Pattern(file, node, names)
		if err != nil {
			return err
		}
		return solver.Constrain(pattern, actual)
	}
	if expected != nil {
		err = constrain(signature, expected)
	} else {
		if len(inputs) != len(signature.Inputs) {
			return ValueBinding{}, true, fmt.Errorf("stream callback arity mismatch")
		}
		for i, input := range inputs {
			if err = constrain(signature.Inputs[i], input); err != nil {
				break
			}
		}
		if err == nil {
			err = constrain(signature.Result, result)
		}
	}
	if err != nil {
		return ValueBinding{}, true, err
	}
	arguments, err := solver.Arguments()
	if err != nil {
		return ValueBinding{}, true, err
	}
	binding, err := c.instantiateStream(op, arguments)
	return binding, true, err
}

func (c *programChecker) specializeStream(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, err
	}
	op := streamOperation(symbol.ID)
	if op == nil {
		return ValueBinding{}, fmt.Errorf("stream specialization of %s is unavailable", symbol.ID)
	}
	arguments := make([]*types.Type, len(args))
	for i, arg := range args {
		arguments[i], err = c.annotation(file, arg, false)
		if err != nil {
			return ValueBinding{}, err
		}
	}
	return c.instantiateStream(op, arguments)
}

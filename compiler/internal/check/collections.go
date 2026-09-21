package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"strings"
)

type CollectionSpecialization struct {
	Operation                   string
	Collection, Entry, Contract *types.Type
}

func collectionOperation(identity string) *catalogue.Operation {
	for _, op := range catalogue.Builtin().Inventory().Operations {
		if op.Identity == identity && op.Lowering.Task == "I25" {
			return &op
		}
	}
	return nil
}
func (c *programChecker) collectionSignature(op *catalogue.Operation) (*resolve.File, []string, *syntax.CallableType, error) {
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
func (c *programChecker) instantiateCollection(op *catalogue.Operation, args []*types.Type) (ValueBinding, error) {
	if len(args) != len(op.Parameters) {
		return ValueBinding{}, fmt.Errorf("collection type argument count mismatch")
	}
	parameters := map[string]*types.Type{}
	for i, p := range op.Parameters {
		parameters[p.Name] = args[i]
	}
	kind := "collections::set<K>"
	for _, input := range op.Inputs {
		if strings.HasPrefix(input.Type, "collections::map<") {
			kind = "collections::map<K,V>"
		}
	}
	if op.Name == "collections::empty_map" {
		kind = "collections::map<K,V>"
	}
	collection, err := c.catalogueType(kind, parameters)
	if err != nil {
		return ValueBinding{}, err
	}
	key, err := types.SpecializationKey(op.Identity, args)
	if err != nil {
		return ValueBinding{}, err
	}
	if old := c.program.Collections[key]; old != nil {
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
	special := &CollectionSpecialization{Operation: op.Identity, Collection: collection, Contract: contract}
	if kind == "collections::map<K,V>" {
		special.Entry, err = c.catalogueType("collections::entry<K,V>", parameters)
		if err != nil {
			return ValueBinding{}, err
		}
	}
	if c.program.Collections == nil {
		c.program.Collections = map[string]*CollectionSpecialization{}
	}
	c.program.Collections[key] = special
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
func (c *programChecker) inferCollection(op *catalogue.Operation, args []syntax.Argument, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
	file, names, signature, err := c.collectionSignature(op)
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
	binding, err := c.instantiateCollection(op, arguments)
	return binding, true, err
}
func (c *programChecker) referenceCollection(op *catalogue.Operation, expected *types.Type, inputs []*types.Type, result *types.Type) (ValueBinding, bool, error) {
	file, names, signature, err := c.collectionSignature(op)
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
			return ValueBinding{}, true, fmt.Errorf("collection callback arity mismatch")
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
	binding, err := c.instantiateCollection(op, arguments)
	return binding, true, err
}

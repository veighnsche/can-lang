package check

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

func (c *programChecker) parameters() map[string]*types.Type {
	if c.current != nil {
		return c.current.Parameters
	}
	return nil
}
func (c *programChecker) bindingIdentity(identity string) string {
	if c.current != nil && c.current.Instance != "" {
		prefix := c.current.Symbol.ID + "/input/"
		if strings.HasPrefix(identity, prefix) {
			return c.current.Instance + "/input/" + strings.TrimPrefix(identity, prefix)
		}
	}
	return identity
}

func (c *programChecker) specialize(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, err
	}
	if op := collectionOperation(symbol.ID); op != nil {
		arguments := make([]*types.Type, len(args))
		for i, arg := range args {
			arguments[i], err = c.annotation(file, arg, false)
			if err != nil {
				return ValueBinding{}, err
			}
		}
		return c.instantiateCollection(op, arguments)
	}
	if op := browserStateOperation(symbol.ID); op != nil {
		arguments := make([]*types.Type, len(args))
		for i, arg := range args {
			arguments[i], err = c.annotation(file, arg, false)
			if err != nil {
				return ValueBinding{}, err
			}
		}
		return c.instantiateBrowserState(op, arguments)
	}
	if codecOperation(symbol.ID) {
		return c.specializeCodec(file, scope, name, args)
	}
	if streamOperation(symbol.ID) != nil {
		return c.specializeStream(file, scope, name, args)
	}
	if httpGenericOperation(symbol.ID) {
		return c.specializeHTTP(file, scope, name, args)
	}
	if formGenericOperation(symbol.ID) {
		return c.specializeForm(file, scope, name, args)
	}
	if sqlGenericOperation(symbol.ID) {
		return c.specializeSQL(file, scope, name, args)
	}
	declaration, ok := symbol.Declaration.(*syntax.FunctionDecl)
	if !ok || len(symbol.Parameters) == 0 {
		return ValueBinding{}, fmt.Errorf("declaration %s is not a generic source function", symbol.ID)
	}
	if len(args) != len(symbol.Parameters) {
		return ValueBinding{}, fmt.Errorf("generic function %s expects %d type arguments", symbol.ID, len(symbol.Parameters))
	}
	arguments := make([]*types.Type, len(args))
	for i, arg := range args {
		arguments[i], err = c.annotation(file, arg, false)
		if err != nil {
			return ValueBinding{}, err
		}
	}
	return c.instantiateFunction(symbol, declaration, arguments, applicationSite(file, name.Span.Start), args)
}

func (c *programChecker) instantiateFunction(symbol *resolve.Symbol, declaration *syntax.FunctionDecl, arguments []*types.Type, request string, substitutions ...[]syntax.TypeNode) (ValueBinding, error) {
	if opaque := opaqueArgument(arguments); opaque != "" {
		// Opaque variables only arise while an exported generic declaration
		// checks symbolically. Same-declaration recursion with identical
		// variables reuses the symbolic contract, exactly like the concrete
		// same-instance cache; any other generic instantiation would need
		// the callee template to be parametric, which the declaration
		// check cannot assume.
		if c.current != nil && c.current.Symbol.ID == symbol.ID && identicalParameters(c.current, symbol, arguments) {
			return ValueBinding{Identity: c.current.Instance, Type: c.bindings[c.current.Instance]}, nil
		}
		caller := "an exported generic declaration"
		if c.current != nil {
			caller = "exported generic declaration " + c.current.Symbol.ID
		}
		return ValueBinding{}, fmt.Errorf("generic function %s cannot be instantiated with opaque type parameter %s from %s: pass the value through a non-generic contract or an explicit callable input", symbol.ID, opaque, caller)
	}
	key, err := types.SpecializationKey(symbol.ID, arguments)
	if err != nil {
		return ValueBinding{}, err
	}
	if len(substitutions) == 1 && c.provesRecursiveGrowth(symbol, arguments, substitutions[0]) {
		return ValueBinding{}, fmt.Errorf("expanding polymorphic recursion from %s to %s", c.current.Identity(), key)
	}
	if existing := c.instances[key]; existing != nil {
		if !containsString(existing.Requests, request) {
			existing.Requests = append(existing.Requests, request)
		}
		return ValueBinding{Identity: key, Type: c.bindings[key]}, nil
	}
	if len(c.instances) >= 4096 {
		return ValueBinding{}, fmt.Errorf("concrete function specialization exceeds implementation limit at %s", symbol.ID)
	}
	parameters := map[string]*types.Type{}
	for i, name := range symbol.Parameters {
		parameters[name] = arguments[i]
	}
	file := c.world.Files[symbol.Source]
	signature, descriptor, fields, err := genericSignature(declaration)
	if err != nil {
		return ValueBinding{}, err
	}
	if len(declaration.Inputs) > 0 && declaration.Inputs[len(declaration.Inputs)-1].Variadic {
		c.variadic[key] = true
	}
	contract, err := c.specializer.Resolve(file, signature, parameters, true)
	if err != nil {
		return ValueBinding{}, fmt.Errorf("specialization %s: %w", key, err)
	}
	descriptor.Contract = contract
	c.callables[key] = descriptor
	c.bindings[key] = contract
	for i, field := range fields {
		c.bindings[key+"/input/"+field.Name.Text] = contract.Inputs()[i]
	}
	fn := &ProgramFunction{Requests: []string{request}, Symbol: symbol, Instance: key, TypeArguments: append([]*types.Type(nil), arguments...), Parameters: parameters}
	c.instances[key] = fn
	c.program.Functions = append(c.program.Functions, fn)
	return ValueBinding{Identity: key, Type: contract}, nil
}

// genericSignature builds the callable signature, declaration descriptor and
// ordered receiver/input fields shared by concrete instantiation and the
// exported-generic symbolic declaration check. The variadic tail input is
// wrapped as an array, exactly as call sites observe it.
func genericSignature(declaration *syntax.FunctionDecl) (*syntax.CallableType, CallableDeclaration, []syntax.Field, error) {
	signature := &syntax.CallableType{Result: declaration.Result, Errors: declaration.Errors}
	descriptor := CallableDeclaration{Kind: resolve.Function, Receiver: declaration.Receiver != nil}
	var fields []syntax.Field
	if declaration.Receiver != nil {
		fields = append(fields, *declaration.Receiver)
		descriptor.Names = append(descriptor.Names, declaration.Receiver.Name.Text)
		descriptor.Near = append(descriptor.Near, false)
	}
	for i, input := range declaration.Inputs {
		field := input.Field
		if input.Variadic {
			if i != len(declaration.Inputs)-1 {
				return nil, CallableDeclaration{}, nil, fmt.Errorf("variadic input must be last")
			}
			field.Type = &syntax.ArrayType{Element: field.Type}
		}
		fields = append(fields, field)
		descriptor.Names = append(descriptor.Names, input.Name.Text)
		descriptor.Near = append(descriptor.Near, input.Near)
	}
	for _, field := range fields {
		signature.Inputs = append(signature.Inputs, field.Type)
	}
	return signature, descriptor, fields, nil
}

// opaqueArgument names the first opaque exported-generic type variable
// mentioned by arguments, or "" when every argument is concrete.
func opaqueArgument(arguments []*types.Type) string {
	for _, argument := range arguments {
		if name := types.OpaqueParameterName(argument); name != "" {
			return name
		}
	}
	return ""
}

// identicalParameters reports whether arguments reproduce the current
// function's parameter bindings exactly, so symbolic self-recursion reuses
// the symbolic contract instead of instantiating a second declaration.
func identicalParameters(current *ProgramFunction, symbol *resolve.Symbol, arguments []*types.Type) bool {
	if len(arguments) != len(symbol.Parameters) {
		return false
	}
	for i, name := range symbol.Parameters {
		if !types.Equal(arguments[i], current.Parameters[name]) {
			return false
		}
	}
	return true
}

// Concrete containment corroborates source substitution evidence; it is not a
// proof by itself. A nominal field can lead through larger arguments and then
// stabilize. Unknown transitions use the separately labelled discovery bound.
func growingArguments(before, after []*types.Type) bool {
	if len(before) == 0 || len(before) != len(after) {
		return false
	}
	strict := false
	for i, old := range before {
		if !containsType(after[i], old, map[string]bool{}) {
			return false
		}
		strict = strict || !types.Equal(old, after[i])
	}
	return strict
}
func containsType(container, want *types.Type, seen map[string]bool) bool {
	if types.Equal(container, want) {
		return true
	}
	if container == nil || seen[container.Identity()] {
		return false
	}
	seen[container.Identity()] = true
	children := container.Arguments()
	if container.Element() != nil {
		children = append(children, container.Element())
	}
	if container.Result() != nil {
		children = append(children, container.Result())
	}
	children = append(children, container.Inputs()...)
	for _, child := range children {
		if containsType(child, want, seen) {
			return true
		}
	}
	return false
}

func applicationSite(file *resolve.File, start int) string {
	return fmt.Sprintf("%s byte %d", file.Source.Syntax.Source.Name(), start)
}
func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

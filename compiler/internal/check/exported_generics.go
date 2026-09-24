package check

import (
	"fmt"
	"reflect"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Exported generic declarations check once under symbolic type variables, in
// addition to the concrete per-instance checks every reachable specialization
// still receives. A type variable is opaque in the declaration body: values
// of variable type only pass through (bind, return, construct composites,
// supply matching variable slots), while every representation-requiring
// operation must arrive as an explicit named callable input or as a callable
// held by a named dictionary-record input. Arithmetic, ordering, equality,
// field access, method calls, indexing, slicing, matching on a bare variable
// and instantiating another generic function with an opaque argument all
// fail at the declaring file, so a dependency body edit cannot silently
// narrow the contract its consumers rely on. Same-declaration recursion with
// identical variables reuses the symbolic contract. Unexported (local)
// generic declarations keep template semantics: only their concrete
// instances check. There is no implicit trait search and no error-set type
// parameter; a bare variable in `emits` is not a nominal error.
func (c *programChecker) checkExportedGenerics(files []*resolve.File) error {
	for _, file := range files {
		for _, node := range file.Source.Syntax.Declarations {
			d, ok := node.(*syntax.FunctionDecl)
			if !ok || len(d.Parameters) == 0 {
				continue
			}
			symbol := file.Package.Scope.Symbols[d.Name.Text]
			if symbol == nil || !symbol.Public {
				continue
			}
			if err := c.symbolicDeclaration(file, symbol, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *programChecker) symbolicDeclaration(file *resolve.File, symbol *resolve.Symbol, d *syntax.FunctionDecl) error {
	defFile := file.Source.Syntax.Source.Name()
	parameters := map[string]*types.Type{}
	arguments := make([]*types.Type, len(symbol.Parameters))
	for i, name := range symbol.Parameters {
		param, err := types.SymbolicParameter(symbol.ID, name)
		if err != nil {
			return source.LocateCode(defFile, d.DeclSpan(), "CAN-CHECK-EXPORTED-GENERIC", err)
		}
		parameters[name] = param
		arguments[i] = param
	}
	signature, descriptor, fields, err := genericSignature(d)
	if err != nil {
		return source.LocateCode(defFile, d.DeclSpan(), "CAN-CHECK-EXPORTED-GENERIC", err)
	}
	// The symbolic identity keys declaration-only contracts. It is never a
	// specialization key, never enters the instance cache or emitted
	// functions, and its checked region is discarded after validation.
	identity := symbol.ID + "/symbolic"
	contract, err := c.specializer.Resolve(file, signature, parameters, true)
	if err != nil {
		err = fmt.Errorf("exported generic function %s: %w", symbol.ID, err)
		return source.LocateCode(defFile, d.DeclSpan(), "CAN-CHECK-EXPORTED-GENERIC", err)
	}
	descriptor.Contract = contract
	c.callables[identity] = descriptor
	c.bindings[identity] = contract
	for i, field := range fields {
		c.bindings[identity+"/input/"+field.Name.Text] = contract.Inputs()[i]
	}
	if len(d.Inputs) > 0 && d.Inputs[len(d.Inputs)-1].Variadic {
		c.variadic[identity] = true
	}
	fn := &ProgramFunction{Symbol: symbol, Instance: identity, TypeArguments: arguments, Parameters: parameters}
	saved := c.current
	c.current = fn
	defer func() { c.current = saved }()
	if err := c.gatherBody(file, reflect.ValueOf(d.Body)); err != nil {
		err = fmt.Errorf("exported generic function %s: %w", symbol.ID, err)
		err = source.Relate(defFile, d.DeclSpan(), "exported generic declared here", err)
		return stampCode(err, "CAN-CHECK-EXPORTED-GENERIC")
	}
	context, err := c.functionContext(fn)
	if err != nil {
		err = fmt.Errorf("exported generic function %s: %w", symbol.ID, err)
		return source.LocateCode(defFile, d.DeclSpan(), "CAN-CHECK-EXPORTED-GENERIC", err)
	}
	if _, err := CheckRegion(context, d.Body); err != nil {
		err = fmt.Errorf("exported generic function %s requires operations on type variables to be explicit named callable or dictionary inputs: %w", symbol.ID, err)
		err = source.Relate(defFile, d.DeclSpan(), "exported generic declared here", err)
		return stampCode(err, "CAN-CHECK-EXPORTED-GENERIC")
	}
	return nil
}

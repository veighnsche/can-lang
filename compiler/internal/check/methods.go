package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

type MethodApplication struct {
	CallbackInputs []*types.Type
	CallbackResult *types.Type
	Receiver       *types.Type
	Name           syntax.Token
	Types          []syntax.TypeNode
	Arguments      []syntax.Argument
	Expected       *types.Type
	Reference      bool
	Expressions    *Expressions
}

func (c *regionChecker) resolveMethod(application MethodApplication) (ValueBinding, error) {
	if c.context.ResolveMethod != nil {
		return c.context.ResolveMethod(application)
	}
	if c.context.Method == nil {
		return ValueBinding{}, fmt.Errorf("missing method resolver")
	}
	return c.context.Method(application.Receiver, application.Name, application.Types)
}
func (c *programChecker) method(file *resolve.File, a MethodApplication) (ValueBinding, error) {
	var receiver *resolve.Symbol
	for _, pkg := range c.world.Packages {
		for _, symbol := range pkg.Scope.Symbols {
			if symbol.ID == a.Receiver.Declaration() {
				receiver = symbol
			}
		}
	}
	symbol, err := file.Method(receiver, a.Name.Text)
	if err != nil {
		return ValueBinding{}, err
	}
	d, ok := symbol.Declaration.(*syntax.FunctionDecl)
	if !ok || d.Receiver == nil {
		return ValueBinding{}, fmt.Errorf("method requires receiver declaration")
	}
	if len(symbol.Parameters) == 0 {
		if len(a.Types) != 0 {
			return ValueBinding{}, fmt.Errorf("non-generic method does not take type arguments")
		}
		return ValueBinding{Identity: symbol.ID, Type: c.bindings[symbol.ID]}, nil
	}
	var arguments []*types.Type
	if len(a.Types) != 0 {
		if len(a.Types) != len(symbol.Parameters) {
			return ValueBinding{}, fmt.Errorf("generic method type argument arity mismatch")
		}
		for _, node := range a.Types {
			arg, err := c.annotation(file, node, false)
			if err != nil {
				return ValueBinding{}, err
			}
			arguments = append(arguments, arg)
		}
	} else {
		seed := typeConstraint{d.Receiver.Type, a.Receiver}
		if a.Reference {
			seeds := []typeConstraint{seed}
			if a.CallbackInputs != nil {
				more, e := callbackConstraints(d, a.CallbackInputs, a.CallbackResult)
				if e != nil {
					return ValueBinding{}, e
				}
				seeds = append(seeds, more...)
			}
			binding, _, err := c.inferDeclaredReference(symbol, d, a.Expected, a.Expressions, applicationSite(file, a.Name.Span.Start), seeds...)
			return binding, err
		}
		constraints, err := genericArguments(d.Inputs, a.Arguments)
		if err != nil {
			return ValueBinding{}, err
		}
		arguments, err = c.inferArguments(c.world.Files[symbol.Source], symbol.Parameters, d.Result, a.Expected, constraints, a.Expressions, seed)
		if err != nil {
			return ValueBinding{}, fmt.Errorf("generic method %s at byte %d: %w", symbol.ID, a.Name.Span.Start, err)
		}
	}
	return c.instantiateFunction(symbol, d, arguments, applicationSite(file, a.Name.Span.Start), a.Types)
}

package check

import (
	"fmt"
	"sort"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// CompletionContext consumes sealed declaration types and the current file's
// eligible-kind resolvers. Type resolution cannot grow the graph during bodies.
// Handlers use their own Identity/Parent and selected Result, with the enclosing
// escaping error contract. Their callers later assemble coordination/native data.
type CompletionContext struct {
	Identity, Parent string
	Kind             ir.RegionKind
	File             *source.File
	Scope            *resolve.Scope
	Result           *types.Type
	Errors           ErrorBound
	Registry         *ErrorRegistry
	Expressions      *Expressions
	Type             func(syntax.TypeNode, bool) (*types.Type, error)
	ErrorName        func(syntax.QualifiedName) (string, error)
	PatternName      func(syntax.QualifiedName) (string, error)
	// Variadic declaration contracts have a final array input in the private ABI.
	Variadic map[string]bool
	Method   func(*types.Type, syntax.Token, []syntax.TypeNode) (ValueBinding, error)
	// Parameters already have resolved identities and are exposed by Expressions.
	Parameters []ir.Local
}
type regionChecker struct {
	context CompletionContext
	region  *ir.Region
	serial  int
	uses    LocalUses
	locals  map[string]*types.Type
}
type bodyScope struct{ symbols *resolve.Scope }

func CheckRegion(context CompletionContext, block syntax.Block) (*ir.Region, error) {
	if context.Identity == "" || context.File == nil || context.Scope == nil || context.Registry == nil || context.Expressions == nil || context.Type == nil || !types.Equal(context.Result, context.Result) {
		return nil, fmt.Errorf("completion region requires resolved context")
	}
	if context.Kind != ir.FunctionRegion && context.Kind != ir.HandlerRegion {
		return nil, fmt.Errorf("unknown completion region kind")
	}
	if context.Kind == ir.FunctionRegion && context.Parent != "" || context.Kind == ir.HandlerRegion && (context.Parent == "" || context.Parent == context.Identity) {
		return nil, fmt.Errorf("invalid completion region parent")
	}
	seenInputs := map[string]bool{}
	for _, input := range context.Parameters {
		if input.Identity == "" || seenInputs[input.Identity] || !types.Equal(input.Type, input.Type) || input.Type.Kind() == types.Void {
			return nil, fmt.Errorf("invalid or duplicate region input")
		}
		seenInputs[input.Identity] = true
	}
	r := &ir.Region{ID: context.Identity, Parent: context.Parent, Source: context.File.Name(), Kind: context.Kind, Span: block.Span, Result: context.Result, Inputs: append([]ir.Local(nil), context.Parameters...)}
	for _, e := range context.Errors.Entries() {
		r.Errors = append(r.Errors, e.Type)
	}
	c := &regionChecker{context: context, region: r, locals: map[string]*types.Type{}, uses: LocalUses{Names: map[*syntax.NameExpr]string{}, Captures: map[*syntax.ReferenceExpr][]string{}}}
	var err error
	r.Body, err = c.block(block, bodyScope{context.Scope})
	if err != nil {
		return nil, fmt.Errorf("%s region %s: %w", r.Source, r.ID, err)
	}
	return r, nil
}
func (c *regionChecker) identity(name string) string {
	c.serial++
	return fmt.Sprintf("%s/local/%d/%s", c.region.ID, c.serial, name)
}
func (c *regionChecker) child(parent bodyScope) bodyScope {
	return bodyScope{resolve.NewScope(parent.symbols)}
}
func (c *regionChecker) bind(scope bodyScope, name string, typ *types.Type) (*ir.Local, error) {
	if name == "" || !types.Equal(typ, typ) || typ.Kind() == types.Void {
		return nil, fmt.Errorf("invalid local data binding")
	}
	local := &ir.Local{Identity: c.identity(name), Type: typ}
	if err := scope.symbols.Define(&resolve.Symbol{ID: local.Identity, Name: name, Kind: resolve.Value}); err != nil {
		return nil, err
	}
	c.locals[local.Identity] = typ
	return local, nil
}
func (c *regionChecker) lexical(scope bodyScope, name syntax.QualifiedName, call bool) (ValueBinding, bool) {
	if name.Package != "" {
		return ValueBinding{}, false
	}
	for s := scope.symbols; s != nil && s != c.context.Scope; s = s.Parent {
		if symbol := s.Symbols[name.Name]; symbol != nil {
			typ := c.locals[symbol.ID]
			if typ != nil && (!call || typ.Kind() == types.Callable) {
				return ValueBinding{symbol.ID, typ}, true
			}
		}
	}
	return ValueBinding{}, false
}
func (c *regionChecker) expressions(scope bodyScope) *Expressions {
	e := *c.context.Expressions
	e.Value = func(name syntax.QualifiedName) (ValueBinding, error) {
		if b, ok := c.lexical(scope, name, false); ok {
			return b, nil
		}
		if c.context.Expressions.Value == nil {
			return ValueBinding{}, fmt.Errorf("unknown value %s", name.Name)
		}
		return c.context.Expressions.Value(name)
	}
	e.Function = func(name syntax.QualifiedName) (ValueBinding, error) {
		if b, ok := c.lexical(scope, name, true); ok {
			return b, nil
		}
		if c.context.Expressions.Function == nil {
			return ValueBinding{}, fmt.Errorf("unknown callable %s", name.Name)
		}
		return c.context.Expressions.Function(name)
	}
	e.ResolvedValue = func(n *syntax.NameExpr, b ValueBinding) { c.uses.Names[n] = b.Identity }
	e.CallCheck = func(n *syntax.CallExpr, expected *types.Type) (*ir.Expression, error) {
		call, err := c.invocation(n, scope)
		if err != nil {
			return nil, err
		}
		if len(call.Errors) != 0 {
			return nil, fmt.Errorf("domain-fallible call requires explicit completion handling")
		}
		if call.Result.Kind() == types.Void {
			return nil, fmt.Errorf("void call is not a data value")
		}
		return &ir.Expression{Kind: ir.InvocationValue, Span: n.Span, Type: call.Result, Invocation: call}, nil
	}
	e.MatchCheck = func(n *syntax.MatchExpr, expected *types.Type) (*ir.Expression, error) {
		if expected == nil || expected.Kind() == types.Void {
			return nil, fmt.Errorf("value match requires an expected data type")
		}
		match, err := c.match(n.Match, scope, expected)
		if err != nil {
			return nil, err
		}
		return &ir.Expression{Kind: ir.MatchValue, Span: n.Span, Type: expected, Match: match}, nil
	}
	return &e
}
func (c *regionChecker) block(block syntax.Block, parent bodyScope) (*ir.Block, error) {
	scope := c.child(parent)
	out := &ir.Block{Span: block.Span}
	var last *ir.Local
	var lastChecker *Expressions
	for _, step := range block.Steps {
		switch n := step.(type) {
		case *syntax.BindingStep:
			typ, err := c.context.Type(n.Type, false)
			if err != nil {
				return nil, err
			}
			e := c.expressions(scope)
			value, err := e.Check(n.Value, typ)
			if err != nil {
				return nil, err
			}
			// Freeze initializer lookup before inserting this local: later shadowing
			// must not change I48's substitution and nested-type evidence.
			before := *scope.symbols
			before.Symbols = map[string]*resolve.Symbol{}
			for k, v := range scope.symbols.Symbols {
				before.Symbols[k] = v
			}
			lastChecker = c.expressions(bodyScope{&before})
			local, err := c.bind(scope, n.Name.Text, typ)
			if err != nil {
				return nil, err
			}
			out.Steps = append(out.Steps, ir.Statement{Local: local, Value: value})
			last = local
		case *syntax.CallStep:
			call, err := c.invocation(n.Call, scope)
			if err != nil {
				return nil, err
			}
			if call.Result.Kind() != types.Void || len(call.Errors) != 0 {
				return nil, fmt.Errorf("call step requires void success and emits []")
			}
			out.Steps = append(out.Steps, ir.Statement{Call: call})
			last = nil
		default:
			return nil, fmt.Errorf("step %T requires its owning native/coordination pass", step)
		}
	}
	var err error
	out.Terminal, err = c.completion(block.Terminal, scope)
	if err != nil {
		return nil, err
	}
	if last != nil {
		if err = CheckLocalForwarding(LocalForwarding{File: c.context.File, Block: block, Binding: ValueBinding{last.Identity, last.Type}, Uses: c.uses, Checker: lastChecker, Expected: c.region.Result}); err != nil {
			return nil, err
		}
	}
	return out, nil
}
func (c *regionChecker) completion(body syntax.Body, scope bodyScope) (*ir.Completion, error) {
	if body == nil {
		return nil, fmt.Errorf("region requires an explicit terminal completion")
	}
	out := &ir.Completion{RegionID: c.region.ID, Span: body.BodySpan()}
	var err error
	switch n := body.(type) {
	case *syntax.SuccessBody:
		out.Kind = ir.SuccessCompletion
		if c.region.Result.Kind() == types.Void {
			if n.Value != nil {
				return nil, fmt.Errorf("void success requires bare ok")
			}
		} else {
			if n.Value == nil {
				return nil, fmt.Errorf("nonvoid success requires a value")
			}
			out.Value, err = c.expressions(scope).Check(n.Value, c.region.Result)
		}
	case *syntax.FailureBody:
		out.Kind = ir.DomainCompletion
		out.Value, err = c.expressions(scope).Check(n.Error, nil)
		if err == nil {
			var bound ErrorBound
			bound, err = c.context.Registry.Bound([]*types.Type{out.Value.Type})
			if err == nil {
				err = c.escaping(bound)
			}
		}
	case *syntax.RelayBody:
		out.Kind = ir.RelayCompletion
		out.Call, err = c.invocation(n.Call, scope)
		if err == nil && !types.Assignable(out.Call.Result, c.region.Result) {
			err = fmt.Errorf("relay success does not fit current region")
		}
		if err == nil {
			var bound ErrorBound
			bound, err = c.context.Registry.Bound(out.Call.Errors)
			if err == nil {
				err = c.escaping(bound)
			}
		}
	case *syntax.DoBody:
		if len(n.Block.Steps) == 0 {
			return nil, fmt.Errorf("do requires two or more steps")
		}
		out.Kind = ir.DoCompletion
		out.Block, err = c.block(n.Block, scope)
	case *syntax.MatchBody:
		out.Kind = ir.MatchCompletion
		out.Match, err = c.match(n.Match, scope, nil)
	default:
		return nil, fmt.Errorf("ordinary value cannot terminate a completion region")
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}
func unionErrors(groups ...[]*types.Type) []*types.Type {
	found := map[string]*types.Type{}
	for _, group := range groups {
		for _, t := range group {
			found[t.Identity()] = t
		}
	}
	keys := make([]string, 0, len(found))
	for k := range found {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*types.Type, 0, len(keys))
	for _, k := range keys {
		out = append(out, found[k])
	}
	return out
}
func (c *regionChecker) invocation(n *syntax.CallExpr, scope bodyScope) (*ir.Invocation, error) {
	if n == nil {
		return nil, fmt.Errorf("missing invocation")
	}
	if len(n.Invocation.Types) != 0 {
		return nil, fmt.Errorf("generic invocation requires specialization")
	}
	out := &ir.Invocation{Span: n.Span}
	e := c.expressions(scope)
	appendSlice := func(receiver *ir.Expression, args []syntax.Argument, span source.Span) error {
		value, err := e.sliceArguments(receiver, args, n)
		if err != nil {
			return err
		}
		out.Steps = append(out.Steps, ir.InvocationStep{Native: value, Span: span, Result: value.Type, SuccessBinding: c.identity("slice")})
		out.Result = value.Type
		return nil
	}
	var first ValueBinding
	var receiver *ir.Expression
	var err error
	switch callee := n.Invocation.Callee.(type) {
	case *syntax.NameExpr:
		first, err = e.Function(callee.Name)
		if err == nil {
			c.uses.Names[callee] = first.Identity
		}
	case *syntax.FieldExpr:
		receiver, err = e.Check(callee.Receiver, nil)
		if err == nil {
			if callee.Field.Text == "slice" && (receiver.Type.Kind() == types.Array || scalar(receiver.Type, "str")) {
				err = appendSlice(receiver, n.Invocation.Arguments, n.Invocation.Span)
				break
			}
			if c.context.Method == nil {
				return nil, fmt.Errorf("missing resolved method contract")
			}
			first, err = c.context.Method(receiver.Type, callee.Field, nil)
		}
	default:
		return nil, fmt.Errorf("call requires an eligible named callable")
	}
	if err != nil {
		return nil, err
	}
	appendStep := func(binding ValueBinding, args []syntax.Argument, receiver *ir.Expression, span source.Span) error {
		if binding.Identity == "" || !types.Equal(binding.Type, binding.Type) || binding.Type.Kind() != types.Callable {
			return fmt.Errorf("invalid resolved callable contract")
		}
		step := ir.InvocationStep{Identity: binding.Identity, Span: span, Result: binding.Type.Result(), Errors: binding.Type.Errors(), SuccessBinding: c.identity("call")}
		var err error
		step.Prepare, step.Arguments, err = c.arguments(e, binding, args, receiver)
		if err != nil {
			return err
		}
		out.Steps = append(out.Steps, step)
		out.Result = step.Result
		out.Errors = unionErrors(out.Errors, step.Errors)
		return nil
	}
	if len(out.Steps) == 0 {
		if err = appendStep(first, n.Invocation.Arguments, receiver, n.Invocation.Span); err != nil {
			return nil, err
		}
	}
	for _, method := range n.Methods {
		if out.Result.Kind() == types.Void {
			return nil, fmt.Errorf("void cannot be a method receiver")
		}
		prior := out.Steps[len(out.Steps)-1]
		receiver = &ir.Expression{Kind: ir.Binding, Type: prior.Result, Text: prior.SuccessBinding, Span: method.Span}
		if method.Name.Text == "slice" && (out.Result.Kind() == types.Array || scalar(out.Result, "str")) {
			if len(method.Types) != 0 {
				return nil, fmt.Errorf("slice does not take type arguments")
			}
			if err = appendSlice(receiver, method.Arguments, method.Span); err != nil {
				return nil, err
			}
			continue
		}
		if c.context.Method == nil {
			return nil, fmt.Errorf("missing resolved method contract")
		}
		binding, err := c.context.Method(out.Result, method.Name, method.Types)
		if err != nil {
			return nil, err
		}
		if err = appendStep(binding, method.Arguments, receiver, method.Span); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *regionChecker) escaping(bound ErrorBound) error {
	if err := c.context.Errors.CheckEscaping(bound); err != nil {
		return err
	}
	var actual []*types.Type
	for _, entry := range bound.Entries() {
		actual = append(actual, entry.Type)
	}
	c.region.Escapes = unionErrors(c.region.Escapes, actual)
	return nil
}

package check

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// callSite carries a generic instantiation request: the human-readable
// request record plus the source location for call-site evidence.
type callSite struct {
	text string
	file *resolve.File
	span source.Span
}

func makeCallSite(file *resolve.File, span source.Span) callSite {
	return callSite{text: applicationSite(file, span.Start), file: file, span: span}
}

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
	if fetchGenericOperation(symbol.ID) {
		return c.specializeFetch(file, scope, name, args)
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
	return c.instantiateFunction(symbol, declaration, arguments, makeCallSite(file, name.Span), args)
}

func (c *programChecker) instantiateFunction(symbol *resolve.Symbol, declaration *syntax.FunctionDecl, arguments []*types.Type, request callSite, substitutions ...[]syntax.TypeNode) (ValueBinding, error) {
	if opaque := opaqueArgument(arguments); opaque != "" {
		// Opaque variables only arise while an exported generic
		// declaration checks symbolically. Same-declaration recursion
		// with identical variables reuses the symbolic contract,
		// exactly like the concrete same-instance cache; every other
		// opaque call resolves as a symbolic proof edge.
		if c.current != nil && c.current.Symbol.ID == symbol.ID && identicalParameters(c.current, symbol, arguments) {
			return ValueBinding{Identity: c.current.Instance, Type: c.bindings[c.current.Instance]}, nil
		}
		return c.symbolicCall(symbol, declaration, arguments, request)
	}
	key, err := types.SpecializationKey(symbol.ID, arguments)
	if err != nil {
		return ValueBinding{}, err
	}
	if len(substitutions) == 1 && c.provesRecursiveGrowth(symbol, arguments, substitutions[0]) {
		return ValueBinding{}, fmt.Errorf("expanding polymorphic recursion from %s to %s", c.current.Identity(), key)
	}
	if existing := c.instances[key]; existing != nil {
		if !containsString(existing.Requests, request.text) {
			existing.Requests = append(existing.Requests, request.text)
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
	fn := &ProgramFunction{Requests: []string{request.text}, Symbol: symbol, Instance: key, TypeArguments: append([]*types.Type(nil), arguments...), Parameters: parameters}
	c.instances[key] = fn
	c.program.Functions = append(c.program.Functions, fn)
	return ValueBinding{Identity: key, Type: contract}, nil
}

// symbolicCall resolves a generic call whose type arguments mention the
// caller's opaque type variables. The callee must be a public generic with
// a committed symbolic proof, or a member of the caller's own provisional
// component. The caller's symbolic type expressions are substituted into
// the callee's checked signature; the callee body is never rechecked under
// the caller's variables. The substituted contract is published under a
// call-site-local proof identity. No SpecializationKey is allocated, no
// instance is cached and no emitted function is created.
func (c *programChecker) symbolicCall(symbol *resolve.Symbol, declaration *syntax.FunctionDecl, arguments []*types.Type, request callSite) (ValueBinding, error) {
	caller := c.current
	if caller == nil || caller.Symbol == nil || !strings.HasSuffix(caller.Instance, "/symbolic") {
		name := "an exported generic declaration"
		if caller != nil && caller.Symbol != nil {
			name = "exported generic declaration " + caller.Symbol.ID
		}
		return ValueBinding{}, fmt.Errorf("generic function %s cannot be instantiated with opaque type parameter %s from %s: pass the value through a non-generic contract or an explicit callable input", symbol.ID, opaqueArgument(arguments), name)
	}
	fail := func(err error) (ValueBinding, error) {
		if request.file != nil && request.file.Source != nil && request.file.Source.Syntax != nil {
			err = source.LocateCode(request.file.Source.Syntax.Source.Name(), request.span, "CAN-CHECK-EXPORTED-GENERIC", err)
		}
		if calleeFile := c.world.Files[symbol.Source]; calleeFile != nil && calleeFile.Source != nil {
			err = source.Relate(calleeFile.Source.Syntax.Source.Name(), declaration.DeclSpan(), "symbolic callee declared here", err)
		}
		return ValueBinding{}, err
	}
	if !symbol.Public {
		return fail(fmt.Errorf("generic function %s cannot be called with opaque type parameter %s from exported generic declaration %s: %s is private and has only concrete-template evidence, so it cannot be a symbolic callee", symbol.ID, opaqueArgument(arguments), caller.Symbol.ID, symbol.ID))
	}
	callerComponent, callerKnown := c.symbolicComponent[caller.Symbol.ID]
	calleeComponent, calleeKnown := c.symbolicComponent[symbol.ID]
	switch {
	case c.symbolicProofs[symbol.ID]:
		// Committed proof: acyclic reuse, including nested G<box<T>>.
	case callerKnown && calleeKnown && callerComponent == calleeComponent:
		// Provisional internal edge: only bare caller formals or fully
		// closed types keep the component's instance set finite.
		if index, name := growingSymbolicEdge(caller, arguments); index >= 0 {
			return fail(fmt.Errorf("expanding symbolic cycle rejected: %s; type argument %d applies a constructor to opaque type parameter %s on an internal component edge, so the instance set cannot stay finite", c.symbolicChain(caller.Symbol.ID, symbol.ID, request.text), index+1, name))
		}
	default:
		return fail(fmt.Errorf("generic function %s cannot be called with opaque type parameter %s from exported generic declaration %s: no committed symbolic proof for %s is visible to this declaration; a failed component publishes no proof and an unvalidated callee cannot be reused", symbol.ID, opaqueArgument(arguments), caller.Symbol.ID, symbol.ID))
	}
	calleeFile := c.world.Files[symbol.Source]
	if calleeFile == nil {
		return fail(fmt.Errorf("generic function %s has no declaring file", symbol.ID))
	}
	signature, descriptor, fields, err := genericSignature(declaration)
	if err != nil {
		return fail(err)
	}
	parameters := map[string]*types.Type{}
	for i, name := range symbol.Parameters {
		parameters[name] = arguments[i]
	}
	contract, err := c.specializer.Resolve(calleeFile, signature, parameters, true)
	if err != nil {
		return fail(fmt.Errorf("symbolic call %s to %s: %w", caller.Symbol.ID, symbol.ID, err))
	}
	descriptor.Contract = contract
	identity := symbolicCallIdentity(caller.Symbol.ID, symbol.ID, arguments)
	c.callables[identity] = descriptor
	c.bindings[identity] = contract
	for i, field := range fields {
		c.bindings[identity+"/input/"+field.Name.Text] = contract.Inputs()[i]
	}
	if len(declaration.Inputs) > 0 && declaration.Inputs[len(declaration.Inputs)-1].Variadic {
		c.variadic[identity] = true
	}
	return ValueBinding{Identity: identity, Type: contract}, nil
}

// symbolicCallIdentity derives a deterministic call-site-local proof
// identity from the caller, the callee and the substituted argument
// identities. Identical substitutions share one proof entry.
func symbolicCallIdentity(caller, callee string, arguments []*types.Type) string {
	parts := []string{"can-symbolic-call-v1", caller, callee}
	for _, argument := range arguments {
		parts = append(parts, argument.Identity())
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return callee + "/symbolic-call/" + hex.EncodeToString(digest[:8])
}

// growingSymbolicEdge reports the first type argument that is neither a
// bare caller formal nor a fully closed type, or -1 when every argument
// keeps the component's instance set finite.
func growingSymbolicEdge(caller *ProgramFunction, arguments []*types.Type) (int, string) {
	for i, argument := range arguments {
		bare := false
		for _, name := range caller.Symbol.Parameters {
			if types.Equal(argument, caller.Parameters[name]) {
				bare = true
				break
			}
		}
		if bare || !types.MentionsParameter(argument) {
			continue
		}
		return i, types.OpaqueParameterName(argument)
	}
	return -1, ""
}

// symbolicChain renders the call chain through the caller's component from
// the callee back to the caller, for expanding-cycle diagnostics. The
// syntactic scan edges carry the return path; the failing edge itself
// closes the cycle at the current request site.
func (c *programChecker) symbolicChain(caller, callee, site string) string {
	if caller == callee {
		return fmt.Sprintf("%s calls itself at %s", caller, site)
	}
	within := map[string][]symbolicScanEdge{}
	for _, edge := range c.symbolicScan {
		if from, ok := c.symbolicComponent[edge.caller]; ok {
			if to, ok := c.symbolicComponent[edge.callee]; ok && from == to {
				within[edge.caller] = append(within[edge.caller], edge)
			}
		}
	}
	previous := map[string]symbolicScanEdge{callee: {}}
	queue := []string{callee}
	for len(queue) > 0 && previous[caller].caller == "" {
		head := queue[0]
		queue = queue[1:]
		// Deterministic return path: first scan edge order wins.
		for _, edge := range within[head] {
			if _, seen := previous[edge.callee]; seen {
				continue
			}
			previous[edge.callee] = edge
			queue = append(queue, edge.callee)
		}
	}
	if previous[caller].caller == "" {
		return fmt.Sprintf("%s calls %s at %s", caller, callee, site)
	}
	var links []string
	for at := caller; at != callee; {
		edge := previous[at]
		links = append([]string{fmt.Sprintf("%s calls %s at %s", edge.caller, edge.callee, edge.site)}, links...)
		at = edge.caller
	}
	links = append([]string{fmt.Sprintf("%s calls %s at %s", caller, callee, site)}, links...)
	return strings.Join(links, ", then ")
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

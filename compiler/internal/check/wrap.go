package check

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Wrapper origins select the original boundary table a policy key belongs to.
const (
	WrapperNative  = "native"
	WrapperEmitted = "emitted"
)

// WrapperKey is one (origin, exact error specialization) policy key against
// the original operation boundary sets. Identity is the runtime hex form
// used for dispatch; Decl is the canonical declaration for set membership
// and diagnostics. Raw native obligations are never generic, so a
// declaration alone is exact there; emitted keys compare specializations.
type WrapperKey struct {
	Origin   string
	Identity string
	Decl     string
}

// WrapperRule is the checked rule for one key: the arm region with its
// escape set, or a default that forwards without a region.
type WrapperRule struct {
	Key     WrapperKey
	Region  *ir.Region
	Escapes []*types.Type
	From    []EscapeProvenance
	Handler string
	Default bool
}

// EscapeProvenance records why one bound member escapes: the selected
// handler for its key plus the inherit delegation chain below it, ending
// in "default" when delegation reaches the origin default.
type EscapeProvenance struct {
	Identity string
	Key      WrapperKey
	Handler  string
	Chain    []string
}

// WrapperPlan is the resolved policy for one wrapper: the effective rule
// for every key of the original boundary sets, the calculated public bound
// with provenance, and the locally declared keys for assertion coverage.
type WrapperPlan struct {
	Base       string
	Root       string
	Failed     string
	Rules      []WrapperRule
	Bound      []*types.Type
	Provenance []EscapeProvenance
	Local      []WrapperKey
}

func (p *WrapperPlan) rule(key WrapperKey) *WrapperRule {
	for i := range p.Rules {
		if p.Rules[i].Key == key {
			return &p.Rules[i]
		}
	}
	return nil
}

func (p *WrapperPlan) provenance(identity string) *EscapeProvenance {
	// One member can escape through several keys; prefer the most
	// informative true path: a delegating rule, then a direct rule, then a
	// bare default, each in deterministic key order.
	var direct, fallback *EscapeProvenance
	for i := range p.Provenance {
		if p.Provenance[i].Identity != identity {
			continue
		}
		entry := &p.Provenance[i]
		if entry.Handler == "" {
			if fallback == nil {
				fallback = entry
			}
			continue
		}
		if len(entry.Chain) != 0 {
			return entry
		}
		if direct == nil {
			direct = entry
		}
	}
	if direct != nil {
		return direct
	}
	return fallback
}

func (c *programChecker) gatherWrapper(file *resolve.File, declaration *syntax.WrapDecl) (*NativeDeclaration, error) {
	symbol := file.Package.Scope.Symbols[declaration.Name.Text]
	_, root, header, err := c.world.WrapperOrigin(file, symbol)
	if err != nil {
		return nil, err
	}
	origin := c.world.Files[root.Source]
	if origin == nil {
		return nil, fmt.Errorf("wrapper root %s has no declaring file", root.ID)
	}
	if c.annotations[origin] == nil {
		c.annotations[origin] = map[string]*types.Type{}
	}
	var state []syntax.Field
	if d, ok := root.Declaration.(*syntax.JudgeDecl); ok {
		state = d.State
	}
	native := &NativeDeclaration{Symbol: symbol, State: state, Descriptor: CallableDeclaration{Kind: symbol.Kind, State: len(state), Grouped: root.Kind == resolve.Judge}}
	connection, err := origin.Lookup(nil, header.Connection, resolve.ConnectionUse)
	if err != nil {
		return nil, err
	}
	native.Connection = connection.ID
	inputs := make([]syntax.Field, 0, len(header.Inputs)+len(state))
	for i, input := range header.Inputs {
		field := input.Field
		if input.Variadic {
			if i != len(header.Inputs)-1 {
				return nil, fmt.Errorf("variadic wrapper input must be last")
			}
			field.Type = &syntax.ArrayType{Element: field.Type}
			c.variadic[symbol.ID] = true
		}
		inputs = append(inputs, field)
		native.Descriptor.Names = append(native.Descriptor.Names, field.Name.Text)
		native.Descriptor.Near = append(native.Descriptor.Near, input.Near)
	}
	for _, field := range state {
		inputs = append(inputs, field)
		native.Descriptor.Names = append(native.Descriptor.Names, field.Name.Text)
		native.Descriptor.Near = append(native.Descriptor.Near, false)
	}
	for _, field := range inputs {
		typ, err := c.gather(origin, field.Type)
		if err != nil {
			return nil, err
		}
		c.bindings[symbol.ID+"/input/"+field.Name.Text] = typ
	}
	// The calculated signature is published by the policy pass once arm
	// escapes are known; callers cannot observe a provisional bound.
	if err := c.gatherBody(file, reflect.ValueOf(declaration)); err != nil {
		return nil, err
	}
	return native, nil
}

// wrapperChain returns the base chain from the wrapper itself down to the
// original fetch/judge root. Resolve already rejects base cycles; the
// guard here keeps checking total regardless.
func (c *programChecker) wrapperChain(file *resolve.File, symbol *resolve.Symbol) ([]*resolve.Symbol, error) {
	chain := []*resolve.Symbol{symbol}
	seen := map[string]bool{symbol.ID: true}
	hop := file
	for {
		current := chain[len(chain)-1]
		declaration, ok := current.Declaration.(*syntax.WrapDecl)
		if !ok {
			return chain, nil
		}
		next, err := hop.Lookup(nil, declaration.Base, resolve.WrapBaseUse)
		if err != nil {
			return nil, err
		}
		if seen[next.ID] {
			return nil, fmt.Errorf("wrapper base cycle at %s", next.ID)
		}
		seen[next.ID] = true
		chain = append(chain, next)
		if next.Kind == resolve.Wrapper {
			hop = c.world.Files[next.Source]
			if hop == nil {
				return nil, fmt.Errorf("wrapper base %s has no declaring file", next.ID)
			}
		}
	}
}

// wrapperCallDeps collects direct wrapper references from policy arms: named
// calls need the callee's calculated bound, and callable references need its
// contract. Calls through explicitly typed values are leaves and need no edge.
func wrapperCallDeps(file *resolve.File, declaration *syntax.WrapDecl) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(name syntax.QualifiedName, usage resolve.Usage) {
		symbol, err := file.Lookup(nil, name, usage)
		if err != nil || symbol.Kind != resolve.Wrapper || seen[symbol.ID] {
			return
		}
		seen[symbol.ID] = true
		ids = append(ids, symbol.ID)
	}
	var visit func(reflect.Value)
	visit = func(value reflect.Value) {
		if !value.IsValid() {
			return
		}
		if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			switch node := value.Interface().(type) {
			case *syntax.CallExpr:
				if name, ok := node.Invocation.Callee.(*syntax.NameExpr); ok {
					add(name.Name, resolve.CallUse)
				}
			case *syntax.ReferenceExpr:
				if name, ok := node.Callee.(*syntax.NameExpr); ok {
					add(name.Name, resolve.ReferenceUse)
				}
			}
			visit(value.Elem())
			return
		}
		switch value.Kind() {
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				visit(value.Field(i))
			}
		case reflect.Slice:
			for i := 0; i < value.Len(); i++ {
				visit(value.Index(i))
			}
		}
	}
	visit(reflect.ValueOf(declaration.Native))
	visit(reflect.ValueOf(declaration.Emitted))
	sort.Strings(ids)
	return ids
}

// boundCycleError is a calculated-bound dependency cycle. It carries the
// still-pending wrapper identities in the chain.
type boundCycleError struct{ involved []string }

func (e *boundCycleError) Error() string {
	return fmt.Sprintf("calculated-bound dependency cycle involving %s", strings.Join(e.involved, ", "))
}

// orderWrappers topologically sorts wrapper policies so every base wrapper
// and every directly called wrapper is computed before its callers.
// Calculated-bound dependency cycles fail here with the involved chain.
func orderWrappers(wrappers []*NativeDeclaration, deps map[string][]string) ([]*NativeDeclaration, error) {
	byID := map[string]*NativeDeclaration{}
	for _, native := range wrappers {
		byID[native.Symbol.ID] = native
	}
	pending := map[string]map[string]bool{}
	for _, native := range wrappers {
		edges := map[string]bool{}
		for _, dep := range deps[native.Symbol.ID] {
			if dep != native.Symbol.ID && byID[dep] != nil {
				edges[dep] = true
			}
		}
		pending[native.Symbol.ID] = edges
	}
	self := []string{}
	for _, native := range wrappers {
		for _, dep := range deps[native.Symbol.ID] {
			if dep == native.Symbol.ID {
				self = append(self, native.Symbol.ID)
			}
		}
	}
	if len(self) > 0 {
		sort.Strings(self)
		return nil, &boundCycleError{involved: self}
	}
	var ordered []*NativeDeclaration
	for len(pending) > 0 {
		var ready []string
		for id, edges := range pending {
			if len(edges) == 0 {
				ready = append(ready, id)
			}
		}
		if len(ready) == 0 {
			rest := make([]string, 0, len(pending))
			for id := range pending {
				rest = append(rest, id)
			}
			sort.Strings(rest)
			return nil, &boundCycleError{involved: rest}
		}
		sort.Strings(ready)
		for _, id := range ready {
			delete(pending, id)
			ordered = append(ordered, byID[id])
		}
		for _, edges := range pending {
			for _, id := range ready {
				delete(edges, id)
			}
		}
	}
	return ordered, nil
}

// checkWrapperPolicies resolves every wrapper base chain before execution,
// checks policy arms in dependency order and publishes each calculated
// signature for callers. Ordinary explicit-bound callees are leaves.
func (c *programChecker) checkWrapperPolicies(program *Program, callables map[string]CallableDeclaration) error {
	var wrappers []*NativeDeclaration
	byID := map[string]*NativeDeclaration{}
	for _, native := range program.Natives {
		if native.Symbol.Kind != resolve.Wrapper {
			continue
		}
		wrappers = append(wrappers, native)
		byID[native.Symbol.ID] = native
	}
	if len(wrappers) == 0 {
		return nil
	}
	deps := map[string][]string{}
	chains := map[string][]*resolve.Symbol{}
	for _, native := range wrappers {
		file := c.world.Files[native.Symbol.Source]
		declaration := native.Symbol.Declaration.(*syntax.WrapDecl)
		chain, err := c.wrapperChain(file, native.Symbol)
		if err != nil {
			return err
		}
		chains[native.Symbol.ID] = chain
		edges := []string{}
		if len(chain) > 1 && chain[1].Kind == resolve.Wrapper {
			edges = append(edges, chain[1].ID)
		}
		edges = append(edges, wrapperCallDeps(file, declaration)...)
		deps[native.Symbol.ID] = edges
	}
	ordered, err := orderWrappers(wrappers, deps)
	if err != nil {
		return err
	}
	for _, native := range ordered {
		if err := c.checkWrapper(program, native, chains[native.Symbol.ID], byID, callables); err != nil {
			return fmt.Errorf("wrapper %s: %w", native.Symbol.Name, err)
		}
	}
	return nil
}

// wrapperSets returns the original boundary sets for one wrapper: the raw
// native identities N, the emitted member types E, and the normalized
// request_failed member that default native rules forward.
func wrapperSets(root *NativeDeclaration) (native []string, emitted []*types.Type, failed *types.Type, err error) {
	native = append([]string(nil), root.Native...)
	// The E set stores canonical declarations; several exact
	// specializations may share one. Every signature member naming a
	// wanted declaration is an emitted key; each wanted declaration must
	// match at least one member.
	wanted := map[string]bool{}
	for _, identity := range root.Emitted {
		wanted[identity] = true
	}
	matched := map[string]bool{}
	for _, typ := range root.Signature.Errors() {
		if !wanted[typ.Declaration()] {
			continue
		}
		emitted = append(emitted, typ)
		matched[typ.Declaration()] = true
	}
	for identity := range wanted {
		if !matched[identity] {
			return nil, nil, nil, fmt.Errorf("emitted obligation has no signature member")
		}
	}
	for _, typ := range root.Signature.Errors() {
		if typ.Declaration() == "can.std.http@1::request_failed" {
			failed = typ
		}
	}
	if failed == nil {
		return nil, nil, nil, fmt.Errorf("wrapper root lacks its normalized failure member")
	}
	return native, emitted, failed, nil
}

// resolveArmKey resolves one policy arm against its origin set: an exact
// nominal specialization, or a bare generic name with exactly one concrete
// specialization in an emitted set per C5.1. Anything else is impossible.
func (c *programChecker) resolveArmKey(file *resolve.File, pattern *syntax.OutcomePattern, origin string, native []string, emitted []*types.Type) (*types.Type, error) {
	named, ok := pattern.Error.(*syntax.NamedType)
	if !ok {
		return nil, fmt.Errorf("policy arm matches one exact nominal error")
	}
	member := func(identity string) *types.Type {
		for _, typ := range emitted {
			if typ.Identity() == identity {
				return typ
			}
		}
		return nil
	}
	inNative := func(identity string) bool {
		for _, id := range native {
			if id == identity {
				return true
			}
		}
		return false
	}
	if len(named.Arguments) == 0 {
		symbol, err := file.Lookup(nil, named.Name, resolve.ErrorUse)
		if err != nil {
			return nil, err
		}
		if len(symbol.Parameters) == 0 {
			typ, err := c.annotation(file, named, false)
			if err != nil {
				return nil, err
			}
			switch origin {
			case WrapperNative:
				if !inNative(typ.Declaration()) {
					return nil, fmt.Errorf("impossible native key %s: not a raw obligation of the wrapped operation", typ.Declaration())
				}
			default:
				if member(typ.Identity()) == nil {
					return nil, fmt.Errorf("impossible emitted key %s: not a domain obligation of the wrapped operation", typ.Declaration())
				}
			}
			return typ, nil
		}
		if origin == WrapperNative {
			return nil, fmt.Errorf("impossible native key %s: raw obligations are never generic", symbol.ID)
		}
		var candidates []*types.Type
		for _, typ := range emitted {
			if typ.Declaration() == symbol.ID {
				candidates = append(candidates, typ)
			}
		}
		switch len(candidates) {
		case 0:
			return nil, fmt.Errorf("impossible emitted key %s: not a domain obligation of the wrapped operation", symbol.ID)
		case 1:
			return candidates[0], nil
		default:
			alternatives := specializationChoices(candidates)
			sort.Strings(alternatives)
			return nil, fmt.Errorf("ambiguous emitted key %s; write one of the exact specializations: %s", symbol.ID, strings.Join(alternatives, ", "))
		}
	}
	typ, err := c.annotation(file, named, false)
	if err != nil {
		return nil, err
	}
	switch origin {
	case WrapperNative:
		if !inNative(typ.Declaration()) {
			return nil, fmt.Errorf("impossible native key %s: not a raw obligation of the wrapped operation", typ.Declaration())
		}
	default:
		if member(typ.Identity()) == nil {
			return nil, fmt.Errorf("impossible emitted key %s: not a domain obligation of the wrapped operation", typ.Declaration())
		}
	}
	return typ, nil
}

func regionHasInherit(region *ir.Region) bool {
	var complete func(*ir.Completion) bool
	var block func(*ir.Block) bool
	complete = func(node *ir.Completion) bool {
		if node == nil {
			return false
		}
		if node.Kind == ir.InheritCompletion {
			return true
		}
		if node.Block != nil {
			if block(node.Block) {
				return true
			}
		}
		if node.Match != nil {
			for i := range node.Match.Arms {
				if complete(node.Match.Arms[i].Body) {
					return true
				}
			}
		}
		return false
	}
	block = func(node *ir.Block) bool {
		if node == nil {
			return false
		}
		return complete(node.Terminal)
	}
	if region == nil || region.Body == nil {
		return false
	}
	return block(region.Body)
}

// wrapperContext mirrors the native root context for policy arms: inherited
// inputs in scope, the inherited result, an open escaping bound while the
// calculated contract is still unknown, and the predecessor rule for inherit.
func (c *programChecker) wrapperContext(program *Program, native *NativeDeclaration, file *resolve.File, result *types.Type, params []ir.Local, inherit *InheritContext, scope *resolve.Scope, callables map[string]CallableDeclaration) CompletionContext {
	ctx := CompletionContext{Kind: ir.HandlerRegion, File: file.Source.Syntax.Source, Scope: scope, Result: result, Errors: OpenBound(), Registry: program.Registry, Expressions: c.expressions(file, scope), Callables: callables, Variadic: c.variadic, Raw: c.rawScope(file), Parameters: params, Inherit: inherit, Wrappers: wrapperPlans(program)}
	ctx.IntrinsicIdentity = func(scope *resolve.Scope, name syntax.QualifiedName) string {
		symbol, err := file.Lookup(scope, name, resolve.CallUse)
		if err != nil {
			return ""
		}
		return symbol.ID
	}
	ctx.CatalogueType = c.catalogueType
	ctx.InferCallback = func(scope *resolve.Scope, name syntax.QualifiedName, inputs []*types.Type, result *types.Type, e *Expressions) (ValueBinding, bool, error) {
		return c.inferCallback(file, scope, name, inputs, result, e)
	}
	ctx.Type = func(node syntax.TypeNode, allowVoid bool) (*types.Type, error) {
		return c.annotation(file, node, allowVoid)
	}
	ctx.ErrorName = func(name syntax.QualifiedName) (string, error) {
		symbol, err := file.Lookup(nil, name, resolve.ErrorUse)
		if err != nil {
			return "", err
		}
		return symbol.ID, nil
	}
	ctx.PatternName = func(name syntax.QualifiedName) (string, error) {
		symbol, err := file.Lookup(nil, name, resolve.TypeUse)
		if err != nil {
			return "", err
		}
		return symbol.ID, nil
	}
	ctx.Specialize = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
		return c.specialize(file, scope, name, args)
	}
	ctx.AggregateType = func(variant *types.Type) (*types.Type, error) { return c.aggregateType(file, variant) }
	ctx.ResolveMethod = func(application MethodApplication) (ValueBinding, error) { return c.method(file, application) }
	ctx.InferReference = func(scope *resolve.Scope, name syntax.QualifiedName, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
		return c.inferReference(file, scope, name, expected, e)
	}
	ctx.InferCall = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.Argument, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
		return c.inferCall(file, scope, name, args, expected, e)
	}
	return ctx
}

// checkWrapper checks one wrapper's local arms, resolves the effective rule
// for every original boundary key and publishes the calculated signature:
// the union of effective-rule escapes with per-member provenance.
func (c *programChecker) checkWrapper(program *Program, native *NativeDeclaration, chain []*resolve.Symbol, byID map[string]*NativeDeclaration, callables map[string]CallableDeclaration) error {
	symbol := native.Symbol
	file := c.world.Files[symbol.Source]
	declaration := symbol.Declaration.(*syntax.WrapDecl)
	rootSymbol := chain[len(chain)-1]
	root := c.rootNative(program, rootSymbol)
	if root == nil {
		return fmt.Errorf("wrapper root %s is not a checked operation", rootSymbol.ID)
	}
	header := syntax.NativeSignature(rootSymbol.Declaration)
	origin := c.world.Files[rootSymbol.Source]
	result, err := c.gather(origin, header.Result)
	if err != nil {
		return err
	}
	raw, emitted, failed, err := wrapperSets(root)
	if err != nil {
		return err
	}
	base := c.world.NativeScopes[declaration]
	if base == nil {
		base = resolve.NewScope(file.Scope)
	}
	params := make([]ir.Local, 0, len(native.Descriptor.Names))
	for _, name := range native.Descriptor.Names {
		id := symbol.ID + "/input/" + name
		typ := c.bindings[id]
		if typ == nil {
			return fmt.Errorf("wrapper input %s has no checked type", name)
		}
		params = append(params, ir.Local{Identity: id, Type: typ})
	}
	seen := map[WrapperKey]bool{}
	local := map[WrapperKey]WrapperRule{}
	locals := []WrapperKey{}
	tables := []struct {
		origin string
		arms   []syntax.WrapArm
	}{{WrapperNative, declaration.Native}, {WrapperEmitted, declaration.Emitted}}
	for _, table := range tables {
		for index, arm := range table.arms {
			keyType, err := c.resolveArmKey(file, arm.Pattern, table.origin, raw, emitted)
			if err != nil {
				return err
			}
			key := WrapperKey{Origin: table.origin, Identity: keyType.Identity(), Decl: keyType.Declaration()}
			if seen[key] {
				return fmt.Errorf("duplicate policy key %s %s", key.Origin, key.Identity)
			}
			seen[key] = true
			armID := fmt.Sprintf("%s/handles/%s/%d", symbol.ID, table.origin, index)
			binding := arm.Pattern.Alias
			bindingName := ""
			if binding != nil {
				bindingName = binding.Text
			} else if named, ok := arm.Pattern.Error.(*syntax.NamedType); ok {
				bindingName = named.Name.Name
			}
			if bindingName == "" {
				return fmt.Errorf("policy arm binds no error value")
			}
			armScope := resolve.NewScope(base)
			errID := armID + "/error"
			if err := armScope.Define(&resolve.Symbol{ID: errID, Name: bindingName, Kind: resolve.Value}); err != nil {
				return err
			}
			c.bindings[errID] = keyType
			alias := bindingName == "all_failed" && keyType.Declaration() == "can.prelude@1::all_failed" && binding == nil
			armParams := append(append([]ir.Local(nil), params...), ir.Local{Identity: errID, Type: keyType, ErrorAlias: alias})
			predecessor, err := c.predecessorRule(chain, byID, key, failed, keyType)
			if err != nil {
				return err
			}
			ctx := c.wrapperContext(program, native, file, result, armParams, &InheritContext{Region: predecessor.region, Escapes: predecessor.escapes}, armScope, callables)
			ctx.Sites = indexLexicalSites(armID, arm.Body)
			ctx.Identity = armID
			ctx.Parent = symbol.ID
			region, err := CheckRegion(ctx, syntax.Block{Span: arm.Body.BodySpan(), Terminal: arm.Body})
			if err != nil {
				return err
			}
			rule := WrapperRule{Key: key, Region: region, Escapes: append([]*types.Type(nil), region.Escapes...), Handler: symbol.ID}
			delegates := regionHasInherit(region)
			predSet := map[string]bool{}
			for _, typ := range predecessor.escapes {
				predSet[typ.Identity()] = true
			}
			for _, typ := range region.Escapes {
				entry := EscapeProvenance{Identity: typ.Identity(), Key: key, Handler: symbol.ID}
				if delegates && predSet[typ.Identity()] {
					entry.Chain = predecessor.chain
				}
				rule.From = append(rule.From, entry)
			}
			native.Regions = append(native.Regions, region)
			local[key] = rule
			locals = append(locals, key)
		}
	}
	plan := &WrapperPlan{Base: chain[1].ID, Root: rootSymbol.ID, Failed: failed.Identity(), Local: locals}
	// Native dispatch identities come from the arms that name them; keys
	// with no holder anywhere keep an empty identity and fall through to
	// the default at runtime.
	hexOf := map[string]string{}
	for key := range local {
		if key.Origin == WrapperNative {
			hexOf[key.Decl] = key.Identity
		}
	}
	for _, ancestor := range chain[1:] {
		holder := byID[ancestor.ID]
		if holder == nil || holder.Wrapper == nil {
			continue
		}
		for _, rule := range holder.Wrapper.Rules {
			if rule.Key.Origin == WrapperNative && rule.Key.Identity != "" && hexOf[rule.Key.Decl] == "" {
				hexOf[rule.Key.Decl] = rule.Key.Identity
			}
		}
	}
	ordered := make([]WrapperKey, 0, len(raw)+len(emitted))
	for _, identity := range raw {
		ordered = append(ordered, WrapperKey{Origin: WrapperNative, Identity: hexOf[identity], Decl: identity})
	}
	sorted := append([]*types.Type(nil), emitted...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Identity() < sorted[j].Identity() })
	for _, typ := range sorted {
		ordered = append(ordered, WrapperKey{Origin: WrapperEmitted, Identity: typ.Identity(), Decl: typ.Declaration()})
	}
	bound := map[string]*types.Type{}
	emittedByID := map[string]*types.Type{}
	for _, typ := range emitted {
		emittedByID[typ.Identity()] = typ
	}
	for _, key := range ordered {
		rule, err := c.effectiveRule(chain, byID, local, key, failed, emittedByID)
		if err != nil {
			return err
		}
		plan.Rules = append(plan.Rules, rule)
		for _, typ := range rule.Escapes {
			if bound[typ.Identity()] == nil {
				bound[typ.Identity()] = typ
			}
		}
		for _, entry := range rule.From {
			if !hasProvenance(plan.Provenance, entry) {
				plan.Provenance = append(plan.Provenance, entry)
			}
		}
	}
	identities := make([]string, 0, len(bound))
	for identity := range bound {
		identities = append(identities, identity)
	}
	sort.Strings(identities)
	for _, identity := range identities {
		plan.Bound = append(plan.Bound, bound[identity])
	}
	native.Wrapper = plan
	inputs := make([]*types.Type, 0, len(native.Descriptor.Names))
	for _, name := range native.Descriptor.Names {
		inputs = append(inputs, c.bindings[symbol.ID+"/input/"+name])
	}
	signature, err := types.CallableOfChecked(result, inputs, plan.Bound)
	if err != nil {
		return err
	}
	native.Signature = signature
	native.Descriptor.Contract = signature
	c.bindings[symbol.ID] = signature
	callables[symbol.ID] = native.Descriptor
	return nil
}

func (c *programChecker) rootNative(program *Program, root *resolve.Symbol) *NativeDeclaration {
	for _, native := range program.Natives {
		if native.Symbol == root {
			return native
		}
	}
	return nil
}

// predecessorRule resolves the rule an inherit delegates to for one key:
// the nearest base holder's local rule, or the origin default.
func (c *programChecker) predecessorRule(chain []*resolve.Symbol, byID map[string]*NativeDeclaration, key WrapperKey, failed, keyType *types.Type) (rule struct {
	region  string
	escapes []*types.Type
	chain   []string
}, err error) {
	for _, ancestor := range chain[1:] {
		holder := byID[ancestor.ID]
		if holder == nil || holder.Wrapper == nil {
			continue
		}
		if !holdsKey(holder.Wrapper, key) {
			continue
		}
		effective := holder.Wrapper.rule(holderKey(holder.Wrapper, key))
		if effective == nil {
			return rule, fmt.Errorf("holder %s has no effective rule for %s %s", ancestor.ID, key.Origin, key.Identity)
		}
		rule.region = effective.Region.ID
		rule.escapes = effective.Escapes
		// The delegation chain extends through the holder's own inherit
		// path below the selected holder.
		rule.chain = append([]string{ancestor.ID}, holderChain(holder.Wrapper, key)...)
		return rule, nil
	}
	rule.region = ""
	if key.Origin == WrapperNative {
		rule.escapes = []*types.Type{failed}
	} else {
		rule.escapes = []*types.Type{keyType}
	}
	rule.chain = []string{"default"}
	return rule, nil
}

// holdsKey reports whether one plan declares a local arm for a key. Native
// keys compare declarations: raw obligations are never generic. Emitted
// keys compare full specializations.
func holdsKey(plan *WrapperPlan, key WrapperKey) bool {
	for _, local := range plan.Local {
		if local.Origin != key.Origin {
			continue
		}
		if key.Origin == WrapperNative && local.Decl == key.Decl {
			return true
		}
		if key.Origin == WrapperEmitted && local == key {
			return true
		}
	}
	return false
}

// holderKey returns the holder's own local key for a lookup: its dispatch
// identity fills native lookups that carry none.
func holderKey(plan *WrapperPlan, key WrapperKey) WrapperKey {
	for _, local := range plan.Local {
		if local.Origin != key.Origin {
			continue
		}
		if key.Origin == WrapperNative && local.Decl == key.Decl {
			return local
		}
		if key.Origin == WrapperEmitted && local == key {
			return local
		}
	}
	return key
}

// holderChain returns the inherit path below one holder for one key: the
// longest chain recorded on the holder's local rule for that key.
func holderChain(plan *WrapperPlan, key WrapperKey) []string {
	rule := plan.rule(key)
	if rule == nil {
		return nil
	}
	var longest []string
	for _, entry := range rule.From {
		if entry.Key == key && len(entry.Chain) > len(longest) {
			longest = entry.Chain
		}
	}
	return append([]string(nil), longest...)
}

// effectiveRule resolves the rule one wrapper executes for one key: its own
// local arm, else the nearest base holder's local rule, else the default.
// Default rules carry an empty handler: the origin default runs.
func (c *programChecker) effectiveRule(chain []*resolve.Symbol, byID map[string]*NativeDeclaration, local map[WrapperKey]WrapperRule, key WrapperKey, failed *types.Type, emittedByID map[string]*types.Type) (WrapperRule, error) {
	if rule, ok := local[key]; ok {
		return rule, nil
	}
	for _, ancestor := range chain[1:] {
		holder := byID[ancestor.ID]
		if holder == nil || holder.Wrapper == nil {
			continue
		}
		if holdsKey(holder.Wrapper, key) {
			effective := holder.Wrapper.rule(holderKey(holder.Wrapper, key))
			if effective == nil {
				return WrapperRule{}, fmt.Errorf("holder %s has no effective rule for %s %s", ancestor.ID, key.Origin, key.Identity)
			}
			return *effective, nil
		}
	}
	rule := WrapperRule{Key: key, Default: true}
	if key.Origin == WrapperNative {
		rule.Escapes = []*types.Type{failed}
		rule.From = []EscapeProvenance{{Identity: failed.Identity(), Key: key, Chain: []string{}}}
		return rule, nil
	}
	keyType := emittedByID[key.Identity]
	if keyType == nil {
		return WrapperRule{}, fmt.Errorf("emitted default for %s has no key type", key.Identity)
	}
	rule.Escapes = []*types.Type{keyType}
	rule.From = []EscapeProvenance{{Identity: keyType.Identity(), Key: key, Chain: []string{}}}
	return rule, nil
}

// policyInjection checks one attached `using failure` row: the injected value
// is closed data in file scope, must typecheck against the exact original
// key set for its origin, cannot forge a standard failure and cannot smuggle
// an arbitrary transport phase string.
func (c *programChecker) policyInjection(file *resolve.File, native *NativeDeclaration, row syntax.Assertion) (WrapperKey, *ir.PolicyInjection, error) {
	mode := row.Mode.Failure
	origin := mode.Origin.Text
	value, err := c.expressions(file, file.Scope).Check(mode.Value, nil)
	if err != nil {
		return WrapperKey{}, nil, err
	}
	if value.Type.Kind() != types.Error {
		return WrapperKey{}, nil, fmt.Errorf("using failure injects an error value")
	}
	identity := value.Type.Identity()
	if value.Type.Declaration() == "can.prelude@1::standard_failure" {
		return WrapperKey{}, nil, fmt.Errorf("using failure cannot forge a standard failure")
	}
	switch origin {
	case WrapperNative:
		found := false
		for _, id := range native.Native {
			if id == value.Type.Declaration() {
				found = true
			}
		}
		if !found {
			return WrapperKey{}, nil, fmt.Errorf("injected failure %s is not a raw obligation of the wrapped operation", value.Type.Declaration())
		}
	default:
		// Emitted injection targets the original boundary, not the
		// calculated bound: fully replaced keys stay injectable.
		if native.Wrapper == nil {
			return WrapperKey{}, nil, fmt.Errorf("wrapper %s has no resolved policy", native.Symbol.Name)
		}
		found := false
		for _, rule := range native.Wrapper.Rules {
			if rule.Key.Origin == WrapperEmitted && rule.Key.Identity == identity {
				found = true
			}
		}
		if !found {
			return WrapperKey{}, nil, fmt.Errorf("injected failure %s is not a domain obligation of the wrapped operation", value.Type.Declaration())
		}
	}
	if value.Type.Declaration() == "can.std.http@1::transport_failed" {
		constructor, ok := mode.Value.(*syntax.ConstructorExpr)
		if !ok || len(constructor.Arguments) != 1 || constructor.Arguments[0].Spread {
			return WrapperKey{}, nil, fmt.Errorf("using failure transport requires its exact phase string")
		}
		literal, ok := constructor.Arguments[0].Value.(*syntax.LiteralExpr)
		if !ok || literal.Token.Kind != syntax.String {
			return WrapperKey{}, nil, fmt.Errorf("using failure transport requires its exact phase string")
		}
		switch literal.Token.Value {
		case "connect", "body", "protocol", "cancelled":
		default:
			return WrapperKey{}, nil, fmt.Errorf("unknown transport phase %q: write connect, body, protocol or cancelled", literal.Token.Value)
		}
	}
	return WrapperKey{Origin: origin, Identity: identity, Decl: value.Type.Declaration()}, &ir.PolicyInjection{Operation: native.Symbol.ID, Origin: origin, Identity: identity, Value: value}, nil
}

func hasProvenance(entries []EscapeProvenance, entry EscapeProvenance) bool {
	for _, prior := range entries {
		if prior.Identity != entry.Identity || prior.Key != entry.Key || prior.Handler != entry.Handler || len(prior.Chain) != len(entry.Chain) {
			continue
		}
		match := true
		for i := range prior.Chain {
			if prior.Chain[i] != entry.Chain[i] {
				match = false
			}
		}
		if match {
			return true
		}
	}
	return false
}

// wrapperNote renders the policy provenance behind one escaping calculated
// member when the invocation is a direct wrapper call.
func (c *regionChecker) wrapperNote(call *ir.Invocation, hex, display string) string {
	if call == nil || len(call.Steps) != 1 {
		return ""
	}
	plan := c.context.Wrappers[call.Steps[0].Identity]
	if plan == nil {
		return ""
	}
	provenance := plan.provenance(hex)
	if provenance == nil {
		return ""
	}
	handler := provenance.Handler
	if handler == "" {
		handler = "default"
	}
	note := fmt.Sprintf("; wrapper %s escapes %s for %s key %s via %s", call.Steps[0].Identity, display, provenance.Key.Origin, provenance.Key.Decl, handler)
	if len(provenance.Chain) != 0 {
		note += " inherit " + strings.Join(provenance.Chain, " -> ")
	}
	return note
}

// relayWrapperNote annotates a relay escaping failure with wrapper policy
// provenance when the relayed call is a direct wrapper invocation.
func (c *regionChecker) relayWrapperNote(call *ir.Invocation, bound ErrorBound, err error) error {
	if err == nil || call == nil || len(call.Steps) != 1 || c.context.Wrappers[call.Steps[0].Identity] == nil {
		return err
	}
	allowed := map[string]bool{}
	for _, entry := range c.context.Errors.Entries() {
		allowed[entry.TypeIdentity] = true
	}
	for _, entry := range bound.Entries() {
		if !allowed[entry.TypeIdentity] {
			display := entry.Declaration.Name
			if len(entry.Arguments) != 0 {
				display = entry.TypeIdentity
			}
			if note := c.wrapperNote(call, entry.TypeIdentity, display); note != "" {
				return fmt.Errorf("%s%s", err.Error(), note)
			}
			return err
		}
	}
	return err
}

func wrapperPlans(program *Program) map[string]*WrapperPlan {
	plans := map[string]*WrapperPlan{}
	for _, native := range program.Natives {
		if native.Wrapper != nil {
			plans[native.Symbol.ID] = native.Wrapper
		}
	}
	return plans
}

package check

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Program contains only sealed types and checked bodies. Source files retain
// their own import scopes even when they contribute to the same flat package.
type Program struct {
	Natives      []*NativeDeclaration
	Connections  map[string]ConnectionPolicy
	World        *resolve.World
	Model        *types.Model
	Registry     *ErrorRegistry
	Functions    []*ProgramFunction
	Initializers []ir.Initializer
	Entry        *ProgramFunction
	Intrinsics   map[string]*types.Type
	Collections  map[string]*CollectionSpecialization
	Codecs       map[string]*CodecSpecialization
	Assertions   []*ir.Assertion
}
type ProgramFunction struct {
	Symbol        *resolve.Symbol
	Region        *ir.Region
	Instance      string
	TypeArguments []*types.Type
	Parameters    map[string]*types.Type
	Requests      []string
}

func (f *ProgramFunction) Identity() string {
	if f.Instance != "" {
		return f.Instance
	}
	return f.Symbol.ID
}

type programChecker struct {
	program     *Program
	specializer *types.Specializer
	instances   map[string]*ProgramFunction
	callables   map[string]CallableDeclaration
	current     *ProgramFunction
	codecs      map[string]*CodecSpecialization
	codecParts  map[string][]*types.Type
	world       *resolve.World
	builder     *types.Builder
	annotations map[*resolve.File]map[string]*types.Type
	bindings    map[string]*types.Type
	variadic    map[string]bool
}

func (c *programChecker) gather(file *resolve.File, node syntax.TypeNode) (*types.Type, error) {
	if c.specializer != nil {
		return c.specializer.Resolve(file, node, c.parameters(), true)
	}
	key := syntax.FormatType(node)
	if t := c.annotations[file][key]; t != nil {
		return t, nil
	}
	t, err := c.builder.Resolve(file, node, nil, true)
	if err != nil {
		return nil, err
	}
	c.annotations[file][key] = t
	return t, nil
}
func (c *programChecker) annotation(file *resolve.File, node syntax.TypeNode, allowVoid bool) (*types.Type, error) {
	if c.specializer != nil && (c.current != nil && c.current.Instance != "" || c.annotations[file][syntax.FormatType(node)] == nil) {
		return c.specializer.Resolve(file, node, c.parameters(), allowVoid)
	}
	t := c.annotations[file][syntax.FormatType(node)]
	if t == nil {
		return nil, fmt.Errorf("concrete type was not gathered: %s", syntax.FormatType(node))
	}
	if !allowVoid && t.Kind() == types.Void {
		return nil, fmt.Errorf("void is not a data type")
	}
	return t, nil
}
func named(name string) syntax.TypeNode {
	return &syntax.NamedType{Name: syntax.QualifiedName{Name: name}}
}

// gatherBody walks syntax data only. It does not resolve expression names as
// types or evaluate expressions. Synthetic constructor annotations must be
// gathered too; local pattern binders are deliberately excluded from lookup.
func (c *programChecker) gatherBody(file *resolve.File, value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		if node, ok := value.Interface().(syntax.TypeNode); ok {
			_, err := c.gather(file, node)
			return err
		}
		switch node := value.Interface().(type) {
		case *syntax.CallExpr:
			if err := c.gatherCodec(file, node.Invocation.Callee, node.Invocation.Types); err != nil {
				return err
			}
		case *syntax.ReferenceExpr:
			if err := c.gatherCodec(file, node.Callee, node.Types); err != nil {
				return err
			}
		}
		if constructor, ok := value.Interface().(*syntax.ConstructorExpr); ok {
			symbol, err := file.Lookup(nil, constructor.Name, resolve.ConstructorUse)
			if err != nil {
				return err
			}
			if len(constructor.Types) != 0 || len(symbol.Parameters) == 0 {
				if _, err = c.gather(file, &syntax.NamedType{Name: constructor.Name, Arguments: constructor.Types}); err != nil {
					return err
				}
			}
		}
		var name *syntax.QualifiedName
		var arguments []syntax.TypeNode
		switch node := value.Interface().(type) {
		case *syntax.NamePattern:
			name = &node.Name
			arguments = node.Types
		case *syntax.ConstructorPattern:
			name = &node.Name
			arguments = node.Types
		case *syntax.OutcomePattern:
			// A bare generic error arm selects a specialization from the call's bound.
			if n, ok := node.Error.(*syntax.NamedType); ok && len(n.Arguments) == 0 {
				symbol, err := file.Lookup(nil, n.Name, resolve.ErrorUse)
				if err == nil && len(symbol.Parameters) != 0 {
					return c.gatherBody(file, reflect.ValueOf(node.Binding))
				}
			}
		}
		if name != nil {
			symbol, err := file.Lookup(nil, *name, resolve.TypeUse)
			if err == nil && (len(arguments) != 0 || len(symbol.Parameters) == 0) {
				if _, err = c.gather(file, &syntax.NamedType{Name: *name, Arguments: arguments}); err != nil {
					return err
				}
			}
		}
		return c.gatherBody(file, value.Elem())
	}
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if err := c.gatherBody(file, value.Field(i)); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if err := c.gatherBody(file, value.Index(i)); err != nil {
				return err
			}
		}
	}
	return nil
}
func (c *programChecker) expressions(file *resolve.File, scope *resolve.Scope) *Expressions {
	lookup := func(name syntax.QualifiedName, use resolve.Usage) (ValueBinding, error) {
		symbol, err := file.Lookup(scope, name, use)
		if err != nil {
			return ValueBinding{}, err
		}
		identity := c.bindingIdentity(symbol.ID)
		typ := c.bindings[identity]
		if typ == nil {
			return ValueBinding{}, fmt.Errorf("%s requires an unimplemented specialization or intrinsic lowering", symbol.ID)
		}
		return ValueBinding{Identity: identity, Type: typ}, nil
	}
	return &Expressions{Scalars: c.annotations[file],
		Value:     func(name syntax.QualifiedName) (ValueBinding, error) { return lookup(name, resolve.ValueUse) },
		Reference: func(name syntax.QualifiedName) (ValueBinding, error) { return lookup(name, resolve.ReferenceUse) },
		Function:  func(name syntax.QualifiedName) (ValueBinding, error) { return lookup(name, resolve.CallUse) },
		Constructor: func(node *syntax.ConstructorExpr, expected *types.Type, expressions *Expressions) (*types.Type, error) {
			symbol, err := file.Lookup(nil, node.Name, resolve.ConstructorUse)
			if err != nil {
				return nil, err
			}
			if len(node.Types) == 0 && len(symbol.Parameters) != 0 {
				return c.inferConstructor(symbol, node, expected, expressions)
			}
			return c.annotation(file, &syntax.NamedType{Name: node.Name, Arguments: node.Types}, false)
		},
	}
}

// CheckProgram connects project resolution, declaration checking, initialization
// ordering and completion regions without using the superseded compiler passes.
func CheckProgram(graph *project.Graph) (*Program, error)          { return checkProgram(graph, true) }
func CheckAssertionProgram(graph *project.Graph) (*Program, error) { return checkProgram(graph, false) }
func checkProgram(graph *project.Graph, requireEntry bool) (*Program, error) {
	world, err := resolve.Build(graph)
	if err != nil {
		return nil, err
	}
	if _, err = types.CheckDeclarations(world); err != nil {
		return nil, err
	}
	registry, err := ErrorDeclarations(world)
	if err != nil {
		return nil, err
	}
	c := &programChecker{world: world, builder: types.NewBuilder(world), annotations: map[*resolve.File]map[string]*types.Type{}, bindings: map[string]*types.Type{}, variadic: map[string]bool{}}
	if err = c.builder.SeedDeclarations(); err != nil {
		return nil, err
	}
	p := &Program{World: world, Registry: registry, Intrinsics: map[string]*types.Type{}, Connections: map[string]ConnectionPolicy{}}
	// Resolve maintained contracts from the catalogue in a private canonical scope.
	// Authored calls still require their declaring file's explicit imports.
	builtinFile := &resolve.File{Scope: world.Prelude, Imports: world.Packages}
	c.annotations[builtinFile] = map[string]*types.Type{}
	for _, op := range catalogue.Builtin().Inventory().Operations {
		if op.Lowering.Task != "I22" && op.Lowering.Task != "I23" && op.Lowering.Task != "I24" && !strings.HasPrefix(op.Name, "bytes::") && op.Name != "io::stdout_write" && op.Name != "io::stderr_write" {
			continue
		}
		signature := &syntax.CallableType{}
		parse := func(text string) (syntax.TypeNode, error) {
			src, e := source.New("can:catalogue", text)
			if e != nil {
				return nil, e
			}
			node, ds := syntax.ParseType(src)
			if len(ds) != 0 {
				return nil, fmt.Errorf("invalid catalogue signature: %v", ds)
			}
			return node, nil
		}
		signature.Result, err = parse(op.Result)
		if err != nil {
			return nil, err
		}
		if op.Receiver != "" {
			receiver, e := parse(op.Receiver)
			if e != nil {
				return nil, e
			}
			signature.Inputs = append(signature.Inputs, receiver)
		}
		for _, input := range op.Inputs {
			node, e := parse(input.Type)
			if e != nil {
				return nil, e
			}
			signature.Inputs = append(signature.Inputs, node)
		}
		for _, failure := range op.Emits {
			node, e := parse(failure)
			if e != nil {
				return nil, e
			}
			signature.Errors.Types = append(signature.Errors.Types, node)
		}
		typ, e := c.gather(builtinFile, signature)
		if e != nil {
			return nil, e
		}
		c.bindings[op.Identity] = typ
		p.Intrinsics[op.Identity] = typ
	}
	var files []*resolve.File
	for _, file := range world.Files {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Source.ID < files[j].Source.ID })
	for _, file := range files {
		c.annotations[file] = map[string]*types.Type{}
		for _, name := range []string{"int", "float", "str", "bool", "void"} {
			if _, err = c.gather(file, named(name)); err != nil {
				return nil, err
			}
		}
		for _, declaration := range file.Source.Syntax.Declarations {
			switch d := declaration.(type) {
			case *syntax.FetchDecl, *syntax.LLMDecl, *syntax.JudgeDecl, *syntax.QuestionDecl, *syntax.ChoiceArmDecl:
				native, e := c.gatherNative(file, declaration)
				if e != nil {
					return nil, e
				}
				p.Natives = append(p.Natives, native)
			case *syntax.ConnectionDecl:
				policy, e := ConnectionDeclaration(d)
				if e != nil {
					return nil, e
				}
				p.Connections[file.Package.Scope.Symbols[d.Name.Text].ID] = policy
			case *syntax.FunctionDecl:
				if err = validateAssertionNames(d); err != nil {
					return nil, err
				}
				symbol := file.Package.Scope.Symbols[d.Name.Text]
				if d.Name.Text == "main" && file.Source.Package.Owner == graph.Root {
					if p.Entry != nil {
						return nil, fmt.Errorf("multiple root main declarations")
					}
					if len(d.Parameters) != 0 || d.Receiver != nil || len(d.Inputs) != 1 || d.Inputs[0].Near || d.Inputs[0].Variadic || syntax.FormatType(d.Result) != "void" || syntax.FormatType(d.Inputs[0].Type) != "str[]" {
						return nil, fmt.Errorf("entry must be a non-generic void main(str[] args)")
					}
					p.Entry = &ProgramFunction{Symbol: symbol}
				}
				if len(d.Parameters) != 0 {
					continue
				} // I46 owns reachable concrete specialization.
				signature := &syntax.CallableType{Result: d.Result, Errors: d.Errors}
				var fields []syntax.Field
				if d.Receiver != nil {
					fields = append(fields, *d.Receiver)
				}
				for i, input := range d.Inputs {
					field := input.Field
					if input.Variadic {
						if i != len(d.Inputs)-1 {
							return nil, fmt.Errorf("variadic input must be last")
						}
						field.Type = &syntax.ArrayType{Element: field.Type}
						c.variadic[symbol.ID] = true
					}
					fields = append(fields, field)
				}
				for _, field := range fields {
					signature.Inputs = append(signature.Inputs, field.Type)
					typ, e := c.gather(file, field.Type)
					if e != nil {
						return nil, e
					}
					c.bindings[symbol.ID+"/input/"+field.Name.Text] = typ
				}
				typ, e := c.gather(file, signature)
				if e != nil {
					return nil, e
				}
				c.bindings[symbol.ID] = typ
				if err = c.gatherBody(file, reflect.ValueOf(d.Assertions)); err != nil {
					return nil, err
				}
				if err = c.gatherBody(file, reflect.ValueOf(d.Body)); err != nil {
					return nil, err
				}
				fn := &ProgramFunction{Symbol: symbol}
				if p.Entry != nil && p.Entry.Symbol == symbol {
					fn = p.Entry
				}
				p.Functions = append(p.Functions, fn)
			case *syntax.ValueDecl:
				typ, e := c.gather(file, d.Binding.Type)
				if e != nil {
					return nil, e
				}
				c.bindings[file.Package.Scope.Symbols[d.Binding.Name.Text].ID] = typ
				if err = c.gatherBody(file, reflect.ValueOf(d.Binding.Value)); err != nil {
					return nil, err
				}
			}
		}
	}
	if requireEntry && p.Entry == nil {
		return nil, fmt.Errorf("root project requires void main(str[] args)")
	}
	p.Model, err = c.builder.Finish()
	if err != nil {
		return nil, err
	}
	c.program = p
	c.instances = map[string]*ProgramFunction{}
	c.specializer, err = types.NewSpecializer(world, p.Model)
	if err != nil {
		return nil, err
	}
	if err = c.checkNativeContracts(p); err != nil {
		return nil, err
	}
	p.Codecs = c.codecs
	for id, special := range c.codecs {
		parts := c.codecParts[id]
		special.Schema, err = types.Schema(special.Data)
		if err != nil {
			return nil, err
		}
		special.Contract, err = types.CallableOfChecked(parts[0], []*types.Type{parts[1]}, []*types.Type{parts[2]})
		if err != nil {
			return nil, err
		}
		p.Intrinsics[id] = special.Contract
	}
	callables := map[string]CallableDeclaration{}
	c.callables = callables
	for _, native := range p.Natives {
		callables[native.Symbol.ID] = native.Descriptor
	}
	for id, typ := range p.Intrinsics {
		names := make([]string, len(typ.Inputs()))
		for i := range names {
			names[i] = fmt.Sprintf("input%d", i)
		}
		receiver := false
		for _, op := range catalogue.Builtin().Inventory().Operations {
			if op.Identity == id {
				receiver = op.Kind == "method"
				break
			}
		}
		callables[id] = CallableDeclaration{Kind: "function", Contract: typ, Names: names, Near: make([]bool, len(names)), Receiver: receiver}
	}
	for _, fn := range p.Functions {
		d := fn.Symbol.Declaration.(*syntax.FunctionDecl)
		descriptor := CallableDeclaration{Kind: "function", Contract: c.bindings[fn.Symbol.ID], Receiver: d.Receiver != nil}
		if d.Receiver != nil {
			descriptor.Names = append(descriptor.Names, d.Receiver.Name.Text)
			descriptor.Near = append(descriptor.Near, false)
		}
		for _, input := range d.Inputs {
			descriptor.Names = append(descriptor.Names, input.Name.Text)
			descriptor.Near = append(descriptor.Near, input.Near)
		}
		callables[fn.Symbol.ID] = descriptor
	}
	if err = c.checkNativeBodies(p, callables); err != nil {
		return nil, err
	}
	if err = c.genericAssertions(files); err != nil {
		return nil, err
	}
	for index := 0; index < len(p.Functions); index++ {
		fn := p.Functions[index]
		c.current = fn
		symbol := fn.Symbol
		file := world.Files[symbol.Source]
		d := symbol.Declaration.(*syntax.FunctionDecl)
		if fn.Instance != "" {
			if err = c.gatherBody(file, reflect.ValueOf(d.Body)); err != nil {
				return nil, fmt.Errorf("specialization %s requested at %s: %w", fn.Identity(), strings.Join(fn.Requests, "; "), err)
			}
		}
		context, e := c.functionContext(fn)
		if e != nil {
			return nil, e
		}
		if fn.Instance == "" {
			assertions, e := c.assertions(file, fn, context)
			if e != nil {
				return nil, e
			}
			p.Assertions = append(p.Assertions, assertions...)
		}
		fn.Region, err = CheckRegion(context, d.Body)
		if err != nil {
			if fn.Instance != "" {
				return nil, fmt.Errorf("concrete function %s (generic source %s, requested at %s): %w", fn.Identity(), symbol.Source.Syntax.Source.Name(), strings.Join(fn.Requests, "; "), err)
			}
			return nil, fmt.Errorf("concrete function %s: %w", fn.Identity(), err)
		}
	}
	c.current = nil
	var initial []InitialValue
	for _, file := range files {
		for _, declaration := range file.Source.Syntax.Declarations {
			if d, ok := declaration.(*syntax.ValueDecl); ok {
				id := file.Package.Scope.Symbols[d.Binding.Name.Text].ID
				initial = append(initial, InitialValue{Identity: id, QualifiedName: id, Source: file.Source.ID, Binding: d.Binding, Type: c.bindings[id], Checker: c.expressions(file, file.Scope)})
			}
		}
	}
	for _, native := range p.Natives {
		if native.ArmDescription != nil {
			file := c.world.Files[native.Symbol.Source]
			d := native.Symbol.Declaration.(*syntax.ChoiceArmDecl)
			initial = append(initial, InitialValue{Identity: native.Symbol.ID, QualifiedName: native.Symbol.ID, Source: file.Source.ID, Binding: syntax.Binding{Value: d.Description}, Type: c.bindings[native.Symbol.ID], Checker: c.expressions(file, file.Scope), ArmDescription: native.ArmDescription})
		}
	}
	p.Initializers, err = Initialization(initial, nil)
	if err != nil {
		return nil, err
	}
	p.Model = c.specializer.Model()
	return p, nil
}

func (c *programChecker) functionContext(fn *ProgramFunction) (CompletionContext, error) {
	symbol := fn.Symbol
	file := c.world.Files[symbol.Source]
	d := symbol.Declaration.(*syntax.FunctionDecl)
	signature := c.bindings[fn.Identity()]
	scope := c.world.Functions[d]
	bound, e := c.program.Registry.Bound(signature.Errors())
	if e != nil {
		return CompletionContext{}, e
	}
	context := CompletionContext{Sites: indexLexicalSites(symbol.ID, d), Identity: fn.Identity(), Kind: ir.FunctionRegion, File: file.Source.Syntax.Source, Scope: scope, Result: signature.Result(), Errors: bound, Registry: c.program.Registry, Expressions: c.expressions(file, scope), Variadic: c.variadic, Callables: c.callables}

	context.IntrinsicIdentity = func(scope *resolve.Scope, name syntax.QualifiedName) string {
		symbol, err := file.Lookup(scope, name, resolve.CallUse)
		if err != nil {
			return ""
		}
		return symbol.ID
	}
	context.CatalogueType = c.catalogueType
	context.InferCallback = func(scope *resolve.Scope, name syntax.QualifiedName, inputs []*types.Type, result *types.Type, e *Expressions) (ValueBinding, bool, error) {
		return c.inferCallback(file, scope, name, inputs, result, e)
	}
	context.Specialize = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
		return c.specialize(file, scope, name, args)
	}
	context.AggregateType = func(variant *types.Type) (*types.Type, error) { return c.aggregateType(file, variant) }
	context.ResolveMethod = func(application MethodApplication) (ValueBinding, error) { return c.method(file, application) }
	context.InferReference = func(scope *resolve.Scope, name syntax.QualifiedName, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
		return c.inferReference(file, scope, name, expected, e)
	}
	context.InferCall = func(scope *resolve.Scope, name syntax.QualifiedName, args []syntax.Argument, expected *types.Type, e *Expressions) (ValueBinding, bool, error) {
		return c.inferCall(file, scope, name, args, expected, e)
	}
	context.Type = func(node syntax.TypeNode, allowVoid bool) (*types.Type, error) {
		return c.annotation(file, node, allowVoid)
	}
	context.ErrorName = func(name syntax.QualifiedName) (string, error) {
		s, e := file.Lookup(nil, name, resolve.ErrorUse)
		if e != nil {
			return "", e
		}
		return s.ID, nil
	}
	context.PatternName = func(name syntax.QualifiedName) (string, error) {
		s, e := file.Lookup(nil, name, resolve.TypeUse)
		if e != nil {
			return "", e
		}
		return s.ID, nil
	}
	var names []string
	if d.Receiver != nil {
		names = append(names, d.Receiver.Name.Text)
	}
	for _, input := range d.Inputs {
		names = append(names, input.Name.Text)
	}
	for _, name := range names {
		id := fn.Identity() + "/input/" + name
		context.Parameters = append(context.Parameters, ir.Local{Identity: id, Type: c.bindings[id]})
	}
	return context, nil
}

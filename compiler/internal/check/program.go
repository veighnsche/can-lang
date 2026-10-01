package check

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/editortrace"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// Target selects the entry shape a checked program enforces. Bun keeps the
// historical void main(str[] args); the browser profile requires an exported
// non-generic void main with no arguments and emits {}. The checker retains
// full source checks for both targets; the emitter prunes production output
// by reachability.
type Target string

const (
	// TargetBun is the default server profile executed by Bun.
	TargetBun Target = "bun"
	// TargetBrowser is the main-thread browser profile.
	TargetBrowser Target = "browser"
)

// Program contains only sealed types and checked bodies. Source files retain
// their own import scopes even when they contribute to the same flat package.
type Program struct {
	Target        Target
	Natives       []*NativeDeclaration
	Connections   map[string]ConnectionPolicy
	World         *resolve.World
	Model         *types.Model
	Registry      *ErrorRegistry
	Functions     []*ProgramFunction
	Initializers  []ir.Initializer
	Entry         *ProgramFunction
	Intrinsics    map[string]*types.Type
	Collections   map[string]*CollectionSpecialization
	BrowserStates map[string]*BrowserStateSpecialization
	Streams       map[string]*StreamSpecialization
	Codecs        map[string]*CodecSpecialization
	HTTPs         map[string]*HTTPSpecialization
	Forms         map[string]*FormSpecialization
	Fetches       map[string]*FetchSpecialization
	Actions       []*ActionDeclaration
	// ActionRoutes and ActionClient record symbol-based action consumer
	// use: mount/url need the server route state object, request/post
	// need the client fetch state object. The emitter initializes only
	// the selected factories.
	ActionRoutes bool
	ActionClient bool
	SQLs         map[string]*SQLSpecialization
	Transactions map[string]*TransactionSpecialization
	Assertions   []*ir.Assertion
	Assets       []project.Asset
	SQL          []ir.SQLDescriptor
	// Warnings carries advisory findings in deterministic order. It never
	// fails checking, emission, build or assert; drivers print it and
	// editor bridges map it to warning severity.
	Warnings []Warning
}
type ProgramFunction struct {
	Symbol *resolve.Symbol
	// Signature is the sealed callable contract proved independently of the
	// body. A nonnil Signature does not prove executable or assertion validity.
	// Source consumers must not combine contracts of distinct generic instances.
	Signature     *types.Type
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
	recovering  bool
	program     *Program
	specializer *types.Specializer
	instances   map[string]*ProgramFunction
	callables   map[string]CallableDeclaration
	current     *ProgramFunction
	codecs      map[string]*CodecSpecialization
	codecParts  map[string][]*types.Type
	https       map[string]*HTTPSpecialization
	httpParts   map[string]*httpParts
	forms       map[string]*FormSpecialization
	fetches     map[string]*FetchSpecialization
	sqlSites    []SQLSiteRecord
	world       *resolve.World
	builder     *types.Builder
	annotations map[*resolve.File]map[string]*types.Type
	// Exact failed ordinary-body annotation nodes are recorded before sealing.
	// This immutable failure map never carries generic instance failures.
	annotationFailures  map[*resolve.File]map[syntax.TypeNode]error
	constructorFailures map[*syntax.ConstructorExpr]error
	bindings            map[string]*types.Type
	variadic            map[string]bool
	templates           map[string]*Template
	// symbolicComponent maps a public generic declaration identity to its
	// dependency component during exported-generic proof. symbolicProofs
	// holds only committed components; nothing is committed until every
	// member body and internal call succeeds. symbolicScan records the
	// syntactic call edges the component graph was built from, for cycle
	// diagnostics. All three are nil outside checkExportedGenerics.
	symbolicComponent map[string]int
	symbolicProofs    map[string]bool
	symbolicScan      []symbolicScanEdge
	warnings          []Warning
}

// warn collects one advisory finding into the program report. Regions
// checked more than once (symbolic bodies, provisional inference) report
// duplicates that collapse on identity.
func (c *programChecker) warn(warning Warning) {
	c.warnings = collectWarnings(c.warnings, warning)
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
	if c.current == nil || c.current.Instance == "" {
		if c.annotationFailures[file][node] != nil {
			return nil, &source.BlockedError{Dependency: fmt.Sprintf("%s:annotation@%d", file.Source.Path, node.TypeSpan().Start)}
		}
	}
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
	return c.gatherBodyMode(file, value, false)
}

// Only ordinary function gathering before model sealing may retain failed
// annotation nodes. Other phases keep their specialization-specific boundaries.
func (c *programChecker) gatherBodyMode(file *resolve.File, value reflect.Value, recovering bool) error {
	recovering = recovering && c.recovering && c.specializer == nil
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		if node, ok := value.Interface().(syntax.TypeNode); ok {
			_, err := c.gather(file, node)
			if err != nil && recovering {
				err = locateGather(file, node.TypeSpan(), err)
				if c.annotationFailures == nil {
					c.annotationFailures = map[*resolve.File]map[syntax.TypeNode]error{}
				}
				if c.annotationFailures[file] == nil {
					c.annotationFailures[file] = map[syntax.TypeNode]error{}
				}
				c.annotationFailures[file][node] = err
			}
			return err
		}
		switch node := value.Interface().(type) {
		case *syntax.CallExpr:
			if err := c.gatherCodec(file, node, node.Invocation.Callee, node.Invocation.Types); err != nil {
				return locateGather(file, node.ExprSpan(), err)
			}
			if err := c.gatherHTTP(file, node.Invocation.Callee, node.Invocation.Types); err != nil {
				return err
			}
			if err := c.gatherForm(file, node, node.Invocation.Callee, node.Invocation.Types); err != nil {
				return err
			}
			if err := c.gatherFetch(file, node, node.Invocation.Callee, node.Invocation.Types); err != nil {
				return err
			}
		case *syntax.ReferenceExpr:
			if err := c.gatherCodec(file, node, node.Callee, node.Types); err != nil {
				return locateGather(file, node.ExprSpan(), err)
			}
			if err := c.gatherHTTP(file, node.Callee, node.Types); err != nil {
				return err
			}
			if err := c.gatherForm(file, node, node.Callee, node.Types); err != nil {
				return err
			}
			if err := c.gatherFetch(file, node, node.Callee, node.Types); err != nil {
				return err
			}
		}
		if constructor, ok := value.Interface().(*syntax.ConstructorExpr); ok {
			symbol, err := file.Lookup(nil, constructor.Name, resolve.ConstructorUse)
			if err == nil && (len(constructor.Types) != 0 || len(symbol.Parameters) == 0) {
				_, err = c.gather(file, &syntax.NamedType{Span: constructor.Name.Span, Name: constructor.Name, Arguments: constructor.Types})
			}
			if err != nil {
				err = locateGather(file, constructor.Name.Span, err)
				if recovering {
					if c.constructorFailures == nil {
						c.constructorFailures = map[*syntax.ConstructorExpr]error{}
					}
					c.constructorFailures[constructor] = err
				}
				return err
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
			if c.bareGenericErrorHead(file, node.Error) {
				return c.gatherBodyMode(file, reflect.ValueOf(node.Binding), recovering)
			}
		}
		if name != nil {
			symbol, err := file.Lookup(nil, *name, resolve.TypeUse)
			if err == nil && (len(arguments) != 0 || len(symbol.Parameters) == 0) {
				if _, err = c.gather(file, &syntax.NamedType{Span: name.Span, Name: *name, Arguments: arguments}); err != nil {
					return err
				}
			}
		}
		return c.gatherBodyMode(file, value.Elem(), recovering)
	}
	// MatchArm.AlternateOutcomes stores value elements, so grouped bare
	// generic heads arrive here as structs rather than through the pointer
	// exception above. They select their specialization from the call's
	// bound exactly like the first head; only the optional binding is
	// gathered. The shared group body still appears once in syntax and is
	// visited once.
	if value.Kind() == reflect.Struct && value.CanInterface() {
		if pattern, ok := value.Interface().(syntax.OutcomePattern); ok && c.bareGenericErrorHead(file, pattern.Error) {
			return c.gatherBodyMode(file, reflect.ValueOf(pattern.Binding), recovering)
		}
	}
	var problems []error
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if err := c.gatherBodyMode(file, value.Field(i), recovering); err != nil {
				if !recovering {
					return err
				}
				problems = append(problems, err)
			}
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if err := c.gatherBodyMode(file, value.Index(i), recovering); err != nil {
				if !recovering {
					return err
				}
				problems = append(problems, err)
			}
		}
	}
	return errors.Join(problems...)
}

// bareGenericErrorHead reports whether node names a generic error without
// specialization arguments. Such heads resolve against the call's bound at
// checking time and must not be gathered as concrete types.
func (c *programChecker) bareGenericErrorHead(file *resolve.File, node syntax.TypeNode) bool {
	named, ok := node.(*syntax.NamedType)
	if !ok || len(named.Arguments) != 0 {
		return false
	}
	symbol, err := file.Lookup(nil, named.Name, resolve.ErrorUse)
	return err == nil && len(symbol.Parameters) != 0
}

// locateGather attaches the gathering file's use-site span to a constructor
// or codec discovery failure, so the pre-pass diagnosis points at the
// offending expression instead of surfacing bare.
func locateGather(file *resolve.File, span source.Span, err error) error {
	if file == nil || file.Source == nil {
		return err
	}
	return source.Locate(file.Source.Path, span, err)
}

// expressionPackage names the checking file's package for owner-record
// confinement. Synthetic catalogue-only files carry no package and fail
// closed against owner representation.
func expressionPackage(file *resolve.File) string {
	if file == nil || file.Package == nil {
		return ""
	}
	return file.Package.ID
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
	expressions := &Expressions{Recover: c.recovering, Checkpoint: c.unitCheckpoint, Scalars: c.annotations[file], Package: expressionPackage(file),
		Value:     func(name syntax.QualifiedName) (ValueBinding, error) { return lookup(name, resolve.ValueUse) },
		Reference: func(name syntax.QualifiedName) (ValueBinding, error) { return lookup(name, resolve.ReferenceUse) },
		Function:  func(name syntax.QualifiedName) (ValueBinding, error) { return lookup(name, resolve.CallUse) },
		Constructor: func(node *syntax.ConstructorExpr, expected *types.Type, expressions *Expressions) (*types.Type, error) {
			if (c.current == nil || c.current.Instance == "") && c.constructorFailures[node] != nil {
				return nil, &source.BlockedError{Dependency: fmt.Sprintf("%s:constructor@%d", file.Source.Path, node.Name.Span.Start)}
			}
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
	if file.Source != nil && file.Source.Syntax != nil {
		expressions.File = file.Source.Syntax.Source
	}
	return expressions
}

// CheckProgram connects project resolution, declaration checking, initialization
// ordering and completion regions without using the superseded compiler passes.
func CheckProgram(graph *project.Graph) (*Program, error) { return checkProgram(graph, true) }
func CheckAssertionProgram(graph *project.Graph) (*Program, error) {
	return checkProgram(graph, false)
}

// CheckBrowserProgram checks the same source with the browser entry shape:
// one exported non-generic void main with no arguments and emits {}. Bun
// main(str[] args) diagnoses here; browser zero-argument main diagnoses
// under CheckProgram. Full source checks run for both targets.
func CheckBrowserProgram(graph *project.Graph) (*Program, error) {
	return checkProgramForTarget(graph, TargetBrowser, true)
}

// CheckBrowserAssertionProgram checks browser source without requiring an
// entry, for assertion-only staging. A present entry may use either valid
// target shape; production browser builds use CheckBrowserProgram.
func CheckBrowserAssertionProgram(graph *project.Graph) (*Program, error) {
	return checkProgramForTarget(graph, TargetBrowser, false)
}

// otherCheckTarget names the alternate entry shape accepted during
// assertion-only staging, when the staged entry never runs.
func otherCheckTarget(target Target) Target {
	if target == TargetBrowser {
		return TargetBun
	}
	return TargetBrowser
}

// CheckProgramForTarget checks source for the named target. Unknown targets
// fail closed.
func CheckProgramForTarget(graph *project.Graph, target Target) (*Program, error) {
	return checkProgramForTarget(graph, target, true)
}

func checkProgram(graph *project.Graph, requireEntry bool) (*Program, error) {
	return checkProgramForTarget(graph, TargetBun, requireEntry)
}

// checkTargetEntry enforces the per-target main shape. Bun keeps the
// historical single str[] input; the browser profile requires no inputs
// and an empty emits bound. Both reject generics, receivers and variadics.
// The diagnostic names the expected target shape so a Bun main fails the
// browser check with the browser expectation and vice versa.
func checkTargetEntry(target Target, d *syntax.FunctionDecl) error {
	if target == TargetBrowser {
		if len(d.Parameters) != 0 ||
			d.Receiver != nil ||
			len(d.Inputs) != 0 ||
			syntax.FormatType(d.Result) != "void" ||
			len(d.Errors.Types) != 0 {
			return fmt.Errorf("browser entry must be an exported non-generic void main with no arguments and emits {}")
		}
		return nil
	}
	if len(d.Parameters) != 0 ||
		d.Receiver != nil ||
		len(d.Inputs) != 1 ||
		d.Inputs[0].Near ||
		d.Inputs[0].Variadic ||
		syntax.FormatType(d.Result) != "void" ||
		syntax.FormatType(d.Inputs[0].Type) != "str[]" {
		return fmt.Errorf("entry must be a non-generic void main(str[] args)")
	}
	return nil
}

func checkProgramForTarget(graph *project.Graph, target Target, requireEntry bool) (*Program, error) {
	if len(graph.Errors) != 0 {
		return nil, errors.Join(graph.Errors...)
	}
	world, err := resolve.Build(graph)
	if err != nil {
		return nil, err
	}
	return checkResolvedProgram(context.Background(), graph, world, target, requireEntry, false)
}

// AnalyzeProgram checks each available declaration using the canonical passes.
// Its partial Program is diagnostic evidence only; strict callers never receive
// a Program when any check fails.
func AnalyzeProgram(graph *project.Graph, world *resolve.World) (*Program, error) {
	return AnalyzeProgramContext(context.Background(), graph, world)
}

func AnalyzeProgramContext(ctx context.Context, graph *project.Graph, world *resolve.World) (*Program, error) {
	program, err := checkResolvedProgram(ctx, graph, world, TargetBun, false, true)
	if cancelled := ctx.Err(); cancelled != nil {
		return nil, cancelled
	}
	return program, err
}

func checkResolvedProgram(ctx context.Context, graph *project.Graph, world *resolve.World, target Target, requireEntry, recovering bool) (result *Program, failure error) {
	var problems []error
	var p *Program
	var c *programChecker
	defer func() {
		if !recovering {
			return
		}
		if failure != nil {
			problems = append(problems, failure)
		}
		failure = errors.Join(problems...)
		if p != nil {
			p.Warnings = c.warnings
			result = p
		}
	}()
	if target != TargetBun && target != TargetBrowser {
		return nil, fmt.Errorf("unknown check target %q: expected %q or %q", string(target), string(TargetBun), string(TargetBrowser))
	}
	var err error
	endDeclarations := editortrace.Stage("check-declarations")
	if recovering {
		_, err = types.AnalyzeDeclarations(world)
	} else {
		_, err = types.CheckDeclarations(world)
	}
	if err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	endDeclarations()
	endSeed := editortrace.Stage("check-seed")
	registry, err := errorDeclarations(world, recovering)
	if err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	c = &programChecker{recovering: recovering, world: world, builder: types.NewBuilder(world), annotations: map[*resolve.File]map[string]*types.Type{}, bindings: map[string]*types.Type{}, variadic: map[string]bool{}, templates: map[string]*Template{}}
	if err = c.builder.SeedDeclarations(); err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	p = &Program{Target: target, World: world, Registry: registry, Intrinsics: map[string]*types.Type{}, Connections: map[string]ConnectionPolicy{}}
	endSeed()
	endCatalogue := editortrace.Stage("check-catalogue-contracts")
	// Resolve maintained contracts from the catalogue in a private canonical scope.
	// Authored calls still require their declaring file's explicit imports.
	builtinFile := &resolve.File{Scope: world.Prelude, Imports: world.Packages}
	c.annotations[builtinFile] = map[string]*types.Type{}
	// One defensive copy serves both loops below: contract gathering here
	// and intrinsic receiver metadata in the callable stage. Both loops
	// read only; the embedded inventory stays unreachable for mutation.
	catalogueOperations := catalogue.Builtin().Inventory().Operations
	for _, op := range catalogueOperations {
		if routeOperation(op.Identity) {
			if err = c.admitRouteOperation(p, builtinFile, op); err != nil {
				return nil, err
			}
			continue
		}
		if browserListenerOperation(op.Identity) {
			if err = c.admitBrowserListener(p, builtinFile, op); err != nil {
				return nil, err
			}
			continue
		}
		if actionOperation(op.Identity) {
			if err = c.admitActionOperation(p, builtinFile, op); err != nil {
				return nil, err
			}
			continue
		}
		if httpGenericOperation(op.Identity) || sqlGenericOperation(op.Identity) || streamGenericOperation(op.Identity) || codecOperation(op.Identity) || formGenericOperation(op.Identity) || fetchGenericOperation(op.Identity) || browserStateOperation(op.Identity) != nil {
			continue
		} // I32, I35, B1-05, codec, form, T23 fetch and T22 state generics specialize per concrete type argument on use.
		if op.Name != catalogue.OpCodecDecodeJsonValue && op.Name != catalogue.OpCodecEncodeJsonValue && op.Lowering.Task != "I22" && op.Lowering.Task != "I23" && op.Lowering.Task != "I24" && !strings.HasPrefix(op.Name, "bytes::") && !strings.HasPrefix(op.Name, "image::") && op.Lowering.Task != "I29" && op.Lowering.Task != "I30" && op.Lowering.Task != "I31" && op.Lowering.Task != "I32" && op.Lowering.Task != "I33" && op.Lowering.Task != "I34" && op.Lowering.Task != "I35" && op.Lowering.Task != "LF08" && op.Lowering.Task != "T22" && op.Lowering.Task != "C02" && op.Lowering.Task != "NT-P28" && op.Lowering.Task != "NT-I04" && op.Lowering.Task != "NT-I01" && op.Lowering.Task != "NT-I11" && op.Lowering.Task != "NT-I02" && op.Lowering.Task != "NT-I03" && op.Lowering.Task != "NT-I13" && !strings.HasPrefix(op.Lowering.Task, "B1-") {
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
	endCatalogue()
	endGather := editortrace.Stage("check-gather")
	var files []*resolve.File
	for _, file := range world.Files {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Source.ID < files[j].Source.ID })
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Wrapper gathering may populate an origin file's cache early;
		// never drop entries another file already computed.
		if c.annotations[file] == nil {
			c.annotations[file] = map[string]*types.Type{}
		}
		for _, name := range []string{"int", "float", "str", "bool", "void"} {
			if _, err = c.gather(file, named(name)); err != nil {
				return nil, err
			}
		}
		for _, declaration := range file.Source.Syntax.Declarations {
			if !world.Checkable(declaration) {
				continue
			}
			gatherErr := func() error {
				switch d := declaration.(type) {
				case *syntax.FetchDecl, *syntax.LLMDecl, *syntax.JudgeDecl, *syntax.QuestionDecl, *syntax.ChoiceArmDecl:
					native, e := c.gatherNative(file, declaration)
					if e != nil {
						return e
					}
					p.Natives = append(p.Natives, native)
				case *syntax.WrapDecl:
					native, e := c.gatherWrapper(file, d)
					if e != nil {
						return e
					}
					p.Natives = append(p.Natives, native)
				case *syntax.ConnectionDecl:
					policy, e := ConnectionDeclaration(file.Source.Syntax.Source, d)
					if e != nil {
						return e
					}
					p.Connections[file.Package.Scope.Symbols[d.Name.Text].ID] = policy
				case *syntax.FunctionDecl:
					if err = validateAssertionNames(file, d); err != nil {
						if !recovering {
							return err
						}
						problems = append(problems, err)
					}
					symbol := world.Declarations[d]
					if !world.Ambiguous[d] && d.Name.Text == "main" && file.Source.Package.Owner == graph.Root {
						if p.Entry != nil {
							return fmt.Errorf("multiple root main declarations")
						}
						if err := checkTargetEntry(target, d); err != nil {
							if !requireEntry && checkTargetEntry(otherCheckTarget(target), d) == nil {
								// Assertion-only staging never runs the
								// entry, so it accepts either valid target
								// shape while still rejecting malformed mains.
							} else {
								return err
							}
						}
						p.Entry = &ProgramFunction{Symbol: symbol}
					}
					if len(d.Parameters) != 0 {
						return nil
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
								return fmt.Errorf("variadic input must be last")
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
							return e
						}
						c.bindings[symbol.ID+"/input/"+field.Name.Text] = typ
					}
					typ, e := c.gather(file, signature)
					if e != nil {
						return e
					}
					c.bindings[symbol.ID] = typ
					fn := &ProgramFunction{Symbol: symbol}
					if p.Entry != nil && p.Entry.Symbol == symbol {
						fn = p.Entry
					}
					// Header evidence survives an independently invalid body.
					p.Functions = append(p.Functions, fn)
					for _, body := range []any{d.Assertions, d.Body} {
						if bodyGatherErr := c.gatherBodyMode(file, reflect.ValueOf(body), recovering); bodyGatherErr != nil {
							if !recovering {
								return bodyGatherErr
							}
							bodyGatherErr = source.Locate(file.Source.Path, d.DeclSpan(), bodyGatherErr)
							problems = append(problems, bodyGatherErr)
							world.Invalid[d] = errors.Join(world.Invalid[d], bodyGatherErr)
						}
					}
				case *syntax.ValueDecl:
					typ, e := c.gather(file, d.Binding.Type)
					if e != nil {
						return e
					}
					c.bindings[file.Package.Scope.Symbols[d.Binding.Name.Text].ID] = typ
					if err = c.gatherBody(file, reflect.ValueOf(d.Binding.Value)); err != nil {
						return err
					}
				case *syntax.ActionDecl:
					if d.Captures != nil {
						if _, e := c.gather(file, d.Captures); e != nil {
							return e
						}
					}
					if d.Input != nil && d.Input.Type != nil {
						if _, e := c.gather(file, d.Input.Type); e != nil {
							return e
						}
					}
					if _, e := c.gather(file, d.Returns); e != nil {
						return e
					}
					for _, kase := range d.Cases {
						if _, e := c.gather(file, &syntax.NamedType{Span: kase.Span, Name: kase.Leaf}); e != nil {
							return e
						}
					}
				}
				return nil
			}()
			if gatherErr != nil {
				gatherErr = source.Locate(file.Source.Path, declaration.DeclSpan(), gatherErr)
				if !recovering {
					return nil, gatherErr
				}
				world.Invalid[declaration] = gatherErr
				for _, symbol := range file.Package.Scope.Symbols {
					if symbol.Declaration == declaration {
						symbol.Invalid = gatherErr
					}
				}
				if !source.IsBlocked(gatherErr) {
					problems = append(problems, gatherErr)
				}
			}

		}
	}
	if requireEntry && p.Entry == nil {
		if target == TargetBrowser {
			return nil, fmt.Errorf("root project requires void main with no arguments and emits {} for target browser")
		}
		return nil, fmt.Errorf("root project requires void main(str[] args)")
	}
	endGather()
	endModel := editortrace.Stage("check-model")
	if recovering {
		p.Model, err = c.builder.FinishRecovering()
	} else {
		p.Model, err = c.builder.Finish()
	}
	if err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	if p.Model == nil {
		return nil, fmt.Errorf("type model unavailable")
	}
	if recovering {
		valid := p.Functions[:0]
		for _, fn := range p.Functions {
			if types.Equal(c.bindings[fn.Symbol.ID], c.bindings[fn.Symbol.ID]) {
				valid = append(valid, fn)
			} else {
				world.Invalid[fn.Symbol.Declaration] = &source.BlockedError{Dependency: "type:" + fn.Symbol.ID}
			}
		}
		p.Functions = valid
		for _, native := range p.Natives {
			if native.Signature != nil && !types.Equal(native.Signature, native.Signature) {
				native.Symbol.Invalid = &source.BlockedError{Dependency: "type:" + native.Symbol.ID}
				world.Invalid[native.Symbol.Declaration] = native.Symbol.Invalid
			}
		}
	}
	for _, fn := range p.Functions {
		if world.Ambiguous[fn.Symbol.Declaration] {
			continue
		}
		signature := c.bindings[fn.Identity()]
		if types.Equal(signature, signature) && signature.Kind() == types.Callable {
			fn.Signature = signature
		}
	}
	c.program = p
	c.instances = map[string]*ProgramFunction{}
	c.specializer, err = types.NewSpecializer(world, p.Model)
	if err != nil {
		return nil, err
	}
	if err = c.checkNativeContracts(p); err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	p.Codecs = c.codecs
	for id := range c.codecs {
		if err = c.finishCodec(id); err != nil {
			special := c.codecs[id]
			if special != nil {
				special.Invalid = err
				err = source.Locate(special.SiteFile, special.SiteSpan, err)
			}
			if special := c.codecs[id]; special != nil && special.SiteFile != "" {
				err = source.Locate(special.SiteFile, special.SiteSpan, err)
			}
			if !recovering {
				return nil, err
			}
			problems = append(problems, err)
		}
	}
	p.HTTPs = c.https
	for id := range c.https {
		if err = c.finishHTTP(id); err != nil {
			special := c.https[id]
			if special != nil {
				special.Invalid = err
				err = source.Locate(special.SiteFile, special.SiteSpan, err)
			}
			if !recovering {
				return nil, err
			}
			problems = append(problems, err)
		}
	}
	p.Forms = c.forms
	for id := range c.forms {
		if err = c.finishForm(id); err != nil {
			special := c.forms[id]
			if special != nil {
				special.Invalid = err
				err = source.Locate(special.SiteFile, special.SiteSpan, err)
			}
			if !recovering {
				return nil, err
			}
			problems = append(problems, err)
		}
	}
	p.Fetches = c.fetches
	for id := range c.fetches {
		if err = c.finishFetch(id); err != nil {
			special := c.fetches[id]
			if special != nil {
				special.Invalid = err
				err = source.Locate(special.SiteFile, special.SiteSpan, err)
			}
			if !recovering {
				return nil, err
			}
			problems = append(problems, err)
		}
	}
	endModel()
	endCallables := editortrace.Stage("check-callables")
	callables := map[string]CallableDeclaration{}
	c.callables = callables
	for _, native := range p.Natives {
		if native.Symbol.Invalid != nil {
			continue
		}
		// Wrapper descriptors publish with their calculated contracts.
		if native.Symbol.Kind == resolve.Wrapper {
			continue
		}
		callables[native.Symbol.ID] = native.Descriptor
	}
	for id, typ := range p.Intrinsics {
		names := make([]string, len(typ.Inputs()))
		for i := range names {
			names[i] = fmt.Sprintf("input%d", i)
		}
		receiver := false
		for _, op := range catalogueOperations {
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
	endCallables()
	endBodies := editortrace.Stage("check-bodies")
	// Action contracts bind after every callable signature is known and
	// the type graph is sealed. Declarations are handler-free, so no
	// executable import is needed to check or export them.
	if err = c.checkActions(files); err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	if err = c.checkWrapperPolicies(p, callables); err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	if err = c.checkTemplates(p, callables); err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	if err = c.checkNativeBodies(p, callables); err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	// Exported-generic declaration proofs resolve declaration-only symbolic
	// Parameter graphs. They run in an isolated fork that is discarded
	// afterwards, so the production model handed to runtime emission stays
	// concrete. Every specializer use is dynamic through c.specializer, so
	// the swap covers the proof's signature, body and region checks.
	production := c.specializer
	rollbackProof := c.unitCheckpoint()
	c.specializer = production.Fork()
	exportedErr := c.checkExportedGenerics(files)
	rollbackProof()
	c.specializer = production
	if exportedErr != nil {
		if !recovering {
			return nil, exportedErr
		}
		problems = append(problems, exportedErr)
	}
	if err = c.genericAssertions(files); err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	for index := 0; index < len(p.Functions); index++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		rollback := c.unitCheckpoint()
		bodyErr := func() error {
			fn := p.Functions[index]
			c.current = fn
			symbol := fn.Symbol
			file := world.Files[symbol.Source]
			d := symbol.Declaration.(*syntax.FunctionDecl)
			if fn.Instance != "" {
				if err = c.gatherBody(file, reflect.ValueOf(d.Body)); err != nil {
					return fmt.Errorf("specialization %s requested at %s: %w", fn.Identity(), strings.Join(fn.Requests, "; "), err)
				}
			}
			context, e := c.functionContext(fn)
			if e != nil {
				return e
			}
			context.Recover = recovering
			if fn.Instance == "" {
				assertions, e := c.assertions(file, fn, context)
				if e != nil {
					if !recovering {
						return e
					}
					world.Invalid[d] = errors.Join(world.Invalid[d], e)
					problems = append(problems, source.Locate(file.Source.Path, d.Name.Span, e))
				}
				p.Assertions = append(p.Assertions, assertions...)
			}
			fn.Region, err = CheckRegion(context, d.Body)
			if err != nil {
				if fn.Instance != "" {
					return fmt.Errorf("concrete function %s (generic source %s, requested at %s): %w", fn.Identity(), symbol.Source.Syntax.Source.Name(), strings.Join(fn.Requests, "; "), err)
				}
				return fmt.Errorf("concrete function %s: %w", fn.Identity(), err)
			}
			return nil
		}()
		if world.Ambiguous[p.Functions[index].Symbol.Declaration] {
			rollback()
			p.Functions[index].Region = nil
		} else if bodyErr != nil {
			rollback()
		}
		if bodyErr != nil {
			fn := p.Functions[index]
			bodyErr = source.Locate(fn.Symbol.Source.Path, fn.Symbol.Declaration.DeclSpan(), bodyErr)
			if !recovering {
				return nil, bodyErr
			}
			problems = append(problems, bodyErr)
			world.Invalid[fn.Symbol.Declaration] = errors.Join(world.Invalid[fn.Symbol.Declaration], bodyErr)
		}

	}
	endBodies()
	c.current = nil
	if err = c.nativeAssertions(p, callables); err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	var initial []InitialValue
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, declaration := range file.Source.Syntax.Declarations {
			if d, ok := declaration.(*syntax.ValueDecl); ok {
				if world.Invalid[declaration] != nil {
					continue
				}
				id := file.Package.Scope.Symbols[d.Binding.Name.Text].ID
				initial = append(initial, InitialValue{Identity: id, QualifiedName: id, Source: file.Source.ID, Binding: d.Binding, Type: c.bindings[id], Checker: c.expressions(file, file.Scope)})
			}
		}
	}
	for _, native := range p.Natives {
		if native.Symbol.Invalid != nil {
			continue
		}
		if native.ArmDescription != nil {
			file := c.world.Files[native.Symbol.Source]
			d := native.Symbol.Declaration.(*syntax.ChoiceArmDecl)
			initial = append(initial, InitialValue{Identity: native.Symbol.ID, QualifiedName: native.Symbol.ID, Source: file.Source.ID, Binding: syntax.Binding{Value: d.Description}, Type: c.bindings[native.Symbol.ID], Checker: c.expressions(file, file.Scope), ArmDescription: native.ArmDescription})
		}
	}
	p.Initializers, err = initialization(initial, nil, recovering)
	if err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	p.Model = c.specializer.Model()
	for _, key := range projectKeys(graph) {
		p.Assets = append(p.Assets, graph.Projects[key].CheckedAssets...)
	}
	p.SQL, err = CheckSQLDescriptors(graph, world, p.Model)
	if err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	sites := c.sqlSites
	for i := range sites {
		site := &sites[i]
		if owner := graph.Projects[site.Owner]; owner != nil {
			site.DescriptorFile = filepath.Join(owner.Root, "can.project.json")
			site.DescriptorSpans = map[string]source.Span{}
			for _, field := range []string{"parameter_type", "row_type", "cardinality"} {
				site.DescriptorSpans[field] = project.JSONFieldSpan(graph.Inputs[site.DescriptorFile], "sql", site.Name, field)
			}
		}
	}
	if recovering {
		validSites := make([]SQLSiteRecord, 0, len(sites))
		available := map[string]bool{}
		for _, descriptor := range p.SQL {
			available[descriptor.Owner+"\x00"+descriptor.Checked.Name] = true
		}
		for _, site := range sites {
			owner := graph.Projects[site.Owner]
			if owner != nil && !available[site.Owner+"\x00"+site.Name] {
				_, declared := owner.Manifest.SQL[site.Name]
				if declared || owner.Manifest.InvalidSQL[site.Name] != nil || owner.Manifest.Invalid["sql"] != nil {
					continue
				}
			}
			validSites = append(validSites, site)
		}
		sites = validSites
	}
	if err = CheckSQLCallSites(p.SQLs, sites, p.SQL); err != nil {
		if !recovering {
			return nil, err
		}
		problems = append(problems, err)
	}
	if recovering {
		for _, file := range files {
			for _, declaration := range file.Source.Syntax.Declarations {
				if fn, ok := declaration.(*syntax.FunctionDecl); ok && len(fn.InvalidAssertions) != 0 {
					world.Invalid[fn] = fmt.Errorf("invalid assertion syntax")
				}
			}
		}
		c.blockDependentRegions()
	}
	noteCyclicRelays(p, c.warn)
	p.Warnings = c.warnings
	return p, nil
}

func projectKeys(graph *project.Graph) []string {
	keys := make([]string, 0, len(graph.Projects))
	for key := range graph.Projects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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
	owner := file.Source.Package.Owner
	context := CompletionContext{Checkpoint: c.unitCheckpoint, Sites: indexLexicalSites(symbol.ID, d), Identity: fn.Identity(), Kind: ir.FunctionRegion, Package: file.Package.ID, File: file.Source.Syntax.Source, Scope: scope, Result: signature.Result(), Errors: bound, Registry: c.program.Registry, Expressions: c.expressions(file, scope), Variadic: c.variadic, Callables: c.callables, Raw: c.rawScope(file), Wrappers: wrapperPlans(c.program), Asset: func(name string) ir.AssetResolution {
		return resolveAssetName(c.world.Graph, owner, name)
	}, SQLSite: func(key, name string, span source.Span) ir.SQLCallSite {
		c.sqlSites = append(c.sqlSites, SQLSiteRecord{Key: key, Owner: owner.Key, Name: name, File: file.Source.Path, Span: span})
		return ir.SQLCallSite{Owner: owner.Key, Name: name}
	}, FormSite: func(operation, key, name string) (ir.FormActionSite, error) {
		return c.formSite(file, operation, key, name)
	}, FetchSite: func(operation, key, name string) (ir.JSONFetchSite, *types.Type, error) {
		return c.fetchSite(file, operation, key, name)
	}, ActionSite: func(operation string, scope *resolve.Scope, name syntax.QualifiedName, rest []syntax.Argument) (ir.ActionSite, *types.Type, error) {
		return c.actionSite(file, scope, operation, name, rest)
	}}

	context.IntrinsicIdentity = func(scope *resolve.Scope, name syntax.QualifiedName) string {
		symbol, err := file.Lookup(scope, name, resolve.CallUse)
		if err != nil {
			return ""
		}
		return symbol.ID
	}
	context.Expand = func(scope *resolve.Scope, row syntax.Assertion) ([]ir.FixtureRow, *Template, error) {
		return c.expandTemplateUse(file, scope, row)
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
	context.Warn = c.warn
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

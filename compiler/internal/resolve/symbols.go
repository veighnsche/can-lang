// Package resolve owns current package identities and eligible-kind lookup.
// Type checking supplies concrete value callability when generic specialization
// needs more evidence than a source annotation can provide.
package resolve

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/catalogue"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// located attaches the failing source span to a Build error so editor
// bridges convert positions from structure instead of message prose.
func located(src *project.Source, span source.Span, err error) error {
	if src == nil {
		return err
	}
	return source.Locate(src.Path, span, err)
}

type Kind string

const (
	Connection    Kind = "connection"
	ChoiceArm     Kind = "choice_arm"
	Record        Kind = "record"
	Error         Kind = "error"
	Variant       Kind = "variant"
	Opaque        Kind = "opaque"
	Primitive     Kind = "primitive"
	TypeParameter Kind = "type_parameter"
	Function      Kind = "function"
	Fetch         Kind = "fetch"
	Question      Kind = "question"
	Judge         Kind = "judge"
	LLM           Kind = "llm"
	Value         Kind = "value"
)

type Usage string

const (
	ConnectionUse  Usage = "connection"
	QuestionUse    Usage = "question"
	ReferenceUse   Usage = "reference"
	TypeUse        Usage = "type"
	ConstructorUse Usage = "constructor"
	CallUse        Usage = "call"
	ValueUse       Usage = "value"
	ErrorUse       Usage = "error"
)

type Symbol struct {
	Name, ID                        string
	Kind                            Kind
	Public, Constructible, Callable bool
	Package                         *Package
	Source                          *project.Source
	Declaration                     syntax.Declaration
	GeneratedQuestion               *syntax.QuestionDecl
	Type                            syntax.TypeNode
	Receiver                        *Symbol
	Parameters                      []string
}

func (s *Symbol) Eligible(usage Usage) bool {
	switch usage {
	case ConnectionUse:
		return s.Kind == Connection
	case QuestionUse:
		return s.Kind == Question
	case ReferenceUse:
		return (s.Kind == Function || s.Kind == Fetch) && s.Receiver == nil
	case TypeUse:
		return s.Kind == Record || s.Kind == Error || s.Kind == Variant || s.Kind == Opaque || s.Kind == Primitive || s.Kind == TypeParameter
	case ConstructorUse:
		return s.Constructible
	case CallUse:
		return (s.Kind == Function && s.Receiver == nil) || s.Kind == Fetch || s.Kind == Judge || s.Kind == LLM || s.Kind == Value && s.Callable
	case ValueUse:
		return s.Kind == Value || s.Kind == ChoiceArm
	case ErrorUse:
		return s.Kind == Error
	default:
		return false
	}
}

type Scope struct {
	Parent   *Scope
	Symbols  map[string]*Symbol
	reserved map[string]bool
}

func NewScope(parent *Scope) *Scope {
	reserved := map[string]bool{}
	if parent != nil {
		reserved = parent.reserved
	}
	return &Scope{Parent: parent, Symbols: map[string]*Symbol{}, reserved: reserved}
}
func (s *Scope) Define(symbol *Symbol) error {
	if s.reserved[symbol.Name] {
		return fmt.Errorf("prelude name %q cannot be redeclared or shadowed", symbol.Name)
	}
	if _, exists := s.Symbols[symbol.Name]; exists {
		return fmt.Errorf("duplicate name %q in one scope", symbol.Name)
	}
	s.Symbols[symbol.Name] = symbol
	return nil
}
func (s *Scope) Lookup(name string, usage Usage) (*Symbol, error) {
	return s.LookupWhere(name, func(symbol *Symbol) bool { return symbol.Eligible(usage) })
}

// LookupWhere lets the checked type graph supply eligibility for a concrete
// generic specialization. It must not freeze a guessed callable type at parsing.
func (s *Scope) LookupWhere(name string, eligible func(*Symbol) bool) (*Symbol, error) {
	for scope := s; scope != nil; scope = scope.Parent {
		if symbol := scope.Symbols[name]; symbol != nil && eligible(symbol) {
			return symbol, nil
		}
	}
	return nil, fmt.Errorf("no eligible declaration for %q", name)
}

type Package struct {
	Name, ID string
	Source   *project.Package
	Scope    *Scope
}
type File struct {
	Source  *project.Source
	Package *Package
	Scope   *Scope
	Imports map[string]*Package
}
type World struct {
	Graph        *project.Graph
	Prelude      *Scope
	Packages     map[string]*Package
	Files        map[*project.Source]*File
	Functions    map[*syntax.FunctionDecl]*Scope
	NativeScopes map[syntax.Declaration]*Scope
}

func Build(graph *project.Graph) (*World, error) {
	w := &World{Graph: graph, Prelude: NewScope(nil), Packages: map[string]*Package{}, Files: map[*project.Source]*File{}, Functions: map[*syntax.FunctionDecl]*Scope{}, NativeScopes: map[syntax.Declaration]*Scope{}}
	if err := w.catalogue(); err != nil {
		return nil, err
	}
	for _, name := range keys(graph.Packages) {
		p := graph.Packages[name]
		pkg := &Package{Name: name, ID: p.ID, Source: p, Scope: NewScope(w.Prelude)}
		w.Packages[name] = pkg
		for _, src := range p.Sources {
			file := &File{Source: src, Package: pkg, Scope: NewScope(pkg.Scope), Imports: map[string]*Package{}}
			w.Files[src] = file
			for _, declaration := range src.Syntax.Declarations {
				symbol := declarationSymbol(declaration)
				symbol.ID = p.ID + "::" + symbol.Name
				symbol.Package = pkg
				symbol.Source = src
				symbol.Declaration = declaration
				if err := pkg.Scope.Define(symbol); err != nil {
					return nil, located(src, declarationNameSpan(declaration), fmt.Errorf("%s: %w", src.Path, err))
				}
				if q, ok := declaration.(*syntax.QuestionDecl); ok && q.RecordName != nil {
					generated := generatedQuestionRecord(q)
					record := &Symbol{Name: q.RecordName.Text, ID: p.ID + "::" + q.RecordName.Text, Kind: Record, Constructible: true, Package: pkg, Source: src, Declaration: generated, GeneratedQuestion: q}
					if err := pkg.Scope.Define(record); err != nil {
						return nil, located(src, q.RecordName.Span, fmt.Errorf("%s: %w", src.Path, err))
					}
				}
			}
		}
	}
	// All tables exist before exports, imports or signatures are inspected. Source
	// package import cycles are legal and do not require loading-order resolution.
	for _, name := range keys(graph.Packages) {
		for _, src := range graph.Packages[name].Sources {
			file := w.Files[src]
			seen := map[string]bool{}
			for _, export := range src.Syntax.Header.Provides {
				symbol := file.Package.Scope.Symbols[export.Text]
				if symbol == nil || symbol.Source != src {
					return nil, located(src, export.Span, fmt.Errorf("%s: provides %q is not declared by this file", src.Path, export.Text))
				}
				if seen[export.Text] {
					return nil, located(src, export.Span, fmt.Errorf("%s: duplicate provides entry %q", src.Path, export.Text))
				}
				seen[export.Text] = true
				symbol.Public = true
			}
		}
	}
	for _, name := range keys(graph.Packages) {
		for _, src := range graph.Packages[name].Sources {
			file := w.Files[src]
			for _, entry := range src.Syntax.Header.Uses {
				target := w.Packages[entry.Package.Text]
				if target == nil {
					return nil, located(src, entry.Package.Span, fmt.Errorf("%s: unknown package %q", src.Path, entry.Package.Text))
				}
				alias := entry.Package.Text
				if entry.Alias != nil {
					alias = entry.Alias.Text
				}
				if file.Imports[alias] != nil || file.Package.Scope.Symbols[alias] != nil || w.Prelude.reserved[alias] {
					return nil, located(src, importEntrySpan(entry), fmt.Errorf("%s: import alias %q collides with another name", src.Path, alias))
				}
				if catalogue.Builtin().IsReservedPackage(alias) && alias != target.Name {
					return nil, located(src, importEntrySpan(entry), fmt.Errorf("import alias %q claims a reserved catalogue package", alias))
				}
				if target.Source != nil {
					owner := src.Package.Owner
					permitted := target.Source.Owner == owner
					for _, dep := range owner.Dependencies {
						if dep == target.Source.Owner {
							permitted = true
						}
					}
					if !permitted {
						return nil, located(src, entry.Package.Span, fmt.Errorf("%s: package %q belongs to an undeclared direct dependency", src.Path, target.Name))
					}
					if err := internalVisibility(src, target.Source); err != nil {
						return nil, located(src, entry.Package.Span, err)
					}
				}
				file.Imports[alias] = target
			}
		}
	}
	for _, name := range keys(graph.Packages) {
		for _, src := range graph.Packages[name].Sources {
			file := w.Files[src]
			for _, declaration := range src.Syntax.Declarations {
				if err := w.signature(file, declaration); err != nil {
					return nil, located(src, declaration.DeclSpan(), fmt.Errorf("%s: %w", src.Path, err))
				}
			}
		}
	}
	return w, nil
}

func declarationSymbol(declaration syntax.Declaration) *Symbol {
	s := &Symbol{}
	var parameters []syntax.Token
	switch d := declaration.(type) {
	case *syntax.ConnectionDecl:
		s.Name = d.Name.Text
		s.Kind = Connection
	case *syntax.FetchDecl:
		s.Name = d.Name.Text
		s.Kind = Fetch
	case *syntax.LLMDecl:
		s.Name = d.Name.Text
		s.Kind = LLM
	case *syntax.JudgeDecl:
		s.Name = d.Name.Text
		s.Kind = Judge
	case *syntax.QuestionDecl:
		s.Name = d.Name.Text
		s.Kind = Question
	case *syntax.ChoiceArmDecl:
		s.Name = d.Name.Text
		s.Kind = ChoiceArm
	case *syntax.RecordDecl:
		s.Name = d.Name.Text
		s.Kind = Record
		s.Constructible = true
		parameters = d.Parameters
	case *syntax.ErrorDecl:
		s.Name = d.Name.Text
		s.Kind = Error
		s.Constructible = true
		parameters = d.Parameters
	case *syntax.VariantDecl:
		s.Name = d.Name.Text
		s.Kind = Variant
		parameters = d.Parameters
	case *syntax.FunctionDecl:
		s.Name = d.Name.Text
		s.Kind = Function
		parameters = d.Parameters
	case *syntax.ValueDecl:
		s.Name = d.Binding.Name.Text
		s.Kind = Value
		s.Type = d.Binding.Type
		_, s.Callable = d.Binding.Type.(*syntax.CallableType)
	default:
		panic("unhandled current declaration")
	}
	for _, p := range parameters {
		s.Parameters = append(s.Parameters, p.Text)
	}
	return s
}

// declarationNameSpan points at the declared name token, falling back to
// the whole declaration when the form carries no name token.
func declarationNameSpan(declaration syntax.Declaration) source.Span {
	switch d := declaration.(type) {
	case *syntax.ConnectionDecl:
		return d.Name.Span
	case *syntax.FetchDecl:
		return d.Name.Span
	case *syntax.LLMDecl:
		return d.Name.Span
	case *syntax.JudgeDecl:
		return d.Name.Span
	case *syntax.QuestionDecl:
		return d.Name.Span
	case *syntax.ChoiceArmDecl:
		return d.Name.Span
	case *syntax.RecordDecl:
		return d.Name.Span
	case *syntax.ErrorDecl:
		return d.Name.Span
	case *syntax.VariantDecl:
		return d.Name.Span
	case *syntax.FunctionDecl:
		return d.Name.Span
	case *syntax.ValueDecl:
		return d.Binding.Name.Span
	default:
		return declaration.DeclSpan()
	}
}

// importEntrySpan points at the alias when one is written, else the package.
func importEntrySpan(entry syntax.Import) source.Span {
	if entry.Alias != nil {
		return entry.Alias.Span
	}
	return entry.Package.Span
}

func (w *World) catalogue() error {
	inv := catalogue.Builtin().Inventory()
	for _, name := range []string{"int", "float", "bool", "str", "void"} {
		w.Prelude.Symbols[name] = &Symbol{Name: name, ID: "can.primitive/" + name, Kind: Primitive, Public: true}
	}
	for _, p := range inv.Packages {
		w.Packages[p.Name] = &Package{Name: p.Name, ID: p.Identity, Scope: NewScope(nil)}
	}
	add := func(name string, symbol *Symbol) error {
		parts := strings.Split(name, "::")
		scope := w.Prelude
		if len(parts) == 2 {
			pkg := w.Packages[parts[0]]
			if pkg == nil {
				return fmt.Errorf("invalid catalogue package")
			}
			symbol.Package = pkg
			scope = pkg.Scope
		}
		symbol.Name = parts[len(parts)-1]
		symbol.Public = true
		return scope.Define(symbol)
	}
	for _, d := range inv.Types {
		symbol := &Symbol{ID: d.Identity, Kind: Kind(d.Kind), Constructible: d.Constructible}
		for _, p := range d.Parameters {
			symbol.Parameters = append(symbol.Parameters, p.Name)
		}
		if err := add(d.Name, symbol); err != nil {
			return err
		}
	}
	for _, d := range inv.Errors {
		symbol := &Symbol{ID: d.Identity, Kind: Error, Constructible: true}
		for _, p := range d.Parameters {
			symbol.Parameters = append(symbol.Parameters, p.Name)
		}
		if err := add(d.Name, symbol); err != nil {
			return err
		}
	}
	for _, d := range inv.Operations {
		if d.Kind == "function" {
			symbol := &Symbol{ID: d.Identity, Kind: Function}
			for _, p := range d.Parameters {
				symbol.Parameters = append(symbol.Parameters, p.Name)
			}
			if err := add(d.Name, symbol); err != nil {
				return err
			}
		}
	}
	for _, name := range inv.Prelude {
		w.Prelude.reserved[name] = true
	}
	return nil
}

func (f *File) Lookup(scope *Scope, name syntax.QualifiedName, usage Usage) (*Symbol, error) {
	if name.Package == "" {
		if scope == nil {
			scope = f.Scope
		}
		return scope.Lookup(name.Name, usage)
	}
	pkg := f.Imports[name.Package]
	if pkg == nil {
		return nil, fmt.Errorf("package qualifier %q is not imported by this file", name.Package)
	}
	symbol := pkg.Scope.Symbols[name.Name]
	if symbol == nil || !symbol.Eligible(usage) {
		return nil, fmt.Errorf("no eligible %s %s::%s", usage, name.Package, name.Name)
	}
	if pkg != f.Package && !symbol.Public {
		return nil, fmt.Errorf("%s::%s is private", name.Package, name.Name)
	}
	return symbol, nil
}

func (f *File) Method(receiver *Symbol, name string) (*Symbol, error) {
	if receiver == nil || receiver.Package == nil || receiver.Kind != Record {
		return nil, fmt.Errorf("method receiver is not a project record")
	}
	pkg := receiver.Package
	if pkg.Source == nil {
		return nil, fmt.Errorf("catalogue methods use their closed operation descriptors")
	}
	method := pkg.Scope.Symbols[name]
	if method == nil || method.Receiver != receiver {
		return nil, fmt.Errorf("record %s has no method %q", receiver.Name, name)
	}
	if pkg != f.Package {
		imported := false
		for _, p := range f.Imports {
			if p == pkg {
				imported = true
			}
		}
		if !imported || !method.Public {
			return nil, fmt.Errorf("method %q is not exported from an imported receiver package", name)
		}
	}
	return method, nil
}

func internalVisibility(caller *project.Source, target *project.Package) error {
	rel, err := filepath.Rel(target.Owner.Root, target.Directory)
	if err != nil {
		return err
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, part := range parts {
		if part == "internal" {
			parent := filepath.Join(append([]string{target.Owner.Root}, parts[:i]...)...)
			if !project.Contains(parent, caller.Path) {
				return fmt.Errorf("%s cannot import internal package %q", caller.Path, target.Name)
			}
		}
	}
	return nil
}
func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (w *World) signature(file *File, declaration syntax.Declaration) error {
	symbol := file.Package.Scope.Symbols[declarationSymbol(declaration).Name]
	scope := NewScope(file.Scope)
	for _, name := range symbol.Parameters {
		if err := scope.Define(&Symbol{Name: name, ID: symbol.ID + "<" + name + ">", Kind: TypeParameter}); err != nil {
			return err
		}
	}
	check := func(typ syntax.TypeNode) error { return file.checkType(scope, typ, symbol.Public, TypeUse) }
	fields := func(fields []syntax.Field) error {
		seen := map[string]bool{}
		for _, field := range fields {
			if seen[field.Name.Text] {
				return fmt.Errorf("duplicate field %q", field.Name.Text)
			}
			seen[field.Name.Text] = true
			if err := check(field.Type); err != nil {
				return err
			}
		}
		return nil
	}
	if syntax.NativeSignature(declaration) != nil {
		return w.nativeSignature(file, scope, symbol, declaration)
	}
	switch d := declaration.(type) {
	case *syntax.ConnectionDecl:
		return nil
	case *syntax.ChoiceArmDecl:
		if err := check(d.Result); err != nil {
			return err
		}
		return file.checkBound(scope, d.Errors, symbol.Public)
	case *syntax.RecordDecl:
		return fields(d.Fields)
	case *syntax.ErrorDecl:
		return fields(d.Fields)
	case *syntax.VariantDecl:
		for _, t := range d.Alternatives {
			if err := check(t); err != nil {
				return err
			}
		}
	case *syntax.ValueDecl:
		return check(d.Binding.Type)
	case *syntax.FunctionDecl:
		if err := check(d.Result); err != nil {
			return err
		}
		if err := file.checkBound(scope, d.Errors, symbol.Public); err != nil {
			return err
		}
		if d.Receiver != nil {
			if err := check(d.Receiver.Type); err != nil {
				return err
			}
			named, ok := d.Receiver.Type.(*syntax.NamedType)
			if !ok {
				return fmt.Errorf("method receiver must be a nominal record")
			}
			receiver, err := file.Lookup(scope, named.Name, TypeUse)
			if err != nil {
				return err
			}
			if receiver.Kind != Record || receiver.Package != file.Package {
				return fmt.Errorf("only a record's declaring package may define its methods")
			}
			symbol.Receiver = receiver
		}
		// Generic parameters, receiver and inputs occupy one collision domain.
		add := func(field syntax.Field) error {
			if err := check(field.Type); err != nil {
				return err
			}
			_, callable := field.Type.(*syntax.CallableType)
			return scope.Define(&Symbol{Name: field.Name.Text, ID: symbol.ID + "/input/" + field.Name.Text, Kind: Value, Type: field.Type, Callable: callable})
		}
		if d.Receiver != nil {
			if err := add(*d.Receiver); err != nil {
				return err
			}
		}
		for _, input := range d.Inputs {
			if err := add(input.Field); err != nil {
				return err
			}
		}
		w.Functions[d] = scope
	}
	return nil
}

func (f *File) checkBound(scope *Scope, bound syntax.ErrorBound, public bool) error {
	for _, typ := range bound.Types {
		if err := f.checkType(scope, typ, public, ErrorUse); err != nil {
			return err
		}
	}
	return nil
}
func (f *File) checkType(scope *Scope, typ syntax.TypeNode, public bool, usage Usage) error {
	if usage == ErrorUse {
		if _, ok := typ.(*syntax.NamedType); !ok {
			return fmt.Errorf("emits requires named error types")
		}
	}
	switch t := typ.(type) {
	case *syntax.NamedType:
		symbol, err := f.Lookup(scope, t.Name, usage)
		if err != nil {
			return err
		}
		if public && symbol.Source != nil && !symbol.Public {
			return fmt.Errorf("exported signature exposes private type %s", symbol.ID)
		}
		for _, arg := range t.Arguments {
			if err := f.checkType(scope, arg, public, TypeUse); err != nil {
				return err
			}
		}
	case *syntax.ArrayType:
		return f.checkType(scope, t.Element, public, TypeUse)
	case *syntax.CallableType:
		if err := f.checkType(scope, t.Result, public, TypeUse); err != nil {
			return err
		}
		for _, input := range t.Inputs {
			if err := f.checkType(scope, input, public, TypeUse); err != nil {
				return err
			}
		}
		return f.checkBound(scope, t.Errors, public)
	case *syntax.ChoiceArmType:
		if err := f.checkType(scope, t.Result, public, TypeUse); err != nil {
			return err
		}
		return f.checkBound(scope, t.Errors, public)
	default:
		return fmt.Errorf("unknown type syntax")
	}
	return nil
}

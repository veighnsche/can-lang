package driver

import (
	"path/filepath"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	compileresolve "github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// Hover is one answered type-at-offset query: the declared contract
// rendered as Markdown plus the hovered token range in editor coordinates.
// The range always covers source in the requesting file, never a jump
// target, so catalogue symbols without source files still hover precisely.
type Hover struct {
	Contents string
	Line     int
	Start    int
	End      int
}

// HoverAt resolves the identifier at an editor offset to its declared
// contract over the checked World: nominal types, calls, constructors,
// module values, and record fields or methods behind annotation-known
// receivers. Contents carries a ```can block in the canonical spelling
// plus the package or catalogue provenance the name resolved through.
// The query is inert — it reads the snapshot and never builds, runs,
// checks, or emits — and ok is false wherever Definition declines, so
// unresolved source never receives a guessed type.
func HoverAt(snapshot *Snapshot, file string, line, character int) (Hover, bool, error) {
	var none Hover
	if snapshot == nil || snapshot.World == nil || snapshot.Graph == nil {
		return none, false, nil
	}
	canonical, err := filepath.EvalSymlinks(file)
	if err != nil {
		return none, false, nil
	}
	var src *project.Source
	for _, p := range snapshot.Graph.Projects {
		for _, s := range p.Sources {
			if s.Path == canonical {
				src = s
			}
		}
	}
	if src == nil || src.Syntax == nil {
		return none, false, nil
	}
	resolved, ok := snapshot.World.Files[src]
	if !ok {
		return none, false, nil
	}
	text, err := source.New(canonical, string(src.Bytes))
	if err != nil {
		return none, false, nil
	}
	offset, err := text.Offset(source.UTF16Position{Line: line, Character: character})
	if err != nil {
		return none, false, nil
	}
	reference := findReference(src.Syntax, offset)
	if reference.empty() {
		return none, false, nil
	}
	contents, span, ok := hoverContents(src.Syntax, resolved, reference, offset)
	if !ok {
		return none, false, nil
	}
	location, ok, err := convertSpan(text, canonical, span)
	if err != nil || !ok {
		return none, false, nil
	}
	return Hover{Contents: contents, Line: location.Line, Start: location.Start, End: location.End}, true, nil
}

// hoverContents renders the contract for one reference and reports the
// requesting-file token span the hover ranges over. Declaration sites
// describe their own declaration; members describe the field or method
// behind the checked receiver; every other name resolves through the
// checked World exactly like Definition, including its local and
// unresolved declines.
func hoverContents(file *syntax.File, resolved *compileresolve.File, reference reference, offset int) (string, source.Span, bool) {
	var none source.Span
	if reference.self {
		contract, ok := describeSelf(file, offset)
		if !ok {
			return "", none, false
		}
		return hoverMessage(contract, packageOrigin(resolved.Package)), reference.span, true
	}
	if reference.member != "" {
		contract, ok := describeMember(resolved, reference)
		if !ok {
			return "", none, false
		}
		return hoverMessage(contract, memberOrigin(resolved, reference)), reference.span, true
	}
	if reference.shadowable {
		if shadowedByLocal(file, offset, reference.name.Name) {
			return "", none, false
		}
	}
	symbol, err := resolved.Lookup(nil, reference.name, reference.usage)
	if err != nil {
		return "", none, false
	}
	contract, ok := describeSymbol(symbol)
	if !ok {
		return "", none, false
	}
	return hoverMessage(contract, symbolOrigin(symbol)), reference.name.Span, true
}

// hoverMessage frames one contract in the canonical spelling with its
// resolution provenance on a second paragraph.
func hoverMessage(contract, origin string) string {
	return "```can\n" + contract + "\n```\n\n" + origin
}

// describeSelf renders the declaration enclosing a declaration-site name:
// functions, values, records, variants, errors, and choice arms by their
// header contract, record and error field names by their field line.
func describeSelf(file *syntax.File, offset int) (string, bool) {
	for _, declaration := range file.Declarations {
		if !contains(declaration.DeclSpan(), offset) {
			continue
		}
		switch decl := declaration.(type) {
		case *syntax.RecordDecl:
			if field, ok := fieldAt(decl.Fields, offset); ok {
				return describeField(decl.Name.Text, field)
			}
		case *syntax.ErrorDecl:
			if field, ok := fieldAt(decl.Fields, offset); ok {
				return describeField(decl.Name.Text, field)
			}
		}
		return describeDeclaration(declaration)
	}
	return "", false
}

// describeMember renders the field or method behind a checked receiver:
// fields by their declared field line, methods by the method's own
// function contract.
func describeMember(file *compileresolve.File, reference reference) (string, bool) {
	record, err := receiverRecord(file, reference.receiver)
	if err != nil {
		return "", false
	}
	if reference.method {
		method, err := file.Method(record, reference.member)
		if err != nil {
			return "", false
		}
		return describeDeclaration(method.Declaration)
	}
	decl, ok := record.Declaration.(*syntax.RecordDecl)
	if !ok {
		return "", false
	}
	for i := range decl.Fields {
		if decl.Fields[i].Name.Text == reference.member {
			return describeField(record.Name, &decl.Fields[i])
		}
	}
	return "", false
}

// describeSymbol renders one resolved symbol: declarations by their
// header contract, declaration-less catalogue and prelude symbols by
// kind and qualified name.
func describeSymbol(symbol *compileresolve.Symbol) (string, bool) {
	if symbol == nil {
		return "", false
	}
	if symbol.Declaration != nil {
		return describeDeclaration(symbol.Declaration)
	}
	name := symbol.Name
	if pkg := symbol.Package; pkg != nil && pkg.Source == nil && pkg.Name != "" {
		name = pkg.Name + "::" + name
	}
	return kindLabel(symbol.Kind) + " " + name, true
}

// describeDeclaration renders the header contract of the declarations
// whose shape is a type contract, mirroring the formatter's canonical
// spelling. Native and harness declarations carry no hoverable type
// contract and decline.
func describeDeclaration(declaration syntax.Declaration) (string, bool) {
	switch decl := declaration.(type) {
	case *syntax.FunctionDecl:
		return describeFunction(decl)
	case *syntax.ValueDecl:
		typ, ok := formatCheckedType(decl.Binding.Type)
		if !ok {
			return "", false
		}
		return typ + " " + decl.Binding.Name.Text, true
	case *syntax.RecordDecl:
		return describeRecord(decl)
	case *syntax.VariantDecl:
		lines := []string{"variant " + decl.Name.Text + formatHoverParameters(decl.Parameters)}
		for _, alternative := range decl.Alternatives {
			typ, ok := formatCheckedType(alternative)
			if !ok {
				return "", false
			}
			lines = append(lines, "    "+typ)
		}
		return strings.Join(lines, "\n"), true
	case *syntax.ErrorDecl:
		fields := make([]string, len(decl.Fields))
		for i := range decl.Fields {
			field, ok := formatHoverField(decl.Fields[i])
			if !ok {
				return "", false
			}
			fields[i] = field
		}
		return "error " + decl.Name.Text + formatHoverParameters(decl.Parameters) + "(" + strings.Join(fields, ", ") + ")", true
	case *syntax.ChoiceArmDecl:
		result, ok := formatCheckedType(decl.Result)
		if !ok {
			return "", false
		}
		bound, ok := formatHoverBound(decl.Errors)
		if !ok {
			return "", false
		}
		return "choice_arm " + result + " " + decl.Name.Text + "\n" + bound, true
	default:
		return "", false
	}
}

// describeFunction renders the callable contract: result, name, receiver,
// error bound, and the given inputs with their near and variadic marks.
func describeFunction(decl *syntax.FunctionDecl) (string, bool) {
	result, ok := formatCheckedType(decl.Result)
	if !ok {
		return "", false
	}
	lines := []string{"fn " + result + " " + decl.Name.Text + formatHoverParameters(decl.Parameters)}
	if decl.Receiver != nil {
		receiver, ok := formatHoverField(*decl.Receiver)
		if !ok {
			return "", false
		}
		lines = append(lines, "on "+receiver)
	}
	bound, ok := formatHoverBound(decl.Errors)
	if !ok {
		return "", false
	}
	lines = append(lines, bound)
	if len(decl.Inputs) > 0 {
		lines = append(lines, "given")
		for _, input := range decl.Inputs {
			typ, ok := formatCheckedType(input.Type)
			if !ok {
				return "", false
			}
			text := ""
			if input.Near {
				text = "near "
			}
			text += typ + " "
			if input.Variadic {
				text += "..."
			}
			lines = append(lines, "    "+text+input.Name.Text)
		}
	}
	return strings.Join(lines, "\n"), true
}

// describeRecord renders the shared-record contract: header plus one
// declared field line each, including callable-typed callback fields.
func describeRecord(decl *syntax.RecordDecl) (string, bool) {
	header := "record " + decl.Name.Text + formatHoverParameters(decl.Parameters)
	if decl.Owner {
		header = "owner " + header
	}
	lines := []string{header}
	for i := range decl.Fields {
		field, ok := formatHoverField(decl.Fields[i])
		if !ok {
			return "", false
		}
		lines = append(lines, "    "+field)
	}
	return strings.Join(lines, "\n"), true
}

// describeField renders one record field behind a checked receiver in
// the receiver-dot-field spelling field uses carry.
func describeField(record string, field *syntax.Field) (string, bool) {
	if field == nil {
		return "", false
	}
	typ, ok := formatCheckedType(field.Type)
	if !ok {
		return "", false
	}
	return typ + " " + record + "." + field.Name.Text, true
}

// fieldAt finds the record or error field whose name token covers the
// offset, for declaration-site field hovers.
func fieldAt(fields []syntax.Field, offset int) (*syntax.Field, bool) {
	for i := range fields {
		if contains(fields[i].Name.Span, offset) {
			return &fields[i], true
		}
	}
	return nil, false
}

// formatHoverField renders one canonical "type name" field line.
func formatHoverField(field syntax.Field) (string, bool) {
	typ, ok := formatCheckedType(field.Type)
	if !ok {
		return "", false
	}
	return typ + " " + field.Name.Text, true
}

// formatHoverBound renders one canonical emits bound.
func formatHoverBound(bound syntax.ErrorBound) (string, bool) {
	parts := make([]string, len(bound.Types))
	for i, typ := range bound.Types {
		rendered, ok := formatCheckedType(typ)
		if !ok {
			return "", false
		}
		parts[i] = rendered
	}
	return "emits [" + strings.Join(parts, ", ") + "]", true
}

// formatHoverParameters renders one canonical type-parameter suffix.
func formatHoverParameters(parameters []syntax.Token) string {
	if len(parameters) == 0 {
		return ""
	}
	names := make([]string, len(parameters))
	for i, parameter := range parameters {
		names[i] = parameter.Text
	}
	return "<" + strings.Join(names, ", ") + ">"
}

// formatCheckedType renders one checked annotation through the canonical
// type spelling. A missing annotation is not rendered as a guess; the
// hover declines instead.
func formatCheckedType(node syntax.TypeNode) (string, bool) {
	if node == nil {
		return "", false
	}
	return syntax.FormatType(node), true
}

// kindLabel names declaration-less symbols by their checked kind.
func kindLabel(kind compileresolve.Kind) string {
	switch kind {
	case compileresolve.TypeParameter:
		return "type parameter"
	default:
		return string(kind)
	}
}

// symbolOrigin reports where a resolved symbol came from: its declaring
// package, else the catalogue for prelude and catalogue symbols.
func symbolOrigin(symbol *compileresolve.Symbol) string {
	if symbol == nil {
		return "catalogue"
	}
	return packageOrigin(symbol.Package)
}

// memberOrigin reports where a hovered member's record came from.
func memberOrigin(file *compileresolve.File, reference reference) string {
	record, err := receiverRecord(file, reference.receiver)
	if err != nil {
		return "catalogue"
	}
	return symbolOrigin(record)
}

// packageOrigin names a resolve package, treating missing and catalogue
// packages as the catalogue.
func packageOrigin(pkg *compileresolve.Package) string {
	if pkg == nil || pkg.Source == nil {
		return "catalogue"
	}
	return "package " + pkg.Name
}

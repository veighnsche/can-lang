package resolve

import (
	"fmt"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

func (w *World) nativeSignature(file *File, scope *Scope, symbol *Symbol, declaration syntax.Declaration) error {
	header := syntax.NativeSignature(declaration)
	connection, err := file.Lookup(scope, header.Connection, ConnectionUse)
	if err != nil {
		return err
	}
	if symbol.Public && !connection.Public {
		return fmt.Errorf("exported signature exposes private connection %s", connection.ID)
	}
	if err := file.checkType(scope, header.Result, symbol.Public, TypeUse); err != nil {
		return err
	}
	if err := file.checkBound(scope, header.Errors, symbol.Public); err != nil {
		return err
	}
	if q, ok := declaration.(*syntax.QuestionDecl); ok && q.RecordName != nil {
		generated := file.Package.Scope.Symbols[q.RecordName.Text]
		if symbol.Public && !generated.Public {
			return fmt.Errorf("exported question cannot hide its generated record %s", q.RecordName.Text)
		}
		if err := w.signature(file, generated.Declaration); err != nil {
			return err
		}
	}
	add := func(field syntax.Field) error {
		if err := file.checkType(scope, field.Type, symbol.Public, TypeUse); err != nil {
			return err
		}
		_, callable := field.Type.(*syntax.CallableType)
		return scope.Define(&Symbol{Name: field.Name.Text, ID: symbol.ID + "/input/" + field.Name.Text, Kind: Value, Type: field.Type, Callable: callable})
	}
	for i, input := range header.Inputs {
		if input.Variadic && i != len(header.Inputs)-1 {
			return fmt.Errorf("variadic native input must be last")
		}
		field := input.Field
		if input.Variadic {
			field.Type = &syntax.ArrayType{Element: field.Type}
		}
		if err := add(field); err != nil {
			return err
		}
	}
	var state []syntax.Field
	switch d := declaration.(type) {
	case *syntax.JudgeDecl:
		state = d.State
	case *syntax.LLMDecl:
		state = d.State
	case *syntax.QuestionDecl:
		for _, binder := range d.Binders {
			if err := scope.Define(&Symbol{Name: binder.Name.Text, ID: symbol.ID + "/metadata/" + binder.Name.Text, Kind: Value, Type: &syntax.NamedType{Name: syntax.QualifiedName{Name: "float"}}}); err != nil {
				return err
			}
		}
	}
	for _, field := range state {
		if err := add(field); err != nil {
			return err
		}
	}
	w.NativeScopes[declaration] = scope
	return nil
}

// WrapperOrigin resolves the immediate base and the original fetch/judge
// root of a wrapper base chain, independent of declaration order. Only
// fetch, judge and wrapper declarations are admitted as bases; question,
// LLM, arm and ordinary targets reject at lookup. Each hop resolves
// through its own declaring file so chains may cross files and packages.
func (w *World) WrapperOrigin(file *File, symbol *Symbol) (base, root *Symbol, header *syntax.NativeHeader, err error) {
	declaration, ok := symbol.Declaration.(*syntax.WrapDecl)
	if !ok {
		return nil, nil, nil, fmt.Errorf("wrapper origin requires a wrap declaration")
	}
	base, err = file.Lookup(nil, declaration.Base, WrapBaseUse)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("wrap requires a fetch, judge or wrapper base: %w", err)
	}
	seen := map[string]bool{symbol.ID: true}
	chain := []*Symbol{symbol}
	current, hop := base, file
	closer, closerFile := declaration, file
	for {
		if seen[current.ID] {
			err := fmt.Errorf("wrapper base cycle at %s", current.ID)
			err = source.LocateCode(closerFile.Source.Syntax.Source.Name(), closer.Base.Span, "CAN-CHECK-BOUND-CYCLE", err)
			for _, link := range chain {
				err = relateDeclaration(link, fmt.Sprintf("override chain passes through %s", link.ID), err)
			}
			return nil, nil, nil, err
		}
		seen[current.ID] = true
		chain = append(chain, current)
		next, ok := current.Declaration.(*syntax.WrapDecl)
		if !ok {
			header = syntax.NativeSignature(current.Declaration)
			if header == nil {
				return nil, nil, nil, fmt.Errorf("wrapper base %s is not an operation", current.ID)
			}
			return base, current, header, nil
		}
		if hop = w.Files[current.Source]; hop == nil {
			return nil, nil, nil, fmt.Errorf("wrapper base %s has no declaring file", current.ID)
		}
		closer, closerFile = next, hop
		current, err = hop.Lookup(nil, next.Base, WrapBaseUse)
		if err != nil {
			return nil, nil, nil, err
		}
	}
}

// relateDeclaration links a failure to a declared symbol's declaration
// span. Symbols without a recorded declaration file are skipped.
func relateDeclaration(symbol *Symbol, note string, err error) error {
	if symbol == nil || symbol.Declaration == nil || symbol.Source == nil || symbol.Source.Syntax == nil || symbol.Source.Syntax.Source == nil {
		return err
	}
	return source.Relate(symbol.Source.Syntax.Source.Name(), symbol.Declaration.DeclSpan(), note, err)
}

func (w *World) wrapperSignature(file *File, scope *Scope, symbol *Symbol, declaration *syntax.WrapDecl) error {
	_, root, header, err := w.WrapperOrigin(file, symbol)
	if err != nil {
		return err
	}
	// Inherited spellings resolve in the root's declaring file; the wrapper
	// scope only receives the defined input symbols.
	origin := w.Files[root.Source]
	if origin == nil {
		return fmt.Errorf("wrapper root %s has no declaring file", root.ID)
	}
	connection, err := origin.Lookup(nil, header.Connection, ConnectionUse)
	if err != nil {
		return err
	}
	if symbol.Public && !connection.Public {
		return fmt.Errorf("exported wrapper exposes private connection %s", connection.ID)
	}
	if err := origin.checkType(origin.Scope, header.Result, symbol.Public, TypeUse); err != nil {
		return err
	}
	add := func(field syntax.Field) error {
		if err := origin.checkType(origin.Scope, field.Type, symbol.Public, TypeUse); err != nil {
			return err
		}
		_, callable := field.Type.(*syntax.CallableType)
		return scope.Define(&Symbol{Name: field.Name.Text, ID: symbol.ID + "/input/" + field.Name.Text, Kind: Value, Type: field.Type, Callable: callable})
	}
	for i, input := range header.Inputs {
		if input.Variadic && i != len(header.Inputs)-1 {
			return fmt.Errorf("variadic wrapper input must be last")
		}
		field := input.Field
		if input.Variadic {
			field.Type = &syntax.ArrayType{Element: field.Type}
		}
		if err := add(field); err != nil {
			return err
		}
	}
	var state []syntax.Field
	switch d := root.Declaration.(type) {
	case *syntax.JudgeDecl:
		state = d.State
	}
	for _, field := range state {
		if err := add(field); err != nil {
			return err
		}
	}
	w.NativeScopes[declaration] = scope
	return nil
}

func generatedQuestionRecord(q *syntax.QuestionDecl) *syntax.RecordDecl {
	record := &syntax.RecordDecl{DeclarationLocation: q.DeclarationLocation, Name: *q.RecordName}
	for _, option := range q.Options {
		if option.Name != nil {
			record.Fields = append(record.Fields, syntax.Field{Span: option.Span, Name: *option.Name, Type: q.Result})
		}
	}
	for _, binder := range q.Binders {
		record.Fields = append(record.Fields, syntax.Field{Span: binder.Name.Span, Name: binder.Name, Type: &syntax.NamedType{Name: syntax.QualifiedName{Name: "float"}}})
	}
	return record
}

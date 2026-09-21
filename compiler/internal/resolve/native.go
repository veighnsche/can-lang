package resolve

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

func (w *World) nativeSignature(file *File, scope *Scope, symbol *Symbol, declaration syntax.Declaration) error {
	header := syntax.NativeSignature(declaration)
	if _, err := file.Lookup(scope, header.Connection, ConnectionUse); err != nil {
		return err
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
		if err := add(input.Field); err != nil {
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

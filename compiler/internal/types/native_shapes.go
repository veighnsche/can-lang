package types

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// This projection discovers declaration shape only. It does not admit an
// expression: the sealed-type expression checker must validate it afterward.
func (b *Builder) nativeShape(file *resolve.File, scope *resolve.Scope, node syntax.Expr) (*Type, error) {
	switch n := node.(type) {
	case *syntax.GroupExpr:
		return b.nativeShape(file, scope, n.Value)
	case *syntax.UpdateExpr:
		return b.nativeShape(file, scope, n.Receiver)
	case *syntax.NameExpr:
		symbol, err := file.Lookup(scope, n.Name, resolve.ValueUse)
		if err != nil {
			return nil, err
		}
		if symbol.Type == nil {
			return nil, fmt.Errorf("spread value lacks a declared data type")
		}
		declaringFile := file
		if symbol.Source != nil {
			declaringFile = b.world.Files[symbol.Source]
		}
		return b.Resolve(declaringFile, symbol.Type, nil, false)
	case *syntax.ConstructorExpr:
		return b.Resolve(file, &syntax.NamedType{Name: n.Name, Arguments: n.Types}, nil, false)
	case *syntax.FieldExpr:
		receiver, err := b.nativeShape(file, scope, n.Receiver)
		if err != nil {
			return nil, err
		}
		if !receiver.defined {
			return nil, fmt.Errorf("cyclic generated record shape dependency")
		}
		for _, field := range receiver.fields {
			if field.Name == n.Field.Text {
				return field.Type, nil
			}
		}
		return nil, fmt.Errorf("unknown spread receiver field %s", n.Field.Text)
	case *syntax.IndexExpr:
		receiver, err := b.nativeShape(file, scope, n.Receiver)
		if err != nil {
			return nil, err
		}
		if receiver.kind == Array {
			return receiver.element, nil
		}
	case *syntax.CallExpr:
		var result *Type
		var err error
		switch callee := n.Invocation.Callee.(type) {
		case *syntax.NameExpr:
			symbol, e := file.Lookup(scope, callee.Name, resolve.CallUse)
			if e != nil {
				return nil, e
			}
			result, err = b.nativeCallResult(file, symbol, n.Invocation.Types)
		case *syntax.FieldExpr:
			receiver, e := b.nativeShape(file, scope, callee.Receiver)
			if e != nil {
				return nil, e
			}
			for _, field := range receiver.fields {
				if field.Name == callee.Field.Text && field.Type.kind == Callable {
					result = field.Type.result
					break
				}
			}
			if result == nil {
				result, err = b.nativeMethodResult(file, receiver, callee.Field.Text, n.Invocation.Types)
			}
		default:
			callable, e := b.nativeShape(file, scope, n.Invocation.Callee)
			if e != nil {
				return nil, e
			}
			if callable.kind != Callable {
				return nil, fmt.Errorf("spread callee is not callable")
			}
			result = callable.result
		}
		if err != nil {
			return nil, err
		}
		for _, method := range n.Methods {
			result, err = b.nativeMethodResult(file, result, method.Name.Text, method.Types)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}
	return nil, fmt.Errorf("generated spread requires statically declared record shape")
}
func (b *Builder) generatedQuestionFields(file *resolve.File, q *syntax.QuestionDecl) ([]Field, error) {
	result, err := b.Resolve(file, q.Result, nil, false)
	if err != nil {
		return nil, err
	}
	var fields []Field
	seen := map[string]bool{}
	add := func(name string, typ *Type) error {
		if seen[name] {
			return fmt.Errorf("duplicate generated field %s", name)
		}
		seen[name] = true
		fields = append(fields, Field{Name: name, Type: typ})
		return nil
	}
	for _, option := range q.Options {
		if option.Name != nil {
			if err = add(option.Name.Text, result); err != nil {
				return nil, err
			}
			continue
		}
		shape, e := b.nativeShape(file, b.world.NativeScopes[q], option.Spread)
		if e != nil {
			return nil, e
		}
		if shape.kind != Record || !shape.defined {
			return nil, fmt.Errorf("generated spread needs a complete ordinary record shape")
		}
		for _, field := range shape.fields {
			if err = add(field.Name, result); err != nil {
				return nil, err
			}
		}
	}
	for _, binder := range q.Binders {
		if err = add(binder.Name.Text, b.graph.scalar("float")); err != nil {
			return nil, err
		}
	}
	return fields, nil
}

func (b *Builder) nativeCallResult(file *resolve.File, symbol *resolve.Symbol, args []syntax.TypeNode) (*Type, error) {
	var result syntax.TypeNode
	switch declaration := symbol.Declaration.(type) {
	case *syntax.FunctionDecl:
		result = declaration.Result
	default:
		if header := syntax.NativeSignature(symbol.Declaration); header != nil {
			result = header.Result
		} else if callable, ok := symbol.Type.(*syntax.CallableType); ok {
			result = callable.Result
		}
	}
	if result == nil {
		return nil, fmt.Errorf("spread call has no declared result shape")
	}
	if len(args) != len(symbol.Parameters) {
		return nil, fmt.Errorf("spread call requires explicit concrete type arguments")
	}
	env := map[string]*Type{}
	for i, arg := range args {
		typ, e := b.Resolve(file, arg, nil, false)
		if e != nil {
			return nil, e
		}
		env[symbol.Parameters[i]] = typ
	}
	declaringFile := file
	if symbol.Source != nil {
		declaringFile = b.world.Files[symbol.Source]
	}
	return b.Resolve(declaringFile, result, env, false)
}
func (b *Builder) nativeMethodResult(file *resolve.File, receiver *Type, name string, args []syntax.TypeNode) (*Type, error) {
	if name == "slice" && receiver.kind == Array {
		return receiver, nil
	}
	for _, pkg := range b.world.Packages {
		for _, symbol := range pkg.Scope.Symbols {
			if symbol.ID == receiver.declaration {
				method, err := file.Method(symbol, name)
				if err != nil {
					return nil, err
				}
				return b.nativeCallResult(file, method, args)
			}
		}
	}
	return nil, fmt.Errorf("spread method has no declared receiver shape")
}

package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// CodecSpecialization admits only the two maintained catalogue operations.
// General source function specialization and inference remain separate work.
type CodecSpecialization struct {
	Operation string
	Data      *types.Type
	Contract  *types.Type
	Schema    types.CodecSchema
}

func codecOperation(identity string) bool {
	return identity == "can.std.codec@1::encode_json" || identity == "can.std.codec@1::decode_json"
}
func (c *programChecker) gatherCodec(file *resolve.File, callee syntax.Expr, args []syntax.TypeNode) error {
	name, ok := callee.(*syntax.NameExpr)
	if !ok {
		return nil
	}
	symbol, err := file.Lookup(nil, name.Name, resolve.CallUse)
	if err != nil || !codecOperation(symbol.ID) {
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("codec call requires one explicit concrete type argument")
	}
	data, err := c.gather(file, args[0])
	if err != nil {
		return err
	}
	key := symbol.ID + "<" + data.Identity() + ">"
	if c.codecs == nil {
		c.codecs = map[string]*CodecSpecialization{}
		c.codecParts = map[string][]*types.Type{}
	}
	if c.codecs[key] != nil {
		return nil
	}
	byteType := &syntax.NamedType{Name: syntax.QualifiedName{Package: "bytes", Name: "buffer"}}
	// Catalogue dependencies resolve in the maintained package namespace, not the
	// author's imports: using codec does not require a redundant bytes import.
	builtin := &resolve.File{Scope: c.world.Prelude, Imports: c.world.Packages}
	c.annotations[builtin] = map[string]*types.Type{}
	buffer, err := c.gather(builtin, byteType)
	if err != nil {
		return err
	}
	invalid, err := c.gather(builtin, &syntax.NamedType{Name: syntax.QualifiedName{Package: "codec", Name: "invalid_data"}})
	if err != nil {
		return err
	}
	result, input := buffer, data
	if symbol.ID == "can.std.codec@1::decode_json" {
		result, input = data, buffer
	}
	// Residual signatures can be derived after graph sealing; retain ingredients.
	c.codecs[key] = &CodecSpecialization{Operation: symbol.ID, Data: data}
	c.codecParts[key] = []*types.Type{result, input, invalid}
	if c.specializer != nil {
		if err = c.finishCodec(key); err != nil {
			return err
		}
	}
	return nil
}

func (c *programChecker) finishCodec(key string) error {
	special := c.codecs[key]
	parts := c.codecParts[key]
	var err error
	special.Schema, err = types.Schema(special.Data)
	if err != nil {
		return err
	}
	special.Contract, err = types.CallableOfChecked(parts[0], []*types.Type{parts[1]}, []*types.Type{parts[2]})
	if err != nil {
		return err
	}
	c.program.Intrinsics[key] = special.Contract
	c.program.Codecs = c.codecs
	if c.callables != nil {
		c.callables[key] = CallableDeclaration{Kind: resolve.Function, Contract: special.Contract, Names: []string{"input0"}, Near: []bool{false}}
	}
	return nil
}
func (c *programChecker) specializeCodec(file *resolve.File, scope *resolve.Scope, name syntax.QualifiedName, args []syntax.TypeNode) (ValueBinding, error) {
	symbol, err := file.Lookup(scope, name, resolve.CallUse)
	if err != nil {
		return ValueBinding{}, err
	}
	if !codecOperation(symbol.ID) || len(args) != 1 {
		return ValueBinding{}, fmt.Errorf("generic invocation requires specialization; codec expects one explicit type")
	}
	data, err := c.annotation(file, args[0], false)
	if err != nil {
		return ValueBinding{}, err
	}
	key := symbol.ID + "<" + data.Identity() + ">"
	special := c.codecs[key]
	if special == nil {
		return ValueBinding{}, fmt.Errorf("missing checked codec specialization")
	}
	return ValueBinding{Identity: key, Type: special.Contract}, nil
}

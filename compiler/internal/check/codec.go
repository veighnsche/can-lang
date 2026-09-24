package check

import (
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// CodecSpecialization admits the maintained catalogue codec operations:
// JSON encode/decode plus the B1-11 document decoders. General source
// function specialization and inference remain separate work.
type CodecSpecialization struct {
	Operation string
	Data      *types.Type
	Contract  *types.Type
	Schema    types.CodecSchema
}

const (
	codecEncodeJSON   = "can.std.codec@1::encode_json"
	codecDecodeJSON   = "can.std.codec@1::decode_json"
	codecDecodeTOML   = "can.std.codec@1::decode_toml"
	codecDecodeYAML   = "can.std.codec@1::decode_yaml"
	codecDecodeJSON5  = "can.std.codec@1::decode_json5"
	codecDecodeJSONL  = "can.std.codec@1::decode_jsonl"
	codecConsumeJSONL = "can.std.codec@1::consume_jsonl"
)

func codecOperation(identity string) bool {
	switch identity {
	case codecEncodeJSON, codecDecodeJSON, codecDecodeTOML, codecDecodeYAML,
		codecDecodeJSON5, codecDecodeJSONL, codecConsumeJSONL:
		return true
	}
	return false
}

// codecDecodeOperation admits the single-document decoders that map a
// buffer onto exactly T. JSONL decode and consume shape their own
// contracts below.
func codecDecodeOperation(identity string) bool {
	switch identity {
	case codecDecodeJSON, codecDecodeTOML, codecDecodeYAML, codecDecodeJSON5:
		return true
	}
	return false
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
	if name := types.OpaqueParameterName(data); name != "" {
		return fmt.Errorf("codec %s cannot use opaque type parameter %s from an exported generic declaration", symbol.ID, name)
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
	// Consume keeps data plus the failure ingredient; the reader, count
	// and handler contracts resolve at finish time through the sealed
	// specializer, which also serves the deferred program path.
	if symbol.ID == codecConsumeJSONL {
		c.codecs[key] = &CodecSpecialization{Operation: symbol.ID, Data: data}
		c.codecParts[key] = []*types.Type{data, invalid}
		if c.specializer != nil {
			if err = c.finishCodec(key); err != nil {
				return err
			}
		}
		return nil
	}
	// JSONL decode keeps the row: the array result derives at finish
	// time once the row graph seals.
	result, input := buffer, data
	if codecDecodeOperation(symbol.ID) || symbol.ID == codecDecodeJSONL {
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
	if special.Operation == codecConsumeJSONL {
		return c.finishConsume(key, special, parts)
	}
	result := parts[0]
	if special.Operation == codecDecodeJSONL {
		result, err = types.ArrayOfChecked(special.Data)
		if err != nil {
			return err
		}
	}
	special.Contract, err = types.CallableOfChecked(result, []*types.Type{parts[1]}, []*types.Type{parts[2]})
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

// finishConsume binds consume_jsonl<T> to (reader, handler) -> int. The
// handler is total: every record crosses as T into a void callback with
// an empty error bound, so failures stay the pump's own invalid_data,
// read_failed and cancelled. Parts carry data plus invalid_data; the
// reader, count and handler contracts resolve here through the sealed
// specializer.
func (c *programChecker) finishConsume(key string, special *CodecSpecialization, parts []*types.Type) error {
	data, invalid := parts[0], parts[1]
	count, err := c.catalogueType("int", map[string]*types.Type{})
	if err != nil {
		return err
	}
	unit, err := c.catalogueType("void", map[string]*types.Type{})
	if err != nil {
		return err
	}
	reader, err := c.catalogueType("stream::reader<bytes::buffer>", map[string]*types.Type{})
	if err != nil {
		return err
	}
	failed, err := c.catalogueType("stream::read_failed", map[string]*types.Type{})
	if err != nil {
		return err
	}
	cancelled, err := c.catalogueType("stream::cancelled", map[string]*types.Type{})
	if err != nil {
		return err
	}
	handler, err := types.CallableOfChecked(unit, []*types.Type{data}, nil)
	if err != nil {
		return err
	}
	special.Contract, err = types.CallableOfChecked(count, []*types.Type{reader, handler}, []*types.Type{invalid, failed, cancelled})
	if err != nil {
		return err
	}
	c.program.Intrinsics[key] = special.Contract
	c.program.Codecs = c.codecs
	if c.callables != nil {
		c.callables[key] = CallableDeclaration{Kind: resolve.Function, Contract: special.Contract, Names: []string{"reader", "callback"}, Near: []bool{false, false}}
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

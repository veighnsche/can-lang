// Package emit lowers checked current-language nodes to native TypeScript.
package emit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// DataExpression is an already-checked synchronous expression. Promise payload
// extraction and protected completion boxes belong to the callable/region pass.
type DataExpression struct {
	Code string
	Type *types.Type
}
type Replacement struct {
	Name  string
	Value DataExpression
}

// DataImports uses identifiers that cannot occur in authored Can source. The
// driver supplies the exact relative path of the private generation runtime.
func DataImports(runtimePath string) string {
	return "import { record as $canRecord, update as $canUpdate, array as $canArray } from " + quote(runtimePath) + ";\n"
}
func quote(s string) string { out, _ := json.Marshal(s); return string(out) }
func Record(typ *types.Type, arguments []DataExpression) (DataExpression, error) {
	if typ == nil || (typ.Kind() != types.Record && typ.Kind() != types.Error) || !types.Equal(typ, typ) {
		return DataExpression{}, fmt.Errorf("constructor requires checked ordinary nominal data")
	}
	fields := typ.Fields()
	if len(fields) != len(arguments) {
		return DataExpression{}, fmt.Errorf("constructor arity mismatch")
	}
	entries := make([]string, len(fields))
	for i, f := range fields {
		if !types.Assignable(arguments[i].Type, f.Type) {
			return DataExpression{}, fmt.Errorf("constructor field %s has wrong type", f.Name)
		}
		entries[i] = "[" + quote(f.Name) + ", (" + arguments[i].Code + ")]"
	}
	return DataExpression{Type: typ, Code: "$canRecord(" + quote(typ.Identity()) + ", [" + strings.Join(entries, ", ") + "])"}, nil
}
func Update(receiver DataExpression, replacements []Replacement) (DataExpression, error) {
	checked := make([]types.Replacement, len(replacements))
	entries := make([]string, len(replacements))
	for i, r := range replacements {
		checked[i] = types.Replacement{Name: r.Name, Type: r.Value.Type}
		entries[i] = "[" + quote(r.Name) + ", (" + r.Value.Code + ")]"
	}
	typ, err := types.CheckUpdate(receiver.Type, checked)
	if err != nil {
		return DataExpression{}, err
	}
	return DataExpression{Type: typ, Code: "$canUpdate((" + receiver.Code + "), [" + strings.Join(entries, ", ") + "])"}, nil
}
func Array(typ *types.Type, elements []DataExpression) (DataExpression, error) {
	if typ == nil || typ.Kind() != types.Array || !types.Equal(typ, typ) {
		return DataExpression{}, fmt.Errorf("array construction requires an expected concrete array type")
	}
	entries := make([]string, len(elements))
	for i, e := range elements {
		if !types.Assignable(e.Type, typ.Element()) {
			return DataExpression{}, fmt.Errorf("array element type mismatch")
		}
		entries[i] = "(" + e.Code + ")"
	}
	return DataExpression{Type: typ, Code: "$canArray([" + strings.Join(entries, ", ") + "])"}, nil
}

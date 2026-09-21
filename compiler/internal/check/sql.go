package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/sql"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// sqlScalar reports whether t is an admitted SQL value type: bool,
// signed-64-bit int (range is validated per value at runtime), finite
// float, scalar str, owned bytes, or one optional layer of these. I35 moves
// this predicate into the shared schema projection; keep the two identical
// until that extraction lands.
func sqlScalar(t *types.Type) bool {
	if t.Kind() == types.Primitive {
		switch t.Declaration() {
		case "bool", "int", "float", "str":
			return true
		}
		return false
	}
	if t.Kind() == types.Opaque && t.Declaration() == "can.std.bytes@1::buffer" {
		return true
	}
	if t.Kind() == types.Variant && t.Declaration() == "can.std.option@1::value" && len(t.Arguments()) == 1 {
		inner := t.Arguments()[0]
		if inner.Kind() == types.Variant {
			return false
		}
		if !sqlScalar(inner) {
			return false
		}
		var some, none bool
		for _, leaf := range t.Leaves() {
			switch leaf.Declaration() {
			case "can.std.option@1::some":
				some = true
			case "can.std.option@1::none":
				none = true
			}
		}
		return some && none
	}
	return false
}

// sqlRecord resolves a manifest type name to an ordinary record in the
// owning project. The name must be a bare package-qualified nominal; type
// arguments and unknown packages are rejected without consulting SQL.
func sqlRecord(world *resolve.World, projects map[string]*project.Project, owner, name string) (*resolve.Symbol, error) {
	parts := strings.Split(name, "::")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("type %q is not a bare package-qualified nominal", name)
	}
	owning, ok := projects[owner]
	if !ok {
		return nil, fmt.Errorf("unknown owning project %q", owner)
	}
	if !project.Identifier(parts[0]) || !project.Identifier(parts[1]) {
		return nil, fmt.Errorf("type %q is not a bare package-qualified nominal", name)
	}
	var symbol *resolve.Symbol
	for _, pkg := range world.Packages {
		if pkg.Source == nil || pkg.Source.Owner != owning || pkg.Name != parts[0] {
			continue
		}
		if candidate, ok := pkg.Scope.Symbols[parts[1]]; ok {
			symbol = candidate
		}
	}
	if symbol == nil {
		return nil, fmt.Errorf("type %q is not declared in project %q", name, owner)
	}
	if symbol.Kind != resolve.Record {
		return nil, fmt.Errorf("type %q is not an ordinary record", name)
	}
	return symbol, nil
}

// CheckSQLDescriptors validates every project's manifest SQL descriptors
// against the finished type model and the upstream parser: parameter and
// row types resolve to ordinary records with admitted scalar fields,
// parameter names match record fields in declaration order, and statements
// check via the SQL package. Descriptors resolve in their owning
// project's manifest only; cross-project sharing flows through checked
// descriptor values, never through merged name tables.
func CheckSQLDescriptors(graph *project.Graph, world *resolve.World, model *types.Model) ([]ir.SQLDescriptor, error) {
	byDeclaration := map[string]*types.Type{}
	for _, typ := range model.Types() {
		if _, exists := byDeclaration[typ.Declaration()]; !exists {
			byDeclaration[typ.Declaration()] = typ
		}
	}
	projects := map[string]*project.Project{}
	for key, p := range graph.Projects {
		projects[key] = p
	}
	var out []ir.SQLDescriptor
	for _, key := range projectKeys(graph) {
		names := make([]string, 0, len(graph.Projects[key].Manifest.SQL))
		for name := range graph.Projects[key].Manifest.SQL {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			manifest := graph.Projects[key].Manifest.SQL[name]
			checked, err := checkSQLDescriptor(byDeclaration, world, projects, key, name, manifest)
			if err != nil {
				return nil, err
			}
			out = append(out, checked)
		}
	}
	return out, nil
}

func checkSQLDescriptor(byDeclaration map[string]*types.Type, world *resolve.World, projects map[string]*project.Project, owner, name string, manifest project.SQLDescriptor) (ir.SQLDescriptor, error) {
	fail := func(format string, args ...any) (ir.SQLDescriptor, error) {
		return ir.SQLDescriptor{}, fmt.Errorf("sql %q/%s: %s", owner, name, fmt.Sprintf(format, args...))
	}
	paramSymbol, err := sqlRecord(world, projects, owner, manifest.ParameterType)
	if err != nil {
		return fail("%v", err)
	}
	param, ok := byDeclaration[paramSymbol.ID]
	if !ok || param.Kind() != types.Record || len(param.Arguments()) != 0 {
		return fail("parameter type %q is not a concrete ordinary record", manifest.ParameterType)
	}
	rowSymbol, err := sqlRecord(world, projects, owner, manifest.RowType)
	if err != nil {
		return fail("%v", err)
	}
	row, ok := byDeclaration[rowSymbol.ID]
	if !ok || row.Kind() != types.Record || len(row.Arguments()) != 0 {
		return fail("row type %q is not a concrete ordinary record", manifest.RowType)
	}
	fields := param.Fields()
	if len(fields) != len(manifest.Parameters) {
		return fail("parameter record %q has %d fields, manifest lists %d", manifest.ParameterType, len(fields), len(manifest.Parameters))
	}
	for i, field := range fields {
		if field.Name != manifest.Parameters[i] {
			return fail("parameter record field %d is %q, manifest lists %q", i+1, field.Name, manifest.Parameters[i])
		}
		if !sqlScalar(field.Type) {
			return fail("parameter %q has non-SQL type %s", field.Name, field.Type.Declaration())
		}
	}
	for _, field := range row.Fields() {
		if !sqlScalar(field.Type) {
			return fail("row field %q has non-SQL type %s", field.Name, field.Type.Declaration())
		}
	}
	checked, err := sql.CheckDescriptor(name, manifest.Statement, manifest.Parameters, manifest.Cardinality, manifest.RowLimitParameter)
	if err != nil {
		return ir.SQLDescriptor{}, err
	}
	return ir.SQLDescriptor{Owner: owner, ParamType: param.Identity(), RowType: row.Identity(), Parameters: append([]string(nil), manifest.Parameters...), Checked: checked}, nil
}

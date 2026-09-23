package check

import (
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/sql"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

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
		if _, err := types.SQLFieldOf(field.Type); err != nil {
			return fail("parameter %q has non-SQL type %s", field.Name, field.Type.Declaration())
		}
	}
	for _, field := range row.Fields() {
		if _, err := types.SQLFieldOf(field.Type); err != nil {
			return fail("row field %q has non-SQL type %s", field.Name, field.Type.Declaration())
		}
	}
	dialect, err := sql.ParseDialect(manifest.Dialect)
	if err != nil {
		return ir.SQLDescriptor{}, err
	}
	checked, err := sql.CheckDescriptorDialect(dialect, name, manifest.Statement, manifest.Parameters, manifest.Cardinality, manifest.RowLimitParameter)
	if err != nil {
		return ir.SQLDescriptor{}, err
	}
	return ir.SQLDescriptor{Owner: owner, ParamType: param.Identity(), RowType: row.Identity(), Parameters: append([]string(nil), manifest.Parameters...), Checked: checked}, nil
}

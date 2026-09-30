package check

import (
	"errors"
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/source"
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
	if symbol.Invalid != nil {
		return nil, &source.BlockedError{Dependency: symbol.ID}
	}
	if symbol.Kind != resolve.Record {
		return nil, fmt.Errorf("type %q is not an ordinary record", name)
	}
	return symbol, nil
}

func checkSQLDescriptor(byDeclaration map[string]*types.Type, world *resolve.World, projects map[string]*project.Project, owner, name string, manifest project.SQLDescriptor) (ir.SQLDescriptor, error) {
	var problems []error
	fail := func(field, format string, args ...any) {
		problems = append(problems, &sqlDescriptorError{Field: field, Err: fmt.Errorf("sql %q/%s: %s", owner, name, fmt.Sprintf(format, args...))})
	}
	resolveRecord := func(field, name string) *types.Type {
		symbol, err := sqlRecord(world, projects, owner, name)
		if err != nil {
			problems = append(problems, &sqlDescriptorError{Field: field, Err: err})
			return nil
		}
		typ, ok := byDeclaration[symbol.ID]
		if !ok || typ.Kind() != types.Record || len(typ.Arguments()) != 0 {
			fail(field, "type %q is not a concrete ordinary record", name)
			return nil
		}
		return typ
	}
	param := resolveRecord("parameter_type", manifest.ParameterType)
	row := resolveRecord("row_type", manifest.RowType)
	if param != nil {
		fields := param.Fields()
		if len(fields) != len(manifest.Parameters) {
			fail("parameters", "parameter record %q has %d fields, manifest lists %d", manifest.ParameterType, len(fields), len(manifest.Parameters))
		}
		for i, field := range fields {
			if i < len(manifest.Parameters) && field.Name != manifest.Parameters[i] {
				fail("parameters", "parameter record field %d is %q, manifest lists %q", i+1, field.Name, manifest.Parameters[i])
			}
			if _, err := types.SQLFieldOf(field.Type); err != nil {
				fail("parameter_type", "parameter %q has non-SQL type %s", field.Name, field.Type.Declaration())
			}
		}
	}
	if row != nil {
		for _, field := range row.Fields() {
			if _, err := types.SQLFieldOf(field.Type); err != nil {
				fail("row_type", "row field %q has non-SQL type %s", field.Name, field.Type.Declaration())
			}
		}
	}
	dialect, err := sql.ParseDialect(manifest.Dialect)
	if err != nil {
		problems = append(problems, &sqlDescriptorError{Field: "dialect", Err: err})
		return ir.SQLDescriptor{}, errors.Join(problems...)
	}
	checked, err := sql.CheckDescriptorDialect(dialect, name, manifest.Statement, manifest.Parameters, manifest.Cardinality, manifest.RowLimitParameter)
	if err != nil {
		problems = append(problems, &sqlDescriptorError{Field: "statement", Err: err})
	}
	if len(problems) != 0 {
		return ir.SQLDescriptor{}, errors.Join(problems...)
	}
	return ir.SQLDescriptor{Owner: owner, ParamType: param.Identity(), RowType: row.Identity(), Parameters: append([]string(nil), manifest.Parameters...), Checked: checked}, nil
}

// sqlDescriptorError carries the canonical manifest field chosen by its validator.
type sqlDescriptorError struct {
	Field string
	Err   error
}

func (e *sqlDescriptorError) Error() string { return e.Err.Error() }
func (e *sqlDescriptorError) Unwrap() error { return e.Err }

package project

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// SQLDescriptor is one manifest SQL descriptor: the raw statement text
// plus its declared shape. Validation against the SQL grammar backends
// happens in check/sql_descriptors.go; this file only decodes.
type SQLDescriptor struct {
	Dialect, Statement, ParameterType, RowType, Cardinality string
	Parameters                                              []string
	RowLimitParameter                                       uint64
}

// decodeSQLMap decodes the manifest "sql" table in name order. Names must
// be nonempty and NUL-free; each entry decodes via parseSQL. The envelope
// owns the field lookup; this owns the entries.
func decodeSQLMap(raw json.RawMessage) (map[string]SQLDescriptor, error) {
	out := map[string]SQLDescriptor{}
	values, err := dictionary(raw)
	if err != nil {
		return nil, fmt.Errorf("sql: %w", err)
	}
	for _, name := range sortedKeys(values) {
		if name == "" || strings.ContainsRune(name, 0) {
			return nil, fmt.Errorf("invalid SQL descriptor name")
		}
		descriptor, err := parseSQL(values[name])
		if err != nil {
			return nil, fmt.Errorf("sql %q: %w", name, err)
		}
		out[name] = descriptor
	}
	return out, nil
}

func parseSQL(raw json.RawMessage) (SQLDescriptor, error) {
	d := SQLDescriptor{}
	fields, err := object(raw, []string{"dialect", "statement", "parameters", "parameter_type", "row_type", "cardinality"}, []string{"row_limit_parameter"})
	if err != nil {
		return d, err
	}
	destinations := map[string]*string{"dialect": &d.Dialect, "statement": &d.Statement, "parameter_type": &d.ParameterType, "row_type": &d.RowType, "cardinality": &d.Cardinality}
	for _, key := range sortedKeys(destinations) {
		dest := destinations[key]
		value, err := text(fields[key])
		if err != nil || value == "" {
			return d, fmt.Errorf("%s must be a nonempty string", key)
		}
		*dest = value
	}
	if d.Dialect != "postgresql" && d.Dialect != "sqlite" && d.Dialect != "mysql" {
		return d, fmt.Errorf("unsupported SQL dialect")
	}
	if d.Cardinality != "one" && d.Cardinality != "optional" && d.Cardinality != "many" && d.Cardinality != "execute" {
		return d, fmt.Errorf("invalid SQL cardinality")
	}
	parameters, err := array(fields["parameters"])
	if err != nil {
		return d, err
	}
	seen := map[string]bool{}
	for _, raw := range parameters {
		value, err := text(raw)
		if err != nil || !Identifier(value) || seen[value] {
			return d, fmt.Errorf("invalid or duplicate SQL parameter name")
		}
		seen[value] = true
		d.Parameters = append(d.Parameters, value)
	}
	for _, name := range []string{d.ParameterType, d.RowType} {
		file, err := source.New("<manifest-type>", name)
		if err != nil {
			return d, err
		}
		typ, diagnostics := syntax.ParseType(file)
		if len(diagnostics) > 0 {
			return d, fmt.Errorf("invalid SQL type %q", name)
		}
		nominal, ok := typ.(*syntax.NamedType)
		if !ok || nominal.Name.Package == "" {
			return d, fmt.Errorf("SQL types must be fully qualified nominal types")
		}
	}
	limit, exists := fields["row_limit_parameter"]
	if d.Cardinality == "execute" {
		if exists {
			return d, fmt.Errorf("execute cannot declare row_limit_parameter")
		}
	} else {
		if !exists {
			return d, fmt.Errorf("row-returning SQL requires row_limit_parameter")
		}
		d.RowLimitParameter, err = integer(limit)
		if err != nil || d.RowLimitParameter != uint64(len(d.Parameters))+1 {
			return d, fmt.Errorf("row_limit_parameter must follow the application parameters")
		}
	}
	return d, nil
}

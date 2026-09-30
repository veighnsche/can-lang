package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
)

// SQLDescriptor is one manifest SQL descriptor. SQL backend validation remains
// compiler-owned; configuration parsing never executes or connects to a database.
type SQLDescriptor struct {
	Dialect, Statement, ParameterType, RowType, Cardinality string
	Parameters                                              []string
	RowLimitParameter                                       uint64
}

func decodeSQLMap(raw json.RawMessage) (map[string]SQLDescriptor, map[string]error, error) {
	out := map[string]SQLDescriptor{}
	invalid := map[string]error{}
	values, err := dictionary(raw)
	if err != nil {
		return out, invalid, jsonFieldError(raw, fmt.Errorf("sql: %w", err))
	}
	var problems []error
	for _, name := range sortedKeys(values) {
		if name == "" || strings.ContainsRune(name, 0) {
			issue := jsonKeyError(raw, fmt.Errorf("invalid SQL descriptor name"), name)
			invalid[name] = issue
			problems = append(problems, issue)
			continue
		}
		descriptor, err := parseSQL(values[name])
		if err != nil {
			issue := jsonChildError(raw, err, name)
			invalid[name] = issue
			problems = append(problems, issue)
			continue
		}
		out[name] = descriptor
	}
	return out, invalid, errors.Join(problems...)
}

func parseSQL(raw json.RawMessage) (SQLDescriptor, error) {
	d := SQLDescriptor{}
	fields, shapeErr := object(raw, []string{"dialect", "statement", "parameters", "parameter_type", "row_type", "cardinality"}, []string{"row_limit_parameter"})
	if fields == nil {
		return d, shapeErr
	}
	problems := []error{shapeErr}
	add := func(err error, path ...string) { problems = append(problems, jsonFieldError(raw, err, path...)) }
	destinations := map[string]*string{"dialect": &d.Dialect, "statement": &d.Statement, "parameter_type": &d.ParameterType, "row_type": &d.RowType, "cardinality": &d.Cardinality}
	for _, key := range sortedKeys(destinations) {
		value, exists := fields[key]
		if !exists {
			continue
		}
		text, err := text(value)
		if err != nil || text == "" {
			add(fmt.Errorf("%s must be a nonempty string", key), key)
			continue
		}
		*destinations[key] = text
	}
	if d.Dialect != "" && d.Dialect != "postgresql" && d.Dialect != "sqlite" && d.Dialect != "mysql" {
		add(fmt.Errorf("unsupported SQL dialect"), "dialect")
	}
	cardinalityValid := d.Cardinality == "one" || d.Cardinality == "optional" || d.Cardinality == "many" || d.Cardinality == "execute"
	if d.Cardinality != "" && !cardinalityValid {
		add(fmt.Errorf("invalid SQL cardinality"), "cardinality")
	}
	parametersValid := false
	if value, exists := fields["parameters"]; exists {
		parameters, err := array(value)
		if err != nil {
			add(err, "parameters")
		} else {
			parametersValid = true
			seen := map[string]int{}
			for i, value := range parameters {
				name, err := text(value)
				if err != nil || !Identifier(name) {
					parametersValid = false
					add(fmt.Errorf("invalid SQL parameter name"), "parameters", strconv.Itoa(i))
					continue
				}
				if first, duplicate := seen[name]; duplicate {
					parametersValid = false
					problems = append(problems, &JSONError{Span: JSONFieldSpan(raw, "parameters", strconv.Itoa(i)), Related: []source.Span{JSONFieldSpan(raw, "parameters", strconv.Itoa(first))}, Err: fmt.Errorf("duplicate SQL parameter name")})
				} else {
					seen[name] = i
				}
				d.Parameters = append(d.Parameters, name)
			}
		}
	}
	for _, key := range []string{"parameter_type", "row_type"} {
		name := *destinations[key]
		if name == "" {
			continue
		}
		file, err := source.New("<manifest-type>", name)
		if err != nil {
			add(err, key)
			continue
		}
		typ, diagnostics := syntax.ParseType(file)
		if len(diagnostics) > 0 {
			add(fmt.Errorf("invalid SQL type %q", name), key)
			continue
		}
		nominal, ok := typ.(*syntax.NamedType)
		if !ok || nominal.Name.Package == "" {
			add(fmt.Errorf("SQL types must be fully qualified nominal types"), key)
		}
	}
	limit, exists := fields["row_limit_parameter"]
	limitValid := false
	if exists && d.Cardinality != "execute" {
		n, err := integer(limit)
		if err != nil {
			add(fmt.Errorf("row_limit_parameter must be an unsigned integer"), "row_limit_parameter")
		} else {
			d.RowLimitParameter, limitValid = n, true
		}
	}
	if cardinalityValid {
		if d.Cardinality == "execute" {
			if exists {
				add(fmt.Errorf("execute cannot declare row_limit_parameter"), "row_limit_parameter")
			}
		} else if !exists {
			// A one-row RETURNING descriptor has no row-limit placeholder.
			if d.Cardinality != "one" {
				add(fmt.Errorf("row-returning SQL requires row_limit_parameter"), "row_limit_parameter")
			}
		} else if limitValid && parametersValid && !(d.Cardinality == "one" && d.RowLimitParameter == 0) && d.RowLimitParameter != uint64(len(d.Parameters))+1 {
			add(fmt.Errorf("row_limit_parameter must follow the application parameters"), "row_limit_parameter")
		}
	}
	return d, errors.Join(problems...)
}

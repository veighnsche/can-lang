package sql

import "fmt"

// Segment is one piece of a checked statement: static Text, or Param N with
// 1-based parameter number. Segments tile the exact statement bytes; text
// resembling parameters inside quotes or comments stays inside Text.
type Segment struct {
	Text  string `json:"text,omitempty"`
	Param int    `json:"param,omitempty"`
}

// Descriptor is a fully checked manifest descriptor statement: dialect,
// cardinality shape, contiguous parameters, and template segments. Type
// correspondence with the parameter and row records is resolved by
// check/sql_descriptors.go; this package never sees Can types.
type Descriptor struct {
	Name        string    `json:"name"`
	Dialect     Dialect   `json:"dialect"`
	Cardinality string    `json:"cardinality"`
	Kind        string    `json:"kind"`
	Segments    []Segment `json:"segments"`
	// Total is the highest parameter number: len(parameters)+1 for
	// row-returning descriptors (limit included), len(parameters) for
	// execute. Limit is the row-limit parameter number, 0 for execute.
	Total   int `json:"total"`
	Limit   int `json:"limit"`
	Version int `json:"version"`
}

func rowReturning(cardinality string) bool { return cardinality != "execute" }

// CheckDescriptor validates one manifest descriptor statement: exactly one
// approved statement, contiguous parameters with the top-level LIMIT as
// the exclusive trailing number for row-returning shapes, and no
// RETURNING for mutations. It returns template segments tiling the input.
func CheckDescriptor(name, statement string, parameters []string, cardinality string, rowLimit uint64) (Descriptor, error) {
	return CheckDescriptorDialect(DialectPostgreSQL, name, statement, parameters, cardinality, rowLimit)
}

// CheckDescriptorDialect validates one descriptor through the named
// dialect backend. PostgreSQL is fully wired; SQLite arrives with its
// grammar adapter; any other tag is rejected before parsing.
func CheckDescriptorDialect(dialect Dialect, name, statement string, parameters []string, cardinality string, rowLimit uint64) (Descriptor, error) {
	out := Descriptor{Name: name, Dialect: dialect, Cardinality: cardinality}
	fail := func(format string, args ...any) (Descriptor, error) {
		return Descriptor{}, fmt.Errorf("sql descriptor %q: %s", name, fmt.Sprintf(format, args...))
	}
	if rowReturning(cardinality) {
		out.Total = len(parameters) + 1
		out.Limit = int(rowLimit)
		if rowLimit != uint64(out.Total) {
			return fail("row_limit_parameter is %d, want %d after %d application parameters", rowLimit, out.Total, len(parameters))
		}
	} else {
		out.Total = len(parameters)
		if rowLimit != 0 {
			return fail("execute must not declare row_limit_parameter")
		}
	}
	analysis, err := Analyze(dialect, name, statement)
	if err != nil {
		return Descriptor{}, err
	}
	if analysis.Failure != nil && len(analysis.Statements) == 0 {
		return fail("%s at character %d", analysis.Failure.Message, analysis.Failure.Cursor)
	}
	if len(analysis.Statements) != 1 {
		return fail("%d statements, want 1", len(analysis.Statements))
	}
	if analysis.Failure != nil {
		return fail("%s at character %d", analysis.Failure.Message, analysis.Failure.Cursor)
	}
	stmt := analysis.Statements[0]
	out.Kind = stmt.Kind
	out.Version = analysis.Version
	present, err := CheckSites(name, analysis.Sites, out.Total)
	if err != nil {
		return Descriptor{}, err
	}
	if err := CheckCardinality(name, stmt, analysis.Sites, cardinality, out.Total, out.Limit); err != nil {
		return Descriptor{}, err
	}
	if err := CheckCoverage(name, parameters, present, out.Total, out.Limit, rowReturning(cardinality)); err != nil {
		return Descriptor{}, err
	}
	out.Segments = TileSegments(statement, analysis.Sites)
	return out, nil
}

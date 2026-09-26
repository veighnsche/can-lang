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
	// row-returning SELECT descriptors (limit included), len(parameters)
	// for execute and for admitted INSERT ... RETURNING descriptors,
	// which bind no row-limit site. Limit is the row-limit parameter
	// number, 0 for execute and RETURNING. Returning marks the one
	// admitted RETURNING shape: INSERT with cardinality one on a
	// RETURNING-capable dialect. Downstream, (cardinality one, limit 0)
	// holds exactly for these descriptors.
	Total     int  `json:"total"`
	Limit     int  `json:"limit"`
	Version   int  `json:"version"`
	Returning bool `json:"returning,omitempty"`
}

func rowReturning(cardinality string) bool { return cardinality != "execute" }

// CheckDescriptor validates one manifest descriptor statement: exactly one
// approved statement, contiguous parameters with the top-level LIMIT as
// the exclusive trailing number for row-returning SELECT shapes, and the
// one admitted RETURNING shape (INSERT with cardinality one, no row-limit
// site) on RETURNING-capable dialects. It returns template segments tiling
// the input.
func CheckDescriptor(name, statement string, parameters []string, cardinality string, rowLimit uint64) (Descriptor, error) {
	return CheckDescriptorDialect(DialectPostgreSQL, name, statement, parameters, cardinality, rowLimit)
}

// CheckDescriptorDialect validates one descriptor through the named
// dialect backend. Any other tag is rejected before parsing. Analysis
// runs before the parameter-count checks because the admitted RETURNING
// shape carries no row-limit site: Total and Limit depend on whether the
// single statement returns rows via LIMIT or via RETURNING. Grammar
// failures therefore report before row-limit mismatches; the manifest
// layer keeps the two consistent in real flows.
func CheckDescriptorDialect(dialect Dialect, name, statement string, parameters []string, cardinality string, rowLimit uint64) (Descriptor, error) {
	out := Descriptor{Name: name, Dialect: dialect, Cardinality: cardinality}
	fail := func(format string, args ...any) (Descriptor, error) {
		return Descriptor{}, fmt.Errorf("sql descriptor %q: %s", name, fmt.Sprintf(format, args...))
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
	if stmt.Returning {
		out.Returning = true
		out.Total = len(parameters)
		out.Limit = 0
		if rowReturning(cardinality) && rowLimit != 0 {
			return fail("RETURNING descriptors must not declare row_limit_parameter")
		}
		present, err := CheckSites(name, analysis.Sites, out.Total)
		if err != nil {
			return Descriptor{}, err
		}
		if err := CheckCardinality(name, dialect, stmt, analysis.Sites, cardinality, out.Total, out.Limit); err != nil {
			return Descriptor{}, err
		}
		if err := CheckCoverage(name, dialect, parameters, present, out.Total, out.Limit, false); err != nil {
			return Descriptor{}, err
		}
		if dialect == DialectSQLite {
			if err := CheckSiteNames(name, parameters, analysis.Sites, out.Limit); err != nil {
				return Descriptor{}, err
			}
		}
		out.Segments = TileSegments(statement, analysis.Sites)
		return out, nil
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
	present, err := CheckSites(name, analysis.Sites, out.Total)
	if err != nil {
		return Descriptor{}, err
	}
	if err := CheckCardinality(name, dialect, stmt, analysis.Sites, cardinality, out.Total, out.Limit); err != nil {
		return Descriptor{}, err
	}
	if err := CheckCoverage(name, dialect, parameters, present, out.Total, out.Limit, rowReturning(cardinality)); err != nil {
		return Descriptor{}, err
	}
	if dialect == DialectSQLite {
		if err := CheckSiteNames(name, parameters, analysis.Sites, out.Limit); err != nil {
			return Descriptor{}, err
		}
	}
	out.Segments = TileSegments(statement, analysis.Sites)
	return out, nil
}

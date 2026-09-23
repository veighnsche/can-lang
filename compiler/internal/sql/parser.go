// Package sql validates manifest SQL descriptors against pinned,
// dialect-specific grammar backends. PostgreSQL parses and scans through
// the actual PostgreSQL 17 grammar and lexer via libpg_query; SQLite
// parses through the pinned tree-sitter parse.y mirror. Each backend
// converts its results into the small compiler-owned Analysis shape here:
// statements, parameter sites with byte spans, kinds, LIMIT shapes,
// RETURNING presence, and verbatim failures. No backend AST node, token
// object, or upstream error type crosses this boundary: descriptors.go
// consumes statements, sites, and failures, and classifies from structure,
// never by matching message text.
package sql

import "fmt"

// Statement is one analyzed statement: its byte span plus the backend's
// top-level kind tag such as SelectStmt. HasLimit reports a top-level
// LIMIT clause; LimitParam is its parameter number when the limit is
// exactly one parameter site, else 0. Returning reports a RETURNING list
// on INSERT/UPDATE/DELETE.
type Statement struct {
	Location   int
	Length     int
	Kind       string
	HasLimit   bool
	LimitParam int
	Returning  bool
}

// ParamSite is one parameter occurrence: its 1-based parameter number
// plus byte offsets into the statement. Sites arrive in source order.
type ParamSite struct {
	Number     int
	Start, End int
}

// Failure is an upstream grammar failure, verbatim: the backend message
// plus a source offset. Callers must classify from structure (statement
// counts, site shapes, spans), never by matching the message text, which
// can change across backend releases.
type Failure struct {
	Message string
	Cursor  int
}

// Analysis is one backend-neutral statement analysis: every statement
// span, every parameter site in source order, or the grammar failure.
// Version is the backend's pinned version stamp, recorded on checked
// descriptors. A nil failure with empty statements means no statements,
// not an error.
type Analysis struct {
	Statements []Statement
	Sites      []ParamSite
	Failure    *Failure
	Version    int
}

// Analyze runs the dialect backend over one descriptor statement. The
// name only prefixes backend-shape errors; grammar failures stay
// verbatim inside the returned analysis.
func Analyze(dialect Dialect, name, statement string) (Analysis, error) {
	switch dialect {
	case DialectPostgreSQL:
		return analyzePostgres(name, statement)
	default:
		return Analysis{}, fmt.Errorf("sql descriptor %q: unsupported SQL dialect %q", name, dialect)
	}
}

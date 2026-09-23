package sql

import (
	"fmt"
	"strconv"
)

// Dialect is a manifest SQL dialect tag. PostgreSQL validates through the
// pinned libpg_query adaptation in postgres.go; SQLite through the
// tree-sitter parse.y mirror in sqlite.go; MySQL through the pinned
// tidb parser in mysql.go. No dialect text is ever fed through another
// dialect's backend.
type Dialect string

const (
	// DialectPostgreSQL is the "postgresql" manifest tag, validated by
	// the pinned PostgreSQL 17 grammar and lexer.
	DialectPostgreSQL Dialect = "postgresql"
	// DialectSQLite is the "sqlite" manifest tag, validated by the
	// pinned tree-sitter SQLite grammar.
	DialectSQLite Dialect = "sqlite"
	// DialectMySQL is the "mysql" manifest tag, validated by the
	// pinned tidb parser.
	DialectMySQL Dialect = "mysql"
)

// ParseDialect resolves a manifest dialect tag to its backend identity.
func ParseDialect(tag string) (Dialect, error) {
	switch Dialect(tag) {
	case DialectPostgreSQL, DialectSQLite, DialectMySQL:
		return Dialect(tag), nil
	default:
		return "", fmt.Errorf("unsupported SQL dialect %q", tag)
	}
}

// limitRef renders a parameter number for diagnostics in the dialect's own
// spelling: $N for PostgreSQL, ?N for SQLite and MySQL.
func limitRef(dialect Dialect, number int) string {
	if dialect == DialectPostgreSQL {
		return "$" + strconv.Itoa(number)
	}
	return "?" + strconv.Itoa(number)
}

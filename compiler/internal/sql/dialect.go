package sql

import "fmt"

// Dialect is a manifest SQL dialect tag. PostgreSQL validates through the
// pinned libpg_query adaptation in postgres.go; SQLite through the
// tree-sitter parse.y mirror in sqlite.go. No dialect text is ever fed
// through another dialect's backend.
type Dialect string

const (
	// DialectPostgreSQL is the "postgresql" manifest tag, validated by
	// the pinned PostgreSQL 17 grammar and lexer.
	DialectPostgreSQL Dialect = "postgresql"
	// DialectSQLite is the "sqlite" manifest tag, validated by the
	// pinned tree-sitter SQLite grammar.
	DialectSQLite Dialect = "sqlite"
)

// ParseDialect resolves a manifest dialect tag to its backend identity.
// MySQL arrives with B1-03; until then only two tags resolve.
func ParseDialect(tag string) (Dialect, error) {
	switch Dialect(tag) {
	case DialectPostgreSQL, DialectSQLite:
		return Dialect(tag), nil
	default:
		return "", fmt.Errorf("unsupported SQL dialect %q", tag)
	}
}

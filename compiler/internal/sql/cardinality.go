package sql

import "fmt"

// selectKinds admits the backend kind tags that count as SELECT for
// row-returning cardinalities. Mutations admit their own tags.
func selectKinds(kind string) bool {
	switch kind {
	case "SelectStmt", "select_statement":
		return true
	}
	return false
}

func mutationKinds(kind string) bool {
	switch kind {
	case "InsertStmt", "UpdateStmt", "DeleteStmt",
		"insert_statement", "update_statement", "delete_statement":
		return true
	}
	return false
}

// insertKinds admits the backend kind tags for INSERT, the only statement
// whose RETURNING clause the qualified F03 slice admits.
func insertKinds(kind string) bool {
	switch kind {
	case "InsertStmt", "insert_statement":
		return true
	}
	return false
}

// CheckCardinality enforces the statement shape a cardinality requires:
// SELECT with a top-level LIMIT binding the trailing number exactly once
// for row-returning shapes, a RETURNING-free mutation for execute, or the
// one admitted RETURNING shape: INSERT with cardinality one on a
// RETURNING-capable dialect. Every RETURNING rejection keeps the
// "RETURNING is not admitted" wording the corpus category pins.
// Backends report kinds in their own tags; both tag families admit here.
// Limit numbers render in the dialect's spelling.
func CheckCardinality(name string, dialect Dialect, stmt Statement, sites []ParamSite, cardinality string, total, limit int) error {
	if stmt.Returning {
		if dialect == DialectMySQL {
			return fmt.Errorf("sql descriptor %q: RETURNING is not admitted on mysql; insert, then SELECT ... WHERE id = LAST_INSERT_ID() in one transaction", name)
		}
		if cardinality != "one" {
			return fmt.Errorf("sql descriptor %q: RETURNING is not admitted under cardinality %s; only INSERT ... RETURNING with cardinality one is admitted", name, cardinality)
		}
		if !insertKinds(stmt.Kind) {
			return fmt.Errorf("sql descriptor %q: RETURNING is not admitted on %s; only INSERT ... RETURNING is admitted", name, stmt.Kind)
		}
		return nil
	}
	if cardinality != "execute" {
		if !selectKinds(stmt.Kind) {
			return fmt.Errorf("sql descriptor %q: cardinality %s requires SELECT, got %s", name, cardinality, stmt.Kind)
		}
		if !stmt.HasLimit {
			return fmt.Errorf("sql descriptor %q: unbounded SELECT requires LIMIT %s", name, limitRef(dialect, limit))
		}
		if stmt.LimitParam != limit {
			return fmt.Errorf("sql descriptor %q: top-level LIMIT must be %s", name, limitRef(dialect, limit))
		}
		uses := 0
		for _, site := range sites {
			if site.Number == limit {
				uses++
			}
		}
		if uses != 1 {
			return fmt.Errorf("sql descriptor %q: row-limit parameter %s appears %d times, want once in LIMIT", name, limitRef(dialect, limit), uses)
		}
		return nil
	}
	if !mutationKinds(stmt.Kind) {
		return fmt.Errorf("sql descriptor %q: execute requires INSERT, UPDATE, or DELETE, got %s", name, stmt.Kind)
	}
	return nil
}

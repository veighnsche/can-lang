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

// CheckCardinality enforces the statement shape a cardinality requires:
// SELECT with a top-level LIMIT binding the trailing number exactly once
// for row-returning shapes, or a RETURNING-free mutation for execute.
// Backends report kinds in their own tags; both tag families admit here.
func CheckCardinality(name string, stmt Statement, sites []ParamSite, cardinality string, total, limit int) error {
	if cardinality != "execute" {
		if !selectKinds(stmt.Kind) {
			return fmt.Errorf("sql descriptor %q: cardinality %s requires SELECT, got %s", name, cardinality, stmt.Kind)
		}
		if !stmt.HasLimit {
			return fmt.Errorf("sql descriptor %q: unbounded SELECT requires LIMIT $%d", name, limit)
		}
		if stmt.LimitParam != limit {
			return fmt.Errorf("sql descriptor %q: top-level LIMIT must be $%d", name, limit)
		}
		uses := 0
		for _, site := range sites {
			if site.Number == limit {
				uses++
			}
		}
		if uses != 1 {
			return fmt.Errorf("sql descriptor %q: row-limit parameter $%d appears %d times, want once in LIMIT", name, limit, uses)
		}
		return nil
	}
	if !mutationKinds(stmt.Kind) {
		return fmt.Errorf("sql descriptor %q: execute requires INSERT, UPDATE, or DELETE, got %s", name, stmt.Kind)
	}
	if stmt.Returning {
		return fmt.Errorf("sql descriptor %q: RETURNING is not admitted", name)
	}
	return nil
}

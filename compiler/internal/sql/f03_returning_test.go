package sql

import (
	"strconv"
	"strings"
	"testing"
)

// TestF03ReturningAdmission pins the qualified F03 slice: INSERT ...
// RETURNING with cardinality one and no row-limit site on the
// RETURNING-capable dialects. Total is the application parameter count,
// Limit is 0, and the checked descriptor marks Returning so downstream
// (cardinality one, limit 0) holds exactly for this shape.
func TestF03ReturningAdmission(t *testing.T) {
	cases := []struct {
		dialect   Dialect
		statement string
		params    []string
		kind      string
		version   int
	}{
		{DialectPostgreSQL, "INSERT INTO f03_gen (payload) VALUES ($1) RETURNING id", []string{"payload"}, "InsertStmt", 170007},
		{DialectPostgreSQL, "INSERT INTO f03_gen (worker, payload) VALUES ($1, $2) RETURNING id, payload", []string{"worker", "payload"}, "InsertStmt", 170007},
		{DialectPostgreSQL, "INSERT INTO f03_keys (id, payload) VALUES ($1, $2) ON CONFLICT DO NOTHING RETURNING id", []string{"id", "payload"}, "InsertStmt", 170007},
		{DialectSQLite, "INSERT INTO t (a) VALUES (?1) RETURNING id", []string{"a"}, "insert_statement", 15},
		{DialectSQLite, "INSERT INTO t (a) VALUES (:a) RETURNING id", []string{"a"}, "insert_statement", 15},
	}
	for _, c := range cases {
		got, err := CheckDescriptorDialect(c.dialect, "f03", c.statement, c.params, "one", 0)
		if err != nil {
			t.Fatalf("%s %s: %v", c.dialect, c.statement, err)
		}
		if !got.Returning || got.Kind != c.kind || got.Cardinality != "one" {
			t.Fatalf("%s: %+v", c.statement, got)
		}
		if got.Total != len(c.params) || got.Limit != 0 || got.Version != c.version {
			t.Fatalf("%s: %+v", c.statement, got)
		}
		var rebuilt strings.Builder
		sites := 0
		for _, segment := range got.Segments {
			if segment.Param > 0 {
				sites++
				if c.dialect == DialectPostgreSQL {
					rebuilt.WriteString("$" + strconv.Itoa(segment.Param))
				} else {
					rebuilt.WriteString("?1")
				}
			} else {
				rebuilt.WriteString(segment.Text)
			}
		}
		if c.dialect == DialectPostgreSQL && rebuilt.String() != c.statement {
			t.Fatalf("%s: segments do not tile the statement", c.statement)
		}
		if c.dialect == DialectSQLite && strings.Contains(c.statement, ":a") {
			if rebuilt.String() != strings.Replace(c.statement, ":a", "?1", 1) {
				t.Fatalf("%s: segments do not tile the statement", c.statement)
			}
		}
		_ = sites
	}
}

// TestF03ReturningRejections pins every unnecessary RETURNING shape as
// rejected: non-INSERT carriers, non-one cardinalities, the mysql
// dialect (whose server has no RETURNING clause), and row-limit or
// coverage incoherence. All keep the corpus "RETURNING is not admitted"
// wording.
func TestF03ReturningRejections(t *testing.T) {
	cases := []struct {
		name       string
		dialect    Dialect
		statement  string
		params     []string
		cardinal   string
		limit      uint64
		wantSubstr string
	}{
		{"pg update", DialectPostgreSQL, "UPDATE t SET a = $1 RETURNING id", []string{"a"}, "one", 0, "RETURNING is not admitted on UpdateStmt"},
		{"pg delete", DialectPostgreSQL, "DELETE FROM t RETURNING id", nil, "one", 0, "RETURNING is not admitted on DeleteStmt"},
		{"lite update", DialectSQLite, "UPDATE t SET a = ?1 RETURNING id", []string{"a"}, "one", 0, "RETURNING is not admitted on update_statement"},
		{"lite delete", DialectSQLite, "DELETE FROM t RETURNING id", nil, "one", 0, "RETURNING is not admitted on delete_statement"},
		{"pg many", DialectPostgreSQL, "INSERT INTO t (a) VALUES ($1) RETURNING id", []string{"a"}, "many", 0, "RETURNING is not admitted under cardinality many"},
		{"pg optional", DialectPostgreSQL, "INSERT INTO t (a) VALUES ($1) RETURNING id", []string{"a"}, "optional", 0, "RETURNING is not admitted under cardinality optional"},
		{"pg execute", DialectPostgreSQL, "INSERT INTO t (a) VALUES ($1) RETURNING id", []string{"a"}, "execute", 0, "RETURNING is not admitted under cardinality execute"},
		{"lite execute", DialectSQLite, "INSERT INTO t (a) VALUES (?1) RETURNING id", []string{"a"}, "execute", 0, "RETURNING is not admitted under cardinality execute"},
		{"mysql insert", DialectMySQL, "INSERT INTO t (a) VALUES (?) RETURNING id", []string{"a"}, "one", 0, "RETURNING is not admitted on mysql"},
		{"mysql update", DialectMySQL, "UPDATE t SET a = ? RETURNING id", []string{"a"}, "one", 0, "RETURNING is not admitted on mysql"},
		{"mysql delete", DialectMySQL, "DELETE FROM t RETURNING id", nil, "one", 0, "RETURNING is not admitted on mysql"},
		{"row limit declared", DialectPostgreSQL, "INSERT INTO t (a) VALUES ($1) RETURNING id", []string{"a"}, "one", 2, "must not declare row_limit_parameter"},
		{"multi", DialectPostgreSQL, "INSERT INTO t (a) VALUES ($1) RETURNING id; SELECT $1", []string{"a"}, "one", 0, "2 statements"},
	}
	for _, c := range cases {
		_, err := CheckDescriptorDialect(c.dialect, c.name, c.statement, c.params, c.cardinal, c.limit)
		if err == nil || !strings.Contains(err.Error(), c.wantSubstr) {
			t.Fatalf("%s: %v, want %q", c.name, err, c.wantSubstr)
		}
	}
	if _, err := CheckDescriptorDialect(DialectMySQL, "map", "INSERT INTO t (a) VALUES (?) RETURNING id", []string{"a"}, "one", 0); err == nil || !strings.Contains(err.Error(), "LAST_INSERT_ID") {
		t.Fatalf("mysql mapping hint: %v", err)
	}
}

// TestF03ReturningCoverage pins parameter accounting for the admitted
// shape: every application parameter must appear, and a missing one
// names the declared parameter — never the row limit, which does not
// exist here. A zero row limit without RETURNING still fails as before.
func TestF03ReturningCoverage(t *testing.T) {
	_, err := CheckDescriptor("gap", "INSERT INTO t (a) VALUES ($1) RETURNING id", []string{"a", "b"}, "one", 0)
	if err == nil || !strings.Contains(err.Error(), `parameter "b" ($2) is not used`) {
		t.Fatalf("gap: %v", err)
	}
	_, err = CheckDescriptor("extra", "INSERT INTO t (a) VALUES ($1) RETURNING id", []string{"a"}, "one", 0)
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	_, err = CheckDescriptor("no-returning", "SELECT id FROM t WHERE x = $1 LIMIT $2", []string{"a"}, "one", 0)
	if err == nil || !strings.Contains(err.Error(), "row_limit_parameter is 0, want 2") {
		t.Fatalf("zero limit without RETURNING: %v", err)
	}
}

// TestF03MySQLMappingSelect pins the mysql side of the per-dialect
// contract: the LAST_INSERT_ID refetch is an ordinary checked SELECT
// needing no checker change.
func TestF03MySQLMappingSelect(t *testing.T) {
	got, err := CheckDescriptorDialect(DialectMySQL, "by_last_id", "SELECT id, payload FROM t WHERE id = LAST_INSERT_ID() LIMIT ?", []string{}, "one", 1)
	if err != nil {
		t.Fatalf("mapping select: %v", err)
	}
	if got.Kind != "SelectStmt" || got.Total != 1 || got.Limit != 1 || got.Returning {
		t.Fatalf("mapping select: %+v", got)
	}
}

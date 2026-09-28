package sql

import (
	"strings"
	"testing"
)

// This suite pins the SQLite backend contract with independently derived
// expectations: every span, number and cursor below is counted from the
// input by hand, never copied from backend output. Statement spans are
// the native half-open byte spans including any terminating semicolon.

// wantSite asserts one reported site equals the hand-derived value.
func wantSite(t *testing.T, statement string, sites []ParamSite, i int, want ParamSite) {
	t.Helper()
	if i >= len(sites) {
		t.Fatalf("%q: only %d sites, want %+v at %d", statement, len(sites), want, i)
	}
	if sites[i] != want {
		t.Fatalf("%q: site %d = %+v, want %+v", statement, i, sites[i], want)
	}
	slice := statement[want.Start:want.End]
	if strings.HasPrefix(want.Ref, "?") {
		// Bare sites render with their number; explicit spellings
		// render verbatim. Either way the span must hold ? or ?NNN.
		if slice != "?" && slice != want.Ref {
			t.Fatalf("%q: site %d span %q, want ref %q", statement, i, slice, want.Ref)
		}
	} else if slice != want.Ref {
		t.Fatalf("%q: site %d span %q, want ref %q", statement, i, slice, want.Ref)
	}
}

func analyzeOK(t *testing.T, statement string) Analysis {
	t.Helper()
	analysis, err := Analyze(DialectSQLite, "t", statement)
	if err != nil {
		t.Fatalf("Analyze(%q): %v", statement, err)
	}
	if analysis.Failure != nil {
		t.Fatalf("Analyze(%q): failure %+v", statement, analysis.Failure)
	}
	return analysis
}

func analyzeFailure(t *testing.T, statement string) Failure {
	t.Helper()
	analysis, err := Analyze(DialectSQLite, "t", statement)
	if err != nil {
		t.Fatalf("Analyze(%q): %v", statement, err)
	}
	if analysis.Failure == nil {
		t.Fatalf("Analyze(%q): want failure, got %+v", statement, analysis)
	}
	if len(analysis.Statements) != 0 || analysis.Version != 0 {
		t.Fatalf("Analyze(%q): failure must clear statements/version: %+v", statement, analysis)
	}
	return *analysis.Failure
}

func TestSQLiteUTF8Spans(t *testing.T) {
	statement := "SELECT \"näme\" FROM users WHERE id = ? LIMIT ?"
	analysis := analyzeOK(t, statement)
	if len(analysis.Statements) != 1 {
		t.Fatalf("statements %+v", analysis.Statements)
	}
	stmt := analysis.Statements[0]
	if stmt.Location != 0 || stmt.Length != 46 || stmt.Kind != "select_statement" {
		t.Fatalf("stmt %+v", stmt)
	}
	if !stmt.HasLimit || stmt.LimitParam != 2 {
		t.Fatalf("limit %+v", stmt)
	}
	if len(analysis.Sites) != 2 {
		t.Fatalf("sites %+v", analysis.Sites)
	}
	wantSite(t, statement, analysis.Sites, 0, ParamSite{Number: 1, Start: 37, End: 38, Ref: "?1"})
	wantSite(t, statement, analysis.Sites, 1, ParamSite{Number: 2, Start: 45, End: 46, Ref: "?2"})
	if analysis.Version != SQLiteVersion {
		t.Fatalf("version %d", analysis.Version)
	}
}

func TestSQLiteCommentDecoys(t *testing.T) {
	statement := "SELECT ? /* :a */ FROM t -- @b\nWHERE x = $c LIMIT ?"
	analysis := analyzeOK(t, statement)
	if len(analysis.Sites) != 3 {
		t.Fatalf("sites %+v", analysis.Sites)
	}
	wantSite(t, statement, analysis.Sites, 0, ParamSite{Number: 1, Start: 7, End: 8, Ref: "?1"})
	wantSite(t, statement, analysis.Sites, 1, ParamSite{Number: 2, Start: 41, End: 43, Ref: "$c"})
	wantSite(t, statement, analysis.Sites, 2, ParamSite{Number: 3, Start: 50, End: 51, Ref: "?3"})
	if stmt := analysis.Statements[0]; !stmt.HasLimit || stmt.LimitParam != 3 {
		t.Fatalf("limit %+v", stmt)
	}
}

func TestSQLiteStringDecoys(t *testing.T) {
	statement := "SELECT 'it''s ?' FROM t WHERE x = ? LIMIT ?"
	analysis := analyzeOK(t, statement)
	if len(analysis.Sites) != 2 {
		t.Fatalf("sites %+v", analysis.Sites)
	}
	wantSite(t, statement, analysis.Sites, 0, ParamSite{Number: 1, Start: 34, End: 35, Ref: "?1"})
	wantSite(t, statement, analysis.Sites, 1, ParamSite{Number: 2, Start: 42, End: 43, Ref: "?2"})
}

func TestSQLiteEmptyAndSeparators(t *testing.T) {
	for _, statement := range []string{"", "   ", "-- only a comment", ";", ";;;"} {
		analysis := analyzeOK(t, statement)
		if len(analysis.Statements) != 0 || len(analysis.Sites) != 0 {
			t.Fatalf("%q: %+v", statement, analysis)
		}
		if analysis.Version != SQLiteVersion {
			t.Fatalf("%q: version %d", statement, analysis.Version)
		}
	}
}

func TestSQLiteSemicolonSpansAreNative(t *testing.T) {
	analysis := analyzeOK(t, "SELECT name FROM users WHERE id = ? LIMIT ?;")
	stmt := analysis.Statements[0]
	if stmt.Location != 0 || stmt.Length != 44 {
		t.Fatalf("trailing semicolon span %+v, want 0+44", stmt)
	}
	multi := analyzeOK(t, "SELECT 1; SELECT 2")
	if len(multi.Statements) != 2 {
		t.Fatalf("statements %+v", multi.Statements)
	}
	if multi.Statements[0].Location != 0 || multi.Statements[0].Length != 9 {
		t.Fatalf("first span %+v, want 0+9", multi.Statements[0])
	}
	if multi.Statements[1].Location != 10 || multi.Statements[1].Length != 8 {
		t.Fatalf("second span %+v, want 10+8", multi.Statements[1])
	}
	if len(multi.Sites) != 0 {
		t.Fatalf("multi sites %+v", multi.Sites)
	}
	txn := analyzeOK(t, "BEGIN; COMMIT")
	if txn.Statements[0].Kind != "begin_statement" || txn.Statements[0].Length != 6 {
		t.Fatalf("begin %+v", txn.Statements[0])
	}
	if txn.Statements[1].Kind != "commit_statement" || txn.Statements[1].Location != 7 {
		t.Fatalf("commit %+v", txn.Statements[1])
	}
}

func TestSQLiteCTESites(t *testing.T) {
	statement := "WITH u AS (SELECT id FROM users WHERE id = ?) SELECT id FROM u LIMIT ?"
	analysis := analyzeOK(t, statement)
	wantSite(t, statement, analysis.Sites, 0, ParamSite{Number: 1, Start: 43, End: 44, Ref: "?1"})
	wantSite(t, statement, analysis.Sites, 1, ParamSite{Number: 2, Start: 69, End: 70, Ref: "?2"})
	multi := analyzeOK(t, "WITH a AS (SELECT * FROM t LIMIT ?), b AS (SELECT * FROM a LIMIT ?) SELECT * FROM b LIMIT ?")
	if len(multi.Sites) != 3 {
		t.Fatalf("sites %+v", multi.Sites)
	}
	wantSite(t, "WITH a AS (SELECT * FROM t LIMIT ?), b AS (SELECT * FROM a LIMIT ?) SELECT * FROM b LIMIT ?", multi.Sites, 0, ParamSite{Number: 1, Start: 33, End: 34, Ref: "?1"})
	wantSite(t, "WITH a AS (SELECT * FROM t LIMIT ?), b AS (SELECT * FROM a LIMIT ?) SELECT * FROM b LIMIT ?", multi.Sites, 1, ParamSite{Number: 2, Start: 65, End: 66, Ref: "?2"})
	wantSite(t, "WITH a AS (SELECT * FROM t LIMIT ?), b AS (SELECT * FROM a LIMIT ?) SELECT * FROM b LIMIT ?", multi.Sites, 2, ParamSite{Number: 3, Start: 90, End: 91, Ref: "?3"})
	if stmt := multi.Statements[0]; !stmt.HasLimit || stmt.LimitParam != 3 {
		t.Fatalf("limit %+v", stmt)
	}
}

func TestSQLiteNestedLimitsIgnored(t *testing.T) {
	analysis := analyzeOK(t, "SELECT * FROM (SELECT * FROM t LIMIT 5) WHERE x = ? LIMIT ?")
	stmt := analysis.Statements[0]
	if !stmt.HasLimit || stmt.LimitParam != 2 {
		t.Fatalf("limit %+v", stmt)
	}
	literal := analyzeOK(t, "SELECT * FROM (SELECT * FROM t LIMIT ?) WHERE x = 1 LIMIT 2")
	if len(literal.Sites) != 1 || literal.Sites[0].Start != 37 || literal.Sites[0].End != 38 {
		t.Fatalf("sites %+v", literal.Sites)
	}
	if stmt := literal.Statements[0]; !stmt.HasLimit || stmt.LimitParam != 0 {
		t.Fatalf("literal outer limit %+v", stmt)
	}
}

func TestSQLiteLimitForms(t *testing.T) {
	for _, c := range []struct {
		statement string
		limit     int
		sites     int
	}{
		{"SELECT * FROM t LIMIT ? + 1", 0, 1},
		{"SELECT * FROM t LIMIT ? OFFSET ?", 0, 2},
		{"SELECT * FROM t LIMIT 5, ?", 0, 1},
		{"SELECT * FROM t LIMIT (SELECT 1)", 0, 0},
		{"SELECT * FROM t LIMIT abs(?)", 0, 1},
		{"SELECT * FROM t LIMIT ?5", 5, 1},
		{"SELECT name FROM users WHERE id = :id LIMIT :lim", 2, 2},
		{"SELECT * FROM t WHERE x = ? LIMIT ? /* c */", 2, 2},
	} {
		analysis := analyzeOK(t, c.statement)
		stmt := analysis.Statements[0]
		if !stmt.HasLimit || stmt.LimitParam != c.limit || len(analysis.Sites) != c.sites {
			t.Fatalf("%q: limit %+v sites %d, want L%d/%d sites", c.statement, stmt, len(analysis.Sites), c.limit, c.sites)
		}
	}
	offset := analyzeOK(t, "SELECT * FROM t LIMIT ? OFFSET ?")
	wantSite(t, "SELECT * FROM t LIMIT ? OFFSET ?", offset.Sites, 0, ParamSite{Number: 1, Start: 22, End: 23, Ref: "?1"})
	wantSite(t, "SELECT * FROM t LIMIT ? OFFSET ?", offset.Sites, 1, ParamSite{Number: 2, Start: 31, End: 32, Ref: "?2"})
	comma := analyzeOK(t, "SELECT * FROM t LIMIT 5, ?")
	wantSite(t, "SELECT * FROM t LIMIT 5, ?", comma.Sites, 0, ParamSite{Number: 1, Start: 25, End: 26, Ref: "?1"})
}

func TestSQLiteNumericLimitName(t *testing.T) {
	statement := "SELECT a FROM t WHERE b = ? LIMIT $1"
	analysis := analyzeOK(t, statement)
	wantSite(t, statement, analysis.Sites, 0, ParamSite{Number: 1, Start: 26, End: 27, Ref: "?1"})
	wantSite(t, statement, analysis.Sites, 1, ParamSite{Number: 2, Start: 34, End: 36, Ref: "$1"})
	if stmt := analysis.Statements[0]; !stmt.HasLimit || stmt.LimitParam != 2 {
		t.Fatalf("limit %+v", stmt)
	}
	got, err := CheckDescriptorDialect(DialectSQLite, "t", statement, []string{"b"}, "one", 2)
	if err != nil {
		t.Fatalf("CheckDescriptor: %v", err)
	}
	if got.Limit != 2 {
		t.Fatalf("descriptor %+v", got)
	}
}

func TestSQLiteMisplacedCompoundClauses(t *testing.T) {
	failure := analyzeFailure(t, "SELECT id FROM t LIMIT 1 UNION SELECT id FROM u")
	if failure.Message != "LIMIT clause should come after UNION not before" || failure.Cursor != 17 {
		t.Fatalf("failure %+v", failure)
	}
	// An early LIMIT plus a final bound LIMIT: the misplaced-clause rule
	// fires rather than being masked by the missing-limit rule.
	masked := analyzeFailure(t, "SELECT id FROM t LIMIT 1 UNION SELECT id FROM u LIMIT ?")
	if !strings.Contains(masked.Message, "UNION not before") {
		t.Fatalf("failure %+v", masked)
	}
	ordered := analyzeFailure(t, "SELECT id FROM t ORDER BY id UNION SELECT id FROM u")
	if ordered.Message != "ORDER BY clause should come after UNION not before" {
		t.Fatalf("failure %+v", ordered)
	}
	valid := analyzeOK(t, "SELECT id FROM t UNION SELECT id FROM u LIMIT ?")
	wantSite(t, "SELECT id FROM t UNION SELECT id FROM u LIMIT ?", valid.Sites, 0, ParamSite{Number: 1, Start: 46, End: 47, Ref: "?1"})
	if stmt := valid.Statements[0]; !stmt.HasLimit || stmt.LimitParam != 1 {
		t.Fatalf("limit %+v", stmt)
	}
}

func TestSQLiteReturningShapes(t *testing.T) {
	decoy := analyzeOK(t, "SELECT * FROM t WHERE x = 'RETURNING'")
	if decoy.Statements[0].Returning {
		t.Fatalf("string decoy marks Returning: %+v", decoy.Statements[0])
	}
	insert := analyzeOK(t, "INSERT INTO t VALUES (1) RETURNING ?")
	if !insert.Statements[0].Returning {
		t.Fatalf("insert Returning %+v", insert.Statements[0])
	}
	wantSite(t, "INSERT INTO t VALUES (1) RETURNING ?", insert.Sites, 0, ParamSite{Number: 1, Start: 35, End: 36, Ref: "?1"})
	update := analyzeOK(t, "UPDATE t SET x = 1 WHERE y = ? RETURNING ?")
	if !update.Statements[0].Returning {
		t.Fatalf("update Returning %+v", update.Statements[0])
	}
	wantSite(t, "UPDATE t SET x = 1 WHERE y = ? RETURNING ?", update.Sites, 0, ParamSite{Number: 1, Start: 29, End: 30, Ref: "?1"})
	wantSite(t, "UPDATE t SET x = 1 WHERE y = ? RETURNING ?", update.Sites, 1, ParamSite{Number: 2, Start: 41, End: 42, Ref: "?2"})
	failure := analyzeFailure(t, "SELECT * FROM t RETURNING id")
	if failure.Message != `near "RETURNING": syntax error` || failure.Cursor != 16 {
		t.Fatalf("failure %+v", failure)
	}
}

func TestSQLiteMalformedInput(t *testing.T) {
	failure := analyzeFailure(t, "SELECT")
	if failure.Message != "incomplete input" || failure.Cursor != 6 {
		t.Fatalf("failure %+v", failure)
	}
	failure = analyzeFailure(t, "SELECT * FROM")
	if failure.Message != "incomplete input" || failure.Cursor != 13 {
		t.Fatalf("failure %+v", failure)
	}
	failure = analyzeFailure(t, "SELECT * FORM users WHERE id = ? LIMIT ?")
	if failure.Message != `near "FORM": syntax error` || failure.Cursor != 9 {
		t.Fatalf("failure %+v", failure)
	}
}

func TestSQLiteUTF8ErrorCursor(t *testing.T) {
	// FORM starts at byte 18; the two-byte ä before it makes the
	// character cursor 17.
	failure := analyzeFailure(t, "SELECT \"näme\", * FORM t")
	if failure.Message != `near "FORM": syntax error` || failure.Cursor != 17 {
		t.Fatalf("failure %+v", failure)
	}
}

func TestSQLiteNULRejected(t *testing.T) {
	// The NUL sits at byte 11; the two-byte ü before it makes the
	// character cursor 10. The trailing second statement is not parsed.
	failure := analyzeFailure(t, "SELECT 'ü'\x00SELECT 2")
	if failure.Message != "statement contains a NUL byte" || failure.Cursor != 10 {
		t.Fatalf("failure %+v", failure)
	}
}

func TestSQLiteSizeCeiling(t *testing.T) {
	analysis, err := analyzeSQLiteWithBudgets("t", "SELECT 123456789", 10, sqliteMaxTraversalNodes)
	if err != nil || analysis.Failure == nil || analysis.Failure.Message != "statement exceeds the size limit" {
		t.Fatalf("oversize: %+v %v", analysis, err)
	}
	exact, err := analyzeSQLiteWithBudgets("t", "SELECT 1", 8, sqliteMaxTraversalNodes)
	if err != nil || exact.Failure != nil || len(exact.Statements) != 1 {
		t.Fatalf("exact size: %+v %v", exact, err)
	}
}

func TestSQLiteNodeBudget(t *testing.T) {
	_, err := analyzeSQLiteWithBudgets("t", "SELECT * FROM t WHERE x = ? LIMIT ?", sqliteMaxStatementBytes, 5)
	if err == nil || !strings.Contains(err.Error(), "exceeds the node limit") {
		t.Fatalf("budget: err=%v", err)
	}
	analysis, err := analyzeSQLiteWithBudgets("t", "SELECT * FROM t WHERE x = ? LIMIT ?", sqliteMaxStatementBytes, sqliteMaxTraversalNodes)
	if err != nil || analysis.Failure != nil || len(analysis.Sites) != 2 {
		t.Fatalf("full budget: %+v %v", analysis, err)
	}
}

func TestSQLiteFlatExpressionTraversal(t *testing.T) {
	// A flat 2000-term chain nests BinaryExpr nodes without counting
	// against the upstream parser's recursion ceiling; the adapter's
	// iterative walk must collect the binds at both ends.
	statement := "SELECT ?," + strings.Repeat("1+", 2000) + "1,? FROM t"
	analysis := analyzeOK(t, statement)
	if len(analysis.Sites) != 2 {
		t.Fatalf("sites %d", len(analysis.Sites))
	}
	wantSite(t, statement, analysis.Sites, 0, ParamSite{Number: 1, Start: 7, End: 8, Ref: "?1"})
	tail := strings.LastIndex(statement, "?")
	wantSite(t, statement, analysis.Sites, 1, ParamSite{Number: 2, Start: tail, End: tail + 1, Ref: "?2"})
}

func TestSQLiteBindBearingVariants(t *testing.T) {
	upsert := analyzeOK(t, "INSERT INTO t VALUES (?) ON CONFLICT DO UPDATE SET x = ?")
	wantSite(t, "INSERT INTO t VALUES (?) ON CONFLICT DO UPDATE SET x = ?", upsert.Sites, 0, ParamSite{Number: 1, Start: 22, End: 23, Ref: "?1"})
	wantSite(t, "INSERT INTO t VALUES (?) ON CONFLICT DO UPDATE SET x = ?", upsert.Sites, 1, ParamSite{Number: 2, Start: 55, End: 56, Ref: "?2"})
	window := analyzeOK(t, "SELECT rank() OVER w FROM t WINDOW w AS (ORDER BY ?) LIMIT ?2")
	wantSite(t, "SELECT rank() OVER w FROM t WINDOW w AS (ORDER BY ?) LIMIT ?2", window.Sites, 0, ParamSite{Number: 1, Start: 50, End: 51, Ref: "?1"})
	wantSite(t, "SELECT rank() OVER w FROM t WINDOW w AS (ORDER BY ?) LIMIT ?2", window.Sites, 1, ParamSite{Number: 2, Start: 59, End: 61, Ref: "?2"})
	if stmt := window.Statements[0]; !stmt.HasLimit || stmt.LimitParam != 2 {
		t.Fatalf("limit %+v", stmt)
	}
}

func TestSQLiteStatementKinds(t *testing.T) {
	for input, want := range map[string]string{
		"SELECT 1":                    "select_statement",
		"INSERT INTO t VALUES (1)":    "insert_statement",
		"UPDATE t SET x = 1":          "update_statement",
		"DELETE FROM t":               "delete_statement",
		"PRAGMA journal_mode=WAL":     "pragma_statement",
		"EXPLAIN SELECT 1":            "explain_statement",
		"EXPLAIN QUERY PLAN SELECT 1": "explain_statement",
		"CREATE TABLE t(x)":           "create_table_statement",
		"CREATE INDEX i ON t(a)":      "create_index_statement",
		"CREATE VIEW v AS SELECT 1":   "create_view_statement",
		"ALTER TABLE t ADD COLUMN c":  "alter_table_statement",
		"DROP TABLE t":                "drop_table_statement",
		"ANALYZE":                     "analyze_statement",
		"VACUUM":                      "vacuum_statement",
		"REINDEX":                     "reindex_statement",
		"SAVEPOINT s":                 "savepoint_statement",
		"RELEASE s":                   "release_statement",
		"ROLLBACK":                    "rollback_statement",
		"ATTACH 'f' AS a":             "attach_statement",
		"DETACH a":                    "detach_statement",
		"DROP INDEX i":                "drop_index_statement",
		"DROP VIEW v":                 "drop_view_statement",
		"DROP TRIGGER tr":             "drop_trigger_statement",
		"CREATE TRIGGER tr AFTER INSERT ON t BEGIN SELECT 1; END": "create_trigger_statement",
		"CREATE VIRTUAL TABLE vt USING fts5(x)":                   "create_virtual_table_statement",
	} {
		analysis := analyzeOK(t, input)
		if len(analysis.Statements) != 1 || analysis.Statements[0].Kind != want {
			t.Fatalf("%q: %+v, want kind %s", input, analysis.Statements, want)
		}
	}
}

func TestSQLiteNestedMisplacedClausesRejected(t *testing.T) {
	for _, c := range []struct {
		statement string
		message   string
		cursor    int
	}{
		{"WITH c AS (SELECT 1 LIMIT 1 UNION SELECT 2) SELECT * FROM c LIMIT ?", "LIMIT clause should come after UNION not before", 20},
		{"SELECT (SELECT 1 LIMIT 1 UNION SELECT 2) LIMIT ?", "LIMIT clause should come after UNION not before", 17},
		{"INSERT INTO t SELECT 1 LIMIT 1 UNION SELECT 2", "LIMIT clause should come after UNION not before", 23},
		// ORDER BY cursors point at the first ordering term, which is
		// what the upstream span covers, inside the offending clause.
		{"WITH c AS (SELECT 1 ORDER BY 1 UNION SELECT 2) SELECT * FROM c LIMIT ?1", "ORDER BY clause should come after UNION not before", 29},
		// The two-byte ä before the CTE makes byte 24 character 23.
		{`WITH "cä" AS (SELECT 1 LIMIT 1 UNION SELECT 2) SELECT * FROM "cä" LIMIT ?`, "LIMIT clause should come after UNION not before", 23},
	} {
		failure := analyzeFailure(t, c.statement)
		if failure.Message != c.message || failure.Cursor != c.cursor {
			t.Fatalf("%q: failure %+v, want %q@%d", c.statement, failure, c.message, c.cursor)
		}
	}
}

func TestSQLiteNestedMisplacedClausesFailDescriptors(t *testing.T) {
	for _, c := range []struct {
		statement   string
		params      []string
		cardinality string
		rowLimit    uint64
		message     string
	}{
		{"WITH c AS (SELECT 1 LIMIT 1 UNION SELECT 2) SELECT * FROM c LIMIT ?", []string{"id"}, "one", 2, "LIMIT clause should come after UNION not before"},
		{"SELECT (SELECT 1 LIMIT 1 UNION SELECT 2) LIMIT ?", []string{"id"}, "one", 2, "LIMIT clause should come after UNION not before"},
		{"INSERT INTO t SELECT 1 LIMIT 1 UNION SELECT 2", nil, "execute", 0, "LIMIT clause should come after UNION not before"},
		{"WITH c AS (SELECT 1 ORDER BY 1 UNION SELECT 2) SELECT * FROM c LIMIT ?1", []string{"id"}, "one", 2, "ORDER BY clause should come after UNION not before"},
	} {
		_, err := CheckDescriptorDialect(DialectSQLite, "t", c.statement, c.params, c.cardinality, c.rowLimit)
		if err == nil || !strings.Contains(err.Error(), c.message) {
			t.Fatalf("%q: err=%v, want syntax rejection %q (not a cardinality/parameter error)", c.statement, err, c.message)
		}
	}
}

func TestSQLiteNestedValidCompoundControls(t *testing.T) {
	// A last-core ORDER BY and LIMIT inside a CTE is legal and parses.
	valid := analyzeOK(t, "WITH c AS (SELECT 1 UNION SELECT 2 ORDER BY 1 LIMIT 1) SELECT * FROM c LIMIT ?")
	if stmt := valid.Statements[0]; !stmt.HasLimit || stmt.LimitParam != 1 || len(valid.Sites) != 1 {
		t.Fatalf("control %+v %+v", valid.Statements[0], valid.Sites)
	}
	scalar := analyzeOK(t, "SELECT (SELECT max(x) FROM t LIMIT 1) FROM u LIMIT ?")
	if stmt := scalar.Statements[0]; !stmt.HasLimit || stmt.LimitParam != 1 {
		t.Fatalf("scalar control %+v", stmt)
	}
	got, err := CheckDescriptorDialect(DialectSQLite, "t", "INSERT INTO t SELECT 1 UNION SELECT 2", nil, "execute", 0)
	if err != nil {
		t.Fatalf("mutation-source control: %v", err)
	}
	if got.Kind != "insert_statement" {
		t.Fatalf("control %+v", got)
	}
	// An inner bound alone must not satisfy the outer SELECT's bound
	// requirement: this fails as unbounded, not as misplaced.
	_, err = CheckDescriptorDialect(DialectSQLite, "t", "WITH c AS (SELECT 1 UNION SELECT 2 LIMIT 1) SELECT * FROM c", nil, "one", 1)
	if err == nil || !strings.Contains(err.Error(), "unbounded SELECT") {
		t.Fatalf("inner-bound control: err=%v", err)
	}
}

func TestSQLiteUpdateDeleteLimitsAdmitted(t *testing.T) {
	// Bun's SQLite build accepts these mutation clauses, so the pinned
	// grammar option must admit them too.
	update := analyzeOK(t, "UPDATE t SET x = ? LIMIT 1")
	wantSite(t, "UPDATE t SET x = ? LIMIT 1", update.Sites, 0, ParamSite{Number: 1, Start: 17, End: 18, Ref: "?1"})
	delete := analyzeOK(t, "DELETE FROM t WHERE x = ? ORDER BY x LIMIT 1")
	wantSite(t, "DELETE FROM t WHERE x = ? ORDER BY x LIMIT 1", delete.Sites, 0, ParamSite{Number: 1, Start: 24, End: 25, Ref: "?1"})
}

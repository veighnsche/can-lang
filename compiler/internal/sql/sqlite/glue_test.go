package sqlite

import "testing"

func parse(t *testing.T, input string) (*Node, bool) {
	t.Helper()
	root, hasError, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q): %v", input, err)
	}
	return root, hasError
}

func TestParseSelectSpans(t *testing.T) {
	root, hasError := parse(t, "SELECT name FROM users WHERE id = :id LIMIT :lim")
	if hasError || root.Type != "source_file" {
		t.Fatalf("root=%s hasError=%v", root.Type, hasError)
	}
	stmts := root.Statements()
	if len(stmts) != 1 || stmts[0].Type != "select_statement" {
		t.Fatalf("statements=%v", stmts)
	}
	var binds []*Node
	stmts[0].Collect("bind_parameter", &binds)
	if len(binds) != 2 || binds[0].Text("SELECT name FROM users WHERE id = :id LIMIT :lim") != ":id" || binds[1].Text("SELECT name FROM users WHERE id = :id LIMIT :lim") != ":lim" {
		t.Fatalf("binds=%v", binds)
	}
	if binds[0].Start != 34 || binds[0].End != 37 || binds[1].Start != 44 || binds[1].End != 48 {
		t.Fatalf("spans=%v,%v", binds[0], binds[1])
	}
}

func TestParseParameterSpellings(t *testing.T) {
	for _, spelling := range []string{"?", "?7", ":name", "@name", "$name"} {
		input := "SELECT * FROM t WHERE x = " + spelling
		root, hasError := parse(t, input)
		var binds []*Node
		root.Collect("bind_parameter", &binds)
		t.Logf("%q: hasError=%v binds=%d", spelling, hasError, len(binds))
		if hasError || len(binds) != 1 || binds[0].Text(input) != spelling {
			t.Errorf("%q: hasError=%v binds=%v", spelling, hasError, binds)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, input := range []string{
		"SELECT * FORM users WHERE id = ?",
		"SELECT name FROM users WHERE nick = 'unterminated",
		"SELECT ",
		"SELECT * FROM t WHERE x = $1",
	} {
		_, hasError := parse(t, input)
		t.Logf("%q: hasError=%v", input, hasError)
		if !hasError {
			t.Errorf("%q: want hasError", input)
		}
	}
}

func TestParseKinds(t *testing.T) {
	for input, want := range map[string]string{
		"PRAGMA journal_mode=WAL":  "pragma_statement",
		"EXPLAIN SELECT 1":         "explain_statement",
		"CREATE TABLE t(x)":        "create_table_statement",
		"DELETE FROM t WHERE x=?":  "delete_statement",
		"INSERT INTO t VALUES (?)": "insert_statement",
	} {
		root, hasError := parse(t, input)
		stmts := root.Statements()
		if hasError || len(stmts) != 1 || stmts[0].Type != want {
			t.Errorf("%q: hasError=%v stmts=%v", input, hasError, stmts)
		}
	}
}

package sql

import (
	"testing"
	"unicode/utf8"
)

func params(tokens []Token) [][2]int {
	var out [][2]int
	for _, token := range tokens {
		if token.Kind == "PARAM" {
			out = append(out, [2]int{token.Start, token.End})
		}
	}
	return out
}

func TestVersionPinsPostgreSQL17(t *testing.T) {
	version, err := Version()
	if err != nil {
		t.Fatal(err)
	}
	if version != 170007 || version/10000 != Major {
		t.Fatalf("parser version %d", version)
	}
}

func TestParseSelectParams(t *testing.T) {
	input := "SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2"
	stmts, failure, err := Parse(input)
	if err != nil || failure != nil {
		t.Fatalf("parse: %v %v", err, failure)
	}
	if len(stmts) != 1 || stmts[0].Location != 0 || stmts[0].Kind != "SelectStmt" {
		t.Fatalf("stmts: %+v", stmts)
	}
	if stmts[0].Length != 0 {
		t.Fatalf("unterminated statement must report length 0, got %+v", stmts[0])
	}
	tokens, failure, err := Scan(input)
	if err != nil || failure != nil {
		t.Fatalf("scan: %v %v", err, failure)
	}
	got := params(tokens)
	if len(got) != 2 || got[0] != [2]int{63, 65} || got[1] != [2]int{84, 86} {
		t.Fatalf("params: %v", got)
	}
	if input[63:65] != "$1" || input[84:86] != "$2" {
		t.Fatal("param spans do not slice the input")
	}
}

func TestScanHidesParamsInStringsQuotesAndComments(t *testing.T) {
	cases := map[string][2]int{
		"SELECT 'it''s $1 not a param'":                         {-1, -1},
		"SELECT E'esc \\' $1', $2":                              {21, 23},
		"SELECT $$body $1$$, $tag$inner $2$tag$, $3":            {40, 42},
		"-- trailing $1 comment\nSELECT $2":                     {30, 32},
		"/* outer /* inner $1 */ still comment $2 */ SELECT $3": {51, 53},
	}
	for input, want := range cases {
		tokens, failure, err := Scan(input)
		if err != nil || failure != nil {
			t.Fatalf("%q: %v %v", input, err, failure)
		}
		got := params(tokens)
		if want[0] < 0 {
			if len(got) != 0 {
				t.Fatalf("%q: params %v", input, got)
			}
			continue
		}
		if len(got) != 1 || got[0] != want {
			t.Fatalf("%q: params %v", input, got)
		}
		if input[got[0][0]:got[0][1]][0] != '$' {
			t.Fatalf("%q: span misses the parameter", input)
		}
		if _, failure, err := Parse(input); err != nil || failure != nil {
			t.Fatalf("%q: parse: %v %v", input, err, failure)
		}
	}
}

func TestScanUsesByteOffsetsAcrossMultibyteText(t *testing.T) {
	input := "-- café ☃ snowman\nSELECT $1"
	tokens, failure, err := Scan(input)
	if err != nil || failure != nil {
		t.Fatalf("scan: %v %v", err, failure)
	}
	got := params(tokens)
	if len(got) != 1 || got[0] != [2]int{28, 30} || input[28:30] != "$1" {
		t.Fatalf("params: %v", got)
	}
	quoted := `SELECT "café" FROM t WHERE x = $1`
	tokens, failure, err = Scan(quoted)
	if err != nil || failure != nil {
		t.Fatalf("scan: %v %v", err, failure)
	}
	got = params(tokens)
	if len(got) != 1 || got[0] != [2]int{32, 34} || quoted[32:34] != "$1" {
		t.Fatalf("params: %v", got)
	}
}

func TestFailuresStayVerbatimWithCursor(t *testing.T) {
	stmts, failure, err := Parse("SELECT FROM WHERE")
	if err != nil || failure == nil || len(stmts) != 0 {
		t.Fatalf("parse: %v %v %+v", err, failure, stmts)
	}
	if failure.Message != `syntax error at or near "WHERE"` || failure.Cursor != 13 {
		t.Fatalf("failure: %+v", failure)
	}
	if tokens, failure, err := Scan("SELECT FROM WHERE"); err != nil || failure != nil || len(tokens) == 0 {
		t.Fatalf("grammatical input still scans: %v %v %d", err, failure, len(tokens))
	}
	stmts, failure, err = Parse("SELECT 'oops")
	if err != nil || failure == nil || len(stmts) != 0 {
		t.Fatalf("parse: %v %v %+v", err, failure, stmts)
	}
	if failure.Message != `unterminated quoted string at or near "'oops"` || failure.Cursor != 8 {
		t.Fatalf("failure: %+v", failure)
	}
	_, failure, err = Scan("SELECT 'oops")
	if err != nil || failure == nil || failure.Cursor != 8 {
		t.Fatalf("scan: %v %+v", err, failure)
	}
}

func TestMultiStatementSpans(t *testing.T) {
	stmts, failure, err := Parse("SELECT $1; SELECT 2")
	if err != nil || failure != nil {
		t.Fatalf("parse: %v %v", err, failure)
	}
	if len(stmts) != 2 || stmts[0] != (Statement{Location: 0, Length: 9, Kind: "SelectStmt"}) || stmts[1] != (Statement{Location: 10, Length: 0, Kind: "SelectStmt"}) {
		t.Fatalf("stmts: %+v", stmts)
	}
	stmts, failure, err = Parse("SELECT 1; SELECT $1;")
	if err != nil || failure != nil {
		t.Fatalf("parse: %v %v", err, failure)
	}
	if len(stmts) != 2 || stmts[0] != (Statement{Location: 0, Length: 8, Kind: "SelectStmt"}) || stmts[1] != (Statement{Location: 9, Length: 10, Kind: "SelectStmt"}) {
		t.Fatalf("stmts: %+v", stmts)
	}
	tokens, failure, err := Scan("SELECT 1; SELECT $1;")
	if err != nil || failure != nil {
		t.Fatalf("scan: %v %v", err, failure)
	}
	var semis [][2]int
	for _, token := range tokens {
		if token.Kind == "ASCII_59" {
			semis = append(semis, [2]int{token.Start, token.End})
		}
	}
	if len(semis) != 2 || semis[0] != [2]int{8, 9} || semis[1] != [2]int{19, 20} {
		t.Fatalf("semicolons: %v", semis)
	}
}

func TestEmptyInputsYieldNoStatements(t *testing.T) {
	for _, input := range []string{"", "-- nothing here\n", ";"} {
		stmts, failure, err := Parse(input)
		if err != nil || failure != nil || len(stmts) != 0 {
			t.Fatalf("%q: %v %v %+v", input, err, failure, stmts)
		}
	}
	if tokens, failure, err := Scan(""); err != nil || failure != nil || len(tokens) != 0 {
		t.Fatalf("empty: %v %v %d", err, failure, len(tokens))
	}
	tokens, failure, err := Scan(";")
	if err != nil || failure != nil {
		t.Fatalf("scan: %v %v", err, failure)
	}
	if len(tokens) != 1 || tokens[0] != (Token{Start: 0, End: 1, Kind: "ASCII_59", Keyword: "NO_KEYWORD"}) {
		t.Fatalf("tokens: %+v", tokens)
	}
	tokens, failure, err = Scan("-- nothing here\n")
	if err != nil || failure != nil {
		t.Fatalf("scan: %v %v", err, failure)
	}
	if len(tokens) != 1 || tokens[0] != (Token{Start: 0, End: 15, Kind: "SQL_COMMENT", Keyword: "NO_KEYWORD"}) {
		t.Fatalf("tokens: %+v", tokens)
	}
}

func TestStatementKindsAndLimitParams(t *testing.T) {
	stmts, failure, err := Parse("INSERT INTO t (a) VALUES ($1) RETURNING id")
	if err != nil || failure != nil || len(stmts) != 1 || stmts[0].Kind != "InsertStmt" {
		t.Fatalf("insert: %v %v %+v", err, failure, stmts)
	}
	stmts, failure, err = Parse("UPDATE t SET a = $1 WHERE id = $2")
	if err != nil || failure != nil || len(stmts) != 1 || stmts[0].Kind != "UpdateStmt" {
		t.Fatalf("update: %v %v %+v", err, failure, stmts)
	}
	stmts, failure, err = Parse("DELETE FROM t WHERE id = $1")
	if err != nil || failure != nil || len(stmts) != 1 || stmts[0].Kind != "DeleteStmt" {
		t.Fatalf("delete: %v %v %+v", err, failure, stmts)
	}
	tokens, failure, err := Scan("SELECT * FROM (SELECT id FROM t LIMIT $1) s LIMIT $2")
	if err != nil || failure != nil {
		t.Fatalf("scan: %v %v", err, failure)
	}
	got := params(tokens)
	if len(got) != 2 || got[0] != [2]int{38, 40} || got[1] != [2]int{50, 52} {
		t.Fatalf("params: %v", got)
	}
}

func TestStatementLimitAndReturningShapes(t *testing.T) {
	cases := []struct {
		input      string
		kind       string
		hasLimit   bool
		limitParam int
		returning  bool
	}{
		{"SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2", "SelectStmt", true, 2, false},
		{"SELECT id FROM t LIMIT 10", "SelectStmt", true, 0, false},
		{"SELECT id FROM t", "SelectStmt", false, 0, false},
		{"SELECT * FROM (SELECT id FROM t LIMIT $1) s", "SelectStmt", false, 0, false},
		{"SELECT * FROM (SELECT id FROM t LIMIT $1) s LIMIT $2", "SelectStmt", true, 2, false},
		{"SELECT id FROM t LIMIT ALL", "SelectStmt", true, 0, false},
		{"WITH active AS (SELECT id FROM t WHERE x = $1) SELECT id FROM active LIMIT $2", "SelectStmt", true, 2, false},
		{"SELECT id FROM a UNION SELECT id FROM b LIMIT $1", "SelectStmt", true, 1, false},
		{"INSERT INTO t (a) VALUES ($1)", "InsertStmt", false, 0, false},
		{"INSERT INTO t (a) VALUES ($1) RETURNING id", "InsertStmt", false, 0, true},
		{"UPDATE t SET a = $1 WHERE id = $2", "UpdateStmt", false, 0, false},
		{"UPDATE t SET a = $1 RETURNING id", "UpdateStmt", false, 0, true},
		{"DELETE FROM t WHERE id = $1", "DeleteStmt", false, 0, false},
		{"DELETE FROM t RETURNING id", "DeleteStmt", false, 0, true},
	}
	for _, c := range cases {
		stmts, failure, err := Parse(c.input)
		if err != nil || failure != nil || len(stmts) != 1 {
			t.Fatalf("%q: %v %v %d", c.input, err, failure, len(stmts))
		}
		got := stmts[0]
		if got.Kind != c.kind || got.HasLimit != c.hasLimit || got.LimitParam != c.limitParam || got.Returning != c.returning {
			t.Fatalf("%q: %+v", c.input, got)
		}
	}
}

func TestKeywordKindsRideAlong(t *testing.T) {
	tokens, failure, err := Scan("SELECT $1")
	if err != nil || failure != nil {
		t.Fatalf("scan: %v %v", err, failure)
	}
	if len(tokens) != 2 || tokens[0].Kind != "SELECT" || tokens[0].Keyword != "RESERVED_KEYWORD" || tokens[1].Kind != "PARAM" || tokens[1].Keyword != "NO_KEYWORD" {
		t.Fatalf("tokens: %+v", tokens)
	}
}

func TestInvalidUTF8StaysDeterministic(t *testing.T) {
	input := "SELECT '\xff\xfe'"
	stmts, failure, firstErr := Parse(input)
	again, failureAgain, secondErr := Parse(input)
	if (firstErr == nil) != (secondErr == nil) || len(stmts) != len(again) {
		t.Fatal("nondeterministic invalid-UTF8 parse")
	}
	if failure != nil && failureAgain != nil && *failure != *failureAgain {
		t.Fatal("nondeterministic invalid-UTF8 failure")
	}
	if !utf8.ValidString(input) && failure == nil && firstErr == nil && len(stmts) != 0 {
		t.Fatalf("invalid UTF-8 parsed silently: %+v", stmts)
	}
}

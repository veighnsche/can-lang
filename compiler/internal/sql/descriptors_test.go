package sql

import (
	"strconv"
	"strings"
	"testing"
)

func TestCheckDescriptorSelect(t *testing.T) {
	statement := "SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2"
	got, err := CheckDescriptor("search_accounts", statement, []string{"term"}, "many", 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "SelectStmt" || got.Total != 2 || got.Limit != 2 || got.Version != 170007 {
		t.Fatalf("%+v", got)
	}
	want := []Segment{{Text: "SELECT id, display_name FROM accounts WHERE display_name ILIKE "}, {Param: 1}, {Text: " ORDER BY id LIMIT "}, {Param: 2}}
	if len(got.Segments) != len(want) {
		t.Fatalf("%+v", got.Segments)
	}
	for i := range want {
		if got.Segments[i] != want[i] {
			t.Fatalf("%+v", got.Segments)
		}
	}
	var rebuilt strings.Builder
	sites := 0
	for _, segment := range got.Segments {
		if segment.Param > 0 {
			sites++
			rebuilt.WriteString("$" + strconv.Itoa(segment.Param))
		} else {
			rebuilt.WriteString(segment.Text)
		}
	}
	if rebuilt.String() != statement || sites != 2 {
		t.Fatal("segments do not tile the statement")
	}
}

func TestCheckDescriptorRepeatsAndHiddenParams(t *testing.T) {
	got, err := CheckDescriptor("r", "SELECT $1, $1 + $2 LIMIT $3", []string{"a", "b"}, "one", 3)
	if err != nil {
		t.Fatal(err)
	}
	params := 0
	for _, segment := range got.Segments {
		if segment.Param > 0 {
			params++
		}
	}
	if params != 4 {
		t.Fatalf("%+v", got.Segments)
	}
	got, err = CheckDescriptor("h", "SELECT 'it''s $1', $$q $2$$, $1 LIMIT $2", []string{"a"}, "optional", 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 2 || len(got.Segments) != 4 {
		t.Fatalf("%+v", got)
	}
}

func TestCheckDescriptorExecuteShapes(t *testing.T) {
	for _, c := range []struct {
		statement string
		params    []string
		kind      string
	}{
		{"INSERT INTO t (a, b) VALUES ($1, $2)", []string{"a", "b"}, "InsertStmt"},
		{"UPDATE t SET a = $1 WHERE id = $2", []string{"a", "id"}, "UpdateStmt"},
		{"DELETE FROM t WHERE id = $1", []string{"id"}, "DeleteStmt"},
		{"DELETE FROM t", nil, "DeleteStmt"},
	} {
		got, err := CheckDescriptor("op", c.statement, c.params, "execute", 0)
		if err != nil {
			t.Fatalf("%s: %v", c.statement, err)
		}
		if got.Kind != c.kind || got.Limit != 0 || got.Total != len(c.params) {
			t.Fatalf("%+v", got)
		}
	}
}

func TestCheckDescriptorRejects(t *testing.T) {
	cases := []struct {
		name       string
		statement  string
		params     []string
		cardinal   string
		limit      uint64
		wantSubstr string
	}{
		{"syntax", "SELECT FROM WHERE", []string{"a"}, "many", 2, "syntax error"},
		{"scan", "SELECT 'oops", []string{"a"}, "many", 2, "unterminated"},
		{"multi", "SELECT 1; SELECT 2", nil, "many", 1, "2 statements"},
		{"empty", "", nil, "many", 1, "0 statements"},
		{"unbounded", "SELECT id FROM t WHERE x = $1", []string{"a"}, "many", 2, "unbounded SELECT"},
		{"literal limit", "SELECT id FROM t LIMIT 10", nil, "one", 1, "top-level LIMIT must be $1"},
		{"wrong limit", "SELECT id FROM t WHERE x = $1 LIMIT $1", []string{"a"}, "many", 2, "must be $2"},
		{"shared limit", "SELECT id FROM t WHERE x = $2 LIMIT $2", []string{"a"}, "many", 2, "appears 2 times"},
		{"gap", "SELECT $1 LIMIT $3", []string{"a", "b"}, "many", 3, `parameter "b" ($2) is not used`},
		{"extra", "SELECT $1 LIMIT $3", []string{"a"}, "many", 2, "no declared parameter"},
		{"misplaced limit", "SELECT $2 LIMIT $1", []string{"a"}, "many", 2, "must be $2"},
		{"select for execute", "SELECT 1", nil, "execute", 0, "requires INSERT"},
		{"returning insert", "INSERT INTO t (a) VALUES ($1) RETURNING id", []string{"a"}, "execute", 0, "RETURNING"},
		{"returning update", "UPDATE t SET a = $1 RETURNING id", []string{"a"}, "execute", 0, "RETURNING"},
		{"returning delete", "DELETE FROM t RETURNING id", nil, "execute", 0, "RETURNING"},
		{"execute limit", "DELETE FROM t", nil, "execute", 1, "must not declare"},
		{"limit order", "SELECT $1 LIMIT $2", []string{"a"}, "many", 1, "want 2"},
		{"insert for many", "INSERT INTO t (a) VALUES ($1)", []string{"a"}, "many", 2, "requires SELECT"},
	}
	for _, c := range cases {
		_, err := CheckDescriptor(c.name, c.statement, c.params, c.cardinal, c.limit)
		if err == nil || !strings.Contains(err.Error(), c.wantSubstr) {
			t.Fatalf("%s: %v, want %q", c.name, err, c.wantSubstr)
		}
	}
}

func TestCheckDescriptorAllowsTrivia(t *testing.T) {
	got, err := CheckDescriptor("t", "-- report\nSELECT id FROM t WHERE x = $1 LIMIT $2; -- done\n", []string{"a"}, "many", 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.Segments[0].Text != "-- report\nSELECT id FROM t WHERE x = " {
		t.Fatalf("%+v", got.Segments)
	}
	last := got.Segments[len(got.Segments)-1]
	if last.Text != "; -- done\n" {
		t.Fatalf("%+v", got.Segments)
	}
}

func TestCheckDescriptorAdmitsRichSelects(t *testing.T) {
	for _, statement := range []string{
		"SELECT a.id, b.name FROM a JOIN b ON b.id = a.id WHERE a.x = $1 LIMIT $2",
		"SELECT * FROM (SELECT id FROM t LIMIT $1) s LIMIT $2",
		"WITH active AS (SELECT id FROM t WHERE x = $1) SELECT id FROM active LIMIT $2",
		"SELECT id FROM a UNION SELECT id FROM b LIMIT $1",
		"SELECT id FROM t WHERE x = $1 OFFSET 5 LIMIT $2",
	} {
		params := []string{"a"}
		limit := uint64(2)
		if strings.Count(statement, "$2") == 0 {
			params, limit = nil, 1
		}
		if _, err := CheckDescriptor("rich", statement, params, "many", limit); err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
}

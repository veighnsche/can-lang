package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
)

const sqlTestSource = `package app
    provides []
    uses [option, bytes]
record search_parameters
    str term
record account_row
    int id
    str display_name
record wide_row
    int id
    str display_name
    option::value<str> note
    bytes::buffer raw
record bad_row
    int[] scores
record pair
    str a
    str b
record no_params
fn void main
    emits []
    given
        str[] args
    asserts
        empty: [] => ok
    ok
`

func writeSQLProject(t *testing.T, manifest string) string {
	t.Helper()
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("can.project.json", manifest)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", sqlTestSource)
	return root
}

func TestSQLDescriptorChecks(t *testing.T) {
	manifest := `{"source_root":"src","error_registry":"can.errors.json","sql":{"search_accounts":{"dialect":"postgresql","statement":"SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"many","row_limit_parameter":2}}}`
	graph, err := project.Load(writeSQLProject(t, manifest))
	if err != nil {
		t.Fatal(err)
	}
	program, err := CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(program.SQL) != 1 {
		t.Fatalf("checked %d descriptors", len(program.SQL))
	}
	got := program.SQL[0]
	if got.Owner != "" || got.ParamType == "" || got.RowType == "" || got.ParamType == got.RowType {
		t.Fatalf("identities %+v", got)
	}
	if got.Checked.Kind != "SelectStmt" || got.Checked.Limit != 2 || got.Checked.Total != 2 || len(got.Parameters) != 1 || got.Parameters[0] != "term" {
		t.Fatalf("%+v", got)
	}
	if len(got.Checked.Segments) != 4 || got.Checked.Segments[1].Param != 1 || got.Checked.Segments[3].Param != 2 {
		t.Fatalf("%+v", got.Checked.Segments)
	}
}

func TestSQLDescriptorOptionAndBytesFields(t *testing.T) {
	manifest := `{"source_root":"src","error_registry":"can.errors.json","sql":{"wide":{"dialect":"postgresql","statement":"SELECT id FROM accounts LIMIT $1","parameters":[],"parameter_type":"app::no_params","row_type":"app::wide_row","cardinality":"one","row_limit_parameter":1}}}`
	graph, err := project.Load(writeSQLProject(t, manifest))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = CheckProgram(graph); err != nil {
		t.Fatalf("option/bytes row rejected: %v", err)
	}
}

func TestSQLDescriptorRejects(t *testing.T) {
	descriptor := func(statement, params, paramType, rowType, cardinal, limit string) string {
		limitField := `,"row_limit_parameter":` + limit
		if limit == "" {
			limitField = ""
		}
		return `{"source_root":"src","error_registry":"can.errors.json","sql":{"q":{"dialect":"postgresql","statement":` + statement + `,"parameters":` + params + `,"parameter_type":` + paramType + `,"row_type":` + rowType + `,"cardinality":` + cardinal + limitField + `}}}`
	}
	good := `"SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2"`
	cases := []struct {
		name     string
		manifest string
		want     string
	}{
		{"unknown param type", descriptor(good, `["term"]`, `"app::missing"`, `"app::account_row"`, `"many"`, `2`), "not declared"},
		{"unknown row type", descriptor(good, `["term"]`, `"app::search_parameters"`, `"app::missing"`, `"many"`, `2`), "not declared"},
		{"unqualified type", descriptor(good, `["term"]`, `"search_parameters"`, `"app::account_row"`, `"many"`, `2`), "fully qualified"},
		{"generic type", descriptor(good, `["term"]`, `"app::search_parameters<int>"`, `"app::account_row"`, `"many"`, `2`), "bare package-qualified"},
		{"non record", descriptor(good, `["term"]`, `"app::main"`, `"app::account_row"`, `"many"`, `2`), "not an ordinary record"},
		{"field count", descriptor(good, `["term","extra"]`, `"app::search_parameters"`, `"app::account_row"`, `"many"`, `3`), "has 1 fields, manifest lists 2"},
		{"field order", descriptor(`"SELECT $1, $2 LIMIT $3"`, `["b","a"]`, `"app::pair"`, `"app::account_row"`, `"many"`, `3`), `field 1 is "a", manifest lists "b"`},
		{"field name", descriptor(good, `["query"]`, `"app::search_parameters"`, `"app::account_row"`, `"many"`, `2`), `field 1 is "term", manifest lists "query"`},
		{"bad row field", descriptor(good, `["term"]`, `"app::search_parameters"`, `"app::bad_row"`, `"many"`, `2`), "non-SQL type"},
		{"unbounded", descriptor(`"SELECT id FROM accounts WHERE display_name ILIKE $1"`, `["term"]`, `"app::search_parameters"`, `"app::account_row"`, `"many"`, `2`), "unbounded SELECT"},
		{"returning", descriptor(`"DELETE FROM accounts RETURNING id"`, `[]`, `"app::no_params"`, `"app::account_row"`, `"execute"`, ``), "RETURNING"},
		{"multi", descriptor(`"SELECT 1; SELECT 2"`, `[]`, `"app::no_params"`, `"app::account_row"`, `"many"`, `1`), "2 statements"},
	}
	for _, c := range cases {
		graph, err := project.Load(writeSQLProject(t, c.manifest))
		if err != nil {
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("%s: load %v, want %q", c.name, err, c.want)
			}
			continue
		}
		if _, err = CheckProgram(graph); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}

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

const sqlQuerySource = `package app
    provides []
    uses [sql, http, option]
record search_parameters
    str term
record account_row
    int id
    str display_name
record id_parameters
    int id
fn account_row by_id
    emits [http::credentials_missing, sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_missing, sql::row_count, sql::schema_mismatch]
    given
        int id
    asserts
        sample: 4 => ok account_row(4, "Bob")
    match call sql::pool_open("CAN_TEST_POSTGRES", 5)
        when
            sample: "CAN_TEST_POSTGRES", 5 => ok
        ok sql::pool pool => match call sql::query_one<id_parameters, account_row>(pool, "account_by_id", id_parameters(id))
            when
                sample: pool, "account_by_id", id_parameters(4) => ok account_row(4, "Bob")
            ok account_row row => ok row
            sql::unsupported_value
            sql::connection_failed
            sql::query_failed
            sql::constraint_failed
            sql::row_missing
            sql::row_count
            sql::schema_mismatch
        http::credentials_missing
        sql::connection_failed
fn option::value<account_row> by_term
    emits [http::credentials_missing, sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_count, sql::schema_mismatch]
    given
        str term
    asserts
        sample: "Bob" => ok option::some(account_row(4, "Bob"))
    match call sql::pool_open("CAN_TEST_POSTGRES", 5)
        when
            sample: "CAN_TEST_POSTGRES", 5 => ok
        ok sql::pool pool => match call sql::query_optional<search_parameters, account_row>(pool, "account_by_term", search_parameters(term))
            when
                sample: pool, "account_by_term", search_parameters("Bob") => ok option::some(account_row(4, "Bob"))
            ok option::value<account_row> found => ok found
            sql::unsupported_value
            sql::connection_failed
            sql::query_failed
            sql::constraint_failed
            sql::row_count
            sql::schema_mismatch
        http::credentials_missing
        sql::connection_failed
fn account_row[] by_rows
    emits [http::credentials_missing, sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed, sql::row_limit, sql::schema_mismatch]
    given
        str term
    asserts
        sample: "son" => ok [account_row(2, "Jason")]
    match call sql::pool_open("CAN_TEST_POSTGRES", 5)
        when
            sample: "CAN_TEST_POSTGRES", 5 => ok
        ok sql::pool pool => match call sql::query_rows<search_parameters, account_row>(pool, "accounts_by_term", search_parameters(term), 10)
            when
                sample: pool, "accounts_by_term", search_parameters("son"), 10 => ok [account_row(2, "Jason")]
            ok account_row[] rows => ok rows
            sql::unsupported_value
            sql::connection_failed
            sql::query_failed
            sql::constraint_failed
            sql::row_limit
            sql::schema_mismatch
        http::credentials_missing
        sql::connection_failed
fn int add_one
    emits [http::credentials_missing, sql::unsupported_value, sql::connection_failed, sql::query_failed, sql::constraint_failed]
    given
        str term
    asserts
        sample: "Zed" => ok 1
    match call sql::pool_open("CAN_TEST_POSTGRES", 5)
        when
            sample: "CAN_TEST_POSTGRES", 5 => ok
        ok sql::pool pool => match call sql::execute<search_parameters>(pool, "add_account", search_parameters(term))
            when
                sample: pool, "add_account", search_parameters("Zed") => ok 1
            ok int affected => ok affected
            sql::unsupported_value
            sql::connection_failed
            sql::query_failed
            sql::constraint_failed
        http::credentials_missing
        sql::connection_failed
fn void main
    emits []
    given
        str[] args
    asserts
        empty: [] => ok
    ok
`

func writeSQLQueryProject(t *testing.T, manifest, source string) string {
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
	write("src/main.can", source)
	return root
}

const sqlQueryManifest = `{"source_root":"src","error_registry":"can.errors.json","sql":{"account_by_id":{"dialect":"postgresql","statement":"SELECT id, display_name FROM accounts WHERE id = $1 LIMIT $2","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::account_row","cardinality":"one","row_limit_parameter":2},"account_by_term":{"dialect":"postgresql","statement":"SELECT id, display_name FROM accounts WHERE display_name = $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"optional","row_limit_parameter":2},"accounts_by_term":{"dialect":"postgresql","statement":"SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"many","row_limit_parameter":2},"add_account":{"dialect":"postgresql","statement":"INSERT INTO accounts (display_name) VALUES ($1)","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"execute"}}}`

func TestSQLQuerySpecializes(t *testing.T) {
	graph, err := project.Load(writeSQLQueryProject(t, sqlQueryManifest, sqlQuerySource))
	if err != nil {
		t.Fatal(err)
	}
	program, err := CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(program.SQLs) != 4 {
		t.Fatalf("specialized %d queries", len(program.SQLs))
	}
	seen := map[string]bool{}
	for key, special := range program.SQLs {
		seen[special.Operation] = true
		if special.Contract == nil || special.P == nil || len(special.Params.Fields) == 0 {
			t.Fatalf("incomplete specialization %s", key)
		}
		if special.Operation == sqlExecute {
			if special.R != nil || len(special.Rows.Fields) != 0 {
				t.Fatalf("execute carries rows %s", key)
			}
			continue
		}
		if special.R == nil || len(special.Rows.Fields) != 2 {
			t.Fatalf("missing row schema %s", key)
		}
		if special.Operation == sqlQueryOptional && (special.ResultSome == "" || special.ResultNone == "" || special.ResultSome == special.ResultNone) {
			t.Fatalf("missing result option identities %s", key)
		}
	}
	for _, operation := range []string{sqlQueryOne, sqlQueryOptional, sqlQueryRows, sqlExecute} {
		if !seen[operation] {
			t.Fatalf("missing specialization %s", operation)
		}
	}
}

func TestSQLQueryRejects(t *testing.T) {
	swap := func(edits ...string) string {
		out := sqlQuerySource
		for i := 0; i < len(edits); i += 2 {
			out = strings.Replace(out, edits[i], edits[i+1], 1)
		}
		return out
	}
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{"variable descriptor", swap(
			"    given\n        int id\n    asserts\n        sample: 4 => ok account_row(4, \"Bob\")",
			"    given\n        int id\n        str name\n    asserts\n        sample: 4, \"x\" => ok account_row(4, \"Bob\")",
			`(pool, "account_by_id", id_parameters(id))`, `(pool, name, id_parameters(id))`,
		), "static string literal"},
		{"unknown descriptor", strings.Replace(sqlQuerySource, `"account_by_id"`, `"missing"`, 2), "undeclared descriptor"},
		{"parameter mismatch", swap(
			`sql::query_one<id_parameters, account_row>(pool, "account_by_id", id_parameters(id))`,
			`sql::query_one<search_parameters, account_row>(pool, "account_by_id", search_parameters("x"))`,
			`sample: pool, "account_by_id", id_parameters(4) => ok account_row(4, "Bob")`,
			`sample: pool, "account_by_id", search_parameters("x") => ok account_row(4, "Bob")`,
		), "binds parameters"},
		{"row mismatch", swap(
			`fn account_row by_id`, `fn search_parameters by_id`,
			`sample: 4 => ok account_row(4, "Bob")`, `sample: 4 => ok search_parameters("Bob")`,
			`sql::query_one<id_parameters, account_row>`, `sql::query_one<id_parameters, search_parameters>`,
			`sample: pool, "account_by_id", id_parameters(4) => ok account_row(4, "Bob")`,
			`sample: pool, "account_by_id", id_parameters(4) => ok search_parameters("Bob")`,
			`ok account_row row => ok row`, `ok search_parameters row => ok row`,
		), "binds rows"},
		{"cardinality mismatch", swap(
			`fn account_row by_id`, `fn account_row[] by_id`,
			`sql::row_missing, sql::row_count, sql::schema_mismatch`, `sql::row_limit, sql::schema_mismatch`,
			"            sql::row_missing\n            sql::row_count\n", "            sql::row_limit\n",
			`sample: 4 => ok account_row(4, "Bob")`, `sample: 4 => ok [account_row(4, "Bob")]`,
			`sql::query_one<id_parameters, account_row>(pool, "account_by_id", id_parameters(id))`,
			`sql::query_rows<id_parameters, account_row>(pool, "account_by_id", id_parameters(id), 10)`,
			`sample: pool, "account_by_id", id_parameters(4) => ok account_row(4, "Bob")`,
			`sample: pool, "account_by_id", id_parameters(4), 10 => ok [account_row(4, "Bob")]`,
			`ok account_row row => ok row`, `ok account_row[] rows => ok rows`,
		), "needs cardinality many"},
		{"missing type args", strings.Replace(sqlQuerySource, `sql::query_one<id_parameters, account_row>`, `sql::query_one`, 1), "explicit type arguments"},
		{"non record parameters", strings.Replace(sqlQuerySource, `sql::execute<search_parameters>`, `sql::execute<str>`, 1), "concrete ordinary record"},
		{"transaction waits", strings.Replace(sqlQuerySource, `sql::query_one<id_parameters, account_row>`, `sql::transaction_query_one<id_parameters, account_row>`, 1), "admitted by I38"},
	}
	for _, c := range cases {
		graph, err := project.Load(writeSQLQueryProject(t, sqlQueryManifest, c.source))
		if err != nil {
			t.Fatalf("%s: load %v", c.name, err)
		}
		if _, err = CheckProgram(graph); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: %v, want %q", c.name, err, c.want)
		}
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

package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const sqlTransactionManifest = `{"source_root":"src","error_registry":"can.errors.json","sql":{` +
	`"account_by_id":{"dialect":"postgresql","statement":"SELECT id, display_name FROM accounts WHERE id = $1 LIMIT $2","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::account_row","cardinality":"one","row_limit_parameter":2},` +
	`"add_account":{"dialect":"postgresql","statement":"INSERT INTO accounts (display_name) VALUES ($1)","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"execute"}}}`

const sqlTransactionSource = `package app
    provides []
    uses [sql, http, option]
record search_parameters
    str term
record account_row
    int id
    str display_name
record id_parameters
    int id
fn sql::decision<int> decide
    emits []
    given
        sql::transaction tx
    asserts
        sample: => ok sql::commit<int>(1)
    match call sql::transaction_execute<search_parameters>(tx, "add_account", search_parameters("Zed"))
        when
            sample: tx, "add_account", search_parameters("Zed") => ok 1
        ok int affected => ok sql::commit<int>(affected)
        sql::unsupported_value => ok sql::rollback<int>(0)
        sql::query_failed => ok sql::rollback<int>(0)
        sql::constraint_failed => ok sql::rollback<int>(0)
fn int run
    emits [http::credentials_missing, sql::connection_failed, sql::transaction_failed, sql::commit_unknown]
    asserts
        sample: => ok 1
    match call sql::pool_open("CAN_TEST_POSTGRES", 5)
        when
            sample: "CAN_TEST_POSTGRES", 5 => ok
        ok sql::pool pool => match call sql::with_transaction<int>(pool, callable decide)
            when
                sample: pool, callable decide => ok 1
            ok int total => ok total
            sql::connection_failed
            sql::transaction_failed
            sql::commit_unknown
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

const sqlTransactionCardinalitySource = `package app
    provides []
    uses [sql, http, option]
record search_parameters
    str term
record id_parameters
    int id
record account_row
    int id
    str display_name
fn sql::decision<int> decide
    emits []
    given
        sql::transaction tx
    asserts
        sample: => ok sql::commit<int>(1)
    match call sql::transaction_query_one<search_parameters, account_row>(tx, "add_account", search_parameters("Zed"))
        when
            sample: tx, "add_account", search_parameters("Zed") => ok account_row(7, "Zed")
        ok account_row row => ok sql::commit<int>(row.id)
        sql::unsupported_value => ok sql::rollback<int>(0)
        sql::query_failed => ok sql::rollback<int>(0)
        sql::constraint_failed => ok sql::rollback<int>(0)
        sql::row_missing => ok sql::rollback<int>(0)
        sql::row_count => ok sql::rollback<int>(0)
        sql::schema_mismatch => ok sql::rollback<int>(0)
fn int run
    emits [http::credentials_missing, sql::connection_failed, sql::transaction_failed, sql::commit_unknown]
    asserts
        sample: => ok 1
    match call sql::pool_open("CAN_TEST_POSTGRES", 5)
        when
            sample: "CAN_TEST_POSTGRES", 5 => ok
        ok sql::pool pool => match call sql::with_transaction<int>(pool, callable decide)
            when
                sample: pool, callable decide => ok 1
            ok int total => ok total
            sql::connection_failed
            sql::transaction_failed
            sql::commit_unknown
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

func writeSQLTransactionProject(t *testing.T, source string) string {
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
	write("can.project.json", sqlTransactionManifest)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", source)
	return root
}

func TestSQLTransactionAccepts(t *testing.T) {
	graph, err := project.Load(writeSQLTransactionProject(t, sqlTransactionSource))
	if err != nil {
		t.Fatal(err)
	}
	program, err := CheckProgram(graph)
	if err != nil {
		t.Fatalf("transaction program rejected: %v", err)
	}
	if len(program.Transactions) != 1 {
		t.Fatalf("checked %d transaction specializations", len(program.Transactions))
	}
	var special *TransactionSpecialization
	for _, candidate := range program.Transactions {
		special = candidate
	}
	if special.Operation != sqlWithTransaction || special.Commit == "" || special.Rollback == "" || special.Commit == special.Rollback {
		t.Fatalf("bad transaction specialization %+v", special)
	}
	if special.Contract.Result().Kind() != types.Primitive || special.T.Identity() == "" {
		t.Fatalf("bad transaction result %+v", special.T)
	}
	inputs := special.Contract.Inputs()
	if len(inputs) != 2 || inputs[0].Declaration() != "can.std.sql@1::pool" || inputs[1].Kind() != types.Callable {
		t.Fatalf("bad transaction inputs %+v", inputs)
	}
	callback := inputs[1]
	if len(callback.Inputs()) != 1 || callback.Inputs()[0].Declaration() != sqlTransactionHandle {
		t.Fatalf("bad callback inputs %+v", callback.Inputs())
	}
	if len(callback.Errors()) != 0 {
		t.Fatalf("callback admits %d errors", len(callback.Errors()))
	}
	// The in-transaction execute shares the pool query specialization table.
	found := false
	for _, sql := range program.SQLs {
		if sql.Operation == sqlTransactionExecute {
			found = true
		}
	}
	if !found {
		t.Fatal("transaction execute left no query specialization")
	}
}

func TestSQLTransactionRejects(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{"callback emits", strings.Replace(sqlTransactionSource, "fn sql::decision<int> decide\n    emits []", "fn sql::decision<int> decide\n    emits [sql::query_failed]", 1), "does not fit expected type"},
		{"callback result", strings.Replace(strings.Replace(sqlTransactionSource, "fn sql::decision<int> decide", "fn int decide", 1), "sample: => ok sql::commit<int>(1)", "sample: => ok 1", 1), "does not fit expected type"},
		{"callback scope input", strings.Replace(sqlTransactionSource, "        sql::transaction tx", "        sql::pool tx", 1), "call arity mismatch"},
		{"void result", strings.Replace(sqlTransactionSource, "sql::with_transaction<int>", "sql::with_transaction<void>", 1), "void is not a data type"},
		{"missing type args", strings.Replace(sqlTransactionSource, "sql::with_transaction<int>(pool, callable decide)", "sql::with_transaction(pool, callable decide)", 1), "explicit type arguments"},
		{"wrong callback target", strings.Replace(sqlTransactionSource, "callable decide", "callable run", 1), "does not fit expected type"},
		{"wrong cardinality", sqlTransactionCardinalitySource, "needs cardinality one"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			graph, err := project.Load(writeSQLTransactionProject(t, tc.source))
			if err != nil {
				t.Fatal(err)
			}
			_, err = CheckProgram(graph)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}

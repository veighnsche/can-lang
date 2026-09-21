package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

func TestSQLDescriptorsEmitManifestTable(t *testing.T) {
	root := t.TempDir()
	source, err := os.ReadFile(filepath.Join("..", "..", "testdata", "current", "sql", "descriptors.can"))
	if err != nil {
		t.Fatal(err)
	}
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
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json","sql":{"search_accounts":{"dialect":"postgresql","statement":"SELECT id, display_name FROM accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"many","row_limit_parameter":2},"wipe":{"dialect":"postgresql","statement":"DELETE FROM accounts","parameters":[],"parameter_type":"app::no_params","row_type":"app::account_row","cardinality":"execute"}}}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", string(source))
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(program.SQL) != 2 {
		t.Fatalf("checked %d descriptors", len(program.SQL))
	}
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var state string
	for _, artifact := range artifacts {
		if artifact.Path == "program/state.ts" {
			state = string(artifact.Bytes)
		}
	}
	for _, needle := range []string{
		"$canSQL=$canCreateSQLDescriptors(",
		`"search_accounts":{"cardinality":"many"`,
		`"text":"SELECT id, display_name FROM accounts WHERE display_name ILIKE "`,
		`{"param":1}`,
		`{"param":2}`,
		`"params":["term"]`,
		`"limit":2,"total":2,"version":170007`,
		`"wipe":{"cardinality":"execute"`,
		`"params":[]`,
		`"limit":0,"total":0`,
	} {
		if !strings.Contains(state, needle) {
			t.Fatalf("missing %s", needle)
		}
	}
	for _, needle := range []string{"can.std.sql@1::query_one", "can.std.sql@1::query_optional", "can.std.sql@1::query_rows", "can.std.sql@1::execute", ".unsafe(", ".simple("} {
		if strings.Contains(state, needle) {
			t.Fatalf("descriptor table carries query execution: %s", needle)
		}
	}
}

func TestSQLQueriesEmitPlansAndSplices(t *testing.T) {
	root := t.TempDir()
	source, err := os.ReadFile(filepath.Join("..", "..", "testdata", "current", "sql", "queries.can"))
	if err != nil {
		t.Fatal(err)
	}
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
	write("can.project.json", `{"source_root":"src","error_registry":"can.errors.json","sql":{"account_by_id":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i35_accounts WHERE id = $1 LIMIT $2","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::account_row","cardinality":"one","row_limit_parameter":2},"account_by_term":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i35_accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"optional","row_limit_parameter":2},"accounts_by_term":{"dialect":"postgresql","statement":"SELECT id, display_name FROM can_i35_accounts WHERE display_name ILIKE $1 ORDER BY id LIMIT $2","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"many","row_limit_parameter":2},"add_account":{"dialect":"postgresql","statement":"INSERT INTO can_i35_accounts (display_name) VALUES ($1)","parameters":["term"],"parameter_type":"app::search_parameters","row_type":"app::account_row","cardinality":"execute"},"cover_by_id":{"dialect":"postgresql","statement":"SELECT id, payload, note FROM can_i35_cover WHERE id = $1 LIMIT $2","parameters":["id"],"parameter_type":"app::id_parameters","row_type":"app::cover_row","cardinality":"one","row_limit_parameter":2}}}`)
	write("can.errors.json", `{"active":[],"retired":[]}`)
	write("src/main.can", string(source))
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(program.SQLs) != 5 {
		t.Fatalf("specialized %d queries", len(program.SQLs))
	}
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var state, packages string
	for _, artifact := range artifacts {
		switch {
		case artifact.Path == "program/state.ts":
			state = string(artifact.Bytes)
		case strings.HasPrefix(artifact.Path, "packages/"):
			packages += string(artifact.Bytes)
		}
	}
	for _, needle := range []string{
		"$canSQLPools=$canCreateSQLPools($canDomain,{credentialsMissing:",
		"$canIsSQLPool($canSQLKinds[identity],value)",
		".queryOne(descriptor,",
		".queryOptional(descriptor,",
		".queryRows(descriptor,",
		".execute(descriptor,",
		`run:(pool:unknown,_name:unknown,params:unknown,maxRows:bigint,descriptor:`,
		`"params":{"root":`,
		`"rows":{"root":`,
		`"fields":[{"name":"term","kind":"str"}]`,
		`"fields":[{"name":"id","kind":"int"},{"name":"display_name","kind":"str"}]`,
		"/platform/sql.ts",
	} {
		if !strings.Contains(state, needle) {
			t.Fatalf("missing %s", needle)
		}
	}
	for _, needle := range []string{
		`$canSQL.declareDescriptor("","account_by_id")`,
		`$canSQL.declareDescriptor("","accounts_by_term")`,
		`"account_by_id"`,
		"$canSQLPools.open",
		"$canSQLPools.close",
	} {
		if !strings.Contains(packages, needle) {
			t.Fatalf("missing package %s", needle)
		}
	}
}

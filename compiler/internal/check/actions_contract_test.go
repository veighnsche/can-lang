package check

import (
	"fmt"
	"strings"
	"testing"
)

// actionContractSource is the selected shared invoice contract: one locked
// package with transparent wire records, finite result variants and three
// handler-free action declarations. The declarations below are the exact
// selected source forms.
const actionContractSource = "package invoice_contract\n" +
	"    provides [invoice_key, grid_line_wire, grid_snapshot, grid_loaded, grid_load_forbidden, grid_load_unavailable, grid_load_outcome, grid_problem, grid_edit_input, grid_saved, grid_invalid, grid_conflict, grid_forbidden, grid_unavailable, grid_edit_outcome, line_wire, invoice_form, field_error, saved, invalid, conflict, forbidden, unavailable, edit_outcome, load_invoice_grid, save_invoice_grid, save_invoice_html]\n" +
	"    uses [form, option]\n" +
	"record invoice_key\n" +
	"    int tenant_id\n" +
	"    int invoice_id\n" +
	"record grid_line_wire\n" +
	"    str key\n" +
	"    str id\n" +
	"    str quantity\n" +
	"    str price\n" +
	"record grid_snapshot\n" +
	"    str revision\n" +
	"    grid_line_wire[] lines\n" +
	"    int total_minor_units\n" +
	"record grid_loaded\n" +
	"    grid_snapshot current\n" +
	"record grid_load_forbidden\n" +
	"    str message\n" +
	"record grid_load_unavailable\n" +
	"    str message\n" +
	"variant grid_load_outcome\n" +
	"    grid_loaded\n" +
	"    grid_load_forbidden\n" +
	"    grid_load_unavailable\n" +
	"record grid_problem\n" +
	"    option::value<str> row_key\n" +
	"    str field\n" +
	"    str message\n" +
	"record grid_edit_input\n" +
	"    str operation_id\n" +
	"    str revision\n" +
	"    grid_line_wire[] lines\n" +
	"record grid_saved\n" +
	"    str operation_id\n" +
	"    grid_snapshot acknowledged\n" +
	"record grid_invalid\n" +
	"    str operation_id\n" +
	"    grid_edit_input draft\n" +
	"    grid_problem[] errors\n" +
	"record grid_conflict\n" +
	"    str operation_id\n" +
	"    grid_edit_input draft\n" +
	"    str message\n" +
	"record grid_forbidden\n" +
	"    str operation_id\n" +
	"    str message\n" +
	"record grid_unavailable\n" +
	"    str operation_id\n" +
	"    str message\n" +
	"variant grid_edit_outcome\n" +
	"    grid_saved\n" +
	"    grid_invalid\n" +
	"    grid_conflict\n" +
	"    grid_forbidden\n" +
	"    grid_unavailable\n" +
	"record line_wire\n" +
	"    str id\n" +
	"    str quantity\n" +
	"    str price\n" +
	"record invoice_form\n" +
	"    str seats\n" +
	"    option::value<str> details\n" +
	"    str revision\n" +
	"    form::rows<line_wire> lines\n" +
	"record field_error\n" +
	"    str field\n" +
	"    str message\n" +
	"record saved\n" +
	"    invoice_form acknowledged\n" +
	"record invalid\n" +
	"    option::value<invoice_form> raw\n" +
	"    field_error[] errors\n" +
	"record conflict\n" +
	"    invoice_form raw\n" +
	"    str message\n" +
	"record forbidden\n" +
	"    str message\n" +
	"record unavailable\n" +
	"    invoice_form raw\n" +
	"    str message\n" +
	"variant edit_outcome\n" +
	"    saved\n" +
	"    invalid\n" +
	"    conflict\n" +
	"    forbidden\n" +
	"    unavailable\n" +
	"action load_invoice_grid\n" +
	"    get \"/api/tenants/:tenant_id/invoices/:invoice_id\"\n" +
	"    captures invoice_key\n" +
	"    input none\n" +
	"    returns grid_load_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        grid_loaded status 200\n" +
	"        grid_load_forbidden status 403\n" +
	"        grid_load_unavailable status 503\n" +
	"action save_invoice_grid\n" +
	"    post \"/api/tenants/:tenant_id/invoices/:invoice_id\"\n" +
	"    captures invoice_key\n" +
	"    json grid_edit_input limit 8192\n" +
	"    returns grid_edit_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        grid_saved status 200\n" +
	"        grid_invalid status 422\n" +
	"        grid_conflict status 409\n" +
	"        grid_forbidden status 403\n" +
	"        grid_unavailable status 503\n" +
	"action save_invoice_html\n" +
	"    post \"/tenants/:tenant_id/invoices/:invoice_id\"\n" +
	"    captures invoice_key\n" +
	"    form invoice_form limit 2048 rows_limit 64\n" +
	"    returns edit_outcome\n" +
	"    body html\n" +
	"    cases\n" +
	"        saved status 200 swap inner\n" +
	"        invalid status 422 swap inner\n" +
	"        conflict status 409 swap inner\n" +
	"        forbidden status 403 swap inner\n" +
	"        unavailable status 503 swap inner\n"

func actionContractProgram(t *testing.T, contract string) *Program {
	t.Helper()
	program, err := programFixture(t, map[string]string{
		"src/contract/contract.can": contract,
		"src/app/main.can":          "package app\n    provides []\n    uses []\n" + actionMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestActionContractSelectedMetadata(t *testing.T) {
	program := actionContractProgram(t, actionContractSource)
	if len(program.Actions) != 3 {
		t.Fatalf("checked %d actions, want 3", len(program.Actions))
	}
	contractID := program.Actions[0].Symbol.Package.ID
	if contractID == "" {
		t.Fatal("contract package has no locked identity")
	}
	for _, action := range program.Actions {
		if action.Symbol.Package.ID != contractID {
			t.Fatalf("action %s escaped the contract package", action.Symbol.ID)
		}
		if action.Symbol.ID != contractID+"::"+action.Symbol.Name {
			t.Fatalf("action identity = %s", action.Symbol.ID)
		}
	}

	load := actionByName(t, program, "load_invoice_grid")
	if load.Method != "GET" || load.Path != "/api/tenants/:tenant_id/invoices/:invoice_id" {
		t.Fatalf("load route = %s %s", load.Method, load.Path)
	}
	if load.CapturesType.Declaration() != contractID+"::invoice_key" {
		t.Fatalf("load captures = %v", load.CapturesType)
	}
	if len(load.Captures) != 2 || load.Captures[0].Name != "tenant_id" || load.Captures[1].Name != "invoice_id" {
		t.Fatalf("load captures = %+v", load.Captures)
	}
	for _, capture := range load.Captures {
		if capture.Type.Declaration() != "int" {
			t.Fatalf("capture %s type = %s", capture.Name, capture.Type.Declaration())
		}
	}
	if load.Input.Mode != "none" || load.Body != "json" || load.Returns.Declaration() != contractID+"::grid_load_outcome" {
		t.Fatalf("load contract = %+v %s %s", load.Input, load.Body, load.Returns.Declaration())
	}
	if len(load.Cases) != 3 || load.Cases[0].Status != 200 || load.Cases[1].Status != 403 || load.Cases[2].Status != 503 {
		t.Fatalf("load cases = %+v", load.Cases)
	}
	for _, kase := range load.Cases {
		if kase.Swap != "" {
			t.Fatalf("JSON case gained swap: %+v", kase)
		}
	}
	if load.ResponseSchema == nil || load.ResponseSchema.Root != load.Returns.Identity() {
		t.Fatal("load lost its JSON response schema")
	}

	save := actionByName(t, program, "save_invoice_grid")
	if save.Method != "POST" || save.Input.Mode != "json" || save.Input.Limit != 8192 {
		t.Fatalf("JSON save input = %+v", save.Input)
	}
	if save.Input.Type.Declaration() != contractID+"::grid_edit_input" || save.Body != "json" {
		t.Fatalf("JSON save wire = %s %s", save.Input.Type.Declaration(), save.Body)
	}
	if save.Returns.Declaration() != contractID+"::grid_edit_outcome" || len(save.Cases) != 5 {
		t.Fatalf("JSON save returns %s with %d cases", save.Returns.Declaration(), len(save.Cases))
	}
	if save.ResponseSchema == nil || save.ResponseSchema.Root != save.Returns.Identity() {
		t.Fatal("JSON save lost its response schema")
	}

	html := actionByName(t, program, "save_invoice_html")
	if html.Method != "POST" || html.Path != "/tenants/:tenant_id/invoices/:invoice_id" {
		t.Fatalf("HTML save route = %s %s", html.Method, html.Path)
	}
	if html.Input.Mode != "form" || html.Input.Limit != 2048 || html.Input.RowsLimit != 64 {
		t.Fatalf("HTML save input = %+v", html.Input)
	}
	if html.Input.Type.Declaration() != contractID+"::invoice_form" || html.Body != "html" {
		t.Fatalf("HTML save wire = %s %s", html.Input.Type.Declaration(), html.Body)
	}
	if html.Returns.Declaration() != contractID+"::edit_outcome" || len(html.Cases) != 5 {
		t.Fatalf("HTML save returns %s with %d cases", html.Returns.Declaration(), len(html.Cases))
	}
	for _, kase := range html.Cases {
		if kase.Swap != "inner" {
			t.Fatalf("HTML case lost its swap policy: %+v", kase)
		}
	}
	if html.ResponseSchema != nil {
		t.Fatal("HTML save gained a JSON response schema")
	}
}

func TestActionContractSharedExports(t *testing.T) {
	// A consumer imports the shared contract package without importing
	// any executable handler: the contract carries none.
	program, err := programFixture(t, map[string]string{
		"src/contract/contract.can": actionContractSource,
		"src/app/main.can": "package app\n    provides []\n    uses [invoice_contract]\n" +
			"fn invoice_contract::invoice_key describe\n" +
			"    emits []\n" +
			"    asserts\n" +
			"        sample: => ok invoice_contract::invoice_key(1, 7)\n" +
			"    ok invoice_contract::invoice_key(1, 7)\n" + actionMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Actions) != 3 {
		t.Fatalf("checked %d actions, want 3", len(program.Actions))
	}
	for _, action := range program.Actions {
		if action.Symbol.Package.Name != "invoice_contract" {
			t.Fatalf("action %s escaped the contract package", action.Symbol.ID)
		}
	}
}

func TestActionContractRejects(t *testing.T) {
	base := actionContractSource
	for name, tc := range map[string]struct {
		text string
		want string
	}{
		"renamed capture field": {
			strings.Replace(base, "record invoice_key\n    int tenant_id\n    int invoice_id\n", "record invoice_key\n    int tenant\n    int invoice_id\n", 1),
			"path capture :tenant_id has no captures field",
		},
		"extra capture field": {
			strings.Replace(base, "record invoice_key\n    int tenant_id\n    int invoice_id\n", "record invoice_key\n    int tenant_id\n    int invoice_id\n    int shard\n", 1),
			"capture shard does not appear in the action path",
		},
		"optional capture field": {
			strings.Replace(base, "record invoice_key\n    int tenant_id\n    int invoice_id\n", "record invoice_key\n    option::value<int> tenant_id\n    int invoice_id\n", 1),
			"capture tenant_id must be str or int",
		},
		"zero limit": {
			strings.Replace(base, "    json grid_edit_input limit 8192\n", "    json grid_edit_input limit 0\n", 1),
			"action limit must be a positive byte count",
		},
		"missing rows limit": {
			strings.Replace(base, "    form invoice_form limit 2048 rows_limit 64\n", "    form invoice_form limit 2048\n", 1),
			"form input with keyed rows requires rows_limit",
		},
		"form answered json": {
			strings.Replace(base, "    returns edit_outcome\n    body html\n", "    returns edit_outcome\n    body json\n", 1),
			"form input requires body html",
		},
		"missing swap": {
			strings.Replace(base, "        saved status 200 swap inner\n", "        saved status 200\n", 1),
			"html action cases require swap inner",
		},
		"swap on json": {
			strings.Replace(base, "        grid_saved status 200\n", "        grid_saved status 200 swap inner\n", 1),
			"swap applies to html actions only",
		},
		"missing case": {
			strings.Replace(base, "        grid_unavailable status 503\n", "", 1),
			"action cases omit returns leaves",
		},
		"legacy braces": {
			strings.Replace(base, "\"/tenants/:tenant_id/invoices/:invoice_id\"", "\"/tenants/{tenant_id}/invoices/{invoice_id}\"", 1),
			"captures occupy one whole :name segment",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.text == base {
				t.Fatal("invalid refusal fixture")
			}
			_, err := programFixture(t, map[string]string{
				"src/contract/contract.can": tc.text,
				"src/app/main.can":          "package app\n    provides []\n    uses []\n" + actionMain,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("contract diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}

func TestActionContractOldSyntaxSpans(t *testing.T) {
	stale := strings.Replace(actionContractSource, "action save_invoice_html\n", "action save_stale\n", 1)
	stale = strings.Replace(stale, "action save_stale\n"+
		"    post \"/tenants/:tenant_id/invoices/:invoice_id\"\n"+
		"    captures invoice_key\n"+
		"    form invoice_form limit 2048 rows_limit 64\n"+
		"    returns edit_outcome\n"+
		"    body html\n"+
		"    cases\n"+
		"        saved status 200 swap inner\n"+
		"        invalid status 422 swap inner\n"+
		"        conflict status 409 swap inner\n"+
		"        forbidden status 403 swap inner\n"+
		"        unavailable status 503 swap inner\n", "action save_stale\n"+
		"    post \"/tenants/save\"\n"+
		"    body form invoice_form\n"+
		"    handles save_html\n"+
		"    result edit_outcome\n"+
		"    cases\n"+
		"        saved => 200\n"+
		"        invalid => 422\n"+
		"        conflict => 409\n"+
		"        forbidden => 403\n"+
		"        unavailable => 503\n", 1)
	if stale == actionContractSource {
		t.Fatal("invalid stale-syntax fixture")
	}
	_, err := programFixture(t, map[string]string{
		"src/contract/contract.can": stale,
		"src/app/main.can":          "package app\n    provides []\n    uses []\n" + actionMain,
	})
	if err == nil || !strings.Contains(err.Error(), "was removed") {
		t.Fatalf("old action syntax admitted or misdiagnosed: %v", err)
	}
	// Parse failures carry a file:line:column position; it must point at
	// the stale body clause.
	line := 1 + strings.Count(stale[:strings.Index(stale, "    body form invoice_form\n")], "\n")
	want := fmt.Sprintf("contract.can:%d:5", line)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("stale-syntax position omits %q: %v", want, err)
	}
}

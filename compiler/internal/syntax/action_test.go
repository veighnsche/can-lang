package syntax

import (
	"strings"
	"testing"
)

// The three selected shared-contract declarations from the upgrade
// contract. They are handler-free: captures name a record, one input
// line carries the wire mode and limits, returns names the finite
// variant, body names the response mode, and cases map leaves to
// statuses with the HTML-only swap policy.
const actionLoadGridSource = "action load_invoice_grid\n" +
	"    get \"/api/tenants/:tenant_id/invoices/:invoice_id\"\n" +
	"    captures invoice_key\n" +
	"    input none\n" +
	"    returns grid_load_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        grid_loaded status 200\n" +
	"        grid_load_forbidden status 403\n" +
	"        grid_load_unavailable status 503\n"

const actionSaveGridSource = "action save_invoice_grid\n" +
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
	"        grid_unavailable status 503\n"

const actionSaveHTMLSource = "action save_invoice_html\n" +
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

const actionStaticSource = "action list_invoices\n" +
	"    get \"/invoices\"\n" +
	"    input none\n" +
	"    returns load_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        found status 200\n" +
	"        denied status 403\n"

func TestActionSelectedDeclarationsParse(t *testing.T) {
	for _, text := range []string{actionLoadGridSource, actionSaveGridSource, actionSaveHTMLSource, actionStaticSource} {
		result := nativeParse(t, text)
		if !result.OK() {
			t.Fatalf("selected action rejected: %v", result.Diagnostics)
		}
	}
	load := nativeParse(t, actionLoadGridSource).File.Declarations[0].(*ActionDecl)
	if load.Name.Text != "load_invoice_grid" || load.Method.Text != "get" {
		t.Fatalf("load lost its name or method: %+v", load)
	}
	if load.Path.Value != "/api/tenants/:tenant_id/invoices/:invoice_id" {
		t.Fatalf("load path = %q", load.Path.Value)
	}
	if load.Captures == nil || FormatType(load.Captures) != "invoice_key" {
		t.Fatalf("load lost its captures record: %+v", load.Captures)
	}
	if load.Input.Mode.Text != "input" || load.Input.Type != nil || load.Input.Limit != nil {
		t.Fatalf("load lost its bodyless input: %+v", load.Input)
	}
	if FormatType(load.Returns) != "grid_load_outcome" || load.Response.Text != "json" {
		t.Fatalf("load lost its returns or body: %+v", load)
	}
	if len(load.Cases) != 3 || load.Cases[0].Leaf.Name != "grid_loaded" || load.Cases[0].Status.Text != "200" || load.Cases[0].Swap != nil {
		t.Fatalf("load lost its case table: %+v", load.Cases)
	}

	save := nativeParse(t, actionSaveGridSource).File.Declarations[0].(*ActionDecl)
	if save.Input.Mode.Text != "json" || FormatType(save.Input.Type) != "grid_edit_input" || save.Input.Limit.Text != "8192" || save.Input.RowsLimit != nil {
		t.Fatalf("JSON save lost its input line: %+v", save.Input)
	}
	if len(save.Cases) != 5 {
		t.Fatalf("JSON save case table has %d rows", len(save.Cases))
	}

	html := nativeParse(t, actionSaveHTMLSource).File.Declarations[0].(*ActionDecl)
	if html.Input.Mode.Text != "form" || FormatType(html.Input.Type) != "invoice_form" || html.Input.Limit.Text != "2048" || html.Input.RowsLimit == nil || html.Input.RowsLimit.Text != "64" {
		t.Fatalf("HTML save lost its input line: %+v", html.Input)
	}
	if html.Response.Text != "html" {
		t.Fatalf("HTML save lost its body mode: %+v", html.Response)
	}
	for _, kase := range html.Cases {
		if kase.Swap == nil || kase.Swap.Text != "swap" {
			t.Fatalf("HTML case lost its swap policy: %+v", kase)
		}
	}

	static := nativeParse(t, actionStaticSource).File.Declarations[0].(*ActionDecl)
	if static.Captures != nil {
		t.Fatalf("static action gained captures: %+v", static.Captures)
	}
}

func TestActionFormatRoundTrip(t *testing.T) {
	for _, text := range []string{actionLoadGridSource, actionSaveGridSource, actionSaveHTMLSource, actionStaticSource} {
		result := nativeParse(t, text)
		if !result.OK() {
			t.Fatal(result.Diagnostics)
		}
		formatted := Format(result.File)
		if !contains(formatted, text) {
			t.Fatalf("formatted output rewrote the action:\n%s", formatted)
		}
		reparsed := nativeParse(t, formatted[len(nativePackage):])
		if !reparsed.OK() {
			t.Fatalf("formatted action does not reparse: %v", reparsed.Diagnostics)
		}
		if Format(reparsed.File) != formatted {
			t.Fatal("action formatting is not stable")
		}
	}
}

func TestActionParsingRejects(t *testing.T) {
	post := func(body string) string {
		return "action save_invoice\n" +
			"    post \"/invoices/save\"\n" + body
	}
	for name, tc := range map[string]struct {
		text string
		want string
	}{
		"unknown method": {
			"action save_invoice\n    put \"/invoices/save\"\n    json invoice_wire limit 100\n    returns save_outcome\n    body json\n    cases\n        saved status 200\n",
			"unknown action method",
		},
		"bare action": {
			"action save_invoice\n",
			"action requires a method and path",
		},
		"missing route": {
			"action save_invoice\n    input none\n",
			"unknown action method",
		},
		"get wire input": {
			"action load_invoice\n    get \"/invoices\"\n    json invoice_wire limit 100\n    returns load_outcome\n    body json\n    cases\n        found status 200\n",
			"GET actions cannot take a wire input",
		},
		"post input none": {
			post("    input none\n    returns save_outcome\n    body json\n    cases\n        saved status 200\n"),
			"POST actions require a json or form input",
		},
		"input not none": {
			post("    input invoice_wire\n    returns save_outcome\n    body json\n    cases\n        saved status 200\n"),
			"spelled input none",
		},
		"missing input": {
			post("    returns save_outcome\n    body json\n    cases\n        saved status 200\n"),
			"action requires an input line",
		},
		"missing limit": {
			post("    json invoice_wire\n    returns save_outcome\n    body json\n    cases\n        saved status 200\n"),
			"requires a byte limit",
		},
		"rows limit on json": {
			post("    json invoice_wire limit 100 rows_limit 4\n    returns save_outcome\n    body json\n    cases\n        saved status 200\n"),
			"rows_limit applies to form input only",
		},
		"old body wire clause": {
			post("    body json invoice_wire\n    returns save_outcome\n    body json\n    cases\n        saved status 200\n"),
			"action body wire input was removed",
		},
		"handles removed": {
			post("    json invoice_wire limit 100\n    handles save_validated\n    returns save_outcome\n    body json\n    cases\n        saved status 200\n"),
			"action handles was removed",
		},
		"result removed": {
			post("    json invoice_wire limit 100\n    result save_outcome\n    body json\n    cases\n        saved status 200\n"),
			"action result was removed",
		},
		"missing returns": {
			post("    json invoice_wire limit 100\n    body json\n    cases\n        saved status 200\n"),
			"action requires a returns clause",
		},
		"missing body": {
			post("    json invoice_wire limit 100\n    returns save_outcome\n    cases\n        saved status 200\n"),
			"action requires a body clause",
		},
		"unknown body mode": {
			post("    json invoice_wire limit 100\n    returns save_outcome\n    body bytes\n    cases\n        saved status 200\n"),
			"unknown action body mode",
		},
		"duplicate input": {
			post("    json invoice_wire limit 100\n    json invoice_wire limit 100\n    returns save_outcome\n    body json\n    cases\n        saved status 200\n"),
			"duplicate action clause",
		},
		"duplicate captures": {
			"action load_invoice\n    get \"/invoices/:invoice_id\"\n    captures invoice_key\n    captures invoice_key\n    input none\n    returns load_outcome\n    body json\n    cases\n        found status 200\n",
			"action requires an input line",
		},
		"old captures block": {
			"action load_invoice\n    get \"/invoices/:invoice_id\"\n    captures\n        str invoice_id\n    input none\n    returns load_outcome\n    body json\n    cases\n        found status 200\n",
			"expected name",
		},
		"missing cases": {
			post("    json invoice_wire limit 100\n    returns save_outcome\n    body json\n"),
			"expected cases",
		},
		"empty cases": {
			post("    json invoice_wire limit 100\n    returns save_outcome\n    body json\n    cases\n"),
			"expected indent",
		},
		"old arrow case": {
			post("    json invoice_wire limit 100\n    returns save_outcome\n    body json\n    cases\n        saved => 200\n"),
			"expected status",
		},
		"case without status": {
			post("    json invoice_wire limit 100\n    returns save_outcome\n    body json\n    cases\n        saved\n"),
			"expected status",
		},
		"unknown swap policy": {
			post("    json invoice_wire limit 100\n    returns save_outcome\n    body json\n    cases\n        saved status 200 swap outer\n"),
			"unknown action swap policy",
		},
		"nonliteral path": {
			"action save_invoice\n    post target\n    json invoice_wire limit 100\n    returns save_outcome\n    body json\n    cases\n        saved status 200\n",
			"expected string",
		},
	} {
		t.Run(name, func(t *testing.T) {
			result := nativeParse(t, tc.text)
			if result.OK() {
				t.Fatalf("invalid action parsed: %s", tc.text)
			}
			joined := ""
			for _, diagnostic := range result.Diagnostics {
				joined += diagnostic.Message + "\n"
			}
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("diagnostics %q omit %q", joined, tc.want)
			}
		})
	}
}

func TestActionHandlesRejectionCarriesSpan(t *testing.T) {
	text := "action save_invoice\n" +
		"    post \"/invoices/save\"\n" +
		"    json invoice_wire limit 100\n" +
		"    handles save_validated\n" +
		"    returns save_outcome\n" +
		"    body json\n" +
		"    cases\n" +
		"        saved status 200\n"
	result := nativeParse(t, text)
	if result.OK() {
		t.Fatal("handles clause parsed")
	}
	full := nativePackage + text
	covered := ""
	for _, diagnostic := range result.Diagnostics {
		if strings.Contains(diagnostic.Message, "action handles was removed") {
			covered = full[diagnostic.Span.Start:diagnostic.Span.End]
		}
	}
	if covered != "handles" {
		t.Fatalf("handles rejection covers %q", covered)
	}
}

func TestActionQualifiedNames(t *testing.T) {
	text := "action save_invoice\n" +
		"    post \"/invoices/save\"\n" +
		"    captures web::invoice_key\n" +
		"    json web::invoice_wire limit 100\n" +
		"    returns web::save_outcome\n" +
		"    body json\n" +
		"    cases\n" +
		"        web::saved status 200\n"
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	decl := result.File.Declarations[0].(*ActionDecl)
	if FormatType(decl.Captures) != "web::invoice_key" {
		t.Fatalf("captures lost its qualifier: %+v", decl.Captures)
	}
	if FormatType(decl.Input.Type) != "web::invoice_wire" {
		t.Fatalf("input lost its qualifier: %+v", decl.Input)
	}
	if FormatType(decl.Returns) != "web::save_outcome" {
		t.Fatalf("returns lost its qualifier: %+v", decl.Returns)
	}
	if decl.Cases[0].Leaf.Package != "web" || decl.Cases[0].Leaf.Name != "saved" {
		t.Fatalf("case leaf lost its qualifier: %+v", decl.Cases[0].Leaf)
	}
	formatted := Format(result.File)
	if !contains(formatted, "captures web::invoice_key") || !contains(formatted, "web::saved status 200") {
		t.Fatalf("formatted output dropped qualifiers:\n%s", formatted)
	}
}

package syntax

import (
	"strings"
	"testing"
)

const actionPostSource = "action save_invoice\n" +
	"    post \"/invoices/save\"\n" +
	"    body json invoice_wire\n" +
	"    handles save_validated\n" +
	"    result save_outcome\n" +
	"    cases\n" +
	"        saved => 200\n" +
	"        rejected => 422\n" +
	"        stale => 409\n" +
	"        denied => 403\n" +
	"        busy => 503\n"

const actionGetSource = "action load_invoice\n" +
	"    get \"/invoices/{invoice_id}\"\n" +
	"    captures\n" +
	"        str invoice_id\n" +
	"    handles load_validated\n" +
	"    result load_outcome\n" +
	"    cases\n" +
	"        found => 200\n" +
	"        denied => 403\n" +
	"        busy => 503\n"

func TestActionPostParsing(t *testing.T) {
	result := nativeParse(t, actionPostSource)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	decl, ok := result.File.Declarations[0].(*ActionDecl)
	if !ok {
		t.Fatalf("action parsed as %T", result.File.Declarations[0])
	}
	if decl.Name.Text != "save_invoice" || decl.Method.Text != "post" {
		t.Fatalf("action lost its name or method: %+v", decl)
	}
	if decl.Path.Value != "/invoices/save" {
		t.Fatalf("action path = %q", decl.Path.Value)
	}
	if decl.Body == nil || decl.Body.Mode.Text != "json" || FormatType(decl.Body.Type) != "invoice_wire" {
		t.Fatalf("action lost its body contract: %+v", decl.Body)
	}
	if len(decl.Captures) != 0 {
		t.Fatalf("exact-path action gained captures: %+v", decl.Captures)
	}
	if decl.Handler.Name != "save_validated" || FormatType(decl.Result) != "save_outcome" {
		t.Fatalf("action lost its handler or result: %+v", decl)
	}
	if len(decl.Cases) != 5 || decl.Cases[0].Leaf.Name != "saved" || decl.Cases[0].Status.Text != "200" {
		t.Fatalf("action lost its case table: %+v", decl.Cases)
	}
}

func TestActionGetCapturesParsing(t *testing.T) {
	result := nativeParse(t, actionGetSource)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	decl := result.File.Declarations[0].(*ActionDecl)
	if decl.Method.Text != "get" || decl.Path.Value != "/invoices/{invoice_id}" {
		t.Fatalf("action lost its route: %+v", decl)
	}
	if decl.Body != nil {
		t.Fatalf("GET action gained a body: %+v", decl.Body)
	}
	if len(decl.Captures) != 1 || decl.Captures[0].Name.Text != "invoice_id" || FormatType(decl.Captures[0].Type) != "str" {
		t.Fatalf("action lost its captures: %+v", decl.Captures)
	}
	if len(decl.Cases) != 3 {
		t.Fatalf("action case table has %d rows", len(decl.Cases))
	}
}

func TestActionFormatRoundTrip(t *testing.T) {
	for _, text := range []string{actionPostSource, actionGetSource} {
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
	qualifier := func(clause string) string {
		return "action save_invoice\n" +
			"    post \"/invoices/save\"\n" +
			"    body json invoice_wire\n" + clause
	}
	for name, tc := range map[string]struct {
		text string
		want string
	}{
		"unknown method": {
			"action save_invoice\n    put \"/invoices/save\"\n    body json invoice_wire\n    handles save_validated\n    result save_outcome\n    cases\n        saved => 200\n",
			"unknown action method",
		},
		"bare action": {
			"action save_invoice\n",
			"action requires a method and path",
		},
		"missing route": {
			"action save_invoice\n    handles save_validated\n",
			"unknown action method",
		},
		"get body": {
			"action load_invoice\n    get \"/invoices\"\n    body json invoice_wire\n    handles load_validated\n    result load_outcome\n    cases\n        found => 200\n",
			"GET actions cannot have a body",
		},
		"post without body": {
			"action save_invoice\n    post \"/invoices/save\"\n    handles save_validated\n    result save_outcome\n    cases\n        saved => 200\n",
			"POST actions require a body",
		},
		"unknown body mode": {
			post("    body bytes invoice_wire\n    handles save_validated\n    result save_outcome\n    cases\n        saved => 200\n"),
			"unknown action body mode",
		},
		"duplicate body": {
			post("    body json invoice_wire\n    body json invoice_wire\n    handles save_validated\n    result save_outcome\n    cases\n        saved => 200\n"),
			"duplicate action clause",
		},
		"missing handles": {
			post("    body json invoice_wire\n    result save_outcome\n    cases\n        saved => 200\n"),
			"action requires a handles clause",
		},
		"missing result": {
			qualifier("    handles save_validated\n    cases\n        saved => 200\n"),
			"action requires a result clause",
		},
		"missing cases": {
			qualifier("    handles save_validated\n    result save_outcome\n"),
			"expected cases",
		},
		"empty cases": {
			qualifier("    handles save_validated\n    result save_outcome\n    cases\n"),
			"expected indent",
		},
		"case without status": {
			qualifier("    handles save_validated\n    result save_outcome\n    cases\n        saved\n"),
			"expected =>",
		},
		"nonliteral path": {
			"action save_invoice\n    post target\n    body json invoice_wire\n    handles save_validated\n    result save_outcome\n    cases\n        saved => 200\n",
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

func TestActionQualifiedHandlerAndLeaves(t *testing.T) {
	text := "action save_invoice\n" +
		"    post \"/invoices/save\"\n" +
		"    body json invoice_wire\n" +
		"    handles web::save_validated\n" +
		"    result web::save_outcome\n" +
		"    cases\n" +
		"        web::saved => 200\n"
	result := nativeParse(t, text)
	if !result.OK() {
		t.Fatal(result.Diagnostics)
	}
	decl := result.File.Declarations[0].(*ActionDecl)
	if decl.Handler.Package != "web" || decl.Handler.Name != "save_validated" {
		t.Fatalf("handler lost its qualifier: %+v", decl.Handler)
	}
	if decl.Cases[0].Leaf.Package != "web" || decl.Cases[0].Leaf.Name != "saved" {
		t.Fatalf("case leaf lost its qualifier: %+v", decl.Cases[0].Leaf)
	}
	formatted := Format(result.File)
	if !contains(formatted, "handles web::save_validated") || !contains(formatted, "web::saved => 200") {
		t.Fatalf("formatted output dropped qualifiers:\n%s", formatted)
	}
}

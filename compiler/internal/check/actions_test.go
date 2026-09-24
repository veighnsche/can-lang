package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
)

const actionMain = "fn void main\n" +
	"    emits []\n" +
	"    given\n" +
	"        str[] arguments\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    ok\n"

const actionSaveDomain = "record invoice_wire\n" +
	"    str label\n" +
	"    int seats\n" +
	"record saved\n" +
	"    str label\n" +
	"record rejected\n" +
	"    str reason\n" +
	"record stale\n" +
	"    int revision\n" +
	"record denied\n" +
	"    str reason\n" +
	"record busy\n" +
	"    str reason\n" +
	"variant save_outcome\n" +
	"    saved\n" +
	"    rejected\n" +
	"    stale\n" +
	"    denied\n" +
	"    busy\n"

const actionSaveHandler = "fn save_outcome save_validated\n" +
	"    emits []\n" +
	"    given\n" +
	"        invoice_wire body\n" +
	"    asserts\n" +
	"        sample: invoice_wire(\"inv-1\", 2) => ok saved(\"inv-1\")\n" +
	"    ok saved(body.label)\n"

const actionSaveAction = "action save_invoice\n" +
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

const actionLoadDomain = "record found\n" +
	"    str label\n" +
	"record missing\n" +
	"    str reason\n" +
	"record unavailable\n" +
	"    str reason\n" +
	"variant load_outcome\n" +
	"    found\n" +
	"    missing\n" +
	"    unavailable\n"

const actionLoadHandler = "fn load_outcome load_validated\n" +
	"    emits []\n" +
	"    given\n" +
	"        str invoice_id\n" +
	"        int line\n" +
	"    asserts\n" +
	"        sample: \"inv-1\", 1 => ok found(\"inv-1\")\n" +
	"    ok found(invoice_id)\n"

const actionLoadAction = "action load_line\n" +
	"    get \"/invoices/{invoice_id}/lines/{line}\"\n" +
	"    captures\n" +
	"        str invoice_id\n" +
	"        int line\n" +
	"    handles load_validated\n" +
	"    result load_outcome\n" +
	"    cases\n" +
	"        found => 200\n" +
	"        missing => 403\n" +
	"        unavailable => 503\n"

func actionWebFile(extra ...string) string {
	decls := actionSaveDomain + actionSaveHandler + actionSaveAction + actionLoadDomain + actionLoadHandler + actionLoadAction
	for _, text := range extra {
		decls += text
	}
	return "package web\n    provides [save_invoice, load_line, invoice_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome]\n    uses []\n" + decls + actionMain
}

func actionByName(t *testing.T, program *Program, name string) *ActionDeclaration {
	t.Helper()
	for _, action := range program.Actions {
		if action.Symbol.Name == name {
			return action
		}
	}
	t.Fatalf("action %s missing from checked table", name)
	return nil
}

func TestActionPostSaveChecks(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/web/web.can": actionWebFile()})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Actions) != 2 {
		t.Fatalf("checked %d actions, want 2", len(program.Actions))
	}
	action := actionByName(t, program, "save_invoice")
	if action.Method != "POST" || action.Path != "/invoices/save" {
		t.Fatalf("save route = %s %s", action.Method, action.Path)
	}
	if len(action.Captures) != 0 {
		t.Fatalf("exact-path action gained captures: %+v", action.Captures)
	}
	if action.Body == nil || action.Body.Mode != "json" {
		t.Fatalf("save action lost its json body: %+v", action.Body)
	}
	fields := map[string]bool{}
	for _, field := range action.Body.Type.Fields() {
		fields[field.Name] = true
	}
	if !fields["label"] || !fields["seats"] {
		t.Fatalf("body wire fields = %v", fields)
	}
	wantHandler := action.Symbol.Package.ID + "::save_validated"
	if action.Handler != wantHandler {
		t.Fatalf("handler = %s, want %s", action.Handler, wantHandler)
	}
	wantResult := action.Symbol.Package.ID + "::save_outcome"
	if action.Result.Declaration() != wantResult {
		t.Fatalf("result = %s, want %s", action.Result.Declaration(), wantResult)
	}
	wantCases := []ActionCase{
		{Leaf: action.Symbol.Package.ID + "::saved", Status: 200},
		{Leaf: action.Symbol.Package.ID + "::rejected", Status: 422},
		{Leaf: action.Symbol.Package.ID + "::stale", Status: 409},
		{Leaf: action.Symbol.Package.ID + "::denied", Status: 403},
		{Leaf: action.Symbol.Package.ID + "::busy", Status: 503},
	}
	if len(action.Cases) != len(wantCases) {
		t.Fatalf("cases = %+v", action.Cases)
	}
	for i, kase := range wantCases {
		if action.Cases[i] != kase {
			t.Fatalf("cases = %+v, want %+v", action.Cases, wantCases)
		}
	}
}

func TestActionGetLoadChecks(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/web/web.can": actionWebFile()})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "load_line")
	if action.Method != "GET" || action.Path != "/invoices/{invoice_id}/lines/{line}" {
		t.Fatalf("load route = %s %s", action.Method, action.Path)
	}
	if action.Body != nil {
		t.Fatalf("GET action gained a body: %+v", action.Body)
	}
	if len(action.Captures) != 2 || action.Captures[0].Name != "invoice_id" || action.Captures[1].Name != "line" {
		t.Fatalf("captures = %+v", action.Captures)
	}
	if action.Captures[0].Type.Declaration() != "str" || action.Captures[1].Type.Declaration() != "int" {
		t.Fatal("captures lost their str/int types")
	}
	want := []ActionCase{
		{Leaf: action.Symbol.Package.ID + "::found", Status: 200},
		{Leaf: action.Symbol.Package.ID + "::missing", Status: 403},
		{Leaf: action.Symbol.Package.ID + "::unavailable", Status: 503},
	}
	if len(action.Cases) != len(want) {
		t.Fatalf("cases = %+v", action.Cases)
	}
	for i, kase := range want {
		if action.Cases[i] != kase {
			t.Fatalf("cases = %+v, want %+v", action.Cases, want)
		}
	}
}

func TestActionFormBodyChecks(t *testing.T) {
	form := "record line_wire\n" +
		"    str name\n" +
		"    str amount\n" +
		"fn save_outcome line_validated\n" +
		"    emits []\n" +
		"    given\n" +
		"        str invoice_id\n" +
		"        line_wire rows\n" +
		"    asserts\n" +
		"        sample: \"inv-1\", line_wire(\"seat\", \"2\") => ok saved(\"inv-1\")\n" +
		"    ok saved(invoice_id)\n" +
		"action append_line\n" +
		"    post \"/invoices/{invoice_id}/lines\"\n" +
		"    captures\n" +
		"        str invoice_id\n" +
		"    body form line_wire\n" +
		"    handles line_validated\n" +
		"    result save_outcome\n" +
		"    cases\n" +
		"        saved => 200\n" +
		"        rejected => 422\n" +
		"        stale => 409\n" +
		"        denied => 403\n" +
		"        busy => 503\n"
	program, err := programFixture(t, map[string]string{"src/web/web.can": actionWebFile(form)})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "append_line")
	if action.Body == nil || action.Body.Mode != "form" {
		t.Fatalf("form action lost its body: %+v", action.Body)
	}
	if len(action.Captures) != 1 || action.Captures[0].Name != "invoice_id" {
		t.Fatalf("form action captures = %+v", action.Captures)
	}
}

func TestActionGetPostSharePath(t *testing.T) {
	load := "fn load_outcome list_validated\n" +
		"    emits []\n" +
		"    asserts\n" +
		"        sample: => ok found(\"all\")\n" +
		"    ok found(\"all\")\n" +
		"action list_invoices\n" +
		"    get \"/invoices/save\"\n" +
		"    handles list_validated\n" +
		"    result load_outcome\n" +
		"    cases\n" +
		"        found => 200\n" +
		"        missing => 403\n" +
		"        unavailable => 503\n"
	program, err := programFixture(t, map[string]string{"src/web/web.can": actionWebFile(load)})
	if err != nil {
		t.Fatal(err)
	}
	get := actionByName(t, program, "list_invoices")
	post := actionByName(t, program, "save_invoice")
	if get.Method != "GET" || post.Method != "POST" || get.Path != post.Path {
		t.Fatal("GET and POST no longer share one path")
	}
	if len(program.Actions) != 3 {
		t.Fatalf("checked %d actions, want 3", len(program.Actions))
	}
}

func TestActionCrossPackageHandler(t *testing.T) {
	web := "package web\n" +
		"    provides [save_validated, invoice_wire, saved, rejected, stale, denied, busy, save_outcome]\n" +
		"    uses []\n" + actionSaveDomain + actionSaveHandler
	app := "package app\n" +
		"    provides [save_invoice]\n" +
		"    uses [web]\n" +
		"action save_invoice\n" +
		"    post \"/invoices/save\"\n" +
		"    body json web::invoice_wire\n" +
		"    handles web::save_validated\n" +
		"    result web::save_outcome\n" +
		"    cases\n" +
		"        web::saved => 200\n" +
		"        web::rejected => 422\n" +
		"        web::stale => 409\n" +
		"        web::denied => 403\n" +
		"        web::busy => 503\n" + actionMain
	program, err := programFixture(t, map[string]string{"src/web/web.can": web, "src/app/main.can": app})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "save_invoice")
	webID := ""
	for _, fn := range program.Functions {
		if fn.Symbol.Package.Name == "web" {
			webID = fn.Symbol.Package.ID
		}
	}
	if webID == "" {
		t.Fatal("web package missing")
	}
	if action.Handler != webID+"::save_validated" || action.Result.Declaration() != webID+"::save_outcome" {
		t.Fatalf("cross-package contract = %s %s", action.Handler, action.Result.Declaration())
	}
	for _, kase := range action.Cases {
		if !strings.HasPrefix(kase.Leaf, webID+"::") {
			t.Fatalf("case leaf %s escaped its package", kase.Leaf)
		}
	}
}

func TestActionWireFieldRenameRebuilds(t *testing.T) {
	renamed := strings.Replace(actionSaveDomain, "    str label\n    int seats\n", "    str title\n    int seats\n", 1)
	renamedHandler := strings.Replace(actionSaveHandler, "body.label", "body.title", 1)
	text := "package web\n    provides [save_invoice, load_line, invoice_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome]\n    uses []\n" + renamed + renamedHandler + actionSaveAction + actionLoadDomain + actionLoadHandler + actionLoadAction + actionMain
	program, err := programFixture(t, map[string]string{"src/web/web.can": text})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "save_invoice")
	fields := map[string]bool{}
	for _, field := range action.Body.Type.Fields() {
		fields[field.Name] = true
	}
	if !fields["title"] || fields["label"] {
		t.Fatalf("rebuilt wire fields = %v", fields)
	}
}

func TestActionRejects(t *testing.T) {
	web := actionWebFile()
	for name, tc := range map[string]struct {
		text string
		want string
	}{
		"duplicate action name": {
			actionWebFile(actionSaveAction),
			`duplicate name "save_invoice"`,
		},
		"action collides with function": {
			actionWebFile("fn str save_invoice\n    emits []\n    asserts\n        sample: => ok \"x\"\n    ok \"x\"\n"),
			`duplicate name "save_invoice"`,
		},
		"duplicate route": {
			actionWebFile("action save_again\n" +
				"    post \"/invoices/save\"\n" +
				"    body json invoice_wire\n" +
				"    handles save_validated\n" +
				"    result save_outcome\n" +
				"    cases\n" +
				"        saved => 200\n" +
				"        rejected => 422\n" +
				"        stale => 409\n" +
				"        denied => 403\n" +
				"        busy => 503\n"),
			"duplicates the POST /invoices/save route",
		},
		"duplicate capture shape": {
			actionWebFile("fn load_outcome other_validated\n" +
				"    emits []\n" +
				"    given\n" +
				"        str name\n" +
				"        int row\n" +
				"    asserts\n" +
				"        sample: \"n\", 1 => ok found(\"n\")\n" +
				"    ok found(name)\n" +
				"action load_other\n" +
				"    get \"/invoices/{name}/lines/{row}\"\n" +
				"    captures\n" +
				"        str name\n" +
				"        int row\n" +
				"    handles other_validated\n" +
				"    result load_outcome\n" +
				"    cases\n" +
				"        found => 200\n" +
				"        missing => 403\n" +
				"        unavailable => 503\n"),
			"duplicates the GET /invoices/{}/lines/{} route",
		},
		"path without slash": {
			strings.Replace(web, "post \"/invoices/save\"", "post \"invoices/save\"", 1),
			"invalid action route path",
		},
		"double slash": {
			strings.Replace(web, "post \"/invoices/save\"", "post \"//invoices/save\"", 1),
			"invalid action route path",
		},
		"reserved prefix": {
			strings.Replace(web, "post \"/invoices/save\"", "post \"/__can/save\"", 1),
			"invalid action route path",
		},
		"legacy capture": {
			strings.Replace(web, "post \"/invoices/save\"", "post \"/invoices/:id\"", 1),
			"invalid action route path",
		},
		"partial capture": {
			strings.Replace(web, "\"/invoices/{invoice_id}/lines/{line}\"", "\"/invoices/{invoice_id}.json\"", 1),
			"captures occupy one whole {name} segment",
		},
		"empty capture": {
			strings.Replace(web, "\"/invoices/{invoice_id}/lines/{line}\"", "\"/invoices/{}/lines\"", 1),
			"captures occupy one whole {name} segment",
		},
		"duplicate path capture": {
			strings.Replace(web, "\"/invoices/{invoice_id}/lines/{line}\"", "\"/invoices/{invoice_id}/{invoice_id}\"", 1),
			"duplicate path capture {invoice_id}",
		},
		"invalid capture name": {
			strings.Replace(web, "\"/invoices/{invoice_id}/lines/{line}\"", "\"/invoices/{Invoice_id}/lines/{line}\"", 1),
			"invalid path capture {Invoice_id}",
		},
		"capture without row": {
			strings.Replace(web, "    captures\n        str invoice_id\n        int line\n", "    captures\n        str invoice_id\n", 1),
			"path capture {line} has no captures row",
		},
		"row without capture": {
			strings.Replace(web, "    captures\n        str invoice_id\n        int line\n", "    captures\n        str invoice_id\n        int line\n        str extra\n", 1),
			"capture extra does not appear in the action path",
		},
		"non-scalar capture": {
			strings.Replace(web, "    captures\n        str invoice_id\n        int line\n", "    captures\n        bool invoice_id\n        int line\n", 1),
			"capture invoice_id must be str or int",
		},
		"duplicate capture rows": {
			strings.Replace(web, "    captures\n        str invoice_id\n        int line\n", "    captures\n        str invoice_id\n        str invoice_id\n        int line\n", 1),
			`duplicate field "invoice_id"`,
		},
		"body not a record": {
			strings.Replace(web, "    body json invoice_wire\n", "    body json save_outcome\n", 1),
			"action body must be a record wire type",
		},
		"body unknown type": {
			strings.Replace(web, "    body json invoice_wire\n", "    body json invoice_draft\n", 1),
			`no eligible declaration for "invoice_draft"`,
		},
		"result not a variant": {
			strings.Replace(web, "    result save_outcome\n", "    result saved\n", 1),
			"action result must be a finite variant",
		},
		"missing case": {
			strings.Replace(web, "        busy => 503\n", "", 1),
			"action cases omit result leaves",
		},
		"duplicate case": {
			strings.Replace(web, "        busy => 503\n", "        busy => 503\n        saved => 200\n", 1),
			"duplicate case for leaf saved",
		},
		"foreign case leaf": {
			strings.Replace(web, "        busy => 503\n", "        found => 503\n", 1),
			"case found is not a leaf of result",
		},
		"unknown case leaf": {
			strings.Replace(web, "        busy => 503\n", "        archived => 503\n", 1),
			`no eligible declaration for "archived"`,
		},
		"status too low": {
			strings.Replace(web, "        saved => 200\n", "        saved => 199\n", 1),
			"action status must be 200-599",
		},
		"status too high": {
			strings.Replace(web, "        saved => 200\n", "        saved => 600\n", 1),
			"action status must be 200-599",
		},
		"bodiless status": {
			strings.Replace(web, "        saved => 200\n", "        saved => 204\n", 1),
			"carries no representation",
		},
		"unknown handler": {
			strings.Replace(web, "    handles save_validated\n", "    handles save_missing\n", 1),
			`no eligible declaration for "save_missing"`,
		},
		"handler wrong arity": {
			strings.Replace(web, "        invoice_wire body\n", "        invoice_wire body\n        str note\n", 1),
			"takes 2 inputs, want 1 capture and body inputs",
		},
		"handler capture renamed": {
			strings.Replace(web, "        str invoice_id\n        int line\n    asserts\n        sample: \"inv-1\", 1 => ok found(\"inv-1\")\n    ok found(invoice_id)\n", "        str invoice_ref\n        int line\n    asserts\n        sample: \"inv-1\", 1 => ok found(\"inv-1\")\n    ok found(invoice_ref)\n", 1),
			"must be capture invoice_id",
		},
		"handler capture mistyped": {
			strings.Replace(web, "        str invoice_id\n        int line\n    asserts\n        sample: \"inv-1\", 1 => ok found(\"inv-1\")\n    ok found(invoice_id)\n", "        int invoice_id\n        int line\n    asserts\n        sample: 7, 1 => ok found(\"inv-1\")\n    ok found(\"inv-1\")\n", 1),
			"capture invoice_id must be",
		},
		"handler body mistyped": {
			strings.Replace(web, actionSaveHandler, strings.Replace(actionSaveHandler, "        invoice_wire body\n", "        saved body\n", 1), 1),
			"body input must be",
		},
		"handler wrong result": {
			strings.Replace(web, "fn save_outcome save_validated", "fn load_outcome save_validated", 1),
			"must return",
		},
		"handler emits": {
			actionEmitsFixture(web),
			"must emit []",
		},
		"handler generic": {
			strings.Replace(web, "fn save_outcome save_validated\n", "fn save_outcome save_validated<item>\n", 1),
			"must be non-generic",
		},
		"handler variadic": {
			strings.Replace(web, "        invoice_wire body\n", "        invoice_wire ...body\n", 1),
			"must not be variadic",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.text == web && !strings.Contains(name, "duplicate") {
				t.Fatal("invalid refusal fixture")
			}
			_, err := programFixture(t, map[string]string{"src/web/web.can": tc.text})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("action diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}

func actionEmitsFixture(web string) string {
	text := strings.Replace(web, "    uses []\n", "    uses [codec]\n", 1)
	emits := strings.Replace(actionSaveHandler, "    emits []\n", "    emits [codec::invalid_data]\n", 1)
	return strings.Replace(text, actionSaveHandler, emits, 1)
}

func TestActionFetchHandlerRejected(t *testing.T) {
	text := "package web\n" +
		"    provides [save_invoice, invoice_wire, saved, rejected, stale, denied, busy, save_outcome]\n" +
		"    uses [http]\n" + actionSaveDomain +
		"connection service\n" +
		"    endpoint \"http://127.0.0.1:1/\"\n" +
		"    auth bearer env \"CAN_ACTION_TOKEN\"\n" +
		"    timeout_ms 5000\n" +
		"    max_body_bytes 8192\n" +
		"fetch saved probe from service\n" +
		"    emits [http::request_failed]\n" +
		"    get \"/probe\"\n" +
		"action save_invoice\n" +
		"    post \"/invoices/save\"\n" +
		"    body json invoice_wire\n" +
		"    handles probe\n" +
		"    result save_outcome\n" +
		"    cases\n" +
		"        saved => 200\n" +
		"        rejected => 422\n" +
		"        stale => 409\n" +
		"        denied => 403\n" +
		"        busy => 503\n" + actionMain
	_, err := programFixture(t, map[string]string{"src/web/web.can": text})
	if err == nil || !strings.Contains(err.Error(), "must be a function") {
		t.Fatalf("fetch handler admitted or misdiagnosed: %v", err)
	}
}

func TestActionOwnerBodyRejected(t *testing.T) {
	owned := "owner record invoice_sealed\n" +
		"    str label\n" +
		"fn save_outcome sealed_validated\n" +
		"    emits []\n" +
		"    given\n" +
		"        invoice_sealed body\n" +
		"    asserts\n" +
		"        sample: invoice_sealed(\"inv-1\") => ok saved(\"inv-1\")\n" +
		"    ok saved(body.label)\n" +
		"action seal_invoice\n" +
		"    post \"/invoices/sealed\"\n" +
		"    body json invoice_sealed\n" +
		"    handles sealed_validated\n" +
		"    result save_outcome\n" +
		"    cases\n" +
		"        saved => 200\n" +
		"        rejected => 422\n" +
		"        stale => 409\n" +
		"        denied => 403\n" +
		"        busy => 503\n"
	_, err := programFixture(t, map[string]string{"src/web/web.can": actionWebFile(owned)})
	if err == nil || !strings.Contains(err.Error(), "is not codec-admissible") {
		t.Fatalf("owner wire body admitted or misdiagnosed: %v", err)
	}
}

func TestActionPrivateHandlerRejected(t *testing.T) {
	web := "package web\n" +
		"    provides [invoice_wire, saved, rejected, stale, denied, busy, save_outcome]\n" +
		"    uses []\n" + actionSaveDomain + actionSaveHandler
	app := "package app\n" +
		"    provides [save_invoice]\n" +
		"    uses [web]\n" +
		"action save_invoice\n" +
		"    post \"/invoices/save\"\n" +
		"    body json web::invoice_wire\n" +
		"    handles web::save_validated\n" +
		"    result web::save_outcome\n" +
		"    cases\n" +
		"        web::saved => 200\n" +
		"        web::rejected => 422\n" +
		"        web::stale => 409\n" +
		"        web::denied => 403\n" +
		"        web::busy => 503\n" + actionMain
	_, err := programFixture(t, map[string]string{"src/web/web.can": web, "src/app/main.can": app})
	if err == nil || !strings.Contains(err.Error(), "is private") {
		t.Fatalf("private handler admitted or misdiagnosed: %v", err)
	}
}

func TestActionResultLeafRenameDiagnoses(t *testing.T) {
	renamed := strings.Replace(actionWebFile(), "record busy\n", "record overloaded\n", 1)
	renamed = strings.Replace(renamed, "    busy\n", "    overloaded\n", 1)
	_, err := programFixture(t, map[string]string{"src/web/web.can": renamed})
	if err == nil || !strings.Contains(err.Error(), `"busy"`) {
		t.Fatalf("renamed result leaf admitted or misdiagnosed: %v", err)
	}
}

func TestActionDiagnosticSpans(t *testing.T) {
	handler := strings.Replace(actionWebFile(), "    handles save_validated\n", "    handles save_missing\n", 1)
	_, err := programFixture(t, map[string]string{"src/web/web.can": handler})
	if err == nil {
		t.Fatal("stale handler admitted")
	}
	located, ok := source.AsLocated(err)
	if !ok {
		t.Fatalf("handler failure lost its span: %v", err)
	}
	if !strings.HasSuffix(located.File, "web.can") {
		t.Fatalf("handler span file = %s", located.File)
	}
	if got := handler[located.Span.Start:located.Span.End]; got != "save_missing" {
		t.Fatalf("handler span covers %q", got)
	}
	path := strings.Replace(actionWebFile(), "post \"/invoices/save\"", "post \"invoices/save\"", 1)
	_, err = programFixture(t, map[string]string{"src/web/web.can": path})
	if err == nil {
		t.Fatal("invalid path admitted")
	}
	located, ok = source.AsLocated(err)
	if !ok {
		t.Fatalf("path failure lost its span: %v", err)
	}
	if got := path[located.Span.Start:located.Span.End]; got != `"invoices/save"` {
		t.Fatalf("path span covers %q", got)
	}
}

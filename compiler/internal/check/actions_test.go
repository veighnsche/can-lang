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

const actionLineKey = "record line_key\n" +
	"    str invoice_id\n" +
	"    int line\n"

const actionSaveAction = "action save_invoice\n" +
	"    post \"/invoices/save\"\n" +
	"    json invoice_wire limit 8192\n" +
	"    returns save_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        saved status 200\n" +
	"        rejected status 422\n" +
	"        stale status 409\n" +
	"        denied status 403\n" +
	"        busy status 503\n"

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

const actionLoadAction = "action load_line\n" +
	"    get \"/invoices/:invoice_id/lines/:line\"\n" +
	"    captures line_key\n" +
	"    input none\n" +
	"    returns load_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        found status 200\n" +
	"        missing status 403\n" +
	"        unavailable status 503\n"

func actionWebFile(extra ...string) string {
	decls := actionSaveDomain + actionLineKey + actionSaveAction + actionLoadDomain + actionLoadAction
	for _, text := range extra {
		decls += text
	}
	return "package web\n    provides [save_invoice, load_line, line_key, invoice_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome]\n    uses []\n" + decls + actionMain
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
	if len(action.Captures) != 0 || action.CapturesType != nil {
		t.Fatalf("exact-path action gained captures: %+v", action.Captures)
	}
	if action.Input.Mode != "json" || action.Input.Limit != 8192 {
		t.Fatalf("save action lost its json input: %+v", action.Input)
	}
	fields := map[string]bool{}
	for _, field := range action.Input.Type.Fields() {
		fields[field.Name] = true
	}
	if !fields["label"] || !fields["seats"] {
		t.Fatalf("input wire fields = %v", fields)
	}
	if action.Body != "json" {
		t.Fatalf("save response body = %s", action.Body)
	}
	wantResult := action.Symbol.Package.ID + "::save_outcome"
	if action.Returns.Declaration() != wantResult {
		t.Fatalf("returns = %s, want %s", action.Returns.Declaration(), wantResult)
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
	if action.Method != "GET" || action.Path != "/invoices/:invoice_id/lines/:line" {
		t.Fatalf("load route = %s %s", action.Method, action.Path)
	}
	if action.Input.Mode != "none" || action.Input.Type != nil {
		t.Fatalf("GET action gained a wire input: %+v", action.Input)
	}
	if action.CapturesType == nil || action.CapturesType.Declaration() != action.Symbol.Package.ID+"::line_key" {
		t.Fatalf("load lost its captures record: %+v", action.CapturesType)
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

func TestActionFormInputChecks(t *testing.T) {
	form := "record invoice_key\n" +
		"    str invoice_id\n" +
		"record line_wire\n" +
		"    str name\n" +
		"    str amount\n" +
		"action append_line\n" +
		"    post \"/invoices/:invoice_id/lines\"\n" +
		"    captures invoice_key\n" +
		"    form line_wire limit 2048\n" +
		"    returns save_outcome\n" +
		"    body html\n" +
		"    cases\n" +
		"        saved status 200 swap inner\n" +
		"        rejected status 422 swap inner\n" +
		"        stale status 409 swap inner\n" +
		"        denied status 403 swap inner\n" +
		"        busy status 503 swap inner\n"
	program, err := programFixture(t, map[string]string{"src/web/web.can": actionWebFile(form)})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "append_line")
	if action.Input.Mode != "form" || action.Input.Limit != 2048 || action.Input.RowsLimit != 0 {
		t.Fatalf("form action lost its input: %+v", action.Input)
	}
	if action.Body != "html" {
		t.Fatalf("form action response body = %s", action.Body)
	}
	if len(action.Captures) != 1 || action.Captures[0].Name != "invoice_id" {
		t.Fatalf("form action captures = %+v", action.Captures)
	}
	for _, kase := range action.Cases {
		if kase.Swap != "inner" {
			t.Fatalf("HTML case lost its swap policy: %+v", kase)
		}
	}
	if action.ResponseSchema != nil {
		t.Fatal("HTML action gained a JSON response schema")
	}
}

func TestActionGetPostSharePath(t *testing.T) {
	load := "action list_invoices\n" +
		"    get \"/invoices/save\"\n" +
		"    input none\n" +
		"    returns load_outcome\n" +
		"    body json\n" +
		"    cases\n" +
		"        found status 200\n" +
		"        missing status 403\n" +
		"        unavailable status 503\n"
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

func TestActionStaticCaptureOverlapAllowed(t *testing.T) {
	// A static route and a capture route of one method overlap with static
	// priority, and one capture shape may serve both methods.
	extra := "action load_new\n" +
		"    get \"/invoices/new\"\n" +
		"    input none\n" +
		"    returns load_outcome\n" +
		"    body json\n" +
		"    cases\n" +
		"        found status 200\n" +
		"        missing status 403\n" +
		"        unavailable status 503\n" +
		"record name_key\n" +
		"    str name\n" +
		"action load_name\n" +
		"    get \"/invoices/:name\"\n" +
		"    captures name_key\n" +
		"    input none\n" +
		"    returns load_outcome\n" +
		"    body json\n" +
		"    cases\n" +
		"        found status 200\n" +
		"        missing status 403\n" +
		"        unavailable status 503\n" +
		"action post_name\n" +
		"    post \"/invoices/:name\"\n" +
		"    captures name_key\n" +
		"    json invoice_wire limit 512\n" +
		"    returns save_outcome\n" +
		"    body json\n" +
		"    cases\n" +
		"        saved status 200\n" +
		"        rejected status 422\n" +
		"        stale status 409\n" +
		"        denied status 403\n" +
		"        busy status 503\n"
	program, err := programFixture(t, map[string]string{"src/web/web.can": actionWebFile(extra)})
	if err != nil {
		t.Fatal(err)
	}
	if len(program.Actions) != 5 {
		t.Fatalf("checked %d actions, want 5", len(program.Actions))
	}
	static := actionByName(t, program, "load_new")
	captured := actionByName(t, program, "load_name")
	if len(static.Captures) != 0 || len(captured.Captures) != 1 {
		t.Fatal("static and capture routes lost their shapes")
	}
}

func TestActionCrossPackageContract(t *testing.T) {
	// The shared contract package exports wire, result and leaf types;
	// the application package declares a handler-free action over them
	// without importing anything executable.
	web := "package web\n" +
		"    provides [invoice_wire, saved, rejected, stale, denied, busy, save_outcome]\n" +
		"    uses []\n" + actionSaveDomain
	app := "package app\n" +
		"    provides [save_invoice]\n" +
		"    uses [web]\n" +
		"action save_invoice\n" +
		"    post \"/invoices/save\"\n" +
		"    json web::invoice_wire limit 1024\n" +
		"    returns web::save_outcome\n" +
		"    body json\n" +
		"    cases\n" +
		"        web::saved status 200\n" +
		"        web::rejected status 422\n" +
		"        web::stale status 409\n" +
		"        web::denied status 403\n" +
		"        web::busy status 503\n" + actionMain
	program, err := programFixture(t, map[string]string{"src/web/web.can": web, "src/app/main.can": app})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "save_invoice")
	webID := strings.TrimSuffix(action.Returns.Declaration(), "::save_outcome")
	if action.Input.Type.Declaration() != webID+"::invoice_wire" {
		t.Fatalf("cross-package input = %s", action.Input.Type.Declaration())
	}
	for _, kase := range action.Cases {
		if !strings.HasPrefix(kase.Leaf, webID+"::") {
			t.Fatalf("case leaf %s escaped its package", kase.Leaf)
		}
	}
	for _, fn := range program.Functions {
		if fn.Symbol.Package.Name != "app" {
			t.Fatalf("contract package carries executable %s", fn.Symbol.ID)
		}
	}
}

func TestActionWireFieldRenameRebuilds(t *testing.T) {
	renamed := strings.Replace(actionSaveDomain, "    str label\n    int seats\n", "    str title\n    int seats\n", 1)
	text := "package web\n    provides [save_invoice, load_line, line_key, invoice_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome]\n    uses []\n" + renamed + actionLineKey + actionSaveAction + actionLoadDomain + actionLoadAction + actionMain
	program, err := programFixture(t, map[string]string{"src/web/web.can": text})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "save_invoice")
	fields := map[string]bool{}
	for _, field := range action.Input.Type.Fields() {
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
				"    json invoice_wire limit 100\n" +
				"    returns save_outcome\n" +
				"    body json\n" +
				"    cases\n" +
				"        saved status 200\n" +
				"        rejected status 422\n" +
				"        stale status 409\n" +
				"        denied status 403\n" +
				"        busy status 503\n"),
			"duplicates the POST /invoices/save route",
		},
		"duplicate capture shape": {
			actionWebFile("record other_key\n" +
				"    str name\n" +
				"    int row\n" +
				"action load_other\n" +
				"    get \"/invoices/:name/lines/:row\"\n" +
				"    captures other_key\n" +
				"    input none\n" +
				"    returns load_outcome\n" +
				"    body json\n" +
				"    cases\n" +
				"        found status 200\n" +
				"        missing status 403\n" +
				"        unavailable status 503\n"),
			"duplicates the GET /invoices/{}/lines/{} route",
		},
		"encoded static duplicate": {
			actionWebFile("action load_plain\n" +
				"    get \"/invoices/new\"\n" +
				"    input none\n" +
				"    returns load_outcome\n" +
				"    body json\n" +
				"    cases\n" +
				"        found status 200\n" +
				"        missing status 403\n" +
				"        unavailable status 503\n" +
				"action load_encoded\n" +
				"    get \"/invoices/n%65w\"\n" +
				"    input none\n" +
				"    returns load_outcome\n" +
				"    body json\n" +
				"    cases\n" +
				"        found status 200\n" +
				"        missing status 403\n" +
				"        unavailable status 503\n"),
			"duplicates the GET /invoices/new route",
		},
		"ambiguous capture overlap": {
			actionWebFile("record left_key\n" +
				"    str name\n" +
				"record right_key\n" +
				"    str row\n" +
				"action load_left\n" +
				"    get \"/invoices/:name/lines\"\n" +
				"    captures left_key\n" +
				"    input none\n" +
				"    returns load_outcome\n" +
				"    body json\n" +
				"    cases\n" +
				"        found status 200\n" +
				"        missing status 403\n" +
				"        unavailable status 503\n" +
				"action load_right\n" +
				"    get \"/invoices/new/:row\"\n" +
				"    captures right_key\n" +
				"    input none\n" +
				"    returns load_outcome\n" +
				"    body json\n" +
				"    cases\n" +
				"        found status 200\n" +
				"        missing status 403\n" +
				"        unavailable status 503\n"),
			"ambiguously overlaps the GET /invoices/new/{} route",
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
		"legacy braces": {
			strings.Replace(web, "post \"/invoices/save\"", "post \"/invoices/{id}\"", 1),
			"captures occupy one whole :name segment",
		},
		"partial capture": {
			strings.Replace(web, "\"/invoices/:invoice_id/lines/:line\"", "\"/invoices/v:1/lines/:line\"", 1),
			"captures occupy one whole :name segment",
		},
		"empty capture": {
			strings.Replace(web, "\"/invoices/:invoice_id/lines/:line\"", "\"/invoices/:/lines\"", 1),
			"captures occupy one whole :name segment",
		},
		"duplicate path capture": {
			strings.Replace(web, "\"/invoices/:invoice_id/lines/:line\"", "\"/invoices/:invoice_id/:invoice_id\"", 1),
			"duplicate path capture :invoice_id",
		},
		"invalid capture name": {
			strings.Replace(web, "\"/invoices/:invoice_id/lines/:line\"", "\"/invoices/:Invoice_id/lines/:line\"", 1),
			"invalid path capture :Invoice_id",
		},
		"capture without record": {
			strings.Replace(web, "    captures line_key\n    input none\n", "    input none\n", 1),
			"action requires a captures record",
		},
		"record without capture": {
			strings.Replace(web, "    post \"/invoices/save\"\n    json invoice_wire limit 8192\n", "    post \"/invoices/save\"\n    captures line_key\n    json invoice_wire limit 8192\n", 1),
			"action path declares no captures",
		},
		"captures not a record": {
			strings.Replace(web, "    captures line_key\n", "    captures load_outcome\n", 1),
			"action captures must be a record type",
		},
		"capture without field": {
			strings.Replace(web, "record line_key\n    str invoice_id\n    int line\n", "record line_key\n    str invoice_id\n", 1),
			"path capture :line has no captures field",
		},
		"field without capture": {
			strings.Replace(web, "record line_key\n    str invoice_id\n    int line\n", "record line_key\n    str invoice_id\n    int line\n    str extra\n", 1),
			"capture extra does not appear in the action path",
		},
		"non-scalar capture": {
			strings.Replace(web, "record line_key\n    str invoice_id\n    int line\n", "record line_key\n    bool invoice_id\n    int line\n", 1),
			"capture invoice_id must be str or int",
		},
		"input not a record": {
			strings.Replace(web, "    json invoice_wire limit 8192\n", "    json save_outcome limit 8192\n", 1),
			"action json input must be a record wire type",
		},
		"input unknown type": {
			strings.Replace(web, "    json invoice_wire limit 8192\n", "    json invoice_draft limit 8192\n", 1),
			`no eligible declaration for "invoice_draft"`,
		},
		"limit zero": {
			strings.Replace(web, "    json invoice_wire limit 8192\n", "    json invoice_wire limit 0\n", 1),
			"action limit must be a positive byte count",
		},
		"json input html body": {
			strings.Replace(web, "    json invoice_wire limit 8192\n    returns save_outcome\n    body json\n", "    json invoice_wire limit 8192\n    returns save_outcome\n    body html\n", 1),
			"json input requires body json",
		},
		"get html body": {
			strings.Replace(web, "    input none\n    returns load_outcome\n    body json\n", "    input none\n    returns load_outcome\n    body html\n", 1),
			"GET actions use body json",
		},
		"swap on json": {
			strings.Replace(web, "        saved status 200\n", "        saved status 200 swap inner\n", 1),
			"swap applies to html actions only",
		},
		"returns not a variant": {
			strings.Replace(web, "    returns save_outcome\n", "    returns saved\n", 1),
			"action returns must be a finite variant",
		},
		"missing case": {
			strings.Replace(web, "        busy status 503\n", "", 1),
			"action cases omit returns leaves",
		},
		"duplicate case": {
			strings.Replace(web, "        busy status 503\n", "        busy status 503\n        saved status 200\n", 1),
			"duplicate case for leaf saved",
		},
		"foreign case leaf": {
			strings.Replace(web, "        busy status 503\n", "        found status 503\n", 1),
			"case found is not a leaf of returns",
		},
		"unknown case leaf": {
			strings.Replace(web, "        busy status 503\n", "        archived status 503\n", 1),
			`no eligible declaration for "archived"`,
		},
		"status too low": {
			strings.Replace(web, "        saved status 200\n", "        saved status 199\n", 1),
			"action status must be 200-599",
		},
		"status too high": {
			strings.Replace(web, "        saved status 200\n", "        saved status 600\n", 1),
			"action status must be 200-599",
		},
		"bodiless status": {
			strings.Replace(web, "        saved status 200\n", "        saved status 204\n", 1),
			"carries no representation",
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

func TestActionResultLeafRenameDiagnoses(t *testing.T) {
	renamed := strings.Replace(actionWebFile(), "record busy\n", "record overloaded\n", 1)
	renamed = strings.Replace(renamed, "    busy\n", "    overloaded\n", 1)
	_, err := programFixture(t, map[string]string{"src/web/web.can": renamed})
	if err == nil || !strings.Contains(err.Error(), `"busy"`) {
		t.Fatalf("renamed result leaf admitted or misdiagnosed: %v", err)
	}
}

func TestActionDiagnosticSpans(t *testing.T) {
	limited := strings.Replace(actionWebFile(), "    json invoice_wire limit 8192\n", "    json invoice_wire limit 0\n", 1)
	_, err := programFixture(t, map[string]string{"src/web/web.can": limited})
	if err == nil {
		t.Fatal("zero limit admitted")
	}
	located, ok := source.AsLocated(err)
	if !ok {
		t.Fatalf("limit failure lost its span: %v", err)
	}
	if !strings.HasSuffix(located.File, "web.can") {
		t.Fatalf("limit span file = %s", located.File)
	}
	if got := limited[located.Span.Start:located.Span.End]; got != "0" {
		t.Fatalf("limit span covers %q", got)
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

func TestActionFormRowsLimitChecks(t *testing.T) {
	rows := "record line_wire\n" +
		"    str sku\n" +
		"record batch_wire\n" +
		"    str customer\n" +
		"    form::rows<line_wire> lines\n" +
		"record stored\n" +
		"    str label\n" +
		"record store_failed\n" +
		"    str reason\n" +
		"variant store_outcome\n" +
		"    stored\n" +
		"    store_failed\n" +
		"action append_batch\n" +
		"    post \"/invoices/append\"\n" +
		"    form batch_wire limit 2048 rows_limit 64\n" +
		"    returns store_outcome\n" +
		"    body html\n" +
		"    cases\n" +
		"        stored status 200 swap inner\n" +
		"        store_failed status 422 swap inner\n"
	provides := "append_batch, line_wire, batch_wire, stored, store_failed, store_outcome"
	text := "package web\n    provides [" + provides + "]\n    uses [form]\n" + rows + actionMain
	program, err := programFixture(t, map[string]string{"src/web/web.can": text})
	if err != nil {
		t.Fatal(err)
	}
	action := actionByName(t, program, "append_batch")
	if action.Input.Mode != "form" || action.Input.Limit != 2048 || action.Input.RowsLimit != 64 {
		t.Fatalf("rows input lost its limits: %+v", action.Input)
	}
	for _, tc := range []struct {
		name string
		text string
		want string
	}{
		{
			"missing rows limit",
			strings.Replace(text, "    form batch_wire limit 2048 rows_limit 64\n", "    form batch_wire limit 2048\n", 1),
			"form input with keyed rows requires rows_limit",
		},
		{
			"rows limit range",
			strings.Replace(text, "    form batch_wire limit 2048 rows_limit 64\n", "    form batch_wire limit 2048 rows_limit 65\n", 1),
			"action rows_limit must be 1-64",
		},
		{
			"rows limit zero",
			strings.Replace(text, "    form batch_wire limit 2048 rows_limit 64\n", "    form batch_wire limit 2048 rows_limit 0\n", 1),
			"action rows_limit must be 1-64",
		},
		{
			"rows limit without rows",
			strings.Replace(text, "    form batch_wire limit 2048 rows_limit 64\n", "    form line_wire limit 2048 rows_limit 64\n", 1),
			"form input without keyed rows takes no rows_limit",
		},
		{
			"form input json body",
			strings.Replace(text, "    returns store_outcome\n    body html\n", "    returns store_outcome\n    body json\n", 1),
			"form input requires body html",
		},
		{
			"missing swap",
			strings.Replace(text, "        stored status 200 swap inner\n", "        stored status 200\n", 1),
			"html action cases require swap inner",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/web/web.can": tc.text})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("rows diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}

func TestActionOwnerInputRejected(t *testing.T) {
	owned := "owner record invoice_sealed\n" +
		"    str label\n" +
		"action seal_invoice\n" +
		"    post \"/invoices/sealed\"\n" +
		"    json invoice_sealed limit 512\n" +
		"    returns save_outcome\n" +
		"    body json\n" +
		"    cases\n" +
		"        saved status 200\n" +
		"        rejected status 422\n" +
		"        stale status 409\n" +
		"        denied status 403\n" +
		"        busy status 503\n"
	_, err := programFixture(t, map[string]string{"src/web/web.can": actionWebFile(owned)})
	if err == nil || !strings.Contains(err.Error(), "is not codec-admissible") {
		t.Fatalf("owner wire input admitted or misdiagnosed: %v", err)
	}
}

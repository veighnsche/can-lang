package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/source"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

const formWireDomain = "record line_wire\n" +
	"    str sku\n" +
	"    str[] tags\n" +
	"    option::value<str> note\n" +
	"record invoice_wire\n" +
	"    str customer\n" +
	"    form::rows<line_wire> lines\n" +
	"record saved\n" +
	"    str label\n" +
	"record rejected\n" +
	"    str reason\n" +
	"variant save_outcome\n" +
	"    saved\n" +
	"    rejected\n"

const formSaveHandler = "fn save_outcome save_validated\n" +
	"    emits []\n" +
	"    given\n" +
	"        invoice_wire body\n" +
	"    asserts\n" +
	"        sample: invoice_wire(\"c\", form::rows<line_wire>([], [])) => ok saved(\"c\")\n" +
	"    ok saved(body.customer)\n"

const formSaveAction = "action save_invoice\n" +
	"    post \"/invoices/save\"\n" +
	"    body form invoice_wire\n" +
	"    handles save_validated\n" +
	"    result save_outcome\n" +
	"    cases\n" +
	"        saved => 200\n" +
	"        rejected => 422\n"

const formRenderers = "fn html::safe render_outcome\n" +
	"    emits []\n" +
	"    given\n" +
	"        save_outcome outcome\n" +
	"    asserts\n" +
	"        sample: saved(\"c\") => ok\n" +
	"    ok call html::text_fragment(\"done\")\n" +
	"fn html::safe render_rejected\n" +
	"    emits []\n" +
	"    given\n" +
	"        form::rejected<invoice_wire> bad\n" +
	"    asserts\n" +
	"        sample: form::rejected<invoice_wire>([], []) => ok\n" +
	"    ok call html::text_fragment(\"bad\")\n"

const formMounted = "fn http::router mounted\n" +
	"    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n" +
	"    asserts\n" +
	"        sample: => ok\n" +
	"    match chain\n" +
	"        call http::serve_form_action<save_outcome, invoice_wire>(\"save_invoice\", callable render_outcome, callable render_rejected) as http::route form\n" +
	"        call http::make_router([form]) as http::router router\n" +
	"        http::invalid_route\n" +
	"        http::duplicate_route\n" +
	"        http::ambiguous_route\n" +
	"        ok => ok router\n"

const formBuilderFns = "fn form::collection use_collection\n" +
	"    emits [form::unknown_field]\n" +
	"    asserts\n" +
	"        sample: => ok\n" +
	"    match call form::named_collection<invoice_wire>(\"lines\")\n" +
	"        form::unknown_field\n" +
	"        ok form::collection coll => ok coll\n" +
	"fn form::field use_field\n" +
	"    emits [form::unknown_field]\n" +
	"    asserts\n" +
	"        sample: => ok\n" +
	"    match call form::named_field<line_wire>(\"sku\")\n" +
	"        form::unknown_field\n" +
	"        ok form::field field => ok field\n" +
	"fn str use_names\n" +
	"    emits [form::unknown_field, form::invalid_name]\n" +
	"    asserts\n" +
	"        sample: => ok \"lines[a1][sku]\"\n" +
	"    match chain\n" +
	"        call form::named_collection<invoice_wire>(\"lines\") as form::collection coll\n" +
	"        call form::named_field<line_wire>(\"sku\") as form::field field\n" +
	"        call form::row_key(\"a1\") as form::row row\n" +
	"        form::unknown_field\n" +
	"        form::invalid_name\n" +
	"        ok => ok call form::input_name(coll, row, field)\n"

func formWebFile(extra ...string) string {
	decls := formWireDomain + formSaveHandler + formSaveAction + formRenderers + formMounted + formBuilderFns
	for _, text := range extra {
		decls += text
	}
	return "package web\n    provides []\n    uses [form, http, html, option]\n" + decls + actionMain
}

func TestFormServeActionBinds(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/web/web.can": formWebFile()})
	if err != nil {
		t.Fatal(err)
	}
	var serve *FormSpecialization
	for _, special := range program.Forms {
		if special.Operation == formServeAction {
			serve = special
		}
	}
	if serve == nil {
		t.Fatalf("serve specialization missing: %+v", program.Forms)
	}
	contract := serve.Contract
	if len(contract.Inputs()) != 3 || contract.Result().Declaration() != "can.std.http@1::route" {
		t.Fatalf("serve contract = %+v", contract)
	}
	if len(contract.Errors()) != 1 || contract.Errors()[0].Declaration() != "can.std.http@1::invalid_route" {
		t.Fatalf("serve errors = %+v", contract.Errors())
	}
	outcome, structural := contract.Inputs()[1], contract.Inputs()[2]
	if outcome.Result().Declaration() != "can.std.html@1::safe" || len(outcome.Errors()) != 0 {
		t.Fatalf("outcome renderer = %+v", outcome)
	}
	if structural.Result().Declaration() != "can.std.html@1::safe" || len(structural.Errors()) != 0 {
		t.Fatalf("structural renderer = %+v", structural)
	}
	if got := outcome.Inputs()[0].Declaration(); !strings.HasSuffix(got, "::save_outcome") {
		t.Fatalf("outcome input = %s", got)
	}
	if got := structural.Inputs()[0].Declaration(); got != "can.std.form@1::rejected" {
		t.Fatalf("structural input = %s", got)
	}
	if serve.Rejected == nil || serve.Rejected.Identity() != structural.Inputs()[0].Identity() {
		t.Fatal("serve lost its rejected wire identity")
	}
}

func TestFormNamedBuildersCheck(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/web/web.can": formWebFile()})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]*FormSpecialization{}
	for _, special := range program.Forms {
		seen[special.Operation] = special
	}
	for _, operation := range []string{formNamedCollection, formNamedField} {
		special := seen[operation]
		if special == nil {
			t.Fatalf("builder %s missing", operation)
		}
		contract := special.Contract
		if len(contract.Inputs()) != 1 || contract.Inputs()[0].Declaration() != "str" {
			t.Fatalf("builder %s inputs = %+v", operation, contract.Inputs())
		}
		if len(contract.Errors()) != 1 || contract.Errors()[0].Declaration() != "can.std.form@1::unknown_field" {
			t.Fatalf("builder %s errors = %+v", operation, contract.Errors())
		}
	}
	if got := seen[formNamedCollection].Contract.Result().Declaration(); got != "can.std.form@1::collection" {
		t.Fatalf("collection result = %s", got)
	}
	if got := seen[formNamedField].Contract.Result().Declaration(); got != "can.std.form@1::field" {
		t.Fatalf("field result = %s", got)
	}
}

func TestFormRowItemIdentityFilled(t *testing.T) {
	program, err := programFixture(t, map[string]string{"src/web/web.can": formWebFile()})
	if err != nil {
		t.Fatal(err)
	}
	var action *ActionDeclaration
	for _, candidate := range program.Actions {
		if candidate.Symbol.Name == "save_invoice" {
			action = candidate
		}
	}
	if action == nil || action.Body == nil {
		t.Fatal("save action lost its form body")
	}
	var rows *types.FormRowSchema
	for _, field := range action.Body.Form.Fields {
		if field.Name == "lines" {
			rows = field.Rows
		}
	}
	if rows == nil || rows.Order != "lines_order" || rows.Collection == "" || rows.Item == "" {
		t.Fatalf("row schema incomplete: %+v", rows)
	}
	if len(rows.Fields) != 3 || rows.Fields[0].Kind != "str" || rows.Fields[1].Kind != "array" || rows.Fields[2].Kind != "optional" {
		t.Fatalf("row fields = %+v", rows.Fields)
	}
}

func TestFormBuilderRefactorDiagnostics(t *testing.T) {
	base := formWebFile()
	cases := []struct {
		name string
		edit func(string) string
		want string
	}{
		{"renamed collection", func(s string) string {
			return strings.Replace(s, "form::named_collection<invoice_wire>(\"lines\")", "form::named_collection<invoice_wire>(\"rows\")", 1)
		}, "has no field"},
		{"collection over scalar", func(s string) string {
			return strings.Replace(s, "form::named_collection<invoice_wire>(\"lines\")", "form::named_collection<invoice_wire>(\"customer\")", 1)
		}, "not a keyed rows collection"},
		{"renamed row field", func(s string) string {
			return strings.Replace(s, "form::named_field<line_wire>(\"sku\")", "form::named_field<line_wire>(\"stock\")", 1)
		}, "has no field"},
		{"field over rows", func(s string) string {
			return strings.Replace(s, "form::named_field<line_wire>(\"sku\")", "form::named_field<invoice_wire>(\"lines\")", 1)
		}, "named_collection"},
		{"non-literal name", func(s string) string {
			return strings.Replace(s, "    match call form::named_collection<invoice_wire>(\"lines\")", "    str tag = \"lines\"\n    match call form::named_collection<invoice_wire>(tag)", 1)
		}, "static string literal"},
		{"non-record wire", func(s string) string {
			return strings.Replace(s, "form::named_collection<invoice_wire>(\"lines\")", "form::named_collection<str>(\"lines\")", 1)
		}, "ordinary record"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/web/web.can": tc.edit(base)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("builder refactor admitted: %v", err)
			}
		})
	}
}

func TestFormServeActionRejects(t *testing.T) {
	base := formWebFile()
	serve := "http::serve_form_action<save_outcome, invoice_wire>(\"save_invoice\", callable render_outcome, callable render_rejected)"
	cases := []struct {
		name string
		edit func(string) string
		want string
	}{
		{"unknown action", func(s string) string {
			return strings.Replace(s, serve, strings.Replace(serve, "\"save_invoice\"", "\"missing_invoice\"", 1), 1)
		}, "unknown form action"},
		{"result mismatch", func(s string) string {
			return strings.Replace(s, "serve_form_action<save_outcome, invoice_wire>", "serve_form_action<saved, invoice_wire>", 1)
		}, "requires a finite variant"},
		{"wire mismatch", func(s string) string {
			return strings.Replace(s, "serve_form_action<save_outcome, invoice_wire>", "serve_form_action<save_outcome, line_wire>", 1)
		}, "wire is"},
		{"outcome arity", func(s string) string {
			return strings.Replace(s, "callable render_outcome", "callable render_rejected", 1)
		}, "does not fit"},
		{"non-literal action", func(s string) string {
			s = strings.Replace(s, "    match chain\n        call http::serve_form_action", "    str action_name = \"save_invoice\"\n    match chain\n        call http::serve_form_action", 1)
			return strings.Replace(s, "\"save_invoice\", callable render_outcome", "action_name, callable render_outcome", 1)
		}, "static string literal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := programFixture(t, map[string]string{"src/web/web.can": tc.edit(base)})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("serve misuse admitted: %v", err)
			}
		})
	}
}

func TestFormServeRequiresFormBody(t *testing.T) {
	json := strings.Replace(formWebFile(), "    body form invoice_wire\n", "    body json invoice_wire\n", 1)
	_, err := programFixture(t, map[string]string{"src/web/web.can": json})
	if err == nil || !strings.Contains(err.Error(), "carries no form body") {
		t.Fatalf("json action served as form: %v", err)
	}
	renamed := strings.Replace(formWebFile(), "    body form invoice_wire\n", "    body form line_wire\n", 1)
	_, err = programFixture(t, map[string]string{"src/web/web.can": renamed})
	if err == nil {
		t.Fatal("rewired form body admitted")
	}
}

func TestFormDiagnosticSpans(t *testing.T) {
	edited := strings.Replace(formWebFile(), "form::named_collection<invoice_wire>(\"lines\")", "form::named_collection<invoice_wire>(\"rows\")", 1)
	_, err := programFixture(t, map[string]string{"src/web/web.can": edited})
	if err == nil {
		t.Fatal("renamed collection admitted")
	}
	located, ok := source.AsLocated(err)
	if !ok {
		t.Fatalf("builder failure lost its span: %v", err)
	}
	if got := edited[located.Span.Start:located.Span.End]; got != `"rows"` {
		t.Fatalf("builder span covers %q", got)
	}
	served := strings.Replace(formWebFile(), "\"save_invoice\", callable render_outcome", "\"missing_invoice\", callable render_outcome", 1)
	_, err = programFixture(t, map[string]string{"src/web/web.can": served})
	if err == nil {
		t.Fatal("unknown action admitted")
	}
	located, ok = source.AsLocated(err)
	if !ok {
		t.Fatalf("serve failure lost its span: %v", err)
	}
	if got := served[located.Span.Start:located.Span.End]; got != `"missing_invoice"` {
		t.Fatalf("serve span covers %q", got)
	}
}

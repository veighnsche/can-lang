package emit

import (
	"strings"
	"testing"
)

const formActionEmitWeb = "package web\n" +
	"    provides []\n" +
	"    uses [form, http, html, option]\n" +
	"record line_wire\n" +
	"    str sku\n" +
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
	"    rejected\n" +
	"fn save_outcome save_validated\n" +
	"    emits []\n" +
	"    given\n" +
	"        invoice_wire body\n" +
	"    asserts\n" +
	"        sample: invoice_wire(\"c\", form::rows<line_wire>([], [])) => ok saved(\"c\")\n" +
	"    ok saved(body.customer)\n" +
	"action save_invoice\n" +
	"    post \"/invoices/save\"\n" +
	"    form invoice_wire limit 2048 rows_limit 64\n" +
	"    returns save_outcome\n" +
	"    body html\n" +
	"    cases\n" +
	"        saved status 200 swap inner\n" +
	"        rejected status 422 swap inner\n" +
	"fn html::safe render_outcome\n" +
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
	"    ok call html::text_fragment(\"bad\")\n" +
	"fn http::router mounted\n" +
	"    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n" +
	"    asserts\n" +
	"        sample: => ok\n" +
	"    match chain\n" +
	"        call http::serve_form_action<save_outcome, invoice_wire>(\"save_invoice\", callable render_outcome, callable render_rejected) as http::route form\n" +
	"        call http::make_router([form]) as http::router router\n" +
	"        http::invalid_route\n" +
	"        http::duplicate_route\n" +
	"        http::ambiguous_route\n" +
	"        ok => ok router\n" +
	"fn form::collection use_collection\n" +
	"    emits [form::unknown_field]\n" +
	"    asserts\n" +
	"        sample: => ok\n" +
	"    match call form::named_collection<invoice_wire>(\"lines\")\n" +
	"        form::unknown_field\n" +
	"        ok form::collection coll => ok coll\n" +
	"fn void main\n" +
	"    emits []\n" +
	"    given\n" +
	"        str[] arguments\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    ok\n"

func TestFormActionEmissionFreezesRowsSchema(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/web/web.can": formActionEmitWeb})
	webID := program.Actions[0].Symbol.Package.ID
	joined := emittedBody(t, program)
	for _, want := range []string{
		`"input":{"mode":"form","type":"` + webID + `::invoice_wire","limit":2048,"rowsLimit":64,"schema":{"root":`,
		`{"name":"lines","kind":"rows","rows":{"row":`,
		`"order":"lines_order"`,
		`"fields":[{"name":"sku","kind":"str"},{"name":"note","kind":"optional"`,
		`"collection":"`,
		`"item":"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("emitted form body omits %s", want)
		}
	}
	if strings.Contains(joined, `"item":""`) {
		t.Fatal("emitted rows schema left the item identity empty")
	}
}

func TestFormServeEmissionSplicesAdapterContract(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/web/web.can": formActionEmitWeb})
	webID := program.Actions[0].Symbol.Package.ID
	saved, missing := "", ""
	for _, leaf := range program.Actions[0].Returns.Leaves() {
		if strings.HasSuffix(leaf.Declaration(), "::saved") {
			saved = leaf.Identity()
		}
		if strings.HasSuffix(leaf.Declaration(), "::rejected") {
			missing = leaf.Identity()
		}
	}
	joined := emittedBody(t, program)
	for _, want := range []string{
		`$canFormActions.serve($canExpr`,
		`"action":"` + webID + `::save_invoice"`,
		`"method":"POST"`,
		`"path":"/invoices/save"`,
		`"cases":[{"leaf":"` + saved + `","status":200},{"leaf":"` + missing + `","status":422}]`,
		`"rejected":"`,
		`"rawEntry":"`,
		`"issue":"`,
		`$canForm.namedCollection($canExpr`,
		`export let $canForm:ReturnType<typeof $canCreateForm>;`,
		`export let $canFormActions:ReturnType<typeof $canCreateFormActions>;`,
		`$canForm=$canCreateForm($canDomain,`,
		`$canFormActions=$canCreateFormActions($canDomain,`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("emitted adapter omits %s", want)
		}
	}
	if !strings.Contains(joined, ", $canFunction") {
		t.Fatal("serve site lost its spliced handler binding")
	}
}

func TestFormStateValuesReachAuthoredModules(t *testing.T) {
	for name, names := range map[string][]ImportName{
		"server":  stateValueImportNames(),
		"browser": browserStateValueImportNames(),
	} {
		have := map[string]bool{}
		for _, entry := range names {
			have[entry.Local] = true
		}
		for _, want := range []string{"$canForm", "$canFormActions"} {
			if !have[want] {
				t.Fatalf("%s state imports omit %s", name, want)
			}
		}
	}
}

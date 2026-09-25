package emit

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

const fetchActionEmitWeb = "package web\n" +
	"    provides [save_invoice, load_line, line_key, invoice_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome]\n" +
	"    uses [http, codec]\n" +
	"record invoice_wire\n" +
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
	"    busy\n" +
	"record line_key\n" +
	"    str invoice_id\n" +
	"    int line\n" +
	"action save_invoice\n" +
	"    post \"/invoices/save\"\n" +
	"    json invoice_wire limit 8192\n" +
	"    returns save_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        saved status 200\n" +
	"        rejected status 422\n" +
	"        stale status 409\n" +
	"        denied status 403\n" +
	"        busy status 503\n" +
	"record found\n" +
	"    str label\n" +
	"record missing\n" +
	"    str reason\n" +
	"record unavailable\n" +
	"    str reason\n" +
	"variant load_outcome\n" +
	"    found\n" +
	"    missing\n" +
	"    unavailable\n" +
	"action load_line\n" +
	"    get \"/invoices/:invoice_id/lines/:line\"\n" +
	"    captures line_key\n" +
	"    input none\n" +
	"    returns load_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        found status 200\n" +
	"        missing status 403\n" +
	"        unavailable status 503\n" +
	"fn load_outcome reload_line\n" +
	"    emits [http::transport_failed, http::invalid_request, http::status_error, codec::invalid_data]\n" +
	"    given\n" +
	"        str invoice_id\n" +
	"        int line\n" +
	"    asserts\n" +
	"        sample: \"inv-1\", 1 => ok found(\"inv-1\")\n" +
	"    match call http::fetch_json_get<load_outcome>(\"load_line\", invoice_id, line)\n" +
	"        http::transport_failed\n" +
	"        http::invalid_request\n" +
	"        http::status_error\n" +
	"        codec::invalid_data\n" +
	"        ok load_outcome got => ok got\n" +
	"fn save_outcome store_invoice\n" +
	"    emits [http::transport_failed, http::invalid_request, http::body_limit, http::status_error, codec::invalid_data]\n" +
	"    given\n" +
	"        invoice_wire body\n" +
	"    asserts\n" +
	"        sample: invoice_wire(\"inv-1\", 2) => ok saved(\"inv-1\")\n" +
	"    match call http::fetch_json_post<save_outcome, invoice_wire>(\"save_invoice\", body)\n" +
	"        http::transport_failed\n" +
	"        http::invalid_request\n" +
	"        http::body_limit\n" +
	"        http::status_error\n" +
	"        codec::invalid_data\n" +
	"        ok save_outcome done => ok done\n" +
	"fn void main\n" +
	"    emits []\n" +
	"    given\n" +
	"        str[] arguments\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    match call reload_line(\"inv-1\", 1)\n" +
	"        http::transport_failed => ok\n" +
	"        http::invalid_request => ok\n" +
	"        http::status_error => ok\n" +
	"        codec::invalid_data => ok\n" +
	"        ok load_outcome got => match call store_invoice(invoice_wire(\"inv-1\", 2))\n" +
	"            http::transport_failed => ok\n" +
	"            http::invalid_request => ok\n" +
	"            http::body_limit => ok\n" +
	"            http::status_error => ok\n" +
	"            codec::invalid_data => ok\n" +
	"            ok save_outcome done => ok\n"

func fetchEmitProgram(t *testing.T) *check.Program {
	t.Helper()
	return actionEmitProgram(t, map[string]string{"src/web/web.can": fetchActionEmitWeb})
}

func artifactByPath(t *testing.T, artifacts []ir.Artifact, path string) string {
	t.Helper()
	for _, artifact := range artifacts {
		if artifact.Path == path {
			return string(artifact.Bytes)
		}
	}
	t.Fatalf("emitted output omits %s", path)
	return ""
}

// actionTableLine extracts the frozen $canActions constant for
// cross-target comparison: one checked program must freeze the identical
// contract for both profiles.
func actionTableLine(t *testing.T, state string) string {
	t.Helper()
	start := strings.Index(state, "export const $canActions = ")
	if start < 0 {
		t.Fatal("state module carries no action table")
	}
	line := state[start:]
	end := strings.Index(line, ";\n")
	if end < 0 {
		t.Fatal("action table is not terminated")
	}
	return line[:end]
}

func TestFetchEmissionSplicesClientContracts(t *testing.T) {
	program := fetchEmitProgram(t)
	webID := program.Actions[0].Symbol.Package.ID
	joined := emittedBody(t, program)
	for _, want := range []string{
		`$canActionFetch.get($canExpr`,
		`$canActionFetch.post($canExpr`,
		`"action":"` + webID + `::load_line"`,
		`"method":"GET"`,
		`"path":"/invoices/:invoice_id/lines/:line"`,
		`"captures":[{"name":"invoice_id","type":"str"},{"name":"line","type":"int"}]`,
		`"action":"` + webID + `::save_invoice"`,
		`"method":"POST"`,
		`"path":"/invoices/save"`,
		`"request":{"root":`,
		`"response":{"root":`,
		`export let $canActionFetch:ReturnType<typeof $canCreateActionFetch>;`,
		`$canActionFetch=$canCreateActionFetch($canDomain,`,
		`createJsonActionFetch`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("emitted fetch omits %s", want)
		}
	}
	if strings.Count(joined, `"request":{"root":`) != 1 {
		t.Fatal("bodyless GET site gained a request contract")
	}
}

func TestFetchEmissionCrossTargetContract(t *testing.T) {
	program := fetchEmitProgram(t)
	dependencies := httpDependencies(t)
	bun, err := ProgramModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	generation, err := BrowserModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	bunState := artifactByPath(t, bun, programStatePath)
	browserState := artifactByPath(t, generation, programStatePath)
	if actionTableLine(t, bunState) != actionTableLine(t, browserState) {
		t.Fatal("browser and Bun profiles froze different action contracts")
	}
	for _, state := range []string{bunState, browserState} {
		if !strings.Contains(state, "$canActionFetch=$canCreateActionFetch($canDomain,") {
			t.Fatal("profile state omits the fetch adapter")
		}
	}
	webPath := program.Functions[0].Symbol.Source.OutputPath
	bunWeb := artifactByPath(t, bun, webPath)
	browserWeb := artifactByPath(t, generation, webPath)
	for _, want := range []string{
		`$canActionFetch.get(`,
		`$canActionFetch.post(`,
		`"/invoices/:invoice_id/lines/:line"`,
	} {
		if !strings.Contains(bunWeb, want) || !strings.Contains(browserWeb, want) {
			t.Fatalf("fetch lowering differs across profiles at %s", want)
		}
	}
	if !strings.Contains(browserWeb, "$canActionFetch") {
		t.Fatal("browser authored module omits the fetch adapter import")
	}
	var entry bool
	for _, artifact := range generation {
		if artifact.Path == browser.BrowserEntry {
			entry = true
		}
	}
	if !entry {
		t.Fatal("browser generation lost its distinct root")
	}
}

package emit

import (
	"strings"
	"testing"
)

// actionContractEmitWeb is a compact shared contract exercising every
// request line: a bodyless JSON GET, a JSON POST and an HTML POST with
// keyed rows. The frozen $canActions table below is the canonical
// metadata interface the mount, dispatch and audit consumers read.
const actionContractEmitWeb = "package contract\n" +
	"    provides [invoice_key, grid_edit_input, grid_loaded, grid_denied, grid_load_outcome, grid_saved, grid_failed, grid_edit_outcome, line_wire, invoice_form, saved, failed, edit_outcome, load_grid, save_grid, save_html]\n" +
	"    uses [form]\n" +
	"record invoice_key\n" +
	"    int tenant_id\n" +
	"    int invoice_id\n" +
	"record grid_edit_input\n" +
	"    str operation_id\n" +
	"    str revision\n" +
	"record grid_loaded\n" +
	"    str revision\n" +
	"record grid_denied\n" +
	"    str message\n" +
	"variant grid_load_outcome\n" +
	"    grid_loaded\n" +
	"    grid_denied\n" +
	"record grid_saved\n" +
	"    str operation_id\n" +
	"record grid_failed\n" +
	"    str operation_id\n" +
	"    str message\n" +
	"variant grid_edit_outcome\n" +
	"    grid_saved\n" +
	"    grid_failed\n" +
	"record line_wire\n" +
	"    str id\n" +
	"record invoice_form\n" +
	"    str seats\n" +
	"    form::rows<line_wire> lines\n" +
	"record saved\n" +
	"    str label\n" +
	"record failed\n" +
	"    str reason\n" +
	"variant edit_outcome\n" +
	"    saved\n" +
	"    failed\n" +
	"action load_grid\n" +
	"    get \"/api/tenants/:tenant_id/invoices/:invoice_id\"\n" +
	"    captures invoice_key\n" +
	"    input none\n" +
	"    returns grid_load_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        grid_loaded status 200\n" +
	"        grid_denied status 403\n" +
	"action save_grid\n" +
	"    post \"/api/tenants/:tenant_id/invoices/:invoice_id\"\n" +
	"    captures invoice_key\n" +
	"    json grid_edit_input limit 8192\n" +
	"    returns grid_edit_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        grid_saved status 200\n" +
	"        grid_failed status 422\n" +
	"action save_html\n" +
	"    post \"/tenants/:tenant_id/invoices/:invoice_id\"\n" +
	"    captures invoice_key\n" +
	"    form invoice_form limit 2048 rows_limit 64\n" +
	"    returns edit_outcome\n" +
	"    body html\n" +
	"    cases\n" +
	"        saved status 200 swap inner\n" +
	"        failed status 422 swap inner\n" +
	"fn void main\n" +
	"    emits []\n" +
	"    given\n" +
	"        str[] arguments\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    ok\n"

func TestActionContractEmissionFreezesMetadata(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/contract/contract.can": actionContractEmitWeb})
	if len(program.Actions) != 3 {
		t.Fatalf("checked %d actions, want 3", len(program.Actions))
	}
	contractID := program.Actions[0].Symbol.Package.ID
	joined := emittedBody(t, program)
	for _, want := range []string{
		`"identity":"` + contractID + `::load_grid"`,
		`"method":"GET"`,
		`"path":"/api/tenants/:tenant_id/invoices/:invoice_id"`,
		`"capturesType":"` + contractID + `::invoice_key"`,
		`"captures":[{"name":"tenant_id","type":"int"},{"name":"invoice_id","type":"int"}]`,
		`"input":{"mode":"none"}`,
		`"returns":"` + contractID + `::grid_load_outcome"`,
		`"identity":"` + contractID + `::save_grid"`,
		`"input":{"mode":"json","type":"` + contractID + `::grid_edit_input","limit":8192,"schema":{`,
		`"identity":"` + contractID + `::save_html"`,
		`"input":{"mode":"form","type":"` + contractID + `::invoice_form","limit":2048,"rowsLimit":64,"schema":{"root":`,
		`"returns":"` + contractID + `::edit_outcome"`,
		`"body":"html"`,
		`"cases":[{"leaf":"` + contractID + `::saved","status":200,"swap":"inner"},{"leaf":"` + contractID + `::failed","status":422,"swap":"inner"}]`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("emitted contract omits %s", want)
		}
	}
	if strings.Contains(joined, `"handler"`) {
		t.Fatal("emitted contract carries a handler binding")
	}
	if count := strings.Count(joined, `"responseSchema":`); count != 2 {
		t.Fatalf("emitted %d response schemas, want 2 (both JSON actions)", count)
	}
	if count := strings.Count(joined, `"body":"html"`); count != 1 {
		t.Fatalf("emitted %d html response modes, want 1", count)
	}
}

func TestSwapPolicyTableReachesHTMLFactory(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/contract/contract.can": actionContractEmitWeb})
	joined := emittedBody(t, program)
	want := `},[],[{"method":"POST","segments":["tenants","{}","invoices","{}"],"cases":[{"status":422,"swap":"inner"}]}]);`
	if !strings.Contains(joined, "$canHTML=$canCreateHTML($canDomain,") || !strings.Contains(joined, want) {
		t.Fatalf("HTML factory call lacks the swap table in %.600s", joined)
	}
	if strings.Count(joined, `"status":200,"swap":"inner"`) != 1 {
		t.Fatal("2xx swap case leaked into the renderer table or left $canActions")
	}
}

func TestSwapPolicyTableOmittedWithoutHTMLActions(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/web/web.can": actionEmitWeb})
	joined := emittedBody(t, program)
	if !strings.Contains(joined, "$canActions") {
		t.Fatal("JSON fixture emitted no action table")
	}
	for _, line := range strings.Split(joined, "\n") {
		if !strings.Contains(line, "$canHTML=$canCreateHTML(") {
			continue
		}
		if !strings.HasSuffix(strings.TrimSpace(line), "},[]);") {
			t.Fatalf("HTML factory call gained a swap table without HTML actions: %s", line)
		}
	}
}

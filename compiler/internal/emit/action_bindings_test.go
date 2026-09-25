package emit

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// actionBindingsEmitServer mounts the compact contract actions with
// request-first handlers and the distinct HTML renderers.
const actionBindingsEmitServer = "package server\n" +
	"    provides [routes]\n" +
	"    uses [contract, action, form, html, http, sql]\n" +
	"fn contract::grid_load_outcome load_grid\n" +
	"    emits []\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        http::request req\n" +
	"        contract::invoice_key key\n" +
	"    asserts\n" +
	"        sample: contract::invoice_key(1, 7) => ok contract::grid_denied(\"denied\")\n" +
	"    ok contract::grid_denied(\"denied\")\n" +
	"fn contract::grid_edit_outcome save_grid\n" +
	"    emits []\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        http::request req\n" +
	"        contract::invoice_key key\n" +
	"        contract::grid_edit_input body\n" +
	"    asserts\n" +
	"        sample: contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r1\") => ok contract::grid_failed(\"op-1\", \"no\")\n" +
	"    ok contract::grid_failed(body.operation_id, \"no\")\n" +
	"fn contract::edit_outcome save_html\n" +
	"    emits []\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        http::request req\n" +
	"        contract::invoice_key key\n" +
	"        contract::invoice_form form\n" +
	"    asserts\n" +
	"        sample: contract::invoice_key(1, 7), contract::invoice_form(\"2\", form::rows<contract::line_wire>([], [])) => ok contract::failed(\"no\")\n" +
	"    ok contract::failed(\"no\")\n" +
	"fn html::safe render_html\n" +
	"    emits []\n" +
	"    given\n" +
	"        contract::edit_outcome outcome\n" +
	"    asserts\n" +
	"        sample: contract::failed(\"no\") => ok\n" +
	"    match outcome\n" +
	"        contract::saved => ok call html::text_fragment(\"saved\")\n" +
	"        contract::failed => ok call html::text_fragment(\"failed\")\n" +
	"fn html::safe render_bad_form\n" +
	"    emits []\n" +
	"    given\n" +
	"        form::rejected<contract::invoice_form> bad\n" +
	"    asserts\n" +
	"        sample: form::rejected<contract::invoice_form>([], []) => ok\n" +
	"    ok call html::text_fragment(\"bad\")\n" +
	"fn http::router routes\n" +
	"    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"    asserts\n" +
	"        sample: => ok\n" +
	"    match chain\n" +
	"        call action::mount(contract::load_grid, callable load_grid) as http::route load\n" +
	"        call action::mount(contract::save_grid, callable save_grid) as http::route save\n" +
	"        call action::mount(contract::save_html, callable save_html, callable render_html, callable render_bad_form) as http::route form\n" +
	"        call http::make_router([load, save, form]) as http::router built\n" +
	"        http::invalid_route\n" +
	"        http::duplicate_route\n" +
	"        http::ambiguous_route\n" +
	"        ok => ok built\n"

// actionBindingsEmitClient exercises the symbol-based URL builder and
// the JSON GET/POST clients.
const actionBindingsEmitClient = "package client\n" +
	"    provides [load_url, fetch_load, fetch_save]\n" +
	"    uses [contract, action, codec, http]\n" +
	"fn str load_url\n" +
	"    emits [action::invalid_path]\n" +
	"    given\n" +
	"        contract::invoice_key key\n" +
	"    asserts\n" +
	"        sample: contract::invoice_key(1, 7) => ok \"/api/tenants/1/invoices/7\"\n" +
	"    match call action::url(contract::load_grid, key)\n" +
	"        action::invalid_path\n" +
	"        ok str built => ok built\n" +
	"fn contract::grid_load_outcome fetch_load\n" +
	"    emits [http::transport_failed, http::invalid_request, http::status_error, codec::invalid_data]\n" +
	"    given\n" +
	"        contract::invoice_key key\n" +
	"    asserts\n" +
	"        found: contract::invoice_key(1, 7) => ok contract::grid_loaded(\"r1\")\n" +
	"    match call action::request(contract::load_grid, key)\n" +
	"        when\n" +
	"            found: contract::invoice_key(1, 7) => ok contract::grid_loaded(\"r1\")\n" +
	"        http::transport_failed\n" +
	"        http::invalid_request\n" +
	"        http::status_error\n" +
	"        codec::invalid_data\n" +
	"        ok contract::grid_load_outcome got => ok got\n" +
	"fn contract::grid_edit_outcome fetch_save\n" +
	"    emits [http::transport_failed, http::invalid_request, http::body_limit, http::status_error, codec::invalid_data]\n" +
	"    given\n" +
	"        contract::invoice_key key\n" +
	"        contract::grid_edit_input body\n" +
	"    asserts\n" +
	"        saved: contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r1\") => ok contract::grid_saved(\"op-1\")\n" +
	"    match call action::post(contract::save_grid, key, body)\n" +
	"        http::transport_failed\n" +
	"        http::invalid_request\n" +
	"        http::body_limit\n" +
	"        http::status_error\n" +
	"        codec::invalid_data\n" +
	"        ok contract::grid_edit_outcome got => ok got\n"

func actionBindingsEmitProgram(t *testing.T, files map[string]string) *check.Program {
	t.Helper()
	base := map[string]string{"src/contract/contract.can": actionContractEmitWeb}
	for name, text := range files {
		base[name] = text
	}
	return actionEmitProgram(t, base)
}

// actionBindingsDependencies extends the walked runtime tree with the
// pending browser client adapter module. UP13 authors the real
// runtime/platform/action-client.ts; the linker only resolves module
// paths, so the stub keeps consumer emission testable meanwhile.
func actionBindingsDependencies(t *testing.T) []ir.Artifact {
	t.Helper()
	return append(httpDependencies(t), ir.Artifact{Path: "runtime/platform/action-client.ts"})
}

func actionBindingsBody(t *testing.T, program *check.Program) string {
	t.Helper()
	artifacts, err := ProgramModules(program, "runtime", actionBindingsDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		bodies = append(bodies, string(artifact.Bytes))
	}
	return strings.Join(bodies, "\n")
}

func TestActionBindingsEmission(t *testing.T) {
	program := actionBindingsEmitProgram(t, map[string]string{
		"src/server/server.can": actionBindingsEmitServer,
		"src/client/client.can": actionBindingsEmitClient,
	})
	if !program.ActionRoutes || !program.ActionClient {
		t.Fatalf("emit state use = routes %v client %v, want both", program.ActionRoutes, program.ActionClient)
	}
	contractID := program.Actions[0].Symbol.Package.ID
	joined := actionBindingsBody(t, program)
	for _, want := range []string{
		// Mount targets: one-callable JSON mounts plus the
		// three-callable HTML form mount.
		`$canActionRoutes.mount(`,
		`$canActionRoutes.mountForm(`,
		`$canActionRoutes.url(`,
		`$canActionClient.request(`,
		`$canActionClient.post(`,
		// JSON mount metadata: route, captures, request and
		// response codecs, finite cases.
		`"action":"` + contractID + `::save_grid","method":"POST","path":"/api/tenants/:tenant_id/invoices/:invoice_id"`,
		`"captures":[{"name":"tenant_id","type":"int"},{"name":"invoice_id","type":"int"}]`,
		`"input":{"mode":"json","type":"` + contractID + `::grid_edit_input","limit":8192,"schema":{"root":`,
		`"returns":"` + contractID + `::grid_edit_outcome","body":"json"`,
		`"responseSchema":{"root":`,
		// HTML mount metadata: form codec, row bound and the
		// per-case guard policy with structural identities.
		`"action":"` + contractID + `::save_html","method":"POST","path":"/tenants/:tenant_id/invoices/:invoice_id"`,
		`"input":{"mode":"form","type":"` + contractID + `::invoice_form","limit":2048,"rowsLimit":64,"schema":{"root":`,
		`"returns":"` + contractID + `::edit_outcome","body":"html"`,
		`"status":200,"swap":"inner"}`,
		`"rejected":"`,
		// URL builder metadata: template plus ordered captures.
		`"action":"` + contractID + `::load_grid","path":"/api/tenants/:tenant_id/invoices/:invoice_id","captures":[{"name":"tenant_id","type":"int"}`,
		// Client metadata reuses the JSON fetch projection.
		`"method":"GET","path":"/api/tenants/:tenant_id/invoices/:invoice_id"`,
		// Selected adapter factories and their failure identities.
		`createActionRoutes`,
		`platform/action-routes.ts`,
		`createActionClient`,
		`platform/action-client.ts`,
		`export let $canActionRoutes: ReturnType<typeof $canCreateActionRoutes>;`,
		`export let $canActionClient: ReturnType<typeof $canCreateActionClient>;`,
		`$canActionRoutes=$canCreateActionRoutes($canDomain,{invalidRoute:"`,
		`$canActionClient=$canCreateActionClient($canDomain,{transport:"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("emitted action bindings omit %s", want)
		}
	}
	if count := strings.Count(joined, `$canActionRoutes.mount(`); count != 2 {
		t.Fatalf("emitted %d JSON mounts, want 2", count)
	}
	if count := strings.Count(joined, `$canActionRoutes.mountForm(`); count != 1 {
		t.Fatalf("emitted %d HTML mounts, want 1", count)
	}
	// The POST request schema appears once under the client spelling:
	// only the post site carries one, while mounts nest their codecs
	// under the shared input spelling asserted above.
	if count := strings.Count(joined, `"request":{"root":`); count != 1 {
		t.Fatalf("emitted %d client request codecs, want 1 (post site)", count)
	}
	for _, empty := range []string{`invalidRoute:""`, `invalidPath:""`, `transport:""`, `invalidRequest:""`, `bodyLimit:""`, `statusError:""`, `invalidData:""`} {
		if strings.Contains(joined, empty) {
			t.Fatalf("action adapter initialized with empty identity %s", empty)
		}
	}
	if strings.Contains(joined, `"swap":"inner"}]`) && strings.Count(joined, `"swap":"inner"`) < 2 {
		t.Fatal("HTML mount lost a per-case guard swap")
	}
}

func TestActionBindingsBrowserClientProjection(t *testing.T) {
	contractWithoutMain := strings.Replace(actionContractEmitWeb, "fn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n", "", 1)
	if contractWithoutMain == actionContractEmitWeb {
		t.Fatal("contract fixture main not found for client-projection pruning fix")
	}
	clientWithMain := actionBindingsEmitClient + "fn void main\n" +
		"    emits []\n" +
		"    given\n" +
		"        str[] arguments\n" +
		"    asserts\n" +
		"        empty: [] => ok\n" +
		"    match call load_url(contract::invoice_key(1, 7))\n" +
		"        action::invalid_path => ok\n" +
		"        ok str built => match call fetch_load(contract::invoice_key(1, 7))\n" +
		"            http::transport_failed => ok\n" +
		"            http::invalid_request => ok\n" +
		"            http::status_error => ok\n" +
		"            codec::invalid_data => ok\n" +
		"            ok contract::grid_load_outcome got => match call fetch_save(contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r1\"))\n" +
		"                http::transport_failed => ok\n" +
		"                http::invalid_request => ok\n" +
		"                http::body_limit => ok\n" +
		"                http::status_error => ok\n" +
		"                codec::invalid_data => ok\n" +
		"                ok contract::grid_edit_outcome done => ok\n"
	program := actionBindingsEmitProgram(t, map[string]string{
		"src/contract/contract.can": contractWithoutMain,
		"src/client/client.can":     clientWithMain,
	})
	contractID := program.Actions[0].Symbol.Package.ID
	artifacts, err := BrowserModules(program, "runtime", actionBindingsDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		bodies = append(bodies, string(artifact.Bytes))
	}
	joined := strings.Join(bodies, "\n")
	for _, want := range []string{
		`$canActionRoutes.url(`,
		`$canActionClient.request(`,
		`$canActionClient.post(`,
		`"action":"` + contractID + `::load_grid"`,
		`export let $canActionClient: ReturnType<typeof $canCreateActionClient>;`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("browser client projection omits %s", want)
		}
	}
	// No mount call or mount-only metadata enters the client-only
	// program. (The shared declaration table still carries its own
	// per-case rows in both profiles; UP11 prunes unreached rows.)
	for _, denied := range []string{`$canActionRoutes.mount(`, `$canActionRoutes.mountForm(`, `"rejected":"`, `"rawEntry":"`} {
		if strings.Contains(joined, denied) {
			t.Fatalf("browser client projection leaked %s", denied)
		}
	}
}

func TestActionBindingsStateIsSelected(t *testing.T) {
	program := actionEmitProgram(t, map[string]string{"src/contract/contract.can": actionContractEmitWeb})
	if program.ActionRoutes || program.ActionClient {
		t.Fatal("declaration-only program selected action state")
	}
	joined := emittedBody(t, program)
	for _, selected := range []string{"$canActionRoutes", "$canActionClient", "createActionRoutes", "createActionClient", "action-client.ts"} {
		if strings.Contains(joined, selected) {
			t.Fatalf("declaration-only emission references %s", selected)
		}
	}
	var _ = ir.ActionSite{}
}

package check

import (
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// actionBindingsModel is the server helper package the UP08 mount
// fixtures bind: header scanning, exact-origin admission and the three
// named protected operations with their exact signatures and finite
// leaf mapping. Header names use the lowercase request-snapshot
// spelling throughout these fixtures.
const actionBindingsModel = "package model\n" +
	"    provides [session_of, exact_origin, load_authorized, save_authorized, save_html_authorized]\n" +
	"    uses [invoice_contract as contract, cookie, form, http, option, sql]\n" +
	"fn option::value<str> find_header\n" +
	"    emits []\n" +
	"    given\n" +
	"        http::header[] headers\n" +
	"        str name\n" +
	"    asserts\n" +
	"        hit: [http::header(\"x-t\", \"1\"), http::header(\"cookie\", \"session=A\")], \"cookie\" => ok option::some(\"session=A\")\n" +
	"        miss: [http::header(\"x-t\", \"1\")], \"cookie\" => ok option::none()\n" +
	"        empty: [], \"cookie\" => ok option::none()\n" +
	"    match headers.length is 0\n" +
	"        false => match headers[0].name is name\n" +
	"            false => match call find_header(call headers.slice(...[1, 1000]), name)\n" +
	"                ok option::value<str> rest => ok rest\n" +
	"            true => ok option::some(headers[0].value)\n" +
	"        true => ok option::none()\n" +
	"fn option::value<str> unique_scan\n" +
	"    emits []\n" +
	"    given\n" +
	"        http::header[] headers\n" +
	"        str name\n" +
	"        option::value<str> found\n" +
	"    asserts\n" +
	"        single: [http::header(\"origin\", \"https://shop.example\")], \"origin\", option::none() => ok option::some(\"https://shop.example\")\n" +
	"        double: [http::header(\"origin\", \"https://a.example\"), http::header(\"origin\", \"https://b.example\")], \"origin\", option::none() => ok option::none()\n" +
	"        empty: [], \"origin\", option::none() => ok option::none()\n" +
	"    match headers.length is 0\n" +
	"        false => match headers[0].name is name\n" +
	"            false => match call unique_scan(call headers.slice(...[1, 1000]), name, found)\n" +
	"                ok option::value<str> rest => ok rest\n" +
	"            true => match found\n" +
	"                option::none => match call unique_scan(call headers.slice(...[1, 1000]), name, option::some(headers[0].value))\n" +
	"                    ok option::value<str> rest => ok rest\n" +
	"                option::some => ok option::none()\n" +
	"        true => ok found\n" +
	"fn option::value<str> unique_header\n" +
	"    emits []\n" +
	"    given\n" +
	"        http::header[] headers\n" +
	"        str name\n" +
	"    asserts\n" +
	"        single: [http::header(\"origin\", \"https://shop.example\")], \"origin\" => ok option::some(\"https://shop.example\")\n" +
	"        double: [http::header(\"origin\", \"https://a.example\"), http::header(\"origin\", \"https://b.example\")], \"origin\" => ok option::none()\n" +
	"    match call unique_scan(headers, name, option::none())\n" +
	"        ok option::value<str> found => ok found\n" +
	"fn option::value<str> session_of\n" +
	"    emits []\n" +
	"    given\n" +
	"        http::header[] headers\n" +
	"    asserts\n" +
	"        present: [http::header(\"cookie\", \"theme=dark; session=A\")] => ok option::some(\"A\")\n" +
	"        absent: [http::header(\"cookie\", \"theme=dark\")] => ok option::none()\n" +
	"        no_cookie: [http::header(\"x-t\", \"1\")] => ok option::none()\n" +
	"    match call find_header(headers, \"cookie\")\n" +
	"        ok option::value<str> found => match found\n" +
	"            option::none => ok option::none()\n" +
	"            option::some => match call cookie::parse(found.value)\n" +
	"                ok cookie::collection jar => match call cookie::get(jar, \"session\")\n" +
	"                    ok option::value<str> id => ok id\n" +
	"fn bool exact_origin\n" +
	"    emits []\n" +
	"    given\n" +
	"        http::header[] headers\n" +
	"        str public_origin\n" +
	"    asserts\n" +
	"        exact: [http::header(\"origin\", \"https://shop.example\")], \"https://shop.example\" => ok true\n" +
	"        foreign: [http::header(\"origin\", \"https://evil.example\")], \"https://shop.example\" => ok false\n" +
	"        missing: [http::header(\"x-t\", \"1\")], \"https://shop.example\" => ok false\n" +
	"    match call unique_header(headers, \"origin\")\n" +
	"        ok option::value<str> found => match found\n" +
	"            option::none => ok false\n" +
	"            option::some => ok found.value is public_origin\n" +
	"fn contract::grid_load_outcome load_authorized\n" +
	"    emits []\n" +
	"    given\n" +
	"        sql::pool pool\n" +
	"        str session\n" +
	"        contract::invoice_key key\n" +
	"    asserts\n" +
	"        denied: \"\", contract::invoice_key(1, 7) => ok contract::grid_load_forbidden(\"denied\")\n" +
	"        found: \"s1\", contract::invoice_key(1, 7) => ok contract::grid_loaded(contract::grid_snapshot(\"r1\", [], 0))\n" +
	"    match session is \"\"\n" +
	"        false => ok contract::grid_loaded(contract::grid_snapshot(\"r1\", [], 0))\n" +
	"        true => ok contract::grid_load_forbidden(\"denied\")\n" +
	"fn contract::grid_edit_outcome save_authorized\n" +
	"    emits []\n" +
	"    given\n" +
	"        sql::pool pool\n" +
	"        str session\n" +
	"        contract::invoice_key key\n" +
	"        contract::grid_edit_input body\n" +
	"    asserts\n" +
	"        denied: \"\", contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r1\", []) => ok contract::grid_forbidden(\"op-1\", \"denied\")\n" +
	"        stale: \"s1\", contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r0\", []) => ok contract::grid_conflict(\"op-1\", contract::grid_edit_input(\"op-1\", \"r0\", []), \"stale\")\n" +
	"        saved: \"s1\", contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r1\", []) => ok contract::grid_saved(\"op-1\", contract::grid_snapshot(\"r2\", [], 0))\n" +
	"    match session is \"\"\n" +
	"        false => match body.revision is \"r1\"\n" +
	"            false => ok contract::grid_conflict(body.operation_id, body, \"stale\")\n" +
	"            true => ok contract::grid_saved(body.operation_id, contract::grid_snapshot(\"r2\", body.lines, 0))\n" +
	"        true => ok contract::grid_forbidden(body.operation_id, \"denied\")\n" +
	"fn contract::edit_outcome save_html_authorized\n" +
	"    emits []\n" +
	"    given\n" +
	"        sql::pool pool\n" +
	"        str session\n" +
	"        contract::invoice_key key\n" +
	"        contract::invoice_form form\n" +
	"    asserts\n" +
	"        denied: \"\", contract::invoice_key(1, 7), contract::invoice_form(\"2\", option::none(), \"r1\", form::rows<contract::line_wire>([], [])) => ok contract::forbidden(\"denied\")\n" +
	"        stale: \"s1\", contract::invoice_key(1, 7), contract::invoice_form(\"2\", option::none(), \"r0\", form::rows<contract::line_wire>([], [])) => ok contract::conflict(contract::invoice_form(\"2\", option::none(), \"r0\", form::rows<contract::line_wire>([], [])), \"stale\")\n" +
	"        saved: \"s1\", contract::invoice_key(1, 7), contract::invoice_form(\"2\", option::none(), \"r1\", form::rows<contract::line_wire>([], [])) => ok contract::saved(contract::invoice_form(\"2\", option::none(), \"r1\", form::rows<contract::line_wire>([], [])))\n" +
	"    match session is \"\"\n" +
	"        false => match form.revision is \"r1\"\n" +
	"            false => ok contract::conflict(form, \"stale\")\n" +
	"            true => ok contract::saved(form)\n" +
	"        true => ok contract::forbidden(\"denied\")\n"

// actionBindingsServer mounts the three selected contract actions with
// request-first handlers capturing the startup pool and public origin,
// plus the distinct normal and structural HTML renderers.
const actionBindingsServer = "package server\n" +
	"    provides [routes]\n" +
	"    uses [invoice_contract as contract, action, form, html, http, option, sql, model]\n" +
	"fn contract::grid_load_outcome load_grid\n" +
	"    emits []\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        http::request req\n" +
	"        contract::invoice_key key\n" +
	"    asserts\n" +
	"        no_session: contract::invoice_key(1, 7) => ok contract::grid_load_forbidden(\"denied\")\n" +
	"    match call http::request_headers(req)\n" +
	"        when\n" +
	"            no_session: req => ok []\n" +
	"        ok http::header[] headers => match call model::session_of(headers)\n" +
	"            ok option::value<str> session => match session\n" +
	"                option::none => ok contract::grid_load_forbidden(\"denied\")\n" +
	"                option::some => match call model::load_authorized(pool, session.value, key)\n" +
	"                    ok contract::grid_load_outcome found => ok found\n" +
	"fn contract::grid_edit_outcome save_grid\n" +
	"    emits []\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        near str public_origin\n" +
	"        http::request req\n" +
	"        contract::invoice_key key\n" +
	"        contract::grid_edit_input body\n" +
	"    asserts\n" +
	"        no_session: \"https://shop.example\", contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r1\", []) => ok contract::grid_forbidden(\"op-1\", \"denied\")\n" +
	"    match call http::request_headers(req)\n" +
	"        when\n" +
	"            no_session: req => ok []\n" +
	"        ok http::header[] headers => match call model::session_of(headers)\n" +
	"            ok option::value<str> session => match session\n" +
	"                option::none => ok contract::grid_forbidden(body.operation_id, \"denied\")\n" +
	"                option::some => match call model::exact_origin(headers, public_origin)\n" +
	"                    ok bool admitted => match admitted\n" +
	"                        false => ok contract::grid_forbidden(body.operation_id, \"denied\")\n" +
	"                        true => match call model::save_authorized(pool, session.value, key, body)\n" +
	"                            ok contract::grid_edit_outcome result => ok result\n" +
	"fn contract::edit_outcome save_html\n" +
	"    emits []\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        near str public_origin\n" +
	"        http::request req\n" +
	"        contract::invoice_key key\n" +
	"        contract::invoice_form form\n" +
	"    asserts\n" +
	"        denied: \"https://shop.example\", contract::invoice_key(1, 7), contract::invoice_form(\"2\", option::none(), \"r1\", form::rows<contract::line_wire>([], [])) => ok contract::forbidden(\"denied\")\n" +
	"    match call http::request_headers(req)\n" +
	"        when\n" +
	"            denied: req => ok []\n" +
	"        ok http::header[] headers => match call model::session_of(headers)\n" +
	"            ok option::value<str> session => match session\n" +
	"                option::none => ok contract::forbidden(\"denied\")\n" +
	"                option::some => match call model::exact_origin(headers, public_origin)\n" +
	"                    ok bool admitted => match admitted\n" +
	"                        false => ok contract::forbidden(\"denied\")\n" +
	"                        true => match call model::save_html_authorized(pool, session.value, key, form)\n" +
	"                            ok contract::edit_outcome result => ok result\n" +
	"fn html::safe render_html\n" +
	"    emits []\n" +
	"    given\n" +
	"        contract::edit_outcome outcome\n" +
	"    asserts\n" +
	"        denied: contract::forbidden(\"denied\") => ok\n" +
	"    match outcome\n" +
	"        contract::saved => ok call html::text_fragment(\"saved\")\n" +
	"        contract::invalid => ok call html::text_fragment(\"invalid\")\n" +
	"        contract::conflict => ok call html::text_fragment(\"conflict\")\n" +
	"        contract::forbidden => ok call html::text_fragment(\"forbidden\")\n" +
	"        contract::unavailable => ok call html::text_fragment(\"unavailable\")\n" +
	"fn html::safe render_bad_form\n" +
	"    emits []\n" +
	"    given\n" +
	"        form::rejected<contract::invoice_form> bad\n" +
	"    asserts\n" +
	"        sample: form::rejected<contract::invoice_form>([], []) => ok\n" +
	"    ok call html::text_fragment(\"bad form\")\n" +
	"fn http::router routes\n" +
	"    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        near str public_origin\n" +
	"    asserts\n" +
	"        sample: \"https://shop.example\" => ok\n" +
	"    match chain\n" +
	"        call action::mount(contract::load_invoice_grid, callable load_grid) as http::route load\n" +
	"        call action::mount(contract::save_invoice_grid, callable save_grid) as http::route save\n" +
	"        call action::mount(contract::save_invoice_html, callable save_html, callable render_html, callable render_bad_form) as http::route form\n" +
	"        call http::make_router([load, save, form]) as http::router built\n" +
	"        http::invalid_route\n" +
	"        http::duplicate_route\n" +
	"        http::ambiguous_route\n" +
	"        ok => ok built\n" +
	"fn str quantity_input\n" +
	"    emits [form::unknown_field, form::invalid_name]\n" +
	"    asserts\n" +
	"        sample: => ok \"lines[a1][quantity]\"\n" +
	"    match chain\n" +
	"        call form::named_collection<contract::invoice_form>(\"lines\") as form::collection coll\n" +
	"        call form::named_field<contract::line_wire>(\"quantity\") as form::field field\n" +
	"        call form::row_key(\"a1\") as form::row row\n" +
	"        form::unknown_field\n" +
	"        form::invalid_name\n" +
	"        ok => ok call form::input_name(coll, row, field)\n"

// actionBindingsClient exercises the symbol-based client constructors:
// the canonical URL builder plus the bodyless GET and wire-body POST
// JSON clients.
const actionBindingsClient = "package client\n" +
	"    provides [load_url, form_url, fetch_load, fetch_save]\n" +
	"    uses [invoice_contract as contract, action, codec, http]\n" +
	"fn str load_url\n" +
	"    emits [action::invalid_path]\n" +
	"    given\n" +
	"        contract::invoice_key key\n" +
	"    asserts\n" +
	"        sample: contract::invoice_key(1, 7) => ok \"/api/tenants/1/invoices/7\"\n" +
	"    match call action::url(contract::load_invoice_grid, key)\n" +
	"        action::invalid_path\n" +
	"        ok str built => ok built\n" +
	"fn str form_url\n" +
	"    emits [action::invalid_path]\n" +
	"    given\n" +
	"        contract::invoice_key key\n" +
	"    asserts\n" +
	"        sample: contract::invoice_key(1, 7) => ok \"/tenants/1/invoices/7\"\n" +
	"    match call action::url(contract::save_invoice_html, key)\n" +
	"        action::invalid_path\n" +
	"        ok str built => ok built\n" +
	"fn contract::grid_load_outcome fetch_load\n" +
	"    emits [http::transport_failed, http::invalid_request, http::status_error, codec::invalid_data]\n" +
	"    given\n" +
	"        contract::invoice_key key\n" +
	"    asserts\n" +
	"        found: contract::invoice_key(1, 7) => ok contract::grid_loaded(contract::grid_snapshot(\"r1\", [], 0))\n" +
	"        denied: contract::invoice_key(1, 9) => ok contract::grid_load_forbidden(\"denied\")\n" +
	"    match call action::request(contract::load_invoice_grid, key)\n" +
	"        when\n" +
	"            found: contract::invoice_key(1, 7) => ok contract::grid_loaded(contract::grid_snapshot(\"r1\", [], 0))\n" +
	"            denied: contract::invoice_key(1, 9) => ok contract::grid_load_forbidden(\"denied\")\n" +
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
	"        saved: contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r1\", []) => ok contract::grid_saved(\"op-1\", contract::grid_snapshot(\"r2\", [], 0))\n" +
	"    match call action::post(contract::save_invoice_grid, key, body)\n" +
	"        when\n" +
	"            saved: contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r1\", []) => ok contract::grid_saved(\"op-1\", contract::grid_snapshot(\"r2\", [], 0))\n" +
	"        http::transport_failed\n" +
	"        http::invalid_request\n" +
	"        http::body_limit\n" +
	"        http::status_error\n" +
	"        codec::invalid_data\n" +
	"        ok contract::grid_edit_outcome got => ok got\n"

// actionBindingsProgram checks the complete consumer program: the shared
// contract, the model helpers, the mounted server and the JSON clients.
func actionBindingsProgram(t *testing.T, files map[string]string) *Program {
	t.Helper()
	base := map[string]string{
		"src/contract/contract.can": actionContractSource,
		"src/model/model.can":       actionBindingsModel,
		"src/server/server.can":     actionBindingsServer,
		"src/client/client.can":     actionBindingsClient,
		"src/app/main.can":          "package app\n    provides []\n    uses []\n" + actionMain,
	}
	for name, text := range files {
		base[name] = text
	}
	program, err := programFixture(t, base)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestActionBindingsMountAndClient(t *testing.T) {
	program := actionBindingsProgram(t, nil)
	if len(program.Actions) != 3 {
		t.Fatalf("checked %d actions, want 3", len(program.Actions))
	}
	if !program.ActionRoutes || !program.ActionClient {
		t.Fatalf("action state use = routes %v client %v, want both", program.ActionRoutes, program.ActionClient)
	}
	contractID := program.Actions[0].Symbol.Package.ID
	steps := actionBindingSteps(t, program)
	sites := map[string][]ir.ActionSite{}
	for _, step := range steps {
		if step.Action == nil {
			continue
		}
		sites[step.Identity] = append(sites[step.Identity], *step.Action)
	}
	if len(sites[actionMount]) != 3 || len(sites[actionURL]) != 2 || len(sites[actionRequest]) != 1 || len(sites[actionPost]) != 1 {
		t.Fatalf("action sites = mount %d url %d request %d post %d", len(sites[actionMount]), len(sites[actionURL]), len(sites[actionRequest]), len(sites[actionPost]))
	}
	load := actionSiteByAction(t, sites[actionMount], contractID+"::load_invoice_grid")
	if load.Method != "GET" || load.Path != "/api/tenants/:tenant_id/invoices/:invoice_id" || load.Body != "json" {
		t.Fatalf("load mount site = %+v", load)
	}
	if load.Response == nil || load.Form != nil {
		t.Fatal("load mount site lost its JSON response codec")
	}
	save := actionSiteByAction(t, sites[actionMount], contractID+"::save_invoice_grid")
	if save.Method != "POST" || save.InputMode != "json" || save.Limit != 8192 || save.Response == nil {
		t.Fatalf("save mount site = %+v", save)
	}
	if save.Request == nil || load.Request != nil {
		t.Fatal("mount sites lost their request codec projection")
	}
	html := actionSiteByAction(t, sites[actionMount], contractID+"::save_invoice_html")
	if html.Method != "POST" || html.InputMode != "form" || html.Limit != 2048 || html.RowsLimit != 64 {
		t.Fatalf("HTML mount site = %+v", html)
	}
	if html.Form == nil || html.Response != nil {
		t.Fatal("HTML mount site lost its form schema")
	}
	if html.Rejected == "" || html.RawEntry == "" || html.Issue == "" {
		t.Fatal("HTML mount site lost its structural-422 identities")
	}
	if len(html.Cases) != 5 {
		t.Fatalf("HTML mount cases = %+v", html.Cases)
	}
	leaves := map[string]bool{}
	for _, kase := range html.Cases {
		if kase.Swap != "inner" || kase.Status < 200 || kase.Leaf == "" || leaves[kase.Leaf] {
			t.Fatalf("HTML guard case = %+v", kase)
		}
		leaves[kase.Leaf] = true
	}
	for _, site := range sites[actionMount] {
		if site.CapturesType == "" || len(site.Captures) != 2 || site.Captures[0].Name != "tenant_id" || site.Captures[1].Name != "invoice_id" {
			t.Fatalf("mount site captures = %+v", site)
		}
	}
	// The static symbol never lowers to a value argument: every step
	// carries exactly its value operands plus the frozen site.
	for _, step := range steps {
		if step.Action == nil {
			continue
		}
		if len(step.Arguments) != len(step.Contract.Inputs()) {
			t.Fatalf("%s step lowers %d values for %d contract inputs", step.Identity, len(step.Arguments), len(step.Contract.Inputs()))
		}
	}
	url := sites[actionURL][0]
	formURL := sites[actionURL][1]
	if formURL.Action == contractID+"::load_invoice_grid" {
		url, formURL = formURL, url
	}
	if url.Action != contractID+"::load_invoice_grid" || url.Method != "GET" || url.Response != nil || url.Form != nil {
		t.Fatalf("url site = %+v", url)
	}
	if formURL.Action != contractID+"::save_invoice_html" || formURL.Method != "POST" || len(formURL.Captures) != 2 {
		t.Fatalf("HTML url site = %+v", formURL)
	}
	request := sites[actionRequest][0]
	if request.Action != contractID+"::load_invoice_grid" || request.Request != nil || request.Response == nil || len(request.Cases) != 3 {
		t.Fatalf("request site = %+v", request)
	}
	post := sites[actionPost][0]
	if post.Action != contractID+"::save_invoice_grid" || post.Request == nil || post.Response == nil || len(post.Cases) != 5 {
		t.Fatalf("post site = %+v", post)
	}
	for _, site := range []ir.ActionSite{url, formURL, request, post} {
		if site.Rejected != "" || site.Form != nil {
			t.Fatalf("client site leaked server bindings: %+v", site)
		}
	}
}

// actionSiteByAction selects one mount site by its action identity.
func actionSiteByAction(t *testing.T, sites []ir.ActionSite, action string) ir.ActionSite {
	t.Helper()
	for _, site := range sites {
		if site.Action == action {
			return site
		}
	}
	t.Fatalf("no mount site for %s", action)
	return ir.ActionSite{}
}

// actionBindingSteps collects every checked invocation step in the
// program's function regions.
func actionBindingSteps(t *testing.T, program *Program) []ir.InvocationStep {
	t.Helper()
	var steps []ir.InvocationStep
	var expression func(*ir.Expression)
	var invocation func(*ir.Invocation)
	var completion func(*ir.Completion)
	var block func(*ir.Block)
	expression = func(value *ir.Expression) {
		if value == nil {
			return
		}
		if value.Invocation != nil {
			invocation(value.Invocation)
		}
		for _, input := range value.Inputs {
			expression(input)
		}
	}
	invocation = func(call *ir.Invocation) {
		if call == nil {
			return
		}
		steps = append(steps, call.Steps...)
	}
	completion = func(done *ir.Completion) {
		if done == nil {
			return
		}
		invocation(done.Call)
		expression(done.Value)
		block(done.Block)
		if done.Match != nil {
			invocation(done.Match.Call)
			for i := range done.Match.Arms {
				completion(done.Match.Arms[i].Body)
				expression(done.Match.Arms[i].Value)
			}
		}
	}
	block = func(body *ir.Block) {
		if body == nil {
			return
		}
		for i := range body.Steps {
			invocation(body.Steps[i].Call)
			expression(body.Steps[i].Value)
		}
		completion(body.Terminal)
	}
	for _, fn := range program.Functions {
		if fn.Region != nil {
			block(fn.Region.Body)
		}
	}
	if len(steps) == 0 {
		t.Fatal("checked program holds no invocation steps")
	}
	return steps
}

func TestActionBindingsDerivedContracts(t *testing.T) {
	program := actionBindingsProgram(t, nil)
	contractID := program.Actions[0].Symbol.Package.ID
	checked := map[string]bool{}
	for _, step := range actionBindingSteps(t, program) {
		if step.Action == nil {
			continue
		}
		contract := step.Contract
		if contract == nil || contract.Kind() != types.Callable {
			t.Fatalf("%s step lost its derived contract", step.Identity)
		}
		switch step.Identity {
		case actionMount:
			if contract.Result().Declaration() != "can.std.http@1::route" || len(contract.Errors()) != 1 || contract.Errors()[0].Declaration() != "can.std.http@1::invalid_route" {
				t.Fatalf("mount contract = %s emits %v", types.CanonicalName(contract.Result()), contract.Errors())
			}
			want := 1
			if strings.HasSuffix(step.Action.Action, "::save_invoice_html") {
				want = 3
			}
			if len(contract.Inputs()) != want || len(step.Arguments) != want {
				t.Fatalf("mount contract takes %d inputs with %d values, want %d", len(contract.Inputs()), len(step.Arguments), want)
			}
			for _, input := range contract.Inputs() {
				if input.Kind() != types.Callable || len(input.Errors()) != 0 {
					t.Fatalf("mount callable input = %v", input)
				}
			}
		case actionURL:
			if contract.Result().Declaration() != "str" || len(contract.Errors()) != 1 || contract.Errors()[0].Declaration() != "can.std.action@1::invalid_path" {
				t.Fatalf("url contract = %s emits %v", types.CanonicalName(contract.Result()), contract.Errors())
			}
			if len(contract.Inputs()) != 1 || contract.Inputs()[0].Declaration() != contractID+"::invoice_key" {
				t.Fatalf("url contract inputs = %v", contract.Inputs())
			}
		case actionRequest:
			if contract.Result().Declaration() != contractID+"::grid_load_outcome" || len(contract.Errors()) != 4 {
				t.Fatalf("request contract = %s emits %d", types.CanonicalName(contract.Result()), len(contract.Errors()))
			}
		case actionPost:
			if contract.Result().Declaration() != contractID+"::grid_edit_outcome" || len(contract.Errors()) != 5 {
				t.Fatalf("post contract = %s emits %d", types.CanonicalName(contract.Result()), len(contract.Errors()))
			}
			if len(contract.Inputs()) != 2 || contract.Inputs()[1].Declaration() != contractID+"::grid_edit_input" {
				t.Fatalf("post contract inputs = %v", contract.Inputs())
			}
		}
		checked[step.Identity] = true
	}
	for _, identity := range []string{actionMount, actionURL, actionRequest, actionPost} {
		if !checked[identity] {
			t.Fatalf("no derived contract for %s", identity)
		}
	}
}

// actionBindingsExtra mounts one more handler shape: the snippet is
// spliced ahead of routes and the load mount is retargeted at it, so a
// handler-shape diagnostic points at the mount instead of a broken body.
func actionBindingsExtra(snippet, target string) map[string][][2]string {
	return map[string][][2]string{
		"src/server/server.can": {
			{"fn http::router routes\n", snippet + "fn http::router routes\n"},
			{"action::mount(contract::load_invoice_grid, callable load_grid)", "action::mount(contract::load_invoice_grid, callable " + target + ")"},
		},
	}
}

func TestActionBindingsRejects(t *testing.T) {
	server := "src/server/server.can"
	client := "src/client/client.can"
	contract := "src/contract/contract.can"
	loadMount := "action::mount(contract::load_invoice_grid, callable load_grid)"
	for name, tc := range map[string]struct {
		mutate map[string][][2]string
		want   string
	}{
		"string mount name": {
			map[string][][2]string{server: {{loadMount, `action::mount("load_invoice_grid", callable load_grid)`}}},
			"requires an action symbol as its first operand, not a string or computed name",
		},
		"computed request name": {
			map[string][][2]string{client: {{"action::request(contract::load_invoice_grid, key)", "action::request(key, key)"}}},
			`cannot resolve action "key"`,
		},
		"unknown mount action": {
			map[string][][2]string{server: {{loadMount, "action::mount(contract::missing_grid, callable load_grid)"}}},
			`cannot resolve action "contract::missing_grid"`,
		},
		"renamed action": {
			map[string][][2]string{contract: {
				{"load_invoice_grid, save_invoice_grid, save_invoice_html]", "load_grid_v2, save_invoice_grid, save_invoice_html]"},
				{"action load_invoice_grid\n", "action load_grid_v2\n"},
			}},
			`cannot resolve action "contract::load_invoice_grid"`,
		},
		"missing request input": {
			actionBindingsExtra("fn contract::grid_load_outcome load_noreq\n"+
				"    emits []\n"+
				"    given\n"+
				"        near sql::pool pool\n"+
				"        contract::invoice_key key\n"+
				"    asserts\n"+
				"        sample: contract::invoice_key(1, 7) => ok contract::grid_load_forbidden(\"denied\")\n"+
				"    ok contract::grid_load_forbidden(\"denied\")\n", "load_noreq"),
			`handler "load_noreq" takes 1 inputs`,
		},
		"swapped request input": {
			actionBindingsExtra("fn contract::grid_load_outcome load_swapped\n"+
				"    emits []\n"+
				"    given\n"+
				"        near sql::pool pool\n"+
				"        contract::invoice_key key\n"+
				"        http::request req\n"+
				"    asserts\n"+
				"        sample: contract::invoice_key(1, 7) => ok contract::grid_load_forbidden(\"denied\")\n"+
				"    match call http::request_headers(req)\n"+
				"        ok http::header[] headers => ok contract::grid_load_forbidden(\"denied\")\n", "load_swapped"),
			`handler "load_swapped" must take http::request first`,
		},
		"wrong handler result": {
			actionBindingsExtra("fn contract::grid_edit_outcome load_wrong_result\n"+
				"    emits []\n"+
				"    given\n"+
				"        near sql::pool pool\n"+
				"        http::request req\n"+
				"        contract::invoice_key key\n"+
				"    asserts\n"+
				"        sample: contract::invoice_key(1, 7) => ok contract::grid_forbidden(\"op-1\", \"denied\")\n"+
				"    ok contract::grid_forbidden(\"op-1\", \"denied\")\n", "load_wrong_result"),
			`handler "load_wrong_result" returns `,
		},
		"fallible handler": {
			actionBindingsExtra("fn contract::grid_load_outcome load_emits\n"+
				"    emits [http::invalid_request]\n"+
				"    given\n"+
				"        near sql::pool pool\n"+
				"        http::request req\n"+
				"        contract::invoice_key key\n"+
				"    asserts\n"+
				"        sample: contract::invoice_key(1, 7) => ok contract::grid_load_forbidden(\"denied\")\n"+
				"    ok contract::grid_load_forbidden(\"denied\")\n", "load_emits"),
			`handler "load_emits" emits [http::invalid_request], but action "contract::load_invoice_grid" requires emits []`,
		},
		"generic handler": {
			actionBindingsExtra("fn contract::grid_load_outcome load_generic<item>\n"+
				"    emits []\n"+
				"    given\n"+
				"        near sql::pool pool\n"+
				"        http::request req\n"+
				"        contract::invoice_key key\n"+
				"        item probe\n"+
				"    asserts\n"+
				"        sample: contract::invoice_key(1, 7), 3 => ok contract::grid_load_forbidden(\"denied\")\n"+
				"    ok contract::grid_load_forbidden(\"denied\")\n", "load_generic"),
			`handler "load_generic" must be non-generic`,
		},
		"variadic handler": {
			actionBindingsExtra("fn contract::grid_load_outcome load_variadic\n"+
				"    emits []\n"+
				"    given\n"+
				"        near sql::pool pool\n"+
				"        http::request req\n"+
				"        contract::invoice_key key\n"+
				"        str... rest\n"+
				"    asserts\n"+
				"        sample: contract::invoice_key(1, 7) => ok contract::grid_load_forbidden(\"denied\")\n"+
				"    ok contract::grid_load_forbidden(\"denied\")\n", "load_variadic"),
			`handler "load_variadic" must be non-variadic`,
		},
		"wrong captures record": {
			actionBindingsExtra("fn contract::grid_load_outcome load_wrongkey\n"+
				"    emits []\n"+
				"    given\n"+
				"        near sql::pool pool\n"+
				"        http::request req\n"+
				"        contract::grid_edit_input key\n"+
				"    asserts\n"+
				"        sample: contract::grid_edit_input(\"op-1\", \"r1\", []) => ok contract::grid_load_forbidden(\"denied\")\n"+
				"    ok contract::grid_load_forbidden(\"denied\")\n", "load_wrongkey"),
			`handler "load_wrongkey" input 2 must be `,
		},
		"bare handler name": {
			map[string][][2]string{server: {{loadMount, "action::mount(contract::load_invoice_grid, load_grid)"}}},
			`no eligible declaration for "load_grid"`,
		},
		"missing renderer pair": {
			map[string][][2]string{server: {{"action::mount(contract::save_invoice_html, callable save_html, callable render_html, callable render_bad_form)", "action::mount(contract::save_invoice_html, callable save_html)"}}},
			"takes its handler plus the normal and structural renderers",
		},
		"swapped renderers": {
			map[string][][2]string{server: {{"callable save_html, callable render_html, callable render_bad_form", "callable save_html, callable render_bad_form, callable render_html"}}},
			`normal renderer "render_bad_form" must take `,
		},
		"fallible renderer": {
			map[string][][2]string{server: {{"fn html::safe render_html\n    emits []\n", "fn html::safe render_html\n    emits [http::invalid_request]\n"}}},
			`normal renderer "render_html" emits [http::invalid_request]`,
		},
		"missing pool capture": {
			map[string][][2]string{server: {{"fn http::router routes\n    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n    given\n        near sql::pool pool\n        near str public_origin\n", "fn http::router routes\n    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n    given\n        near str public_origin\n"}}},
			"near capture pool of ",
		},
		"mistyped pool capture": {
			map[string][][2]string{server: {
				{"fn http::router routes\n    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n    given\n        near sql::pool pool\n", "fn http::router routes\n    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n    given\n        near str pool\n"},
				{"        sample: \"https://shop.example\" => ok\n    match chain\n", "        sample: \"pool\", \"https://shop.example\" => ok\n    match chain\n"},
			}},
			"requires exact declared type",
		},
		"request on JSON POST": {
			map[string][][2]string{client: {{"action::request(contract::load_invoice_grid, key)", "action::request(contract::save_invoice_grid, key)"}}},
			"requires a bodyless JSON GET action",
		},
		"request on HTML POST": {
			map[string][][2]string{client: {{"action::request(contract::load_invoice_grid, key)", "action::request(contract::save_invoice_html, key)"}}},
			`is HTML POST`,
		},
		"body on GET request": {
			map[string][][2]string{client: {{"action::request(contract::load_invoice_grid, key)", "action::request(contract::load_invoice_grid, key, key)"}}},
			"takes its captures record and no body",
		},
		"post on GET": {
			map[string][][2]string{client: {{"action::post(contract::save_invoice_grid, key, body)", "action::post(contract::load_invoice_grid, key, body)"}}},
			"requires a JSON POST action",
		},
		"post on HTML": {
			map[string][][2]string{client: {{"action::post(contract::save_invoice_grid, key, body)", "action::post(contract::save_invoice_html, key, body)"}}},
			`is HTML POST`,
		},
		"post without body": {
			map[string][][2]string{client: {{"action::post(contract::save_invoice_grid, key, body)", "action::post(contract::save_invoice_grid, key)"}}},
			"requires its captures record and wire body",
		},
		"url with extra operand": {
			map[string][][2]string{client: {{"action::url(contract::load_invoice_grid, key)", "action::url(contract::load_invoice_grid, key, key)"}}},
			"requires its captures record",
		},
		"url with mistyped captures": {
			map[string][][2]string{client: {{"action::url(contract::load_invoice_grid, key)", `action::url(contract::load_invoice_grid, "1/7")`}}},
			"does not fit expected type",
		},
		"renamed capture field": {
			map[string][][2]string{contract: {{"record invoice_key\n    int tenant_id\n    int invoice_id\n", "record invoice_key\n    int tenant\n    int invoice_id\n"}}},
			"path capture :tenant_id has no captures field",
		},
		"renamed form field": {
			map[string][][2]string{contract: {{"record line_wire\n    str id\n    str quantity\n", "record line_wire\n    str id\n    str qty\n"}}},
			`has no field "quantity"`,
		},
		"mount with type arguments": {
			map[string][][2]string{server: {{"action::mount(contract::load_invoice_grid,", "action::mount<contract::grid_load_outcome>(contract::load_invoice_grid,"}}},
			"is not a generic source function",
		},
	} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{
				"src/contract/contract.can": actionContractSource,
				"src/model/model.can":       actionBindingsModel,
				"src/server/server.can":     actionBindingsServer,
				"src/client/client.can":     actionBindingsClient,
				"src/app/main.can":          "package app\n    provides []\n    uses []\n" + actionMain,
			}
			for file, pairs := range tc.mutate {
				for _, pair := range pairs {
					if !strings.Contains(files[file], pair[0]) {
						t.Fatalf("mutation anchor missing: %q", pair[0])
					}
					files[file] = strings.Replace(files[file], pair[0], pair[1], 1)
				}
			}
			_, err := programFixture(t, files)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("consumer diagnostic omits %q: %v", tc.want, err)
			}
		})
	}
}

func TestActionBindingsRejectCallableReference(t *testing.T) {
	server := strings.Replace(actionBindingsServer, "fn http::router routes\n",
		"fn http::route bad_ref\n"+
			"    emits []\n"+
			"    asserts\n"+
			"        sample: => ok\n"+
			"    callable http::route () emits [http::invalid_route] bad = callable action::mount\n"+
			"    ok call bad()\n"+
			"fn http::router routes\n", 1)
	files := map[string]string{
		"src/contract/contract.can": actionContractSource,
		"src/model/model.can":       actionBindingsModel,
		"src/server/server.can":     server,
		"src/client/client.can":     actionBindingsClient,
		"src/app/main.can":          "package app\n    provides []\n    uses []\n" + actionMain,
	}
	_, err := programFixture(t, files)
	if err == nil || !strings.Contains(err.Error(), "require a direct call with an action symbol") {
		t.Fatalf("callable reference diagnostic omits the direct-call rule: %v", err)
	}
}

// actionBindingsTiny is a single-package captureless program: both JSON
// actions declare no captures, so every consumer omits the capture
// operand instead of inventing an empty record. Symbols are unqualified.
const actionBindingsTiny = "package tiny\n" +
	"    provides []\n" +
	"    uses [action, codec, http, sql]\n" +
	"record note_wire\n" +
	"    str title\n" +
	"    str body\n" +
	"record note_saved\n" +
	"    str title\n" +
	"record note_failed\n" +
	"    str reason\n" +
	"variant note_outcome\n" +
	"    note_saved\n" +
	"    note_failed\n" +
	"action note_save\n" +
	"    post \"/notes/save\"\n" +
	"    json note_wire limit 1024\n" +
	"    returns note_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        note_saved status 200\n" +
	"        note_failed status 422\n" +
	"action ping\n" +
	"    get \"/ping\"\n" +
	"    input none\n" +
	"    returns note_outcome\n" +
	"    body json\n" +
	"    cases\n" +
	"        note_saved status 200\n" +
	"        note_failed status 422\n" +
	"fn note_outcome save_note\n" +
	"    emits []\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        http::request req\n" +
	"        note_wire body\n" +
	"    asserts\n" +
	"        sample: note_wire(\"t\", \"b\") => ok note_saved(\"t\")\n" +
	"    match call http::request_headers(req)\n" +
	"        ok http::header[] headers => ok note_saved(body.title)\n" +
	"fn note_outcome check_ping\n" +
	"    emits []\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"        http::request req\n" +
	"    asserts\n" +
	"        sample: => ok note_failed(\"down\")\n" +
	"    match call http::request_headers(req)\n" +
	"        ok http::header[] headers => ok note_failed(\"down\")\n" +
	"fn http::router routes\n" +
	"    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]\n" +
	"    given\n" +
	"        near sql::pool pool\n" +
	"    asserts\n" +
	"        sample: => ok\n" +
	"    match chain\n" +
	"        call action::mount(note_save, callable save_note) as http::route save\n" +
	"        call action::mount(ping, callable check_ping) as http::route probe\n" +
	"        call http::make_router([save, probe]) as http::router built\n" +
	"        http::invalid_route\n" +
	"        http::duplicate_route\n" +
	"        http::ambiguous_route\n" +
	"        ok => ok built\n" +
	"fn str ping_url\n" +
	"    emits [action::invalid_path]\n" +
	"    asserts\n" +
	"        sample: => ok \"/ping\"\n" +
	"    match call action::url(ping)\n" +
	"        action::invalid_path\n" +
	"        ok str built => ok built\n" +
	"fn note_outcome fetch_ping\n" +
	"    emits [http::transport_failed, http::invalid_request, http::status_error, codec::invalid_data]\n" +
	"    asserts\n" +
	"        sample: => ok note_saved(\"t\")\n" +
	"    match call action::request(ping)\n" +
	"        when\n" +
	"            sample: => ok note_saved(\"t\")\n" +
	"        http::transport_failed\n" +
	"        http::invalid_request\n" +
	"        http::status_error\n" +
	"        codec::invalid_data\n" +
	"        ok note_outcome got => ok got\n" +
	"fn note_outcome fetch_note\n" +
	"    emits [http::transport_failed, http::invalid_request, http::body_limit, http::status_error, codec::invalid_data]\n" +
	"    given\n" +
	"        note_wire body\n" +
	"    asserts\n" +
	"        sample: note_wire(\"t\", \"b\") => ok note_saved(\"t\")\n" +
	"    match call action::post(note_save, body)\n" +
	"        when\n" +
	"            sample: note_wire(\"t\", \"b\") => ok note_saved(\"t\")\n" +
	"        http::transport_failed\n" +
	"        http::invalid_request\n" +
	"        http::body_limit\n" +
	"        http::status_error\n" +
	"        codec::invalid_data\n" +
	"        ok note_outcome got => ok got\n"

func TestActionBindingsNoCapture(t *testing.T) {
	program, err := programFixture(t, map[string]string{
		"src/tiny/tiny.can": actionBindingsTiny,
		"src/app/main.can":  "package app\n    provides []\n    uses []\n" + actionMain,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !program.ActionRoutes || !program.ActionClient {
		t.Fatalf("tiny state use = routes %v client %v, want both", program.ActionRoutes, program.ActionClient)
	}
	counts := map[string]int{}
	for _, step := range actionBindingSteps(t, program) {
		if step.Action == nil {
			continue
		}
		counts[step.Identity]++
		if step.Action.CapturesType != "" || len(step.Action.Captures) != 0 {
			t.Fatalf("%s site gained captures: %+v", step.Identity, step.Action)
		}
		switch step.Identity {
		case actionMount:
			if len(step.Contract.Inputs()) != 1 {
				t.Fatalf("captureless mount contract takes %d operands, want 1 handler", len(step.Contract.Inputs()))
			}
			want := 2
			if strings.HasSuffix(step.Action.Action, "::ping") {
				want = 1
			}
			handler := step.Contract.Inputs()[0]
			if len(handler.Inputs()) != want {
				t.Fatalf("captureless handler takes %d inputs, want %d", len(handler.Inputs()), want)
			}
		case actionURL, actionRequest:
			if len(step.Contract.Inputs()) != 0 || len(step.Arguments) != 0 {
				t.Fatalf("%s step keeps value operands without captures", step.Identity)
			}
		case actionPost:
			if len(step.Contract.Inputs()) != 1 || len(step.Arguments) != 1 {
				t.Fatalf("captureless post step lost its wire body")
			}
		}
	}
	if counts[actionMount] != 2 || counts[actionURL] != 1 || counts[actionRequest] != 1 || counts[actionPost] != 1 {
		t.Fatalf("tiny sites = %+v", counts)
	}
}

func TestActionBindingsBodyModeSwitch(t *testing.T) {
	// Switching a JSON action to a form wire keeps the declaration valid
	// but invalidates the JSON-only post construction: the old JSON body
	// can never address the form action.
	decls := actionBindingsTiny[:strings.Index(actionBindingsTiny, "fn note_outcome save_note\n")]
	client := "fn note_outcome fetch_note\n" +
		"    emits [http::transport_failed, http::invalid_request, http::body_limit, http::status_error, codec::invalid_data]\n" +
		"    given\n" +
		"        note_wire body\n" +
		"    asserts\n" +
		"        sample: note_wire(\"t\", \"b\") => ok note_saved(\"t\")\n" +
		"    match call action::post(note_save, body)\n" +
		"        http::transport_failed\n" +
		"        http::invalid_request\n" +
		"        http::body_limit\n" +
		"        http::status_error\n" +
		"        codec::invalid_data\n" +
		"        ok note_outcome got => ok got\n"
	base := decls + client
	if _, err := programFixture(t, map[string]string{
		"src/tiny/tiny.can": base,
		"src/app/main.can":  "package app\n    provides []\n    uses []\n" + actionMain,
	}); err != nil {
		t.Fatalf("JSON post fixture rejected: %v", err)
	}
	switched := strings.Replace(base, "    post \"/notes/save\"\n    json note_wire limit 1024\n", "    post \"/notes/save\"\n    form note_wire limit 1024\n", 1)
	switched = strings.Replace(switched, "    form note_wire limit 1024\n    returns note_outcome\n    body json\n    cases\n        note_saved status 200\n        note_failed status 422\n", "    form note_wire limit 1024\n    returns note_outcome\n    body html\n    cases\n        note_saved status 200 swap inner\n        note_failed status 422 swap inner\n", 1)
	if switched == base {
		t.Fatal("body-mode switch did not rewrite the declaration")
	}
	_, err := programFixture(t, map[string]string{
		"src/tiny/tiny.can": switched,
		"src/app/main.can":  "package app\n    provides []\n    uses []\n" + actionMain,
	})
	if err == nil || !strings.Contains(err.Error(), `requires a JSON POST action, but "note_save" is HTML POST`) {
		t.Fatalf("switched post diagnostic omits the JSON-only rule: %v", err)
	}
}

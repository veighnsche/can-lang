package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/resolve"
	"github.com/veighnsche/can-lang/compiler/internal/source"
)

func browserFixture(t *testing.T, files map[string]string) *check.Program {
	t.Helper()
	root := t.TempDir()
	all := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
	}
	for name, text := range files {
		all[name] = text
	}
	for name, text := range all {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestBrowserAdmitsPureProgram(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [text]
fn int shout
    emits []
    given
        int value
    asserts
        sample: 3 => ok 3
    ok value
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call shout(1)
        ok int shouted => ok
`})
	if err := CheckProgram(program); err != nil {
		t.Fatalf("pure program rejected: %v", err)
	}
}

func TestBrowserAdmitsSharedWireCodec(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [codec, bytes, option]
record point
    int x
    int y
fn point load
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"x\":1,\"y\":2}" => ok point(1, 2)
    match chain
        call bytes::from_utf8(text) as bytes::buffer raw
        call codec::decode_json<point>(raw) as point found
        codec::invalid_data
        ok => ok found
fn void main
    emits [codec::invalid_data]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call load("{\"x\":1,\"y\":2}")
        codec::invalid_data
        ok point found => ok
`})
	if err := CheckProgram(program); err != nil {
		t.Fatalf("wire-codec program rejected: %v", err)
	}
	if len(program.Codecs) == 0 {
		t.Fatal("expected shared codec specializations")
	}
}

func TestBrowserAdmitsErrorIdentityWithoutCapability(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [files]
fn str describe
    emits [files::not_found]
    given
        int missing
    asserts
        sample: 7 => files::not_found("gone")
    files::not_found("gone")
fn void main
    emits [files::not_found]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call describe(1)
        files::not_found
        ok str text => ok
`})
	if err := CheckProgram(program); err != nil {
		t.Fatalf("error-identity program rejected: %v", err)
	}
}

func TestBrowserRejectsDirectServerCapabilities(t *testing.T) {
	cases := map[string]struct {
		source string
		want   string
	}{
		"sql": {`package app
    provides []
    uses [sql]
fn void main
    emits [sql::connection_failed]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call sql::sqlite_open_memory()
        sql::connection_failed
        ok sql::pool opened => ok
`, "can.std.sql@1::sqlite_open_memory"},
		"process": {`package app
    provides []
    uses [process, files]
fn void main
    emits [files::not_found, process::invalid_config]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call process::which("sh")
        files::not_found
        process::invalid_config
        ok str found => ok
`, "can.std.process@1::which"},
		"files": {`package app
    provides []
    uses [files, codec]
fn void main
    emits [files::not_found, files::denied, files::invalid_path, files::unexpected_kind, files::limit_exceeded, codec::invalid_data, files::io_error]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call files::read_text("/tmp/x", 8)
        files::not_found
        files::denied
        files::invalid_path
        files::unexpected_kind
        files::limit_exceeded
        codec::invalid_data
        files::io_error
        ok str text => ok
`, "can.std.files@1::read_text"},
		"env": {`package app
    provides []
    uses [env, http]
fn void main
    emits [env::invalid_name, http::credentials_missing]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call env::required("HOME")
        env::invalid_name
        http::credentials_missing
        ok str value => ok
`, "can.std.env@1::required"},
		"crypto": {`package app
    provides []
    uses [crypto, bytes, codec]
fn void main
    emits [codec::invalid_data]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call bytes::from_utf8("x")
        codec::invalid_data
        ok bytes::buffer raw => match call crypto::sha256(raw)
            ok bytes::buffer digest => ok
`, "can.std.crypto@1::sha256"},
		"io": {`package app
    provides []
    uses [io, bytes, codec]
fn void main
    emits [codec::invalid_data, io::write_failed]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call bytes::from_utf8("x")
        codec::invalid_data
        ok bytes::buffer raw => match call io::stdout_write(raw)
            io::write_failed
            ok int written => ok
`, "can.std.io@1::stdout_write"},
		"websocket": {`package app
    provides []
    uses [ws]
fn void main
    emits [ws::connect_failed, ws::invalid_url, ws::invalid_protocol, ws::limit_exceeded]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call ws::connect("wss://example.invalid", [], 8, 8, 8, 8, false)
        ws::connect_failed
        ws::invalid_url
        ws::invalid_protocol
        ws::limit_exceeded
        ok ws::connection opened => ok
`, "can.std.ws@1::connect"},
	}
	for name, fixture := range cases {
		t.Run(name, func(t *testing.T) {
			program := browserFixture(t, map[string]string{"src/main.can": fixture.source})
			err := CheckProgram(program)
			if err == nil {
				t.Fatalf("server capability admitted: %s", name)
			}
			if !strings.Contains(err.Error(), fixture.want) {
				t.Fatalf("diagnostic names %q, want %q", err.Error(), fixture.want)
			}
			if !strings.Contains(err.Error(), "can.project.root/app::main") {
				t.Fatalf("diagnostic loses the reference chain: %v", err)
			}
			var located *source.LocatedError
			if !errors.As(err, &located) {
				t.Fatalf("diagnostic carries no span: %v", err)
			}
			if !strings.HasSuffix(located.File, filepath.Join("src", "main.can")) {
				t.Fatalf("diagnostic file = %q", located.File)
			}
			if located.Span.Start >= located.Span.End {
				t.Fatalf("diagnostic span is empty: %+v", located.Span)
			}
		})
	}
}

func TestBrowserRejectsTransitiveHelperPath(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [env, http]
fn str helper
    emits [env::invalid_name, http::credentials_missing]
    given
        str name
    asserts
        sample: "HOME" => ok "fixture"
    match call env::required(name)
        env::invalid_name
        http::credentials_missing
        ok str value => ok value
fn str middle
    emits [env::invalid_name, http::credentials_missing]
    given
        str name
    asserts
        sample: "HOME" => ok "fixture"
    relay call helper(name)
fn void main
    emits [env::invalid_name, http::credentials_missing]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call middle("HOME")
        env::invalid_name
        http::credentials_missing
        ok str value => ok
`})
	err := CheckProgram(program)
	if err == nil {
		t.Fatal("transitive server capability admitted")
	}
	for _, want := range []string{"can.std.env@1::required", "app::middle", "app::helper"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic %q loses %q", err.Error(), want)
		}
	}
}

func TestBrowserRejectsCallableReferencePath(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [env, http]
fn str helper
    emits [env::invalid_name, http::credentials_missing]
    given
        str name
    asserts
        sample: "HOME" => ok "fixture"
    match call env::required(name)
        env::invalid_name
        http::credentials_missing
        ok str value => ok value
fn void main
    emits [env::invalid_name, http::credentials_missing]
    given
        str[] arguments
    asserts
        empty: [] => ok
    callable str (str) emits [env::invalid_name, http::credentials_missing] action = callable helper
    match call action("HOME")
        env::invalid_name
        http::credentials_missing
        ok str value => ok
`})
	err := CheckProgram(program)
	if err == nil {
		t.Fatal("callable server capability admitted")
	}
	for _, want := range []string{"can.std.env@1::required", "app::helper"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic %q loses %q", err.Error(), want)
		}
	}
}

func TestBrowserRejectsGenericSpecializationPath(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [env, http]
fn str leak<item>
    emits [env::invalid_name, http::credentials_missing]
    given
        item value
    asserts
        integer: 3 => ok "fixture"
    match call env::required("HOME")
        env::invalid_name
        http::credentials_missing
        ok str found => ok found
fn void main
    emits [env::invalid_name, http::credentials_missing]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call leak<int>(3)
        env::invalid_name
        http::credentials_missing
        ok str value => ok
`})
	err := CheckProgram(program)
	if err == nil {
		t.Fatal("generic server capability admitted")
	}
	for _, want := range []string{"can.std.env@1::required", "app::leak"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic %q loses %q", err.Error(), want)
		}
	}
}

func TestBrowserRejectsUnreachableEmittedFunction(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [http]
fn http::server_response health
    emits []
    given
        http::request req
    asserts
        sample: => ok
    ok call http::response_text(call http::status_ok(), call http::empty_server_headers(), "healthy")
fn http::router table
    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]
    asserts
        sample: => ok
    match chain
        call http::route_get("/health", callable health) as http::route probe
        call http::make_router([probe]) as http::router built
        http::invalid_route
        http::duplicate_route
        http::ambiguous_route
        ok => ok built
fn http::server boot
    emits [http::invalid_server_config, http::invalid_route, http::duplicate_route, http::ambiguous_route, http::bind_failed]
    given
        int port
    asserts
        sample: 0 => ok
    match call http::make_server_config("127.0.0.1", port, 65536, 5000)
        http::invalid_server_config
        ok http::server_config config => match call table()
            http::invalid_route
            http::duplicate_route
            http::ambiguous_route
            ok http::router built => match call http::server_start(config, built)
                when
                    sample: config, built => ok
                http::bind_failed
                ok http::server sturdy => ok sturdy
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`})
	err := CheckProgram(program)
	if err == nil {
		t.Fatal("unreachable server capability admitted")
	}
	if !strings.Contains(err.Error(), "can.std.http@1::make_server_config") {
		t.Fatalf("diagnostic %q loses the server operation", err.Error())
	}
	if !strings.Contains(err.Error(), "unreachable from main") {
		t.Fatalf("diagnostic %q loses the unreachable emission note", err.Error())
	}
}

func TestBrowserRejectsDirectOperationCallable(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package app
    provides []
    uses [env, http]
fn void main
    emits [env::invalid_name, http::credentials_missing]
    given
        str[] arguments
    asserts
        empty: [] => ok
    callable str (str) emits [env::invalid_name, http::credentials_missing] action = callable env::required
    match call action("HOME")
        env::invalid_name
        http::credentials_missing
        ok str value => ok
`})
	err := CheckProgram(program)
	if err == nil {
		t.Fatal("direct operation callable admitted")
	}
	if !strings.Contains(err.Error(), "can.std.env@1::required") {
		t.Fatalf("diagnostic %q loses the operation", err.Error())
	}
}

func TestForbiddenReasonCoversSpecializations(t *testing.T) {
	for identity, denied := range map[string]bool{
		"can.std.sql@1::query_rows":                              true,
		"can.std.sql@1::query_rows/instance/0123456789abcdef":    true,
		"can.std.http@1::server_start":                           true,
		"can.std.http@1::server_start/instance/0123456789abcdef": true,
		"can.std.cookie@1::parse":                                true,
		"can.std.cookie@1::get":                                  true,
		"can.std.cookie@1::make":                                 true,
		"can.std.cookie@1::serialize":                            true,
		"can.std.cookie@1::expire":                               true,
		"can.std.cookie@1::parse/instance/0123456789abcdef":      true,
		"can.std.csrf@1::generate":                               true,
		"can.std.csrf@1::verify":                                 true,
		"can.std.codec@1::decode_toml":                           true,
		"can.std.codec@1::decode_yaml":                           true,
		"can.std.codec@1::decode_json5":                          true,
		"can.std.markdown@1::render_text_html":                   true,
		"can.std.markdown@1::render_safe":                        true,
		"can.std.http@1::response_text":                          false,
		"can.std.codec@1::decode_json/instance/0123456789abcdef": false,
		"can.std.clock@1::sleep_millis":                          false,
		"can.std.log@1::write_info":                              false,
		"can.intrinsic.str@1::includes":                          false,
		"can.project.root/app::helper":                           false,
		"can.project.root/app::helper/instance/0123456789abcdef": false,
	} {
		_, forbidden := ForbiddenReason(identity)
		if forbidden != denied {
			t.Fatalf("ForbiddenReason(%q) = %v, want %v", identity, forbidden, denied)
		}
	}
}

func TestBrowserRejectsNativeDeclarations(t *testing.T) {
	source, err := os.ReadFile("../../testdata/current/fetch/main.can")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"src/main.can": string(source)}
	entries, err := os.ReadDir("../../testdata/current/fetch/fixtures")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile("../../testdata/current/fetch/fixtures/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		files["src/fixtures/"+entry.Name()] = string(data)
	}
	program := browserFixture(t, files)
	if len(program.Natives) == 0 {
		t.Fatal("fixture lost its native declarations")
	}
	err = CheckProgram(program)
	if err == nil {
		t.Fatal("native declarations admitted")
	}
	if !strings.Contains(err.Error(), "native declaration") {
		t.Fatalf("diagnostic %q loses the native declaration", err.Error())
	}
}

func TestBrowserRejectsServerBackedStructures(t *testing.T) {
	native := &check.NativeDeclaration{Symbol: &resolve.Symbol{ID: "can.project.root/app::load"}}
	cases := map[string]*check.Program{
		"natives":     {Natives: []*check.NativeDeclaration{native}},
		"connections": {Connections: map[string]check.ConnectionPolicy{"service": {}}},
		"descriptors": {SQL: []ir.SQLDescriptor{{}}},
		"queries":     {SQLs: map[string]*check.SQLSpecialization{"can.std.sql@1::query_rows/instance/00": {}}},
		"stream":      {Streams: map[string]*check.StreamSpecialization{"can.std.stream@1::read_many/instance/00": {}}},
	}
	for name, program := range cases {
		t.Run(name, func(t *testing.T) {
			if err := CheckProgram(program); err == nil {
				t.Fatalf("server-backed %s admitted", name)
			}
		})
	}
}

func TestBrowserRejectsInitializerCapabilityPath(t *testing.T) {
	// Checked initializers are inert, so no fixture can smuggle a call in;
	// the gate still walks them fail-closed against future rule changes.
	program := &check.Program{
		Entry: &check.ProgramFunction{Instance: "main", Region: &ir.Region{}},
		Initializers: []ir.Initializer{{Identity: "hook", Value: &ir.Expression{
			Kind:       ir.InvocationValue,
			Invocation: &ir.Invocation{Steps: []ir.InvocationStep{{Identity: "can.std.env@1::required"}}},
		}}},
	}
	err := CheckProgram(program)
	if err == nil {
		t.Fatal("initializer server capability admitted")
	}
	if !strings.Contains(err.Error(), "can.std.env@1::required") {
		t.Fatalf("diagnostic %q loses the operation", err.Error())
	}
}

func TestBrowserAdmitsOwnerFactoryUse(t *testing.T) {
	program := browserFixture(t, map[string]string{
		"src/mail/box.can": `package mail
    provides [email, email_wire, make_email, email_address, load_wire]
    uses [codec, bytes]
owner record email
    str address
record email_wire
    str address
fn email make_email
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok email("a@b")
    ok email(address)
fn str email_address
    emits []
    given
        email mail
    asserts
        sample: email("a@b") => ok "a@b"
    ok mail.address
fn email_wire load_wire
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"address\":\"a@b\"}" => ok email_wire("a@b")
    match chain
        call bytes::from_utf8(text) as bytes::buffer encoded
        call codec::decode_json<email_wire>(encoded) as email_wire decoded
        codec::invalid_data
        ok => ok decoded
`,
		"src/app/main.can": `package app
    provides []
    uses [mail]
fn str describe
    emits []
    given
        str address
    asserts
        sample: "a@b" => ok "a@b"
    mail::email held = call mail::make_email(address)
    ok call mail::email_address(held)
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call describe("a@b")
        ok str text => ok
`,
	})
	if err := CheckProgram(program); err != nil {
		t.Fatalf("owner factory program rejected: %v", err)
	}
}

func TestBrowserRejectsFormActionBinding(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package web
    provides []
    uses [form, http, html, option]
record line_wire
    str sku
    str[] tags
    option::value<str> note
record invoice_wire
    str customer
    form::rows<line_wire> lines
record saved
    str label
record rejected
    str reason
variant save_outcome
    saved
    rejected
fn save_outcome save_validated
    emits []
    given
        invoice_wire body
    asserts
        sample: invoice_wire("c", form::rows<line_wire>([], [])) => ok saved("c")
    ok saved(body.customer)
action save_invoice
    post "/invoices/save"
    form invoice_wire limit 2048 rows_limit 64
    returns save_outcome
    body html
    cases
        saved status 200 swap inner
        rejected status 422 swap inner
fn html::safe render_outcome
    emits []
    given
        save_outcome outcome
    asserts
        sample: saved("c") => ok
    ok call html::text_fragment("done")
fn html::safe render_rejected
    emits []
    given
        form::rejected<invoice_wire> bad
    asserts
        sample: form::rejected<invoice_wire>([], []) => ok
    ok call html::text_fragment("bad")
fn http::router mounted
    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]
    asserts
        sample: => ok
    match chain
        call http::serve_form_action<save_outcome, invoice_wire>("save_invoice", callable render_outcome, callable render_rejected) as http::route form
        call http::make_router([form]) as http::router router
        http::invalid_route
        http::duplicate_route
        http::ambiguous_route
        ok => ok router
fn void main
    emits []
    given
        str[] arguments
    asserts
        empty: [] => ok
    ok
`})
	if len(program.Actions) != 1 {
		t.Fatalf("expected one canonical action, got %d", len(program.Actions))
	}
	err := CheckProgram(program)
	if err == nil {
		t.Fatal("server form-action binding admitted")
	}
	for _, want := range []string{"save_invoice", "POST /invoices/save", "mounts a server handler", "unreachable from main"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic %q loses %q", err.Error(), want)
		}
	}
	var located *source.LocatedError
	if !errors.As(err, &located) {
		t.Fatalf("diagnostic carries no span: %v", err)
	}
	if !strings.HasSuffix(located.File, filepath.Join("src", "main.can")) {
		t.Fatalf("diagnostic file = %q", located.File)
	}
}

func TestBrowserAdmitsJsonFetchClients(t *testing.T) {
	program := browserFixture(t, map[string]string{"src/main.can": `package web
    provides [save_invoice, load_line, line_key, invoice_wire, saved, rejected, stale, denied, busy, save_outcome, found, missing, unavailable, load_outcome]
    uses [http, codec]
record invoice_wire
    str label
    int seats
record saved
    str label
record rejected
    str reason
record stale
    int revision
record denied
    str reason
record busy
    str reason
variant save_outcome
    saved
    rejected
    stale
    denied
    busy
record line_key
    str invoice_id
    int line
action save_invoice
    post "/invoices/save"
    json invoice_wire limit 8192
    returns save_outcome
    body json
    cases
        saved status 200
        rejected status 422
        stale status 409
        denied status 403
        busy status 503
record found
    str label
record missing
    str reason
record unavailable
    str reason
variant load_outcome
    found
    missing
    unavailable
action load_line
    get "/invoices/:invoice_id/lines/:line"
    captures line_key
    input none
    returns load_outcome
    body json
    cases
        found status 200
        missing status 403
        unavailable status 503
fn load_outcome reload_line
    emits [http::transport_failed, http::invalid_request, http::status_error, codec::invalid_data]
    given
        str invoice_id
        int line
    asserts
        sample: "inv-1", 1 => ok found("inv-1")
    match call http::fetch_json_get<load_outcome>("load_line", invoice_id, line)
        http::transport_failed
        http::invalid_request
        http::status_error
        codec::invalid_data
        ok load_outcome got => ok got
fn save_outcome store_invoice
    emits [http::transport_failed, http::invalid_request, http::body_limit, http::status_error, codec::invalid_data]
    given
        invoice_wire body
    asserts
        sample: invoice_wire("inv-1", 2) => ok saved("inv-1")
    match call http::fetch_json_post<save_outcome, invoice_wire>("save_invoice", body)
        http::transport_failed
        http::invalid_request
        http::body_limit
        http::status_error
        codec::invalid_data
        ok save_outcome done => ok done
fn void main
    emits [http::transport_failed, http::invalid_request, http::status_error, codec::invalid_data]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call reload_line("inv-1", 1)
        http::transport_failed
        http::invalid_request
        http::status_error
        codec::invalid_data
        ok load_outcome got => ok
`})
	if len(program.Actions) != 2 {
		t.Fatalf("expected two canonical actions, got %d", len(program.Actions))
	}
	if len(program.Fetches) != 2 {
		t.Fatalf("expected two JSON fetch client sites, got %d", len(program.Fetches))
	}
	if err := CheckProgram(program); err != nil {
		t.Fatalf("JSON fetch clients rejected: %v", err)
	}
}

func TestBrowserInheritsOwnerWireBoundary(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"can.project.json": `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":  `{"active":[],"retired":[]}`,
		"src/main.can": `package app
    provides []
    uses [codec, bytes]
owner record email
    str address
fn email load
    emits [codec::invalid_data]
    given
        str text
    asserts
        sample: "{\"address\":\"a@b\"}" => ok email("a@b")
    match chain
        call bytes::from_utf8(text) as bytes::buffer encoded
        call codec::decode_json<email>(encoded) as email decoded
        codec::invalid_data
        ok => ok decoded
fn void main
    emits [codec::invalid_data]
    given
        str[] arguments
    asserts
        empty: [] => ok
    match call load("{\"address\":\"a@b\"}")
        codec::invalid_data
        ok email found => ok
`,
	}
	for name, text := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	_, err = check.CheckProgram(graph)
	if err == nil {
		t.Fatal("wire decode into an owner record admitted")
	}
	if !strings.Contains(err.Error(), "owner") {
		t.Fatalf("checker diagnostic %q loses the owner boundary", err.Error())
	}
}

func auditArtifacts(t *testing.T) []ir.Artifact {
	t.Helper()
	runtime := "runtime/r-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	modules := map[string]struct {
		imports []string
		body    string
	}{
		BrowserEntry:       {[]string{"./program/state.ts", "./packages/p-0/s-0.ts"}, "export async function $canBrowserMain(): Promise<void> {}\n"},
		"program/state.ts": {[]string{"../" + runtime + "/domain.ts"}, "export function $canInitialize(): void {}\n"},
		"packages/p-0/s-0.ts": {
			[]string{"../../program/state.ts"},
			"export async function $canFunction0(): Promise<unknown> { return null; }\n",
		},
	}
	runtimeModule := ir.Artifact{Path: runtime + "/domain.ts", Bytes: []byte("export function x(): void {}\n"), Runtime: true}
	artifacts := []ir.Artifact{runtimeModule}
	files := map[string]string{}
	for name, module := range modules {
		var body strings.Builder
		for _, spec := range module.imports {
			fmt.Fprintf(&body, "import %q;\n", spec)
		}
		body.WriteString(module.body)
		text := body.String()
		artifacts = append(artifacts, ir.Artifact{Path: name, Bytes: []byte(text), Imports: module.imports})
		sum := sha256.Sum256([]byte(text))
		files[name] = hex.EncodeToString(sum[:])
	}
	asset, err := AssetBytes(files)
	if err != nil {
		t.Fatal(err)
	}
	artifacts = append(artifacts, ir.Artifact{Path: AssetPath, Bytes: asset})
	return artifacts
}

func TestAuditAcceptsCleanBrowserGraph(t *testing.T) {
	if err := AuditArtifacts(auditArtifacts(t)); err != nil {
		t.Fatalf("clean graph rejected: %v", err)
	}
}

func TestAuditRejectsBrowserViolations(t *testing.T) {
	base := auditArtifacts(t)
	runtime := "runtime/r-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	at := func(artifacts []ir.Artifact, path string) *ir.Artifact {
		for i := range artifacts {
			if artifacts[i].Path == path {
				return &artifacts[i]
			}
		}
		t.Fatalf("missing %s", path)
		return nil
	}
	cases := map[string]func([]ir.Artifact) []ir.Artifact{
		"missing asset": func(artifacts []ir.Artifact) []ir.Artifact {
			for i := range artifacts {
				if artifacts[i].Path == AssetPath {
					artifacts[i].Path = "browser/other.json"
				}
			}
			return artifacts
		},
		"native imports": func(artifacts []ir.Artifact) []ir.Artifact {
			module := at(artifacts, "packages/p-0/s-0.ts")
			module.NativeImports = []string{"node:fs"}
			return artifacts
		},
		"absolute edge": func(artifacts []ir.Artifact) []ir.Artifact {
			module := at(artifacts, BrowserEntry)
			module.Imports = []string{"/etc/passwd.ts"}
			module.Bytes = []byte("import \"/etc/passwd.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n")
			return artifacts
		},
		"remote edge": func(artifacts []ir.Artifact) []ir.Artifact {
			module := at(artifacts, BrowserEntry)
			module.Imports = []string{"https://example.invalid/app.ts"}
			module.Bytes = []byte("import \"https://example.invalid/app.ts\";\nexport async function $canBrowserMain(): Promise<void> {}\n")
			return artifacts
		},
		"forbidden runtime edge": func(artifacts []ir.Artifact) []ir.Artifact {
			module := at(artifacts, "packages/p-0/s-0.ts")
			module.Imports = []string{"../../" + runtime + "/platform/sql/pool.ts"}
			module.Bytes = []byte("import \"../../" + runtime + "/platform/sql/pool.ts\";\nexport async function $canFunction0(): Promise<unknown> { return null; }\n")
			return append(artifacts, ir.Artifact{Path: runtime + "/platform/sql/pool.ts", Bytes: []byte("export function pool(): void {}\n"), Runtime: true})
		},
		"host token": func(artifacts []ir.Artifact) []ir.Artifact {
			module := at(artifacts, "packages/p-0/s-0.ts")
			module.Bytes = []byte("import \"../../program/state.ts\";\nexport const x = process.env.HOME;\n")
			return artifacts
		},
		"server mapping": func(artifacts []ir.Artifact) []ir.Artifact {
			module := at(artifacts, "packages/p-0/s-0.ts")
			module.Mappings = []ir.Mapping{{Operation: "fetch_request"}}
			return artifacts
		},
		"tampered bytes": func(artifacts []ir.Artifact) []ir.Artifact {
			module := at(artifacts, "packages/p-0/s-0.ts")
			module.Bytes = []byte("import \"../../program/state.ts\";\nexport const x = 1;\n")
			return artifacts
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			artifacts := make([]ir.Artifact, len(base))
			copy(artifacts, base)
			artifacts = mutate(artifacts)
			if err := AuditArtifacts(artifacts); err == nil {
				t.Fatalf("violation admitted: %s", name)
			}
		})
	}
	t.Run("missing entry", func(t *testing.T) {
		var filtered []ir.Artifact
		for _, artifact := range base {
			if artifact.Path != BrowserEntry {
				filtered = append(filtered, artifact)
			}
		}
		if err := AuditArtifacts(filtered); err == nil {
			t.Fatal("missing entry admitted")
		}
	})
	t.Run("bun entry present", func(t *testing.T) {
		artifacts := append(append([]ir.Artifact(nil), base...), ir.Artifact{Path: BunEntry, Bytes: []byte("x\n")})
		if err := AuditArtifacts(artifacts); err == nil {
			t.Fatal("bun entry admitted")
		}
	})
}

func TestAssetBindsContent(t *testing.T) {
	asset, err := AssetBytes(map[string]string{"browser.ts": strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := ParseAsset(asset)
	if err != nil {
		t.Fatalf("asset rejected: %v", err)
	}
	if manifest.Profile != Profile || manifest.Entry != BrowserEntry {
		t.Fatalf("asset identity = %+v", manifest)
	}
	again, err := AssetBytes(map[string]string{"browser.ts": strings.Repeat("a", 64)})
	if err != nil || string(again) != string(asset) {
		t.Fatal("asset rendering is not deterministic")
	}
	var decoded Asset
	if err := json.Unmarshal(asset, &decoded); err != nil {
		t.Fatal(err)
	}
	decoded.ContentSHA256 = strings.Repeat("0", 64)
	tampered, _ := json.Marshal(decoded)
	if _, err := ParseAsset(tampered); err == nil {
		t.Fatal("tampered asset admitted")
	}
	decoded.ContentSHA256 = manifest.ContentSHA256
	decoded.Profile = "worker"
	worker, _ := json.Marshal(decoded)
	if _, err := ParseAsset(worker); err == nil {
		t.Fatal("worker asset admitted")
	}
}

func TestParseTarget(t *testing.T) {
	if target, err := ParseTarget("bun"); err != nil || target != TargetBun {
		t.Fatalf("bun rejected: %v %q", target, err)
	}
	if target, err := ParseTarget("browser"); err != nil || target != TargetBrowser {
		t.Fatalf("browser rejected: %v %q", target, err)
	}
	for _, denied := range []string{"", "worker", "browser-worker", "node", "bun.js", "BROWSER"} {
		if _, err := ParseTarget(denied); err == nil {
			t.Fatalf("target %q admitted", denied)
		}
	}
}

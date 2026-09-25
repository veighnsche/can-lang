package emit

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veighnsche/can-lang/compiler/internal/ir"
)

// liveMountContract is the shared action contract for the live mount test:
// a bodyless JSON GET, a JSON POST and an HTML form POST with keyed rows,
// all addressed by the same int64 captures record.
const liveMountContract = "package contract\n" +
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
	"        failed status 422 swap inner\n"

// liveMountServer mounts the contract actions with request-first handlers,
// serves the combined router on a fixed loopback port, and waits for a
// shutdown signal. The handlers branch on captures and echo the wire body
// so live HTTP proves captures and codecs reach real generated code.
const liveMountServer = "package server\n" +
	"    provides [routes, main]\n" +
	"    uses [contract, action, form, html, http]\n" +
	"fn contract::grid_load_outcome load_grid\n" +
	"    emits []\n" +
	"    given\n" +
	"        http::request req\n" +
	"        contract::invoice_key key\n" +
	"    asserts\n" +
	"        sample: contract::invoice_key(1, 7) => ok contract::grid_loaded(\"r1\")\n" +
	"    match key.tenant_id is 1\n" +
	"        false => ok contract::grid_denied(\"foreign\")\n" +
	"        true => ok contract::grid_loaded(\"r1\")\n" +
	"fn contract::grid_edit_outcome save_grid\n" +
	"    emits []\n" +
	"    given\n" +
	"        http::request req\n" +
	"        contract::invoice_key key\n" +
	"        contract::grid_edit_input body\n" +
	"    asserts\n" +
	"        sample: contract::invoice_key(1, 7), contract::grid_edit_input(\"op-1\", \"r1\") => ok contract::grid_saved(\"op-1\")\n" +
	"    ok contract::grid_saved(body.operation_id)\n" +
	"fn contract::edit_outcome save_html\n" +
	"    emits []\n" +
	"    given\n" +
	"        http::request req\n" +
	"        contract::invoice_key key\n" +
	"        contract::invoice_form form\n" +
	"    asserts\n" +
	"        sample: contract::invoice_key(1, 7), contract::invoice_form(\"2\", form::rows<contract::line_wire>([], [])) => ok contract::saved(\"yes\")\n" +
	"    match key.tenant_id is 1\n" +
	"        false => ok contract::failed(\"no\")\n" +
	"        true => ok contract::saved(\"yes\")\n" +
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
	"        ok => ok built\n" +
	"fn void main\n" +
	"    emits [http::invalid_server_config, http::invalid_route, http::duplicate_route, http::ambiguous_route, http::bind_failed, http::shutdown_failed]\n" +
	"    given\n" +
	"        str[] args\n" +
	"    asserts\n" +
	"        empty: [] => ok\n" +
	"    match call http::make_server_config(\"127.0.0.1\", 18621, 65536, 5000)\n" +
	"        http::invalid_server_config\n" +
	"        ok http::server_config config => match call routes()\n" +
	"            http::invalid_route\n" +
	"            http::duplicate_route\n" +
	"            http::ambiguous_route\n" +
	"            ok http::router built => match call http::server_start(config, built)\n" +
	"                http::bind_failed\n" +
	"                ok http::server sturdy => match call http::server_wait(sturdy)\n" +
	"                    http::shutdown_failed\n" +
	"                    ok => ok\n"

// liveRuntimeArtifacts stages the maintained runtime tree as dependency
// artifacts with real bytes, so the emitted program executes against the
// actual adapters instead of link-only stubs.
func liveRuntimeArtifacts(t *testing.T) []ir.Artifact {
	t.Helper()
	root := "../../../runtime"
	var out []ir.Artifact
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".ts") {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out = append(out, ir.Artifact{Path: filepath.ToSlash(filepath.Join("runtime", relative)), Bytes: raw})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func liveWrite(t *testing.T, root, name string, data []byte) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func liveWait(t *testing.T, address string, diagnose func() string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("live server never listened on %s: %v\ndiag: %s", address, err, diagnose())
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func liveGet(t *testing.T, url string) (int, string) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(body)
}

func livePost(t *testing.T, url, contentType, body string) (int, string) {
	t.Helper()
	response, err := http.Post(url, contentType, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(raw)
}

func liveRaw(t *testing.T, address, method, target string, headers map[string]string, body string) (int, string, string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var head strings.Builder
	fmt.Fprintf(&head, "%s %s HTTP/1.1\r\nHost: x\r\nConnection: close\r\n", method, target)
	for name, text := range headers {
		fmt.Fprintf(&head, "%s: %s\r\n", name, text)
	}
	if body != "" {
		fmt.Fprintf(&head, "Content-Length: %d\r\n", len(body))
	}
	head.WriteString("\r\n")
	if _, err := conn.Write([]byte(head.String() + body)); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}
	headBody := strings.SplitN(string(raw), "\r\n\r\n", 2)
	status := 0
	fmt.Sscanf(headBody[0], "HTTP/1.1 %d", &status)
	allow := ""
	scanner := bufio.NewScanner(strings.NewReader(headBody[0]))
	for scanner.Scan() {
		if name, text, ok := strings.Cut(scanner.Text(), ":"); ok && strings.EqualFold(name, "allow") {
			allow = strings.TrimSpace(text)
		}
	}
	text := ""
	if len(headBody) == 2 {
		text = headBody[1]
	}
	return status, allow, text
}

// TestActionMountLiveHTTP compiles the contract and server fixtures to
// real TypeScript, executes the emitted entry under Bun, and drives the
// mounted actions over live HTTP. Mounts, captures, codecs and renderers
// all come from checked Can through the emitter; the test supplies only
// transport and assertions.
func TestActionMountLiveHTTP(t *testing.T) {
	bunPath, err := exec.LookPath("bun")
	if err != nil {
		t.Skip("bun is required for live emitted execution")
	}
	program := actionEmitProgram(t, map[string]string{
		"src/contract/contract.can": liveMountContract,
		"src/server/server.can":     liveMountServer,
	})
	if !program.ActionRoutes || program.ActionClient {
		t.Fatalf("emit state use = routes %v client %v, want routes only", program.ActionRoutes, program.ActionClient)
	}
	artifacts, err := ProgramModules(program, "runtime", liveRuntimeArtifacts(t))
	if err != nil {
		t.Fatal(err)
	}
	var joined strings.Builder
	for _, artifact := range artifacts {
		joined.Write(artifact.Bytes)
		joined.WriteByte('\n')
	}
	for _, want := range []string{
		`$canActionRoutes.mount(`,
		`$canActionRoutes.mountForm(`,
		`createActionRoutes`,
		`platform/action-routes.ts`,
	} {
		if !strings.Contains(joined.String(), want) {
			t.Fatalf("emitted live program omits %s", want)
		}
	}
	root := t.TempDir()
	if keep := os.Getenv("CAN_LIVE_KEEP"); keep != "" {
		root = keep
		os.RemoveAll(root)
		os.MkdirAll(root, 0700)
	}
	t.Logf("live stage: %s", root)
	for _, artifact := range artifacts {
		if len(artifact.Bytes) == 0 {
			continue
		}
		liveWrite(t, root, artifact.Path, artifact.Bytes)
	}
	// The driver normally encodes source maps; the empty index satisfies
	// the entry diagnostics the same way for a fixture with no mappings.
	liveWrite(t, root, "diagnostics/source-index.json", []byte(`{"schemaVersion":1,"kind":"can.source-index","sources":[],"modules":[]}`))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bunPath, "entry.ts")
	cmd.Dir = root
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// The runtime reads its launcher environment snapshot from fd 3, like
	// the driver provides; the fixture needs no environment entries.
	pipeRead, pipeWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.ExtraFiles = []*os.File{pipeRead}
	if err := cmd.Start(); err != nil {
		pipeRead.Close()
		pipeWrite.Close()
		t.Fatal(err)
	}
	pipeRead.Close()
	envWrote := make(chan error, 1)
	go func() {
		_, err := pipeWrite.Write([]byte("{}"))
		pipeWrite.Close()
		envWrote <- err
	}()
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	address := "127.0.0.1:18621"
	liveWait(t, address, func() string { return "stdout: " + stdout.String() + "\nstderr: " + stderr.String() })
	base := "http://" + address
	// Captured JSON GET reaches the generated handler, which branches on
	// the typed captures.
	if status, body := liveGet(t, base+"/api/tenants/1/invoices/7"); status != 200 || !strings.Contains(body, `"revision":"r1"`) {
		t.Fatalf("live load = %d %s, want 200 with r1", status, body)
	}
	if status, body := liveGet(t, base+"/api/tenants/2/invoices/7"); status != 403 || !strings.Contains(body, `"message":"foreign"`) {
		t.Fatalf("live denied = %d %s, want 403 foreign", status, body)
	}
	// JSON POST echoes the decoded wire body through the generated handler.
	if status, body := livePost(t, base+"/api/tenants/1/invoices/7", "application/json", `{"operation_id":"op-9","revision":"r2"}`); status != 200 || !strings.Contains(body, `"operation_id":"op-9"`) {
		t.Fatalf("live save = %d %s, want 200 with op-9", status, body)
	}
	// HTML form POST renders through the generated outcome renderer, and a
	// structural violation renders through the structural renderer.
	if status, body := livePost(t, base+"/tenants/1/invoices/7", "application/x-www-form-urlencoded", "seats=2&lines_order=r1&lines%5Br1%5D%5Bid%5D=x"); status != 200 || !strings.Contains(body, "saved") {
		t.Fatalf("live form = %d %s, want 200 saved", status, body)
	}
	if status, body := livePost(t, base+"/tenants/1/invoices/7", "application/x-www-form-urlencoded", "seats=2&lines_order=r1&lines_order=r1"); status != 422 || !strings.Contains(body, "bad") {
		t.Fatalf("live structural = %d %s, want 422 bad", status, body)
	}
	if status, _, _ := liveRaw(t, address, "POST", "/tenants/%31/invoices/7", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "seats=2"); status != 400 {
		t.Fatalf("live encoded int = %d, want 400", status)
	}
	if status, allow, _ := liveRaw(t, address, "PATCH", "/api/tenants/1/invoices/7", nil, ""); status != 405 || allow != "GET, POST" {
		t.Fatalf("live method = %d %q, want 405 GET, POST", status, allow)
	}
	if status, _, _ := liveRaw(t, address, "GET", "/api/tenants/a%2Fb/invoices/7", nil, ""); status != 400 {
		t.Fatalf("live separator = %d, want 400", status)
	}
	if status, _, _ := liveRaw(t, address, "GET", "/nope", nil, ""); status != 404 {
		t.Fatalf("live missing = %d, want 404", status)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	if out := strings.TrimSpace(stderr.String()); out != "" {
		t.Fatalf("live server wrote diagnostics: %s", out)
	}
}

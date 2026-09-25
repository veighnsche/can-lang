package driver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/browser"
	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/emit"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/project"
	"github.com/veighnsche/can-lang/compiler/internal/types"
)

// T24 invoice grid coverage: the Can-authored editable grid checks
// clean, passes the strict browser capability closure its server
// project fails, agrees with the server on both action contracts
// and wire codecs, and emits a browser-only bundle. Staged assert
// and browser-target builds live in tests/integration.
func gridProgram(t *testing.T, rel string) *check.Program {
	t.Helper()
	return gridProgramForTarget(t, rel, check.TargetBun)
}

func gridProgramForTarget(t *testing.T, rel string, target check.Target) *check.Program {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", rel))
	if err != nil {
		t.Fatal(err)
	}
	graph, err := project.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	program, err := check.CheckProgramForTarget(graph, target)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestInvoiceGridChecks(t *testing.T) {
	program := gridProgramForTarget(t, "examples/invoice-grid", check.TargetBrowser)
	if len(program.Assertions) == 0 {
		t.Fatal("expected grid assertions")
	}
	if program.Entry == nil {
		t.Fatal("expected a grid entry point")
	}
}

func TestInvoiceGridBrowserClosure(t *testing.T) {
	grid := gridProgramForTarget(t, "examples/invoice-grid", check.TargetBrowser)
	if err := browser.CheckProgram(grid); err != nil {
		t.Fatalf("grid must pass browser capability closure: %v", err)
	}
	server := gridProgram(t, "examples/invoice")
	if err := browser.CheckProgram(server); err == nil {
		t.Fatal("server project must fail browser closure (sql/env/crypto reach server-only natives)")
	}
}

func gridAction(t *testing.T, program *check.Program, name string) *check.ActionDeclaration {
	t.Helper()
	for _, action := range program.Actions {
		if action.Symbol.Name == name {
			return action
		}
	}
	t.Fatalf("missing action %s", name)
	return nil
}

// normalizeSchema rewrites project-qualified codec identities onto their
// package-qualified canonical tags so two projects can be compared for
// exact wire agreement.
func normalizeSchema(schema types.CodecSchema) string {
	byIdentity := map[string]string{}
	for _, node := range schema.Nodes {
		byIdentity[node.Identity] = node.Name
	}
	rewrite := func(identity string) string {
		if name, ok := byIdentity[identity]; ok {
			return name
		}
		return identity
	}
	type field struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	type node struct {
		Kind    string   `json:"kind"`
		Name    string   `json:"name"`
		Element string   `json:"element,omitempty"`
		Fields  []field  `json:"fields,omitempty"`
		Leaves  []string `json:"leaves,omitempty"`
	}
	out := struct {
		Root  string `json:"root"`
		Nodes []node `json:"nodes"`
	}{Root: rewrite(schema.Root)}
	for _, item := range schema.Nodes {
		converted := node{Kind: string(item.Kind), Name: item.Name}
		if item.Element != "" {
			converted.Element = rewrite(item.Element)
		}
		for _, itemField := range item.Fields {
			converted.Fields = append(converted.Fields, field{Name: itemField.Name, Type: rewrite(itemField.Type)})
		}
		for _, leaf := range item.Leaves {
			converted.Leaves = append(converted.Leaves, rewrite(leaf))
		}
		out.Nodes = append(out.Nodes, converted)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func TestInvoiceGridCrossTargetContract(t *testing.T) {
	grid := gridProgramForTarget(t, "examples/invoice-grid", check.TargetBrowser)
	server := gridProgram(t, "examples/invoice")
	// UP20 re-pin: both targets import the shared handler-free contract, so
	// every action on either side resolves to the same canonical locked
	// declaration identity. Neither side mirrors an action locally.
	const sharedContract = "can.project.lineage/billing/invoice_contract::"
	sharedIDs := func(program *check.Program, label string) map[string]bool {
		t.Helper()
		ids := map[string]bool{}
		for _, action := range program.Actions {
			if !strings.HasPrefix(action.Symbol.ID, sharedContract) {
				t.Fatalf("%s action %q escapes the shared contract: %q", label, action.Symbol.Name, action.Symbol.ID)
			}
			ids[action.Symbol.ID] = true
		}
		return ids
	}
	gridIDs, serverIDs := sharedIDs(grid, "grid"), sharedIDs(server, "server")
	if len(gridIDs) != 3 || len(serverIDs) != 3 || len(gridIDs) != len(serverIDs) {
		t.Fatalf("expected the 3 shared actions on both sides, grid %d server %d", len(gridIDs), len(serverIDs))
	}
	for id := range gridIDs {
		if !serverIDs[id] {
			t.Fatalf("server lacks shared action %s", id)
		}
	}
	if !grid.ActionClient {
		t.Fatal("expected grid checked action::request/post/url sites")
	}
	if len(grid.Fetches) != 0 {
		t.Fatalf("grid must not use legacy string-named fetch sites, found %d", len(grid.Fetches))
	}
	for _, name := range []string{"save_invoice_grid", "load_invoice_grid"} {
		wire, live := gridAction(t, grid, name), gridAction(t, server, name)
		if wire.Method != live.Method || wire.Path != live.Path {
			t.Fatalf("%s route drifted: grid %s %s vs server %s %s", name, wire.Method, wire.Path, live.Method, live.Path)
		}
		if len(wire.Captures) != len(live.Captures) {
			t.Fatalf("%s capture arity drifted", name)
		}
		for i := range wire.Captures {
			if wire.Captures[i].Name != live.Captures[i].Name ||
				types.CanonicalName(wire.Captures[i].Type) != types.CanonicalName(live.Captures[i].Type) {
				t.Fatalf("%s capture %d drifted", name, i)
			}
		}
		if (wire.Input.Mode == "none") != (live.Input.Mode == "none") {
			t.Fatalf("%s body presence drifted", name)
		}
		if wire.Input.Mode != "none" {
			if wire.Input.Mode != live.Input.Mode ||
				types.CanonicalName(wire.Input.Type) != types.CanonicalName(live.Input.Type) {
				t.Fatalf("%s body drifted: %s %s vs %s %s", name, wire.Input.Mode,
					types.CanonicalName(wire.Input.Type), live.Input.Mode, types.CanonicalName(live.Input.Type))
			}
			if normalizeSchema(wire.Input.Schema) != normalizeSchema(live.Input.Schema) {
				t.Fatalf("%s request codec drifted", name)
			}
		}
		if types.CanonicalName(wire.Returns) != types.CanonicalName(live.Returns) {
			t.Fatalf("%s result drifted", name)
		}
		if len(wire.Cases) != len(live.Cases) {
			t.Fatalf("%s case arity drifted", name)
		}
		for i := range wire.Cases {
			if wire.Cases[i] != live.Cases[i] {
				t.Fatalf("%s case %d drifted: %+v vs %+v", name, i, wire.Cases[i], live.Cases[i])
			}
		}
		if wire.ResponseSchema == nil || live.ResponseSchema == nil {
			t.Fatalf("%s missing response schema", name)
		}
		if normalizeSchema(*wire.ResponseSchema) != normalizeSchema(*live.ResponseSchema) {
			t.Fatalf("%s response codec drifted", name)
		}
	}
}

func gridRuntimeDependencies(t *testing.T) []ir.Artifact {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "runtime"))
	if err != nil {
		t.Fatal(err)
	}
	var dependencies []ir.Artifact
	var walk func(dir, prefix string)
	walk = func(dir, prefix string) {
		items, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			path := filepath.Join(dir, item.Name())
			rel := prefix + item.Name()
			if item.IsDir() {
				walk(path, rel+"/")
				continue
			}
			if !strings.HasSuffix(item.Name(), ".ts") {
				continue
			}
			dependencies = append(dependencies, ir.Artifact{Path: "runtime/" + rel})
		}
	}
	walk(root, "")
	return dependencies
}

func TestInvoiceGridBrowserEmission(t *testing.T) {
	program := gridProgramForTarget(t, "examples/invoice-grid", check.TargetBrowser)
	artifacts, err := emit.BrowserModules(program, "runtime", gridRuntimeDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		bodies = append(bodies, string(artifact.Bytes))
	}
	joined := strings.Join(bodies, "\n")
	for _, want := range []string{
		"$canBrowserMain",
		"$canCreateBrowser",
		"$canCreateActionClient",
		"$canActionClient.request",
		"$canActionClient.post",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("browser emission missing %s", want)
		}
	}
	for _, banned := range []string{
		"$canActionFetch.get",
		"$canActionFetch.post",
	} {
		if strings.Contains(joined, banned) {
			t.Fatalf("browser emission must not bind the legacy string-named fetch client %s", banned)
		}
	}
	for _, banned := range []string{
		"platform/server.ts",
		"platform/sql/",
		"platform/env.ts",
		"platform/crypto/",
		"platform/files/",
		"platform/process/",
		"platform/io.ts",
		"platform/s3.ts",
		"platform/websocket.ts",
		"platform/stream/",
	} {
		for _, artifact := range artifacts {
			if strings.Contains(artifact.Path, "runtime/") {
				continue
			}
			if strings.Contains(string(artifact.Bytes), banned) {
				t.Fatalf("browser emission references server-only %s in %s", banned, artifact.Path)
			}
		}
	}
}

// UP21 shared-contract mutation suite: every edit is made once in a
// disposable copy of shared/invoice-contract, synced into both consumer
// vendors with both locks rewritten, then both targets are checked and
// emitted in-process. Stale consumers must fail with located
// diagnostics; repaired consumers must agree exactly and carry the new
// contract bytes without mirrored route/status literals. Live HTTP,
// SQLite and DOM legs for the same edits run staged in
// tests/integration/invoice_contract_test.go.

const (
	contractSharedRel  = "shared/invoice-contract"
	contractServerRel  = "examples/invoice"
	contractGridRel    = "examples/invoice-grid"
	contractSourceFile = "src/invoice_contract/invoice_contract.can"
	contractVendorFile = "vendor/billing/src/invoice_contract/invoice_contract.can"
	contractBillingID  = "can.project.lineage/billing"
	contractActionID   = "can.project.lineage/billing/invoice_contract::"
)

type contractWorkspace struct {
	shared string
	server string
	grid   string
}

func contractRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func contractCopyTree(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0700); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "dist" {
			continue
		}
		from, to := filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			contractCopyTree(t, from, to)
			continue
		}
		data, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func contractWorkspaceFor(t *testing.T) contractWorkspace {
	t.Helper()
	root := contractRepoRoot(t)
	base := t.TempDir()
	ws := contractWorkspace{
		shared: filepath.Join(base, "shared"),
		server: filepath.Join(base, "server"),
		grid:   filepath.Join(base, "grid"),
	}
	contractCopyTree(t, filepath.Join(root, contractSharedRel), ws.shared)
	contractCopyTree(t, filepath.Join(root, contractServerRel), ws.server)
	contractCopyTree(t, filepath.Join(root, contractGridRel), ws.grid)
	return ws
}

func contractReadFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func contractWriteFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// contractReplaceAll rewrites every occurrence of old in the named file
// and reports how many it replaced; zero is a fatal misuse.
func contractReplaceAll(t *testing.T, name, old, new string) int {
	t.Helper()
	content := contractReadFile(t, name)
	count := strings.Count(content, old)
	if count == 0 {
		t.Fatalf("contract edit anchor %q missing from %s", old, name)
	}
	contractWriteFile(t, name, strings.ReplaceAll(content, old, new))
	return count
}

// contractReplaceOnce rewrites exactly one occurrence; any other arity
// is a fatal misuse so edits stay isolated.
func contractReplaceOnce(t *testing.T, name, old, new string) {
	t.Helper()
	if count := contractReplaceAll(t, name, old, new); count != 1 {
		t.Fatalf("contract edit anchor %q occurs %d times in %s, want 1", old, count, name)
	}
}

// rewriteSharedContract applies replacements to the shared copy once,
// syncs the edited file into both consumer vendors, and rewrites both
// locks. It returns the new source digest shared by both locks.
func (ws contractWorkspace) rewriteSharedContract(t *testing.T, replacements [][2]string) string {
	t.Helper()
	shared := filepath.Join(ws.shared, contractSourceFile)
	for _, pair := range replacements {
		contractReplaceAll(t, shared, pair[0], pair[1])
	}
	edited, err := os.ReadFile(shared)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{ws.server, ws.grid} {
		vendor := filepath.Join(root, contractVendorFile)
		if err := os.WriteFile(vendor, edited, 0600); err != nil {
			t.Fatal(err)
		}
	}
	serverDigest := contractRelock(t, ws.server)
	gridDigest := contractRelock(t, ws.grid)
	if serverDigest != gridDigest {
		t.Fatalf("relock diverged: server %s grid %s", serverDigest, gridDigest)
	}
	return serverDigest
}

// contractVendorDigest recomputes the billing source digest over the
// vendor tree.
func contractVendorDigest(t *testing.T, root string) string {
	t.Helper()
	vendor := filepath.Join(root, "vendor/billing")
	manifestRaw, err := os.ReadFile(filepath.Join(vendor, "can.project.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SourceRoot string `json:"source_root"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil || manifest.SourceRoot == "" {
		t.Fatalf("invalid vendor manifest %v %s", err, manifestRaw)
	}
	var sources []project.SourceBytes
	walkRoot := filepath.Join(vendor, manifest.SourceRoot)
	err = filepath.WalkDir(walkRoot, func(name string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".can") {
			return err
		}
		rel, err := filepath.Rel(walkRoot, name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		sources = append(sources, project.SourceBytes{Path: path.Join(filepath.ToSlash(filepath.Dir(rel)), entry.Name()), Bytes: data})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) == 0 {
		t.Fatal("vendor tree holds no .can sources")
	}
	digest, err := project.SourceDigest(sources)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

// contractWriteLockDigest rewrites the billing source digest in the
// project's lock. Manifest, fixtures and registry pins are untouched:
// only .can bytes move.
func contractWriteLockDigest(t *testing.T, root, digest string) {
	t.Helper()
	lockPath := filepath.Join(root, "can.lock.json")
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var lock map[string]any
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	projects, _ := lock["projects"].(map[string]any)
	entry, _ := projects[contractBillingID].(map[string]any)
	if entry == nil {
		t.Fatalf("lock lacks %s", contractBillingID)
	}
	entry["source_sha256"] = digest
	rewritten, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	contractWriteFile(t, lockPath, string(rewritten))
}

// contractRelock recomputes the vendor digest, rewrites the lock, and
// requires the tree to load.
func contractRelock(t *testing.T, root string) string {
	t.Helper()
	digest := contractVendorDigest(t, root)
	contractWriteLockDigest(t, root, digest)
	if _, err := project.Load(root); err != nil {
		t.Fatalf("relocked tree fails to load: %v", err)
	}
	return digest
}

func checkContractTree(t *testing.T, root string, target check.Target) (*check.Program, error) {
	t.Helper()
	graph, err := project.Load(root)
	if err != nil {
		return nil, err
	}
	return check.CheckProgramForTarget(graph, target)
}

func mustCheckContractTree(t *testing.T, root string, target check.Target) *check.Program {
	t.Helper()
	program, err := checkContractTree(t, root, target)
	if err != nil {
		t.Fatalf("check %s: %v", root, err)
	}
	return program
}

// expectContractFailure requires checking to fail with a diagnostic
// containing every want token and returns the full diagnostic text so
// the test records its source span.
func expectContractFailure(t *testing.T, root string, target check.Target, wants ...string) string {
	t.Helper()
	_, err := checkContractTree(t, root, target)
	if err == nil {
		t.Fatalf("check %s unexpectedly passed, want %q", root, wants)
	}
	for _, want := range wants {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("check %s diagnostic %q lacks %q", root, err.Error(), want)
		}
	}
	return err.Error()
}

func emitContractTree(t *testing.T, program *check.Program, browserTarget bool) string {
	t.Helper()
	var artifacts []ir.Artifact
	var err error
	if browserTarget {
		artifacts, err = emit.BrowserModules(program, "runtime", gridRuntimeDependencies(t))
	} else {
		artifacts, err = emit.ProgramModules(program, "runtime", gridRuntimeDependencies(t))
	}
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		bodies = append(bodies, string(artifact.Bytes))
	}
	return strings.Join(bodies, "\n")
}

func digestString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func contractAction(t *testing.T, program *check.Program, name string) *check.ActionDeclaration {
	t.Helper()
	for _, action := range program.Actions {
		if action.Symbol.Name == name {
			return action
		}
	}
	t.Fatalf("missing action %s", name)
	return nil
}

// normalizeFormSchema renders a form schema with project-qualified
// identities reduced to package-qualified tags so two projects can be
// compared for exact wire agreement.
func normalizeFormSchema(schema types.FormSchema) string {
	normalize := func(identity string) string {
		if index := strings.LastIndex(identity, "/"); index >= 0 {
			identity = identity[index+1:]
		}
		if index := strings.LastIndex(identity, "::"); index >= 0 {
			// Keep the package-qualified tail only.
			head := identity[:index]
			if slash := strings.LastIndex(head, "/"); slash >= 0 {
				head = head[slash+1:]
			}
			return head + identity[index:]
		}
		return identity
	}
	converted := struct {
		Root   string `json:"root"`
		Fields []struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
			Some string `json:"some,omitempty"`
			None string `json:"none,omitempty"`
			Rows string `json:"rows,omitempty"`
		} `json:"fields"`
	}{Root: normalize(schema.Root)}
	for _, field := range schema.Fields {
		rows := ""
		if field.Rows != nil {
			raw, err := json.Marshal(field.Rows)
			if err != nil {
				panic(err)
			}
			rows = normalize(string(raw))
		}
		converted.Fields = append(converted.Fields, struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
			Some string `json:"some,omitempty"`
			None string `json:"none,omitempty"`
			Rows string `json:"rows,omitempty"`
		}{Name: field.Name, Kind: normalize(field.Kind), Some: normalize(field.Some), None: normalize(field.None), Rows: rows})
	}
	raw, err := json.Marshal(converted)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// assertContractAgreement requires all three shared actions to agree
// exactly across targets: route, captures, body mode and limits,
// result type, case table with statuses and swap policy, and wire
// codecs. kindOf names the comparison for diagnostics.
func assertContractAgreement(t *testing.T, kindOf string, grid, server *check.Program) {
	t.Helper()
	for _, name := range []string{"load_invoice_grid", "save_invoice_grid", "save_invoice_html"} {
		wire, live := contractAction(t, grid, name), contractAction(t, server, name)
		if !strings.HasPrefix(wire.Symbol.ID, contractActionID) || !strings.HasPrefix(live.Symbol.ID, contractActionID) {
			t.Fatalf("%s %s escapes the shared contract: %q vs %q", kindOf, name, wire.Symbol.ID, live.Symbol.ID)
		}
		if wire.Method != live.Method || wire.Path != live.Path {
			t.Fatalf("%s %s route drifted: grid %s %s vs server %s %s", kindOf, name, wire.Method, wire.Path, live.Method, live.Path)
		}
		if len(wire.Captures) != len(live.Captures) {
			t.Fatalf("%s %s capture arity drifted", kindOf, name)
		}
		for i := range wire.Captures {
			if wire.Captures[i].Name != live.Captures[i].Name ||
				types.CanonicalName(wire.Captures[i].Type) != types.CanonicalName(live.Captures[i].Type) {
				t.Fatalf("%s %s capture %d drifted", kindOf, name, i)
			}
		}
		if wire.Input.Mode != live.Input.Mode || wire.Input.Limit != live.Input.Limit || wire.Input.RowsLimit != live.Input.RowsLimit {
			t.Fatalf("%s %s input drifted: %+v vs %+v", kindOf, name, wire.Input, live.Input)
		}
		switch wire.Input.Mode {
		case "json":
			if types.CanonicalName(wire.Input.Type) != types.CanonicalName(live.Input.Type) {
				t.Fatalf("%s %s body type drifted", kindOf, name)
			}
			if normalizeSchema(wire.Input.Schema) != normalizeSchema(live.Input.Schema) {
				t.Fatalf("%s %s request codec drifted", kindOf, name)
			}
		case "form":
			if types.CanonicalName(wire.Input.Type) != types.CanonicalName(live.Input.Type) {
				t.Fatalf("%s %s form type drifted", kindOf, name)
			}
			if normalizeFormSchema(wire.Input.Form) != normalizeFormSchema(live.Input.Form) {
				t.Fatalf("%s %s form codec drifted:\n%s\n%s", kindOf, name, normalizeFormSchema(wire.Input.Form), normalizeFormSchema(live.Input.Form))
			}
		}
		if types.CanonicalName(wire.Returns) != types.CanonicalName(live.Returns) {
			t.Fatalf("%s %s result drifted", kindOf, name)
		}
		if wire.Body != live.Body {
			t.Fatalf("%s %s body mode drifted: %s vs %s", kindOf, name, wire.Body, live.Body)
		}
		if len(wire.Cases) != len(live.Cases) {
			t.Fatalf("%s %s case arity drifted", kindOf, name)
		}
		for i := range wire.Cases {
			if wire.Cases[i] != live.Cases[i] {
				t.Fatalf("%s %s case %d drifted: %+v vs %+v", kindOf, name, i, wire.Cases[i], live.Cases[i])
			}
		}
		if (wire.ResponseSchema == nil) != (live.ResponseSchema == nil) {
			t.Fatalf("%s %s response schema presence drifted", kindOf, name)
		}
		if wire.ResponseSchema != nil && normalizeSchema(*wire.ResponseSchema) != normalizeSchema(*live.ResponseSchema) {
			t.Fatalf("%s %s response codec drifted", kindOf, name)
		}
	}
}

// contractMirrorTokens are route, capture and status literals that may
// appear only in the shared contract: a consumer copy fails the audit.
var contractMirrorTokens = []string{
	"/api/", "/tenants/:", ":tenant_id", ":invoice_id", ":document_id",
	"status 200", "status 201", "status 422", "status 409", "status 403", "status 503",
}

// auditNoContractMirror requires consumer sources outside vendor trees
// to carry no copied route, capture or status literal.
func auditNoContractMirror(t *testing.T, ws contractWorkspace) {
	t.Helper()
	if err := findContractMirror(t, ws); err != nil {
		t.Fatal(err)
	}
}

func findContractMirror(t *testing.T, ws contractWorkspace) error {
	t.Helper()
	for _, root := range []string{ws.server, ws.grid} {
		var found error
		err := filepath.WalkDir(filepath.Join(root, "src"), func(name string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".can") {
				return err
			}
			content := stripCanComments(contractReadFile(t, name))
			for _, token := range contractMirrorTokens {
				if strings.Contains(content, token) {
					found = fmt.Errorf("mirror audit: %s copies contract literal %q", name, token)
					return found
				}
			}
			return nil
		})
		if found != nil {
			return found
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	return nil
}

// stripCanComments drops // prose to end of line outside string
// literals so the mirror audit judges code, not documentation.
func stripCanComments(content string) string {
	var out strings.Builder
	for _, line := range strings.Split(content, "\n") {
		inString := false
		cut := -1
		for i := 0; i+1 < len(line); i++ {
			switch line[i] {
			case '"':
				inString = !inString
			case '/':
				if !inString && line[i+1] == '/' {
					cut = i
				}
			}
			if cut >= 0 {
				break
			}
		}
		if cut >= 0 {
			line = line[:cut]
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

type contractEvidence struct {
	Edit   string `json:"edit"`
	Phase  string `json:"phase"`
	Target string `json:"target"`
	Result string `json:"result"`
	Detail string `json:"detail"`
}

func logContractEvidence(t *testing.T, rows []contractEvidence) {
	t.Helper()
	raw, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("contract evidence:\n%s", raw)
}

func TestContractWorkspacePristine(t *testing.T) {
	ws := contractWorkspaceFor(t)
	contractText := contractReadFile(t, filepath.Join(ws.shared, contractSourceFile))
	for _, want := range []string{
		`get "/api/tenants/:tenant_id/invoices/:invoice_id"`,
		`json grid_edit_input limit 8192`,
		"grid_saved status 200",
		"grid_invalid status 422",
		"grid_conflict status 409",
		"grid_forbidden status 403",
		"grid_unavailable status 503",
	} {
		if !strings.Contains(contractText, want) {
			t.Fatalf("pristine contract lacks %q", want)
		}
	}
	if strings.Contains(contractText, "handles") {
		t.Fatal("pristine contract carries a handles clause")
	}
	// The relock method must reproduce the committed digest byte for
	// byte on unedited trees, or edited locks cannot be trusted.
	committed := "16b137f9cfebe39b9617e4fec00f0aaa985055c80310ee0399de7694365620e5"
	for _, root := range []string{ws.server, ws.grid} {
		if digest := contractRelock(t, root); digest != committed {
			t.Fatalf("relock recomputed %s, want committed %s", digest, committed)
		}
	}
	grid := mustCheckContractTree(t, ws.grid, check.TargetBrowser)
	server := mustCheckContractTree(t, ws.server, check.TargetBun)
	if err := browser.CheckProgram(grid); err != nil {
		t.Fatalf("pristine grid fails browser closure: %v", err)
	}
	assertContractAgreement(t, "pristine", grid, server)
	auditNoContractMirror(t, ws)
	logContractEvidence(t, []contractEvidence{
		{Edit: "pristine", Phase: "relock", Target: "both", Result: "pass", Detail: "recomputed " + committed[:12]},
		{Edit: "pristine", Phase: "agreement", Target: "both", Result: "pass", Detail: "3 shared actions agree"},
	})
}

func TestContractEditRoute(t *testing.T) {
	ws := contractWorkspaceFor(t)
	var evidence []contractEvidence
	digest := ws.rewriteSharedContract(t, [][2]string{
		{`get "/api/tenants/:tenant_id/invoices/:invoice_id"`, `get "/api/v2/tenants/:tenant_id/invoices/:invoice_id"`},
	})
	evidence = append(evidence, contractEvidence{Edit: "route", Phase: "relock", Target: "both", Result: "pass", Detail: digest[:12]})
	// Checked consumers update automatically: no consumer source moves.
	grid := mustCheckContractTree(t, ws.grid, check.TargetBrowser)
	server := mustCheckContractTree(t, ws.server, check.TargetBun)
	evidence = append(evidence, contractEvidence{Edit: "route", Phase: "check", Target: "both", Result: "pass", Detail: "no consumer repair"})
	load := contractAction(t, server, "load_invoice_grid")
	if load.Method != "GET" || load.Path != "/api/v2/tenants/:tenant_id/invoices/:invoice_id" {
		t.Fatalf("load route = %s %s", load.Method, load.Path)
	}
	assertContractAgreement(t, "route", grid, server)
	if err := browser.CheckProgram(grid); err != nil {
		t.Fatalf("edited grid fails browser closure: %v", err)
	}
	gridBytes := emitContractTree(t, grid, true)
	serverBytes := emitContractTree(t, server, false)
	for _, bytes := range []string{gridBytes, serverBytes} {
		if !strings.Contains(bytes, "/api/v2/tenants") {
			t.Fatal("emission lacks the edited load path")
		}
	}
	evidence = append(evidence,
		contractEvidence{Edit: "route", Phase: "emit", Target: "grid", Result: "pass", Detail: digestString(gridBytes)[:12]},
		contractEvidence{Edit: "route", Phase: "emit", Target: "server", Result: "pass", Detail: digestString(serverBytes)[:12]},
	)
	auditNoContractMirror(t, ws)
	// The audit is live: a planted route copy in consumer code fails it.
	planted := filepath.Join(ws.grid, "src/web/web.can")
	contractWriteFile(t, planted, contractReadFile(t, planted)+"\nfn str probe_mirror\n    emits []\n    asserts\n        sample: => ok \"/api/v2/tenants/:tenant_id/invoices/:invoice_id\"\n    ok \"/api/v2/tenants/:tenant_id/invoices/:invoice_id\"\n")
	if err := findContractMirror(t, ws); err == nil {
		t.Fatal("mirror audit passed a planted route copy")
	} else {
		evidence = append(evidence, contractEvidence{Edit: "route", Phase: "audit", Target: "grid", Result: "reject", Detail: err.Error()})
	}
	logContractEvidence(t, evidence)
}

func TestContractEditCapture(t *testing.T) {
	ws := contractWorkspaceFor(t)
	var evidence []contractEvidence
	digest := ws.rewriteSharedContract(t, [][2]string{{"invoice_id", "document_id"}})
	evidence = append(evidence, contractEvidence{Edit: "capture", Phase: "relock", Target: "both", Result: "pass", Detail: digest[:12]})
	staleGrid := expectContractFailure(t, ws.grid, check.TargetBrowser, "invoice_id")
	staleServer := expectContractFailure(t, ws.server, check.TargetBun, "invoice_id")
	evidence = append(evidence,
		contractEvidence{Edit: "capture", Phase: "stale", Target: "grid", Result: "reject", Detail: staleGrid},
		contractEvidence{Edit: "capture", Phase: "stale", Target: "server", Result: "reject", Detail: staleServer},
	)
	repaired := 0
	for _, root := range []string{ws.server, ws.grid} {
		err := filepath.WalkDir(filepath.Join(root, "src"), func(name string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".can") {
				return err
			}
			content := contractReadFile(t, name)
			if count := strings.Count(content, "key.invoice_id"); count > 0 {
				contractWriteFile(t, name, strings.ReplaceAll(content, "key.invoice_id", "key.document_id"))
				repaired += count
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if repaired == 0 {
		t.Fatal("capture repair found no key.invoice_id uses")
	}
	grid := mustCheckContractTree(t, ws.grid, check.TargetBrowser)
	server := mustCheckContractTree(t, ws.server, check.TargetBun)
	evidence = append(evidence, contractEvidence{Edit: "capture", Phase: "check", Target: "both", Result: "pass", Detail: fmt.Sprintf("%d named uses repaired", repaired)})
	for _, program := range []*check.Program{grid, server} {
		for _, name := range []string{"load_invoice_grid", "save_invoice_grid", "save_invoice_html"} {
			action := contractAction(t, program, name)
			if len(action.Captures) != 2 || action.Captures[0].Name != "tenant_id" || action.Captures[1].Name != "document_id" {
				t.Fatalf("%s captures = %+v", name, action.Captures)
			}
			if types.CanonicalName(action.Captures[1].Type) != "int" {
				t.Fatalf("%s document_id type = %s", name, types.CanonicalName(action.Captures[1].Type))
			}
		}
	}
	assertContractAgreement(t, "capture", grid, server)
	if err := browser.CheckProgram(grid); err != nil {
		t.Fatalf("edited grid fails browser closure: %v", err)
	}
	gridBytes := emitContractTree(t, grid, true)
	serverBytes := emitContractTree(t, server, false)
	for _, bytes := range []string{gridBytes, serverBytes} {
		if !strings.Contains(bytes, "document_id") {
			t.Fatal("emission lacks the renamed capture")
		}
	}
	evidence = append(evidence,
		contractEvidence{Edit: "capture", Phase: "emit", Target: "grid", Result: "pass", Detail: digestString(gridBytes)[:12]},
		contractEvidence{Edit: "capture", Phase: "emit", Target: "server", Result: "pass", Detail: digestString(serverBytes)[:12]},
	)
	auditNoContractMirror(t, ws)
	logContractEvidence(t, evidence)
}

func TestContractEditField(t *testing.T) {
	ws := contractWorkspaceFor(t)
	var evidence []contractEvidence
	digest := ws.rewriteSharedContract(t, [][2]string{{
		"record grid_edit_input\n    str operation_id\n    str revision\n    grid_line_wire[] lines",
		"record grid_edit_input\n    str operation_id\n    str expected_revision\n    grid_line_wire[] lines",
	}})
	evidence = append(evidence, contractEvidence{Edit: "field", Phase: "relock", Target: "both", Result: "pass", Detail: digest[:12]})
	staleServer := expectContractFailure(t, ws.server, check.TargetBun, "revision")
	evidence = append(evidence, contractEvidence{Edit: "field", Phase: "stale", Target: "server", Result: "reject", Detail: staleServer})
	// The grid builds every wire value positionally, so it may keep
	// checking — but only with the moved codec, never a stale one.
	grid := mustCheckContractTree(t, ws.grid, check.TargetBrowser)
	save := contractAction(t, grid, "save_invoice_grid")
	if !strings.Contains(normalizeSchema(save.Input.Schema), "expected_revision") {
		t.Fatalf("positional grid kept a stale codec: %s", normalizeSchema(save.Input.Schema))
	}
	evidence = append(evidence, contractEvidence{Edit: "field", Phase: "positional", Target: "grid", Result: "pass", Detail: "codec moved with the shared field"})
	// Only the JSON check reads grid_edit_input: the form check's
	// body is invoice_form, whose own revision field never moves.
	contractReplaceOnce(t, filepath.Join(ws.server, "src/model/model.can"),
		"    match call text::to_int(body.revision)\n        text::invalid_number => ok records::flawed_save(\"bad revision\", \"revision\")\n        ok int revision => match revision >= 1\n            false => ok records::flawed_save(\"bad revision\", \"revision\")\n            true => match call json_parse_lines(body.lines, revision, 0, [])",
		"    match call text::to_int(body.expected_revision)\n        text::invalid_number => ok records::flawed_save(\"bad revision\", \"revision\")\n        ok int revision => match revision >= 1\n            false => ok records::flawed_save(\"bad revision\", \"revision\")\n            true => match call json_parse_lines(body.lines, revision, 0, [])")
	server := mustCheckContractTree(t, ws.server, check.TargetBun)
	evidence = append(evidence, contractEvidence{Edit: "field", Phase: "check", Target: "server", Result: "pass", Detail: "1 named use repaired"})
	assertContractAgreement(t, "field", grid, server)
	if err := browser.CheckProgram(grid); err != nil {
		t.Fatalf("edited grid fails browser closure: %v", err)
	}
	gridBytes := emitContractTree(t, grid, true)
	serverBytes := emitContractTree(t, server, false)
	for _, bytes := range []string{gridBytes, serverBytes} {
		if !strings.Contains(bytes, "expected_revision") {
			t.Fatal("emission lacks the renamed field")
		}
	}
	evidence = append(evidence,
		contractEvidence{Edit: "field", Phase: "emit", Target: "grid", Result: "pass", Detail: digestString(gridBytes)[:12]},
		contractEvidence{Edit: "field", Phase: "emit", Target: "server", Result: "pass", Detail: digestString(serverBytes)[:12]},
	)
	auditNoContractMirror(t, ws)
	logContractEvidence(t, evidence)
}

func TestContractEditLeaf(t *testing.T) {
	ws := contractWorkspaceFor(t)
	var evidence []contractEvidence
	digest := ws.rewriteSharedContract(t, [][2]string{{"grid_invalid", "grid_rejected"}})
	evidence = append(evidence, contractEvidence{Edit: "leaf", Phase: "relock", Target: "both", Result: "pass", Detail: digest[:12]})
	staleGrid := expectContractFailure(t, ws.grid, check.TargetBrowser, "grid_invalid")
	staleServer := expectContractFailure(t, ws.server, check.TargetBun, "grid_invalid")
	evidence = append(evidence,
		contractEvidence{Edit: "leaf", Phase: "stale", Target: "grid", Result: "reject", Detail: staleGrid},
		contractEvidence{Edit: "leaf", Phase: "stale", Target: "server", Result: "reject", Detail: staleServer},
	)
	repaired := 0
	for _, root := range []string{ws.server, ws.grid} {
		err := filepath.WalkDir(filepath.Join(root, "src"), func(name string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".can") {
				return err
			}
			content := contractReadFile(t, name)
			if count := strings.Count(content, "grid_invalid"); count > 0 {
				contractWriteFile(t, name, strings.ReplaceAll(content, "grid_invalid", "grid_rejected"))
				repaired += count
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if repaired == 0 {
		t.Fatal("leaf repair found no grid_invalid uses")
	}
	grid := mustCheckContractTree(t, ws.grid, check.TargetBrowser)
	server := mustCheckContractTree(t, ws.server, check.TargetBun)
	evidence = append(evidence, contractEvidence{Edit: "leaf", Phase: "check", Target: "both", Result: "pass", Detail: fmt.Sprintf("%d named uses repaired", repaired)})
	for _, program := range []*check.Program{grid, server} {
		save := contractAction(t, program, "save_invoice_grid")
		found := false
		for _, kase := range save.Cases {
			if strings.Contains(kase.Leaf, "grid_rejected") {
				found = true
				if kase.Status != 422 {
					t.Fatalf("grid_rejected status = %d", kase.Status)
				}
			}
			if strings.Contains(kase.Leaf, "grid_invalid") {
				t.Fatalf("stale leaf survives in %+v", kase)
			}
		}
		if !found {
			t.Fatal("renamed leaf missing from the case table")
		}
	}
	assertContractAgreement(t, "leaf", grid, server)
	if err := browser.CheckProgram(grid); err != nil {
		t.Fatalf("edited grid fails browser closure: %v", err)
	}
	gridBytes := emitContractTree(t, grid, true)
	serverBytes := emitContractTree(t, server, false)
	for _, bytes := range []string{gridBytes, serverBytes} {
		if !strings.Contains(bytes, "grid_rejected") {
			t.Fatal("emission lacks the renamed leaf")
		}
		if strings.Contains(bytes, "grid_invalid") {
			t.Fatal("emission still interprets the old tag")
		}
	}
	evidence = append(evidence,
		contractEvidence{Edit: "leaf", Phase: "emit", Target: "grid", Result: "pass", Detail: digestString(gridBytes)[:12]},
		contractEvidence{Edit: "leaf", Phase: "emit", Target: "server", Result: "pass", Detail: digestString(serverBytes)[:12]},
	)
	auditNoContractMirror(t, ws)
	logContractEvidence(t, evidence)
}

func TestContractEditStatus(t *testing.T) {
	ws := contractWorkspaceFor(t)
	var evidence []contractEvidence
	shared := filepath.Join(ws.shared, contractSourceFile)
	contractReplaceOnce(t, shared, "grid_saved status 200", "grid_saved status 201")
	edited, err := os.ReadFile(shared)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{ws.server, ws.grid} {
		if err := os.WriteFile(filepath.Join(root, contractVendorFile), edited, 0600); err != nil {
			t.Fatal(err)
		}
	}
	digestServer, digestGrid := contractRelock(t, ws.server), contractRelock(t, ws.grid)
	if digestServer != digestGrid {
		t.Fatalf("relock diverged: %s vs %s", digestServer, digestGrid)
	}
	evidence = append(evidence, contractEvidence{Edit: "status", Phase: "relock", Target: "both", Result: "pass", Detail: digestServer[:12]})
	grid := mustCheckContractTree(t, ws.grid, check.TargetBrowser)
	server := mustCheckContractTree(t, ws.server, check.TargetBun)
	for _, program := range []*check.Program{grid, server} {
		save := contractAction(t, program, "save_invoice_grid")
		if save.Cases[0].Status != 201 || !strings.Contains(save.Cases[0].Leaf, "grid_saved") {
			t.Fatalf("saved case = %+v", save.Cases[0])
		}
	}
	assertContractAgreement(t, "status", grid, server)
	if err := browser.CheckProgram(grid); err != nil {
		t.Fatalf("edited grid fails browser closure: %v", err)
	}
	evidence = append(evidence, contractEvidence{Edit: "status", Phase: "check", Target: "both", Result: "pass", Detail: "metadata moved automatically"})
	auditNoContractMirror(t, ws)
	emitContractTree(t, grid, true)
	emitContractTree(t, server, false)
	logContractEvidence(t, evidence)
}

func TestContractEditLimit(t *testing.T) {
	ws := contractWorkspaceFor(t)
	var evidence []contractEvidence
	digest := ws.rewriteSharedContract(t, [][2]string{{"json grid_edit_input limit 8192", "json grid_edit_input limit 128"}})
	evidence = append(evidence, contractEvidence{Edit: "limit", Phase: "relock", Target: "both", Result: "pass", Detail: digest[:12]})
	grid := mustCheckContractTree(t, ws.grid, check.TargetBrowser)
	server := mustCheckContractTree(t, ws.server, check.TargetBun)
	for _, program := range []*check.Program{grid, server} {
		save := contractAction(t, program, "save_invoice_grid")
		if save.Input.Mode != "json" || save.Input.Limit != 128 {
			t.Fatalf("save input = %+v", save.Input)
		}
	}
	assertContractAgreement(t, "limit", grid, server)
	if err := browser.CheckProgram(grid); err != nil {
		t.Fatalf("edited grid fails browser closure: %v", err)
	}
	evidence = append(evidence, contractEvidence{Edit: "limit", Phase: "check", Target: "both", Result: "pass", Detail: "budget moved on both sides"})
	auditNoContractMirror(t, ws)
	emitContractTree(t, grid, true)
	emitContractTree(t, server, false)
	logContractEvidence(t, evidence)
}

const contractSaveJSONBlock = `    json grid_edit_input limit 8192
    returns grid_edit_outcome
    body json
    cases
        grid_saved status 200
        grid_invalid status 422
        grid_conflict status 409
        grid_forbidden status 403
        grid_unavailable status 503`

const contractSaveHTMLBlock = `    form invoice_form limit 2048 rows_limit 64
    returns edit_outcome
    body html
    cases
        saved status 200 swap inner
        invalid status 422 swap inner
        conflict status 409 swap inner
        forbidden status 403 swap inner
        unavailable status 503 swap inner`

func TestContractEditBodyMode(t *testing.T) {
	ws := contractWorkspaceFor(t)
	var evidence []contractEvidence
	shared := filepath.Join(ws.shared, contractSourceFile)
	contractReplaceOnce(t, shared, contractSaveJSONBlock, contractSaveHTMLBlock)
	edited, err := os.ReadFile(shared)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{ws.server, ws.grid} {
		if err := os.WriteFile(filepath.Join(root, contractVendorFile), edited, 0600); err != nil {
			t.Fatal(err)
		}
	}
	digestServer, digestGrid := contractRelock(t, ws.server), contractRelock(t, ws.grid)
	if digestServer != digestGrid {
		t.Fatalf("relock diverged: %s vs %s", digestServer, digestGrid)
	}
	evidence = append(evidence, contractEvidence{Edit: "body-mode", Phase: "relock", Target: "both", Result: "pass", Detail: digestServer[:12]})
	staleGrid := expectContractFailure(t, ws.grid, check.TargetBrowser, "save_invoice_grid")
	staleServer := expectContractFailure(t, ws.server, check.TargetBun, "save_invoice_grid")
	evidence = append(evidence,
		contractEvidence{Edit: "body-mode", Phase: "stale", Target: "grid", Result: "reject", Detail: staleGrid},
		contractEvidence{Edit: "body-mode", Phase: "stale", Target: "server", Result: "reject", Detail: staleServer},
	)
	// The server is repaired into an HTML-only mount reusing the
	// checked form handler and renderers; the JSON grid has no form
	// client, so it stays rejected and never treats HTML as JSON.
	contractReplaceOnce(t, filepath.Join(ws.server, "src/web/web.can"),
		"call action::mount(contract::save_invoice_grid, callable save_grid) as http::route save",
		"call action::mount(contract::save_invoice_grid, callable save_html, callable render_html, callable render_bad_form) as http::route save")
	server := mustCheckContractTree(t, ws.server, check.TargetBun)
	flipped := contractAction(t, server, "save_invoice_grid")
	if flipped.Input.Mode != "form" || flipped.Body != "html" || types.CanonicalName(flipped.Returns) != "invoice_contract::edit_outcome" {
		t.Fatalf("flipped action = %+v returns %s", flipped.Input, types.CanonicalName(flipped.Returns))
	}
	stillStale := expectContractFailure(t, ws.grid, check.TargetBrowser, "save_invoice_grid")
	evidence = append(evidence,
		contractEvidence{Edit: "body-mode", Phase: "check", Target: "server", Result: "pass", Detail: "HTML-only mount repaired"},
		contractEvidence{Edit: "body-mode", Phase: "stale", Target: "grid", Result: "reject", Detail: stillStale},
	)
	serverBytes := emitContractTree(t, server, false)
	if !strings.Contains(serverBytes, "/api/tenants") {
		t.Fatal("repaired server emission lost the flipped route")
	}
	evidence = append(evidence, contractEvidence{Edit: "body-mode", Phase: "emit", Target: "server", Result: "pass", Detail: digestString(serverBytes)[:12]})
	auditNoContractMirror(t, ws)
	logContractEvidence(t, evidence)
}

func TestContractNegatives(t *testing.T) {
	t.Run("handles", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		shared := filepath.Join(ws.shared, contractSourceFile)
		// A handles line where the returns clause is due trips the
		// removed-clause diagnostic at parse time.
		contractReplaceOnce(t, shared,
			"    returns grid_load_outcome\n",
			"    handles load_grid\n    returns grid_load_outcome\n")
		edited, err := os.ReadFile(shared)
		if err != nil {
			t.Fatal(err)
		}
		for _, root := range []string{ws.server, ws.grid} {
			if err := os.WriteFile(filepath.Join(root, contractVendorFile), edited, 0600); err != nil {
				t.Fatal(err)
			}
			contractWriteLockDigest(t, root, contractVendorDigest(t, root))
		}
		_, err = project.Load(ws.server)
		if err == nil || !strings.Contains(err.Error(), "handles was removed") {
			t.Fatalf("handles admitted or misdiagnosed: %v", err)
		}
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "handles", Target: "server", Result: "reject", Detail: err.Error()}})
	})
	t.Run("mirrored-action", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		mirror := `
action load_invoice_grid_local
    get "/api/tenants/:tenant_id/invoices/:invoice_id"
    captures contract::invoice_key
    input none
    returns contract::grid_load_outcome
    body json
    cases
        contract::grid_loaded status 200
        contract::grid_load_forbidden status 403
        contract::grid_load_unavailable status 503
`
		name := filepath.Join(ws.server, "src/web/web.can")
		contractWriteFile(t, name, contractReadFile(t, name)+mirror)
		detail := expectContractFailure(t, ws.server, check.TargetBun, "load_invoice_grid_local")
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "mirrored-action", Target: "server", Result: "reject", Detail: detail}})
	})
	t.Run("missing-case", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		shared := filepath.Join(ws.shared, contractSourceFile)
		contractReplaceOnce(t, shared, "        grid_saved status 200\n", "")
		edited, err := os.ReadFile(shared)
		if err != nil {
			t.Fatal(err)
		}
		for _, root := range []string{ws.server, ws.grid} {
			if err := os.WriteFile(filepath.Join(root, contractVendorFile), edited, 0600); err != nil {
				t.Fatal(err)
			}
			contractRelock(t, root)
		}
		detail := expectContractFailure(t, ws.server, check.TargetBun, "save_invoice_grid")
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "missing-case", Target: "server", Result: "reject", Detail: detail}})
	})
	t.Run("extra-case", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		ws.rewriteSharedContract(t, [][2]string{{"        grid_saved status 200\n", "        grid_saved status 200\n        grid_bogus status 200\n"}})
		detail := expectContractFailure(t, ws.server, check.TargetBun, "grid_bogus")
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "extra-case", Target: "server", Result: "reject", Detail: detail}})
	})
	t.Run("computed-symbol", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		contractReplaceOnce(t, filepath.Join(ws.server, "src/web/web.can"),
			"action::mount(contract::load_invoice_grid, callable load_grid)",
			`action::mount("load_invoice_grid", callable load_grid)`)
		detail := expectContractFailure(t, ws.server, check.TargetBun, "string or computed name")
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "computed-symbol", Target: "server", Result: "reject", Detail: detail}})
	})
	t.Run("capture-type", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		ws.rewriteSharedContract(t, [][2]string{{
			"record invoice_key\n    int tenant_id\n    int invoice_id",
			"record invoice_key\n    int tenant_id\n    bool invoice_id",
		}})
		detail := expectContractFailure(t, ws.server, check.TargetBun, "invoice_id")
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "capture-type", Target: "server", Result: "reject", Detail: detail}})
	})
	t.Run("request-position", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		name := filepath.Join(ws.server, "src/web/web.can")
		contractReplaceOnce(t, name,
			"fn contract::grid_edit_outcome save_grid\n    emits []\n    given\n        near sql::pool pool\n        near str public_origin\n        near int window\n        http::request req\n        contract::invoice_key key\n        contract::grid_edit_input body\n",
			"fn contract::grid_edit_outcome save_grid\n    emits []\n    given\n        near sql::pool pool\n        near str public_origin\n        near int window\n        contract::invoice_key key\n        contract::grid_edit_input body\n        http::request req\n")
		detail := expectContractFailure(t, ws.server, check.TargetBun, "save_grid")
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "request-position", Target: "server", Result: "reject", Detail: detail}})
	})
	t.Run("mistyped-handler", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		name := filepath.Join(ws.server, "src/web/web.can")
		contractReplaceOnce(t, name, "fn contract::grid_edit_outcome save_grid\n", "fn contract::grid_load_outcome save_grid\n")
		detail := expectContractFailure(t, ws.server, check.TargetBun, "save_grid")
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "mistyped-handler", Target: "server", Result: "reject", Detail: detail}})
	})
	t.Run("missing-pool", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		name := filepath.Join(ws.server, "src/web/web.can")
		contractReplaceOnce(t, name,
			"fn contract::grid_edit_outcome save_grid\n    emits []\n    given\n        near sql::pool pool\n",
			"fn contract::grid_edit_outcome save_grid\n    emits []\n    given\n")
		detail := expectContractFailure(t, ws.server, check.TargetBun, "save_grid")
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "missing-pool", Target: "server", Result: "reject", Detail: detail}})
	})
	t.Run("server-closure-import", func(t *testing.T) {
		ws := contractWorkspaceFor(t)
		name := filepath.Join(ws.grid, "src/web/web.can")
		contractReplaceOnce(t, name,
			"uses [billing::invoice_contract as contract, records, model, action, browser, codec, http, option, text]",
			"uses [billing::invoice_contract as contract, records, model, action, browser, codec, env, http, option, text]")
		contractReplaceOnce(t, name,
			"                ok browser::view view => match chain\n                    call browser::create_element(view, \"p\") as browser::node status",
			"                ok browser::view view => match chain\n                    call env::required(\"contract_probe\") as str probe\n                    call browser::create_element(view, \"p\") as browser::node status")
		contractReplaceOnce(t, name,
			"                    call browser::create_text(view, notice) as browser::node text",
			"                    call browser::create_text(view, notice + probe) as browser::node text")
		contractReplaceOnce(t, name,
			"                    browser::disposed => ok\n                    browser::rejected => ok\n                    ok => ok\n\n/// Boot one grid",
			"                    browser::disposed => ok\n                    browser::rejected => ok\n                    env::invalid_name => ok\n                    http::credentials_missing => ok\n                    ok => ok\n\n/// Boot one grid")
		// The env call is well-formed Can, so target checking passes;
		// the capability closure rejects the server-only operation.
		program := mustCheckContractTree(t, ws.grid, check.TargetBrowser)
		err := browser.CheckProgram(program)
		if err == nil || !strings.Contains(err.Error(), "can.std.env@1") {
			t.Fatalf("browser closure admitted the env import: %v", err)
		}
		logContractEvidence(t, []contractEvidence{{Edit: "negative", Phase: "server-closure-import", Target: "grid", Result: "reject", Detail: err.Error()}})
	})
}

package driver

import (
	"encoding/json"
	"os"
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

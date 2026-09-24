package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veighnsche/can-lang/compiler/internal/check"
	"github.com/veighnsche/can-lang/compiler/internal/project"
)

const scenarioEmitHelper = "package helper\n    provides [read, checkout]\n    uses [text]\nscenario checkout\nfn str read\n    emits []\n    asserts\n        unit: => ok \"fixture\"\n    match call text::from_int(7)\n        when\n            scenario checkout: 7 => ok \"fixture\"\n            unit: 7 => ok \"fixture\"\n        ok str result => ok result\n"

const scenarioEmitApp = "package app\n    provides []\n    uses [helper]\nfn str read_customer\n    emits []\n    asserts\n        customer: => ok \"fixture\" link helper::checkout\n    ok call helper::read()\nfn void main\n    emits []\n    given\n        str[] arguments\n    asserts\n        empty: [] => ok\n    ok\n"

func scenarioEmitProgram(t *testing.T) *check.Program {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"can.project.json":      `{"source_root":"src","error_registry":"can.errors.json"}`,
		"can.errors.json":       `{"active":[],"retired":[]}`,
		"src/helper/helper.can": scenarioEmitHelper,
		"src/app/main.can":      scenarioEmitApp,
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
	program, err := check.CheckProgram(graph)
	if err != nil {
		t.Fatal(err)
	}
	return program
}

func TestScenarioEmissionCarriesOwnerAndLinks(t *testing.T) {
	program := scenarioEmitProgram(t)
	helperID := ""
	for _, fn := range program.Functions {
		if fn.Symbol.Package.Name == "helper" {
			helperID = fn.Symbol.Package.ID
		}
	}
	if helperID == "" {
		t.Fatal("helper package missing")
	}
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		bodies = append(bodies, string(artifact.Bytes))
	}
	joined := strings.Join(bodies, "\n")
	for _, want := range []string{
		`{selector:"checkout", owner:"` + helperID + `", scenario:"` + helperID + `::checkout"`,
		`{selector:"unit", owner:"` + helperID + `"`,
		`"links":["` + helperID + `::checkout"]`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("emitted assertions omit %s", want)
		}
	}
	if strings.Count(joined, "scenario:\"") != 1 {
		t.Fatal("plain rows gained a scenario identity")
	}
}

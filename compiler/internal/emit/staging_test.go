package emit

import (
	"strings"
	"testing"
)

// Staging entry lists every root identity and runs the suite otherwise.
func TestStagingModulesEntryListsRoots(t *testing.T) {
	program := scenarioEmitProgram(t)
	artifacts, err := StagingModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var entry []byte
	for _, artifact := range artifacts {
		if artifact.Path == "entry.ts" {
			entry = artifact.Bytes
		}
	}
	if entry == nil {
		t.Fatal("staging bundle has no entry.ts")
	}
	body := string(entry)
	if !strings.Contains(body, "$canStageRoots") {
		t.Fatal("staging entry does not embed root identities")
	}
	for _, root := range []string{`"package":"can.project.root/app"`, `"name":"customer"`, `"name":"empty"`, `"name":"unit"`} {
		if !strings.Contains(body, root) {
			t.Fatalf("staging entry omits root identity %s", root)
		}
	}
	if !strings.Contains(body, `"--list"`) {
		t.Fatal("staging entry has no --list branch")
	}
	listOpen := strings.Index(body, `if ($canStageArgs.includes("--list"))`)
	runOpen := strings.Index(body, "await $canRunAssertionRoot")
	if listOpen < 0 || runOpen < 0 || listOpen > runOpen {
		t.Fatal("staging entry does not branch list before run")
	}
	listBranch := body[listOpen:runOpen]
	for _, subject := range []string{"$canCase", "$canActual", "$canExpected"} {
		if strings.Contains(listBranch, subject) {
			t.Fatalf("staging --list branch touches subject %s", subject)
		}
	}
	if !strings.Contains(body, "$canRunAssertionRoot") {
		t.Fatal("staging entry does not run the suite")
	}
}

// Production emission never compiles the staging judge.
func TestProgramModulesExcludeStagingEntry(t *testing.T) {
	program := scenarioEmitProgram(t)
	artifacts, err := ProgramModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range artifacts {
		if strings.Contains(string(artifact.Bytes), "$canStageRoots") {
			t.Fatalf("production bundle stages the judge in %s", artifact.Path)
		}
	}
}

// Staging without assertion roots is refused, like assertion emission.
func TestStagingModulesNeedsRoots(t *testing.T) {
	program := scenarioEmitProgram(t)
	program.Assertions = nil
	if _, err := StagingModules(program, "runtime", nil); err == nil {
		t.Fatal("staging without roots succeeded")
	}
}

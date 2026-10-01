package emit

import (
	"strings"
	"testing"
)

func TestCEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/c/main.can")
	dependencies := httpDependencies(t)
	artifacts, err := AssertionModules(program, "runtime", dependencies)
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, artifact := range artifacts {
		bodies = append(bodies, string(artifact.Bytes))
	}
	text := strings.Join(bodies, "\n")
	for _, fragment := range []string{
		"$canCreateCIngress",
		"test-support/slices/i02/c.ts",
		"$canC.parseModule",
		"$canC.checkModule",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("c emission omits %q", fragment)
		}
	}
}

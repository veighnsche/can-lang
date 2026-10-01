package emit

import (
	"strings"
	"testing"
)

func TestDescriptorEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/descriptor/main.can")
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
		"$canCreateDescriptorDelivery",
		"test-support/slices/i11/descriptor.ts",
		"$canDescriptor.launchChild",
		"$canDescriptor.collectStatus",
		"$canDescriptor.waitChild",
		"$canDescriptor.killChild",
		"$canDescriptor.releaseLaunch",
		"$canDescriptor.readFacts",
		"$canDescriptor.expectedAck",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("descriptor emission omits %q", fragment)
		}
	}
}

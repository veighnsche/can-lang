package emit

import (
	"strings"
	"testing"
)

func TestDescriptorLeaseEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/lease/main.can")
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
		"$canCreateLeaseProbe",
		"test-support/slices/i12/lease.ts",
		"$canLease.probeLease",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("descriptor lease emission omits %q", fragment)
		}
	}
}

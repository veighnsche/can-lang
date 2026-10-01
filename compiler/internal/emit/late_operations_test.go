package emit

import (
	"strings"
	"testing"
)

func TestLateEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/late/main.can")
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
		"$canCreateLateOccurrence",
		"test-support/slices/i03/late.ts",
		"$canLate.select",
		"$canLate.enroll",
		"$canLate.armGate",
		"$canLate.emit",
		"$canLate.witnessTerminal",
		"$canLate.observe",
		"$canLate.grantLease",
		"$canLate.observeLease",
		"$canLate.releaseLease",
		"$canLate.reconcile",
		"$canLate.killWorker",
		"$canLate.readOutcome",
		"$canLate.readCounters",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("late emission omits %q", fragment)
		}
	}
}

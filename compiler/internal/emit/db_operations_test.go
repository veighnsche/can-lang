package emit

import (
	"strings"
	"testing"
)

func TestDbEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/db/main.can")
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
		"$canCreateDbObserver",
		"test-support/slices/i13/db.ts",
		"$canDb.openNamespace",
		"$canDb.receipt",
		"$canDb.closeNamespace",
		"$canDb.seed",
		"$canDb.pinConnection",
		"$canDb.connectionTokenForTest",
		"$canDb.unpinConnection",
		"$canDb.beginRead",
		"$canDb.fetch",
		"$canDb.endRead",
		"$canDb.compareRow",
		"$canDb.recordCompile",
		"$canDb.compileRecord",
		"$canDb.recordReturning",
		"$canDb.payloadFacts",
		"$canDb.recordFinalRows",
		"$canDb.finalFacts",
		"$canDb.comparePayloadToFinal",
		"$canDb.compareClaimedRow",
		"$canDb.creditVerdict",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("db emission omits %q", fragment)
		}
	}
}

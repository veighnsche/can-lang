package emit

import (
	"strings"
	"testing"
)

func TestDbPoisonEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/poison/main.can")
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
		"$canCreateDbPoison",
		"test-support/slices/i15/db_poison.ts",
		"$canDbPoison.beginPoisonAttempt",
		"$canDbPoison.beginControlAttempt",
		"$canDbPoison.attemptRecord",
		"$canDbPoison.recordPoisonError",
		"$canDbPoison.errorFacts",
		"$canDbPoison.recordCallbackReport",
		"$canDbPoison.callbackFacts",
		"$canDbPoison.recordPoisonSettlement",
		"$canDbPoison.poisonSettlementFacts",
		"$canDbPoison.recordFreshRead",
		"$canDbPoison.freshFacts",
		"$canDbPoison.recordReplayRead",
		"$canDbPoison.replayFacts",
		"$canDbPoison.compareFreshToReplay",
		"$canDbPoison.sentinelPresentInFresh",
		"$canDbPoison.sentinelPresentInReplay",
		"$canDbPoison.settlementAgreesWithReads",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("db poison emission omits %q", fragment)
		}
	}
}

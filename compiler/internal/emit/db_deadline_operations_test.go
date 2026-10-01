package emit

import (
	"strings"
	"testing"
)

func TestDbDeadlineEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/deadline/main.can")
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
		"$canCreateDbDeadline",
		"test-support/slices/i16/db_deadline.ts",
		"$canDbDeadline.dispatch",
		"$canDbDeadline.dispatchFacts",
		"$canDbDeadline.observeDeadline",
		"$canDbDeadline.deadlineFacts",
		"$canDbDeadline.recordDriverSettlement",
		"$canDbDeadline.driverFacts",
		"$canDbDeadline.recordServerAck",
		"$canDbDeadline.serverFacts",
		"$canDbDeadline.acquireCancelGrant",
		"$canDbDeadline.requestCancel",
		"$canDbDeadline.cancelFacts",
		"$canDbDeadline.quiesceEngine",
		"$canDbDeadline.fence",
		"$canDbDeadline.fenceFacts",
		"$canDbDeadline.leaseFacts",
		"$canDbDeadline.deadlineRelease",
		"$canDbDeadline.deadlineReleaseAck",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("db deadline emission omits %q", fragment)
		}
	}
}

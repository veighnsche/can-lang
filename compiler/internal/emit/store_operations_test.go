package emit

import (
	"strings"
	"testing"
)

func TestStoreEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/object-store/main.can")
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
		"$canCreateStore",
		"test-support/slices/i17/store.ts",
		"$canStore.openPrefix",
		"$canStore.receipt",
		"$canStore.closePrefix",
		"$canStore.openSession",
		"$canStore.sessionTokenForTest",
		"$canStore.closeSession",
		"$canStore.put",
		"$canStore.settleWrite",
		"$canStore.pending",
		"$canStore.get",
		"$canStore.delete",
		"$canStore.compareBytes",
		"$canStore.list",
		"$canStore.sealCleanup",
		"$canStore.sharedDigest",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("object-store emission omits %q", fragment)
		}
	}
}

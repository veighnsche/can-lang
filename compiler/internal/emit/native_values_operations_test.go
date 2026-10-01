package emit

import (
	"strings"
	"testing"
)

func TestNativeValuesEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/native-values/main.can")
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
		"$canCreateNativeValues",
		"test-support/slices/i01/native.ts",
		"$canNative.open",
		"$canNative.describe",
		"$canNative.make",
		"$canNative.invoke",
		"$canNative.settle",
		"$canNative.observe",
		"$canNative.allocateGate",
		"$canNative.release",
		"$canNative.installFault",
		"$canNative.restore",
		"$canNative.close",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("native-values emission omits %q", fragment)
		}
	}
}

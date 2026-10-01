package emit

import (
	"strings"
	"testing"
)

func TestOwnerTransportEmission(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/test/owner.can")
	artifacts, err := AssertionModules(program, "runtime", httpDependencies(t))
	if err != nil {
		t.Fatal(err)
	}
	var joined strings.Builder
	for _, artifact := range artifacts {
		joined.Write(artifact.Bytes)
	}
	for _, want := range []string{
		"$canTest.owner.admitGrant",
		"$canTest.owner.releaseGrant",
		"$canTest.transport.openChannel",
		"$canTest.transport.sendEnvelope",
		"$canTest.transport.recvEnvelope",
		"$canTest.transport.closeChannel",
		"$canTest.transport.pendingDepth",
		"$canCreateTestSupport",
		"export let $canTest:ReturnType<typeof $canCreateTestSupport>;",
		"test-support/transport.ts",
	} {
		if !strings.Contains(joined.String(), want) {
			t.Fatalf("missing %s", want)
		}
	}
}

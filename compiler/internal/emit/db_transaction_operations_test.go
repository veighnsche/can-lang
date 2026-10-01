package emit

import (
	"strings"
	"testing"
)

func TestDbTransactionEmissionUsesSharedContribution(t *testing.T) {
	program := sourceProgram(t, "../../testdata/current/transactions/main.can")
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
		"$canCreateDbTransaction",
		"test-support/slices/i14/db_transaction.ts",
		"$canDbTxn.enterCallback",
		"$canDbTxn.actorFacts",
		"$canDbTxn.recordLastInsertId",
		"$canDbTxn.lastInsertIdFacts",
		"$canDbTxn.compareLastInsertId",
		"$canDbTxn.recordSettlement",
		"$canDbTxn.settlementFacts",
		"$canDbTxn.release",
		"$canDbTxn.releaseAck",
		"$canTest.owner",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("db transaction emission omits %q", fragment)
		}
	}
}

package failureconventions

import (
	"context"
	"testing"
	"time"
)

// TestRetryConventionGreen is the B01 v1 leg: the result-data retry
// helper serves one domain with an oracle-verified attempt/sequence
// contract. v2 extends this project to two domains, faithful failure
// sets, trace wrappers, and standard-fault transparency.
func TestRetryConventionGreen(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	bundle := resolveBundle(t, ctx)
	project := stageProject(t, "retry")
	report := requireGreen(t, runAssert(t, ctx, bundle, project))
	requireRealCan(t, report)
	const wantRoots = 14
	if len(report.Assertions) != wantRoots {
		t.Fatalf("want %d roots, got %d: %v", wantRoots, len(report.Assertions), rootNames(report))
	}
	t.Logf("retry v1: %d roots green", len(report.Assertions))
}

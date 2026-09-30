package check

import (
	"errors"
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/ir"
	"github.com/veighnsche/can-lang/compiler/internal/syntax"
	"github.com/veighnsche/can-lang/compiler/internal/types"
	"testing"
)

func TestRecoveryChildCheckpointRestoresInferenceState(t *testing.T) {
	aggregate := &aggregateInference{identity: "original", discovery: true}
	region := &ir.Region{ID: "region"}
	callback := 0
	checker := &regionChecker{context: CompletionContext{Recover: true, Checkpoint: func() func() { return func() { callback++ } }}, aggregate: aggregate, region: region, serial: 7, locals: map[string]*types.Type{}, uses: LocalUses{Names: map[*syntax.NameExpr]string{}, Captures: map[*syntax.ReferenceExpr][]string{}}}
	rollback := checker.childCheckpoint()
	checker.serial = 100
	checker.aggregate.typ = &types.Type{}
	checker.aggregate.observed = true
	checker.aggregate.deferred = true
	checker.region.Escapes = append(checker.region.Escapes, &types.Type{})
	checker.locals["bad"] = &types.Type{}
	checker.uses.Names[&syntax.NameExpr{}] = "bad"
	rollback()
	if checker.serial != 7 || checker.aggregate != aggregate || aggregate.typ != nil || aggregate.observed || aggregate.deferred || len(region.Escapes) != 0 || len(checker.locals) != 0 || len(checker.uses.Names) != 0 || callback != 1 {
		t.Fatal("failed child changed committed checker state")
	}
}
func TestRecoveryMixedAggregateErrorIsNotDeferred(t *testing.T) {
	checker := &regionChecker{aggregate: &aggregateInference{discovery: true}}
	if checker.deferAggregate(errors.Join(&deferredAggregate{}, fmt.Errorf("independent invalid operand"))) {
		t.Fatal("direct sibling error swallowed by aggregate discovery")
	}
	if !checker.deferAggregate(errors.Join(&deferredAggregate{}, fmt.Errorf("wrapped: %w", &deferredAggregate{}))) {
		t.Fatal("pure discovery obligation was rejected")
	}
}

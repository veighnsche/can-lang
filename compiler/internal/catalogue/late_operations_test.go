package catalogue

import (
	"reflect"
	"testing"
)

// TestLateOccurrenceCatalogueContract pins the NT-I03 late-occurrence
// surface: 13 owner-first operations over the adapter-held K06 service
// (select, enroll, arm_gate, emit, witness_terminal, observe,
// grant_lease, observe_lease, release_lease, reconcile, kill_worker,
// read_outcome, read_counters), 11 inert fact records, and the
// late_fault{kind, reason} error. The Jev x3 consult
// (evidence/I03-jev, unanimous adapter_held_service) chose the I04
// doubles precedent over N-owner dispatch and mechanics deferral: K06
// is deterministic local mechanics, and the late:: namespace plus
// header documentation keep it distinct from the future P09/P11
// authorities. Ten fallible ops emit test::stale_handle and
// late::late_fault; the three pure reads emit test::stale_handle
// only. (The consult briefs said 12 methods; the engine actually
// exposes 13 public methods including counters. The shape decision
// does not hinge on the count; this contract pins all 13.)
func TestLateOccurrenceCatalogueContract(t *testing.T) {
	c := Builtin()
	target, revision := c.Inventory().TargetID, c.Inventory().Revision
	operation := func(name string) Operation {
		t.Helper()
		op, err := c.Operation(name, target, revision)
		if err != nil {
			t.Fatal(err)
		}
		return op
	}
	if err := c.CheckProjectPackage("late"); err == nil {
		t.Fatal("late package is not distribution-owned")
	}
	fieldsOf := func(name string) map[string]string {
		t.Helper()
		typ, ok := c.Type(name)
		if !ok {
			t.Fatalf("%s missing", name)
		}
		fields := map[string]string{}
		for _, field := range typ.Fields {
			fields[field.Name] = field.Type
		}
		return fields
	}
	event := fieldsOf("late::event_facts")
	if event["participant"] != "str" || event["identity"] != "str" || event["kind"] != "str" || event["seq"] != "int" || event["late"] != "bool" || event["dropped"] != "int[]" {
		t.Fatalf("late::event_facts fields differ: %+v", event)
	}
	observation := fieldsOf("late::observation_facts")
	if observation["terminal"] != "option::value<str>" || observation["binding"] != "option::value<str>" || observation["seq"] != "int" {
		t.Fatalf("late::observation_facts fields differ: %+v", observation)
	}
	outcome := fieldsOf("late::outcome_facts")
	if outcome["outcome"] != "str" || outcome["worker_terminal"] != "option::value<str>" || outcome["observer_terminal"] != "option::value<str>" || outcome["worker_dead"] != "bool" {
		t.Fatalf("late::outcome_facts fields differ: %+v", outcome)
	}
	counters := fieldsOf("late::counters")
	for _, name := range []string{"participants", "admitted", "late", "dropped", "terminals", "leases", "releases", "observations", "rejected"} {
		if counters[name] != "int" {
			t.Fatalf("late::counters %s differs: %+v", name, counters)
		}
	}
	lease := fieldsOf("late::lease_facts")
	if lease["lease"] != "str" || lease["holder"] != "str" || lease["released"] != "bool" {
		t.Fatalf("late::lease_facts fields differ: %+v", lease)
	}
	type descriptor struct {
		result string
		inputs []string
		emits  []string
	}
	owner := "test::owner"
	stale := []string{"test::stale_handle"}
	both := []string{"test::stale_handle", "late::late_fault"}
	want := map[string]descriptor{
		"late::select":           {"late::select_facts", []string{owner, "str"}, both},
		"late::enroll":           {"late::enroll_facts", []string{owner, "str"}, both},
		"late::arm_gate":         {"late::gate_facts", []string{owner, "str"}, both},
		"late::emit":             {"late::event_facts", []string{owner, "str", "str", "int"}, both},
		"late::witness_terminal": {"late::terminal_facts", []string{owner, "str", "str"}, both},
		"late::observe":          {"late::observation_facts", []string{owner, "str", "str", "int"}, both},
		"late::grant_lease":      {"late::lease_facts", []string{owner, "str"}, both},
		"late::observe_lease":    {"late::lease_facts", []string{owner, "str", "str"}, both},
		"late::release_lease":    {"late::release_facts", []string{owner, "str", "str"}, both},
		"late::reconcile":        {"late::reconcile_facts", []string{owner, "str"}, both},
		"late::kill_worker":      {"late::outcome_facts", []string{owner}, stale},
		"late::read_outcome":     {"late::outcome_facts", []string{owner}, stale},
		"late::read_counters":    {"late::counters", []string{owner}, stale},
	}
	if len(want) != 13 {
		t.Fatalf("contract pins %d operations, want 13", len(want))
	}
	for name, descriptor := range want {
		op := operation(name)
		if op.Result != descriptor.result || len(op.Inputs) != len(descriptor.inputs) {
			t.Fatalf("%s descriptor differs: %+v", name, op)
		}
		for i, input := range descriptor.inputs {
			if op.Inputs[i].Type != input {
				t.Fatalf("%s input %d differs: %+v", name, i, op.Inputs)
			}
		}
		if op.Inputs[0].Type != "test::owner" {
			t.Fatalf("%s is not owner-first: %+v", name, op.Inputs)
		}
		if !reflect.DeepEqual(op.Emits, descriptor.emits) {
			t.Fatalf("%s emits differ: %+v", name, op.Emits)
		}
		if op.Lowering.Task != "NT-I03" || !reflect.DeepEqual(op.Refs, []string{"NT-I03"}) || op.Assertion != "supplied" {
			t.Fatalf("%s traceability differs: task=%q refs=%v assertion=%q", name, op.Lowering.Task, op.Refs, op.Assertion)
		}
		if len(op.Lowering.Native) == 0 || op.Lowering.Adapter == "" {
			t.Fatalf("%s names no native recipe", name)
		}
	}
}

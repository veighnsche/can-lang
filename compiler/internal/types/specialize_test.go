package types

import "testing"

func TestSpecializationExtendsSealedEvidence(t *testing.T) {
	b, file := buildSource(t, sourceHeader+`record node
    node[] children
record box<item>
    item value
record recursive<item>
    item value
    recursive<item>[] children
`)
	node, err := b.Resolve(file, annotation(t, "node"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := b.Finish()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSpecializer(b.world, initial)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Resolve(file, annotation(t, "recursive<item>"), map[string]*Type{"item": node}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !Equal(node, node) || !Equal(first, first) {
		t.Fatal("extension invalidated sealed evidence")
	}
	if first.Fields()[1].Type.Element() != first {
		t.Fatal("recursive instance lost its back edge")
	}
	if !Equal(first.Fields()[0].Type, node) {
		t.Fatal("imported recursive argument changed identity")
	}
	if first.Fields()[0].Type == node {
		t.Fatal("new builder borrowed a mutable source node")
	}
	if node.Fields()[0].Type.Element() != node {
		t.Fatal("original recursive graph was changed")
	}
	second, err := s.Resolve(file, annotation(t, "recursive<node>"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatal("same concrete instance not interned")
	}
	before := len(s.Model().Types())
	if _, err := s.Resolve(file, annotation(t, "box<void>"), nil, false); err == nil {
		t.Fatal("void data specialization accepted")
	}
	if len(s.Model().Types()) != before || !Equal(first, first) || !Equal(node, node) {
		t.Fatal("failed extension poisoned existing evidence")
	}
	if _, err := s.Resolve(file, annotation(t, "box<node>"), nil, false); err != nil {
		t.Fatal(err)
	}
}

func TestSpecializationKeysAndAggregateConstraints(t *testing.T) {
	file, values := inferenceFixture(t)
	key, err := SpecializationKey("package::function", []*Type{values["node"]})
	if err != nil {
		t.Fatal(err)
	}
	again, err := SpecializationKey("package::function", []*Type{values["node"]})
	if err != nil || again != key {
		t.Fatal("unstable specialization identity")
	}
	other, _ := SpecializationKey("other::function", []*Type{values["node"]})
	if other == key {
		t.Fatal("different declaration merged")
	}
	other, _ = SpecializationKey("package::function", []*Type{values["int"]})
	if other == key {
		t.Fatal("different arguments merged")
	}
	pattern, err := Pattern(file, annotation(t, "all_failed<item>"), []string{"item"})
	if err != nil {
		t.Fatal(err)
	}
	solver, _ := NewInference([]string{"item"})
	if err := solver.Constrain(pattern, values["all_failed<failures>"]); err != nil {
		t.Fatal(err)
	}
	args, err := solver.Arguments()
	if err != nil || !Equal(args[0], values["failures"]) {
		t.Fatal("expected aggregate did not supply exact failure variant")
	}
	if _, err := SpecializationKey("package::function", []*Type{values["void"]}); err == nil {
		t.Fatal("void specialization key admitted")
	}
}

func TestFailedExtensionSealDoesNotPoisonEarlierTypes(t *testing.T) {
	b, file := buildSource(t, sourceHeader+`record box<item>
    item value
variant either<left, right>
    box<left>
    box<right>
`)
	integer, err := b.Resolve(file, annotation(t, "int"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := b.Finish()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSpecializer(b.world, initial)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := s.Model()
	if _, err := s.Resolve(file, annotation(t, "either<int, int>"), nil, false); err == nil {
		t.Fatal("duplicate concrete variant leaves admitted")
	}
	if !Equal(integer, integer) || len(s.Model().Types()) != len(snapshot.Types()) {
		t.Fatal("failed sealing invalidated earlier evidence")
	}
	valid, err := s.Resolve(file, annotation(t, "either<int, str>"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(valid.Leaves()) != 2 {
		t.Fatal("successful extension lost variant leaves")
	}
	if len(snapshot.Types()) != 1 {
		t.Fatal("earlier model snapshot mutated")
	}
}

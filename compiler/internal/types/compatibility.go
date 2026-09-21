package types

import "fmt"

func Equal(a, b *Type) bool {
	return a != nil && b != nil && a.graph.sealed && b.graph.sealed && a.id == b.id
}

// Assignable permits named variant inclusion and finite callable-bound widening.
// It never recursively widens generic arguments, arrays or callable signatures.
func Assignable(actual, expected *Type) bool {
	if Equal(actual, expected) {
		return true
	}
	if actual == nil || expected == nil || !actual.graph.sealed || !expected.graph.sealed {
		return false
	}
	// Even phantom generic arguments are invariant. Coincidentally equal leaf
	// sets do not erase distinct specializations of the same nominal template.
	if actual.declaration != "" && actual.declaration == expected.declaration && (len(actual.arguments) != 0 || len(expected.arguments) != 0) {
		return false
	}
	if expected.kind == Variant {
		allowed := map[string]bool{}
		for _, leaf := range expected.Leaves() {
			allowed[leaf.id] = true
		}
		if actual.kind == Variant {
			for _, leaf := range actual.Leaves() {
				if !allowed[leaf.id] {
					return false
				}
			}
			return true
		}
		return allowed[actual.id]
	}
	if actual.kind != expected.kind || (actual.kind != Callable && actual.kind != ChoiceArm) {
		return false
	}
	if !Equal(actual.result, expected.result) || len(actual.inputs) != len(expected.inputs) {
		return false
	}
	for i, t := range actual.inputs {
		if !Equal(t, expected.inputs[i]) {
			return false
		}
	}
	allowed := map[string]bool{}
	for _, e := range expected.errors {
		allowed[e.id] = true
	}
	for _, e := range actual.errors {
		if !allowed[e.id] {
			return false
		}
	}
	return true
}

type Replacement struct {
	Name string
	Type *Type
}

// CheckUpdate preserves the receiver's exact nominal specialization. Evaluation
// order and fresh native record construction are lowering responsibilities.
func CheckUpdate(receiver *Type, replacements []Replacement) (*Type, error) {
	if receiver == nil || !receiver.graph.sealed || (receiver.kind != Record && receiver.kind != Error) {
		return nil, fmt.Errorf("copy-update requires an ordinary record or error")
	}
	if len(replacements) == 0 {
		return nil, fmt.Errorf("copy-update needs at least one replacement")
	}
	fields := map[string]*Type{}
	for _, f := range receiver.fields {
		fields[f.Name] = f.Type
	}
	seen := map[string]bool{}
	for _, r := range replacements {
		if seen[r.Name] {
			return nil, fmt.Errorf("duplicate replacement %q", r.Name)
		}
		seen[r.Name] = true
		expected, ok := fields[r.Name]
		if !ok {
			return nil, fmt.Errorf("unknown field %q", r.Name)
		}
		if !Assignable(r.Type, expected) {
			return nil, fmt.Errorf("wrong replacement type for %q", r.Name)
		}
	}
	return receiver, nil
}
